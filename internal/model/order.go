package model

import (
	"time"

	"github.com/google/uuid"
)

// Order represents a customer order.
type Order struct {
	ID         uuid.UUID `json:"id" db:"id"`
	CouponCode *string   `json:"couponCode,omitempty" db:"coupon_code"`
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time `json:"updatedAt" db:"updated_at"`
}

// OrderItem represents a line item in an order.
type OrderItem struct {
	ID        uuid.UUID `json:"-" db:"id"`
	OrderID   uuid.UUID `json:"-" db:"order_id"`
	ProductID string    `json:"productId" db:"product_id"`
	Quantity  int       `json:"quantity" db:"quantity"`
}

// OrderRequest represents the request payload for creating an order.
type OrderRequest struct {
	CouponCode *string            `json:"couponCode,omitempty"`
	Items      []OrderItemRequest `json:"items"`
}

// OrderItemRequest represents a single item in an order request.
type OrderItemRequest struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// OrderResponse represents the response payload for an order.
type OrderResponse struct {
	ID       uuid.UUID   `json:"id"`
	Items    []OrderItem `json:"items"`
	Products []Product   `json:"products"`
}
