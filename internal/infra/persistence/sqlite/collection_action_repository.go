package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/action"
	"github.com/sieryo/invoice-extractor/internal/app/document"
)

type CollectionActionRepository struct {
	db *sql.DB
}

func NewCollectionActionRepository(db *sql.DB) *CollectionActionRepository {
	return &CollectionActionRepository{db: db}
}

func (r *CollectionActionRepository) CreateAction(
	ctx context.Context,
	act *action.CollectionAction,
) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO collection_actions (
			id, user_id, collection_id, document_type,
			action_type, status, message, params_json,
			snapshot_json, snapshot_hash, snapshot_total,
			rerun_of_action_id, idempotency_key,
			total_count, success_count, warning_count, failed_count, skipped_count,
			started_at, finished_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		act.ID,
		act.UserID,
		act.CollectionID,
		act.DocumentType,
		act.ActionType,
		act.Status,
		act.Message,
		act.ParamsJSON,
		act.SnapshotJSON,
		act.SnapshotHash,
		act.SnapshotTotal,
		act.RerunOfAction,
		act.IdempotencyKey,
		act.TotalCount,
		act.SuccessCount,
		act.WarningCount,
		act.FailedCount,
		act.SkippedCount,
		act.StartedAt,
		act.FinishedAt,
		act.CreatedAt,
		act.UpdatedAt,
	)
	return err
}

func (r *CollectionActionRepository) FindActionByID(
	ctx context.Context,
	actionID string,
) (*action.CollectionAction, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, user_id, collection_id, document_type,
			action_type, status, message, params_json,
			snapshot_json, snapshot_hash, snapshot_total,
			rerun_of_action_id, idempotency_key,
			total_count, success_count, warning_count, failed_count, skipped_count,
			started_at, finished_at, created_at, updated_at
		FROM collection_actions
		WHERE id = ?
	`, actionID)

	return scanCollectionAction(row)
}

func (r *CollectionActionRepository) FindActionByIdempotency(
	ctx context.Context,
	collectionID string,
	actionType string,
	idempotencyKey string,
) (*action.CollectionAction, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, user_id, collection_id, document_type,
			action_type, status, message, params_json,
			snapshot_json, snapshot_hash, snapshot_total,
			rerun_of_action_id, idempotency_key,
			total_count, success_count, warning_count, failed_count, skipped_count,
			started_at, finished_at, created_at, updated_at
		FROM collection_actions
		WHERE collection_id = ?
		  AND action_type = ?
		  AND idempotency_key = ?
		LIMIT 1
	`, collectionID, actionType, idempotencyKey)

	act, err := scanCollectionAction(row)
	if errors.Is(err, action.ErrActionNotFound) {
		return nil, nil
	}
	return act, err
}

func (r *CollectionActionRepository) ListActions(
	ctx context.Context,
	collectionID string,
	status string,
	limit int,
	offset int,
) ([]*action.CollectionAction, error) {
	query := `
		SELECT
			id, user_id, collection_id, document_type,
			action_type, status, message, params_json,
			snapshot_json, snapshot_hash, snapshot_total,
			rerun_of_action_id, idempotency_key,
			total_count, success_count, warning_count, failed_count, skipped_count,
			started_at, finished_at, created_at, updated_at
		FROM collection_actions
		WHERE collection_id = ?
	`
	args := []any{collectionID}
	if strings.TrimSpace(status) != "" {
		query += ` AND status = ?`
		args = append(args, strings.ToLower(strings.TrimSpace(status)))
	}
	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*action.CollectionAction, 0)
	for rows.Next() {
		act, err := scanCollectionAction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, act)
	}
	return out, rows.Err()
}

func (r *CollectionActionRepository) ListPendingActions(ctx context.Context) ([]*action.CollectionAction, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, user_id, collection_id, document_type,
			action_type, status, message, params_json,
			snapshot_json, snapshot_hash, snapshot_total,
			rerun_of_action_id, idempotency_key,
			total_count, success_count, warning_count, failed_count, skipped_count,
			started_at, finished_at, created_at, updated_at
		FROM collection_actions
		WHERE status IN ('queued', 'running')
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*action.CollectionAction, 0)
	for rows.Next() {
		act, err := scanCollectionAction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, act)
	}
	return out, rows.Err()
}

func (r *CollectionActionRepository) ListActionItems(
	ctx context.Context,
	actionID string,
) ([]*action.CollectionActionItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			ai.id, ai.action_id, ai.document_id, d.source_name,
			ai.status, ai.message, ai.warnings_json, ai.error, ai.created_at
		FROM collection_action_items ai
		LEFT JOIN documents d ON d.id = ai.document_id
		WHERE ai.action_id = ?
		ORDER BY ai.created_at ASC
	`, actionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*action.CollectionActionItem, 0)
	for rows.Next() {
		item, err := scanCollectionActionItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *CollectionActionRepository) ListActionOutputs(
	ctx context.Context,
	actionID string,
) ([]*action.CollectionActionOutput, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, action_id, kind, name, object_ref, mime_type, size_bytes, checksum, created_at
		FROM collection_action_outputs
		WHERE action_id = ?
		ORDER BY created_at ASC
	`, actionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*action.CollectionActionOutput, 0)
	for rows.Next() {
		output, err := scanCollectionActionOutput(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, output)
	}
	return out, rows.Err()
}

func (r *CollectionActionRepository) SetActionRunning(
	ctx context.Context,
	actionID string,
	startedAt time.Time,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE collection_actions
		SET status = 'running', started_at = ?, updated_at = ?
		WHERE id = ?
	`, startedAt, startedAt, actionID)
	return err
}

func (r *CollectionActionRepository) SetActionFinished(
	ctx context.Context,
	actionID string,
	status action.Status,
	message string,
	total int,
	success int,
	warning int,
	failed int,
	skipped int,
	finishedAt time.Time,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE collection_actions
		SET
			status = ?,
			message = ?,
			total_count = ?,
			success_count = ?,
			warning_count = ?,
			failed_count = ?,
			skipped_count = ?,
			finished_at = ?,
			updated_at = ?
		WHERE id = ?
	`, status, message, total, success, warning, failed, skipped, finishedAt, finishedAt, actionID)
	return err
}

func (r *CollectionActionRepository) AddActionItems(
	ctx context.Context,
	items []*action.CollectionActionItem,
) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO collection_action_items (
			id, action_id, document_id, status, message, warnings_json, error, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		_, err := stmt.ExecContext(ctx,
			item.ID,
			item.ActionID,
			item.DocumentID,
			item.Status,
			item.Message,
			item.WarningsJSON,
			item.Error,
			item.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *CollectionActionRepository) AddActionOutputs(
	ctx context.Context,
	outputs []*action.CollectionActionOutput,
) error {
	if len(outputs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO collection_action_outputs (
			id, action_id, kind, name, object_ref, mime_type, size_bytes, checksum, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, output := range outputs {
		_, err := stmt.ExecContext(ctx,
			output.ID,
			output.ActionID,
			output.Kind,
			output.Name,
			output.ObjectRef,
			output.MimeType,
			output.SizeBytes,
			output.Checksum,
			output.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *CollectionActionRepository) ListSnapshotDocuments(
	ctx context.Context,
	collectionID string,
	documentType document.DocumentType,
	statuses []string,
) ([]action.SnapshotDocument, error) {
	if len(statuses) == 0 {
		statuses = []string{"ready", "warning"}
	}

	placeholders := make([]string, len(statuses))
	args := make([]any, 0, len(statuses)+2)
	args = append(args, collectionID, documentType)
	for i, status := range statuses {
		placeholders[i] = "?"
		args = append(args, status)
	}

	query := `
		SELECT
			id, source_name, source_order, status, document_tag, source_sha256, normalized_ref, audit_ref, raw_ref
		FROM documents
		WHERE collection_id = ?
		  AND document_type = ?
		  AND deleted_at IS NULL
		  AND status IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY source_order ASC, id ASC
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]action.SnapshotDocument, 0)
	for rows.Next() {
		var (
			doc         action.SnapshotDocument
			documentTag sql.NullString
			sourceSHA   sql.NullString
			auditRef    sql.NullString
			rawRef      sql.NullString
		)

		if err := rows.Scan(
			&doc.DocumentID,
			&doc.SourceName,
			&doc.SourceOrder,
			&doc.Status,
			&documentTag,
			&sourceSHA,
			&doc.NormalizedRef,
			&auditRef,
			&rawRef,
		); err != nil {
			return nil, err
		}

		if documentTag.Valid {
			doc.DocumentTag = documentTag.String
		}
		if sourceSHA.Valid {
			doc.SourceSHA256 = sourceSHA.String
		}
		if auditRef.Valid {
			doc.AuditRef = auditRef.String
		}
		if rawRef.Valid {
			doc.RawRef = rawRef.String
		}

		out = append(out, doc)
	}

	return out, rows.Err()
}

func (r *CollectionActionRepository) ListSnapshotDocumentsByIDs(
	ctx context.Context,
	collectionID string,
	documentType document.DocumentType,
	documentIDs []string,
) ([]action.SnapshotDocument, error) {
	if len(documentIDs) == 0 {
		return []action.SnapshotDocument{}, nil
	}

	placeholders := make([]string, len(documentIDs))
	args := make([]any, 0, len(documentIDs)+2)
	args = append(args, collectionID, documentType)
	for i, id := range documentIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := `
		SELECT
			id, source_name, source_order, status, document_tag, source_sha256, normalized_ref, audit_ref, raw_ref
		FROM documents
		WHERE collection_id = ?
		  AND document_type = ?
		  AND deleted_at IS NULL
		  AND id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY source_order ASC, id ASC
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]action.SnapshotDocument, 0, len(documentIDs))
	for rows.Next() {
		var (
			doc         action.SnapshotDocument
			documentTag sql.NullString
			sourceSHA   sql.NullString
			auditRef    sql.NullString
			rawRef      sql.NullString
		)

		if err := rows.Scan(
			&doc.DocumentID,
			&doc.SourceName,
			&doc.SourceOrder,
			&doc.Status,
			&documentTag,
			&sourceSHA,
			&doc.NormalizedRef,
			&auditRef,
			&rawRef,
		); err != nil {
			return nil, err
		}

		if documentTag.Valid {
			doc.DocumentTag = documentTag.String
		}
		if sourceSHA.Valid {
			doc.SourceSHA256 = sourceSHA.String
		}
		if auditRef.Valid {
			doc.AuditRef = auditRef.String
		}
		if rawRef.Valid {
			doc.RawRef = rawRef.String
		}

		out = append(out, doc)
	}

	return out, rows.Err()
}

func scanCollectionAction(row rowScanner) (*action.CollectionAction, error) {
	var (
		act action.CollectionAction

		docTypeRaw  string
		statusRaw   string
		messageRaw  sql.NullString
		paramsRaw   []byte
		snapshotRaw []byte
		rerunRaw    sql.NullString
		idemRaw     sql.NullString
		startedRaw  sql.NullTime
		finishedRaw sql.NullTime
	)

	if err := row.Scan(
		&act.ID,
		&act.UserID,
		&act.CollectionID,
		&docTypeRaw,
		&act.ActionType,
		&statusRaw,
		&messageRaw,
		&paramsRaw,
		&snapshotRaw,
		&act.SnapshotHash,
		&act.SnapshotTotal,
		&rerunRaw,
		&idemRaw,
		&act.TotalCount,
		&act.SuccessCount,
		&act.WarningCount,
		&act.FailedCount,
		&act.SkippedCount,
		&startedRaw,
		&finishedRaw,
		&act.CreatedAt,
		&act.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, action.ErrActionNotFound
		}
		return nil, err
	}

	act.DocumentType = document.DocumentType(docTypeRaw)
	act.Status = action.Status(statusRaw)
	if messageRaw.Valid {
		act.Message = messageRaw.String
	}
	if len(paramsRaw) > 0 && json.Valid(paramsRaw) {
		act.ParamsJSON = paramsRaw
	}
	if len(snapshotRaw) > 0 && json.Valid(snapshotRaw) {
		act.SnapshotJSON = snapshotRaw
	}
	if rerunRaw.Valid {
		v := rerunRaw.String
		act.RerunOfAction = &v
	}
	if idemRaw.Valid {
		v := idemRaw.String
		act.IdempotencyKey = &v
	}
	if startedRaw.Valid {
		t := startedRaw.Time
		act.StartedAt = &t
	}
	if finishedRaw.Valid {
		t := finishedRaw.Time
		act.FinishedAt = &t
	}

	return &act, nil
}

func scanCollectionActionItem(row rowScanner) (*action.CollectionActionItem, error) {
	var (
		item        action.CollectionActionItem
		docIDRaw    sql.NullString
		sourceName  sql.NullString
		statusRaw   string
		messageRaw  sql.NullString
		warningsRaw []byte
		errorRaw    sql.NullString
	)

	if err := row.Scan(
		&item.ID,
		&item.ActionID,
		&docIDRaw,
		&sourceName,
		&statusRaw,
		&messageRaw,
		&warningsRaw,
		&errorRaw,
		&item.CreatedAt,
	); err != nil {
		return nil, err
	}

	if docIDRaw.Valid {
		v := docIDRaw.String
		item.DocumentID = &v
	}
	if sourceName.Valid {
		v := sourceName.String
		item.SourceName = &v
	}
	item.Status = action.ItemStatus(statusRaw)
	if messageRaw.Valid {
		item.Message = messageRaw.String
	}
	if len(warningsRaw) > 0 && json.Valid(warningsRaw) {
		item.WarningsJSON = warningsRaw
	}
	if errorRaw.Valid {
		item.Error = errorRaw.String
	}

	return &item, nil
}

func scanCollectionActionOutput(row rowScanner) (*action.CollectionActionOutput, error) {
	var (
		output      action.CollectionActionOutput
		kindRaw     string
		mimeRaw     sql.NullString
		sizeRaw     sql.NullInt64
		checksumRaw sql.NullString
	)

	if err := row.Scan(
		&output.ID,
		&output.ActionID,
		&kindRaw,
		&output.Name,
		&output.ObjectRef,
		&mimeRaw,
		&sizeRaw,
		&checksumRaw,
		&output.CreatedAt,
	); err != nil {
		return nil, err
	}

	output.Kind = action.OutputKind(kindRaw)
	if mimeRaw.Valid {
		output.MimeType = mimeRaw.String
	}
	if sizeRaw.Valid {
		output.SizeBytes = sizeRaw.Int64
	}
	if checksumRaw.Valid {
		output.Checksum = checksumRaw.String
	}

	return &output, nil
}
