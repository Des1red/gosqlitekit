// initilize db
package database

import (
	"database/sql"
	"log"

	"github.com/Des1red/gosqlitekit/sqlitekit"
)

var DB *sql.DB

func InitDb() {
	err := sqlitekit.SetConfig(sqlitekit.Config{
		WAL:          false,
		ForeignKeys:  true,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
	})

	if err != nil {
		panic(err)
	}
	err = sqlitekit.Initialize(
		"socialnet.db",
		"example/database/schema",
	)

	if err != nil {
		log.Fatal(err)
	}
	DB = sqlitekit.DB()
}
