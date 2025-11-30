package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestCORS(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectHandler  bool
	}{
		{
			name:           "Preflight request",
			method:         http.MethodOptions,
			expectedStatus: http.StatusNoContent,
			expectHandler:  false,
		},
		{
			name:           "GET request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectHandler:  true,
		},
		{
			name:           "POST request",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
			expectHandler:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := CORS(testHandler)

			req := httptest.NewRequest(tt.method, "/test", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectHandler, handlerCalled)
			assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
			assert.Equal(t, "Content-Type, X-API-Key", w.Header().Get("Access-Control-Allow-Headers"))
		})
	}
}

func TestAPIKeyAuth(t *testing.T) {
	logger := zerolog.Nop()
	validAPIKey := "test-api-key-123"

	tests := []struct {
		name           string
		path           string
		apiKey         string
		expectedStatus int
		expectHandler  bool
	}{
		{
			name:           "Valid API key",
			path:           "/api/products",
			apiKey:         validAPIKey,
			expectedStatus: http.StatusOK,
			expectHandler:  true,
		},
		{
			name:           "Invalid API key",
			path:           "/api/products",
			apiKey:         "invalid-key",
			expectedStatus: http.StatusUnauthorized,
			expectHandler:  false,
		},
		{
			name:           "Missing API key",
			path:           "/api/products",
			apiKey:         "",
			expectedStatus: http.StatusUnauthorized,
			expectHandler:  false,
		},
		{
			name:           "Health check bypasses auth",
			path:           "/health",
			apiKey:         "",
			expectedStatus: http.StatusOK,
			expectHandler:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := APIKeyAuth(validAPIKey, logger)(testHandler)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectHandler, handlerCalled)
		})
	}
}

func TestLogging(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name           string
		method         string
		path           string
		handlerStatus  int
		expectedStatus int
	}{
		{
			name:           "Successful request",
			method:         http.MethodGet,
			path:           "/api/products",
			handlerStatus:  http.StatusOK,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Not found request",
			method:         http.MethodGet,
			path:           "/api/unknown",
			handlerStatus:  http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Server error",
			method:         http.MethodPost,
			path:           "/api/orders",
			handlerStatus:  http.StatusInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.handlerStatus)
			})

			handler := Logging(logger)(testHandler)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestRecovery(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name           string
		shouldPanic    bool
		panicValue     interface{}
		expectedStatus int
	}{
		{
			name:           "No panic",
			shouldPanic:    false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Panic with string",
			shouldPanic:    true,
			panicValue:     "something went wrong",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Panic with error",
			shouldPanic:    true,
			panicValue:     assert.AnError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.shouldPanic {
					panic(tt.panicValue)
				}
				w.WriteHeader(http.StatusOK)
			})

			handler := Recovery(logger)(testHandler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			// Ensure we don't panic in the test
			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.shouldPanic {
				assert.Contains(t, w.Body.String(), "internal server error")
			}
		})
	}
}

func TestResponseWriter(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedStatus int
	}{
		{
			name:           "Status OK",
			statusCode:     http.StatusOK,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Status Created",
			statusCode:     http.StatusCreated,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Status Not Found",
			statusCode:     http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Status Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			rw.WriteHeader(tt.statusCode)

			assert.Equal(t, tt.expectedStatus, rw.statusCode)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
