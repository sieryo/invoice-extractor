package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, filename := range sqlFiles {
		var exists bool
		err := db.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = ?)",
			filename,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", filename, err)
		}

		if exists {
			continue
		}

		slog.Info("Applying migration", "filename", filename)

		content, err := fs.ReadFile(
			migrationsFS,
			"migrations/"+filename,
		)

		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to exec %s: %w", filename, err)
		}

		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (filename) VALUES (?)",
			filename,
		); err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}
