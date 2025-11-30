package handler

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
)

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log the error but don't expose it to the client
		return
	}
}

// writeError writes an error response with the given status code and message.
func writeError(w http.ResponseWriter, status int, message string, logger zerolog.Logger) {
	logger.Error().Str("error", message).Int("status", status).Msg("handler error")
	writeJSON(w, status, ErrorResponse{Error: message})
}
