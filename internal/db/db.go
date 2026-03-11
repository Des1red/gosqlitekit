package db

import (
	"database/sql"
	"os"

	"github.com/Des1red/sqlitekit/internal/logs"
	"github.com/Des1red/sqlitekit/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Open(path string, cfg models.Config) error {

	logs.PrintDBInit(path, cfg)
	var err error

	DB, err = sql.Open("sqlite3", path)
	if err != nil {
		logs.PrintDBFail(err)
		return err
	}

	if err = DB.Ping(); err != nil {
		logs.PrintDBFail(err)
		return err
	}

	if cfg.WAL {
		if _, err := DB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
			logs.PrintDBFail(err)
			return err
		}
	}

	if cfg.ForeignKeys {
		if _, err := DB.Exec("PRAGMA foreign_keys = ON;"); err != nil {
			logs.PrintDBFail(err)
			return err
		}
	}

	if _, err := DB.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
		logs.PrintDBFail(err)
		return err
	}

	DB.SetMaxOpenConns(cfg.MaxOpenConns)
	DB.SetMaxIdleConns(cfg.MaxIdleConns)

	if _, err := os.Stat(path); err == nil {
		_ = os.Chmod(path, 0600)
	}

	logs.PrintDBReady()

	return nil
}
