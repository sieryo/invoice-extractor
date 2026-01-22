package database

import (
	"database/sql"
	"os"
)

func RunMigration(db *sql.DB, path string) error {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	_, err = db.Exec(string(sqlBytes))
	return err
}
