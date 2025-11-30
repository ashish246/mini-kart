package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"mini-kart/internal/config"
	"mini-kart/internal/database"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDB represents a test database instance.
type TestDB struct {
	Container *postgres.PostgresContainer
	Pool      *pgxpool.Pool
	ConnStr   string
}

// SetupTestDB creates a PostgreSQL test container and connection pool.
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	ctx := context.Background()

	// Create PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Create connection pool
	dbConfig := config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "testuser",
		Password:        "testpass",
		Database:        "testdb",
		MaxConnections:  10,
		MinConnections:  2,
		MaxConnLifetime: 300,
	}

	logger := zerolog.Nop()
	pool, err := database.NewPool(ctx, dbConfig, logger)
	if err != nil {
		// Try with connection string directly
		poolConfig, parseErr := pgxpool.ParseConfig(connStr)
		if parseErr != nil {
			t.Fatalf("failed to parse connection string: %v", parseErr)
		}
		pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			t.Fatalf("failed to create connection pool: %v", err)
		}
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	// Create schema
	createSchema(t, pool)

	t.Cleanup(func() {
		pool.Close()
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	return &TestDB{
		Container: postgresContainer,
		Pool:      pool,
		ConnStr:   connStr,
	}
}

// createSchema creates the database schema for testing.
func createSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	schema := `
		CREATE TABLE IF NOT EXISTS products (
			id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			price DECIMAL(10, 2) NOT NULL,
			category VARCHAR(100) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS orders (
			id UUID PRIMARY KEY,
			coupon_code VARCHAR(50),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS order_items (
			id UUID PRIMARY KEY,
			order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
			product_id VARCHAR(50) NOT NULL REFERENCES products(id),
			quantity INTEGER NOT NULL CHECK (quantity > 0),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
		CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);
	`

	_, err := pool.Exec(ctx, schema)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
}

// SeedProducts inserts test product data into the database.
func SeedProducts(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	products := []struct {
		id       string
		name     string
		price    float64
		category string
	}{
		{"P001", "Test Product 1", 10.00, "Category A"},
		{"P002", "Test Product 2", 20.00, "Category B"},
		{"P003", "Test Product 3", 30.00, "Category A"},
		{"P004", "Test Product 4", 40.00, "Category C"},
		{"P005", "Test Product 5", 50.00, "Category B"},
	}

	for _, p := range products {
		_, err := pool.Exec(ctx,
			"INSERT INTO products (id, name, price, category) VALUES ($1, $2, $3, $4)",
			p.id, p.name, p.price, p.category,
		)
		if err != nil {
			t.Fatalf("failed to seed product %s: %v", p.id, err)
		}
	}
}

// CleanupDB cleans all data from test tables.
func CleanupDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	tables := []string{"order_items", "orders", "products"}
	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Logf("failed to clean table %s: %v", table, err)
		}
	}
}
