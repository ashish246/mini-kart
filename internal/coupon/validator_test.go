package coupon

import (
	"context"
	"testing"

	"mini-kart/internal/model"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultValidatorConfig(t *testing.T) {
	config := DefaultValidatorConfig()

	require.NotNil(t, config)
	assert.Equal(t, 3, len(config.FilePaths))
	assert.Equal(t, 2, config.MinMatchCount)
	assert.Equal(t, "data/coupons/couponbase1.gz", config.FilePaths[0])
	assert.Equal(t, "data/coupons/couponbase2.gz", config.FilePaths[1])
	assert.Equal(t, "data/coupons/couponbase3.gz", config.FilePaths[2])
}

func TestNewValidator_Success(t *testing.T) {
	logger := zerolog.Nop()

	// Create test coupon files
	file1 := createTestCouponFile(t, "coupon1.gz", []string{"VALIDCODE1", "VALIDCODE2", "COMMON123"})
	file2 := createTestCouponFile(t, "coupon2.gz", []string{"VALIDCODE2", "VALIDCODE3", "COMMON123"})
	file3 := createTestCouponFile(t, "coupon3.gz", []string{"VALIDCODE3", "VALIDCODE4"})

	config := &ValidatorConfig{
		FilePaths:     []string{file1, file2, file3},
		MinMatchCount: 2,
	}

	loader := NewFileLoader(logger)
	ctx := context.Background()

	validator, err := NewValidator(ctx, config, loader, logger)

	require.NoError(t, err)
	require.NotNil(t, validator)

	// Cleanup
	err = validator.Close()
	assert.NoError(t, err)
}

func TestNewValidator_FileLoadError(t *testing.T) {
	logger := zerolog.Nop()

	config := &ValidatorConfig{
		FilePaths:     []string{"/nonexistent/file1.gz", "/nonexistent/file2.gz"},
		MinMatchCount: 2,
	}

	loader := NewFileLoader(logger)
	ctx := context.Background()

	validator, err := NewValidator(ctx, config, loader, logger)

	require.Error(t, err)
	assert.Nil(t, validator)
	assert.Contains(t, err.Error(), "failed to load coupon file")
}

func TestValidator_Validate_ValidCode(t *testing.T) {
	logger := zerolog.Nop()

	// Create test coupon files with overlapping codes
	file1 := createTestCouponFile(t, "coupon1.gz", []string{
		"VALIDCODE1",
		"COMMON1234",
		"TESTPROMO1",
	})
	file2 := createTestCouponFile(t, "coupon2.gz", []string{
		"VALIDCODE2",
		"COMMON1234", // Appears in file1 and file2
		"TESTPROMO2",
	})
	file3 := createTestCouponFile(t, "coupon3.gz", []string{
		"VALIDCODE3",
		"TESTPROMO3",
	})

	config := &ValidatorConfig{
		FilePaths:     []string{file1, file2, file3},
		MinMatchCount: 2,
	}

	loader := NewFileLoader(logger)
	ctx := context.Background()

	validator, err := NewValidator(ctx, config, loader, logger)
	require.NoError(t, err)
	defer validator.Close()

	tests := []struct {
		name      string
		promoCode string
		expectErr error
	}{
		{
			name:      "Valid code in 2 files",
			promoCode: "COMMON1234",
			expectErr: nil,
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

func TestValidator_Validate_InvalidLength(t *testing.T) {
	logger := zerolog.Nop()

	file1 := createTestCouponFile(t, "coupon1.gz", []string{"VALIDCODE1"})
	file2 := createTestCouponFile(t, "coupon2.gz", []string{"VALIDCODE1"})
	file3 := createTestCouponFile(t, "coupon3.gz", []string{"VALIDCODE1"})

	config := &ValidatorConfig{
		FilePaths:     []string{file1, file2, file3},
		MinMatchCount: 2,
	}

	loader := NewFileLoader(logger)
	ctx := context.Background()

	validator, err := NewValidator(ctx, config, loader, logger)
	require.NoError(t, err)
	defer validator.Close()

	tests := []struct {
		name      string
		promoCode string
	}{
		{
			name:      "Too short - 7 characters",
			promoCode: "SHORT12",
		},
		{
			name:      "Too short - 1 character",
			promoCode: "A",
		},
		{
			name:      "Too long - 11 characters",
			promoCode: "TOOLONGCODE",
		},
		{
			name:      "Too long - 20 characters",
			promoCode: "WAYTOOLONGPROMOCODE1",
		},
		{
			name:      "Empty string",
			promoCode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(ctx, tt.promoCode)

			require.Error(t, err)
			assert.Equal(t, model.ErrInvalidPromoLength, err)
		})
	}
}

func TestValidator_Validate_ValidLength(t *testing.T) {
	logger := zerolog.Nop()

	// Create test files where these codes appear in at least 2 files
	file1 := createTestCouponFile(t, "coupon1.gz", []string{
		"EIGHTCHR",
		"NINECHARS",
		"TENCHARS10",
	})
	file2 := createTestCouponFile(t, "coupon2.gz", []string{
		"EIGHTCHR",
		"NINECHARS",
		"TENCHARS10",
	})
	file3 := createTestCouponFile(t, "coupon3.gz", []string{
		"OTHERCODE",
	})

	config := &ValidatorConfig{
		FilePaths:     []string{file1, file2, file3},
		MinMatchCount: 2,
	}

	loader := NewFileLoader(logger)
	ctx := context.Background()

	validator, err := NewValidator(ctx, config, loader, logger)
	require.NoError(t, err)
	defer validator.Close()

	tests := []struct {
		name      string
		promoCode string
	}{
		{
			name:      "8 characters",
			promoCode: "EIGHTCHR",
		},
		{
			name:      "9 characters",
			promoCode: "NINECHARS",
		},
		{
			name:      "10 characters",
			promoCode: "TENCHARS10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(ctx, tt.promoCode)

			require.NoError(t, err)
		})
	}
}

func TestValidator_Validate_InsufficientMatches(t *testing.T) {
	logger := zerolog.Nop()

	file1 := createTestCouponFile(t, "coupon1.gz", []string{
		"ONLYINONE",
		"VALIDCODE1",
	})
	file2 := createTestCouponFile(t, "coupon2.gz", []string{
		"VALIDCODE1",
		"DIFFERENT1",
	})
	file3 := createTestCouponFile(t, "coupon3.gz", []string{
		"DIFFERENT2",
	})

	config := &ValidatorConfig{
		FilePaths:     []string{file1, file2, file3},
		MinMatchCount: 2,
	}

	loader := NewFileLoader(logger)
	ctx := context.Background()

	validator, err := NewValidator(ctx, config, loader, logger)
	require.NoError(t, err)
	defer validator.Close()

	tests := []struct {
		name      string
		promoCode string
	}{
		{
			name:      "Code in only one file",
			promoCode: "ONLYINONE",
		},
		{
			name:      "Code not in any file",
			promoCode: "NOTEXIST1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(ctx, tt.promoCode)

			require.Error(t, err)
			assert.Equal(t, model.ErrInvalidPromoCode, err)
		})
	}
}

func TestValidator_Validate_AllThreeFiles(t *testing.T) {
	logger := zerolog.Nop()

	// Code appears in all 3 files
	file1 := createTestCouponFile(t, "coupon1.gz", []string{"EVERYWHERE"})
	file2 := createTestCouponFile(t, "coupon2.gz", []string{"EVERYWHERE"})
	file3 := createTestCouponFile(t, "coupon3.gz", []string{"EVERYWHERE"})

	config := &ValidatorConfig{
		FilePaths:     []string{file1, file2, file3},
		MinMatchCount: 2,
	}

	loader := NewFileLoader(logger)
	ctx := context.Background()

	validator, err := NewValidator(ctx, config, loader, logger)
	require.NoError(t, err)
	defer validator.Close()

	err = validator.Validate(ctx, "EVERYWHERE")
	require.NoError(t, err)
}

func TestValidator_Validate_ExactlyTwoFiles(t *testing.T) {
	logger := zerolog.Nop()

	// Code appears in exactly 2 files
	file1 := createTestCouponFile(t, "coupon1.gz", []string{"INTWOFILES"})
	file2 := createTestCouponFile(t, "coupon2.gz", []string{"INTWOFILES"})
	file3 := createTestCouponFile(t, "coupon3.gz", []string{"OTHERCODE"})

	config := &ValidatorConfig{
		FilePaths:     []string{file1, file2, file3},
		MinMatchCount: 2,
	}

	loader := NewFileLoader(logger)
	ctx := context.Background()

	validator, err := NewValidator(ctx, config, loader, logger)
	require.NoError(t, err)
	defer validator.Close()

	err = validator.Validate(ctx, "INTWOFILES")
	require.NoError(t, err)
}

func TestValidator_Validate_CaseSensitive(t *testing.T) {
	logger := zerolog.Nop()

	file1 := createTestCouponFile(t, "coupon1.gz", []string{"UPPERCASE1"})
	file2 := createTestCouponFile(t, "coupon2.gz", []string{"UPPERCASE1"})
	file3 := createTestCouponFile(t, "coupon3.gz", []string{"OTHERCODE"})

	config := &ValidatorConfig{
		FilePaths:     []string{file1, file2, file3},
		MinMatchCount: 2,
	}

	loader := NewFileLoader(logger)
	ctx := context.Background()

	validator, err := NewValidator(ctx, config, loader, logger)
	require.NoError(t, err)
	defer validator.Close()

	// Exact match should work
	err = validator.Validate(ctx, "UPPERCASE1")
	require.NoError(t, err)

	// Different case should fail
	err = validator.Validate(ctx, "uppercase1")
	require.Error(t, err)
	assert.Equal(t, model.ErrInvalidPromoCode, err)
}

func TestValidator_Close(t *testing.T) {
	logger := zerolog.Nop()

	file1 := createTestCouponFile(t, "coupon1.gz", []string{"CODE1"})
	file2 := createTestCouponFile(t, "coupon2.gz", []string{"CODE2"})
	file3 := createTestCouponFile(t, "coupon3.gz", []string{"CODE3"})

	config := &ValidatorConfig{
		FilePaths:     []string{file1, file2, file3},
		MinMatchCount: 2,
	}

	loader := NewFileLoader(logger)
	ctx := context.Background()

	validator, err := NewValidator(ctx, config, loader, logger)
	require.NoError(t, err)

	err = validator.Close()
	assert.NoError(t, err)
}
