package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/ingest"
)

type DocumentRepositoryV2 struct {
	db *sql.DB
}

func NewDocumentRepositoryV2(db *sql.DB) *DocumentRepositoryV2 {
	return &DocumentRepositoryV2{db: db}
}

func (r *DocumentRepositoryV2) FindByID(
	ctx context.Context,
	id string,
) (*ingest.DocumentRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, user_id, collection_id, document_type,
			document_tag, source_name, source_size_bytes, source_mime, source_sha256, source_order,
			status, message, normalized_ref, audit_ref, raw_ref
		FROM documents
		WHERE id = ?
		LIMIT 1
	`, id)

	var (
		doc ingest.DocumentRecord

		documentTag sql.NullString
		sourceSize  sql.NullInt64
		sourceMime  sql.NullString
		message     sql.NullString
		auditRef    sql.NullString
		rawRef      sql.NullString
	)

	if err := row.Scan(
		&doc.ID,
		&doc.UserID,
		&doc.CollectionID,
		&doc.DocumentType,
		&documentTag,
		&doc.SourceName,
		&sourceSize,
		&sourceMime,
		&doc.SourceSHA256,
		&doc.SourceOrder,
		&doc.Status,
		&message,
		&doc.NormalizedRef,
		&auditRef,
		&rawRef,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if sourceSize.Valid {
		doc.SourceSizeBytes = sourceSize.Int64
	}
	if documentTag.Valid {
		doc.DocumentTag = documentTag.String
	}
	if sourceMime.Valid {
		doc.SourceMIME = sourceMime.String
	}
	if message.Valid {
		doc.Message = message.String
	}
	if auditRef.Valid {
		v := auditRef.String
		doc.AuditRef = &v
	}
	if rawRef.Valid {
		v := rawRef.String
		doc.RawRef = &v
	}

	return &doc, nil
}

func (r *DocumentRepositoryV2) FindActiveByHash(
	ctx context.Context,
	collectionID string,
	documentType string,
	sha256 string,
) (*ingest.DocumentRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, user_id, collection_id, document_type,
			document_tag, source_name, source_size_bytes, source_mime, source_sha256, source_order,
			status, message, normalized_ref, audit_ref, raw_ref
		FROM documents
		WHERE collection_id = ?
		  AND document_type = ?
		  AND source_sha256 = ?
		  AND deleted_at IS NULL
		LIMIT 1
	`, collectionID, documentType, sha256)

	var (
		doc ingest.DocumentRecord

		documentTag sql.NullString
		sourceSize  sql.NullInt64
		sourceMime  sql.NullString
		message     sql.NullString
		auditRef    sql.NullString
		rawRef      sql.NullString
	)

	if err := row.Scan(
		&doc.ID,
		&doc.UserID,
		&doc.CollectionID,
		&doc.DocumentType,
		&documentTag,
		&doc.SourceName,
		&sourceSize,
		&sourceMime,
		&doc.SourceSHA256,
		&doc.SourceOrder,
		&doc.Status,
		&message,
		&doc.NormalizedRef,
		&auditRef,
		&rawRef,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if sourceSize.Valid {
		doc.SourceSizeBytes = sourceSize.Int64
	}
	if documentTag.Valid {
		doc.DocumentTag = documentTag.String
	}
	if sourceMime.Valid {
		doc.SourceMIME = sourceMime.String
	}
	if message.Valid {
		doc.Message = message.String
	}
	if auditRef.Valid {
		v := auditRef.String
		doc.AuditRef = &v
	}
	if rawRef.Valid {
		v := rawRef.String
		doc.RawRef = &v
	}

	return &doc, nil
}

func (r *DocumentRepositoryV2) Create(ctx context.Context, doc *ingest.DocumentRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO documents (
			id, user_id, collection_id, document_type,
			document_tag, source_name, source_size_bytes, source_mime, source_sha256, source_order,
			status, message, normalized_ref, audit_ref, raw_ref
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		doc.ID,
		doc.UserID,
		doc.CollectionID,
		doc.DocumentType,
		doc.DocumentTag,
		doc.SourceName,
		doc.SourceSizeBytes,
		doc.SourceMIME,
		doc.SourceSHA256,
		doc.SourceOrder,
		doc.Status,
		doc.Message,
		doc.NormalizedRef,
		doc.AuditRef,
		doc.RawRef,
	)
	return err
}

func (r *DocumentRepositoryV2) ListByCollection(
	ctx context.Context,
	collectionID string,
	status string,
	limit int,
	offset int,
) ([]*ingest.DocumentRecord, error) {
	query := `
		SELECT
			id, user_id, collection_id, document_type,
			document_tag, source_name, source_size_bytes, source_mime, source_sha256, source_order,
			status, message, normalized_ref, audit_ref, raw_ref
		FROM documents
		WHERE collection_id = ?
		  AND deleted_at IS NULL
	`
	args := []any{collectionID}

	normalizedStatus := strings.ToLower(strings.TrimSpace(status))
	if normalizedStatus != "" {
		query += ` AND status = ?`
		args = append(args, normalizedStatus)
	}

	query += ` ORDER BY source_order ASC, id ASC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*ingest.DocumentRecord, 0)
	for rows.Next() {
		var (
			doc ingest.DocumentRecord

			documentTag sql.NullString
			sourceSize  sql.NullInt64
			sourceMime  sql.NullString
			message     sql.NullString
			auditRef    sql.NullString
			rawRef      sql.NullString
		)

		if err := rows.Scan(
			&doc.ID,
			&doc.UserID,
			&doc.CollectionID,
			&doc.DocumentType,
			&documentTag,
			&doc.SourceName,
			&sourceSize,
			&sourceMime,
			&doc.SourceSHA256,
			&doc.SourceOrder,
			&doc.Status,
			&message,
			&doc.NormalizedRef,
			&auditRef,
			&rawRef,
		); err != nil {
			return nil, err
		}

		if sourceSize.Valid {
			doc.SourceSizeBytes = sourceSize.Int64
		}
		if documentTag.Valid {
			doc.DocumentTag = documentTag.String
		}
		if sourceMime.Valid {
			doc.SourceMIME = sourceMime.String
		}
		if message.Valid {
			doc.Message = message.String
		}
		if auditRef.Valid {
			v := auditRef.String
			doc.AuditRef = &v
		}
		if rawRef.Valid {
			v := rawRef.String
			doc.RawRef = &v
		}

		out = append(out, &doc)
	}

	return out, rows.Err()
}
