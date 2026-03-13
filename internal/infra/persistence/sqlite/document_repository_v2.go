package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/sieryo/invoice-extractor/internal/app/ingest"
)

type DocumentRepositoryV2 struct {
	db *sql.DB
}

func NewDocumentRepositoryV2(db *sql.DB) *DocumentRepositoryV2 {
	return &DocumentRepositoryV2{db: db}
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
			source_name, source_size_bytes, source_mime, source_sha256, source_order,
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

		sourceSize sql.NullInt64
		sourceMime sql.NullString
		message    sql.NullString
		auditRef   sql.NullString
		rawRef     sql.NullString
	)

	if err := row.Scan(
		&doc.ID,
		&doc.UserID,
		&doc.CollectionID,
		&doc.DocumentType,
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
			source_name, source_size_bytes, source_mime, source_sha256, source_order,
			status, message, normalized_ref, audit_ref, raw_ref
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		doc.ID,
		doc.UserID,
		doc.CollectionID,
		doc.DocumentType,
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
