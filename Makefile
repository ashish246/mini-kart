.PHONY: help build run run-local run-dev test test-unit test-integration test-all test-verbose test-coverage lint format clean docker-up docker-down postgres-start postgres-stop db-reset migrate-up migrate-down generate-coupons test-db-connection test-pg-server install-tools

# Default target
.DEFAULT_GOAL := help

# Database connection string for local development
DB_URL ?= postgres://postgres:postgres@localhost:5432/minikart?sslmode=disable

# Migrations path
MIGRATIONS_PATH = migrations
# API Build Version
VERSION=v1.0.0

# help: Display this help message
help:
	@echo "Available Make targets:"
	@echo ""
	@echo "Development:"
	@echo "  build              Build the application"
	@echo "  run                Run the application (via Docker)"
	@echo "  run-local          Run the application locally (without Docker)"
	@echo "  run-dev            Run the application with go run (loads .env file)"
	@echo "  test               Run unit tests"
	@echo "  test-unit          Run unit tests"
	@echo "  test-integration   Run integration tests"
	@echo "  test-all           Run all tests (unit + integration)"
	@echo "  test-verbose       Run tests with verbose output"
	@echo "  test-coverage      Run tests with coverage report"
	@echo "  lint               Run linter"
	@echo "  format             Format code"
	@echo ""
	@echo "Docker & Database:"
	@echo "  docker-up          Start all Docker services (PostgreSQL + API)"
	@echo "  docker-down        Stop all Docker services"
	@echo "  postgres-start     Start only PostgreSQL database"
	@echo "  postgres-stop      Stop PostgreSQL database"
	@echo "  db-reset           Reset database (drop all tables and recreate)"
	@echo "  migrate-up         Run database migrations (up)"
	@echo "  migrate-down       Rollback database migrations (down)"
	@echo ""
	@echo "Development Utilities:"
	@echo "  generate-coupons   Generate sample coupon files for testing"
	@echo "  test-db-connection Test connection to minikart database"
	@echo "  test-pg-server     Test PostgreSQL server and list databases"
	@echo ""
	@echo "Cleanup:"
	@echo "  clean              Remove build artifacts"
	@echo ""
	@echo "Tools:"
	@echo "  install-tools      Install development tools"

# build: Build the application
build:
	@echo "Building application..."
	@go build -o bin/api-$(VERSION) -ldflags="-s -w -X 'main.version=$(VERSION)' -X 'main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" cmd/api/main.go
	@echo "Build complete: bin/api"

# run: Run the application (via Docker)
run:
	@echo "Starting application via Docker Compose..."
	@docker compose up api

# run-local: Run the application locally (without Docker)
run-local: build
	@echo "Starting application locally..."
	@./bin/api

# run-dev: Run the application using go run with .env file
run-dev:
	@echo "Starting application with go run..."
	@if [ -f .env ]; then \
		echo "Loading environment from .env file..."; \
		export $$(cat .env | grep -v '^#' | xargs) && go run cmd/api/main.go; \
	else \
		echo "Warning: .env file not found. Using default/system environment variables."; \
		go run cmd/api/main.go; \
	fi

# test: Run unit tests
test:
	@echo "Running unit tests..."
	@go test -short ./internal/...

# test-unit: Run unit tests
test-unit:
	@echo "Running unit tests..."
	@go test -short ./internal/...

# test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -v ./test/integration/... -timeout 5m

# test-all: Run all tests (unit + integration)
test-all:
	@echo "Running all tests..."
	@go test -v ./internal/... ./test/integration/... -timeout 5m

# test-verbose: Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	@go test -v ./internal/...

# test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./internal/...
	@go tool cover -func=coverage.out | grep total
	@echo "Coverage report: coverage.out"

# lint: Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run
	@go fmt ./cmd/... ./internal/...

# format: Format code
format:
	@echo "Formatting code..."
	@go fmt ./cmd/... ./internal/...

# docker-up: Start Docker services (PostgreSQL)
docker-up:
	@echo "Starting Docker services..."
	@docker compose up -d
	@echo "Waiting for database to be ready..."
	@sleep 3

# docker-down: Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	@docker compose down

# postgres-start: Start only PostgreSQL database
postgres-start:
	@echo "Starting PostgreSQL database..."
	@docker compose up -d postgres
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 3
	@echo "PostgreSQL is ready"

# postgres-stop: Stop PostgreSQL database
postgres-stop:
	@echo "Stopping PostgreSQL database..."
	@docker compose stop postgres

# migrate-up: Run database migrations (up)
migrate-up:
	@echo "Running database migrations..."
	@migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up
	@echo "Migrations completed successfully"

# migrate-down: Rollback database migrations (down)
migrate-down:
	@echo "Rolling back database migrations..."
	@migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down
	@echo "Migrations rolled back successfully"

# db-reset: Reset database (drop all tables and recreate)
db-reset:
	@echo "Resetting database..."
	@echo "WARNING: This will drop all tables and data!"
	@migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" drop -f || true
	@echo "Running migrations..."
	@migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up
	@echo "Database reset complete"

# db-reset: Reset database (drop all tables and recreate)
db-drop:
	@echo "Dropping database..."
	@echo "WARNING: This will drop all tables and data!"
	@migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" drop -f || true
	@echo "Database drop complete"

# generate-coupons: Generate sample coupon files for testing
generate-coupons:
	@echo "Generating sample coupon files..."
	@go run scripts/generate_sample_coupons.go
	@echo "Sample coupon files generated in data/coupons/"

# test-db-connection: Test connection to minikart database
test-db-connection:
	@echo "Testing connection to minikart database..."
	@go run scripts/test_db_connection.go

# test-pg-server: Test PostgreSQL server and list databases
test-pg-server:
	@echo "Testing PostgreSQL server connection..."
	@go run scripts/test_postgres_db.go

# clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out
	@rm -f internal/*/coverage*.out
	@echo "Clean complete"

# install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed"
