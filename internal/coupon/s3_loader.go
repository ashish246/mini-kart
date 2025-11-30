package coupon

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog"
)

// s3Loader implements Loader for reading gzipped coupon files from AWS S3.
type s3Loader struct {
	client *s3.Client
	bucket string
	logger zerolog.Logger
}

// NewS3Loader creates a new S3-based coupon loader.
func NewS3Loader(ctx context.Context, bucket, region string, logger zerolog.Logger) (Loader, error) {
	logger = logger.With().Str("component", "s3-coupon-loader").Logger()

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		logger.Error().Err(err).Msg("failed to load AWS configuration")
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	logger.Info().
		Str("bucket", bucket).
		Str("region", region).
		Msg("S3 loader initialised")

	return &s3Loader{
		client: client,
		bucket: bucket,
		logger: logger,
	}, nil
}

// Load reads a gzipped coupon file from S3 and returns a CouponSet.
// The key parameter should be the full S3 key (including any prefix).
func (l *s3Loader) Load(ctx context.Context, key string) (CouponSet, error) {
	l.logger.Info().
		Str("bucket", l.bucket).
		Str("key", key).
		Msg("loading coupon file from S3")

	// Get object from S3
	result, err := l.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(l.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		l.logger.Error().
			Err(err).
			Str("bucket", l.bucket).
			Str("key", key).
			Msg("failed to get object from S3")
		return nil, fmt.Errorf("failed to get object from S3 (bucket=%s, key=%s): %w", l.bucket, key, err)
	}
	defer result.Body.Close()

	// Create gzip reader
	gzipReader, err := gzip.NewReader(result.Body)
	if err != nil {
		l.logger.Error().
			Err(err).
			Str("bucket", l.bucket).
			Str("key", key).
			Msg("failed to create gzip reader")
		return nil, fmt.Errorf("failed to create gzip reader for S3 object %s: %w", key, err)
	}
	defer gzipReader.Close()

	// Create coupon set with estimated capacity
	set := NewMapCouponSet(100_000_000).(*mapCouponSet)

	// Read line by line
	scanner := bufio.NewScanner(gzipReader)
	// Set larger buffer for better performance with big files
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	lineCount := 0
	for scanner.Scan() {
		// Check context cancellation periodically
		if lineCount%1_000_000 == 0 {
			select {
			case <-ctx.Done():
				l.logger.Warn().
					Str("bucket", l.bucket).
					Str("key", key).
					Msg("coupon loading cancelled")
				return nil, ctx.Err()
			default:
			}
		}

		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			set.Add(line)
			lineCount++
		}
	}

	if err := scanner.Err(); err != nil {
		l.logger.Error().
			Err(err).
			Str("bucket", l.bucket).
			Str("key", key).
			Msg("error reading coupon file from S3")
		return nil, fmt.Errorf("error reading coupon file from S3 %s: %w", key, err)
	}

	l.logger.Info().
		Str("bucket", l.bucket).
		Str("key", key).
		Int("coupons_loaded", set.Size()).
		Msg("coupon file loaded successfully from S3")

	return set, nil
}

// FallbackLoader implements a loader that tries S3 first, then falls back to local file system.
type fallbackLoader struct {
	s3Loader   Loader
	fileLoader Loader
	s3Prefix   string
	logger     zerolog.Logger
	s3Enabled  bool
}

// NewFallbackLoader creates a loader that tries S3 first, then falls back to local file system.
// If s3Loader is nil, it will only use the file loader.
func NewFallbackLoader(s3Loader, fileLoader Loader, s3Prefix string, s3Enabled bool, logger zerolog.Logger) Loader {
	return &fallbackLoader{
		s3Loader:   s3Loader,
		fileLoader: fileLoader,
		s3Prefix:   s3Prefix,
		s3Enabled:  s3Enabled,
		logger:     logger.With().Str("component", "fallback-loader").Logger(),
	}
}

// Load attempts to load from S3 first, then falls back to local file system.
// For S3, it prepends the s3Prefix to the filePath.
// For local file system, it uses the filePath as-is.
func (l *fallbackLoader) Load(ctx context.Context, filePath string) (CouponSet, error) {
	// Try S3 first if enabled and s3Loader is configured
	if l.s3Enabled && l.s3Loader != nil {
		// Construct S3 key by combining prefix and filepath
		s3Key := l.s3Prefix + filePath

		l.logger.Info().
			Str("s3_key", s3Key).
			Str("local_fallback", filePath).
			Msg("attempting to load from S3")

		set, err := l.s3Loader.Load(ctx, s3Key)
		if err == nil {
			l.logger.Info().
				Str("s3_key", s3Key).
				Msg("successfully loaded from S3")
			return set, nil
		}

		l.logger.Warn().
			Err(err).
			Str("s3_key", s3Key).
			Msg("failed to load from S3, falling back to local file system")
	} else {
		l.logger.Debug().
			Bool("s3_enabled", l.s3Enabled).
			Bool("has_s3_loader", l.s3Loader != nil).
			Msg("S3 disabled or not configured, using local file system")
	}

	// Fall back to local file system
	l.logger.Info().
		Str("file_path", filePath).
		Msg("loading from local file system")

	return l.fileLoader.Load(ctx, filePath)
}
