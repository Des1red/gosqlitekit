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

	if err := syncSchemaVersionMeta(); err != nil {
		return err
	}

	if err := ensureMigrationTables(); err != nil {
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

	fail := func(name string, err error) error {
		failCount++
		logs.PrintMigration("FAIL", name, width)
		logs.PrintSummary(appliedCount, skipCount, failCount)
		return err
	}

	for _, name := range sqlFiles {
		file, err := loadFileState(files, name)
		if err != nil {
			return fail(name, err)
		}

		storedFile, err := getStoredFile(name)
		if err != nil {
			return fail(name, err)
		}

		if storedFile == nil {
			if err := applyNewFile(file); err != nil {
				return fail(name, err)
			}

			appliedCount++
			logs.PrintMigration("APPLY", name, width)
			continue
		}

		if storedFile.Checksum == file.Checksum && storedFile.StmtCount == len(file.Statements) {
			skipCount++
			logs.PrintMigration("SKIP", name, width)
			continue
		}

		storedStatements, err := getStoredStatements(name)
		if err != nil {
			return fail(name, err)
		}

		appendFrom, changed, err := validateAppendOnly(*storedFile, storedStatements, file)
		if err != nil {
			return fail(name, err)
		}

		if !changed {
			skipCount++
			logs.PrintMigration("SKIP", name, width)
			continue
		}

		if err := applyAppendedStatements(file, appendFrom); err != nil {
			return fail(name, err)
		}

		appliedCount++
		logs.PrintMigration("APPLY", name, width)
	}

	logs.PrintFooter(width)
	logs.PrintSummary(appliedCount, skipCount, failCount)

	return nil
}

func applyNewFile(file FileState) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}

	if err := execStatements(tx, file.Name, file.Statements); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := insertFileMetadata(tx, file); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := insertStatementMetadata(tx, file.Name, file.Statements); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func applyAppendedStatements(file FileState, appendFrom int) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}

	newStatements := file.Statements[appendFrom:]

	if err := execStatements(tx, file.Name, newStatements); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := insertStatementMetadata(tx, file.Name, newStatements); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := updateFileMetadata(tx, file); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
