package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"mini-kart/internal/model"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockProductRepository is a mock implementation of ProductRepository.
type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) GetAll(ctx context.Context, limit, offset int) ([]model.Product, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Product), args.Error(1)
}

func (m *MockProductRepository) GetByID(ctx context.Context, id string) (*model.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Product), args.Error(1)
}

func (m *MockProductRepository) GetByIDs(ctx context.Context, ids []string) ([]model.Product, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Product), args.Error(1)
}

func (m *MockProductRepository) ValidateProductsExist(ctx context.Context, ids []string) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func TestProductService_GetAll(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	testProducts := []model.Product{
		{ID: "P001", Name: "Product 1", Price: 10.00, Category: "Cat1", CreatedAt: time.Now()},
		{ID: "P002", Name: "Product 2", Price: 20.00, Category: "Cat2", CreatedAt: time.Now()},
	}

	tests := []struct {
		name          string
		limit         int
		offset        int
		expectedLimit int
		mockReturn    []model.Product
		mockError     error
		expectError   bool
	}{
		{
			name:          "Success with valid pagination",
			limit:         10,
			offset:        0,
			expectedLimit: 10,
			mockReturn:    testProducts,
			mockError:     nil,
			expectError:   false,
		},
		{
			name:          "Success with zero limit defaults to 10",
			limit:         0,
			offset:        0,
			expectedLimit: 10,
			mockReturn:    testProducts,
			mockError:     nil,
			expectError:   false,
		},
		{
			name:          "Success with negative limit defaults to 10",
			limit:         -5,
			offset:        0,
			expectedLimit: 10,
			mockReturn:    testProducts,
			mockError:     nil,
			expectError:   false,
		},
		{
			name:          "Success with limit exceeding max caps at 100",
			limit:         200,
			offset:        0,
			expectedLimit: 100,
			mockReturn:    testProducts,
			mockError:     nil,
			expectError:   false,
		},
		{
			name:          "Success with negative offset defaults to 0",
			limit:         10,
			offset:        -10,
			expectedLimit: 10,
			mockReturn:    testProducts,
			mockError:     nil,
			expectError:   false,
		},
		{
			name:          "Repository error",
			limit:         10,
			offset:        0,
			expectedLimit: 10,
			mockReturn:    nil,
			mockError:     errors.New("database error"),
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockProductRepository)
			service := NewProductService(mockRepo, logger)

			expectedOffset := tt.offset
			if expectedOffset < 0 {
				expectedOffset = 0
			}

			mockRepo.On("GetAll", ctx, tt.expectedLimit, expectedOffset).
				Return(tt.mockReturn, tt.mockError)

			products, err := service.GetAll(ctx, tt.limit, tt.offset)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, products)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.mockReturn, products)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestProductService_GetByID(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	testProduct := &model.Product{
		ID:        "P001",
		Name:      "Product 1",
		Price:     10.00,
		Category:  "Cat1",
		CreatedAt: time.Now(),
	}

	tests := []struct {
		name        string
		productID   string
		mockReturn  *model.Product
		mockError   error
		expectError bool
		expectedErr error
	}{
		{
			name:        "Success",
			productID:   "P001",
			mockReturn:  testProduct,
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "Product not found",
			productID:   "P999",
			mockReturn:  nil,
			mockError:   nil,
			expectError: true,
			expectedErr: model.ErrProductNotFound,
		},
		{
			name:        "Empty product ID",
			productID:   "",
			mockReturn:  nil,
			mockError:   nil,
			expectError: true,
			expectedErr: model.ErrProductNotFound,
		},
		{
			name:        "Repository error",
			productID:   "P001",
			mockReturn:  nil,
			mockError:   errors.New("database error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockProductRepository)
			service := NewProductService(mockRepo, logger)

			if tt.productID != "" {
				mockRepo.On("GetByID", ctx, tt.productID).
					Return(tt.mockReturn, tt.mockError)
			}

			product, err := service.GetByID(ctx, tt.productID)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, product)
				if tt.expectedErr != nil {
					assert.Equal(t, tt.expectedErr, err)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.mockReturn, product)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestProductService_GetByIDs(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	testProducts := []model.Product{
		{ID: "P001", Name: "Product 1", Price: 10.00, Category: "Cat1", CreatedAt: time.Now()},
		{ID: "P002", Name: "Product 2", Price: 20.00, Category: "Cat2", CreatedAt: time.Now()},
	}

	tests := []struct {
		name        string
		productIDs  []string
		mockReturn  []model.Product
		mockError   error
		expectError bool
	}{
		{
			name:        "Success with multiple IDs",
			productIDs:  []string{"P001", "P002"},
			mockReturn:  testProducts,
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "Success with single ID",
			productIDs:  []string{"P001"},
			mockReturn:  testProducts[:1],
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "Empty ID list returns empty result",
			productIDs:  []string{},
			mockReturn:  nil,
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "Repository error",
			productIDs:  []string{"P001", "P002"},
			mockReturn:  nil,
			mockError:   errors.New("database error"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockProductRepository)
			service := NewProductService(mockRepo, logger)

			if len(tt.productIDs) > 0 {
				mockRepo.On("GetByIDs", ctx, tt.productIDs).
					Return(tt.mockReturn, tt.mockError)
			}

			products, err := service.GetByIDs(ctx, tt.productIDs)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, products)
			} else {
				require.NoError(t, err)
				if len(tt.productIDs) == 0 {
					assert.Empty(t, products)
				} else {
					assert.Equal(t, tt.mockReturn, products)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
