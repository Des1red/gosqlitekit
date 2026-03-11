package models

import (
	"fmt"
	"sync"
)

type Config struct {
	WAL          bool
	ForeignKeys  bool
	MaxOpenConns int
	MaxIdleConns int
}

var (
	mu sync.RWMutex

	ConfigDefaults = Config{
		WAL:          true,
		ForeignKeys:  true,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}

	configCurrent = ConfigDefaults
	initialized   = false
)

func SetConfig(cfg Config) error {

	mu.Lock()
	defer mu.Unlock()

	if initialized {
		return fmt.Errorf("sqlitekit: config cannot be changed after initialization")
	}

	if err := validateConfig(cfg); err != nil {
		return err
	}

	configCurrent = cfg
	return nil
}

func GetConfig() Config {
	mu.RLock()
	defer mu.RUnlock()
	return configCurrent
}

func LockConfig() {
	mu.Lock()
	initialized = true
	mu.Unlock()
}

func validateConfig(cfg Config) error {

	if cfg.MaxOpenConns <= 0 {
		return fmt.Errorf("invalid config: MaxOpenConns must be > 0")
	}

	if cfg.MaxIdleConns <= 0 {
		return fmt.Errorf("invalid config: MaxIdleConns must be > 0")
	}

	if cfg.MaxIdleConns > cfg.MaxOpenConns {
		return fmt.Errorf("invalid config: MaxIdleConns cannot exceed MaxOpenConns")
	}

	return nil
}
