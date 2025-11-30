package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mini-kart/internal/config"
	"mini-kart/internal/coupon"
	"mini-kart/internal/database"
	"mini-kart/internal/handler"
	"mini-kart/internal/repository"
	"mini-kart/internal/router"
	"mini-kart/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	logger := config.NewLogger(cfg.Logger)
	logger.Info().Msg("starting mini-kart API server")

	// Create context for application lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize database connection pool
	pool, err := database.NewPool(ctx, cfg.Database, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer pool.Close()

	// Initialize repositories
	productRepo := repository.NewProductRepository(pool, logger)
	orderRepo := repository.NewOrderRepository(pool, logger)

	// Initialize coupon loader with S3 and local fallback
	fileLoader := coupon.NewFileLoader(logger)
	var couponLoader coupon.Loader

	if cfg.S3.Enabled {
		// Create S3 loader
		s3Loader, err := coupon.NewS3Loader(ctx, cfg.S3.Bucket, cfg.S3.Region, logger)
		if err != nil {
			logger.Warn().
				Err(err).
				Msg("failed to initialise S3 loader, falling back to local file system only")
			couponLoader = fileLoader
		} else {
			couponLoader = s3Loader
		}
	} else {
		// S3 disabled, use local file system only
		couponLoader = fileLoader
		logger.Info().Msg("using local file system for coupon files (S3 disabled)")
	}

	// Initialize coupon validator
	validatorConfig := coupon.DefaultValidatorConfig()
	validator, err := coupon.NewValidator(ctx, validatorConfig, couponLoader, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize coupon validator: %w", err)
	}
	defer validator.Close()

	// Initialize services
	productService := service.NewProductService(productRepo, logger)
	orderService := service.NewOrderService(orderRepo, productRepo, validator, logger)

	// Initialize HTTP handlers
	productHandler := handler.NewProductHandler(productService, logger)
	orderHandler := handler.NewOrderHandler(orderService, logger)

	// Initialize router
	mux := router.New(productHandler, orderHandler, cfg.Auth.APIKey, logger)

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for errors from the server
	serverErrors := make(chan error, 1)

	// Start HTTP server in a goroutine
	go func() {
		logger.Info().
			Str("address", cfg.Server.Address()).
			Msg("HTTP server started")
		serverErrors <- server.ListenAndServe()
	}()

	// Channel to listen for interrupt signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or an error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		logger.Info().
			Str("signal", sig.String()).
			Msg("shutdown signal received, starting graceful shutdown")

		// Create a context with timeout for shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		// Attempt graceful shutdown
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error().Err(err).Msg("failed to shutdown server gracefully")
			// Force close
			if closeErr := server.Close(); closeErr != nil {
				logger.Error().Err(closeErr).Msg("failed to close server")
			}
			return fmt.Errorf("server shutdown failed: %w", err)
		}

		logger.Info().Msg("server shutdown completed")
	}

	return nil
}
