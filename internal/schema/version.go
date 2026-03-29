package schema

import (
	"database/sql"
	"fmt"

	"github.com/Des1red/gosqlitekit/internal/db"
)

const (
	SQLiteKitVersion    = "dev"
	SchemaFormatVersion = "v1.1.0"
)

func ensureMetaTable() error {
	_, err := db.DB.Exec(`
	CREATE TABLE IF NOT EXISTS schema_meta (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`)
	if err != nil {
		return fmt.Errorf("sqlitekit: create schema_meta table: %w", err)
	}

	return nil
}

func getMetaValue(key string) (string, error) {
	var value string

	err := db.DB.QueryRow(`
		SELECT value
		FROM schema_meta
		WHERE key = ?
		LIMIT 1
	`, key).Scan(&value)

	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("sqlitekit: load schema meta %s: %w", key, err)
	}

	return value, nil
}

func setMetaValue(key, value string) error {
	_, err := db.DB.Exec(`
		INSERT INTO schema_meta(key, value, updated_at)
		VALUES(?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = CURRENT_TIMESTAMP
	`, key, value)
	if err != nil {
		return fmt.Errorf("sqlitekit: set schema meta %s: %w", key, err)
	}

	return nil
}

func syncSchemaVersionMeta() error {
	if err := ensureMetaTable(); err != nil {
		return err
	}

	incompatible, err := sqlitekitTablesNeedRebuild()
	if err != nil {
		return err
	}

	storedKitVersion, err := getMetaValue("sqlitekit_version")
	if err != nil {
		return err
	}

	storedSchemaFormat, err := getMetaValue("schema_format_version")
	if err != nil {
		return err
	}

	if incompatible || shouldRebuildInternalMetadata(storedKitVersion, storedSchemaFormat) {
		if err := rebuildInternalMetadataTables(); err != nil {
			return err
		}
	}

	if err := setMetaValue("sqlitekit_version", SQLiteKitVersion); err != nil {
		return err
	}

	if err := setMetaValue("schema_format_version", SchemaFormatVersion); err != nil {
		return err
	}

	return nil
}

func shouldRebuildInternalMetadata(storedKitVersion, storedSchemaFormat string) bool {
	if storedKitVersion == "" && storedSchemaFormat == "" {
		return false
	}

	if storedKitVersion == SQLiteKitVersion {
		return false
	}

	return storedSchemaFormat != "" && storedSchemaFormat != SchemaFormatVersion
}

func rebuildInternalMetadataTables() error {
	_, err := db.DB.Exec(`
	DROP TABLE IF EXISTS schema_migration_statements;
	DROP TABLE IF EXISTS schema_migration_files;
	`)
	if err != nil {
		return fmt.Errorf("sqlitekit: rebuild internal metadata tables: %w", err)
	}

	return nil
}

func sqlitekitTablesNeedRebuild() (bool, error) {
	ok, err := tableHasRequiredColumns("schema_migration_files",
		"name", "checksum", "stmt_count", "applied_at", "updated_at",
	)
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}

	ok, err = tableHasRequiredColumns("schema_migration_statements",
		"file_name", "stmt_index", "stmt_hash", "stmt_sql", "applied_at",
	)
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}

	ok, err = tableHasPrimaryKeyColumns("schema_migration_files", "name")
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}

	ok, err = tableHasPrimaryKeyColumns("schema_migration_statements", "file_name", "stmt_index")
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}

	ok, err = tableHasForeignKey("schema_migration_statements", "file_name", "schema_migration_files", "name")
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}

	return false, nil
}

func tableHasRequiredColumns(table string, required ...string) (bool, error) {
	exists, err := tableExists(table)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	rows, err := db.DB.Query(fmt.Sprintf(`PRAGMA table_info(%s);`, table))
	if err != nil {
		return false, fmt.Errorf("sqlitekit: inspect columns for %s: %w", table, err)
	}
	defer rows.Close()

	cols := make(map[string]bool)

	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var dfltValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return false, fmt.Errorf("sqlitekit: scan table_info for %s: %w", table, err)
		}

		cols[name] = true
	}

	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("sqlitekit: iterate table_info for %s: %w", table, err)
	}

	for _, name := range required {
		if !cols[name] {
			return false, nil
		}
	}

	return true, nil
}

func tableHasPrimaryKeyColumns(table string, expected ...string) (bool, error) {
	exists, err := tableExists(table)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	rows, err := db.DB.Query(fmt.Sprintf(`PRAGMA table_info(%s);`, table))
	if err != nil {
		return false, fmt.Errorf("sqlitekit: inspect primary key for %s: %w", table, err)
	}
	defer rows.Close()

	pkCols := make(map[int]string)

	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var dfltValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return false, fmt.Errorf("sqlitekit: scan primary key info for %s: %w", table, err)
		}

		if pk > 0 {
			pkCols[pk] = name
		}
	}

	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("sqlitekit: iterate primary key info for %s: %w", table, err)
	}

	if len(pkCols) != len(expected) {
		return false, nil
	}

	for i, name := range expected {
		if pkCols[i+1] != name {
			return false, nil
		}
	}

	return true, nil
}

func tableHasForeignKey(table, fromCol, refTable, refCol string) (bool, error) {
	exists, err := tableExists(table)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	rows, err := db.DB.Query(fmt.Sprintf(`PRAGMA foreign_key_list(%s);`, table))
	if err != nil {
		return false, fmt.Errorf("sqlitekit: inspect foreign keys for %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var seq int
		var tableName string
		var from string
		var to string
		var onUpdate string
		var onDelete string
		var match string

		if err := rows.Scan(&id, &seq, &tableName, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return false, fmt.Errorf("sqlitekit: scan foreign key info for %s: %w", table, err)
		}

		if tableName == refTable && from == fromCol && to == refCol {
			return true, nil
		}
	}

	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("sqlitekit: iterate foreign key info for %s: %w", table, err)
	}

	return false, nil
}

func tableExists(name string) (bool, error) {
	var found string

	err := db.DB.QueryRow(`
		SELECT name
		FROM sqlite_master
		WHERE type = 'table' AND name = ?
		LIMIT 1
	`, name).Scan(&found)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("sqlitekit: check table existence for %s: %w", name, err)
	}

	return true, nil
}
