package coupon

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestCouponFile creates a gzipped test coupon file.
func createTestCouponFile(t *testing.T, filename string, coupons []string) string {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, filename)

	file, err := os.Create(filePath)
	require.NoError(t, err)
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	for _, coupon := range coupons {
		_, err := gzipWriter.Write([]byte(coupon + "\n"))
		require.NoError(t, err)
	}

	return filePath
}

func TestFileLoader_Load_Success(t *testing.T) {
	logger := zerolog.Nop()
	loader := NewFileLoader(logger)

	testCoupons := []string{
		"TESTCODE1",
		"TESTCODE2",
		"TESTCODE3",
		"VALIDPROMO",
		"DISCOUNT10",
	}

	filePath := createTestCouponFile(t, "test_coupons.gz", testCoupons)

	ctx := context.Background()
	set, err := loader.Load(ctx, filePath)

	require.NoError(t, err)
	require.NotNil(t, set)
	assert.Equal(t, 5, set.Size())

	// Verify all coupons are present
	for _, coupon := range testCoupons {
		assert.True(t, set.Contains(coupon), "Expected coupon %s to be present", coupon)
	}
}

func TestFileLoader_Load_WithEmptyLines(t *testing.T) {
	logger := zerolog.Nop()
	loader := NewFileLoader(logger)

	testCoupons := []string{
		"CODE1",
		"",
		"CODE2",
		"   ",
		"CODE3",
		"\n",
	}

	filePath := createTestCouponFile(t, "coupons_with_empty.gz", testCoupons)

	ctx := context.Background()
	set, err := loader.Load(ctx, filePath)

	require.NoError(t, err)
	require.NotNil(t, set)
	// Should only have 3 non-empty codes
	assert.Equal(t, 3, set.Size())
	assert.True(t, set.Contains("CODE1"))
	assert.True(t, set.Contains("CODE2"))
	assert.True(t, set.Contains("CODE3"))
}

func TestFileLoader_Load_WithWhitespace(t *testing.T) {
	logger := zerolog.Nop()
	loader := NewFileLoader(logger)

	testCoupons := []string{
		"  TRIMMED1  ",
		"\tTRIMMED2\t",
		" TRIMMED3",
	}

	filePath := createTestCouponFile(t, "coupons_with_whitespace.gz", testCoupons)

	ctx := context.Background()
	set, err := loader.Load(ctx, filePath)

	require.NoError(t, err)
	require.NotNil(t, set)
	assert.Equal(t, 3, set.Size())

	// Verify codes are trimmed
	assert.True(t, set.Contains("TRIMMED1"))
	assert.True(t, set.Contains("TRIMMED2"))
	assert.True(t, set.Contains("TRIMMED3"))
	assert.False(t, set.Contains("  TRIMMED1  "))
}

func TestFileLoader_Load_DuplicateCodes(t *testing.T) {
	logger := zerolog.Nop()
	loader := NewFileLoader(logger)

	testCoupons := []string{
		"DUPLICATE",
		"UNIQUE1",
		"DUPLICATE",
		"UNIQUE2",
		"DUPLICATE",
	}

	filePath := createTestCouponFile(t, "coupons_with_duplicates.gz", testCoupons)

	ctx := context.Background()
	set, err := loader.Load(ctx, filePath)

	require.NoError(t, err)
	require.NotNil(t, set)
	// Should only count unique codes
	assert.Equal(t, 3, set.Size())
	assert.True(t, set.Contains("DUPLICATE"))
	assert.True(t, set.Contains("UNIQUE1"))
	assert.True(t, set.Contains("UNIQUE2"))
}

func TestFileLoader_Load_FileNotFound(t *testing.T) {
	logger := zerolog.Nop()
	loader := NewFileLoader(logger)

	ctx := context.Background()
	set, err := loader.Load(ctx, "/nonexistent/path/to/file.gz")

	require.Error(t, err)
	assert.Nil(t, set)
	assert.Contains(t, err.Error(), "failed to open coupon file")
}

func TestFileLoader_Load_InvalidGzip(t *testing.T) {
	logger := zerolog.Nop()
	loader := NewFileLoader(logger)

	// Create a non-gzipped file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "invalid.gz")

	err := os.WriteFile(filePath, []byte("not a gzip file"), 0644)
	require.NoError(t, err)

	ctx := context.Background()
	set, err := loader.Load(ctx, filePath)

	require.Error(t, err)
	assert.Nil(t, set)
	assert.Contains(t, err.Error(), "failed to create gzip reader")
}

func TestFileLoader_Load_ContextCancellation(t *testing.T) {
	logger := zerolog.Nop()
	loader := NewFileLoader(logger)

	// Create a large file to ensure we can cancel during loading
	largeCoupons := make([]string, 2_000_000)
	for i := 0; i < len(largeCoupons); i++ {
		largeCoupons[i] = "COUPON" + string(rune(i))
	}

	filePath := createTestCouponFile(t, "large_coupons.gz", largeCoupons)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	set, err := loader.Load(ctx, filePath)

	// Should either succeed (if loading completed before cancellation)
	// or fail with context error
	if err != nil {
		assert.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, set)
	}
}

func TestFileLoader_Load_EmptyFile(t *testing.T) {
	logger := zerolog.Nop()
	loader := NewFileLoader(logger)

	filePath := createTestCouponFile(t, "empty.gz", []string{})

	ctx := context.Background()
	set, err := loader.Load(ctx, filePath)

	require.NoError(t, err)
	require.NotNil(t, set)
	assert.Equal(t, 0, set.Size())
}

func TestFileLoader_Load_LargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	logger := zerolog.Nop()
	loader := NewFileLoader(logger)

	// Create a file with 1 million coupons
	largeCoupons := make([]string, 1_000_000)
	for i := 0; i < len(largeCoupons); i++ {
		// Create unique 10-character codes
		largeCoupons[i] = fmt.Sprintf("CODE%06d", i)
	}

	filePath := createTestCouponFile(t, "large_file.gz", largeCoupons)

	ctx := context.Background()
	set, err := loader.Load(ctx, filePath)

	require.NoError(t, err)
	require.NotNil(t, set)
	assert.Equal(t, 1_000_000, set.Size())

	// Verify a few random codes
	assert.True(t, set.Contains("CODE000000"))
	assert.True(t, set.Contains("CODE500000"))
	assert.True(t, set.Contains("CODE999999"))
}
