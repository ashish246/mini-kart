package integration

import (
	"context"
	"testing"

	"mini-kart/internal/model"
	"mini-kart/internal/repository"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := SetupTestDB(t)
	logger := zerolog.Nop()
	repo := repository.NewProductRepository(testDB.Pool, logger)

	ctx := context.Background()

	t.Run("GetAll returns seeded products", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		products, err := repo.GetAll(ctx, 10, 0)
		require.NoError(t, err)
		assert.Len(t, products, 5)
		assert.Equal(t, "P001", products[0].ID)
	})

	t.Run("GetAll with pagination", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		products, err := repo.GetAll(ctx, 2, 0)
		require.NoError(t, err)
		assert.Len(t, products, 2)

		products, err = repo.GetAll(ctx, 2, 2)
		require.NoError(t, err)
		assert.Len(t, products, 2)
	})

	t.Run("GetByID returns correct product", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		product, err := repo.GetByID(ctx, "P001")
		require.NoError(t, err)
		require.NotNil(t, product)
		assert.Equal(t, "P001", product.ID)
		assert.Equal(t, "Test Product 1", product.Name)
		assert.Equal(t, 10.00, product.Price)
	})

	t.Run("GetByID returns nil for non-existent product", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)

		product, err := repo.GetByID(ctx, "P999")
		require.NoError(t, err)
		assert.Nil(t, product)
	})

	t.Run("GetByIDs returns multiple products", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		products, err := repo.GetByIDs(ctx, []string{"P001", "P003", "P005"})
		require.NoError(t, err)
		assert.Len(t, products, 3)
	})

	t.Run("ValidateProductsExist succeeds for valid products", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		err := repo.ValidateProductsExist(ctx, []string{"P001", "P002"})
		require.NoError(t, err)
	})

	t.Run("ValidateProductsExist fails for invalid products", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		err := repo.ValidateProductsExist(ctx, []string{"P001", "P999"})
		require.Error(t, err)
		assert.Equal(t, model.ErrProductNotFound, err)
	})
}

func TestOrderRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := SetupTestDB(t)
	logger := zerolog.Nop()
	repo := repository.NewOrderRepository(testDB.Pool, logger)

	ctx := context.Background()

	t.Run("CreateOrder and CreateOrderItems", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		// Begin transaction
		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)

		// Create order
		orderID := uuid.New()
		couponCode := "TESTCODE"
		order := &model.Order{
			ID:         orderID,
			CouponCode: &couponCode,
		}

		err = repo.CreateOrder(ctx, tx, order)
		require.NoError(t, err)

		// Create order items
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
				Quantity:  1,
			},
		}

		err = repo.CreateOrderItems(ctx, tx, items)
		require.NoError(t, err)

		// Commit transaction
		err = tx.Commit(ctx)
		require.NoError(t, err)

		// Verify order was created
		retrievedOrder, retrievedItems, err := repo.GetByID(ctx, orderID)
		require.NoError(t, err)
		require.NotNil(t, retrievedOrder)
		assert.Equal(t, orderID, retrievedOrder.ID)
		assert.Equal(t, &couponCode, retrievedOrder.CouponCode)
		assert.Len(t, retrievedItems, 2)
	})

	t.Run("GetByID returns nil for non-existent order", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)

		order, items, err := repo.GetByID(ctx, uuid.New())
		require.NoError(t, err)
		assert.Nil(t, order)
		assert.Nil(t, items)
	})

	t.Run("Transaction rollback", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		// Begin transaction
		tx, err := repo.BeginTx(ctx)
		require.NoError(t, err)

		// Create order
		orderID := uuid.New()
		order := &model.Order{
			ID:         orderID,
			CouponCode: nil,
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
	})
}
