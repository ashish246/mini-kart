package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"mini-kart/internal/model"
	"mini-kart/internal/service"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// OrderHandler handles order-related HTTP requests.
type OrderHandler struct {
	service service.OrderService
	logger  zerolog.Logger
}

// NewOrderHandler creates a new order handler.
func NewOrderHandler(service service.OrderService, logger zerolog.Logger) *OrderHandler {
	return &OrderHandler{
		service: service,
		logger:  logger.With().Str("handler", "order").Logger(),
	}
}

// Create handles POST /api/orders requests.
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", h.logger)
		return
	}

	var req model.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", h.logger)
		return
	}

	order, err := h.service.CreateOrder(r.Context(), &req)
	if err != nil {
		// Determine appropriate status code based on error type
		status := http.StatusInternalServerError
		message := "failed to create order"

		switch err {
		case model.ErrInvalidPromoCode:
			status = http.StatusBadRequest
			message = "invalid promo code"
		case model.ErrProductNotFound:
			status = http.StatusBadRequest
			message = "one or more products not found"
		case model.ErrInvalidQuantity:
			status = http.StatusBadRequest
			message = "invalid quantity"
		default:
			if strings.Contains(err.Error(), "required") ||
				strings.Contains(err.Error(), "must contain") ||
				strings.Contains(err.Error(), "nil") {
				status = http.StatusBadRequest
				message = err.Error()
			}
		}

		writeError(w, status, message, h.logger)
		return
	}

	writeJSON(w, http.StatusCreated, order)
}

// GetByID handles GET /api/orders/{id} requests.
func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", h.logger)
		return
	}

	// Extract order ID from path
	// Expecting path: /api/orders/{id}
	path := r.URL.Path
	if len(path) < len("/api/orders/") {
		writeError(w, http.StatusBadRequest, "order ID is required", h.logger)
		return
	}
	orderIDStr := path[len("/api/orders/"):]

	if orderIDStr == "" {
		writeError(w, http.StatusBadRequest, "order ID is required", h.logger)
		return
	}

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order ID format", h.logger)
		return
	}

	order, err := h.service.GetByID(r.Context(), orderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve order", h.logger)
		return
	}

	if order == nil {
		writeError(w, http.StatusNotFound, "order not found", h.logger)
		return
	}

	writeJSON(w, http.StatusOK, order)
}
