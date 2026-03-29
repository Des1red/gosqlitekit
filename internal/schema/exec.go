package schema

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"strings"
)

type Statement struct {
	Index int
	SQL   string
	Hash  string
}

type FileState struct {
	Name       string
	Checksum   string
	Statements []Statement
}

func loadFileState(files fs.FS, filename string) (FileState, error) {
	content, err := fs.ReadFile(files, filename)
	if err != nil {
		return FileState{}, fmt.Errorf("sqlitekit: read migration file %s: %w", filename, err)
	}

	statements, checksum, err := parseStatements(string(content))
	if err != nil {
		return FileState{}, fmt.Errorf("sqlitekit: parse migration file %s: %w", filename, err)
	}

	return FileState{
		Name:       filename,
		Checksum:   checksum,
		Statements: statements,
	}, nil
}

func execStatements(tx *sql.Tx, filename string, stmts []Statement) error {
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt.SQL); err != nil {
			return fmt.Errorf(
				"sqlitekit: execute migration %s statement %d: %w",
				filename,
				stmt.Index,
				err,
			)
		}
	}
	return nil
}

func validateAppendOnly(storedFile StoredFile, stored []StoredStatement, current FileState) (appendFrom int, changed bool, err error) {
	if len(stored) != storedFile.StmtCount {
		return 0, false, fmt.Errorf(
			"sqlitekit: migration metadata mismatch for %s: file stmt_count=%d stored statements=%d",
			current.Name,
			storedFile.StmtCount,
			len(stored),
		)
	}

	if len(current.Statements) < storedFile.StmtCount {
		return 0, false, fmt.Errorf(
			"sqlitekit: migration file changed incompatibly: %s now has fewer statements (%d) than previously applied (%d)",
			current.Name,
			len(current.Statements),
			storedFile.StmtCount,
		)
	}

	for i := 0; i < storedFile.StmtCount; i++ {
		if i >= len(stored) {
			return 0, false, fmt.Errorf(
				"sqlitekit: migration metadata mismatch for %s at statement %d",
				current.Name,
				i,
			)
		}
		if stored[i].StmtIndex != i {
			return 0, false, fmt.Errorf(
				"sqlitekit: migration statement order mismatch for %s: expected stored index %d, got %d",
				current.Name,
				i,
				stored[i].StmtIndex,
			)
		}

		if current.Statements[i].Hash != stored[i].StmtHash {
			return 0, false, fmt.Errorf(
				"sqlitekit: migration file changed incompatibly: %s statement %d was modified after being applied; previously applied statements are immutable; append new statements or create a new migration file",
				current.Name,
				i,
			)
		}
	}

	if len(current.Statements) == storedFile.StmtCount {
		return storedFile.StmtCount, false, nil
	}

	return storedFile.StmtCount, true, nil
}

func parseStatements(content string) ([]Statement, string, error) {
	clean := stripComments(content)
	parts := strings.Split(clean, ";")

	var statements []Statement
	var normalizedForChecksum []string

	for _, part := range parts {
		norm := normalizeSQL(part)
		if norm == "" {
			continue
		}

		statements = append(statements, Statement{
			Index: len(statements),
			SQL:   norm,
			Hash:  hashText(norm),
		})
		normalizedForChecksum = append(normalizedForChecksum, norm)
	}

	checksum := hashText(strings.Join(normalizedForChecksum, ";\n"))

	return statements, checksum, nil
}

func stripComments(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))

	inBlock := false

	for _, line := range lines {
		var b strings.Builder

		for i := 0; i < len(line); i++ {
			if inBlock {
				if i+1 < len(line) && line[i] == '*' && line[i+1] == '/' {
					inBlock = false
					i++
				}
				continue
			}

			if i+1 < len(line) && line[i] == '/' && line[i+1] == '*' {
				inBlock = true
				i++
				continue
			}

			if i+1 < len(line) && line[i] == '-' && line[i+1] == '-' {
				break
			}

			b.WriteByte(line[i])
		}

		out = append(out, b.String())
	}

	return strings.Join(out, "\n")
}

func normalizeSQL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}

	return strings.Join(fields, " ")
}

func hashText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
