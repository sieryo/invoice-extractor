package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/sieryo/invoice-extractor/internal/domain/collection"
)

type CollectionRepository struct {
	db *sql.DB
}

func NewCollectionRepository(db *sql.DB) *CollectionRepository {
	return &CollectionRepository{db: db}
}

func (r *CollectionRepository) Create(
	ctx context.Context,
	c *collection.Collection,
) error {
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}

	c.SyncLegacyStatus()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO collections (
			id, user_id, parent_id, name, node_type, document_type, phase,
			total_count, ready_count, warning_count, failed_count, duplicate_count,
			created_at, updated_at, deleted_at, deleted_by, delete_reason
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		c.ID,
		c.UserID,
		c.Parent,
		c.Name,
		c.NodeType,
		c.DocumentType,
		c.Phase,
		c.TotalCount,
		c.ReadyCount,
		c.WarningCount,
		c.FailedCount,
		c.DuplicateCount,
		c.CreatedAt,
		c.UpdatedAt,
		c.DeletedAt,
		c.DeletedBy,
		c.DeleteReason,
	)
	return err
}

func (r *CollectionRepository) FindByID(
	ctx context.Context,
	id string,
) (*collection.Collection, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, user_id, parent_id, name, node_type, document_type, phase,
			total_count, ready_count, warning_count, failed_count, duplicate_count,
			created_at, updated_at, deleted_at, deleted_by, delete_reason
		FROM collections
		WHERE id = ?
	`, id)

	c, err := scanCollection(row)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (r *CollectionRepository) ListByUserID(
	ctx context.Context,
	userID string,
) ([]*collection.Collection, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id, user_id, parent_id, name, node_type, document_type, phase,
			total_count, ready_count, warning_count, failed_count, duplicate_count,
			created_at, updated_at, deleted_at, deleted_by, delete_reason
		FROM collections
		WHERE user_id = ?
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCollections(rows)
}

func (r *CollectionRepository) ListChildren(
	ctx context.Context,
	userID string,
	parentID *string,
) ([]*collection.Collection, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if parentID == nil {
		rows, err = r.db.QueryContext(ctx, `
			SELECT
				id, user_id, parent_id, name, node_type, document_type, phase,
				total_count, ready_count, warning_count, failed_count, duplicate_count,
				created_at, updated_at, deleted_at, deleted_by, delete_reason
			FROM collections
			WHERE user_id = ?
			  AND parent_id IS NULL
			  AND deleted_at IS NULL
			ORDER BY created_at DESC
		`, userID)
	} else {
		rows, err = r.db.QueryContext(ctx, `
			SELECT
				id, user_id, parent_id, name, node_type, document_type, phase,
				total_count, ready_count, warning_count, failed_count, duplicate_count,
				created_at, updated_at, deleted_at, deleted_by, delete_reason
			FROM collections
			WHERE user_id = ?
			  AND parent_id = ?
			  AND deleted_at IS NULL
			ORDER BY created_at DESC
		`, userID, *parentID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCollections(rows)
}

func (r *CollectionRepository) UpdatePhase(
	ctx context.Context,
	id string,
	phase collection.Phase,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE collections
		SET phase = ?, updated_at = ?
		WHERE id = ?
		  AND deleted_at IS NULL
	`, phase, time.Now(), id)
	return err
}

func (r *CollectionRepository) UpdateSummary(
	ctx context.Context,
	id string,
	total int,
	ready int,
	warning int,
	failed int,
	duplicate int,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE collections
		SET
			total_count = ?,
			ready_count = ?,
			warning_count = ?,
			failed_count = ?,
			duplicate_count = ?,
			updated_at = ?
		WHERE id = ?
		  AND deleted_at IS NULL
	`, total, ready, warning, failed, duplicate, time.Now(), id)
	return err
}

func (r *CollectionRepository) Restore(
	ctx context.Context,
	id string,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE collections
		SET deleted_at = NULL, deleted_by = NULL, delete_reason = NULL, updated_at = ?
		WHERE id = ?
	`, time.Now(), id)
	return err
}

func (r *CollectionRepository) UpdateStatus(
	ctx context.Context,
	id string,
	status collection.Status,
) error {
	phase := collection.PhaseReady
	switch strings.ToLower(string(status)) {
	case string(collection.StatusCommitted):
		phase = collection.PhaseProcessing
	case string(collection.StatusExpired):
		phase = collection.PhaseReady
	}

	return r.UpdatePhase(ctx, id, phase)
}

func (r *CollectionRepository) UpdateName(
	ctx context.Context,
	id string,
	name string,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE collections
		SET name = ?, updated_at = ?
		WHERE id = ?
		  AND deleted_at IS NULL
	`, name, time.Now(), id)
	return err
}

func (r *CollectionRepository) Delete(
	ctx context.Context,
	id string,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE collections
		SET deleted_at = ?, updated_at = ?
		WHERE id = ?
		  AND deleted_at IS NULL
	`, time.Now(), time.Now(), id)
	return err
}

func scanCollections(rows *sql.Rows) ([]*collection.Collection, error) {
	var res []*collection.Collection
	for rows.Next() {
		c, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, c)
	}
	return res, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCollection(row scanner) (*collection.Collection, error) {
	var (
		c            collection.Collection
		parentID     sql.NullString
		docType      sql.NullString
		deletedAt    sql.NullTime
		deletedBy    sql.NullString
		deleteReason sql.NullString
	)

	if err := row.Scan(
		&c.ID,
		&c.UserID,
		&parentID,
		&c.Name,
		&c.NodeType,
		&docType,
		&c.Phase,
		&c.TotalCount,
		&c.ReadyCount,
		&c.WarningCount,
		&c.FailedCount,
		&c.DuplicateCount,
		&c.CreatedAt,
		&c.UpdatedAt,
		&deletedAt,
		&deletedBy,
		&deleteReason,
	); err != nil {
		return nil, err
	}

	if parentID.Valid {
		c.Parent = &parentID.String
	}
	if docType.Valid && docType.String != "" {
		d := collection.DocumentType(docType.String)
		c.DocumentType = &d
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		c.DeletedAt = &t
	}
	if deletedBy.Valid {
		v := deletedBy.String
		c.DeletedBy = &v
	}
	if deleteReason.Valid {
		v := deleteReason.String
		c.DeleteReason = &v
	}

	c.SyncLegacyStatus()
	return &c, nil
}
