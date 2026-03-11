package schema

import (
	"database/sql"
	"fmt"
	"io/fs"
	"strings"

	"github.com/Des1red/sqlitekit/internal/db"
)

func execSQLFile(files fs.FS, filename string) error {

	content, err := fs.ReadFile(files, filename)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", filename, err)
	}

	queries := strings.Split(string(content), ";")

	for _, q := range queries {

		q = strings.TrimSpace(q)

		if q == "" {
			continue
		}

		if _, err := db.DB.Exec(q); err != nil {
			return fmt.Errorf("error executing %s: %w", filename, err)
		}
	}

	return nil
}

func alreadyApplied(name string) (bool, error) {

	var exists int

	err := db.DB.QueryRow(
		"SELECT 1 FROM schema_migrations WHERE name = ? LIMIT 1",
		name,
	).Scan(&exists)

	if err == sql.ErrNoRows {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func recordMigration(name string) error {
	_, err := db.DB.Exec(
		"INSERT INTO schema_migrations(name) VALUES(?)",
		name,
	)
	return err
}
