package coupon

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

// fileLoader implements Loader for reading gzipped coupon files.
type fileLoader struct {
	logger zerolog.Logger
}

// NewFileLoader creates a new file-based coupon loader.
func NewFileLoader(logger zerolog.Logger) Loader {
	return &fileLoader{
		logger: logger.With().Str("component", "coupon-loader").Logger(),
	}
}

// Load reads a gzipped coupon file and returns a CouponSet.
// The file is expected to contain one coupon code per line.
func (l *fileLoader) Load(ctx context.Context, filePath string) (CouponSet, error) {
	l.logger.Info().Str("file", filePath).Msg("loading coupon file")

	// Open the gzipped file
	file, err := os.Open(filePath)
	if err != nil {
		l.logger.Error().Err(err).Str("file", filePath).Msg("failed to open coupon file")
		return nil, fmt.Errorf("failed to open coupon file %s: %w", filePath, err)
	}
	defer file.Close()

	// Create gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		l.logger.Error().Err(err).Str("file", filePath).Msg("failed to create gzip reader")
		return nil, fmt.Errorf("failed to create gzip reader for %s: %w", filePath, err)
	}
	defer gzipReader.Close()

	// Create coupon set with estimated capacity
	// For a 1GB file with 100M codes, pre-allocate to reduce reallocations
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
				l.logger.Warn().Str("file", filePath).Msg("coupon loading cancelled")
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
		l.logger.Error().Err(err).Str("file", filePath).Msg("error reading coupon file")
		return nil, fmt.Errorf("error reading coupon file %s: %w", filePath, err)
	}

	l.logger.Info().
		Str("file", filePath).
		Int("coupons_loaded", set.Size()).
		Msg("coupon file loaded successfully")

	return set, nil
}
