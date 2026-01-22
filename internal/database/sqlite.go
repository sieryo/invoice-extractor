package database

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

func NewSQLite(path string) *sql.DB {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal("failed to open sqlite:", err)
	}

	// SQLite recommended pragmas
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
