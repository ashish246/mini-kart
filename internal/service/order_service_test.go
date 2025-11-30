package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"mini-kart/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockOrderRepository is a mock implementation of OrderRepository.
type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	// Return a MockTx interface value, not a pointer
	if tx, ok := args.Get(0).(pgx.Tx); ok {
		return tx, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockOrderRepository) CreateOrder(ctx context.Context, tx pgx.Tx, order *model.Order) error {
	args := m.Called(ctx, tx, order)
	return args.Error(0)
}

func (m *MockOrderRepository) CreateOrderItems(ctx context.Context, tx pgx.Tx, items []model.OrderItem) error {
	args := m.Called(ctx, tx, items)
	return args.Error(0)
}

func (m *MockOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Order, []model.OrderItem, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).(*model.Order), args.Get(1).([]model.OrderItem), args.Error(2)
}

// MockCouponValidator is a mock implementation of Validator.
type MockCouponValidator struct {
	mock.Mock
}

func (m *MockCouponValidator) Validate(ctx context.Context, promoCode string) error {
	args := m.Called(ctx, promoCode)
	return args.Error(0)
}

func (m *MockCouponValidator) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockTx is a minimal mock implementation of pgx.Tx for testing.
type MockTx struct {
	mock.Mock
	committed  bool
	rolledBack bool
}

func (m *MockTx) Commit(ctx context.Context) error {
	args := m.Called(ctx)
	m.committed = true
	return args.Error(0)
}

func (m *MockTx) Rollback(ctx context.Context) error {
	args := m.Called(ctx)
	m.rolledBack = true
	return args.Error(0)
}

// Stub methods to satisfy pgx.Tx interface - these are not used in our tests
func (m *MockTx) Begin(ctx context.Context) (pgx.Tx, error) { return nil, nil }
func (m *MockTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *MockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (m *MockTx) LargeObjects() pgx.LargeObjects                               { return pgx.LargeObjects{} }
func (m *MockTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (m *MockTx) Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error) {
	return
}
func (m *MockTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (m *MockTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row { return nil }
func (m *MockTx) Conn() *pgx.Conn                                               { return nil }

func TestOrderService_CreateOrder_Success(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	couponCode := "VALIDCODE1"
	req := &model.OrderRequest{
		CouponCode: &couponCode,
		Items: []model.OrderItemRequest{
			{ProductID: "P001", Quantity: 2},
			{ProductID: "P002", Quantity: 1},
		},
	}

	testProducts := []model.Product{
		{ID: "P001", Name: "Product 1", Price: 10.00, Category: "Cat1", CreatedAt: time.Now()},
		{ID: "P002", Name: "Product 2", Price: 20.00, Category: "Cat2", CreatedAt: time.Now()},
	}

	mockOrderRepo := new(MockOrderRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockCouponValidator)
	mockTx := new(MockTx)

	service := NewOrderService(mockOrderRepo, mockProductRepo, mockValidator, logger)

	// Set up expectations
	mockValidator.On("Validate", ctx, couponCode).Return(nil)
	mockProductRepo.On("ValidateProductsExist", ctx, []string{"P001", "P002"}).Return(nil)
	mockOrderRepo.On("BeginTx", ctx).Return(mockTx, nil)
	mockOrderRepo.On("CreateOrder", ctx, mockTx, mock.AnythingOfType("*model.Order")).Return(nil)
	mockOrderRepo.On("CreateOrderItems", ctx, mockTx, mock.AnythingOfType("[]model.OrderItem")).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockProductRepo.On("GetByIDs", ctx, []string{"P001", "P002"}).Return(testProducts, nil)

	// Execute
	resp, err := service.CreateOrder(ctx, req)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEqual(t, uuid.Nil, resp.ID)
	assert.Len(t, resp.Items, 2)
	assert.Len(t, resp.Products, 2)

	mockValidator.AssertExpectations(t)
	mockProductRepo.AssertExpectations(t)
	mockOrderRepo.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestOrderService_CreateOrder_WithoutCoupon(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	req := &model.OrderRequest{
		CouponCode: nil,
		Items: []model.OrderItemRequest{
			{ProductID: "P001", Quantity: 1},
		},
	}

	testProducts := []model.Product{
		{ID: "P001", Name: "Product 1", Price: 10.00, Category: "Cat1", CreatedAt: time.Now()},
	}

	mockOrderRepo := new(MockOrderRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockCouponValidator)
	mockTx := new(MockTx)

	service := NewOrderService(mockOrderRepo, mockProductRepo, mockValidator, logger)

	// Set up expectations (coupon validation should not be called)
	mockProductRepo.On("ValidateProductsExist", ctx, []string{"P001"}).Return(nil)
	mockOrderRepo.On("BeginTx", ctx).Return(mockTx, nil)
	mockOrderRepo.On("CreateOrder", ctx, mockTx, mock.AnythingOfType("*model.Order")).Return(nil)
	mockOrderRepo.On("CreateOrderItems", ctx, mockTx, mock.AnythingOfType("[]model.OrderItem")).Return(nil)
	mockTx.On("Commit", ctx).Return(nil)
	mockProductRepo.On("GetByIDs", ctx, []string{"P001"}).Return(testProducts, nil)

	// Execute
	resp, err := service.CreateOrder(ctx, req)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, resp)

	mockProductRepo.AssertExpectations(t)
	mockOrderRepo.AssertExpectations(t)
	mockTx.AssertExpectations(t)
	mockValidator.AssertNotCalled(t, "Validate")
}

func TestOrderService_CreateOrder_InvalidCoupon(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	couponCode := "INVALID123"
	req := &model.OrderRequest{
		CouponCode: &couponCode,
		Items: []model.OrderItemRequest{
			{ProductID: "P001", Quantity: 1},
		},
	}

	mockOrderRepo := new(MockOrderRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockCouponValidator)

	service := NewOrderService(mockOrderRepo, mockProductRepo, mockValidator, logger)

	// Set up expectations
	mockValidator.On("Validate", ctx, couponCode).Return(model.ErrInvalidPromoCode)

	// Execute
	resp, err := service.CreateOrder(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Equal(t, model.ErrInvalidPromoCode, err)
	assert.Nil(t, resp)

	mockValidator.AssertExpectations(t)
	mockProductRepo.AssertNotCalled(t, "ValidateProductsExist")
	mockOrderRepo.AssertNotCalled(t, "BeginTx")
}

func TestOrderService_CreateOrder_ProductNotFound(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	req := &model.OrderRequest{
		Items: []model.OrderItemRequest{
			{ProductID: "P999", Quantity: 1},
		},
	}

	mockOrderRepo := new(MockOrderRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockCouponValidator)

	service := NewOrderService(mockOrderRepo, mockProductRepo, mockValidator, logger)

	// Set up expectations
	mockProductRepo.On("ValidateProductsExist", ctx, []string{"P999"}).Return(model.ErrProductNotFound)

	// Execute
	resp, err := service.CreateOrder(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Equal(t, model.ErrProductNotFound, err)
	assert.Nil(t, resp)

	mockProductRepo.AssertExpectations(t)
	mockOrderRepo.AssertNotCalled(t, "BeginTx")
}

func TestOrderService_CreateOrder_ValidationErrors(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	mockOrderRepo := new(MockOrderRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockCouponValidator)

	service := NewOrderService(mockOrderRepo, mockProductRepo, mockValidator, logger)

	tests := []struct {
		name        string
		req         *model.OrderRequest
		expectedErr error
	}{
		{
			name:        "Nil request",
			req:         nil,
			expectedErr: nil, // Will error with "order request is nil"
		},
		{
			name: "Empty items",
			req: &model.OrderRequest{
				Items: []model.OrderItemRequest{},
			},
			expectedErr: nil, // Will error with "order must contain at least one item"
		},
		{
			name: "Empty product ID",
			req: &model.OrderRequest{
				Items: []model.OrderItemRequest{
					{ProductID: "", Quantity: 1},
				},
			},
			expectedErr: nil, // Will error with "product ID is required"
		},
		{
			name: "Zero quantity",
			req: &model.OrderRequest{
				Items: []model.OrderItemRequest{
					{ProductID: "P001", Quantity: 0},
				},
			},
			expectedErr: model.ErrInvalidQuantity,
		},
		{
			name: "Negative quantity",
			req: &model.OrderRequest{
				Items: []model.OrderItemRequest{
					{ProductID: "P001", Quantity: -5},
				},
			},
			expectedErr: model.ErrInvalidQuantity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.CreateOrder(ctx, tt.req)

			require.Error(t, err)
			assert.Nil(t, resp)
			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

func TestOrderService_CreateOrder_TransactionRollback(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	req := &model.OrderRequest{
		Items: []model.OrderItemRequest{
			{ProductID: "P001", Quantity: 1},
		},
	}

	mockOrderRepo := new(MockOrderRepository)
	mockProductRepo := new(MockProductRepository)
	mockValidator := new(MockCouponValidator)
	mockTx := new(MockTx)

	service := NewOrderService(mockOrderRepo, mockProductRepo, mockValidator, logger)

	// Set up expectations
	mockProductRepo.On("ValidateProductsExist", ctx, []string{"P001"}).Return(nil)
	mockOrderRepo.On("BeginTx", ctx).Return(mockTx, nil)
	mockOrderRepo.On("CreateOrder", ctx, mockTx, mock.AnythingOfType("*model.Order")).
		Return(errors.New("database error"))
	mockTx.On("Rollback", ctx).Return(nil)

	// Execute
	resp, err := service.CreateOrder(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, resp)

	mockProductRepo.AssertExpectations(t)
	mockOrderRepo.AssertExpectations(t)
	mockTx.AssertExpectations(t)
}

func TestOrderService_GetByID(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	orderID := uuid.New()
	order := &model.Order{
		ID:         orderID,
		CouponCode: nil,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	items := []model.OrderItem{
		{ID: uuid.New(), OrderID: orderID, ProductID: "P001", Quantity: 2},
		{ID: uuid.New(), OrderID: orderID, ProductID: "P002", Quantity: 1},
	}

	products := []model.Product{
		{ID: "P001", Name: "Product 1", Price: 10.00, Category: "Cat1", CreatedAt: time.Now()},
		{ID: "P002", Name: "Product 2", Price: 20.00, Category: "Cat2", CreatedAt: time.Now()},
	}

	tests := []struct {
		name         string
		orderID      uuid.UUID
		mockOrder    *model.Order
		mockItems    []model.OrderItem
		mockError    error
		mockProducts []model.Product
		expectNil    bool
		expectError  bool
	}{
		{
			name:         "Success",
			orderID:      orderID,
			mockOrder:    order,
			mockItems:    items,
			mockError:    nil,
			mockProducts: products,
			expectNil:    false,
			expectError:  false,
		},
		{
			name:        "Order not found",
			orderID:     uuid.New(),
			mockOrder:   nil,
			mockItems:   nil,
			mockError:   nil,
			expectNil:   true,
			expectError: false,
		},
		{
			name:        "Repository error",
			orderID:     orderID,
			mockOrder:   nil,
			mockItems:   nil,
			mockError:   errors.New("database error"),
			expectNil:   false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOrderRepo := new(MockOrderRepository)
			mockProductRepo := new(MockProductRepository)
			mockValidator := new(MockCouponValidator)

			service := NewOrderService(mockOrderRepo, mockProductRepo, mockValidator, logger)

			mockOrderRepo.On("GetByID", ctx, tt.orderID).Return(tt.mockOrder, tt.mockItems, tt.mockError)

			if tt.mockOrder != nil && !tt.expectError {
				productIDs := []string{"P001", "P002"}
				mockProductRepo.On("GetByIDs", ctx, productIDs).Return(tt.mockProducts, nil)
			}

			resp, err := service.GetByID(ctx, tt.orderID)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tt.expectNil {
				assert.Nil(t, resp)
			} else if !tt.expectError {
				require.NotNil(t, resp)
				assert.Equal(t, tt.orderID, resp.ID)
				assert.Equal(t, tt.mockItems, resp.Items)
				assert.Equal(t, tt.mockProducts, resp.Products)
			}

			mockOrderRepo.AssertExpectations(t)
			mockProductRepo.AssertExpectations(t)
		})
	}
}
