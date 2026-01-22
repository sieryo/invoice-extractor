package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func NewSQLite(path string) *sql.DB {
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatal("failed to create db directory:", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal("failed to open sqlite:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("failed to ping db:", err)
	}

	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
	}

	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			log.Fatal(err)
		}
	}

	return db
}
