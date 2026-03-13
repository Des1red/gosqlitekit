package sqlitekit

import (
	"database/sql"

	"github.com/Des1red/gosqlitekit/internal/db"
)

func DB() *sql.DB {
	return db.DB
}
