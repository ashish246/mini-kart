package coupon

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// mockLoader is a mock implementation of the Loader interface for testing.
type mockLoader struct {
	loadFunc func(ctx context.Context, filePath string) (CouponSet, error)
}

func (m *mockLoader) Load(ctx context.Context, filePath string) (CouponSet, error) {
	if m.loadFunc != nil {
		return m.loadFunc(ctx, filePath)
	}
	return nil, errors.New("not implemented")
}

func TestFallbackLoader_S3Success(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	// Create mock S3 loader that succeeds
	s3Set := NewMapCouponSet(10)
	s3Set.(*mapCouponSet).Add("S3CODE123")
	s3Loader := &mockLoader{
		loadFunc: func(ctx context.Context, filePath string) (CouponSet, error) {
			assert.Equal(t, "coupons/test.gz", filePath, "S3 key should have prefix")
			return s3Set, nil
		},
	}

	// Create mock file loader (should not be called)
	fileLoader := &mockLoader{
		loadFunc: func(ctx context.Context, filePath string) (CouponSet, error) {
			t.Error("file loader should not be called when S3 succeeds")
			return nil, errors.New("should not be called")
		},
	}

	// Create fallback loader
	fallback := NewFallbackLoader(s3Loader, fileLoader, "coupons/", true, logger)

	// Load should succeed with S3
	set, err := fallback.Load(ctx, "test.gz")
	assert.NoError(t, err)
	assert.NotNil(t, set)
	assert.True(t, set.Contains("S3CODE123"))
}

func TestFallbackLoader_S3FailsFallsBackToLocal(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	// Create mock S3 loader that fails
	s3Loader := &mockLoader{
		loadFunc: func(ctx context.Context, filePath string) (CouponSet, error) {
			return nil, errors.New("S3 connection failed")
		},
	}

	// Create mock file loader that succeeds
	localSet := NewMapCouponSet(10)
	localSet.(*mapCouponSet).Add("LOCALCODE1")
	fileLoader := &mockLoader{
		loadFunc: func(ctx context.Context, filePath string) (CouponSet, error) {
			assert.Equal(t, "test.gz", filePath, "local file path should not have prefix")
			return localSet, nil
		},
	}

	// Create fallback loader
	fallback := NewFallbackLoader(s3Loader, fileLoader, "coupons/", true, logger)

	// Load should fall back to local
	set, err := fallback.Load(ctx, "test.gz")
	assert.NoError(t, err)
	assert.NotNil(t, set)
	assert.True(t, set.Contains("LOCALCODE1"))
}

func TestFallbackLoader_S3Disabled(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	// Create mock S3 loader (should not be called)
	s3Loader := &mockLoader{
		loadFunc: func(ctx context.Context, filePath string) (CouponSet, error) {
			t.Error("S3 loader should not be called when S3 is disabled")
			return nil, errors.New("should not be called")
		},
	}

	// Create mock file loader that succeeds
	localSet := NewMapCouponSet(10)
	localSet.(*mapCouponSet).Add("LOCALCODE2")
	fileLoader := &mockLoader{
		loadFunc: func(ctx context.Context, filePath string) (CouponSet, error) {
			assert.Equal(t, "test.gz", filePath)
			return localSet, nil
		},
	}

	// Create fallback loader with S3 disabled
	fallback := NewFallbackLoader(s3Loader, fileLoader, "coupons/", false, logger)

	// Load should use local only
	set, err := fallback.Load(ctx, "test.gz")
	assert.NoError(t, err)
	assert.NotNil(t, set)
	assert.True(t, set.Contains("LOCALCODE2"))
}

func TestFallbackLoader_S3LoaderNil(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	// Create mock file loader
	localSet := NewMapCouponSet(10)
	localSet.(*mapCouponSet).Add("LOCALCODE3")
	fileLoader := &mockLoader{
		loadFunc: func(ctx context.Context, filePath string) (CouponSet, error) {
			return localSet, nil
		},
	}

	// Create fallback loader with nil S3 loader
	fallback := NewFallbackLoader(nil, fileLoader, "coupons/", true, logger)

	// Load should use local only
	set, err := fallback.Load(ctx, "test.gz")
	assert.NoError(t, err)
	assert.NotNil(t, set)
	assert.True(t, set.Contains("LOCALCODE3"))
}

func TestFallbackLoader_BothFail(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	// Create mock S3 loader that fails
	s3Loader := &mockLoader{
		loadFunc: func(ctx context.Context, filePath string) (CouponSet, error) {
			return nil, errors.New("S3 error")
		},
	}

	// Create mock file loader that also fails
	fileLoader := &mockLoader{
		loadFunc: func(ctx context.Context, filePath string) (CouponSet, error) {
			return nil, errors.New("file not found")
		},
	}

	// Create fallback loader
	fallback := NewFallbackLoader(s3Loader, fileLoader, "coupons/", true, logger)

	// Load should fail
	set, err := fallback.Load(ctx, "test.gz")
	assert.Error(t, err)
	assert.Nil(t, set)
	assert.Contains(t, err.Error(), "file not found")
}

func TestFallbackLoader_PrefixHandling(t *testing.T) {
	logger := zerolog.Nop()
	ctx := context.Background()

	tests := []struct {
		name       string
		s3Prefix   string
		filePath   string
		expectedS3 string
	}{
		{
			name:       "prefix with trailing slash",
			s3Prefix:   "coupons/",
			filePath:   "file.gz",
			expectedS3: "coupons/file.gz",
		},
		{
			name:       "prefix without trailing slash",
			s3Prefix:   "coupons",
			filePath:   "file.gz",
			expectedS3: "couponsfile.gz",
		},
		{
			name:       "empty prefix",
			s3Prefix:   "",
			filePath:   "file.gz",
			expectedS3: "file.gz",
		},
		{
			name:       "nested prefix",
			s3Prefix:   "data/coupons/prod/",
			filePath:   "file.gz",
			expectedS3: "data/coupons/prod/file.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock S3 loader
			s3Set := NewMapCouponSet(10)
			s3Loader := &mockLoader{
				loadFunc: func(ctx context.Context, filePath string) (CouponSet, error) {
					assert.Equal(t, tt.expectedS3, filePath)
					return s3Set, nil
				},
			}

			fileLoader := &mockLoader{} // Won't be called

			fallback := NewFallbackLoader(s3Loader, fileLoader, tt.s3Prefix, true, logger)
			_, err := fallback.Load(ctx, tt.filePath)
			assert.NoError(t, err)
		})
	}
}
