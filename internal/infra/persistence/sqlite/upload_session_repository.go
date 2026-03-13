package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/document"
	"github.com/sieryo/invoice-extractor/internal/app/ingest"
)

type UploadSessionRepository struct {
	db *sql.DB
}

func NewUploadSessionRepository(db *sql.DB) *UploadSessionRepository {
	return &UploadSessionRepository{db: db}
}

func (r *UploadSessionRepository) Create(ctx context.Context, session *ingest.UploadSession) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO upload_sessions (
			id, user_id, collection_id, document_type, status,
			total_chunks, uploaded_chunks, processed_chunks, failed_chunks, duplicate_chunks,
			last_heartbeat_at, started_at, finished_at, expires_at, client_session_key, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		session.ID,
		session.UserID,
		session.CollectionID,
		session.DocumentType,
		session.Status,
		session.TotalChunks,
		session.UploadedChunks,
		session.ProcessedChunks,
		session.FailedChunks,
		session.DuplicateChunks,
		session.LastHeartbeatAt,
		session.StartedAt,
		session.FinishedAt,
		session.ExpiresAt,
		session.ClientSessionKey,
		session.MetadataJSON,
	)
	return err
}

func (r *UploadSessionRepository) FindByID(ctx context.Context, id string) (*ingest.UploadSession, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, user_id, collection_id, document_type, status,
			total_chunks, uploaded_chunks, processed_chunks, failed_chunks, duplicate_chunks,
			last_heartbeat_at, started_at, finished_at, expires_at, client_session_key, metadata_json
		FROM upload_sessions
		WHERE id = ?
	`, id)

	return scanUploadSession(row)
}

func (r *UploadSessionRepository) ListActive(ctx context.Context) ([]*ingest.UploadSession, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, user_id, collection_id, document_type, status,
			total_chunks, uploaded_chunks, processed_chunks, failed_chunks, duplicate_chunks,
			last_heartbeat_at, started_at, finished_at, expires_at, client_session_key, metadata_json
		FROM upload_sessions
		WHERE status IN ('created', 'receiving', 'processing', 'finalized')
		ORDER BY started_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*ingest.UploadSession
	for rows.Next() {
		s, err := scanUploadSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *UploadSessionRepository) ListActiveByCollection(ctx context.Context, collectionID string) ([]*ingest.UploadSession, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, user_id, collection_id, document_type, status,
			total_chunks, uploaded_chunks, processed_chunks, failed_chunks, duplicate_chunks,
			last_heartbeat_at, started_at, finished_at, expires_at, client_session_key, metadata_json
		FROM upload_sessions
		WHERE collection_id = ?
		  AND status IN ('created', 'receiving', 'processing', 'finalized')
		ORDER BY started_at DESC
	`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*ingest.UploadSession
	for rows.Next() {
		s, err := scanUploadSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *UploadSessionRepository) UpdateStatus(ctx context.Context, id string, status ingest.SessionStatus) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE upload_sessions
		SET status = ?, last_heartbeat_at = ?
		WHERE id = ?
	`, status, time.Now(), id)
	return err
}

func (r *UploadSessionRepository) TouchHeartbeat(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE upload_sessions
		SET last_heartbeat_at = ?
		WHERE id = ?
	`, time.Now(), id)
	return err
}

func (r *UploadSessionRepository) IncrementUploadedChunk(ctx context.Context, id string, totalChunkCandidate int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE upload_sessions
		SET
			uploaded_chunks = uploaded_chunks + 1,
			total_chunks = CASE
				WHEN total_chunks < ? THEN ?
				ELSE total_chunks
			END,
			last_heartbeat_at = ?
		WHERE id = ?
	`, totalChunkCandidate, totalChunkCandidate, time.Now(), id)
	return err
}

func (r *UploadSessionRepository) IncrementProcessedChunk(ctx context.Context, id string, failedDelta int, duplicateDelta int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE upload_sessions
		SET
			processed_chunks = processed_chunks + 1,
			failed_chunks = failed_chunks + ?,
			duplicate_chunks = duplicate_chunks + ?,
			last_heartbeat_at = ?
		WHERE id = ?
	`, failedDelta, duplicateDelta, time.Now(), id)
	return err
}

func (r *UploadSessionRepository) SetFinished(ctx context.Context, id string, status ingest.SessionStatus) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE upload_sessions
		SET status = ?, finished_at = ?, last_heartbeat_at = ?
		WHERE id = ?
	`, status, now, now, id)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUploadSession(row rowScanner) (*ingest.UploadSession, error) {
	var (
		s ingest.UploadSession

		docTypeRaw string
		statusRaw  string

		lastHeartbeat sql.NullTime
		finishedAt    sql.NullTime
		expiresAt     sql.NullTime
		clientKey     sql.NullString
		metadata      []byte
	)

	if err := row.Scan(
		&s.ID,
		&s.UserID,
		&s.CollectionID,
		&docTypeRaw,
		&statusRaw,
		&s.TotalChunks,
		&s.UploadedChunks,
		&s.ProcessedChunks,
		&s.FailedChunks,
		&s.DuplicateChunks,
		&lastHeartbeat,
		&s.StartedAt,
		&finishedAt,
		&expiresAt,
		&clientKey,
		&metadata,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ingest.ErrSessionNotFound
		}
		return nil, err
	}

	s.DocumentType = document.DocumentType(docTypeRaw)
	s.Status = ingest.SessionStatus(statusRaw)

	if lastHeartbeat.Valid {
		t := lastHeartbeat.Time
		s.LastHeartbeatAt = &t
	}
	if finishedAt.Valid {
		t := finishedAt.Time
		s.FinishedAt = &t
	}
	if expiresAt.Valid {
		t := expiresAt.Time
		s.ExpiresAt = &t
	}
	if clientKey.Valid {
		v := clientKey.String
		s.ClientSessionKey = &v
	}
	if len(metadata) > 0 && !json.Valid(metadata) {
		metadata = nil
	}
	if len(metadata) > 0 {
		s.MetadataJSON = metadata
	}

	return &s, nil
}
