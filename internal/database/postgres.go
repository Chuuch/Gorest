package database

import (
	"context"
	"fmt"
	"time"

	"github.com/chuuch/gorest/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(
	ctx context.Context,
	cfg config.DatabaseConfig,
) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=$s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConnections
	poolConfig.MinConns = cfg.MinConnections
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("ping postgres: %w", &err)
	}

	return pool, nil
}
