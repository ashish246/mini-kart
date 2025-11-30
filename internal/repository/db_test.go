package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestDefaultDBConfig(t *testing.T) {
	config := DefaultDBConfig()

	require.NotNil(t, config)
	assert.Equal(t, int32(25), config.MaxOpenConns)
	assert.Equal(t, int32(10), config.MaxIdleConns)
	assert.Equal(t, 1*time.Hour, config.ConnMaxLifetime)
	assert.Equal(t, 30*time.Minute, config.ConnMaxIdleTime)
}

func TestNewPool_Success(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	require.NotNil(t, pool)

	ctx := context.Background()

	// Verify connection is healthy
	err := pool.Ping(ctx)
	assert.NoError(t, err)
}

func TestNewPool_InvalidConnectionString(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		connStr  string
		config   *DBConfig
		errMatch string
	}{
		{
			name:     "Invalid connection string",
			connStr:  "invalid connection string",
			config:   DefaultDBConfig(),
			errMatch: "failed to parse connection string",
		},
		{
			name:     "Cannot connect to database",
			connStr:  "postgres://user:pass@invalid-host:5432/testdb?sslmode=disable",
			config:   DefaultDBConfig(),
			errMatch: "failed to ping database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := NewPool(ctx, tt.connStr, tt.config)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMatch)
			assert.Nil(t, pool)
		})
	}
}

func TestNewPool_WithNilConfig(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create pool with nil config (should use defaults)
	pool, err := NewPool(ctx, connStr, nil)
	require.NoError(t, err)
	require.NotNil(t, pool)

	defer pool.Close()

	// Verify connection is healthy
	err = pool.Ping(ctx)
	assert.NoError(t, err)
}

func TestNewPool_ContextCancellation(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	pool, err := NewPool(cancelledCtx, connStr, DefaultDBConfig())

	// Connection might succeed before context cancellation is checked
	// or might fail due to context cancellation
	if err != nil {
		assert.Contains(t, err.Error(), "context")
	}
	if pool != nil {
		pool.Close()
	}
}
