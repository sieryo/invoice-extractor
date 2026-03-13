package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/app/ingest"
)

type CollectionHistoryRepository struct {
	db *sql.DB
}

func NewCollectionHistoryRepository(db *sql.DB) *CollectionHistoryRepository {
	return &CollectionHistoryRepository{db: db}
}

func (r *CollectionHistoryRepository) EnsureUploadHistory(
	ctx context.Context,
	userID string,
	collectionID string,
	sessionID string,
) (*ingest.CollectionHistory, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, user_id, collection_id, action_type, session_id, triggered_by, status,
			started_at, finished_at, total_count, ready_count, warning_count, failed_count, duplicate_count, metadata_json
		FROM collection_history
		WHERE action_type = 'upload'
		  AND session_id = ?
		LIMIT 1
	`, sessionID)

	h, err := scanCollectionHistory(row)
	if err == nil {
		return h, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	id := uuid.NewString()
	now := time.Now()
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO collection_history (
			id, user_id, collection_id, action_type, session_id, triggered_by, status, started_at
		) VALUES (?, ?, ?, 'upload', ?, 'user', 'running', ?)
	`, id, userID, collectionID, sessionID, now)
	if err != nil {
		return nil, err
	}

	return &ingest.CollectionHistory{
		ID:           id,
		UserID:       userID,
		CollectionID: collectionID,
		ActionType:   "upload",
		SessionID:    &sessionID,
		TriggeredBy:  "user",
		Status:       "running",
		StartedAt:    now,
	}, nil
}

func (r *CollectionHistoryRepository) AddItems(ctx context.Context, items []*ingest.CollectionHistoryItem) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO collection_history_items (
			id, history_id, user_id, collection_id, document_type,
			source_name, source_size_bytes, source_mime, source_sha256, source_order,
			item_status, message, document_id, duplicate_of_id, duplicate_key,
			warnings_json, errors_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		_, err := stmt.ExecContext(ctx,
			item.ID,
			item.HistoryID,
			item.UserID,
			item.CollectionID,
			item.DocumentType,
			item.SourceName,
			item.SourceSizeBytes,
			item.SourceMIME,
			item.SourceSHA256,
			item.SourceOrder,
			item.ItemStatus,
			item.Message,
			item.DocumentID,
			item.DuplicateOfID,
			item.DuplicateKey,
			item.WarningsJSON,
			item.ErrorsJSON,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *CollectionHistoryRepository) IncrementSummary(
	ctx context.Context,
	historyID string,
	total int,
	ready int,
	warning int,
	failed int,
	duplicate int,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE collection_history
		SET
			total_count = total_count + ?,
			ready_count = ready_count + ?,
			warning_count = warning_count + ?,
			failed_count = failed_count + ?,
			duplicate_count = duplicate_count + ?
		WHERE id = ?
	`, total, ready, warning, failed, duplicate, historyID)
	return err
}

func (r *CollectionHistoryRepository) SetStatus(ctx context.Context, historyID string, status string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE collection_history
		SET status = ?, finished_at = ?
		WHERE id = ?
	`, status, now, historyID)
	return err
}

func scanCollectionHistory(row rowScanner) (*ingest.CollectionHistory, error) {
	var (
		h ingest.CollectionHistory

		sessionID sql.NullString
		finished  sql.NullTime
		meta      []byte
	)

	if err := row.Scan(
		&h.ID,
		&h.UserID,
		&h.CollectionID,
		&h.ActionType,
		&sessionID,
		&h.TriggeredBy,
		&h.Status,
		&h.StartedAt,
		&finished,
		&h.TotalCount,
		&h.ReadyCount,
		&h.WarningCount,
		&h.FailedCount,
		&h.DuplicateCount,
		&meta,
	); err != nil {
		return nil, err
	}

	if sessionID.Valid {
		v := sessionID.String
		h.SessionID = &v
	}
	if finished.Valid {
		t := finished.Time
		h.FinishedAt = &t
	}
	if len(meta) > 0 && json.Valid(meta) {
		h.MetadataJSON = meta
	}

	return &h, nil
}
