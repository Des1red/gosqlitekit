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

func loadFileState(
	files fs.FS,
	filename string,
) (FileState, error) {
	content,
		err :=
		fs.ReadFile(
			files,
			filename,
		)

	if err != nil {
		return FileState{},
			fmt.Errorf(
				"sqlitekit: read migration file %s: %w",
				filename,
				err,
			)
	}

	statements,
		checksum,
		err :=
		parseStatements(
			string(
				content,
			),
		)

	if err != nil {
		return FileState{},
			fmt.Errorf(
				"sqlitekit: parse migration file %s: %w",
				filename,
				err,
			)
	}

	return FileState{
			Name: filename,

			Checksum: checksum,

			Statements: statements,
		},
		nil
}

func execStatements(
	tx *sql.Tx,
	filename string,
	stmts []Statement,
) error {
	for _, stmt := range stmts {

		if _,
			err :=
			tx.Exec(
				stmt.SQL,
			); err != nil {

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

func validateAppendOnly(
	storedFile StoredFile,
	stored []StoredStatement,
	current FileState,
) (
	appendFrom int,
	changed bool,
	err error,
) {
	if len(
		stored,
	) !=
		storedFile.StmtCount {

		return 0,
			false,
			fmt.Errorf(
				"sqlitekit: migration metadata mismatch for %s: file stmt_count=%d stored statements=%d",
				current.Name,
				storedFile.StmtCount,
				len(
					stored,
				),
			)
	}

	if len(
		current.Statements,
	) <
		storedFile.StmtCount {

		return 0,
			false,
			fmt.Errorf(
				"sqlitekit: migration file changed incompatibly: %s now has fewer statements (%d) than previously applied (%d)",
				current.Name,
				len(
					current.Statements,
				),
				storedFile.StmtCount,
			)
	}

	for i := 0; i <
		storedFile.StmtCount; i++ {

		if i >=
			len(
				stored,
			) {

			return 0,
				false,
				fmt.Errorf(
					"sqlitekit: migration metadata mismatch for %s at statement %d",
					current.Name,
					i,
				)
		}

		if stored[i].StmtIndex !=
			i {

			return 0,
				false,
				fmt.Errorf(
					"sqlitekit: migration statement order mismatch for %s: expected stored index %d, got %d",
					current.Name,
					i,
					stored[i].StmtIndex,
				)
		}

		if current.Statements[i].Hash !=
			stored[i].StmtHash {

			return 0,
				false,
				fmt.Errorf(
					"sqlitekit: migration file changed incompatibly: %s statement %d was modified after being applied; previously applied statements are immutable; append new statements or create a new migration file",
					current.Name,
					i,
				)
		}
	}

	if len(
		current.Statements,
	) ==
		storedFile.StmtCount {

		return storedFile.StmtCount,
			false,
			nil
	}

	return storedFile.StmtCount,
		true,
		nil
}

func parseStatements(
	content string,
) (
	[]Statement,
	string,
	error,
) {
	parts,
		err :=
		splitSQLStatements(
			content,
		)

	if err != nil {
		return nil,
			"",
			err
	}

	statements :=
		make(
			[]Statement,
			0,
			len(
				parts,
			),
		)

	normalizedForChecksum :=
		make(
			[]string,
			0,
			len(
				parts,
			),
		)

	for _, part := range parts {

		sqlText :=
			strings.TrimSpace(
				part,
			)

		if sqlText == "" {
			continue
		}

		norm :=
			normalizeSQL(
				sqlText,
			)

		if norm == "" {
			continue
		}

		statements =
			append(
				statements,
				Statement{
					Index: len(
						statements,
					),

					/*
						Execute the actual SQL rather
						than the normalized form.

						Normalization is used only
						for migration identity.
					*/
					SQL: sqlText,

					Hash: hashText(
						norm,
					),
				},
			)

		normalizedForChecksum =
			append(
				normalizedForChecksum,
				norm,
			)
	}

	checksum :=
		hashText(
			strings.Join(
				normalizedForChecksum,
				";\n",
			),
		)

	return statements,
		checksum,
		nil
}

/*
splitSQLStatements separates complete SQLite
statements without breaking:

  - trigger BEGIN ... END blocks
  - CASE ... END expressions
  - quoted strings containing semicolons
  - quoted identifiers containing semicolons
  - line comments
  - block comments

A normal statement ends at its next top-level
semicolon.

A CREATE TRIGGER statement ends only after the
trigger body's matching END;
*/
func splitSQLStatements(
	content string,
) (
	[]string,
	error,
) {
	result :=
		make(
			[]string,
			0,
		)

	var statement strings.Builder

	var token strings.Builder

	prefix :=
		make(
			[]string,
			0,
			3,
		)

	inSingleQuote :=
		false

	inDoubleQuote :=
		false

	inBacktick :=
		false

	inBracket :=
		false

	inLineComment :=
		false

	inBlockComment :=
		false

	isTrigger :=
		false

	triggerBodyStarted :=
		false

	beginDepth :=
		0

	caseDepth :=
		0

	resetStatementState :=
		func() {
			statement.Reset()

			token.Reset()

			prefix =
				prefix[:0]

			isTrigger =
				false

			triggerBodyStarted =
				false

			beginDepth =
				0

			caseDepth =
				0
		}

	updateTriggerPrefix :=
		func(
			word string,
		) {
			if isTrigger ||
				len(
					prefix,
				) >=
					3 {

				return
			}

			prefix =
				append(
					prefix,
					word,
				)

			if len(
				prefix,
			) == 2 &&
				prefix[0] ==
					"CREATE" &&
				prefix[1] ==
					"TRIGGER" {

				isTrigger =
					true

				return
			}

			if len(
				prefix,
			) == 3 &&
				prefix[0] ==
					"CREATE" &&
				(prefix[1] ==
					"TEMP" ||
					prefix[1] ==
						"TEMPORARY") &&
				prefix[2] ==
					"TRIGGER" {

				isTrigger =
					true
			}
		}

	flushToken :=
		func() {
			if token.Len() ==
				0 {

				return
			}

			word :=
				strings.ToUpper(
					token.String(),
				)

			token.Reset()

			updateTriggerPrefix(
				word,
			)

			if !isTrigger {
				return
			}

			switch word {

			case "BEGIN":

				triggerBodyStarted =
					true

				beginDepth++

			case "CASE":

				if triggerBodyStarted {
					caseDepth++
				}

			case "END":

				if !triggerBodyStarted {
					return
				}

				if caseDepth > 0 {
					caseDepth--

					return
				}

				if beginDepth > 0 {
					beginDepth--
				}
			}
		}

	appendStatement :=
		func() {
			value :=
				strings.TrimSpace(
					statement.String(),
				)

			/*
				Keep internal semicolons but remove
				the final statement delimiter.
			*/
			value =
				strings.TrimSpace(
					strings.TrimSuffix(
						value,
						";",
					),
				)

			if value != "" {
				result =
					append(
						result,
						value,
					)
			}

			resetStatementState()
		}

	for index := 0; index <
		len(
			content,
		); index++ {

		current :=
			content[index]

		/*
			Line comment.
		*/
		if inLineComment {
			if current ==
				'\n' {

				inLineComment =
					false

				statement.WriteByte(
					'\n',
				)
			}

			continue
		}

		/*
			Block comment.
		*/
		if inBlockComment {
			if current ==
				'*' &&
				index+1 <
					len(
						content,
					) &&
				content[index+1] ==
					'/' {

				inBlockComment =
					false

				index++

				statement.WriteByte(
					' ',
				)
			}

			continue
		}

		/*
			Single quoted string.

			SQLite escapes a single quote by
			doubling it:

				'it''s valid'
		*/
		if inSingleQuote {
			statement.WriteByte(
				current,
			)

			if current ==
				'\'' {

				if index+1 <
					len(
						content,
					) &&
					content[index+1] ==
						'\'' {

					statement.WriteByte(
						content[index+1],
					)

					index++

					continue
				}

				inSingleQuote =
					false
			}

			continue
		}

		/*
			Double quoted identifier/string.
		*/
		if inDoubleQuote {
			statement.WriteByte(
				current,
			)

			if current ==
				'"' {

				if index+1 <
					len(
						content,
					) &&
					content[index+1] ==
						'"' {

					statement.WriteByte(
						content[index+1],
					)

					index++

					continue
				}

				inDoubleQuote =
					false
			}

			continue
		}

		/*
			Backtick quoted identifier.
		*/
		if inBacktick {
			statement.WriteByte(
				current,
			)

			if current ==
				'`' {

				if index+1 <
					len(
						content,
					) &&
					content[index+1] ==
						'`' {

					statement.WriteByte(
						content[index+1],
					)

					index++

					continue
				}

				inBacktick =
					false
			}

			continue
		}

		/*
			SQLite bracket quoted identifier.
		*/
		if inBracket {
			statement.WriteByte(
				current,
			)

			if current ==
				']' {

				inBracket =
					false
			}

			continue
		}

		/*
			Comment starts.
		*/
		if current ==
			'-' &&
			index+1 <
				len(
					content,
				) &&
			content[index+1] ==
				'-' {

			flushToken()

			inLineComment =
				true

			index++

			continue
		}

		if current ==
			'/' &&
			index+1 <
				len(
					content,
				) &&
			content[index+1] ==
				'*' {

			flushToken()

			inBlockComment =
				true

			index++

			continue
		}

		/*
			Quote starts.
		*/
		switch current {

		case '\'':

			flushToken()

			inSingleQuote =
				true

			statement.WriteByte(
				current,
			)

			continue

		case '"':

			flushToken()

			inDoubleQuote =
				true

			statement.WriteByte(
				current,
			)

			continue

		case '`':

			flushToken()

			inBacktick =
				true

			statement.WriteByte(
				current,
			)

			continue

		case '[':

			flushToken()

			inBracket =
				true

			statement.WriteByte(
				current,
			)

			continue
		}

		/*
			Track SQL keywords so trigger BEGIN/END
			and CASE/END nesting can be distinguished.
		*/
		if isSQLWordByte(
			current,
		) {
			token.WriteByte(
				current,
			)
		} else {
			flushToken()
		}

		statement.WriteByte(
			current,
		)

		if current !=
			';' {

			continue
		}

		/*
			Normal SQL statements terminate at
			every top-level semicolon.

			Triggers terminate only after their
			BEGIN block has closed.
		*/
		if !isTrigger {
			appendStatement()

			continue
		}

		if triggerBodyStarted &&
			beginDepth ==
				0 {

			appendStatement()
		}
	}

	flushToken()

	if inBlockComment {
		return nil,
			fmt.Errorf(
				"unterminated block comment",
			)
	}

	if inSingleQuote {
		return nil,
			fmt.Errorf(
				"unterminated single-quoted string",
			)
	}

	if inDoubleQuote {
		return nil,
			fmt.Errorf(
				"unterminated double-quoted value",
			)
	}

	if inBacktick {
		return nil,
			fmt.Errorf(
				"unterminated backtick identifier",
			)
	}

	if inBracket {
		return nil,
			fmt.Errorf(
				"unterminated bracket identifier",
			)
	}

	if isTrigger &&
		triggerBodyStarted &&
		beginDepth >
			0 {

		return nil,
			fmt.Errorf(
				"incomplete CREATE TRIGGER statement",
			)
	}

	/*
		Support a final SQL statement without a
		trailing semicolon, matching the previous
		parser's behavior.
	*/
	if strings.TrimSpace(
		statement.String(),
	) != "" {

		appendStatement()
	}

	return result,
		nil
}

func isSQLWordByte(
	value byte,
) bool {
	return (value >= 'a' &&
		value <= 'z') ||
		(value >= 'A' &&
			value <= 'Z') ||
		(value >= '0' &&
			value <= '9') ||
		value ==
			'_'
}

func normalizeSQL(
	s string,
) string {
	s =
		strings.TrimSpace(
			s,
		)

	if s == "" {
		return ""
	}

	fields :=
		strings.Fields(
			s,
		)

	if len(
		fields,
	) == 0 {

		return ""
	}

	return strings.Join(
		fields,
		" ",
	)
}

func hashText(
	s string,
) string {
	sum :=
		sha256.Sum256(
			[]byte(
				s,
			),
		)

	return hex.EncodeToString(
		sum[:],
	)
}
