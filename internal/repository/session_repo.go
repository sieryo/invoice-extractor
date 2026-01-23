package repository

import (
	"database/sql"
	"time"

	"github.com/sieryo/invoice-extractor/internal/models"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(s *models.Session) error {
	_, err := r.db.Exec(`
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES (?, ?, ?)
	`, s.ID, s.UserID, s.ExpiresAt)
	return err
}

func (r *SessionRepository) GetByID(id string) (*models.Session, error) {
	row := r.db.QueryRow(`
		SELECT id, user_id, expires_at
		FROM sessions
		WHERE id = ?
	`, id)

	var s models.Session
	if err := row.Scan(&s.ID, &s.UserID, &s.ExpiresAt); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepository) Delete(id string) error {
	_, err := r.db.Exec(`
		DELETE FROM sessions
		WHERE id = ?
	`, id)
	return err
}

func (r *SessionRepository) CleanupExpired() error {
	_, err := r.db.Exec(`
		DELETE FROM sessions
		WHERE expires_at < ?
	`, time.Now())
	return err
}
