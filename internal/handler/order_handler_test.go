package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mini-kart/internal/model"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockOrderService is a mock implementation of OrderService.
type MockOrderService struct {
	mock.Mock
}

func (m *MockOrderService) CreateOrder(ctx context.Context, req *model.OrderRequest) (*model.OrderResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.OrderResponse), args.Error(1)
}

func (m *MockOrderService) GetByID(ctx context.Context, id uuid.UUID) (*model.OrderResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.OrderResponse), args.Error(1)
}

func TestOrderHandler_Create(t *testing.T) {
	logger := zerolog.Nop()

	orderID := uuid.New()
	testResponse := &model.OrderResponse{
		ID: orderID,
		Items: []model.OrderItem{
			{ID: uuid.New(), OrderID: orderID, ProductID: "P001", Quantity: 2},
		},
		Products: []model.Product{
			{ID: "P001", Name: "Product 1", Price: 10.00, Category: "Cat1", CreatedAt: time.Now()},
		},
	}

	tests := []struct {
		name           string
		method         string
		requestBody    interface{}
		mockReturn     *model.OrderResponse
		mockError      error
		expectedStatus int
		expectService  bool
	}{
		{
			name:   "Success",
			method: http.MethodPost,
			requestBody: &model.OrderRequest{
				Items: []model.OrderItemRequest{
					{ProductID: "P001", Quantity: 2},
				},
			},
			mockReturn:     testResponse,
			mockError:      nil,
			expectedStatus: http.StatusCreated,
			expectService:  true,
		},
		{
			name:   "Invalid promo code",
			method: http.MethodPost,
			requestBody: &model.OrderRequest{
				CouponCode: func() *string { s := "INVALID"; return &s }(),
				Items: []model.OrderItemRequest{
					{ProductID: "P001", Quantity: 2},
				},
			},
			mockReturn:     nil,
			mockError:      model.ErrInvalidPromoCode,
			expectedStatus: http.StatusBadRequest,
			expectService:  true,
		},
		{
			name:   "Product not found",
			method: http.MethodPost,
			requestBody: &model.OrderRequest{
				Items: []model.OrderItemRequest{
					{ProductID: "P999", Quantity: 2},
				},
			},
			mockReturn:     nil,
			mockError:      model.ErrProductNotFound,
			expectedStatus: http.StatusBadRequest,
			expectService:  true,
		},
		{
			name:   "Invalid quantity",
			method: http.MethodPost,
			requestBody: &model.OrderRequest{
				Items: []model.OrderItemRequest{
					{ProductID: "P001", Quantity: -1},
				},
			},
			mockReturn:     nil,
			mockError:      model.ErrInvalidQuantity,
			expectedStatus: http.StatusBadRequest,
			expectService:  true,
		},
		{
			name:   "Validation error - required field",
			method: http.MethodPost,
			requestBody: &model.OrderRequest{
				Items: []model.OrderItemRequest{},
			},
			mockReturn:     nil,
			mockError:      errors.New("order must contain at least one item"),
			expectedStatus: http.StatusBadRequest,
			expectService:  true,
		},
		{
			name:           "Invalid JSON",
			method:         http.MethodPost,
			requestBody:    "invalid json",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectService:  false,
		},
		{
			name:           "Method not allowed",
			method:         http.MethodGet,
			requestBody:    nil,
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectService:  false,
		},
		{
			name:   "Service internal error",
			method: http.MethodPost,
			requestBody: &model.OrderRequest{
				Items: []model.OrderItemRequest{
					{ProductID: "P001", Quantity: 2},
				},
			},
			mockReturn:     nil,
			mockError:      errors.New("database connection failed"),
			expectedStatus: http.StatusInternalServerError,
			expectService:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOrderService)
			handler := NewOrderHandler(mockService, logger)

			var body []byte
			if tt.requestBody != nil {
				if str, ok := tt.requestBody.(string); ok {
					body = []byte(str)
				} else {
					var err error
					body, err = json.Marshal(tt.requestBody)
					require.NoError(t, err)
				}
			}

			if tt.expectService {
				mockService.On("CreateOrder", mock.Anything, mock.AnythingOfType("*model.OrderRequest")).
					Return(tt.mockReturn, tt.mockError)
			}

			req := httptest.NewRequest(tt.method, "/api/orders", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Create(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectService {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestOrderHandler_GetByID(t *testing.T) {
	logger := zerolog.Nop()

	orderID := uuid.New()
	testResponse := &model.OrderResponse{
		ID: orderID,
		Items: []model.OrderItem{
			{ID: uuid.New(), OrderID: orderID, ProductID: "P001", Quantity: 2},
		},
		Products: []model.Product{
			{ID: "P001", Name: "Product 1", Price: 10.00, Category: "Cat1", CreatedAt: time.Now()},
		},
	}

	tests := []struct {
		name           string
		method         string
		path           string
		mockReturn     *model.OrderResponse
		mockError      error
		expectedStatus int
		expectService  bool
		orderID        uuid.UUID
	}{
		{
			name:           "Success",
			method:         http.MethodGet,
			path:           "/api/orders/" + orderID.String(),
			mockReturn:     testResponse,
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectService:  true,
			orderID:        orderID,
		},
		{
			name:           "Order not found - service returns nil",
			method:         http.MethodGet,
			path:           "/api/orders/" + uuid.New().String(),
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusNotFound,
			expectService:  true,
			orderID:        uuid.New(),
		},
		{
			name:           "Order not found - service returns error",
			method:         http.MethodGet,
			path:           "/api/orders/" + uuid.New().String(),
			mockReturn:     nil,
			mockError:      errors.New("order not found"),
			expectedStatus: http.StatusInternalServerError,
			expectService:  true,
			orderID:        uuid.New(),
		},
		{
			name:           "Invalid UUID format",
			method:         http.MethodGet,
			path:           "/api/orders/invalid-uuid",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectService:  false,
		},
		{
			name:           "Missing order ID",
			method:         http.MethodGet,
			path:           "/api/orders/",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectService:  false,
		},
		{
			name:           "Method not allowed",
			method:         http.MethodPut,
			path:           "/api/orders/" + orderID.String(),
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectService:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockOrderService)
			handler := NewOrderHandler(mockService, logger)

			if tt.expectService {
				mockService.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
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
