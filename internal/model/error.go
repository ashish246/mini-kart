package model

// ErrorResponse represents a standardised error response.
type ErrorResponse struct {
	Error         string `json:"error"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId,omitempty"`
}

// Standard error codes for API responses
const (
	ErrCodeInvalidJSON        = "INVALID_JSON"
	ErrCodeMissingField       = "MISSING_FIELD"
	ErrCodeInvalidPromoCode   = "INVALID_PROMO_CODE"
	ErrCodeInvalidPromoLength = "INVALID_PROMO_LENGTH"
	ErrCodeProductNotFound    = "PRODUCT_NOT_FOUND"
	ErrCodeInvalidQuantity    = "INVALID_QUANTITY"
	ErrCodeUnauthorised       = "UNAUTHORIZED"
	ErrCodeForbidden          = "FORBIDDEN"
	ErrCodeInternalError      = "INTERNAL_ERROR"
)

// Domain errors for business logic
type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string {
	return e.Message
}

// NewDomainError creates a new domain error
func NewDomainError(code, message string) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
	}
}

// Common domain errors
var (
	ErrInvalidPromoCode   = NewDomainError(ErrCodeInvalidPromoCode, "Promo code must appear in at least two coupon files")
	ErrInvalidPromoLength = NewDomainError(ErrCodeInvalidPromoLength, "Promo code must be between 8 and 10 characters")
	ErrProductNotFound    = NewDomainError(ErrCodeProductNotFound, "One or more products not found")
	ErrInvalidQuantity    = NewDomainError(ErrCodeInvalidQuantity, "Quantity must be greater than zero")
)
