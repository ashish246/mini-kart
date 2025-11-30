# Mini-Kart API Server Requirements

## Introduction

This document specifies the requirements for building a Go-based food ordering API server (mini-kart) that implements the OpenAPI 3.1 specification. The system focuses on concurrent promo code validation using Go routines and channels, with PostgreSQL for data persistence. The implementation emphasises clean architecture, performance, scalability, and proper separation of concerns across controller, service, model, and utility layers.

## 1. API Endpoint Implementation

**User Story:** As an API consumer, I want to interact with standard RESTful endpoints for products and orders, so that I can build a food ordering application.

**Acceptance Criteria:**

1. <a name="1.1"></a>The system SHALL implement GET /product endpoint that returns available products with optional pagination
2. <a name="1.2"></a>The system SHALL support optional query parameters limit (default 100) and offset (default 0) for GET /product
3. <a name="1.3"></a>The system SHALL implement GET /product/{productId} endpoint that returns a single product by ID
4. <a name="1.4"></a>The system SHALL treat productId as a string value to match OpenAPI schema definition
5. <a name="1.5"></a>The system SHALL implement POST /order endpoint that creates a new order with optional coupon code
6. <a name="1.6"></a>The system SHALL conform to the OpenAPI 3.1 specification provided at orderfoodonline.deno.dev/public/openapi.yaml
7. <a name="1.7"></a>The system SHALL return HTTP 200 with product array for successful GET /product requests
8. <a name="1.8"></a>The system SHALL return HTTP 200 with product object for successful GET /product/{productId} requests
9. <a name="1.9"></a>The system SHALL return HTTP 404 when product ID does not exist
10. <a name="1.10"></a>The system SHALL return HTTP 400 for malformed request structure (invalid JSON, missing required fields)
11. <a name="1.11"></a>The system SHALL return HTTP 200 with order object for successful POST /order requests
12. <a name="1.12"></a>The system SHALL return HTTP 422 for semantic validation failures (invalid promo code, non-existent product IDs, invalid quantities)
13. <a name="1.13"></a>The system SHALL accept Content-Type: application/json for all POST requests
14. <a name="1.14"></a>The system SHALL return Content-Type: application/json for all responses

## 2. Authentication and Security

**User Story:** As an API administrator, I want to secure the order endpoint with API key authentication, so that only authorised clients can place orders.

**Acceptance Criteria:**

1. <a name="2.1"></a>The system SHALL require api_key header for POST /order endpoint
2. <a name="2.2"></a>The system SHALL load valid API key from environment variable API_KEY with default value "apitest"
3. <a name="2.3"></a>The system SHALL validate api_key header against configured API key value
4. <a name="2.4"></a>The system SHALL return HTTP 401 when api_key header is missing
5. <a name="2.5"></a>The system SHALL return HTTP 403 when api_key header value does not match configured key
6. <a name="2.6"></a>The system SHALL validate and sanitise all user inputs to prevent injection attacks
7. <a name="2.7"></a>The system SHALL use parameterised queries for all database operations
8. <a name="2.8"></a>The system SHALL not expose internal error details or stack traces in API responses

## 3. Promo Code Validation

**User Story:** As an API consumer, I want to apply valid promo codes to orders, so that validated codes can be tracked with customer orders.

**Acceptance Criteria:**

1. <a name="3.1"></a>The system SHALL validate promo codes against three coupon files (couponbase1.gz, couponbase2.gz, couponbase3.gz)
2. <a name="3.2"></a>The system SHALL accept promo codes with length between 8 and 10 characters (inclusive)
3. <a name="3.3"></a>The system SHALL consider a promo code valid only if found in at least two of the three coupon files
4. <a name="3.4"></a>The system SHALL perform case-insensitive promo code matching to improve user experience
5. <a name="3.5"></a>The system SHALL reject promo codes that do not meet length requirements with error code "INVALID_PROMO_LENGTH"
6. <a name="3.6"></a>The system SHALL reject promo codes found in fewer than two files with error code "INVALID_PROMO_CODE"
7. <a name="3.7"></a>The system SHALL return HTTP 422 with structured error when invalid promo code is provided
8. <a name="3.8"></a>The system SHALL store validated promo code with the order in coupon_code field
9. <a name="3.9"></a>The system SHALL allow orders without promo codes (coupon_code is optional)
10. <a name="3.10"></a>The system SHALL complete promo validation before starting database transaction to avoid holding connections during I/O

## 4. Coupon File Specifications

**User Story:** As a developer, I want clear specifications for coupon file format and location, so that the system can reliably process promo codes.

**Acceptance Criteria:**

1. <a name="4.1"></a>The system SHALL locate coupon files in the data/coupons/ directory relative to project root
2. <a name="4.2"></a>The system SHALL expect files named couponbase1.gz, couponbase2.gz, and couponbase3.gz
3. <a name="4.3"></a>The system SHALL store coupon files in Git LFS (Large File Storage) due to their size
4. <a name="4.4"></a>The system SHALL load coupon files from Git LFS tracked location directly at runtime on server startup
5. <a name="4.5"></a>The system SHALL read files in gzip compressed format
6. <a name="4.6"></a>The system SHALL expect file format as one promo code per line, UTF-8 encoded, with no header row
7. <a name="4.7"></a>The system SHALL treat empty lines and lines starting with # as comments to be ignored
8. <a name="4.8"></a>The system SHALL trim whitespace from each line before comparison
9. <a name="4.9"></a>The system SHALL handle coupon files up to 1GB compressed size (approximately 100 million codes)
10. <a name="4.10"></a>The system SHALL load coupon file paths from configuration with defaults: data/coupons/couponbase1.gz, data/coupons/couponbase2.gz, data/coupons/couponbase3.gz
11. <a name="4.11"></a>The system SHALL fail at startup if any configured coupon file is missing or unreadable
12. <a name="4.12"></a>The system SHALL validate file accessibility during startup health check
13. <a name="4.13"></a>The system SHALL NOT include large coupon files in the deployment artefact (files managed via Git LFS)

## 5. Concurrent File Processing

**User Story:** As a system administrator, I want promo code validation to use concurrent processing, so that validation completes quickly even with large coupon files.

**Acceptance Criteria:**

1. <a name="5.1"></a>The system SHALL use separate go-routines to read each coupon file concurrently
2. <a name="5.2"></a>The system SHALL stream file contents using bufio.Scanner without loading entire files into memory
3. <a name="5.3"></a>The system SHALL use buffered channels to communicate results between go-routines
4. <a name="5.4"></a>The system SHALL use context.WithCancel to cancel running go-routines once promo code is found in two files
5. <a name="5.5"></a>The system SHALL decompress .gz files during streaming using compress/gzip
6. <a name="5.6"></a>The system SHALL return validation result as soon as two matches are confirmed
7. <a name="5.7"></a>The system SHALL properly clean up resources (file handles, go-routines) on completion or cancellation using defer
8. <a name="5.8"></a>The system SHALL use atomic operations or mutex for thread-safe match counting
9. <a name="5.9"></a>The system SHALL timeout promo validation after 5 seconds using context.WithTimeout
10. <a name="5.10"></a>The system SHALL handle concurrent validation requests without race conditions (verified by go test -race)

## 6. Data Models and Persistence

**User Story:** As a developer, I want complete data models with PostgreSQL persistence, so that product and order data is stored reliably with proper schema.

**Acceptance Criteria:**

1. <a name="6.1"></a>The system SHALL define Product model with fields: id (string), name (string), price (float64), category (string)
2. <a name="6.2"></a>The system SHALL define Order model with fields: id (UUID), coupon_code (string, nullable), created_at (time.Time), updated_at (time.Time)
3. <a name="6.3"></a>The system SHALL define OrderItem model with fields: id (UUID), order_id (UUID), product_id (string), quantity (int)
4. <a name="6.4"></a>The system SHALL create products table with schema: id TEXT PRIMARY KEY, name TEXT NOT NULL, price DECIMAL(10,2) NOT NULL, category TEXT NOT NULL
5. <a name="6.5"></a>The system SHALL create orders table with schema: id UUID PRIMARY KEY, coupon_code TEXT, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL
6. <a name="6.6"></a>The system SHALL create order_items table with schema: id UUID PRIMARY KEY, order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE, product_id TEXT NOT NULL REFERENCES products(id), quantity INTEGER NOT NULL CHECK (quantity > 0)
7. <a name="6.7"></a>The system SHALL create index on products(category) for category-based queries
8. <a name="6.8"></a>The system SHALL create index on order_items(order_id) for order retrieval
9. <a name="6.9"></a>The system SHALL generate UUIDv4 for new orders and order items
10. <a name="6.10"></a>The system SHALL validate that all product IDs in order request exist in database before creating order
11. <a name="6.11"></a>The system SHALL validate that all item quantities are positive integers greater than zero
12. <a name="6.12"></a>The system SHALL use database transaction encompassing: order insert, all order_items inserts
13. <a name="6.13"></a>The system SHALL set created_at and updated_at to current UTC timestamp on order creation
14. <a name="6.14"></a>The system SHALL provide database migration files using golang-migrate or similar tool
15. <a name="6.15"></a>The system SHALL return order response including: order id, items array, products array with full product details

## 7. Error Response Format

**User Story:** As an API consumer, I want consistent error response structure, so that I can handle errors programmatically.

**Acceptance Criteria:**

1. <a name="7.1"></a>The system SHALL return errors in JSON format with fields: error (string), message (string), code (string)
2. <a name="7.2"></a>The system SHALL use error codes: INVALID_JSON, MISSING_FIELD, INVALID_PROMO_CODE, INVALID_PROMO_LENGTH, PRODUCT_NOT_FOUND, INVALID_QUANTITY, UNAUTHORIZED, FORBIDDEN, INTERNAL_ERROR
3. <a name="7.3"></a>The system SHALL include human-readable message describing the error
4. <a name="7.4"></a>The system SHALL not expose stack traces, database errors, or internal paths in error responses
5. <a name="7.5"></a>The system SHALL include correlation ID in error responses for tracing
6. <a name="7.6"></a>The system SHALL return HTTP 400 with INVALID_JSON code for malformed JSON
7. <a name="7.7"></a>The system SHALL return HTTP 400 with MISSING_FIELD code for missing required fields
8. <a name="7.8"></a>The system SHALL return HTTP 422 with PRODUCT_NOT_FOUND code when product IDs don't exist
9. <a name="7.9"></a>The system SHALL return HTTP 422 with INVALID_PROMO_CODE or INVALID_PROMO_LENGTH for promo validation failures
10. <a name="7.10"></a>The system SHALL return HTTP 500 with INTERNAL_ERROR code for unexpected server errors

## 8. Code Structure and Architecture

**User Story:** As a developer, I want clean separation of concerns with proper layering, so that the codebase is maintainable and testable.

**Acceptance Criteria:**

1. <a name="8.1"></a>The system SHALL organise code into packages: cmd/api (main), internal/handler, internal/service, internal/repository, internal/model, internal/config, internal/coupon
2. <a name="8.2"></a>The system SHALL implement handlers that handle HTTP requests/responses and call services
3. <a name="8.3"></a>The system SHALL implement services that contain business logic and orchestrate repositories
4. <a name="8.4"></a>The system SHALL implement repositories that handle database operations using interfaces
5. <a name="8.5"></a>The system SHALL define models as pure data structures in internal/model
6. <a name="8.6"></a>The system SHALL place coupon validation logic in internal/coupon package
7. <a name="8.7"></a>The system SHALL use dependency injection via constructor functions
8. <a name="8.8"></a>The system SHALL define interfaces for repositories and external dependencies to enable mocking
9. <a name="8.9"></a>The system SHALL follow single responsibility principle for all functions and structs
10. <a name="8.10"></a>The system SHALL limit function length to approximately 50 lines (flexible for readability)
11. <a name="8.11"></a>The system SHALL limit file length to approximately 700 lines (split larger files)
12. <a name="8.12"></a>The system SHALL keep cyclomatic complexity under 10 for all functions

## 9. Configuration Management

**User Story:** As a system administrator, I want centralised configuration with environment variables, so that deployment settings can be managed easily.

**Acceptance Criteria:**

1. <a name="9.1"></a>The system SHALL load configuration from environment variables with fallback to .env file
2. <a name="9.2"></a>The system SHALL provide .env.example file documenting all configuration variables
3. <a name="9.3"></a>The system SHALL validate all required environment variables at startup
4. <a name="9.4"></a>The system SHALL fail fast with clear error messages listing missing configuration
5. <a name="9.5"></a>The system SHALL support environment variables: DATABASE_URL (required), SERVER_PORT (default 8080), API_KEY (default "apitest"), COUPON_FILE_1 (default data/couponbase1.gz), COUPON_FILE_2 (default data/couponbase2.gz), COUPON_FILE_3 (default data/couponbase3.gz), LOG_LEVEL (default INFO), PROMO_VALIDATION_TIMEOUT (default 5s), DB_MAX_OPEN_CONNS (default 25), DB_MAX_IDLE_CONNS (default 10), GRACEFUL_SHUTDOWN_TIMEOUT (default 30s)
6. <a name="9.6"></a>The system SHALL use structured Config struct loaded once at startup
7. <a name="9.7"></a>The system SHALL not access environment variables directly outside config package
8. <a name="9.8"></a>The system SHALL never commit .env file to version control (include in .gitignore)
9. <a name="9.9"></a>The system SHALL validate configuration values (e.g., port range, positive integers)

## 10. Error Handling and Logging

**User Story:** As a developer and operator, I want structured error handling and logging, so that issues can be diagnosed and resolved quickly.

**Acceptance Criteria:**

1. <a name="10.1"></a>The system SHALL use structured logging library (e.g., zerolog or zap) with JSON output
2. <a name="10.2"></a>The system SHALL generate correlation ID (UUIDv4) for each request and include in all logs
3. <a name="10.3"></a>The system SHALL include correlation ID in X-Correlation-ID response header
4. <a name="10.4"></a>The system SHALL log at levels: ERROR (failures requiring attention), WARN (degraded state), INFO (normal operations), DEBUG (detailed diagnostics)
5. <a name="10.5"></a>The system SHALL log each request with: method, path, status, duration, correlation_id
6. <a name="10.6"></a>The system SHALL log errors with: error message, correlation_id, context (e.g., product_id, order_id)
7. <a name="10.7"></a>The system SHALL never log sensitive data (api_key values, full request bodies containing potential PII)
8. <a name="10.8"></a>The system SHALL include fields in all logs: timestamp (RFC3339), level, service (mini-kart), correlation_id
9. <a name="10.9"></a>The system SHALL handle database connection errors with exponential backoff retry (max 3 attempts)
10. <a name="10.10"></a>The system SHALL handle file system errors (missing coupon files) by failing startup with clear error
11. <a name="10.11"></a>The system SHALL return HTTP 500 for unexpected errors while logging full details internally
12. <a name="10.12"></a>The system SHALL allow log level configuration via LOG_LEVEL environment variable

## 11. Testing Requirements

**User Story:** As a developer, I want comprehensive automated tests, so that code quality and correctness are maintained.

**Acceptance Criteria:**

1. <a name="11.1"></a>The system SHALL have unit tests for all service layer functions using table-driven approach
2. <a name="11.2"></a>The system SHALL have unit tests for promo validation logic with test coupon files
3. <a name="11.3"></a>The system SHALL have integration tests for all API endpoints using httptest
4. <a name="11.4"></a>The system SHALL have integration tests using testcontainers for PostgreSQL
5. <a name="11.5"></a>The system SHALL have concurrent tests running go test -race to detect race conditions
6. <a name="11.6"></a>The system SHALL have benchmark tests for promo validation measuring allocations and duration
7. <a name="11.7"></a>The system SHALL achieve minimum 80% code coverage measured by go test -cover
8. <a name="11.8"></a>The system SHALL exclude cmd/api/main.go and generated code from coverage requirements
9. <a name="11.9"></a>The system SHALL use descriptive test names in format Test<Function>_<Scenario>_<Expected>
10. <a name="11.10"></a>The system SHALL follow Arrange-Act-Assert pattern in all tests
11. <a name="11.11"></a>The system SHALL mock repositories using interfaces in service tests
12. <a name="11.12"></a>The system SHALL create test fixtures in testdata/ directory
13. <a name="11.13"></a>The system SHALL run unit tests in under 5 seconds (integration tests can be slower)

## 12. Observability and Operations

**User Story:** As an operator, I want health checks, metrics, and graceful shutdown, so that the service can be monitored and deployed reliably.

**Acceptance Criteria:**

1. <a name="12.1"></a>The system SHALL provide GET /health endpoint returning JSON with status field
2. <a name="12.2"></a>The system SHALL return HTTP 200 with {"status": "healthy"} when all dependencies are available
3. <a name="12.3"></a>The system SHALL return HTTP 503 with {"status": "unhealthy", "details": {...}} when database is unreachable
4. <a name="12.4"></a>The system SHALL check database connectivity with SELECT 1 query in health endpoint
5. <a name="12.5"></a>The system SHALL provide GET /metrics endpoint with Prometheus text format metrics
6. <a name="12.6"></a>The system SHALL track metrics: http_requests_total (counter with labels: method, path, status), http_request_duration_seconds (histogram), promo_validation_duration_seconds (histogram), promo_validation_errors_total (counter), database_queries_total (counter with labels: query_type, status)
7. <a name="12.7"></a>The system SHALL handle SIGTERM and SIGINT signals for graceful shutdown
8. <a name="12.8"></a>The system SHALL stop accepting new HTTP requests immediately on shutdown signal
9. <a name="12.9"></a>The system SHALL wait up to 30 seconds (configurable) for in-flight requests to complete
10. <a name="12.10"></a>The system SHALL close database connection pool during shutdown
11. <a name="12.11"></a>The system SHALL log shutdown initiation and completion
12. <a name="12.12"></a>The system SHALL use correlation IDs consistently across all service layers

## 13. Performance and Scalability

**User Story:** As a system architect, I want the system to handle concurrent requests efficiently, so that it can scale to meet demand.

**Acceptance Criteria:**

1. <a name="13.1"></a>The system SHALL complete promo code validation in under 500ms for 3 files each containing up to 1 million codes (~10MB compressed each)
2. <a name="13.2"></a>The system SHALL handle at least 100 concurrent POST /order requests without errors
3. <a name="13.3"></a>The system SHALL use pgx connection pool for database connections
4. <a name="13.4"></a>The system SHALL configure connection pool with max 25 open connections, max 10 idle connections
5. <a name="13.5"></a>The system SHALL set HTTP read timeout to 10 seconds, write timeout to 10 seconds
6. <a name="13.6"></a>The system SHALL set HTTP idle timeout to 120 seconds
7. <a name="13.7"></a>The system SHALL set database query context timeout to 10 seconds
8. <a name="13.8"></a>The system SHALL timeout promo validation after 5 seconds (configurable)
9. <a name="13.9"></a>The system SHALL be stateless with no in-memory session storage
10. <a name="13.10"></a>The system SHALL handle database connection pool exhaustion by queuing requests up to timeout

## 14. Build and Development Tooling

**User Story:** As a developer, I want standardised build tooling via Makefile, so that common operations are simple and consistent.

**Acceptance Criteria:**

1. <a name="14.1"></a>The system SHALL provide Makefile with phony targets for all commands
2. <a name="14.2"></a>The system SHALL provide make build command executing go build -ldflags="-s -w" -o bin/api cmd/api/main.go
3. <a name="14.3"></a>The system SHALL provide make test command executing go test -v -race -cover ./...
4. <a name="14.4"></a>The system SHALL provide make test-coverage command generating HTML coverage report
5. <a name="14.5"></a>The system SHALL provide make lint command executing golangci-lint run
6. <a name="14.6"></a>The system SHALL provide make fmt command executing gofmt and goimports
7. <a name="14.7"></a>The system SHALL provide make run command starting server with air for hot reload or go run
8. <a name="14.8"></a>The system SHALL provide make docker-up command executing docker-compose up -d
9. <a name="14.9"></a>The system SHALL provide make docker-down command executing docker-compose down
10. <a name="14.10"></a>The system SHALL provide make security-scan command executing gosec ./...
11. <a name="14.11"></a>The system SHALL provide make migrate-up and make migrate-down commands for database migrations
12. <a name="14.12"></a>The system SHALL provide make seed command to populate sample product data
13. <a name="14.13"></a>The system SHALL use Go 1.21 or later
14. <a name="14.14"></a>The system SHALL pass golangci-lint without errors or warnings
15. <a name="14.15"></a>The system SHALL pass gosec scan with no medium or high severity issues

## 15. Docker and Deployment

**User Story:** As a developer, I want Docker configuration for local development, so that dependencies can be managed easily.

**Acceptance Criteria:**

1. <a name="15.1"></a>The system SHALL provide docker-compose.yml defining services: postgres, server
2. <a name="15.2"></a>The system SHALL configure PostgreSQL service using postgres:16-alpine image
3. <a name="15.3"></a>The system SHALL mount PostgreSQL data to named volume for persistence
4. <a name="15.4"></a>The system SHALL configure PostgreSQL with environment variables: POSTGRES_DB=minikart, POSTGRES_USER=postgres, POSTGRES_PASSWORD=postgres
5. <a name="15.5"></a>The system SHALL expose PostgreSQL on port 5432
6. <a name="15.6"></a>The system SHALL provide Dockerfile using multi-stage build: builder stage with Go, runtime stage with alpine
7. <a name="15.7"></a>The system SHALL copy coupon files to /app/data/ in Docker image
8. <a name="15.8"></a>The system SHALL expose server on port 8080
9. <a name="15.9"></a>The system SHALL configure Docker health check calling GET /health
10. <a name="15.10"></a>The system SHALL set health check interval to 30s, timeout to 10s, retries to 3
11. <a name="15.11"></a>The system SHALL run server as non-root user in container
12. <a name="15.12"></a>The system SHALL set working directory to /app in container
13. <a name="15.13"></a>The system SHALL provide .dockerignore file excluding unnecessary files from build context

## 16. Database Seeding and Sample Data

**User Story:** As a developer, I want sample data for testing, so that I can verify API functionality easily.

**Acceptance Criteria:**

1. <a name="16.1"></a>The system SHALL provide seed data SQL file with minimum 10 sample products
2. <a name="16.2"></a>The system SHALL include products across multiple categories (e.g., Waffle, Sandwich, Beverage, Dessert)
3. <a name="16.3"></a>The system SHALL provide make seed command to load seed data
4. <a name="16.4"></a>The system SHALL make seed command idempotent (safe to run multiple times)
5. <a name="16.5"></a>The system SHALL use realistic product names and prices in seed data
6. <a name="16.6"></a>The system SHALL document sample valid promo codes in README for testing (e.g., HAPPYHRS, FIFTYOFF)

## 17. Documentation

**User Story:** As a developer, I want clear documentation, so that I can understand, set up, and maintain the system.

**Acceptance Criteria:**

1. <a name="17.1"></a>The system SHALL provide README.md with sections: Overview, Features, Prerequisites, Setup, Running, Testing, API Endpoints, Configuration
2. <a name="17.2"></a>The system SHALL document all Makefile commands in README
3. <a name="17.3"></a>The system SHALL document all environment variables in .env.example with descriptions
4. <a name="17.4"></a>The system SHALL provide example API requests using curl or httpie
5. <a name="17.5"></a>The system SHALL document promo code validation rules clearly
6. <a name="17.6"></a>The system SHALL document database schema in migrations or separate schema.sql
7. <a name="17.7"></a>The system SHALL keep README concise (under 500 lines)
8. <a name="17.8"></a>The system SHALL use Australian English spelling throughout documentation
