package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// CORS adds CORS headers to the response.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// APIKeyAuth validates the API key from the X-API-Key header.
func APIKeyAuth(apiKey string, logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for health check endpoint
			if r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			providedKey := r.Header.Get("X-API-Key")
			if providedKey == "" {
				logger.Warn().Str("path", r.URL.Path).Msg("missing API key")
				http.Error(w, "unauthorised: missing API key", http.StatusUnauthorized)
				return
			}

			if providedKey != apiKey {
				logger.Warn().
					Str("path", r.URL.Path).
					Str("provided_key", providedKey[:min(8, len(providedKey))]).
					Msg("invalid API key")
				http.Error(w, "unauthorised: invalid API key", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Logging logs HTTP requests with timing information.
func Logging(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", rw.statusCode).
				Dur("duration", duration).
				Str("remote_addr", r.RemoteAddr).
				Msg("http request")
		})
	}
}

// Recovery recovers from panics and returns a 500 error.
func Recovery(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error().
						Interface("panic", err).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Msg("panic recovered")

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": "internal server error"}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
