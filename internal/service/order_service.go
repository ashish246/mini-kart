package service

import (
	"context"
	"fmt"
	"time"

	"mini-kart/internal/coupon"
	"mini-kart/internal/model"
	"mini-kart/internal/repository"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// orderService implements OrderService.
type orderService struct {
	orderRepo   repository.OrderRepository
	productRepo repository.ProductRepository
	validator   coupon.Validator
	logger      zerolog.Logger
}

// NewOrderService creates a new order service.
func NewOrderService(
	orderRepo repository.OrderRepository,
	productRepo repository.ProductRepository,
	validator coupon.Validator,
	logger zerolog.Logger,
) OrderService {
	return &orderService{
		orderRepo:   orderRepo,
		productRepo: productRepo,
		validator:   validator,
		logger:      logger.With().Str("service", "order").Logger(),
	}
}

// CreateOrder creates a new order with optional coupon code validation.
func (s *orderService) CreateOrder(ctx context.Context, req *model.OrderRequest) (*model.OrderResponse, error) {
	// Validate request
	if err := s.validateOrderRequest(req); err != nil {
		return nil, err
	}

	// Validate coupon code if provided
	if req.CouponCode != nil && *req.CouponCode != "" {
		if err := s.validator.Validate(ctx, *req.CouponCode); err != nil {
			s.logger.Warn().
				Str("coupon_code", *req.CouponCode).
				Err(err).
				Msg("invalid coupon code")
			return nil, err
		}
		s.logger.Debug().Str("coupon_code", *req.CouponCode).Msg("coupon code validated")
	}

	// Extract product IDs and validate they exist
	productIDs := make([]string, len(req.Items))
	for i, item := range req.Items {
		productIDs[i] = item.ProductID
	}

	if err := s.productRepo.ValidateProductsExist(ctx, productIDs); err != nil {
		s.logger.Warn().
			Int("product_count", len(productIDs)).
			Err(err).
			Msg("product validation failed")
		return nil, err
	}

	// Start transaction
	tx, err := s.orderRepo.BeginTx(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to begin transaction")
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Ensure transaction is rolled back on error
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				s.logger.Error().Err(rbErr).Msg("failed to rollback transaction")
			}
		}
	}()

	// Create order
	now := time.Now()
	order := &model.Order{
		ID:         uuid.New(),
		CouponCode: req.CouponCode,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err = s.orderRepo.CreateOrder(ctx, tx, order); err != nil {
		s.logger.Error().Err(err).Str("order_id", order.ID.String()).Msg("failed to create order")
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Create order items
	orderItems := make([]model.OrderItem, len(req.Items))
	for i, item := range req.Items {
		orderItems[i] = model.OrderItem{
			ID:        uuid.New(),
			OrderID:   order.ID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	if err = s.orderRepo.CreateOrderItems(ctx, tx, orderItems); err != nil {
		s.logger.Error().
			Err(err).
			Str("order_id", order.ID.String()).
			Int("item_count", len(orderItems)).
			Msg("failed to create order items")
		return nil, fmt.Errorf("failed to create order items: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		s.logger.Error().Err(err).Str("order_id", order.ID.String()).Msg("failed to commit transaction")
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Retrieve product details
	products, err := s.productRepo.GetByIDs(ctx, productIDs)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to retrieve product details")
		return nil, fmt.Errorf("failed to retrieve product details: %w", err)
	}

	s.logger.Info().
		Str("order_id", order.ID.String()).
		Int("item_count", len(orderItems)).
		Msg("order created successfully")

	return &model.OrderResponse{
		ID:       order.ID,
		Items:    orderItems,
		Products: products,
	}, nil
}

// GetByID retrieves an order by its ID with all items and product details.
func (s *orderService) GetByID(ctx context.Context, id uuid.UUID) (*model.OrderResponse, error) {
	order, items, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("order_id", id.String()).Msg("failed to get order")
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order == nil {
		s.logger.Debug().Str("order_id", id.String()).Msg("order not found")
		return nil, nil
	}

	// Extract product IDs
	productIDs := make([]string, len(items))
	for i, item := range items {
		productIDs[i] = item.ProductID
	}

	// Retrieve product details
	products, err := s.productRepo.GetByIDs(ctx, productIDs)
	if err != nil {
		s.logger.Error().Err(err).Str("order_id", id.String()).Msg("failed to retrieve product details")
		return nil, fmt.Errorf("failed to retrieve product details: %w", err)
	}

	return &model.OrderResponse{
		ID:       order.ID,
		Items:    items,
		Products: products,
	}, nil
}

// validateOrderRequest validates the order request.
func (s *orderService) validateOrderRequest(req *model.OrderRequest) error {
	if req == nil {
		return fmt.Errorf("order request is nil")
	}

	if len(req.Items) == 0 {
		return fmt.Errorf("order must contain at least one item")
	}

	// Validate each item
	for i, item := range req.Items {
		if item.ProductID == "" {
			return fmt.Errorf("item %d: product ID is required", i)
		}

		if item.Quantity <= 0 {
			s.logger.Warn().
				Int("item_index", i).
				Str("product_id", item.ProductID).
				Int("quantity", item.Quantity).
				Msg("invalid quantity")
			return model.ErrInvalidQuantity
		}
	}

	return nil
}
