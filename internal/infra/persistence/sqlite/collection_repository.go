package sqlite

import (
	"context"
	"database/sql"
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
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO collections (
			id, user_id, name, status, created_at, expired_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`,
		c.ID,
		c.UserID,
		c.Name,
		c.Status,
		c.CreatedAt,
		c.ExpiredAt,
	)
	return err
}

func (r *CollectionRepository) FindByID(
	ctx context.Context,
	id string,
) (*collection.Collection, error) {

	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, status, created_at, expired_at
		FROM collections
		WHERE id = ?
	`, id)

	var c collection.Collection
	if err := row.Scan(
		&c.ID,
		&c.UserID,
		&c.Name,
		&c.Status,
		&c.CreatedAt,
		&c.ExpiredAt,
	); err != nil {
		return nil, err
	}

	return &c, nil
}

func (r *CollectionRepository) ListByUserID(
	ctx context.Context,
	userID string,
) ([]*collection.Collection, error) {

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, name, status, created_at, expired_at
		FROM collections
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*collection.Collection

	for rows.Next() {
		var c collection.Collection
		if err := rows.Scan(
			&c.ID,
			&c.UserID,
			&c.Name,
			&c.Status,
			&c.CreatedAt,
			&c.ExpiredAt,
		); err != nil {
			return nil, err
		}
		res = append(res, &c)
	}

	return res, rows.Err()
}

func (r *CollectionRepository) UpdateStatus(
	ctx context.Context,
	id string,
	status collection.Status,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE collections
		SET status = ?
		WHERE id = ?
	`, status, id)
	return err
}

func (r *CollectionRepository) Expire(
	ctx context.Context,
	now time.Time,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE collections
		SET status = ?, expired_at = ?
		WHERE status != ?
		  AND expired_at IS NOT NULL
		  AND expired_at <= ?
	`,
		collection.StatusExpired,
		now,
		collection.StatusExpired,
		now,
	)
	return err
}
