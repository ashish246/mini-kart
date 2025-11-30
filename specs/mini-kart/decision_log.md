# Mini-Kart Decision Log

## Feature Name Decision
**Date:** 2025-11-27
**Decision:** Use "mini-kart" as the feature name instead of "mini-kart-api"
**Rationale:** User preference for shorter, cleaner name
**Impact:** Directory structure at specs/mini-kart/

## Data Storage Decision
**Date:** 2025-11-27
**Decision:** Use PostgreSQL database for data persistence
**Rationale:** User requirement for full relational database with persistence
**Impact:** Requires PostgreSQL setup, migrations, and connection pooling
**Alternatives Considered:** In-memory store, SQLite, MongoDB

## Coupon File Handling Decision
**Date:** 2025-11-27 (Updated: 2025-11-30)
**Decision:** Coupon files (couponbase1.gz, couponbase2.gz, couponbase3.gz) are stored in Git LFS and loaded directly at server startup
**Rationale:**
- Large compressed files (up to 1GB each) cannot be part of deployment artefacts
- Git LFS enables version control for large files without bloating the repository
- Files are loaded from Git LFS tracked location at runtime on server startup
- No download logic needed - files are already present via Git LFS checkout
**Impact:**
- Files stored in data/coupons/ directory tracked by Git LFS
- Server loads files directly from filesystem at startup
- Developers must have Git LFS installed and configured
- CI/CD pipelines must support Git LFS
**Alternatives Considered:**
- Download at startup from S3 (rejected: adds external dependency and startup latency)
- Include in deployment artefact (rejected: 1GB+ files too large for deployment packages)
- Store in repository without LFS (rejected: would bloat repository significantly)

## Testing Scope Decision
**Date:** 2025-11-27
**Decision:** Implement comprehensive testing including unit, integration, concurrent, and performance tests
**Rationale:** User requirement for all test types to ensure quality and correctness
**Impact:** Significant test coverage expected (80% minimum), requires test infrastructure for all types
**Coverage:** Unit tests for all services/utils, integration tests for APIs, concurrent tests for go-routines, performance benchmarks for promo validation

## Additional Features Decision
**Date:** 2025-11-27
**Decision:** Include health check endpoint, metrics endpoint, graceful shutdown, and request logging
**Rationale:** User requirement for production-ready observability and operations features
**Impact:** Additional endpoints at /health and /metrics, signal handling for graceful shutdown, structured logging with correlation IDs
**Alternatives Considered:** Basic implementation without observability features

---

## Critical Issues Resolution (Option A)

### Product ID Data Type Resolution
**Date:** 2025-11-27
**Decision:** Product IDs are strings (TEXT type in database)
**Rationale:** OpenAPI schema definition shows id as string with example "10". String provides flexibility for various ID formats (numeric, alphanumeric, UUIDs)
**Impact:** Path parameter {productId} accepts any string, validation focuses on existence not format
**Updated Requirements:** Section 1.4

### Coupon File Specifications
**Date:** 2025-11-27 (Updated: 2025-11-30)
**Decision:** Complete file specification added as Section 4
**Details:**
- **Location:** data/coupons/ directory relative to project root
- **Names:** couponbase1.gz, couponbase2.gz, couponbase3.gz
- **Storage:** Git LFS (Large File Storage) for version control
- **Loading:** Files loaded directly from Git LFS tracked location at server startup
- **Format:** One promo code per line, UTF-8, no headers, gzip compressed
- **Size:** Up to 1GB compressed (approximately 100 million codes)
- **Comments:** Lines starting with # or empty lines ignored
- **Configuration:** Paths configurable via COUPON_FILE_1, COUPON_FILE_2, COUPON_FILE_3 env vars
**Rationale:** Provides clear, testable specification for file handling. Git LFS enables efficient version control for large binary files.
**Impact:**
- Startup validation required to ensure files are accessible
- Clear error messages for missing files
- Git LFS must be installed and configured in development and CI/CD environments
**Updated Requirements:** New Section 4 "Coupon File Specifications"

### Promo Code Success Logic
**Date:** 2025-11-27
**Decision:** Valid promo codes are stored with orders but do not affect pricing
**Details:**
- Validated codes stored in orders.coupon_code field (nullable TEXT)
- No discount calculation in this version
- Validation occurs before database transaction
- Optional field in API request
**Rationale:** Challenge focuses on concurrent validation, not business pricing logic. Storage allows future discount implementation
**Impact:** Order model includes coupon_code, validation is pure existence check
**Updated Requirements:** Sections 3.8-3.10, 6.2, 6.5

### Complete Database Schema
**Date:** 2025-11-27
**Decision:** Full schema specification with types, constraints, indexes
**Products Table:**
- id TEXT PRIMARY KEY
- name TEXT NOT NULL
- price DECIMAL(10,2) NOT NULL
- category TEXT NOT NULL
- Index on category

**Orders Table:**
- id UUID PRIMARY KEY
- coupon_code TEXT (nullable)
- created_at TIMESTAMPTZ NOT NULL
- updated_at TIMESTAMPTZ NOT NULL

**Order_Items Table:**
- id UUID PRIMARY KEY
- order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE
- product_id TEXT NOT NULL REFERENCES products(id)
- quantity INTEGER NOT NULL CHECK (quantity > 0)
- Index on order_id

**Rationale:** Complete schema ensures data integrity, proper relationships, and query performance
**Impact:** Requires migration files, clear database setup process
**Updated Requirements:** Section 6.4-6.8

### API Key Configuration
**Date:** 2025-11-27
**Decision:** API key loaded from environment variable with default
**Details:**
- Environment variable: API_KEY
- Default value: "apitest" (for development)
- Configurable per environment
**Rationale:** Balances ease of development (default value) with production security (configurable)
**Impact:** .env.example documents API_KEY, deployment can override
**Updated Requirements:** Section 2.2, 9.5

### Transaction Boundary Definition
**Date:** 2025-11-27
**Decision:** Promo validation occurs BEFORE database transaction
**Sequence:**
1. Validate promo code (5 seconds max, concurrent file I/O)
2. Begin database transaction
3. Validate product IDs exist
4. Insert order record
5. Insert all order_items records
6. Commit transaction

**Rationale:** Avoids holding database connection during I/O operations (500ms-5s), improves connection pool efficiency
**Impact:** Slight TOCTOU window (promo could theoretically become invalid between validation and commit), but acceptable for this use case
**Updated Requirements:** Section 3.10, 6.12

### Error Response Format Standardisation
**Date:** 2025-11-27
**Decision:** Consistent JSON error structure with defined error codes
**Format:**
```json
{
  "error": "INVALID_PROMO_CODE",
  "message": "Promo code must appear in at least two coupon files",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Error Codes:**
- INVALID_JSON, MISSING_FIELD, INVALID_PROMO_CODE, INVALID_PROMO_LENGTH
- PRODUCT_NOT_FOUND, INVALID_QUANTITY, UNAUTHORIZED, FORBIDDEN, INTERNAL_ERROR

**Rationale:** Enables programmatic error handling, consistent client experience
**Impact:** All error responses follow this structure, correlation IDs for tracing
**Updated Requirements:** New Section 7 "Error Response Format"

### HTTP Status Code Clarification
**Date:** 2025-11-27
**Decision:** Clear distinction between 400 and 422
- **HTTP 400:** Malformed requests (invalid JSON, missing required fields)
- **HTTP 422:** Semantic validation failures (invalid promo, non-existent products, invalid quantities)

**Rationale:** Follows REST best practices - 400 for syntax errors, 422 for semantic errors
**Impact:** Error handling logic differentiates parse/structural errors from validation errors
**Updated Requirements:** Sections 1.10, 1.12

### Pagination Support
**Date:** 2025-11-27
**Decision:** Add optional pagination to GET /product endpoint
**Parameters:**
- limit (default 100) - maximum products to return
- offset (default 0) - number of products to skip

**Rationale:** Prevents memory/performance issues with large product catalogs
**Impact:** Repository layer implements LIMIT/OFFSET queries
**Updated Requirements:** Sections 1.1-1.2

### Case-Insensitive Promo Codes
**Date:** 2025-11-27
**Decision:** Change from case-sensitive to case-insensitive matching
**Rationale:** Better user experience, typical for promo codes (HAPPYHRS = happyhrs = HappyHrs)
**Impact:** Convert codes to uppercase/lowercase before comparison
**Updated Requirements:** Section 3.4 (changed from 3.6)

### UUID Version Selection
**Date:** 2025-11-27
**Decision:** Use UUIDv4 for orders and order_items
**Rationale:** Random UUIDs, standard library support, good collision resistance
**Impact:** Use google/uuid package or similar
**Updated Requirements:** Section 6.9

### Performance Requirement Quantification
**Date:** 2025-11-27 (Updated: 2025-11-30)
**Decision:** Define coupon file sizes as up to 1GB compressed per file (approximately 100 million codes each)
**Rationale:**
- Real-world coupon databases can be very large
- Makes performance requirements testable and realistic
- 1GB compressed files require efficient streaming and concurrent processing
**Impact:**
- Benchmark tests should use appropriately sized test files
- Performance targets remain the same (P95 < 500ms, 5s timeout)
- Streaming and early exit optimisations are critical for performance
**Updated Requirements:** Sections 4.9, 13.1

### Concurrent Validation Implementation Details
**Date:** 2025-11-27
**Decision:** Specify exact Go patterns for concurrent processing
**Details:**
- bufio.Scanner for line-by-line streaming
- compress/gzip for decompression
- context.WithCancel for cancellation
- context.WithTimeout for 5-second timeout
- Atomic operations or mutex for match counting
- Buffered channels for results

**Rationale:** Provides clear implementation guidance while allowing some flexibility
**Impact:** Implementation must use these patterns, testable with go test -race
**Updated Requirements:** Section 5 (renamed and expanded from Section 4)

### Logging and Observability Specifications
**Date:** 2025-11-27
**Decision:** Comprehensive logging and metrics specification
**Logging:**
- Structured JSON logging (zerolog or zap)
- Correlation IDs (UUIDv4) for all requests
- X-Correlation-ID response header
- Levels: ERROR, WARN, INFO, DEBUG

**Metrics:**
- Prometheus format at /metrics
- http_requests_total, http_request_duration_seconds
- promo_validation_duration_seconds, promo_validation_errors_total
- database_queries_total

**Rationale:** Production-ready observability from the start
**Impact:** Instrumentation throughout codebase, metrics library required
**Updated Requirements:** Sections 10, 12

### Configuration Environment Variables
**Date:** 2025-11-27
**Decision:** Complete list of environment variables with defaults
**Variables:**
- DATABASE_URL (required)
- SERVER_PORT (default 8080)
- API_KEY (default "apitest")
- COUPON_FILE_1, COUPON_FILE_2, COUPON_FILE_3 (defaults: data/*.gz)
- LOG_LEVEL (default INFO)
- PROMO_VALIDATION_TIMEOUT (default 5s)
- DB_MAX_OPEN_CONNS (default 25)
- DB_MAX_IDLE_CONNS (default 10)
- GRACEFUL_SHUTDOWN_TIMEOUT (default 30s)

**Rationale:** Clear configuration surface, sensible defaults for development, customisable for production
**Impact:** Config struct with validation, .env.example documents all
**Updated Requirements:** Section 9.5

### Timeout Specifications
**Date:** 2025-11-27
**Decision:** Define all timeout values
- HTTP read/write: 10 seconds
- HTTP idle: 120 seconds
- Database query: 10 seconds
- Promo validation: 5 seconds
- Graceful shutdown: 30 seconds

**Rationale:** Prevents resource exhaustion, defines clear behaviour under load
**Impact:** Context timeouts throughout codebase
**Updated Requirements:** Sections 13.5-13.8, 12.9

### Documentation and Seeding Requirements
**Date:** 2025-11-27
**Decision:** Add sections for database seeding and documentation standards
**Seeding:**
- Minimum 10 sample products
- Multiple categories (Waffle, Sandwich, Beverage, Dessert)
- make seed command (idempotent)
- Document sample valid promo codes

**Documentation:**
- README with standard sections
- All Makefile commands documented
- .env.example with descriptions
- Example API requests
- Australian English spelling

**Rationale:** Improves developer experience, enables quick start
**Impact:** Additional setup artifacts required
**Updated Requirements:** New Sections 16-17

### Order Lifecycle Scope
**Date:** 2025-11-27
**Decision:** Scope limited to order creation only
**Rationale:** Challenge focuses on concurrent promo validation, not full order management
**Future:** Order retrieval, modification, cancellation endpoints can be added later
**Impact:** No GET /order or PATCH /order endpoints in initial implementation
**Documented:** Implied by requirements scope

### Health Check Behaviour
**Date:** 2025-11-27
**Decision:** Health check returns 503 when database unavailable
**Details:**
- HTTP 200 + {"status": "healthy"} when healthy
- HTTP 503 + {"status": "unhealthy", "details": {...}} when unhealthy
- Check: SELECT 1 query to database

**Rationale:** Standard health check pattern, load balancers can remove unhealthy instances
**Impact:** May cause temporary unavailability during database maintenance
**Updated Requirements:** Sections 12.2-12.4

---

## Phase 2: Design Decisions

### Library Selection
**Date:** 2025-11-27
**Decisions:**
- **pgx/v5:** PostgreSQL driver (high performance, native Go, excellent pooling)
- **zerolog:** Structured logging (zero-allocation, fast JSON output)
- **golang-migrate:** Database migrations (industry standard, CLI + library)
- **google/uuid:** UUID generation (standard, UUIDv4 support)
- **prometheus/client_golang:** Metrics (de facto Go standard)
- **testcontainers-go:** Integration testing (real PostgreSQL in tests)

**Alternatives Considered:**
- database/sql + pq: Less performant than pgx
- zap: Comparable to zerolog, chose zerolog for simpler API
- sql-migrate: Less feature-complete than golang-migrate

**Impact:** Dependencies defined, build configuration set

### Package Structure
**Date:** 2025-11-27
**Decision:** Use internal/ for all application code
**Structure:**
- cmd/api/main.go - Entry point
- internal/config, handler, service, repository, model, middleware, router
- internal/coupon - Coupon validation package
- internal/database - Database connection pooling

**Rationale:** Follows Go best practices, clear separation of concerns, prevents external package imports
**Impact:** Code organisation, import paths

### Middleware Stack
**Date:** 2025-11-27
**Decision:** Middleware order: Recovery → CorrelationID → Logger → Auth
**Rationale:**
- Recovery first to catch all panics
- CorrelationID early for consistent tracing
- Logger after correlation ID available
- Auth last (only applied to specific routes)

**Impact:** Request processing pipeline

### Promo Validation Concurrency Pattern
**Date:** 2025-11-27
**Decision:** Use buffered channels with context cancellation
**Pattern:**
```go
results := make(chan bool, 3) // Buffered
ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
defer cancel()

// Launch 3 go-routines
for _, file := range files {
    go searchFile(ctx, file, code, results)
}

// Collect with early exit on 2 matches
matchCount := 0
for i := 0; i < 3; i++ {
    select {
    case found := <-results:
        if found {
            matchCount++
            if matchCount >= 2 {
                cancel() // Stop remaining
                return true, nil
            }
        }
    case <-ctx.Done():
        return false, ctx.Err()
    }
}
```

**Rationale:** Buffered channel prevents go-routine blocking, context enables clean cancellation
**Impact:** Performance optimization, resource cleanup guaranteed

### Repository Interface Pattern
**Date:** 2025-11-27
**Decision:** Use interfaces for all repositories
**Example:**
```go
type ProductRepository interface {
    ListProducts(ctx context.Context, limit, offset int) ([]model.Product, error)
    GetProductByID(ctx context.Context, id string) (*model.Product, error)
}
```

**Rationale:** Enables mocking in tests, dependency inversion principle
**Impact:** Testability, flexibility to swap implementations

### Transaction Management Pattern
**Date:** 2025-11-27
**Decision:** Service layer manages transactions, repository receives pgx.Tx
**Pattern:**
```go
// Service layer
tx, err := r.orderRepo.BeginTx(ctx)
defer tx.Rollback(ctx) // Rollback if not committed

err = r.orderRepo.CreateOrder(ctx, tx, order)
err = r.orderRepo.CreateOrderItems(ctx, tx, items)

return tx.Commit(ctx)
```

**Rationale:** Business logic controls transaction boundaries
**Impact:** Clear transaction scope, easier testing

### Error Handling Strategy
**Date:** 2025-11-27
**Decision:** Map internal errors to domain errors, never expose internals
**Pattern:**
- Service returns domain errors (InvalidPromoCode, ProductNotFound)
- Handler maps domain errors to HTTP status + error code
- Database errors wrapped and logged, generic 500 returned

**Rationale:** Security (no information leakage), consistent client experience
**Impact:** Error response format, logging requirements

### Logging Strategy
**Date:** 2025-11-27
**Decision:** Structured JSON logging with correlation IDs
**Format:**
```json
{
  "time": 1701234567,
  "level": "info",
  "service": "mini-kart",
  "correlation_id": "uuid",
  "method": "POST",
  "path": "/order",
  "status": 200,
  "duration_ms": 523,
  "message": "request completed"
}
```

**Rationale:** Machine-parseable, correlation enables request tracing
**Impact:** Observability, debugging capability

### Configuration Management Approach
**Date:** 2025-11-27
**Decision:** Single Config struct loaded at startup with validation
**Approach:**
- Load from environment variables
- Fallback to .env file for local development
- Validate all values at startup
- Fail fast with clear error messages

**Rationale:** Fail early, clear configuration surface, testable
**Impact:** Startup sequence, error handling

### Database Migration Strategy
**Date:** 2025-11-27
**Decision:** Use golang-migrate with numbered migrations
**Pattern:**
- 000001_create_products.up.sql / .down.sql
- 000002_create_orders.up.sql / .down.sql
- migrations/ directory
- make migrate-up / migrate-down commands

**Rationale:** Industry standard, versioned schema changes, rollback capability
**Impact:** Database setup process, CI/CD integration

---

## Design Review Fixes (Post-Critique)

### Critical Technical Issue Fixes
**Date:** 2025-11-27

**1. Middleware Ordering Correction**
- **Original:** Recovery → CorrelationID → Logger → Auth
- **Fixed:** Recovery → CorrelationID → Auth → Logger
- **Rationale:** Auth failures (401/403) should be properly logged, not caught by recovery
- **Impact:** Correct error handling and logging for authentication failures

**2. Promo Validator Race Condition Fix**
- **Original:** `matchCount := 0` (not thread-safe)
- **Fixed:** `var matchCount atomic.Int32` with atomic operations
- **Rationale:** Multiple go-routines can find matches simultaneously, need atomic counter
- **Impact:** Eliminates race condition in concurrent file processing

**3. Health Check Enhancement**
- **Added:** Connection pool stats (total/idle/acquired connections)
- **Added:** Warning when pool is exhausted
- **Added:** Coupon file accessibility checks
- **Rationale:** Provides operational visibility into system health beyond basic connectivity
- **Impact:** Better debugging, early warning of resource exhaustion

**4. Query Optimization for Product Validation**
- **Added:** Separate `ValidateProductsExist` query returning only IDs
- **Original:** `GetProductsByIDs` fetched full product details for validation
- **Rationale:** Validation only needs IDs, reduces network I/O
- **Impact:** Faster validation, reduced bandwidth usage

**5. Performance Target Clarification**
- **Added:** Explicit P95 designation for targets
- **Added:** Note that 5s timeout is safety net, not normal operation
- **Added:** Note that connection pool size requires load testing
- **Rationale:** Clarifies expectations, documents provisional values
- **Impact:** Clear performance requirements, acknowledges need for empirical testing

### Business Questions Documented
**Date:** 2025-11-27
**Decision:** Document outstanding business questions as "Outstanding Business Questions & TODOs" section in design.md

**Key Questions:**
1. Promo code usage limits (single-use vs multi-use) - HIGH priority
2. Connection pool sizing based on actual load - MEDIUM priority
3. Rate limiting requirements - HIGH priority for production
4. Performance optimization strategy - LOW priority (measure first)

**Rationale:** These require business stakeholder input or empirical measurement. Document as known gaps rather than blocking implementation.

**Impact:** Proceed to implementation with documented risks and decision points for future refinement

### Decision on Design Critique Feedback
**Date:** 2025-11-27
**Decision:** Accept Option A approach
- Fix critical technical issues immediately
- Document business questions for later clarification
- Proceed to task breakdown with documented caveats

**Rationale:** Technical issues are objective and can be fixed now. Business questions require stakeholder input or load testing that will happen during/after implementation.

**Impact:** Unblocks progress while maintaining awareness of open questions

---

## Phase 3: Implementation Task Breakdown

### Development Plan Scope Assessment
**Date:** 2025-11-28
**Assessment:** Complex Plan (target 350-600 lines)
**Justification:**
- Greenfield implementation with 30+ files across multiple subsystems
- 6+ subsystems: HTTP API, business logic, data access, concurrent file processing, middleware, observability
- 192 requirements across 17 sections
- Comprehensive test suite required (unit, integration, concurrent, E2E)
- Concurrent processing with go-routines and channels
- Clean architecture with multiple layers requiring coordination

**Alternatives Considered:**
- Standard plan (too small for this scope)
- Multiple separate plans (unnecessary - single cohesive system)

**Impact:** Created 8-phase plan with ~580 lines covering entire implementation lifecycle

### Phase Structure Decision
**Date:** 2025-11-28
**Decision:** 8-phase implementation plan
**Phases:**
1. Project Foundation & Database Schema (infrastructure setup)
2. Core Models & Repository Layer (data access)
3. Concurrent Promo Code Validation System (core challenge requirement)
4. Business Logic Layer (services with transaction management)
5. HTTP Layer (controllers and middleware stack)
6. Configuration, Observability & Application Bootstrap (logging, metrics, startup)
7. Integration & End-to-End Testing (testcontainers, E2E, performance)
8. Containerisation, Linting & Final Review (Docker, linting, security, verification)

**Rationale:**
- Logical progression from foundation to completion
- Each phase delivers reviewable value
- Testing integrated throughout phases, not deferred to end
- Concurrent validation (Phase 3) isolated early as it's the core technical challenge
- Final phase ensures all success criteria met before human review

**Impact:** Clear implementation roadmap with checkpoints after each phase

### Task Granularity Guidelines
**Date:** 2025-11-28
**Decision:** Tasks describe outcomes, not specific code implementations
**Examples of Appropriate Task Granularity:**
- ✅ "Implement domain models in internal/model/" (what to achieve)
- ✅ "Create database migration files with proper up/down scripts" (outcome)
- ✅ "Write unit tests for promo validation with race detector" (verification)
- ❌ "Add line 45: if (x && y) return true" (too prescriptive)
- ❌ "Improve the API" (too vague)

**Rationale:** Allows executing agent flexibility in implementation while maintaining clear success criteria

**Impact:** Tasks are specific enough to guide work but flexible enough for good technical decisions

### Testing Strategy Throughout Phases
**Date:** 2025-11-28
**Decision:** Integrate testing within each phase, not just at end
**Pattern:**
- Phase 2: Unit tests for repositories
- Phase 3: Concurrent validation tests with race detector
- Phase 4: Service layer unit tests with mocking
- Phase 5: HTTP layer tests (middleware, controllers)
- Phase 6: Observability verification (logs, metrics)
- Phase 7: Integration tests with testcontainers, E2E API tests, performance tests
- Phase 8: Full test suite verification

**Rationale:** Test-driven approach catches issues early, each phase is independently verifiable

**Impact:** Higher confidence in incremental progress, easier debugging

### Success Criteria Definition
**Date:** 2025-11-28
**Decision:** 15 measurable success criteria covering all aspects
**Categories:**
- Functional completeness (all API endpoints, promo validation, authentication)
- Performance targets (5s timeout, P95 500ms, test coverage >80%)
- Quality gates (all tests pass, zero linting errors, builds successfully)
- Observability (structured logs with correlation IDs, Prometheus metrics)
- Deployment (Docker image builds and runs)

**Rationale:** Objective, measurable criteria eliminate ambiguity about "done"

**Impact:** Clear definition of completion, no subjective "looks good enough"

### Makefile as Single Entry Point
**Date:** 2025-11-28
**Decision:** Create Makefile with all common operations
**Targets:** migrate-up, migrate-down, build, test-unit, test-integration, test-e2e, test-all, test-race, lint, run, docker-up, docker-down, dev
**Rationale:** Consistent interface for all operations, documented in single location, easy for new developers

**Impact:** Clear operational commands, reduced cognitive load

### Testcontainers for Integration Testing
**Date:** 2025-11-28
**Decision:** Use testcontainers-go for integration tests requiring PostgreSQL
**Approach:**
- Start PostgreSQL container automatically in tests
- Run migrations on test database
- Seed with test data
- Cleanup after tests

**Rationale:** Tests use real PostgreSQL without manual setup, consistent behaviour across environments

**Impact:** Higher test fidelity, no mocking of database layer in integration tests

### Docker Multi-Stage Build
**Date:** 2025-11-28
**Decision:** Use multi-stage Dockerfile (builder + runtime)
**Stages:**
1. Builder: golang:1.21+ alpine - compile binary with optimisation flags
2. Runtime: alpine:latest - copy binary + migrations only

**Rationale:** Smaller final image (<50MB), faster deployments, no build tools in production image

**Impact:** Efficient container image, improved security posture

### Outstanding Business Questions Handling
**Date:** 2025-11-28
**Decision:** Proceed with implementation while documenting open questions
**Open Questions:**
- Promo code usage limits/tracking (HIGH priority - affects TOCTOU risk)
- Connection pool sizing (MEDIUM - requires load testing)
- Rate limiting (HIGH - production readiness)

**Approach:** Implement without usage tracking initially, add TODO comments where business decisions needed

**Rationale:** Technical implementation can proceed independently, business logic can be added when clarified

**Impact:** Unblocked for implementation, clear markers for future enhancement

### Phase Checkpoint Strategy
**Date:** 2025-11-28
**Decision:** Every phase ends with critical self-review and human review checkpoint
**Pattern:**
- Perform critical self-review of changes and fix issues found
- STOP and wait for human review

**Rationale:** Catches issues early, prevents compounding mistakes, ensures alignment before proceeding

**Impact:** Higher quality, incremental user feedback, reduces rework

### Specification Language in Requirements
**Date:** 2025-11-28
**Decision:** Use RFC 2119 keywords (MUST/SHALL/SHOULD/MAY) in task requirements
**Usage:**
- MUST/SHALL: Mandatory requirements
- SHOULD: Strongly recommended but not mandatory
- MAY: Optional features
- MUST NOT: Explicit exclusions

**Rationale:** Removes ambiguity, clear distinction between mandatory and optional

**Impact:** Executing agent has clear priorities and constraints

### Git LFS for Coupon Files
**Date:** 2025-11-28 (Updated: 2025-11-30)
**Decision:** Use Git LFS to store large compressed coupon files and load them at server startup
**Files Tracked:**
- `*.gz` files (couponbase1.gz, couponbase2.gz, couponbase3.gz)
- Located in `data/coupons/` directory
- Each file up to 1GB compressed (approximately 100 million codes)

**Loading Mechanism:**
- Files are loaded directly from Git LFS tracked location at server startup
- No download logic required - files are already present after Git LFS checkout
- Server reads files from filesystem using streaming approach
- Files are NOT included in deployment artefacts

**Configuration:**
- `.gitattributes` file: `*.gz filter=lfs diff=lfs merge=lfs -text`
- Requires Git LFS installation: `git lfs install`
- Files tracked in `data/coupons/` directory

**Rationale:**
- Large binary files (up to 1GB each) would bloat Git repository without LFS
- Git LFS enables efficient cloning and version control for large files
- Maintains repository performance and reduces clone times
- Large data files cannot be part of deployment artefacts
- Files are loaded at startup from Git LFS tracked location

**Alternatives Considered:**
- Store files directly in Git (rejected: 1GB+ files would bloat repository significantly)
- Download at startup from external source (rejected: adds external dependency and complexity)
- Include in deployment artefact (rejected: 1GB+ files too large for deployment packages)
- Store outside repository (rejected: complicates deployment and version control)

**Impact:**
- Developers must have Git LFS installed (`git lfs install`)
- README must document Git LFS as prerequisite
- CI/CD pipelines must support Git LFS
- Initial setup includes `.gitattributes` configuration
- Coupon files stored efficiently in Git repository
- Server startup validates that files are accessible
- Deployment process must ensure Git LFS files are available in runtime environment
