package repository

import (
	"database/sql"

	"github.com/sieryo/invoice-extractor/internal/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(u *model.User) error {
	_, err := r.db.Exec(`
		INSERT INTO users (id, username, password_hash)
		VALUES (?, ?, ?)
	`, u.ID, u.Username, u.PasswordHash)

	return err
}

func (r *UserRepository) GetByID(id string) (*model.User, error) {
	row := r.db.QueryRow(`
		SELECT id, username, password_hash, created_at
		FROM users
		WHERE id = ?
	`, id)

	var u model.User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	row := r.db.QueryRow(`
		SELECT id, username, password_hash, created_at
		FROM users
		WHERE username = ?
	`, username)

	var u model.User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) List() ([]model.User, error) {
	rows, err := r.db.Query(`
		SELECT id, username, password_hash, created_at
		FROM users
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *UserRepository) Delete(id string) error {
	_, err := r.db.Exec(`
		DELETE FROM users WHERE id = ?
	`, id)
	return err
}

func (r *UserRepository) BulkDelete(ids []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`DELETE FROM users WHERE id = ?`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, id := range ids {
		if _, err := stmt.Exec(id); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
