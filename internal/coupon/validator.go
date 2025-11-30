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
	// No mutex needed - coupon sets are read-only after initialization
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

	// Check presence in coupon files concurrently with early termination
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
// Uses worker pool pattern with early termination when 2 matches are found.
func (v *validator) countMatches(ctx context.Context, promoCode string) int {
	// Use buffered channel to prevent goroutine leaks on early termination
	resultChan := make(chan bool, len(v.couponSets))
	doneChan := make(chan struct{})
	defer close(doneChan)

	// Launch workers for each coupon set
	// Workers will exit early if doneChan is closed
	for _, set := range v.couponSets {
		go func(s CouponSet) {
			// Check if we should exit early
			select {
			case <-doneChan:
				return
			case <-ctx.Done():
				return
			default:
			}

			found := s.Contains(promoCode)

			// Try to send result, but exit if done or context cancelled
			select {
			case resultChan <- found:
			case <-doneChan:
				return
			case <-ctx.Done():
				return
			}
		}(set)
	}

	// Count matches with early termination
	matches := 0
	checked := 0

	for checked < len(v.couponSets) {
		select {
		case found := <-resultChan:
			checked++
			if found {
				matches++
				// Early termination: if we have 2 matches, we're done
				if matches >= 2 {
					return matches
				}
			}
			// Early termination: if we can't possibly get 2 matches, exit
			remaining := len(v.couponSets) - checked
			if matches+remaining < 2 {
				return matches
			}
		case <-ctx.Done():
			return matches
		}
	}

	return matches
}

// Close releases resources held by the validator.
func (v *validator) Close() error {
	// Clear coupon sets to allow GC to reclaim memory
	v.couponSets = nil

	v.logger.Info().Msg("coupon validator closed")

	return nil
}
