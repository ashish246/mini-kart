package main

import (
	"compress/gzip"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// generateSampleCoupons creates sample coupon files for testing.
// File 1: Contains codes A, B, C, X, Y
// File 2: Contains codes B, C, D, X, Z
// File 3: Contains codes C, D, E, Y, Z
// Valid codes (in at least 2 files): B, C, D, X, Y, Z
// Invalid codes (in only 1 file): A, E
func main() {
	dataDir := "data/coupons"

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create directory: %v", err)
	}

	// Sample coupon sets
	coupons := map[string][]string{
		"couponbase1.gz": {
			"VALIDONE1",  // In file 1 and 2
			"VALIDTWO12", // In file 1 and 2
			"ALLTHREE1",  // In all 3 files
			"ONLYONE111", // Only in file 1
			"SUMMER2024", // In file 1 and 3
		},
		"couponbase2.gz": {
			"VALIDONE1",  // In file 1 and 2
			"VALIDTWO12", // In file 1 and 2
			"ALLTHREE1",  // In all 3 files
			"ONLYTWO222", // Only in file 2
			"WINTER2024", // In file 2 and 3
		},
		"couponbase3.gz": {
			"WINTER2024",  // In file 2 and 3
			"SUMMER2024",  // In file 1 and 3
			"ALLTHREE1",   // In all 3 files
			"ONLYTHREE3",  // Only in file 3
			"SPRING2024",  // In file 3 only
		},
	}

	for filename, codes := range coupons {
		filePath := filepath.Join(dataDir, filename)

		if err := createCouponFile(filePath, codes); err != nil {
			log.Fatalf("Failed to create %s: %v", filename, err)
		}

		fmt.Printf("Created %s with %d codes\n", filePath, len(codes))
	}

	fmt.Println("\nSample coupon files created successfully!")
	fmt.Println("\nValid codes (appear in at least 2 files):")
	fmt.Println("  - VALIDONE1  (files 1, 2)")
	fmt.Println("  - VALIDTWO12 (files 1, 2)")
	fmt.Println("  - ALLTHREE1  (files 1, 2, 3)")
	fmt.Println("  - SUMMER2024 (files 1, 3)")
	fmt.Println("  - WINTER2024 (files 2, 3)")
	fmt.Println("\nInvalid codes (appear in only 1 file):")
	fmt.Println("  - ONLYONE111 (file 1 only)")
	fmt.Println("  - ONLYTWO222 (file 2 only)")
	fmt.Println("  - ONLYTHREE3 (file 3 only)")
	fmt.Println("  - SPRING2024 (file 3 only)")
}

func createCouponFile(filePath string, coupons []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	for _, coupon := range coupons {
		if _, err := fmt.Fprintf(gzipWriter, "%s\n", coupon); err != nil {
			return fmt.Errorf("failed to write coupon: %w", err)
		}
	}

	return nil
}
