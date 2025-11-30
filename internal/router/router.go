package router

import (
	"net/http"
	"strings"

	"mini-kart/internal/handler"
	"mini-kart/internal/middleware"

	"github.com/rs/zerolog"
)

// New creates a new HTTP router with all routes and middleware configured.
func New(
	productHandler *handler.ProductHandler,
	orderHandler *handler.OrderHandler,
	apiKey string,
	logger zerolog.Logger,
) http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint (no authentication required)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	})

	// Product handler function
	productRouteHandler := func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a request for a specific product ID
		if r.URL.Path != "/api/products" && r.URL.Path != "/api/products/" {
			productHandler.GetByID(w, r)
			return
		}
		productHandler.GetAll(w, r)
	}

	// Register product routes (both with and without trailing slash)
	mux.HandleFunc("/api/products", productRouteHandler)
	mux.HandleFunc("/api/products/", productRouteHandler)

	// Order handler function
	orderRouteHandler := func(w http.ResponseWriter, r *http.Request) {
		// Route based on method and path
		if r.Method == http.MethodPost && (r.URL.Path == "/api/orders" || r.URL.Path == "/api/orders/") {
			orderHandler.Create(w, r)
			return
		}

		// Check if this is a request for a specific order ID
		if strings.HasPrefix(r.URL.Path, "/api/orders/") && r.URL.Path != "/api/orders/" {
			orderHandler.GetByID(w, r)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}

	// Register order routes (both with and without trailing slash)
	mux.HandleFunc("/api/orders", orderRouteHandler)
	mux.HandleFunc("/api/orders/", orderRouteHandler)

	// Apply middleware in order: Recovery -> Logging -> CORS -> APIKeyAuth
	var handler http.Handler = mux
	handler = middleware.APIKeyAuth(apiKey, logger)(handler)
	handler = middleware.CORS(handler)
	handler = middleware.Logging(logger)(handler)
	handler = middleware.Recovery(logger)(handler)

	return handler
}
