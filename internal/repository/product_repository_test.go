package repository

import (
	"context"
	"testing"
	"time"

	"mini-kart/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB creates a PostgreSQL testcontainer and returns a connection pool.
func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
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

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create connection pool
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	// Create schema
	createSchema(t, pool)

	// Cleanup function
	cleanup := func() {
		pool.Close()
		_ = pgContainer.Terminate(ctx)
	}

	return pool, cleanup
}

// createSchema creates the necessary database schema for testing.
func createSchema(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()

	schema := `
		CREATE TABLE IF NOT EXISTS products (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			price DECIMAL(10,2) NOT NULL CHECK (price >= 0),
			category TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);
		CREATE INDEX IF NOT EXISTS idx_products_created_at ON products(created_at DESC);
	`

	_, err := pool.Exec(ctx, schema)
	require.NoError(t, err)
}

// seedProducts inserts test products into the database.
func seedProducts(t *testing.T, pool *pgxpool.Pool, products []model.Product) {
	ctx := context.Background()

	query := `
		INSERT INTO products (id, name, price, category, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	for _, p := range products {
		_, err := pool.Exec(ctx, query, p.ID, p.Name, p.Price, p.Category, p.CreatedAt)
		require.NoError(t, err)
	}
}

func TestProductRepository_GetAll(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := zerolog.Nop()
	repo := NewProductRepository(pool, logger)

	now := time.Now()
	testProducts := []model.Product{
		{ID: "P001", Name: "Product A", Price: 10.00, Category: "Cat1", CreatedAt: now},
		{ID: "P002", Name: "Product B", Price: 20.00, Category: "Cat2", CreatedAt: now},
		{ID: "P003", Name: "Product C", Price: 30.00, Category: "Cat1", CreatedAt: now},
		{ID: "P004", Name: "Product D", Price: 40.00, Category: "Cat3", CreatedAt: now},
		{ID: "P005", Name: "Product E", Price: 50.00, Category: "Cat2", CreatedAt: now},
	}
	seedProducts(t, pool, testProducts)

	tests := []struct {
		name     string
		limit    int
		offset   int
		expected int
	}{
		{
			name:     "Get all products",
			limit:    10,
			offset:   0,
			expected: 5,
		},
		{
			name:     "Get first page",
			limit:    2,
			offset:   0,
			expected: 2,
		},
		{
			name:     "Get second page",
			limit:    2,
			offset:   2,
			expected: 2,
		},
		{
			name:     "Get last page",
			limit:    2,
			offset:   4,
			expected: 1,
		},
		{
			name:     "Offset beyond results",
			limit:    10,
			offset:   10,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			products, err := repo.GetAll(ctx, tt.limit, tt.offset)

			require.NoError(t, err)
			assert.Len(t, products, tt.expected)

			// Verify products are ordered by name
			for i := 1; i < len(products); i++ {
				assert.LessOrEqual(t, products[i-1].Name, products[i].Name)
			}
		})
	}
}

func TestProductRepository_GetByID(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := zerolog.Nop()
	repo := NewProductRepository(pool, logger)

	now := time.Now()
	testProduct := model.Product{
		ID:        "P001",
		Name:      "Test Product",
		Price:     99.99,
		Category:  "TestCat",
		CreatedAt: now,
	}
	seedProducts(t, pool, []model.Product{testProduct})

	tests := []struct {
		name      string
		id        string
		expectNil bool
	}{
		{
			name:      "Product exists",
			id:        "P001",
			expectNil: false,
		},
		{
			name:      "Product does not exist",
			id:        "P999",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			product, err := repo.GetByID(ctx, tt.id)

			require.NoError(t, err)

			if tt.expectNil {
				assert.Nil(t, product)
			} else {
				require.NotNil(t, product)
				assert.Equal(t, testProduct.ID, product.ID)
				assert.Equal(t, testProduct.Name, product.Name)
				assert.Equal(t, testProduct.Price, product.Price)
				assert.Equal(t, testProduct.Category, product.Category)
			}
		})
	}
}

func TestProductRepository_GetByIDs(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := zerolog.Nop()
	repo := NewProductRepository(pool, logger)

	now := time.Now()
	testProducts := []model.Product{
		{ID: "P001", Name: "Product A", Price: 10.00, Category: "Cat1", CreatedAt: now},
		{ID: "P002", Name: "Product B", Price: 20.00, Category: "Cat2", CreatedAt: now},
		{ID: "P003", Name: "Product C", Price: 30.00, Category: "Cat1", CreatedAt: now},
	}
	seedProducts(t, pool, testProducts)

	tests := []struct {
		name     string
		ids      []string
		expected int
	}{
		{
			name:     "Get multiple products",
			ids:      []string{"P001", "P002", "P003"},
			expected: 3,
		},
		{
			name:     "Get subset of products",
			ids:      []string{"P001", "P003"},
			expected: 2,
		},
		{
			name:     "Some products do not exist",
			ids:      []string{"P001", "P999"},
			expected: 1,
		},
		{
			name:     "No products exist",
			ids:      []string{"P998", "P999"},
			expected: 0,
		},
		{
			name:     "Empty ID list",
			ids:      []string{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			products, err := repo.GetByIDs(ctx, tt.ids)

			require.NoError(t, err)
			assert.Len(t, products, tt.expected)

			// Verify products are ordered by name
			for i := 1; i < len(products); i++ {
				assert.LessOrEqual(t, products[i-1].Name, products[i].Name)
			}
		})
	}
}

func TestProductRepository_ValidateProductsExist(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := zerolog.Nop()
	repo := NewProductRepository(pool, logger)

	now := time.Now()
	testProducts := []model.Product{
		{ID: "P001", Name: "Product A", Price: 10.00, Category: "Cat1", CreatedAt: now},
		{ID: "P002", Name: "Product B", Price: 20.00, Category: "Cat2", CreatedAt: now},
		{ID: "P003", Name: "Product C", Price: 30.00, Category: "Cat1", CreatedAt: now},
	}
	seedProducts(t, pool, testProducts)

	tests := []struct {
		name      string
		ids       []string
		expectErr bool
	}{
		{
			name:      "All products exist",
			ids:       []string{"P001", "P002", "P003"},
			expectErr: false,
		},
		{
			name:      "Subset of products exist",
			ids:       []string{"P001", "P002"},
			expectErr: false,
		},
		{
			name:      "Some products do not exist",
			ids:       []string{"P001", "P999"},
			expectErr: true,
		},
		{
			name:      "No products exist",
			ids:       []string{"P998", "P999"},
			expectErr: true,
		},
		{
			name:      "Empty ID list",
			ids:       []string{},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			err := repo.ValidateProductsExist(ctx, tt.ids)

			if tt.expectErr {
				require.Error(t, err)
				assert.Equal(t, model.ErrProductNotFound, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestProductRepository_ErrorPaths(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	logger := zerolog.Nop()
	repo := NewProductRepository(pool, logger)

	now := time.Now()
	testProducts := []model.Product{
		{ID: "P001", Name: "Product A", Price: 10.00, Category: "Cat1", CreatedAt: now},
	}
	seedProducts(t, pool, testProducts)

	// Close the pool to simulate database errors
	pool.Close()

	t.Run("GetAll with closed pool", func(t *testing.T) {
		ctx := context.Background()
		products, err := repo.GetAll(ctx, 10, 0)

		require.Error(t, err)
		assert.Nil(t, products)
	})

	t.Run("GetByID with closed pool", func(t *testing.T) {
		ctx := context.Background()
		product, err := repo.GetByID(ctx, "P001")

		require.Error(t, err)
		assert.Nil(t, product)
	})

	t.Run("GetByIDs with closed pool", func(t *testing.T) {
		ctx := context.Background()
		products, err := repo.GetByIDs(ctx, []string{"P001"})

		require.Error(t, err)
		assert.Nil(t, products)
	})

	t.Run("ValidateProductsExist with closed pool", func(t *testing.T) {
		ctx := context.Background()
		err := repo.ValidateProductsExist(ctx, []string{"P001"})

		require.Error(t, err)
	})
}
