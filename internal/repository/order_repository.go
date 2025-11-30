package repository

import (
	"context"
	"fmt"

	"mini-kart/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// orderRepository implements the OrderRepository interface using PostgreSQL.
type orderRepository struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// NewOrderRepository creates a new PostgreSQL-backed order repository.
func NewOrderRepository(pool *pgxpool.Pool, logger zerolog.Logger) OrderRepository {
	return &orderRepository{
		pool:   pool,
		logger: logger.With().Str("repository", "order").Logger(),
	}
}

// BeginTx starts a new database transaction.
func (r *orderRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		r.logger.Error().Err(err).Msg("failed to begin transaction")
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// CreateOrder inserts a new order within the provided transaction.
func (r *orderRepository) CreateOrder(ctx context.Context, tx pgx.Tx, order *model.Order) error {
	query := `
		INSERT INTO orders (id, coupon_code, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := tx.Exec(ctx, query, order.ID, order.CouponCode, order.CreatedAt, order.UpdatedAt)
	if err != nil {
		r.logger.Error().
			Err(err).
			Str("order_id", order.ID.String()).
			Msg("failed to create order")
		return fmt.Errorf("failed to create order: %w", err)
	}

	r.logger.Debug().
		Str("order_id", order.ID.String()).
		Msg("order created successfully")

	return nil
}

// CreateOrderItems inserts multiple order items within the provided transaction.
func (r *orderRepository) CreateOrderItems(ctx context.Context, tx pgx.Tx, items []model.OrderItem) error {
	if len(items) == 0 {
		return nil
	}

	query := `
		INSERT INTO order_items (id, order_id, product_id, quantity)
		VALUES ($1, $2, $3, $4)
	`

	batch := &pgx.Batch{}
	for _, item := range items {
		batch.Queue(query, item.ID, item.OrderID, item.ProductID, item.Quantity)
	}

	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	for i := 0; i < len(items); i++ {
		_, err := results.Exec()
		if err != nil {
			r.logger.Error().
				Err(err).
				Str("order_id", items[i].OrderID.String()).
				Str("product_id", items[i].ProductID).
				Msg("failed to create order item")
			return fmt.Errorf("failed to create order item: %w", err)
		}
	}

	r.logger.Debug().
		Int("count", len(items)).
		Msg("order items created successfully")

	return nil
}

// GetByID retrieves an order by its ID along with its items.
func (r *orderRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Order, []model.OrderItem, error) {
	// Retrieve order
	orderQuery := `
		SELECT id, coupon_code, created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	var order model.Order
	err := r.pool.QueryRow(ctx, orderQuery, id).Scan(
		&order.ID,
		&order.CouponCode,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			r.logger.Debug().Str("order_id", id.String()).Msg("order not found")
			return nil, nil, nil
		}
		r.logger.Error().Err(err).Str("order_id", id.String()).Msg("failed to query order")
		return nil, nil, fmt.Errorf("failed to query order: %w", err)
	}

	// Retrieve order items
	itemsQuery := `
		SELECT id, order_id, product_id, quantity
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`

	rows, err := r.pool.Query(ctx, itemsQuery, id)
	if err != nil {
		r.logger.Error().
			Err(err).
			Str("order_id", id.String()).
			Msg("failed to query order items")
		return nil, nil, fmt.Errorf("failed to query order items: %w", err)
	}
	defer rows.Close()

	var items []model.OrderItem
	for rows.Next() {
		var item model.OrderItem
		err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity)
		if err != nil {
			r.logger.Error().Err(err).Msg("failed to scan order item row")
			return nil, nil, fmt.Errorf("failed to scan order item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("error iterating order item rows")
		return nil, nil, fmt.Errorf("error iterating order items: %w", err)
	}

	return &order, items, nil
}
