package database

import (
	"context"
	"fmt"
	"time"

	"mini-kart/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// NewPool creates a new PostgreSQL connection pool.
func NewPool(ctx context.Context, cfg config.DatabaseConfig, logger zerolog.Logger) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure pool settings
	poolConfig.MaxConns = int32(cfg.MaxConnections)
	poolConfig.MinConns = int32(cfg.MinConnections)
	poolConfig.MaxConnLifetime = time.Duration(cfg.MaxConnLifetime) * time.Second
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	logger.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Database).
		Int("max_connections", cfg.MaxConnections).
		Int("min_connections", cfg.MinConnections).
		Msg("creating database connection pool")

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info().Msg("database connection pool created successfully")

	return pool, nil
}
