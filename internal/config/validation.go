package config

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

func validate(cfg *Config) error {
	v := validator.New()

	if err := v.Struct(cfg); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	if cfg.Database.MinConnections > cfg.Database.MaxConnections {
		return fmt.Errorf("database.min_connections cannot exceed database.max_connections")
	}

	if cfg.App.Environment == "production" {
		if cfg.Database.SSLMode == "disable" {
			return fmt.Errorf("database.ssl_mode cannot be disabled in production")
		}
	}

	return nil
}
