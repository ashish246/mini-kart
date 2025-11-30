package repository

import (
	"context"
	"fmt"

	"mini-kart/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// productRepository implements the ProductRepository interface using PostgreSQL.
type productRepository struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// NewProductRepository creates a new PostgreSQL-backed product repository.
func NewProductRepository(pool *pgxpool.Pool, logger zerolog.Logger) ProductRepository {
	return &productRepository{
		pool:   pool,
		logger: logger.With().Str("repository", "product").Logger(),
	}
}

// GetAll retrieves all products with pagination support.
func (r *productRepository) GetAll(ctx context.Context, limit, offset int) ([]model.Product, error) {
	query := `
		SELECT id, name, price, category, created_at
		FROM products
		ORDER BY name
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		r.logger.Error().Err(err).
			Int("limit", limit).
			Int("offset", offset).
			Msg("failed to query products")
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Category, &p.CreatedAt)
		if err != nil {
			r.logger.Error().Err(err).Msg("failed to scan product row")
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("error iterating product rows")
		return nil, fmt.Errorf("error iterating products: %w", err)
	}

	return products, nil
}

// GetByID retrieves a single product by its ID.
func (r *productRepository) GetByID(ctx context.Context, id string) (*model.Product, error) {
	query := `
		SELECT id, name, price, category, created_at
		FROM products
		WHERE id = $1
	`

	var p model.Product
	err := r.pool.QueryRow(ctx, query, id).Scan(&p.ID, &p.Name, &p.Price, &p.Category, &p.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			r.logger.Debug().Str("product_id", id).Msg("product not found")
			return nil, nil
		}
		r.logger.Error().Err(err).Str("product_id", id).Msg("failed to query product")
		return nil, fmt.Errorf("failed to query product: %w", err)
	}

	return &p, nil
}

// GetByIDs retrieves multiple products by their IDs.
func (r *productRepository) GetByIDs(ctx context.Context, ids []string) ([]model.Product, error) {
	if len(ids) == 0 {
		return []model.Product{}, nil
	}

	query := `
		SELECT id, name, price, category, created_at
		FROM products
		WHERE id = ANY($1)
		ORDER BY name
	`

	rows, err := r.pool.Query(ctx, query, ids)
	if err != nil {
		r.logger.Error().Err(err).Int("count", len(ids)).Msg("failed to query products by IDs")
		return nil, fmt.Errorf("failed to query products by IDs: %w", err)
	}
	defer rows.Close()

	var products []model.Product
	for rows.Next() {
		var p model.Product
		err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Category, &p.CreatedAt)
		if err != nil {
			r.logger.Error().Err(err).Msg("failed to scan product row")
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error().Err(err).Msg("error iterating product rows")
		return nil, fmt.Errorf("error iterating products: %w", err)
	}

	return products, nil
}

// ValidateProductsExist checks if all provided product IDs exist in the database.
// Returns error if any product ID does not exist.
func (r *productRepository) ValidateProductsExist(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Query to check how many of the provided IDs exist
	query := `
		SELECT COUNT(DISTINCT id)
		FROM products
		WHERE id = ANY($1)
	`

	var count int
	err := r.pool.QueryRow(ctx, query, ids).Scan(&count)
	if err != nil {
		r.logger.Error().Err(err).Int("count", len(ids)).Msg("failed to validate products exist")
		return fmt.Errorf("failed to validate products exist: %w", err)
	}

	if count != len(ids) {
		r.logger.Warn().
			Int("expected", len(ids)).
			Int("found", count).
			Msg("not all product IDs exist")
		return model.ErrProductNotFound
	}

	return nil
}
