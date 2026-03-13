package sqlitekit

import "github.com/Des1red/gosqlitekit/internal/models"

type Config = models.Config

func SetConfig(cfg Config) error {
	return models.SetConfig(cfg)
}
