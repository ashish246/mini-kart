package coupon

import (
	"context"
	"testing"

	"mini-kart/internal/model"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_WithSampleCouponFiles tests the validator with real sample coupon files.
// This test requires the sample coupon files to be generated first by running:
// go run scripts/generate_sample_coupons.go
func TestIntegration_WithSampleCouponFiles(t *testing.T) {
	logger := zerolog.Nop()

	// Try both relative paths (from project root and from package directory)
	possiblePaths := [][]string{
		{
			"data/coupons/couponbase1.gz",
			"data/coupons/couponbase2.gz",
			"data/coupons/couponbase3.gz",
		},
		{
			"../../data/coupons/couponbase1.gz",
			"../../data/coupons/couponbase2.gz",
			"../../data/coupons/couponbase3.gz",
		},
	}

	var config *ValidatorConfig
	for _, paths := range possiblePaths {
		config = &ValidatorConfig{
			FilePaths:     paths,
			MinMatchCount: 2,
		}
		loader := NewFileLoader(logger)
		ctx := context.Background()

		validator, err := NewValidator(ctx, config, loader, logger)
		if err == nil {
			defer validator.Close()
			// Successfully loaded, run tests with this validator
			runIntegrationTests(t, ctx, validator)
			return
		}
	}

	t.Skipf("Skipping integration test - sample coupon files not found. Run: go run scripts/generate_sample_coupons.go")
}

func runIntegrationTests(t *testing.T, ctx context.Context, validator Validator) {

	tests := []struct {
		name      string
		promoCode string
		expectErr error
	}{
		// Valid codes (appear in at least 2 files)
		{
			name:      "Valid code in files 1 and 2",
			promoCode: "VALIDONE1",
			expectErr: nil,
		},
		{
			name:      "Valid code in files 1 and 2 (second code)",
			promoCode: "VALIDTWO12",
			expectErr: nil,
		},
		{
			name:      "Valid code in all 3 files",
			promoCode: "ALLTHREE1",
			expectErr: nil,
		},
		{
			name:      "Valid code in files 1 and 3",
			promoCode: "SUMMER2024",
			expectErr: nil,
		},
		{
			name:      "Valid code in files 2 and 3",
			promoCode: "WINTER2024",
			expectErr: nil,
		},

		// Invalid codes (appear in only 1 file)
		{
			name:      "Invalid code - only in file 1",
			promoCode: "ONLYONE111",
			expectErr: model.ErrInvalidPromoCode,
		},
		{
			name:      "Invalid code - only in file 2",
			promoCode: "ONLYTWO222",
			expectErr: model.ErrInvalidPromoCode,
		},
		{
			name:      "Invalid code - only in file 3",
			promoCode: "ONLYTHREE3",
			expectErr: model.ErrInvalidPromoCode,
		},
		{
			name:      "Invalid code - only in file 3 (second code)",
			promoCode: "SPRING2024",
			expectErr: model.ErrInvalidPromoCode,
		},

		// Invalid codes (do not exist)
		{
			name:      "Invalid code - does not exist",
			promoCode: "NOTEXIST1",
			expectErr: model.ErrInvalidPromoCode,
		},

		// Invalid length
		{
			name:      "Invalid length - too short",
			promoCode: "SHORT12",
			expectErr: model.ErrInvalidPromoLength,
		},
		{
			name:      "Invalid length - too long",
			promoCode: "TOOLONGCODE",
			expectErr: model.ErrInvalidPromoLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(ctx, tt.promoCode)

			if tt.expectErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestIntegration_ConcurrentValidation tests that the validator handles concurrent requests correctly.
func TestIntegration_ConcurrentValidation(t *testing.T) {
	logger := zerolog.Nop()

	// Try both relative paths
	possiblePaths := [][]string{
		{
			"data/coupons/couponbase1.gz",
			"data/coupons/couponbase2.gz",
			"data/coupons/couponbase3.gz",
		},
		{
			"../../data/coupons/couponbase1.gz",
			"../../data/coupons/couponbase2.gz",
			"../../data/coupons/couponbase3.gz",
		},
	}

	var validator Validator
	ctx := context.Background()

	for _, paths := range possiblePaths {
		config := &ValidatorConfig{
			FilePaths:     paths,
			MinMatchCount: 2,
		}
		loader := NewFileLoader(logger)

		v, err := NewValidator(ctx, config, loader, logger)
		if err == nil {
			validator = v
			defer validator.Close()
			break
		}
	}

	if validator == nil {
		t.Skipf("Skipping integration test - sample coupon files not found. Run: go run scripts/generate_sample_coupons.go")
		return
	}

	// Run 100 concurrent validations
	const numGoroutines = 100

	type result struct {
		code string
		err  error
	}

	resultChan := make(chan result, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			code := "ALLTHREE1"
			if index%2 == 0 {
				code = "NOTEXIST1"
			}

			err := validator.Validate(ctx, code)
			resultChan <- result{code: code, err: err}
		}(i)
	}

	// Collect results
	validCount := 0
	invalidCount := 0

	for i := 0; i < numGoroutines; i++ {
		res := <-resultChan
		if res.err == nil {
			validCount++
			assert.Equal(t, "ALLTHREE1", res.code)
		} else {
			invalidCount++
			assert.Equal(t, "NOTEXIST1", res.code)
			assert.Equal(t, model.ErrInvalidPromoCode, res.err)
		}
	}

	close(resultChan)

	// Verify results
	assert.Equal(t, 50, validCount, "Expected 50 valid codes")
	assert.Equal(t, 50, invalidCount, "Expected 50 invalid codes")
}
