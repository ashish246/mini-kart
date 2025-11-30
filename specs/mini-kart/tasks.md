# Mini-Kart API Server Implementation Plan

## Overview

This plan guides the implementation of a Go-based food ordering API server that implements the OpenAPI 3.1 specification with concurrent promo code validation. The system validates promo codes by streaming three large compressed files using go-routines and channels, stores orders in PostgreSQL, and provides observability through structured logging and Prometheus metrics.

## Current State

**Starting Point:**
- Greenfield Go project with no existing code
- Comprehensive requirements documented (192 acceptance criteria across 17 sections in `specs/mini-kart/requirements.md`)
- Detailed architecture and design documented in `specs/mini-kart/design.md`
- Decision log maintained in `specs/mini-kart/decision_log.md`

**Technical Foundation:**
- Go 1.21+ will be used (verify latest stable version)
- PostgreSQL 15+ for data persistence
- pgx/v5 driver for high-performance database access
- Clean architecture: controller → service → repository → model
- Concurrent promo validation using go-routines, channels, and context cancellation

## Requirements

**Functional Requirements:**

1. The system MUST implement all API endpoints defined in the OpenAPI specification:
   - `GET /products` - List all products with pagination
   - `POST /orders` - Create new order with promo code validation
   - `GET /orders/{id}` - Retrieve order details
   - `GET /health` - Health check with connection pool monitoring

2. The system MUST validate promo codes using concurrent file processing:
   - Stream three gzipped files (couponbase1.gz, couponbase2.gz, couponbase3.gz) without loading entire files into memory
   - Use separate go-routines for each file
   - Use channels to communicate results between go-routines
   - Cancel remaining go-routines when promo code found in 2+ files
   - Complete validation within 5 seconds (timeout) with target P95 of 500ms

3. The system MUST validate promo codes according to business rules:
   - Length between 8-10 characters (inclusive)
   - Found in at least 2 of 3 coupon files
   - Case-insensitive matching
   - Store validated promo code with order (no discount applied)

4. The system MUST validate orders with proper error handling:
   - All product IDs exist in database (HTTP 422 if invalid)
   - Request body conforms to schema (HTTP 400 if malformed)
   - Atomic transaction for order + order_items creation
   - Promo validation occurs before transaction begins

5. The system MUST implement authentication middleware:
   - Require `X-API-Key` header for protected endpoints
   - API key configurable via environment variable
   - Return HTTP 401 for missing/invalid API key
   - Skip authentication for `/health` endpoint

6. The system MUST provide observability:
   - Structured JSON logging using zerolog
   - Correlation ID for request tracing
   - Prometheus metrics for response times, request counts, errors
   - Health check includes database pool stats and coupon file accessibility

**Technical Constraints:**

1. The implementation MUST use Go 1.21+ features (verify latest stable version during setup)
2. The implementation MUST NOT use deprecated packages (e.g., `ioutil`)
3. The implementation MUST follow clean architecture patterns:
   - No business logic in controllers
   - No HTTP concerns in services
   - Repository interface for database abstraction
4. The implementation MUST use pgx/v5 connection pool (not database/sql)
5. The implementation MUST use context for timeout and cancellation throughout
6. The implementation MUST use atomic operations for thread-safe concurrent access
7. Database migrations MUST be versioned and reversible using golang-migrate
8. All timestamps MUST be stored in UTC with timezone awareness
9. Response times MUST target P95 < 500ms for happy path
10. The implementation MUST validate all user input before processing

**Exclusions:**

1. The implementation MUST NOT implement discount calculation for promo codes (store only)
2. The implementation MUST NOT implement user authentication (only API key auth)
3. The implementation MUST NOT implement promo code usage tracking or limits (documented as outstanding business question)
4. The implementation MUST NOT build Windows support

**Prerequisites:**

1. Go 1.21+ MUST be installed
2. PostgreSQL 15+ MUST be available (local or Docker)
3. Coupon files (couponbase1.gz, couponbase2.gz, couponbase3.gz) MUST be downloaded to `/data/coupons/` directory
4. Docker MUST be available for testcontainers integration tests

## Assumptions

**Development Environment:**
- Makefile will be used as single entry point for common operations (build, test, lint, run)
- Development database will be provided via Docker Compose
- `.env.example` will document all required environment variables

**Testing Approach:**
- Unit tests will mock database and file I/O
- Integration tests will use testcontainers-go for real PostgreSQL
- Concurrent validation tests will use small test fixtures (not full 5GB files)
- Test execution must be fast (<30 seconds for full suite excluding integration tests)

**Outstanding Business Questions (documented, not blocking implementation):**
- Promo code usage limits and tracking requirements (Section 8.1 of design.md)
- Connection pool sizing for production load (provisional: 25 connections)
- Rate limiting requirements for API endpoints

## Success Criteria

1. ✅ All API endpoints respond according to OpenAPI specification
2. ✅ Promo code validation completes within 5 seconds (P95 target: 500ms)
3. ✅ Concurrent promo validation correctly cancels remaining go-routines when code found in 2+ files
4. ✅ All 192 requirements in `requirements.md` are satisfied
5. ✅ Database transactions are atomic (order + order_items created together or not at all)
6. ✅ Health check returns connection pool stats and verifies coupon file accessibility
7. ✅ All endpoints require authentication except `/health`
8. ✅ Test coverage >80% for all non-trivial code
9. ✅ All tests pass locally without external service dependencies (except testcontainers)
10. ✅ Linting passes with zero errors or warnings
11. ✅ Application builds successfully with no errors or warnings
12. ✅ Docker image builds successfully and runs application
13. ✅ Makefile provides: lint, format, test, build, run targets
14. ✅ Structured JSON logs include correlation IDs for request tracing
15. ✅ Prometheus metrics exposed on `/metrics` endpoint

---

## Development Plan

### Phase 1: Project Foundation & Database Schema

**Goal:** Establish project structure, dependencies, and database schema with migrations.

- [ ] Initialise Go module with latest stable Go version (verify with `go version`)
  - [ ] Run `go mod init mini-kart` in project root
  - [ ] Verify Go 1.21+ is being used
- [ ] Configure Git LFS for large coupon files:
  - [ ] Initialise Git LFS: `git lfs install`
  - [ ] Create `.gitattributes` file to track compressed coupon files: `*.gz filter=lfs diff=lfs merge=lfs -text`
  - [ ] Track data/coupons directory pattern specifically if needed
  - [ ] Verify Git LFS is configured: `git lfs track`
- [ ] Create clean architecture directory structure following design.md:
  - [ ] `cmd/api/` - Application entry point
  - [ ] `internal/config/` - Configuration management
  - [ ] `internal/model/` - Domain models
  - [ ] `internal/repository/` - Data access interfaces and implementations
  - [ ] `internal/service/` - Business logic
  - [ ] `internal/handler/` - HTTP handlers
  - [ ] `internal/middleware/` - HTTP middleware (recovery, correlation ID, auth, logger)
  - [ ] `internal/coupon/` - Promotional code validation package
  - [ ] `migrations/` - Database migration files
  - [ ] `test/` - Integration and E2E tests
- [ ] Add core dependencies with latest stable versions:
  - [ ] `github.com/jackc/pgx/v5` - PostgreSQL driver
  - [ ] `github.com/rs/zerolog` - Structured logging
  - [ ] `github.com/golang-migrate/migrate/v4` - Database migrations
  - [ ] `github.com/prometheus/client_golang` - Prometheus metrics
  - [ ] `github.com/testcontainers/testcontainers-go` - Integration testing (dev dependency)
  - [ ] Run `go mod tidy` to clean up dependencies
- [ ] Create database migration files in `migrations/` directory:
  - [ ] `000001_create_products_table.up.sql` and `.down.sql` (columns: id TEXT PRIMARY KEY, name TEXT, price DECIMAL(10,2), category TEXT, created_at TIMESTAMPTZ)
  - [ ] `000002_create_orders_table.up.sql` and `.down.sql` (columns: id UUID PRIMARY KEY, promo_code TEXT, total_price DECIMAL(10,2), created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)
  - [ ] `000003_create_order_items_table.up.sql` and `.down.sql` (columns: id UUID PRIMARY KEY, order_id UUID REFERENCES orders(id), product_id TEXT REFERENCES products(id), quantity INTEGER, price DECIMAL(10,2))
  - [ ] `000004_add_indexes.up.sql` and `.down.sql` (indexes on orders.created_at, order_items.order_id)
- [ ] Create seed data migration for products table:
  - [ ] `000005_seed_products.up.sql` with sample food products across all categories from requirements.md
- [ ] Create `.env.example` file documenting all required environment variables:
  - [ ] DATABASE_URL, API_KEY, SERVER_PORT, LOG_LEVEL, COUPON_FILE_PATH_1/2/3
- [ ] Create basic `Makefile` with initial targets:
  - [ ] `migrate-up` - Run database migrations
  - [ ] `migrate-down` - Rollback migrations
  - [ ] `build` - Build the application
  - [ ] `test` - Run tests
  - [ ] `lint` - Run golangci-lint
- [ ] Create `docker-compose.yml` for local PostgreSQL development database
- [ ] Test database migrations run successfully:
  - [ ] Start PostgreSQL via Docker Compose
  - [ ] Run `make migrate-up` and verify all migrations apply
  - [ ] Run `make migrate-down` and verify rollback works
  - [ ] Run `make migrate-up` again to restore schema
- [ ] Perform a critical self-review of your changes and fix any issues found
- [ ] STOP and wait for human review

### Phase 2: Core Models & Repository Layer

**Goal:** Implement domain models and repository pattern for database access.

- [ ] Implement domain models in `internal/model/`:
  - [ ] `product.go` - Product model (ID, Name, Price, Category, CreatedAt)
  - [ ] `order.go` - Order model (ID, PromoCode, TotalPrice, Items, CreatedAt, UpdatedAt)
  - [ ] `order_item.go` - OrderItem model (ID, OrderID, ProductID, Quantity, Price)
- [ ] Implement repository interfaces in `internal/repository/`:
  - [ ] `product_repository.go` - Interface: GetAll(ctx, limit, offset), GetByIDs(ctx, ids), ValidateProductsExist(ctx, ids)
  - [ ] `order_repository.go` - Interface: Create(ctx, order), GetByID(ctx, id)
- [ ] Implement PostgreSQL repository implementations:
  - [ ] `product_repository_postgres.go` - Implement ProductRepository using pgx.Pool
  - [ ] `order_repository_postgres.go` - Implement OrderRepository with transaction support
  - [ ] Use pgx named parameters and proper error handling
  - [ ] Use context for timeout control on all queries
- [ ] Create database connection pool manager in `internal/repository/`:
  - [ ] `db.go` - NewPool(ctx, connString) function returning *pgx.Pool
  - [ ] Configure pool with 25 max connections, 5 min connections, 30-minute max lifetime
  - [ ] Add pool.Ping(ctx) verification on initialization
- [ ] Write unit tests for repositories in `internal/repository/`:
  - [ ] Mock pgx.Pool using interfaces for unit tests
  - [ ] Test happy paths for all repository methods
  - [ ] Test error handling (connection errors, not found, constraint violations)
  - [ ] Test pagination logic for GetAll
- [ ] Run repository tests and verify they pass:
  - [ ] Execute `make test` for repository package
  - [ ] Verify test coverage >80% for repository implementations
- [ ] Perform a critical self-review of your changes and fix any issues found
- [ ] STOP and wait for human review

### Phase 3: Concurrent Promo Code Validation System

**Goal:** Implement streaming file reader and concurrent promo validation with go-routines, channels, and context cancellation.

- [ ] Implement streaming gzip file reader in `internal/coupon/`:
  - [ ] `loader.go` - File loading functionality for compressed coupon files
  - [ ] Use bufio.Scanner to read line-by-line without loading entire file
  - [ ] Use gzip.NewReader for decompression
  - [ ] Load coupon codes into memory-efficient set structure
  - [ ] Handle file I/O errors appropriately
- [ ] Implement concurrent promo validator in `internal/coupon/`:
  - [ ] `validator.go` - Validate(ctx, promoCode) function
  - [ ] Pre-validate promo code length (8-10 characters), return early if invalid
  - [ ] Convert promo code to uppercase for case-insensitive matching
  - [ ] Check against pre-loaded coupon sets from multiple files
  - [ ] Use concurrent-safe set implementation
  - [ ] Track matches across multiple coupon files
  - [ ] Return true if valid (found in 2+ files), false if invalid
- [ ] Implement coupon set data structure in `internal/coupon/`:
  - [ ] `set.go` - Thread-safe set implementation for coupon codes
  - [ ] Support concurrent read operations
  - [ ] Efficient membership testing
- [ ] Create test fixtures for promo validation tests in `test/fixtures/`:
  - [ ] Create small gzipped test files (1000 lines each)
  - [ ] Include known valid codes (present in 2+ files)
  - [ ] Include known invalid codes (present in 0-1 files)
- [ ] Write comprehensive tests for promo validation in `internal/coupon/`:
  - [ ] Test valid promo codes (found in 2+ files)
  - [ ] Test invalid promo codes (found in 0-1 files)
  - [ ] Test case-insensitive matching (HAPPYHRS, happyhrs, HappyHrs)
  - [ ] Test length validation (7 chars, 8 chars, 10 chars, 11 chars)
  - [ ] Test concurrent validation calls (race detector enabled with `go test -race`)
  - [ ] Test error handling (missing files, corrupted gzip)
  - [ ] Test coupon set operations
- [ ] Run promo validation tests with race detector:
  - [ ] Execute `go test -race ./internal/coupon/...`
  - [ ] Verify all tests pass with no race conditions detected
  - [ ] Verify test coverage >85% for coupon package
- [ ] Perform a critical self-review of your changes and fix any issues found
- [ ] STOP and wait for human review

### Phase 4: Business Logic Layer (Services)

**Goal:** Implement service layer with business logic and transaction management.

- [ ] Implement ProductService in `internal/service/`:
  - [ ] `product_service.go` - NewProductService(repo ProductRepository)
  - [ ] ListProducts(ctx, limit, offset) - Paginated product listing with validation
  - [ ] ValidateProductIDs(ctx, productIDs) - Verify all product IDs exist
  - [ ] GetProductsByIDs(ctx, productIDs) - Retrieve full product details
  - [ ] Add input validation for pagination (limit 1-100, offset ≥ 0)
- [ ] Implement OrderService in `internal/service/`:
  - [ ] `order_service.go` - NewOrderService(orderRepo, productRepo, promoValidator)
  - [ ] CreateOrder(ctx, request) - Full order creation flow:
    - Validate request body (product IDs, quantities, promo code if provided)
    - If promo code provided, validate using promo validator (5s timeout)
    - Begin validation phase: call ValidateProductIDs to ensure all products exist
    - Calculate total price based on product prices × quantities
    - Begin database transaction using pgx.BeginTx
    - Create order record (generate UUIDv4, store promo code, total price)
    - Create all order_items records within same transaction
    - Commit transaction if all successful, rollback on any error
    - Return created order with full details
  - [ ] GetOrder(ctx, orderID) - Retrieve order by ID with all items
  - [ ] Add comprehensive input validation and error handling
- [ ] Write unit tests for ProductService in `internal/service/`:
  - [ ] Mock ProductRepository interface
  - [ ] Test successful product listing with pagination
  - [ ] Test pagination validation (invalid limits/offsets)
  - [ ] Test product ID validation (all exist, some missing)
- [ ] Write unit tests for OrderService in `internal/service/`:
  - [ ] Mock OrderRepository, ProductRepository, and PromoValidator
  - [ ] Test successful order creation without promo code
  - [ ] Test successful order creation with valid promo code
  - [ ] Test order creation failure with invalid promo code
  - [ ] Test order creation failure with invalid product IDs (HTTP 422)
  - [ ] Test transaction rollback on order_items creation failure
  - [ ] Test timeout handling for promo validation
  - [ ] Test input validation (negative quantities, empty product list)
- [ ] Run service layer tests and verify they pass:
  - [ ] Execute `make test` for service package
  - [ ] Verify test coverage >80% for service implementations
- [ ] Perform a critical self-review of your changes and fix any issues found
- [ ] STOP and wait for human review

### Phase 5: HTTP Layer (Controllers & Middleware)

**Goal:** Implement HTTP handlers, middleware stack, and request/response handling.

- [ ] Implement middleware stack in `internal/middleware/`:
  - [ ] `recovery.go` - Panic recovery middleware (catches panics, logs, returns HTTP 500)
  - [ ] `correlation_id.go` - Generate/extract X-Correlation-ID header for request tracing
  - [ ] `auth.go` - API key authentication (check X-API-Key header against env var, return 401 if invalid)
  - [ ] `logger.go` - Request/response logging with correlation ID, method, path, status, duration
  - [ ] Apply middleware in correct order: Recovery → CorrelationID → Auth (conditionally) → Logger
- [ ] Implement HTTP handlers in `internal/handler/`:
  - [ ] `product_handler.go` - NewProductHandler(productService)
    - ListProducts handler: GET /products with query params (limit, offset)
    - GetProduct handler: GET /products/{id}
    - Parse pagination parameters and validate
    - Call service layer and return JSON response
    - Handle errors with proper HTTP status codes
  - [ ] `order_handler.go` - NewOrderHandler(orderService)
    - CreateOrder handler: POST /orders with JSON body
    - Parse and validate request body
    - Call service layer for order creation
    - Return HTTP 201 with created order details
    - Handle errors: 400 (malformed JSON), 422 (invalid products), 500 (server error)
    - GetOrder handler: GET /orders/{id} with path parameter
    - Validate UUID format of order ID
    - Return HTTP 404 if order not found
  - [ ] `handler.go` - Health check handler
    - Health check handler: GET /health (no auth required)
    - Return HTTP 200 with health status JSON
- [ ] Implement HTTP router setup in `internal/router/`:
  - [ ] `router.go` - New(handlers, apiKey, logger) function
  - [ ] Use standard library net/http.ServeMux
  - [ ] Register all routes with appropriate middleware
  - [ ] Skip authentication middleware for /health endpoint
- [ ] Implement request/response utilities in `internal/handler/`:
  - [ ] Helper functions for JSON responses in handler files
  - [ ] Proper error handling and status codes
- [ ] Write unit tests for middleware in `internal/middleware/`:
  - [ ] Test recovery middleware catches panics and returns 500
  - [ ] Test correlation ID middleware generates and propagates ID
  - [ ] Test auth middleware validates API key correctly (valid, missing, invalid)
  - [ ] Test logger middleware logs request/response details
- [ ] Write unit tests for handlers in `internal/handler/`:
  - [ ] Mock service layer interfaces
  - [ ] Test successful product listing with pagination
  - [ ] Test successful product retrieval by ID
  - [ ] Test successful order creation with and without promo code
  - [ ] Test order creation errors (400, 422, 500 responses)
  - [ ] Test order retrieval (success and 404)
  - [ ] Test health check endpoint
- [ ] Run HTTP layer tests and verify they pass:
  - [ ] Execute `make test` for handler and middleware packages
  - [ ] Verify test coverage >80% for HTTP layer
- [ ] Perform a critical self-review of your changes and fix any issues found
- [ ] STOP and wait for human review

### Phase 6: Configuration, Observability & Application Bootstrap

**Goal:** Implement configuration management, structured logging, Prometheus metrics, and main application entry point.

- [ ] Implement configuration management in `internal/config/`:
  - [ ] `config.go` - Config struct with all application settings
  - [ ] Load() function to read from environment variables with validation
  - [ ] Validate required variables (DATABASE_URL, API_KEY) are present
  - [ ] Provide sensible defaults (SERVER_PORT=8080, LOG_LEVEL=info)
  - [ ] Return error if required configuration is missing
- [ ] Implement structured logging setup in `internal/config/`:
  - [ ] `logger.go` - SetupLogger(level) function using zerolog
  - [ ] Configure JSON output format
  - [ ] Set log level from environment (debug, info, warn, error)
  - [ ] Add global fields (service="mini-kart", version)
- [ ] Implement Prometheus metrics in `internal/middleware/`:
  - [ ] `metrics.go` - Define Prometheus metrics:
    - http_requests_total (counter with labels: method, path, status)
    - http_request_duration_seconds (histogram with labels: method, path)
    - promo_validation_duration_seconds (histogram)
    - promo_validation_total (counter with labels: result={valid,invalid,error})
  - [ ] Register metrics with Prometheus registry
  - [ ] Create metrics middleware to record HTTP metrics
- [ ] Implement metrics instrumentation in service layer:
  - [ ] Wrap promo validation calls with duration measurement
  - [ ] Record validation results (valid, invalid, error, timeout)
- [ ] Implement main application in `cmd/api/main.go`:
  - [ ] Load configuration and validate
  - [ ] Setup structured logger
  - [ ] Initialise database connection pool
  - [ ] Verify coupon files exist and are readable (via validator initialisation)
  - [ ] Initialise all layers (repositories, services, handlers)
  - [ ] Setup HTTP router with middleware
  - [ ] Start HTTP server with graceful shutdown on SIGINT/SIGTERM
  - [ ] Log server start with configuration summary
- [ ] Implement graceful shutdown in `cmd/api/main.go`:
  - [ ] Listen for OS signals (SIGINT, SIGTERM)
  - [ ] On signal: stop accepting new requests, drain existing requests (30s timeout), close database pool
  - [ ] Log shutdown process
  - [ ] Force close if graceful shutdown fails
- [ ] Update Makefile with additional targets:
  - [ ] `run` - Start the application locally
  - [ ] `dev` - Run with live reload (using air or similar)
  - [ ] `docker-up` - Start Docker Compose services
  - [ ] `docker-down` - Stop Docker Compose services
- [ ] Test application startup and configuration:
  - [ ] Start PostgreSQL via `make docker-up`
  - [ ] Run database migrations via `make migrate-up`
  - [ ] Set environment variables from `.env.example`
  - [ ] Start application via `make run`
  - [ ] Verify logs show successful startup with connection pool info
  - [ ] Verify health check endpoint responds: `curl http://localhost:8080/health`
  - [ ] Verify metrics endpoint responds: `curl http://localhost:8080/metrics`
  - [ ] Test graceful shutdown with Ctrl+C
- [ ] Perform a critical self-review of your changes and fix any issues found
- [ ] STOP and wait for human review

### Phase 7: Integration & End-to-End Testing

**Goal:** Write integration tests with real database and end-to-end API tests.

- [ ] Setup testcontainers infrastructure in `test/`:
  - [ ] `testcontainer.go` - Helper to start PostgreSQL container for tests
  - [ ] Run migrations on test database automatically
  - [ ] Seed test database with sample products
  - [ ] Provide cleanup function to stop container after tests
- [ ] Write integration tests for repositories in `test/integration/`:
  - [ ] `product_repository_test.go` - Test with real PostgreSQL database
    - Test GetAll with pagination
    - Test GetByIDs with valid and invalid IDs
    - Test ValidateProductsExist for various scenarios
  - [ ] `order_repository_test.go` - Test with real PostgreSQL database
    - Test Create with transaction commit
    - Test Create with transaction rollback
    - Test GetByID for existing and non-existing orders
    - Verify foreign key constraints
- [ ] Write end-to-end API tests in `test/e2e/`:
  - [ ] `api_test.go` - Test full HTTP API with real database and promo validator
    - Test GET /products endpoint with pagination
    - Test POST /orders without promo code (success)
    - Test POST /orders with valid promo code (success)
    - Test POST /orders with invalid promo code (422 error)
    - Test POST /orders with invalid product IDs (422 error)
    - Test POST /orders with malformed JSON (400 error)
    - Test GET /orders/{id} for existing order (success)
    - Test GET /orders/{id} for non-existing order (404 error)
    - Test GET /health endpoint (no auth required)
    - Test authentication: requests without X-API-Key header (401 error)
    - Test authentication: requests with invalid X-API-Key (401 error)
- [ ] Write concurrent promo validation performance tests in `test/performance/`:
  - [ ] `promo_concurrent_test.go` - Test concurrent validation scenarios
    - Test 10 concurrent validation requests
    - Test 50 concurrent validation requests
    - Test 100 concurrent validation requests
    - Measure P95 response time (target <500ms for small test files)
    - Verify no race conditions with `go test -race`
    - Verify proper context cancellation and go-routine cleanup
- [ ] Update Makefile test targets:
  - [ ] `test-unit` - Run unit tests only (fast)
  - [ ] `test-integration` - Run integration tests with testcontainers
  - [ ] `test-e2e` - Run end-to-end API tests
  - [ ] `test-all` - Run all tests (unit + integration + e2e)
  - [ ] `test-race` - Run all tests with race detector enabled
- [ ] Run full test suite and verify all tests pass:
  - [ ] Execute `make test-unit` and verify all unit tests pass (<10 seconds)
  - [ ] Execute `make test-integration` and verify integration tests pass
  - [ ] Execute `make test-e2e` and verify all API scenarios work
  - [ ] Execute `make test-race` and verify no race conditions detected
  - [ ] Verify overall test coverage >80%: `go test -cover ./...`
- [ ] Perform a critical self-review of your changes and fix any issues found
- [ ] STOP and wait for human review

### Phase 8: Containerisation, Linting & Final Review

**Goal:** Build Docker image, run linting, perform security review, and verify all success criteria met.

- [ ] Create Dockerfile with multi-stage build:
  - [ ] Stage 1 (builder): Use golang:1.21+ alpine image
    - Copy go.mod and go.sum, run `go mod download`
    - Copy source code
    - Build binary with `-ldflags="-s -w"` for smaller binary
  - [ ] Stage 2 (runtime): Use alpine:latest image
    - Copy binary from builder stage
    - Copy migrations directory
    - Expose port 8080
    - Set non-root user for security
    - Set entrypoint to run application
- [ ] Create .dockerignore file:
  - [ ] Exclude .git, .env, test files, IDE files, documentation
- [ ] Update docker-compose.yml with application service:
  - [ ] Add mini-kart-api service using local Dockerfile
  - [ ] Configure environment variables
  - [ ] Mount coupon files volume
  - [ ] Expose port 8080
  - [ ] Add dependency on PostgreSQL service
- [ ] Test Docker build and run:
  - [ ] Build Docker image: `docker build -t mini-kart:latest .`
  - [ ] Verify image size is reasonable (<50MB for alpine-based image)
  - [ ] Start full stack via `make docker-up`
  - [ ] Verify application starts successfully in container
  - [ ] Test health check: `curl http://localhost:8080/health`
  - [ ] Test API endpoints through container
  - [ ] Stop stack via `make docker-down`
- [ ] Setup linting configuration:
  - [ ] Create `.golangci.yml` with strict linting rules
  - [ ] Enable linters: govet, errcheck, staticcheck, gosimple, ineffassign, unused, misspell
  - [ ] Configure line length limits and cyclomatic complexity thresholds
- [ ] Run linting and fix all issues:
  - [ ] Execute `make lint` (golangci-lint run)
  - [ ] Fix all linting errors and warnings
  - [ ] Re-run linting until zero issues remain
- [ ] Run security scanning:
  - [ ] Install and run gosec: `gosec ./...`
  - [ ] Review and fix any security issues (SQL injection, hardcoded credentials, etc.)
  - [ ] Verify no sensitive data in logs or error messages
- [ ] Generate Postman collection for API testing:
  - [ ] Use OpenAPI specification to generate Postman collection
  - [ ] Configure collection with environment variables (API_KEY, BASE_URL)
  - [ ] Add example requests for all endpoints (GET /products, POST /orders, GET /orders/{id}, GET /health)
  - [ ] Include pre-request scripts for authentication (X-API-Key header)
  - [ ] Include test scripts to validate responses
  - [ ] Export collection to `docs/postman_collection.json`
  - [ ] Document in README how to import and use the Postman collection
- [ ] Add comprehensive go doc documentation:
  - [ ] Add package-level documentation comments to all packages (internal/*, cmd/api)
  - [ ] Add function documentation (go doc) to all exported functions explaining purpose, parameters, return values, and any errors
  - [ ] Add function documentation to all non-trivial unexported functions
  - [ ] Add struct and interface documentation explaining purpose and usage
  - [ ] Add documentation around complex logic and algorithms (especially in coupon validator)
  - [ ] DO NOT add documentation for trivial getters, setters, or self-explanatory functions
  - [ ] Ensure documentation follows Go documentation conventions (starts with function/type name)
  - [ ] Run `go doc` to verify documentation is properly generated and readable
  - [ ] Verify documentation uses Australian English spelling throughout
- [ ] Perform comprehensive final review:
  - [ ] Review all 192 requirements in `specs/mini-kart/requirements.md` and verify each is satisfied
  - [ ] Review architecture in `specs/mini-kart/design.md` and verify implementation matches design
  - [ ] Verify middleware stack order: Recovery → CorrelationID → Auth → Logger
  - [ ] Verify promo validation uses atomic operations for thread safety
  - [ ] Verify database transactions are atomic (order + order_items)
  - [ ] Verify context cancellation works for concurrent go-routines
  - [ ] Verify all error responses follow standardised format
  - [ ] Verify all timestamps are UTC with timezone awareness
  - [ ] Verify API key authentication is enforced (except /health)
- [ ] Verify all success criteria are met:
  - [ ] Run full test suite: `make test-all` (all tests pass)
  - [ ] Run linting: `make lint` (zero errors/warnings)
  - [ ] Build application: `make build` (builds successfully)
  - [ ] Build Docker image (builds successfully)
  - [ ] Start application and test all endpoints manually
  - [ ] Verify Prometheus metrics are exposed on /metrics
  - [ ] Verify structured JSON logs include correlation IDs
  - [ ] Verify promo validation completes within 5 seconds
  - [ ] Verify test coverage >80%: `go test -cover ./...`
- [ ] Update README.md with concise documentation:
  - [ ] Prerequisites section including Git LFS requirement
  - [ ] Quick start instructions (Git LFS setup, coupon file download, setup, run, test)
  - [ ] Environment variables reference (link to .env.example)
  - [ ] API endpoints summary (link to OpenAPI spec)
  - [ ] Development workflow (Makefile targets)
  - [ ] Architecture overview (link to design.md)
  - [ ] Do NOT add extensive documentation beyond essentials
- [ ] Perform final critical self-review:
  - [ ] Verify no TODO comments remain in code
  - [ ] Verify no debug logging or console.log statements remain
  - [ ] Verify no hardcoded credentials, API keys, or localhost URLs
  - [ ] Verify Australian English spelling used throughout (e.g., "summarise", "initialise", "colour")
  - [ ] Verify all files follow project conventions (package names, file structure)
- [ ] STOP and wait for human review and approval

---

## Notes

**Key Implementation Patterns:**

1. **Error Handling:** Use errors.Is() and errors.As() for error type checking. Return domain-specific errors from service layer that controllers translate to HTTP status codes.

2. **Context Usage:** Pass context.Context as first parameter to all functions performing I/O. Use context.WithTimeout for operations with time limits.

3. **Transaction Management:** Use pgx.BeginTx() for database transactions. Always defer rollback and only commit on success.

4. **Concurrency:** Use go-routines with channels for concurrent processing. Always provide context cancellation mechanism. Use atomic operations for shared state.

5. **Testing:** Mock interfaces at layer boundaries (repository, service interfaces). Use testcontainers for integration tests with real database. Use small test fixtures for concurrent validation tests.

**Reference Documents:**
- Requirements: `specs/mini-kart/requirements.md` (192 acceptance criteria)
- Design: `specs/mini-kart/design.md` (architecture, patterns, algorithms)
- Decisions: `specs/mini-kart/decision_log.md` (all architectural decisions)

---

## Working Notes

**Purpose:** This section is for the executing agent to track complex issues, troubleshooting attempts, and problem-solving progress during development.

**Format:** Use freely - bullet points, links, error messages, debugging notes. Keep updated as issues are resolved.

---
