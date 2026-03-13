package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/ingest"
)

type UploadChunkRepository struct {
	db *sql.DB
}

func NewUploadChunkRepository(db *sql.DB) *UploadChunkRepository {
	return &UploadChunkRepository{db: db}
}

func (r *UploadChunkRepository) Create(ctx context.Context, chunk *ingest.UploadChunk) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO upload_session_chunks (
			id, session_id, chunk_index, status, idempotency_key, request_checksum,
			file_count, size_bytes, job_id, error_message, payload_json,
			created_at, started_at, finished_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		chunk.ID,
		chunk.SessionID,
		chunk.ChunkIndex,
		chunk.Status,
		chunk.IdempotencyKey,
		chunk.RequestChecksum,
		chunk.FileCount,
		chunk.SizeBytes,
		chunk.JobID,
		chunk.ErrorMessage,
		chunk.PayloadJSON,
		chunk.CreatedAt,
		chunk.StartedAt,
		chunk.FinishedAt,
	)
	return err
}

func (r *UploadChunkRepository) FindByID(ctx context.Context, id string) (*ingest.UploadChunk, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, session_id, chunk_index, status, idempotency_key, request_checksum,
			file_count, size_bytes, job_id, error_message, payload_json,
			created_at, started_at, finished_at
		FROM upload_session_chunks
		WHERE id = ?
	`, id)
	return scanUploadChunk(row)
}

func (r *UploadChunkRepository) FindBySessionAndIndex(
	ctx context.Context,
	sessionID string,
	chunkIndex int,
) (*ingest.UploadChunk, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, session_id, chunk_index, status, idempotency_key, request_checksum,
			file_count, size_bytes, job_id, error_message, payload_json,
			created_at, started_at, finished_at
		FROM upload_session_chunks
		WHERE session_id = ? AND chunk_index = ?
	`, sessionID, chunkIndex)
	return scanUploadChunk(row)
}

func (r *UploadChunkRepository) ListBySession(ctx context.Context, sessionID string) ([]*ingest.UploadChunk, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, session_id, chunk_index, status, idempotency_key, request_checksum,
			file_count, size_bytes, job_id, error_message, payload_json,
			created_at, started_at, finished_at
		FROM upload_session_chunks
		WHERE session_id = ?
		ORDER BY chunk_index ASC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*ingest.UploadChunk
	for rows.Next() {
		ch, err := scanUploadChunk(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ch)
	}
	return out, rows.Err()
}

func (r *UploadChunkRepository) UpdateStatus(
	ctx context.Context,
	id string,
	status ingest.ChunkStatus,
	errMsg *string,
) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE upload_session_chunks
		SET
			status = ?,
			error_message = ?,
			started_at = CASE
				WHEN ? = 'processing' AND started_at IS NULL THEN ?
				ELSE started_at
			END,
			finished_at = CASE
				WHEN ? IN ('done', 'failed', 'duplicate') THEN ?
				ELSE finished_at
			END
		WHERE id = ?
	`, status, errMsg, status, now, status, now, id)
	return err
}

func (r *UploadChunkRepository) UpdateJobID(ctx context.Context, id string, jobID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE upload_session_chunks
		SET job_id = ?
		WHERE id = ?
	`, jobID, id)
	return err
}

func scanUploadChunk(row rowScanner) (*ingest.UploadChunk, error) {
	var (
		ch ingest.UploadChunk

		statusRaw string

		reqChecksum sql.NullString
		jobID       sql.NullString
		errMsg      sql.NullString
		payload     []byte
		startedAt   sql.NullTime
		finishedAt  sql.NullTime
	)

	if err := row.Scan(
		&ch.ID,
		&ch.SessionID,
		&ch.ChunkIndex,
		&statusRaw,
		&ch.IdempotencyKey,
		&reqChecksum,
		&ch.FileCount,
		&ch.SizeBytes,
		&jobID,
		&errMsg,
		&payload,
		&ch.CreatedAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ingest.ErrChunkNotFound
		}
		return nil, err
	}

	ch.Status = ingest.ChunkStatus(statusRaw)
	if reqChecksum.Valid {
		v := reqChecksum.String
		ch.RequestChecksum = &v
	}
	if jobID.Valid {
		v := jobID.String
		ch.JobID = &v
	}
	if errMsg.Valid {
		v := errMsg.String
		ch.ErrorMessage = &v
	}
	if len(payload) > 0 && !json.Valid(payload) {
		payload = nil
	}
	if len(payload) > 0 {
		ch.PayloadJSON = payload
	}
	if startedAt.Valid {
		t := startedAt.Time
		ch.StartedAt = &t
	}
	if finishedAt.Valid {
		t := finishedAt.Time
		ch.FinishedAt = &t
	}

	return &ch, nil
}
