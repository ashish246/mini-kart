package coupon

import (
	"context"
	"fmt"
	"sync"

	"mini-kart/internal/model"

	"github.com/rs/zerolog"
)

// validator implements Validator with concurrent coupon file lookups.
type validator struct {
	couponSets []CouponSet
	logger     zerolog.Logger
	mu         sync.RWMutex
}

// ValidatorConfig holds configuration for the coupon validator.
type ValidatorConfig struct {
	// FilePaths is the list of coupon file paths to load.
	FilePaths []string

	// MinMatchCount is the minimum number of files a code must appear in.
	// Default: 2
	MinMatchCount int
}

// DefaultValidatorConfig returns the default validator configuration.
func DefaultValidatorConfig() *ValidatorConfig {
	return &ValidatorConfig{
		FilePaths: []string{
			"data/coupons/couponbase1.gz",
			"data/coupons/couponbase2.gz",
			"data/coupons/couponbase3.gz",
		},
		MinMatchCount: 2,
	}
}

// NewValidator creates a new coupon validator.
// It loads all coupon files at initialization time.
func NewValidator(ctx context.Context, config *ValidatorConfig, loader Loader, logger zerolog.Logger) (Validator, error) {
	if config == nil {
		config = DefaultValidatorConfig()
	}

	logger = logger.With().Str("component", "coupon-validator").Logger()

	logger.Info().
		Int("file_count", len(config.FilePaths)).
		Int("min_match_count", config.MinMatchCount).
		Msg("initialising coupon validator")

	v := &validator{
		couponSets: make([]CouponSet, 0, len(config.FilePaths)),
		logger:     logger,
	}

	// Load all coupon files concurrently
	type loadResult struct {
		index int
		set   CouponSet
		err   error
	}

	resultChan := make(chan loadResult, len(config.FilePaths))
	var wg sync.WaitGroup

	for i, filePath := range config.FilePaths {
		wg.Add(1)
		go func(index int, path string) {
			defer wg.Done()

			set, err := loader.Load(ctx, path)
			resultChan <- loadResult{
				index: index,
				set:   set,
				err:   err,
			}
		}(i, filePath)
	}

	// Wait for all loads to complete
	wg.Wait()
	close(resultChan)

	// Collect results in order
	results := make([]loadResult, len(config.FilePaths))
	for result := range resultChan {
		results[result.index] = result
	}

	// Check for errors and populate coupon sets
	for i, result := range results {
		if result.err != nil {
			logger.Error().
				Err(result.err).
				Str("file", config.FilePaths[i]).
				Msg("failed to load coupon file")
			return nil, fmt.Errorf("failed to load coupon file %s: %w", config.FilePaths[i], result.err)
		}
		v.couponSets = append(v.couponSets, result.set)
		logger.Info().
			Str("file", config.FilePaths[i]).
			Int("size", result.set.Size()).
			Msg("coupon file loaded")
	}

	totalCoupons := 0
	for _, set := range v.couponSets {
		totalCoupons += set.Size()
	}

	logger.Info().
		Int("total_coupons", totalCoupons).
		Msg("coupon validator initialised successfully")

	return v, nil
}

// Validate checks if a promo code is valid.
// A valid promo code must:
// - Be between 8 and 10 characters in length
// - Appear in at least 2 out of 3 coupon files
func (v *validator) Validate(ctx context.Context, promoCode string) error {
	// Validate length first (cheap check)
	if len(promoCode) < 8 || len(promoCode) > 10 {
		v.logger.Debug().
			Str("promo_code", promoCode).
			Int("length", len(promoCode)).
			Msg("promo code length invalid")
		return model.ErrInvalidPromoLength
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	// Check presence in coupon files concurrently
	matchCount := v.countMatches(ctx, promoCode)

	if matchCount < 2 {
		v.logger.Debug().
			Str("promo_code", promoCode).
			Int("match_count", matchCount).
			Msg("promo code not found in sufficient files")
		return model.ErrInvalidPromoCode
	}

	v.logger.Debug().
		Str("promo_code", promoCode).
		Int("match_count", matchCount).
		Msg("promo code validated successfully")

	return nil
}

// countMatches counts how many coupon files contain the given promo code.
// Searches are performed concurrently for optimal performance.
func (v *validator) countMatches(ctx context.Context, promoCode string) int {
	type matchResult struct {
		found bool
	}

	resultChan := make(chan matchResult, len(v.couponSets))
	var wg sync.WaitGroup

	for _, set := range v.couponSets {
		wg.Add(1)
		go func(s CouponSet) {
			defer wg.Done()

			// Check for context cancellation
			select {
			case <-ctx.Done():
				resultChan <- matchResult{found: false}
				return
			default:
			}

			found := s.Contains(promoCode)
			resultChan <- matchResult{found: found}
		}(set)
	}

	// Wait for all searches to complete
	wg.Wait()
	close(resultChan)

	// Count matches
	matches := 0
	for result := range resultChan {
		if result.found {
			matches++
		}
	}

	return matches
}

// Close releases resources held by the validator.
func (v *validator) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Clear coupon sets to allow GC to reclaim memory
	v.couponSets = nil

	v.logger.Info().Msg("coupon validator closed")

	return nil
}
