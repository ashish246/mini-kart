package service

import (
	"context"

	"mini-kart/internal/model"

	"github.com/google/uuid"
)

// ProductService defines operations for product management.
type ProductService interface {
	// GetAll retrieves all products with pagination.
	GetAll(ctx context.Context, limit, offset int) ([]model.Product, error)

	// GetByID retrieves a single product by ID.
	GetByID(ctx context.Context, id string) (*model.Product, error)

	// GetByIDs retrieves multiple products by their IDs.
	GetByIDs(ctx context.Context, ids []string) ([]model.Product, error)
}

// OrderService defines operations for order management.
type OrderService interface {
	// CreateOrder creates a new order with optional coupon code validation.
	CreateOrder(ctx context.Context, req *model.OrderRequest) (*model.OrderResponse, error)

	// GetByID retrieves an order by its ID with all items and product details.
	GetByID(ctx context.Context, id uuid.UUID) (*model.OrderResponse, error)
}
