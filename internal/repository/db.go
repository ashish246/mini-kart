package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBConfig holds database connection pool configuration.
type DBConfig struct {
	MaxOpenConns    int32
	MaxIdleConns    int32
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultDBConfig returns sensible default database configuration.
func DefaultDBConfig() *DBConfig {
	return &DBConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 1 * time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
	}
}

// NewPool creates a new PostgreSQL connection pool with the provided configuration.
// It verifies connectivity by pinging the database.
func NewPool(ctx context.Context, connString string, config *DBConfig) (*pgxpool.Pool, error) {
	if config == nil {
		config = DefaultDBConfig()
	}

	// Parse connection string into config
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = config.MaxOpenConns
	poolConfig.MinConns = config.MaxIdleConns
	poolConfig.MaxConnLifetime = config.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = config.ConnMaxIdleTime

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connectivity
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
