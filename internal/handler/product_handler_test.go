package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mini-kart/internal/model"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProductService is a mock implementation of ProductService.
type MockProductService struct {
	mock.Mock
}

func (m *MockProductService) GetAll(ctx context.Context, limit, offset int) ([]model.Product, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Product), args.Error(1)
}

func (m *MockProductService) GetByID(ctx context.Context, id string) (*model.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Product), args.Error(1)
}

func (m *MockProductService) GetByIDs(ctx context.Context, ids []string) ([]model.Product, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Product), args.Error(1)
}

func TestProductHandler_GetAll(t *testing.T) {
	logger := zerolog.Nop()

	testProducts := []model.Product{
		{ID: "P001", Name: "Product 1", Price: 10.00, Category: "Cat1", CreatedAt: time.Now()},
		{ID: "P002", Name: "Product 2", Price: 20.00, Category: "Cat2", CreatedAt: time.Now()},
	}

	tests := []struct {
		name           string
		method         string
		queryParams    string
		mockReturn     []model.Product
		mockError      error
		expectedStatus int
		expectService  bool
		limit          int
		offset         int
	}{
		{
			name:           "Success with default pagination",
			method:         http.MethodGet,
			queryParams:    "",
			mockReturn:     testProducts,
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectService:  true,
			limit:          10,
			offset:         0,
		},
		{
			name:           "Success with custom pagination",
			method:         http.MethodGet,
			queryParams:    "?limit=5&offset=10",
			mockReturn:     testProducts,
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectService:  true,
			limit:          5,
			offset:         10,
		},
		{
			name:           "Invalid limit parameter",
			method:         http.MethodGet,
			queryParams:    "?limit=invalid",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectService:  false,
		},
		{
			name:           "Invalid offset parameter",
			method:         http.MethodGet,
			queryParams:    "?offset=invalid",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectService:  false,
		},
		{
			name:           "Service error",
			method:         http.MethodGet,
			queryParams:    "",
			mockReturn:     nil,
			mockError:      errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
			expectService:  true,
			limit:          10,
			offset:         0,
		},
		{
			name:           "Method not allowed",
			method:         http.MethodPost,
			queryParams:    "",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectService:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockProductService)
			handler := NewProductHandler(mockService, logger)

			if tt.expectService {
				mockService.On("GetAll", mock.Anything, tt.limit, tt.offset).
					Return(tt.mockReturn, tt.mockError)
			}

			req := httptest.NewRequest(tt.method, "/api/products"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handler.GetAll(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectService {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestProductHandler_GetByID(t *testing.T) {
	logger := zerolog.Nop()

	testProduct := &model.Product{
		ID:        "P001",
		Name:      "Product 1",
		Price:     10.00,
		Category:  "Cat1",
		CreatedAt: time.Now(),
	}

	tests := []struct {
		name           string
		method         string
		path           string
		mockReturn     *model.Product
		mockError      error
		expectedStatus int
		expectService  bool
		productID      string
	}{
		{
			name:           "Success",
			method:         http.MethodGet,
			path:           "/api/products/P001",
			mockReturn:     testProduct,
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectService:  true,
			productID:      "P001",
		},
		{
			name:           "Product not found - service returns nil",
			method:         http.MethodGet,
			path:           "/api/products/P999",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusNotFound,
			expectService:  true,
			productID:      "P999",
		},
		{
			name:           "Product not found - service returns error",
			method:         http.MethodGet,
			path:           "/api/products/P999",
			mockReturn:     nil,
			mockError:      model.ErrProductNotFound,
			expectedStatus: http.StatusNotFound,
			expectService:  true,
			productID:      "P999",
		},
		{
			name:           "Missing product ID",
			method:         http.MethodGet,
			path:           "/api/products/",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectService:  false,
		},
		{
			name:           "Method not allowed",
			method:         http.MethodPost,
			path:           "/api/products/P001",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectService:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockProductService)
			handler := NewProductHandler(mockService, logger)

			if tt.expectService {
				mockService.On("GetByID", mock.Anything, tt.productID).
					Return(tt.mockReturn, tt.mockError)
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			handler.GetByID(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectService {
				mockService.AssertExpectations(t)
			}
		})
	}
}
