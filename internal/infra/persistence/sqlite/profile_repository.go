package sqlite

import (
	"database/sql"

	"github.com/sieryo/invoice-extractor/internal/app/profile"
)

type ProfileRepository struct {
	db *sql.DB
}

func NewProfileRepository(db *sql.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) Create(p *profile.Profile) error {
	_, err := r.db.Exec(`
		INSERT INTO users (id, name, alias, cutoff_date, npwp, tku_id, password_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Name, p.Alias, p.CutoffDate, p.NPWP, p.TKUID, p.PasswordHash)
	return err
}

func (r *ProfileRepository) GetByID(id string) (*profile.Profile, error) {
	row := r.db.QueryRow(`
		SELECT id, name, alias, cutoff_date, npwp, tku_id, password_hash, created_at
		FROM users
		WHERE id = ?
	`, id)

	var p profile.Profile
	if err := row.Scan(&p.ID, &p.Name, &p.Alias, &p.CutoffDate, &p.NPWP, &p.TKUID, &p.PasswordHash, &p.CreatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProfileRepository) GetByName(name string) (*profile.Profile, error) {
	row := r.db.QueryRow(`
		SELECT id, name, alias, cutoff_date, npwp, tku_id, password_hash, created_at
		FROM users
		WHERE name = ?
	`, name)

	var p profile.Profile
	if err := row.Scan(&p.ID, &p.Name, &p.Alias, &p.CutoffDate, &p.NPWP, &p.TKUID, &p.PasswordHash, &p.CreatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProfileRepository) GetByAlias(alias string) (*profile.Profile, error) {
	row := r.db.QueryRow(`
		SELECT id, name, alias, cutoff_date, npwp, tku_id, password_hash, created_at
		FROM users
		WHERE alias = ?
	`, alias)

	var p profile.Profile
	if err := row.Scan(&p.ID, &p.Name, &p.Alias, &p.CutoffDate, &p.NPWP, &p.TKUID, &p.PasswordHash, &p.CreatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProfileRepository) List() ([]profile.Profile, error) {
	rows, err := r.db.Query(`
		SELECT id, name, alias, cutoff_date, npwp, tku_id, password_hash, created_at
		FROM users
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []profile.Profile
	for rows.Next() {
		var p profile.Profile
		if err := rows.Scan(&p.ID, &p.Name, &p.Alias, &p.CutoffDate, &p.NPWP, &p.TKUID, &p.PasswordHash, &p.CreatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

