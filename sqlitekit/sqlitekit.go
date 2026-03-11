package sqlitekit

import (
	"io/fs"
	"os"

	"github.com/Des1red/sqlitekit/internal/db"
	"github.com/Des1red/sqlitekit/internal/models"
	"github.com/Des1red/sqlitekit/internal/schema"
)

func Initialize(dbPath, schemaDir string) error {
	cfg := models.GetConfig()
	models.LockConfig()
	if err := db.Open(dbPath, cfg); err != nil {
		return err
	}

	return schema.Apply(os.DirFS(schemaDir))
}

func InitializeEmbedded(dbPath string, migrations fs.FS) error {
	cfg := models.GetConfig()
	models.LockConfig()
	if err := db.Open(dbPath, cfg); err != nil {
		return err
	}

	if err := schema.Apply(migrations); err != nil {
		return err
	}

	return nil
}
