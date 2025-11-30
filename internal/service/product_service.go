package service

import (
	"context"
	"fmt"

	"mini-kart/internal/model"
	"mini-kart/internal/repository"

	"github.com/rs/zerolog"
)

// productService implements ProductService.
type productService struct {
	productRepo repository.ProductRepository
	logger      zerolog.Logger
}

// NewProductService creates a new product service.
func NewProductService(productRepo repository.ProductRepository, logger zerolog.Logger) ProductService {
	return &productService{
		productRepo: productRepo,
		logger:      logger.With().Str("service", "product").Logger(),
	}
}

// GetAll retrieves all products with pagination.
func (s *productService) GetAll(ctx context.Context, limit, offset int) ([]model.Product, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	products, err := s.productRepo.GetAll(ctx, limit, offset)
	if err != nil {
		s.logger.Error().Err(err).
			Int("limit", limit).
			Int("offset", offset).
			Msg("failed to get all products")
		return nil, fmt.Errorf("failed to get products: %w", err)
	}

	s.logger.Debug().
		Int("count", len(products)).
		Int("limit", limit).
		Int("offset", offset).
		Msg("retrieved products")

	return products, nil
}

// GetByID retrieves a single product by ID.
func (s *productService) GetByID(ctx context.Context, id string) (*model.Product, error) {
	if id == "" {
		s.logger.Warn().Msg("product ID is empty")
		return nil, model.ErrProductNotFound
	}

	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("product_id", id).Msg("failed to get product by ID")
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	if product == nil {
		s.logger.Debug().Str("product_id", id).Msg("product not found")
		return nil, model.ErrProductNotFound
	}

	return product, nil
}

// GetByIDs retrieves multiple products by their IDs.
func (s *productService) GetByIDs(ctx context.Context, ids []string) ([]model.Product, error) {
	if len(ids) == 0 {
		return []model.Product{}, nil
	}

	products, err := s.productRepo.GetByIDs(ctx, ids)
	if err != nil {
		s.logger.Error().Err(err).Int("count", len(ids)).Msg("failed to get products by IDs")
		return nil, fmt.Errorf("failed to get products: %w", err)
	}

	s.logger.Debug().
		Int("requested", len(ids)).
		Int("found", len(products)).
		Msg("retrieved products by IDs")

	return products, nil
}
