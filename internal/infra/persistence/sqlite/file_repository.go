package sqlite

import (
	"context"
	"database/sql"

	"github.com/sieryo/invoice-extractor/internal/domain/file"
)

type FileRepository struct {
	db *sql.DB
}

func NewFileRepository(db *sql.DB) *FileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) Create(
	ctx context.Context,
	f *file.FileObject,
) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO files (
			id, collection_id, name, state, path, size, mime_type
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		f.ID,
		f.CollectionID,
		f.Name,
		f.State,
		f.Path,
		f.Size,
		f.MimeType,
	)
	return err
}

func (r *FileRepository) CreateBulk(
	ctx context.Context,
	files []*file.FileObject,
) error {

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO files (
			id, collection_id, name, state, path, size, mime_type
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, f := range files {
		_, err := stmt.Exec(
			f.ID,
			f.CollectionID,
			f.Name,
			f.State,
			f.Path,
			f.Size,
			f.MimeType,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *FileRepository) FindByID(
	ctx context.Context,
	id string,
) (*file.FileObject, error) {

	row := r.db.QueryRowContext(ctx, `
		SELECT id, collection_id, name, state, path, size, mime_type
		FROM files
		WHERE id = ?
	`, id)

	var f file.FileObject
	if err := row.Scan(
		&f.ID,
		&f.CollectionID,
		&f.Name,
		&f.State,
		&f.Path,
		&f.Size,
		&f.MimeType,
	); err != nil {
		return nil, err
	}

	return &f, nil
}

func (r *FileRepository) ListByCollection(
	ctx context.Context,
	collectionID string,
) ([]*file.FileObject, error) {

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, collection_id, name, state, path, size, mime_type
		FROM files
		WHERE collection_id = ?
		ORDER BY created_at ASC
	`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*file.FileObject

	for rows.Next() {
		var f file.FileObject
		if err := rows.Scan(
			&f.ID,
			&f.CollectionID,
			&f.Name,
			&f.State,
			&f.Path,
			&f.Size,
			&f.MimeType,
		); err != nil {
			return nil, err
		}
		res = append(res, &f)
	}

	return res, rows.Err()
}

func (r *FileRepository) UpdateState(
	ctx context.Context,
	id string,
	state file.FileState,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE files
		SET state = ?
		WHERE id = ?
	`, state, id)
	return err
}

func (r *FileRepository) Delete(
	ctx context.Context,
	id string,
) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM files
		WHERE id = ?
	`, id)
	return err
}

func (r *FileRepository) DeleteBulk(
	ctx context.Context,
	ids []string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		DELETE FROM files
		WHERE id = ?
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, id := range ids {
		if _, err := stmt.ExecContext(ctx, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *FileRepository) DeleteByCollection(
	ctx context.Context,
	collectionID string,
) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM files
		WHERE collection_id = ?
	`, collectionID)
	return err
}
