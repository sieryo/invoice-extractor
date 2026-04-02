package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/app/document"
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
			id, history_id, user_id, collection_id, collection_kind, source_format, document_type,
			source_name, source_size_bytes, source_mime, source_sha256, source_order,
			item_status, message, document_id, duplicate_of_id, duplicate_key,
			warnings_json, errors_json
		) VALUES (?, ?, ?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			item.CollectionKind,
			item.SourceFormat,
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

func (r *CollectionHistoryRepository) FindByID(
	ctx context.Context,
	historyID string,
) (*ingest.CollectionHistory, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, user_id, collection_id, action_type, session_id, triggered_by, status,
			started_at, finished_at, total_count, ready_count, warning_count, failed_count, duplicate_count, metadata_json
		FROM collection_history
		WHERE id = ?
	`, historyID)

	h, err := scanCollectionHistory(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ingest.ErrHistoryNotFound
		}
		return nil, err
	}
	return h, nil
}

func (r *CollectionHistoryRepository) ListByCollection(
	ctx context.Context,
	collectionID string,
	limit int,
	offset int,
) ([]*ingest.CollectionHistory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, user_id, collection_id, action_type, session_id, triggered_by, status,
			started_at, finished_at, total_count, ready_count, warning_count, failed_count, duplicate_count, metadata_json
		FROM collection_history
		WHERE collection_id = ?
		ORDER BY started_at DESC
		LIMIT ? OFFSET ?
	`, collectionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*ingest.CollectionHistory, 0)
	for rows.Next() {
		h, scanErr := scanCollectionHistory(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *CollectionHistoryRepository) ListItems(
	ctx context.Context,
	historyID string,
	status string,
	limit int,
	offset int,
) ([]*ingest.CollectionHistoryItem, error) {
	query := `
		SELECT
			id, history_id, user_id, collection_id, collection_kind, source_format,
			source_name, source_size_bytes, source_mime, source_sha256, source_order,
			item_status, message, document_id, duplicate_of_id, duplicate_key,
			warnings_json, errors_json, created_at
		FROM collection_history_items
		WHERE history_id = ?
	`
	args := []any{historyID}

	normalizedStatus := strings.ToLower(strings.TrimSpace(status))
	if normalizedStatus != "" {
		query += ` AND item_status = ?`
		args = append(args, normalizedStatus)
	}

	query += ` ORDER BY created_at ASC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*ingest.CollectionHistoryItem, 0)
	for rows.Next() {
		item, scanErr := scanCollectionHistoryItem(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	return out, rows.Err()
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

func scanCollectionHistoryItem(row rowScanner) (*ingest.CollectionHistoryItem, error) {
	var (
		item ingest.CollectionHistoryItem

		collectionKindRaw string
		sourceFormatRaw   string
		sourceSize        sql.NullInt64
		sourceMIME        sql.NullString
		sourceSHA         sql.NullString
		sourceOrder       sql.NullInt64
		message           sql.NullString
		documentID        sql.NullString
		duplicateOfID     sql.NullString
		duplicateKey      sql.NullString
		warningsRaw       []byte
		errorsRaw         []byte
		createdAt         time.Time
	)

	if err := row.Scan(
		&item.ID,
		&item.HistoryID,
		&item.UserID,
		&item.CollectionID,
		&collectionKindRaw,
		&sourceFormatRaw,
		&item.SourceName,
		&sourceSize,
		&sourceMIME,
		&sourceSHA,
		&sourceOrder,
		&item.ItemStatus,
		&message,
		&documentID,
		&duplicateOfID,
		&duplicateKey,
		&warningsRaw,
		&errorsRaw,
		&createdAt,
	); err != nil {
		return nil, err
	}

	item.CollectionKind = document.CollectionKind(collectionKindRaw)
	item.SourceFormat = document.SourceFormat(sourceFormatRaw)
	if sourceSize.Valid {
		item.SourceSizeBytes = sourceSize.Int64
	}
	if sourceMIME.Valid {
		item.SourceMIME = sourceMIME.String
	}
	if sourceSHA.Valid {
		item.SourceSHA256 = sourceSHA.String
	}
	if sourceOrder.Valid {
		item.SourceOrder = int(sourceOrder.Int64)
	}
	if message.Valid {
		item.Message = message.String
	}
	if documentID.Valid {
		v := documentID.String
		item.DocumentID = &v
	}
	if duplicateOfID.Valid {
		v := duplicateOfID.String
		item.DuplicateOfID = &v
	}
	if duplicateKey.Valid {
		v := duplicateKey.String
		item.DuplicateKey = &v
	}
	if len(warningsRaw) > 0 && json.Valid(warningsRaw) {
		item.WarningsJSON = warningsRaw
	}
	if len(errorsRaw) > 0 && json.Valid(errorsRaw) {
		item.ErrorsJSON = errorsRaw
	}
	_ = createdAt

	return &item, nil
}
