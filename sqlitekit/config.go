package sqlitekit

import "github.com/Des1red/sqlitekit/internal/models"

type Config = models.Config

func SetConfig(cfg Config) error {
	return models.SetConfig(cfg)
}
