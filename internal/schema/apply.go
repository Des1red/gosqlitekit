package schema

import (
	"io/fs"
	"path/filepath"
	"sort"

	"github.com/Des1red/gosqlitekit/internal/db"
	"github.com/Des1red/gosqlitekit/internal/logs"
)

func Apply(files fs.FS) error {

	appliedCount := 0
	skipCount := 0
	failCount := 0

	_, err := db.DB.Exec(`
	CREATE TABLE IF NOT EXISTS schema_migrations (
	    name TEXT PRIMARY KEY,
	    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`)
	if err != nil {
		return err
	}

	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return err
	}

	var sqlFiles []string

	for _, f := range entries {
		if filepath.Ext(f.Name()) == ".sql" {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}

	sort.Strings(sqlFiles)

	width := logs.GetWidth(sqlFiles)
	logs.PrintBox(width)

	for _, name := range sqlFiles {

		applied, err := alreadyApplied(name)
		if err != nil {
			failCount++
			logs.PrintMigration("FAIL", name, width)
			logs.PrintSummary(appliedCount, skipCount, failCount)
			return err
		}

		if applied {
			skipCount++
			logs.PrintMigration("SKIP", name, width)
			continue
		}

		if err := execSQLFile(files, name); err != nil {
			failCount++
			logs.PrintMigration("FAIL", name, width)
			return err
		}

		if err := recordMigration(name); err != nil {
			failCount++
			logs.PrintMigration("FAIL", name, width)
			return err
		}

		appliedCount++
		logs.PrintMigration("APPLY", name, width)
	}

	logs.PrintFooter(width)
	logs.PrintSummary(appliedCount, skipCount, failCount)

	return nil
}
