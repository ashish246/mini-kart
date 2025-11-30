package repository

import (
	"context"
	"testing"
	"time"

	"mini-kart/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createOrderSchema creates the necessary order-related database schema for testing.
func createOrderSchema(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()

	schema := `
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

		CREATE TABLE IF NOT EXISTS products (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			price DECIMAL(10,2) NOT NULL CHECK (price >= 0),
			category TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS orders (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			coupon_code TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS order_items (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
			product_id TEXT NOT NULL REFERENCES products(id),
			quantity INTEGER NOT NULL CHECK (quantity > 0)
		);
	`

	_, err := pool.Exec(ctx, schema)
	require.NoError(t, err)
}

// setupOrderTestDB creates a test database with order schema.
func setupOrderTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	pool, cleanup := setupTestDB(t)
	createOrderSchema(t, pool)
	return pool, cleanup
}

func TestOrderRepository_BeginTx(t *testing.T) {
	pool, cleanup := setupOrderTestDB(t)
	defer cleanup()

	logger := zerolog.Nop()
	repo := NewOrderRepository(pool, logger)

	ctx := context.Background()

	tx, err := repo.BeginTx(ctx)

	require.NoError(t, err)
	require.NotNil(t, tx)

	// Rollback to cleanup
	err = tx.Rollback(ctx)
	assert.NoError(t, err)
}

func TestOrderRepository_CreateOrder(t *testing.T) {
	pool, cleanup := setupOrderTestDB(t)
	defer cleanup()

	logger := zerolog.Nop()
	repo := NewOrderRepository(pool, logger)

	ctx := context.Background()
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	now := time.Now()
	orderID := uuid.New()
	couponCode := "TESTCODE123"

	tests := []struct {
		name  string
		order *model.Order
	}{
		{
			name: "Create order with coupon code",
			order: &model.Order{
				ID:         orderID,
				CouponCode: &couponCode,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
		{
			name: "Create order without coupon code",
			order: &model.Order{
				ID:         uuid.New(),
				CouponCode: nil,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.CreateOrder(ctx, tx, tt.order)

			require.NoError(t, err)

			// Verify order was created
			var count int
			err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM orders WHERE id = $1", tt.order.ID).Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 1, count)
		})
	}
}

func TestOrderRepository_CreateOrderItems(t *testing.T) {
	pool, cleanup := setupOrderTestDB(t)
	defer cleanup()

	logger := zerolog.Nop()
	repo := NewOrderRepository(pool, logger)

	ctx := context.Background()

	// Seed products
	now := time.Now()
	testProducts := []model.Product{
		{ID: "P001", Name: "Product A", Price: 10.00, Category: "Cat1", CreatedAt: now},
		{ID: "P002", Name: "Product B", Price: 20.00, Category: "Cat2", CreatedAt: now},
	}
	seedProducts(t, pool, testProducts)

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Create order
	orderID := uuid.New()
	order := &model.Order{
		ID:         orderID,
		CouponCode: nil,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	err = repo.CreateOrder(ctx, tx, order)
	require.NoError(t, err)

	tests := []struct {
		name  string
		items []model.OrderItem
	}{
		{
			name: "Create multiple order items",
			items: []model.OrderItem{
				{
					ID:        uuid.New(),
					OrderID:   orderID,
					ProductID: "P001",
					Quantity:  2,
				},
				{
					ID:        uuid.New(),
					OrderID:   orderID,
					ProductID: "P002",
					Quantity:  3,
				},
			},
		},
		{
			name: "Create single order item",
			items: []model.OrderItem{
				{
					ID:        uuid.New(),
					OrderID:   orderID,
					ProductID: "P001",
					Quantity:  1,
				},
			},
		},
		{
			name:  "Create empty order items",
			items: []model.OrderItem{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.CreateOrderItems(ctx, tx, tt.items)

			require.NoError(t, err)

			if len(tt.items) > 0 {
				// Verify items were created
				var count int
				err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM order_items WHERE id = $1", tt.items[0].ID).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count)
			}
		})
	}
}

func TestOrderRepository_GetByID(t *testing.T) {
	pool, cleanup := setupOrderTestDB(t)
	defer cleanup()

	logger := zerolog.Nop()
	repo := NewOrderRepository(pool, logger)

	ctx := context.Background()

	// Seed products
	now := time.Now()
	testProducts := []model.Product{
		{ID: "P001", Name: "Product A", Price: 10.00, Category: "Cat1", CreatedAt: now},
		{ID: "P002", Name: "Product B", Price: 20.00, Category: "Cat2", CreatedAt: now},
	}
	seedProducts(t, pool, testProducts)

	// Create order with items
	orderID := uuid.New()
	couponCode := "TESTCODE123"
	order := &model.Order{
		ID:         orderID,
		CouponCode: &couponCode,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)

	err = repo.CreateOrder(ctx, tx, order)
	require.NoError(t, err)

	items := []model.OrderItem{
		{
			ID:        uuid.New(),
			OrderID:   orderID,
			ProductID: "P001",
			Quantity:  2,
		},
		{
			ID:        uuid.New(),
			OrderID:   orderID,
			ProductID: "P002",
			Quantity:  3,
		},
	}

	err = repo.CreateOrderItems(ctx, tx, items)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		orderID       uuid.UUID
		expectNil     bool
		expectedItems int
	}{
		{
			name:          "Order exists with items",
			orderID:       orderID,
			expectNil:     false,
			expectedItems: 2,
		},
		{
			name:          "Order does not exist",
			orderID:       uuid.New(),
			expectNil:     true,
			expectedItems: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrievedOrder, retrievedItems, err := repo.GetByID(ctx, tt.orderID)

			require.NoError(t, err)

			if tt.expectNil {
				assert.Nil(t, retrievedOrder)
				assert.Nil(t, retrievedItems)
			} else {
				require.NotNil(t, retrievedOrder)
				assert.Equal(t, order.ID, retrievedOrder.ID)
				assert.Equal(t, order.CouponCode, retrievedOrder.CouponCode)

				require.Len(t, retrievedItems, tt.expectedItems)

				// Verify items (create a map for order-independent comparison)
				itemsByProductID := make(map[string]model.OrderItem)
				for _, item := range retrievedItems {
					itemsByProductID[item.ProductID] = item
				}

				for _, expectedItem := range items {
					actualItem, found := itemsByProductID[expectedItem.ProductID]
					require.True(t, found, "Product %s not found in retrieved items", expectedItem.ProductID)
					assert.Equal(t, expectedItem.OrderID, actualItem.OrderID)
					assert.Equal(t, expectedItem.Quantity, actualItem.Quantity)
				}
			}
		})
	}
}

func TestOrderRepository_TransactionRollback(t *testing.T) {
	pool, cleanup := setupOrderTestDB(t)
	defer cleanup()

	logger := zerolog.Nop()
	repo := NewOrderRepository(pool, logger)

	ctx := context.Background()

	// Start transaction
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)

	// Create order
	now := time.Now()
	orderID := uuid.New()
	order := &model.Order{
		ID:         orderID,
		CouponCode: nil,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err = repo.CreateOrder(ctx, tx, order)
	require.NoError(t, err)

	// Rollback transaction
	err = tx.Rollback(ctx)
	require.NoError(t, err)

	// Verify order was not persisted
	retrievedOrder, _, err := repo.GetByID(ctx, orderID)
	require.NoError(t, err)
	assert.Nil(t, retrievedOrder)
}

func TestOrderRepository_TransactionCommit(t *testing.T) {
	pool, cleanup := setupOrderTestDB(t)
	defer cleanup()

	logger := zerolog.Nop()
	repo := NewOrderRepository(pool, logger)

	ctx := context.Background()

	// Start transaction
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)

	// Create order
	now := time.Now()
	orderID := uuid.New()
	order := &model.Order{
		ID:         orderID,
		CouponCode: nil,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err = repo.CreateOrder(ctx, tx, order)
	require.NoError(t, err)

	// Commit transaction
	err = tx.Commit(ctx)
	require.NoError(t, err)

	// Verify order was persisted
	retrievedOrder, _, err := repo.GetByID(ctx, orderID)
	require.NoError(t, err)
	require.NotNil(t, retrievedOrder)
	assert.Equal(t, orderID, retrievedOrder.ID)
}

func TestOrderRepository_ErrorPaths(t *testing.T) {
	pool, cleanup := setupOrderTestDB(t)
	defer cleanup()

	logger := zerolog.Nop()
	repo := NewOrderRepository(pool, logger)

	ctx := context.Background()

	// Seed products for testing
	now := time.Now()
	testProducts := []model.Product{
		{ID: "P001", Name: "Product A", Price: 10.00, Category: "Cat1", CreatedAt: now},
	}
	seedProducts(t, pool, testProducts)

	// Create a test order
	tx, err := repo.BeginTx(ctx)
	require.NoError(t, err)

	orderID := uuid.New()
	order := &model.Order{
		ID:         orderID,
		CouponCode: nil,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	err = repo.CreateOrder(ctx, tx, order)
	require.NoError(t, err)

	err = tx.Commit(ctx)
	require.NoError(t, err)

	// Close the pool to simulate database errors
	pool.Close()

	t.Run("BeginTx with closed pool", func(t *testing.T) {
		tx, err := repo.BeginTx(ctx)

		require.Error(t, err)
		assert.Nil(t, tx)
	})

	t.Run("GetByID with closed pool", func(t *testing.T) {
		retrievedOrder, items, err := repo.GetByID(ctx, orderID)

		require.Error(t, err)
		assert.Nil(t, retrievedOrder)
		assert.Nil(t, items)
	})
}
