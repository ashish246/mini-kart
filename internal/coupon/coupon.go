package coupon

import (
	"context"
)

// Validator defines the interface for promo code validation.
type Validator interface {
	// Validate checks if a promo code is valid.
	// A valid promo code must:
	// - Be between 8 and 10 characters in length
	// - Appear in at least 2 out of 3 coupon files
	Validate(ctx context.Context, promoCode string) error

	// Close releases resources held by the validator.
	Close() error
}

// CouponSet represents a set of coupon codes for fast lookup.
type CouponSet interface {
	// Contains checks if a coupon code exists in the set.
	Contains(code string) bool

	// Size returns the number of coupons in the set.
	Size() int
}

// Loader defines the interface for loading coupon files.
type Loader interface {
	// Load reads a gzipped coupon file and returns a CouponSet.
	Load(ctx context.Context, filePath string) (CouponSet, error)
}
