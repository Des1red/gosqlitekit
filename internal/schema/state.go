package schema

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Des1red/gosqlitekit/internal/db"
)

type StoredFile struct {
	Name      string
	Checksum  string
	StmtCount int
	AppliedAt time.Time
	UpdatedAt time.Time
}

type StoredStatement struct {
	FileName  string
	StmtIndex int
	StmtHash  string
	StmtSQL   string
	AppliedAt time.Time
}

type MetaValue struct {
	Key   string
	Value string
}

func ensureMigrationTables() error {
	_, err := db.DB.Exec(`
	CREATE TABLE IF NOT EXISTS schema_migration_files (
		name TEXT PRIMARY KEY,
		checksum TEXT NOT NULL,
		stmt_count INTEGER NOT NULL,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS schema_migration_statements (
		file_name TEXT NOT NULL,
		stmt_index INTEGER NOT NULL,
		stmt_hash TEXT NOT NULL,
		stmt_sql TEXT NOT NULL,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (file_name, stmt_index),
		FOREIGN KEY (file_name) REFERENCES schema_migration_files(name) ON DELETE CASCADE
	);
	`)
	if err != nil {
		return fmt.Errorf("sqlitekit: create migration metadata tables: %w", err)
	}
	return nil
}

func getStoredFile(name string) (*StoredFile, error) {
	var f StoredFile

	err := db.DB.QueryRow(`
		SELECT name, checksum, stmt_count, applied_at, updated_at
		FROM schema_migration_files
		WHERE name = ?
		LIMIT 1
	`, name).Scan(
		&f.Name,
		&f.Checksum,
		&f.StmtCount,
		&f.AppliedAt,
		&f.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("sqlitekit: load migration file metadata for %s: %w", name, err)
	}

	return &f, nil
}

func getStoredStatements(name string) ([]StoredStatement, error) {
	rows, err := db.DB.Query(`
		SELECT file_name, stmt_index, stmt_hash, stmt_sql, applied_at
		FROM schema_migration_statements
		WHERE file_name = ?
		ORDER BY stmt_index ASC
	`, name)
	if err != nil {
		return nil, fmt.Errorf("sqlitekit: load migration statements for %s: %w", name, err)
	}
	defer rows.Close()

	var stmts []StoredStatement

	for rows.Next() {
		var s StoredStatement
		if err := rows.Scan(
			&s.FileName,
			&s.StmtIndex,
			&s.StmtHash,
			&s.StmtSQL,
			&s.AppliedAt,
		); err != nil {
			return nil, fmt.Errorf("sqlitekit: scan migration statement for %s: %w", name, err)
		}
		stmts = append(stmts, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlitekit: iterate migration statements for %s: %w", name, err)
	}

	return stmts, nil
}

func insertFileMetadata(tx *sql.Tx, file FileState) error {
	_, err := tx.Exec(`
		INSERT INTO schema_migration_files(name, checksum, stmt_count)
		VALUES(?, ?, ?)
	`, file.Name, file.Checksum, len(file.Statements))
	if err != nil {
		return fmt.Errorf("sqlitekit: insert migration file metadata for %s: %w", file.Name, err)
	}
	return nil
}

func insertStatementMetadata(tx *sql.Tx, fileName string, stmts []Statement) error {
	for _, stmt := range stmts {
		_, err := tx.Exec(`
			INSERT INTO schema_migration_statements(file_name, stmt_index, stmt_hash, stmt_sql)
			VALUES(?, ?, ?, ?)
		`, fileName, stmt.Index, stmt.Hash, stmt.SQL)
		if err != nil {
			return fmt.Errorf("sqlitekit: insert migration statement metadata for %s statement %d: %w", fileName, stmt.Index, err)
		}
	}
	return nil
}

func updateFileMetadata(tx *sql.Tx, file FileState) error {
	_, err := tx.Exec(`
		UPDATE schema_migration_files
		SET checksum = ?, stmt_count = ?, updated_at = CURRENT_TIMESTAMP
		WHERE name = ?
	`, file.Checksum, len(file.Statements), file.Name)
	if err != nil {
		return fmt.Errorf("sqlitekit: update migration file metadata for %s: %w", file.Name, err)
	}
	return nil
}
