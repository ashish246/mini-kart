package repository

import (
	"context"

	"mini-kart/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ProductRepository defines the interface for product data access operations.
type ProductRepository interface {
	// GetAll retrieves all products with pagination support.
	GetAll(ctx context.Context, limit, offset int) ([]model.Product, error)

	// GetByID retrieves a single product by its ID.
	GetByID(ctx context.Context, id string) (*model.Product, error)

	// GetByIDs retrieves multiple products by their IDs.
	GetByIDs(ctx context.Context, ids []string) ([]model.Product, error)

	// ValidateProductsExist checks if all provided product IDs exist in the database.
	// Returns error if any product ID does not exist.
	ValidateProductsExist(ctx context.Context, ids []string) error
}

// OrderRepository defines the interface for order data access operations.
type OrderRepository interface {
	// BeginTx starts a new database transaction.
	BeginTx(ctx context.Context) (pgx.Tx, error)

	// CreateOrder inserts a new order within the provided transaction.
	CreateOrder(ctx context.Context, tx pgx.Tx, order *model.Order) error

	// CreateOrderItems inserts multiple order items within the provided transaction.
	CreateOrderItems(ctx context.Context, tx pgx.Tx, items []model.OrderItem) error

	// GetByID retrieves an order by its ID along with its items.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Order, []model.OrderItem, error)
}
