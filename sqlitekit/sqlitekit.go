package sqlitekit

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/Des1red/sqlitekit/internal/db"
	"github.com/Des1red/sqlitekit/internal/models"
	"github.com/Des1red/sqlitekit/internal/schema"
)

func Initialize(dbPath, schemaDir string) error {
	cfg := models.GetConfig()
	models.LockConfig()
	// ensure schema directory exists
	if _, err := os.Stat(schemaDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("sqlitekit: schema directory does not exist: %s", schemaDir)
		}
		return err
	}
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

	return schema.Apply(migrations)
}
