package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mini-kart/internal/coupon"
	"mini-kart/internal/handler"
	"mini-kart/internal/model"
	"mini-kart/internal/repository"
	"mini-kart/internal/router"
	"mini-kart/internal/service"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T, testDB *TestDB) http.Handler {
	t.Helper()

	logger := zerolog.Nop()
	ctx := context.Background()

	// Initialize repositories
	productRepo := repository.NewProductRepository(testDB.Pool, logger)
	orderRepo := repository.NewOrderRepository(testDB.Pool, logger)

	// Initialize coupon validator with test config
	couponLoader := coupon.NewFileLoader(logger)
	validatorConfig := &coupon.ValidatorConfig{
		FilePaths:     []string{}, // Empty for tests
		MinMatchCount: 1,
	}
	validator, err := coupon.NewValidator(ctx, validatorConfig, couponLoader, logger)
	require.NoError(t, err)
	t.Cleanup(func() {
		validator.Close()
	})

	// Initialize services
	productService := service.NewProductService(productRepo, logger)
	orderService := service.NewOrderService(orderRepo, productRepo, validator, logger)

	// Initialize handlers
	productHandler := handler.NewProductHandler(productService, logger)
	orderHandler := handler.NewOrderHandler(orderService, logger)

	// Create router
	return router.New(productHandler, orderHandler, "test-api-key", logger)
}

func TestProductAPI_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := SetupTestDB(t)
	server := setupTestServer(t, testDB)

	t.Run("GET /api/products returns all products", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
		req.Header.Set("X-API-Key", "test-api-key")
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var products []model.Product
		err := json.NewDecoder(w.Body).Decode(&products)
		require.NoError(t, err)
		assert.Len(t, products, 5)
	})

	t.Run("GET /api/products with pagination", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		req := httptest.NewRequest(http.MethodGet, "/api/products?limit=2&offset=0", nil)
		req.Header.Set("X-API-Key", "test-api-key")
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var products []model.Product
		err := json.NewDecoder(w.Body).Decode(&products)
		require.NoError(t, err)
		assert.Len(t, products, 2)
	})

	t.Run("GET /api/products/{id} returns specific product", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		req := httptest.NewRequest(http.MethodGet, "/api/products/P001", nil)
		req.Header.Set("X-API-Key", "test-api-key")
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var product model.Product
		err := json.NewDecoder(w.Body).Decode(&product)
		require.NoError(t, err)
		assert.Equal(t, "P001", product.ID)
		assert.Equal(t, "Test Product 1", product.Name)
	})

	t.Run("GET /api/products/{id} returns 404 for non-existent product", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)

		req := httptest.NewRequest(http.MethodGet, "/api/products/P999", nil)
		req.Header.Set("X-API-Key", "test-api-key")
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("GET /api/products without API key returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("GET /health returns 200 without API key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestOrderAPI_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := SetupTestDB(t)
	server := setupTestServer(t, testDB)

	t.Run("POST /api/orders creates order successfully", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		orderReq := &model.OrderRequest{
			CouponCode: nil,
			Items: []model.OrderItemRequest{
				{ProductID: "P001", Quantity: 2},
				{ProductID: "P002", Quantity: 1},
			},
		}

		body, err := json.Marshal(orderReq)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "test-api-key")
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp model.OrderResponse
		err = json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		assert.NotEqual(t, "", resp.ID.String())
		assert.Len(t, resp.Items, 2)
		assert.Len(t, resp.Products, 2)
	})

	t.Run("POST /api/orders fails with non-existent product", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		orderReq := &model.OrderRequest{
			Items: []model.OrderItemRequest{
				{ProductID: "P999", Quantity: 1},
			},
		}

		body, err := json.Marshal(orderReq)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "test-api-key")
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("POST /api/orders fails with invalid quantity", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		orderReq := &model.OrderRequest{
			Items: []model.OrderItemRequest{
				{ProductID: "P001", Quantity: -1},
			},
		}

		body, err := json.Marshal(orderReq)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "test-api-key")
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("POST /api/orders without API key returns 401", func(t *testing.T) {
		orderReq := &model.OrderRequest{
			Items: []model.OrderItemRequest{
				{ProductID: "P001", Quantity: 1},
			},
		}

		body, err := json.Marshal(orderReq)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("GET /api/orders/{id} returns order", func(t *testing.T) {
		CleanupDB(t, testDB.Pool)
		SeedProducts(t, testDB.Pool)

		// First create an order
		orderReq := &model.OrderRequest{
			Items: []model.OrderItemRequest{
				{ProductID: "P001", Quantity: 1},
			},
		}

		body, err := json.Marshal(orderReq)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "test-api-key")
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		var createResp model.OrderResponse
		err = json.NewDecoder(w.Body).Decode(&createResp)
		require.NoError(t, err)

		// Now retrieve the order
		req = httptest.NewRequest(http.MethodGet, "/api/orders/"+createResp.ID.String(), nil)
		req.Header.Set("X-API-Key", "test-api-key")
		w = httptest.NewRecorder()

		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var getResp model.OrderResponse
		err = json.NewDecoder(w.Body).Decode(&getResp)
		require.NoError(t, err)
		assert.Equal(t, createResp.ID, getResp.ID)
	})
}

func TestCORS_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDB := SetupTestDB(t)
	server := setupTestServer(t, testDB)

	t.Run("OPTIONS request returns CORS headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/products", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	})
}
