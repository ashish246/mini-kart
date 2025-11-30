package handler

import (
	"net/http"
	"strconv"

	"mini-kart/internal/service"

	"github.com/rs/zerolog"
)

// ProductHandler handles product-related HTTP requests.
type ProductHandler struct {
	service service.ProductService
	logger  zerolog.Logger
}

// NewProductHandler creates a new product handler.
func NewProductHandler(service service.ProductService, logger zerolog.Logger) *ProductHandler {
	return &ProductHandler{
		service: service,
		logger:  logger.With().Str("handler", "product").Logger(),
	}
}

// GetAll handles GET /api/products requests with pagination.
func (h *ProductHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", h.logger)
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 10 // default
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid limit parameter", h.logger)
			return
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		var err error
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid offset parameter", h.logger)
			return
		}
	}

	products, err := h.service.GetAll(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve products", h.logger)
		return
	}

	writeJSON(w, http.StatusOK, products)
}

// GetByID handles GET /api/products/{id} requests.
func (h *ProductHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", h.logger)
		return
	}

	// Extract product ID from path
	// Expecting path: /api/products/{id}
	// Simple extraction without routing library
	path := r.URL.Path
	if len(path) < len("/api/products/") {
		writeError(w, http.StatusBadRequest, "product ID is required", h.logger)
		return
	}
	productID := path[len("/api/products/"):]

	if productID == "" {
		writeError(w, http.StatusBadRequest, "product ID is required", h.logger)
		return
	}

	product, err := h.service.GetByID(r.Context(), productID)
	if err != nil {
		writeError(w, http.StatusNotFound, "product not found", h.logger)
		return
	}

	if product == nil {
		writeError(w, http.StatusNotFound, "product not found", h.logger)
		return
	}

	writeJSON(w, http.StatusOK, product)
}
