# Mini-Kart API Server - Architectural Design

## Overview

Mini-Kart is a Go-based food ordering API server implementing OpenAPI 3.1 specification with a focus on concurrent promo code validation. The system uses PostgreSQL for persistence, pgx for database connections, zerolog for structured logging, and golang-migrate for schema management. The architecture emphasises clean separation of concerns, testability, and production-ready observability.

**Key Design Principles:**
- **Clean Architecture:** Clear separation between handlers, services, repositories, and models
- **Concurrency by Design:** Promo validation uses go-routines and channels for parallel file processing
- **Fail-Fast:** Configuration validation at startup, graceful error handling at runtime
- **Observable:** Structured logging with correlation IDs, Prometheus metrics, health checks
- **Testable:** Dependency injection, interface-based repositories, comprehensive test coverage

---

## System Architecture

### High-Level Architecture

```mermaid
graph TB
    Client[HTTP Client]

    subgraph "Mini-Kart API Server"
        Router[HTTP Router/Mux]
        Middleware[Middleware Layer]

        subgraph "Handler Layer"
            ProductHandler[Product Handler]
            OrderHandler[Order Handler]
            HealthHandler[Health Handler]
        end

        subgraph "Service Layer"
            ProductService[Product Service]
            OrderService[Order Service]
        end

        subgraph "Repository Layer"
            ProductRepo[Product Repository]
            OrderRepo[Order Repository]
        end

        subgraph "Internal Packages"
            CouponValidator[Coupon Validator]
            Logger[Zerolog Logger]
            Database[Database Pool]
        end

        Config[Configuration]
    end

    PostgreSQL[(PostgreSQL Database)]
    CouponFiles[Coupon Files<br/>data/*.gz]

    Client -->|HTTP Request| Router
    Router --> Middleware
    Middleware -->|Auth, Logging, Recovery| ProductHandler
    Middleware -->|Auth, Logging, Recovery| OrderHandler
    Middleware --> HealthHandler

    ProductHandler --> ProductService
    OrderHandler --> OrderService

    ProductService --> ProductRepo
    OrderService --> OrderRepo
    OrderService --> CouponValidator

    ProductRepo -->|pgx Pool| PostgreSQL
    OrderRepo -->|pgx Pool| PostgreSQL

    CouponValidator -->|Load & Validate| CouponFiles

    Middleware --> Logger
    ProductHandler --> Logger
    OrderHandler --> Logger

    Config -.->|Inject| ProductHandler
    Config -.->|Inject| OrderHandler
    Config -.->|Inject| CouponValidator
    Config -.->|Inject| Database

    classDef handler fill:#E8EAF6,stroke:#7986CB,color:#3F51B5
    classDef service fill:#F3E5F5,stroke:#BA68C8,color:#8E24AA
    classDef repository fill:#E0F2F1,stroke:#4DB6AC,color:#00897B
    classDef data fill:#E3F2FD,stroke:#64B5F6,color:#1976D2
    classDef internal fill:#FFF3E0,stroke:#FFB74D,color:#F57C00

    class ProductHandler,OrderHandler,HealthHandler handler
    class ProductService,OrderService service
    class ProductRepo,OrderRepo repository
    class PostgreSQL,CouponFiles data
    class CouponValidator,Logger,Database,Config internal
```

### Request Flow - Order Creation with Promo Validation

```mermaid
sequenceDiagram
    participant Client
    participant Router
    participant Middleware
    participant OrderHandler
    participant OrderService
    participant PromoValidator
    participant File1 as Coupon File 1
    participant File2 as Coupon File 2
    participant File3 as Coupon File 3
    participant OrderRepo
    participant DB as PostgreSQL

    Client->>Router: POST /order {items, couponCode}
    Router->>Middleware: Route request
    Middleware->>Middleware: Generate correlation ID
    Middleware->>Middleware: Verify API key
    Middleware->>OrderHandler: Authenticated request
    OrderHandler->>OrderService: CreateOrder(ctx, orderReq)

    alt Has Coupon Code
        OrderService->>PromoValidator: ValidatePromo(ctx, code)

        par Concurrent File Reading
            PromoValidator->>File1: Read & search (go-routine 1)
            PromoValidator->>File2: Read & search (go-routine 2)
            PromoValidator->>File3: Read & search (go-routine 3)
        end

        File1-->>PromoValidator: Found in file 1
        File2-->>PromoValidator: Found in file 2

        Note over PromoValidator: Match count = 2<br/>Cancel remaining go-routines

        PromoValidator-->>OrderService: Valid (found in 2+ files)
    end

    OrderService->>OrderRepo: BeginTransaction()
    OrderRepo->>DB: BEGIN
    OrderService->>OrderRepo: ValidateProducts(productIDs)
    OrderRepo->>DB: SELECT FROM products WHERE id IN (...)
    DB-->>OrderRepo: Product records
    OrderRepo-->>OrderService: All products exist

    OrderService->>OrderRepo: CreateOrder(order)
    OrderRepo->>DB: INSERT INTO orders
    OrderService->>OrderRepo: CreateOrderItems(items)
    OrderRepo->>DB: INSERT INTO order_items

    OrderService->>OrderRepo: CommitTransaction()
    OrderRepo->>DB: COMMIT
    DB-->>OrderRepo: Success
    OrderRepo-->>OrderService: Order created
    OrderService-->>OrderHandler: Order response
    OrderHandler->>Middleware: HTTP 200 + JSON
    Middleware->>Middleware: Log request metrics
    Middleware->>Client: Response
```

---

## Component Architecture

### Package Structure

```
mini-kart/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go              # Configuration loading and validation
│   ├── controller/
│   │   ├── product_handler.go     # Product HTTP handlers
│   │   ├── order_handler.go       # Order HTTP handlers
│   │   ├── health_handler.go      # Health check handler
│   │   └── metrics_handler.go     # Metrics endpoint handler
│   ├── service/
│   │   ├── product_service.go     # Product business logic
│   │   └── order_service.go       # Order business logic
│   ├── repository/
│   │   ├── repository.go          # Repository interfaces
│   │   ├── product_repo.go        # Product database operations
│   │   └── order_repo.go          # Order database operations
│   ├── model/
│   │   ├── product.go             # Product data structures
│   │   ├── order.go               # Order data structures
│   │   └── error.go               # Error response structures
│   └── middleware/
│       ├── auth.go                # API key authentication
│       ├── logger.go              # Request logging middleware
│       ├── recovery.go            # Panic recovery middleware
│       └── correlation.go         # Correlation ID injection
├── pkg/
│   └── promo/
│       ├── validator.go           # Promo code validation logic
│       └── validator_test.go      # Promo validation tests
├── migrations/
│   ├── 000001_create_products.up.sql
│   ├── 000001_create_products.down.sql
│   ├── 000002_create_orders.up.sql
│   ├── 000002_create_orders.down.sql
│   ├── 000003_create_order_items.up.sql
│   └── 000003_create_order_items.down.sql
├── data/
│   └── coupons/
│       ├── couponbase1.gz         # Coupon file 1 (Git LFS tracked, up to 1GB)
│       ├── couponbase2.gz         # Coupon file 2 (Git LFS tracked, up to 1GB)
│       └── couponbase3.gz         # Coupon file 3 (Git LFS tracked, up to 1GB)
├── testdata/
│   ├── test_coupons1.gz           # Test coupon files
│   ├── test_coupons2.gz
│   └── test_coupons3.gz
├── scripts/
│   └── seed_products.sql          # Sample product data
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── .env.example
└── go.mod
```

### Coupon File Management

**Storage Mechanism:**
- Coupon files are stored in Git LFS (Large File Storage) due to their size (up to 1GB compressed each)
- Files are tracked in the `data/coupons/` directory
- Git LFS configuration in `.gitattributes`: `*.gz filter=lfs diff=lfs merge=lfs -text`

**Loading Mechanism:**
- Files are loaded directly from the Git LFS tracked location at server startup
- No download logic required - files are present after `git lfs pull` or repository clone
- Server validates file accessibility during startup using `os.Stat()`
- Server fails fast at startup if any configured coupon file is missing or unreadable

**File Specifications:**
- Format: Gzip compressed text files
- Encoding: UTF-8
- Structure: One promo code per line
- Size: Up to 1GB compressed (approximately 100 million codes)
- Comments: Lines starting with `#` or empty lines are ignored

**Deployment Considerations:**
- Large data files are NOT included in deployment artefacts
- Files are managed via Git LFS and must be available in the runtime environment
- CI/CD pipelines must support Git LFS to access files
- Developers must have Git LFS installed: `git lfs install`

---

## Component Specifications

### 1. Configuration Layer (`internal/config`)

**Purpose:** Load, validate, and provide configuration to all components.

**Key Responsibilities:**
- Load from environment variables with .env fallback
- Validate all required configuration at startup
- Provide type-safe configuration struct
- Fail fast with clear error messages

**Configuration Structure:**
```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Coupon   CouponConfig
    Logging  LoggingConfig
    Auth     AuthConfig
}

type ServerConfig struct {
    Port                    int           // Default: 8080
    ReadTimeout             time.Duration // Default: 10s
    WriteTimeout            time.Duration // Default: 10s
    IdleTimeout             time.Duration // Default: 120s
    GracefulShutdownTimeout time.Duration // Default: 30s
}

type DatabaseConfig struct {
    URL            string // Required: postgres://...
    MaxOpenConns   int    // Default: 25
    MaxIdleConns   int    // Default: 10
    ConnMaxLife    time.Duration // Default: 1h
    ConnMaxIdle    time.Duration // Default: 30m
    QueryTimeout   time.Duration // Default: 10s
}

type CouponConfig struct {
    File1              string        // Default: data/coupons/couponbase1.gz
    File2              string        // Default: data/coupons/couponbase2.gz
    File3              string        // Default: data/coupons/couponbase3.gz
    ValidationTimeout  time.Duration // Default: 5s
}

type LoggingConfig struct {
    Level  string // Default: INFO (DEBUG, INFO, WARN, ERROR)
}

type AuthConfig struct {
    APIKey string // Default: apitest
}
```

**Validation Rules:**
- DATABASE_URL must be valid PostgreSQL connection string
- Server port must be between 1024-65535
- All timeout values must be positive
- Coupon files must exist and be readable
- Log level must be valid (DEBUG, INFO, WARN, ERROR)

---

### 2. Handler Layer (`internal/handler`)

**Purpose:** Handle HTTP requests, validate input, call services, return responses.

**Design Pattern:** Each handler is a struct with injected dependencies.

#### Product Handler

```go
type ProductHandler struct {
    service ProductService
    logger  zerolog.Logger
}

// GET /api/products?limit=100&offset=0
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request)

// GET /api/products/{id}
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request)
```

**Responsibilities:**
- Parse query parameters (limit, offset for pagination)
- Parse path parameters (productId)
- Call service layer methods
- Format responses as JSON
- Handle service errors and map to HTTP status codes
- Log requests and responses

#### Order Handler

```go
type OrderHandler struct {
    service OrderService
    logger  zerolog.Logger
}

// POST /api/orders
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request)

// GET /api/orders/{id}
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request)
```

**Responsibilities:**
- Parse and validate JSON request body
- Extract correlation ID from context
- Call order service
- Return order response or error
- Log validation outcomes

#### Health Handler

**Note:** Health check is implemented in `internal/handler/handler.go` as a simple handler function.

```go
// GET /health
func HealthHandler(w http.ResponseWriter, r *http.Request)
```

**Response Format:**
```json
{
  "status": "healthy",
  "checks": {
    "database": "ok"
  }
}
```

Or when unhealthy:
```json
{
  "status": "unhealthy",
  "checks": {
    "database": "connection timeout"
  }
}
```

**Note:** Prometheus metrics endpoint was planned but not implemented in current version.

---

### 3. Middleware Layer (`internal/middleware`)

**Purpose:** Cross-cutting concerns applied to all requests.

**Middleware Stack Order:**
```
Recovery → CorrelationID → Auth (conditionally) → Logger
```

**Rationale:**
- Recovery first to catch all panics (including from other middleware)
- CorrelationID early so all logs have it
- Auth before logger so auth failures (401/403) are properly logged
- Logger last to capture complete request/response cycle

#### Correlation ID Middleware

```go
func CorrelationID(next http.Handler) http.Handler
```

**Behavior:**
- Generate UUIDv4 for each request
- Store in request context
- Add `X-Correlation-ID` response header
- Pass to downstream handlers

#### Authentication Middleware

```go
func RequireAPIKey(apiKey string) func(http.Handler) http.Handler
```

**Behavior:**
- Extract `api_key` header
- Compare with configured API key
- Return 401 if missing, 403 if invalid
- Pass to next handler if valid

**Applied To:** POST /order endpoint only (not applied globally)

#### Logging Middleware

```go
func RequestLogger(logger zerolog.Logger) func(http.Handler) http.Handler
```

**Behavior:**
- Log request: method, path, correlation_id
- Wrap response writer to capture status code
- Log response: status, duration, correlation_id
- Include error details for 4xx/5xx responses

**Log Format:**
```json
{
  "time": "2025-11-27T10:30:00Z",
  "level": "info",
  "service": "mini-kart",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "POST",
  "path": "/order",
  "status": 200,
  "duration_ms": 523,
  "message": "request completed"
}
```

#### Recovery Middleware

```go
func Recovery(logger zerolog.Logger) func(http.Handler) http.Handler
```

**Behavior:**
- Catch panics in handlers
- Log panic with stack trace
- Return HTTP 500 with INTERNAL_ERROR code
- Prevent server crash

---

### 4. Service Layer (`internal/service`)

**Purpose:** Business logic, orchestration, validation.

#### Product Service Interface

```go
type ProductService interface {
    ListProducts(ctx context.Context, limit, offset int) ([]model.Product, error)
    GetProductByID(ctx context.Context, id string) (*model.Product, error)
}
```

**Implementation:**
```go
type productService struct {
    repo   repository.ProductRepository
    logger zerolog.Logger
}

func NewProductService(repo repository.ProductRepository, logger zerolog.Logger) ProductService
```

**Responsibilities:**
- Validate pagination parameters (limit > 0, offset >= 0)
- Call repository methods
- Log errors with context
- Return domain errors (not database errors)

#### Order Service Interface

```go
type OrderService interface {
    CreateOrder(ctx context.Context, req model.OrderRequest) (*model.OrderResponse, error)
}
```

**Implementation:**
```go
type orderService struct {
    orderRepo repository.OrderRepository
    productRepo repository.ProductRepository
    promoValidator promo.Validator
    logger zerolog.Logger
}

func NewOrderService(
    orderRepo repository.OrderRepository,
    productRepo repository.ProductRepository,
    promoValidator promo.Validator,
    logger zerolog.Logger,
) OrderService
```

**CreateOrder Flow:**

1. **Validate Input:**
   - Items array not empty
   - All quantities > 0
   - Product IDs not empty

2. **Validate Promo Code (if provided):**
   - Call promo validator with 5s timeout
   - Return 422 if invalid

3. **Begin Transaction:**
   - Start database transaction

4. **Validate Products Exist:**
   - Query all product IDs in single SELECT
   - Return 422 if any missing

5. **Create Order:**
   - Generate UUIDv4 for order
   - Set created_at, updated_at to now (UTC)
   - Insert order record

6. **Create Order Items:**
   - Generate UUIDv4 for each item
   - Insert all items

7. **Commit Transaction:**
   - Commit if all successful
   - Rollback on any error

8. **Build Response:**
   - Fetch full product details
   - Return order with items and products

---

### 5. Repository Layer (`internal/repository`)

**Purpose:** Database access, SQL query execution.

**Design Pattern:** Interface-based repositories for testability.

#### Repository Interfaces

```go
type ProductRepository interface {
    ListProducts(ctx context.Context, limit, offset int) ([]model.Product, error)
    GetProductByID(ctx context.Context, id string) (*model.Product, error)
    GetProductsByIDs(ctx context.Context, ids []string) ([]model.Product, error)
}

type OrderRepository interface {
    CreateOrder(ctx context.Context, tx pgx.Tx, order *model.Order) error
    CreateOrderItems(ctx context.Context, tx pgx.Tx, items []model.OrderItem) error
    BeginTx(ctx context.Context) (pgx.Tx, error)
}
```

#### Product Repository Implementation

```go
type productRepository struct {
    pool   *pgxpool.Pool
    logger zerolog.Logger
}

func NewProductRepository(pool *pgxpool.Pool, logger zerolog.Logger) ProductRepository
```

**Key Queries:**

```sql
-- ListProducts with pagination
SELECT id, name, price, category
FROM products
ORDER BY name
LIMIT $1 OFFSET $2

-- GetProductByID
SELECT id, name, price, category
FROM products
WHERE id = $1

-- ValidateProductsExist (used in order validation - optimised)
-- Returns only IDs to verify existence, not full details
SELECT id
FROM products
WHERE id = ANY($1)

-- GetProductsByIDs (used to fetch full details for response)
SELECT id, name, price, category
FROM products
WHERE id = ANY($1)
```

**Error Handling:**
- `pgx.ErrNoRows` → return nil, nil (not found)
- Connection errors → return nil, wrapped error
- Other errors → return nil, wrapped error with context

#### Order Repository Implementation

```go
type orderRepository struct {
    pool   *pgxpool.Pool
    logger zerolog.Logger
}

func NewOrderRepository(pool *pgxpool.Pool, logger zerolog.Logger) OrderRepository
```

**Transaction Pattern:**
```go
func (r *orderRepository) BeginTx(ctx context.Context) (pgx.Tx, error) {
    return r.pool.Begin(ctx)
}
```

**Key Operations:**

```sql
-- CreateOrder
INSERT INTO orders (id, coupon_code, created_at, updated_at)
VALUES ($1, $2, $3, $4)

-- CreateOrderItems
INSERT INTO order_items (id, order_id, product_id, quantity)
VALUES ($1, $2, $3, $4)
```

**Transaction Management:**
- Service layer begins transaction
- Repository receives `pgx.Tx` for transaction operations
- Service layer commits or rolls back

---

### 6. Coupon Validation Package (`internal/coupon`)

**Purpose:** Concurrent coupon code validation against pre-loaded coupon sets.

**Interface:**
```go
type Validator interface {
    Validate(ctx context.Context, code string) (bool, error)
    Close() error
}
```

**Implementation:**
```go
type Validator struct {
    sets   []*CouponSet
    config ValidatorConfig
    logger zerolog.Logger
}

func NewValidator(ctx context.Context, config ValidatorConfig, loader Loader, logger zerolog.Logger) (*Validator, error)
```

**Note:** The actual implementation loads coupon files into memory-efficient sets at startup rather than streaming files on each validation request. This provides faster validation times and better resource utilisation.

**Validation Algorithm:**

```mermaid
flowchart TD
    Start([Start ValidatePromo]) --> CheckLength{Length<br/>8-10 chars?}
    CheckLength -->|No| ReturnInvalid[Return false,<br/>INVALID_PROMO_LENGTH]
    CheckLength -->|Yes| NormaliseCode[Normalise code<br/>to uppercase]
    NormaliseCode --> CreateContext[Create context<br/>with 5s timeout<br/>and cancellation]
    CreateContext --> InitChannels[Create results channel<br/>buffered size 3]
    InitChannels --> LaunchGoroutines[Launch 3 go-routines<br/>for each file]

    LaunchGoroutines --> File1[Go-routine 1:<br/>Search File 1]
    LaunchGoroutines --> File2[Go-routine 2:<br/>Search File 2]
    LaunchGoroutines --> File3[Go-routine 3:<br/>Search File 3]

    File1 --> OpenFile1[Open & decompress<br/>couponbase1.gz]
    File2 --> OpenFile2[Open & decompress<br/>couponbase2.gz]
    File3 --> OpenFile3[Open & decompress<br/>couponbase3.gz]

    OpenFile1 --> StreamRead1[Stream with<br/>bufio.Scanner]
    OpenFile2 --> StreamRead2[Stream with<br/>bufio.Scanner]
    OpenFile3 --> StreamRead3[Stream with<br/>bufio.Scanner]

    StreamRead1 --> Match1{Code<br/>found?}
    StreamRead2 --> Match2{Code<br/>found?}
    StreamRead3 --> Match3{Code<br/>found?}

    Match1 -->|Yes| SendResult1[Send true to<br/>results channel]
    Match2 -->|Yes| SendResult2[Send true to<br/>results channel]
    Match3 -->|Yes| SendResult3[Send true to<br/>results channel]

    Match1 -->|No| SendResult1No[Send false to<br/>results channel]
    Match2 -->|No| SendResult2No[Send false to<br/>results channel]
    Match3 -->|No| SendResult3No[Send false to<br/>results channel]

    SendResult1 --> Collect
    SendResult2 --> Collect
    SendResult3 --> Collect
    SendResult1No --> Collect
    SendResult2No --> Collect
    SendResult3No --> Collect

    Collect[Main go-routine:<br/>Collect results]

    Collect --> CountMatches{Match count<br/>>= 2?}
    CountMatches -->|Yes| CancelContext[Cancel context<br/>stop remaining<br/>go-routines]
    CountMatches -->|No| MoreResults{All 3 results<br/>received?}

    MoreResults -->|No| Timeout{Context<br/>timeout?}
    Timeout -->|Yes| ReturnTimeout[Return false,<br/>timeout error]
    Timeout -->|No| Collect

    MoreResults -->|Yes| FinalCheck{Match count<br/>>= 2?}
    FinalCheck -->|Yes| CancelContext
    FinalCheck -->|No| ReturnInvalidPromo[Return false,<br/>INVALID_PROMO_CODE]

    CancelContext --> ReturnValid[Return true, nil]

    ReturnInvalid --> End([End])
    ReturnValid --> End
    ReturnInvalidPromo --> End
    ReturnTimeout --> End

    classDef decision fill:#FFF3E0,stroke:#FFB74D,color:#F57C00
    classDef process fill:#E0F2F1,stroke:#4DB6AC,color:#00897B
    classDef goroutine fill:#E8EAF6,stroke:#7986CB,color:#3F51B5
    classDef result fill:#F3E5F5,stroke:#BA68C8,color:#8E24AA

    class CheckLength,Match1,Match2,Match3,CountMatches,MoreResults,Timeout,FinalCheck decision
    class NormaliseCode,CreateContext,InitChannels,OpenFile1,OpenFile2,OpenFile3,StreamRead1,StreamRead2,StreamRead3,CancelContext process
    class File1,File2,File3 goroutine
    class ReturnValid,ReturnInvalid,ReturnInvalidPromo,ReturnTimeout result
```

**Key Implementation Details:**

1. **Case-Insensitive Matching:**
   ```go
   code = strings.ToUpper(strings.TrimSpace(code))
   ```

2. **Concurrent File Reading:**
   ```go
   func searchFile(ctx context.Context, filepath, code string, results chan<- bool) {
       defer func() {
           // Ensure channel send even on panic
           if r := recover(); r != nil {
               results <- false
           }
       }()

       file, err := os.Open(filepath)
       if err != nil {
           results <- false
           return
       }
       defer file.Close()

       gzReader, err := gzip.NewReader(file)
       if err != nil {
           results <- false
           return
       }
       defer gzReader.Close()

       scanner := bufio.Scanner(gzReader)
       for scanner.Scan() {
           select {
           case <-ctx.Done():
               results <- false
               return
           default:
               line := strings.ToUpper(strings.TrimSpace(scanner.Text()))
               // Skip comments and empty lines
               if line == "" || strings.HasPrefix(line, "#") {
                   continue
               }
               if line == code {
                   results <- true
                   return
               }
           }
       }
       results <- false
   }
   ```

3. **Result Collection with Early Exit:**
   ```go
   var matchCount atomic.Int32  // Thread-safe counter
   resultsReceived := 0

   for resultsReceived < 3 {
       select {
       case found := <-results:
           resultsReceived++
           if found {
               newCount := matchCount.Add(1)
               if newCount >= 2 {
                   cancel() // Stop remaining go-routines
                   return true, nil
               }
           }
       case <-ctx.Done():
           return false, fmt.Errorf("promo validation timeout")
       }
   }

   return matchCount.Load() >= 2, nil
   ```

4. **Resource Cleanup:**
   - `defer file.Close()` in each go-routine
   - `defer gzReader.Close()` for gzip reader
   - Context cancellation stops go-routines
   - Buffered channel prevents go-routine leaks

---

## Data Models

### Product Model

```go
type Product struct {
    ID       string  `json:"id" db:"id"`
    Name     string  `json:"name" db:"name"`
    Price    float64 `json:"price" db:"price"`
    Category string  `json:"category" db:"category"`
}
```

### Order Models

```go
type Order struct {
    ID         uuid.UUID  `json:"id" db:"id"`
    CouponCode *string    `json:"couponCode,omitempty" db:"coupon_code"`
    CreatedAt  time.Time  `json:"createdAt" db:"created_at"`
    UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

type OrderItem struct {
    ID        uuid.UUID `json:"-" db:"id"`
    OrderID   uuid.UUID `json:"-" db:"order_id"`
    ProductID string    `json:"productId" db:"product_id"`
    Quantity  int       `json:"quantity" db:"quantity"`
}

type OrderRequest struct {
    CouponCode *string           `json:"couponCode,omitempty"`
    Items      []OrderItemRequest `json:"items"`
}

type OrderItemRequest struct {
    ProductID string `json:"productId"`
    Quantity  int    `json:"quantity"`
}

type OrderResponse struct {
    ID       uuid.UUID   `json:"id"`
    Items    []OrderItem `json:"items"`
    Products []Product   `json:"products"`
}
```

### Error Model

```go
type ErrorResponse struct {
    Error         string `json:"error"`
    Message       string `json:"message"`
    CorrelationID string `json:"correlation_id,omitempty"`
}

// Standard error codes
const (
    ErrCodeInvalidJSON       = "INVALID_JSON"
    ErrCodeMissingField      = "MISSING_FIELD"
    ErrCodeInvalidPromoCode  = "INVALID_PROMO_CODE"
    ErrCodeInvalidPromoLength = "INVALID_PROMO_LENGTH"
    ErrCodeProductNotFound   = "PRODUCT_NOT_FOUND"
    ErrCodeInvalidQuantity   = "INVALID_QUANTITY"
    ErrCodeUnauthorized      = "UNAUTHORIZED"
    ErrCodeForbidden         = "FORBIDDEN"
    ErrCodeInternalError     = "INTERNAL_ERROR"
)
```

---

## Database Schema

### Migration Files

**000001_create_products.up.sql:**
```sql
CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    price DECIMAL(10,2) NOT NULL CHECK (price >= 0),
    category TEXT NOT NULL
);

CREATE INDEX idx_products_category ON products(category);
```

**000001_create_products.down.sql:**
```sql
DROP INDEX IF EXISTS idx_products_category;
DROP TABLE IF EXISTS products;
```

**000002_create_orders.up.sql:**
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    coupon_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_created_at ON orders(created_at DESC);
```

**000002_create_orders.down.sql:**
```sql
DROP INDEX IF EXISTS idx_orders_created_at;
DROP TABLE IF EXISTS orders;
```

**000003_create_order_items.up.sql:**
```sql
CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0)
);

CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
```

**000003_create_order_items.down.sql:**
```sql
DROP INDEX IF EXISTS idx_order_items_product_id;
DROP INDEX IF EXISTS idx_order_items_order_id;
DROP TABLE IF EXISTS order_items;
```

---

## Testing Strategy

### Unit Tests

**Location:** Alongside source files (`*_test.go`)

**Coverage Targets:**
- Service layer: 100% (pure business logic)
- Promo validator: 100% (concurrent logic)
- Middleware: 80%+ (HTTP-heavy code)
- Handlers: 70%+ (integration-like)

**Testing Patterns:**

1. **Table-Driven Tests:**
```go
func TestOrderService_CreateOrder(t *testing.T) {
    tests := []struct {
        name          string
        request       model.OrderRequest
        mockSetup     func(*mocks.MockOrderRepo, *mocks.MockPromoValidator)
        expectedError string
        expectedOrder *model.OrderResponse
    }{
        {
            name: "success with valid promo",
            request: model.OrderRequest{
                CouponCode: stringPtr("HAPPYHRS"),
                Items: []model.OrderItemRequest{
                    {ProductID: "1", Quantity: 2},
                },
            },
            mockSetup: func(repo *mocks.MockOrderRepo, promo *mocks.MockPromoValidator) {
                promo.On("ValidatePromo", mock.Anything, "HAPPYHRS").Return(true, nil)
                repo.On("BeginTx", mock.Anything).Return(&mockTx{}, nil)
                // ... more mocking
            },
            expectedError: "",
            expectedOrder: &model.OrderResponse{...},
        },
        // More test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            mockOrderRepo := new(mocks.MockOrderRepo)
            mockPromoValidator := new(mocks.MockPromoValidator)
            tt.mockSetup(mockOrderRepo, mockPromoValidator)

            service := NewOrderService(mockOrderRepo, nil, mockPromoValidator, logger)

            // Act
            result, err := service.CreateOrder(context.Background(), tt.request)

            // Assert
            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expectedOrder, result)
            }

            mockOrderRepo.AssertExpectations(t)
            mockPromoValidator.AssertExpectations(t)
        })
    }
}
```

2. **Promo Validator Concurrent Tests:**
```go
func TestPromoValidator_Concurrent(t *testing.T) {
    validator := setupTestValidator(t)

    // Run validation from multiple go-routines
    const numGoroutines = 50
    results := make(chan bool, numGoroutines)

    for i := 0; i < numGoroutines; i++ {
        go func() {
            valid, err := validator.ValidatePromo(context.Background(), "HAPPYHRS")
            assert.NoError(t, err)
            results <- valid
        }()
    }

    // Collect results
    for i := 0; i < numGoroutines; i++ {
        valid := <-results
        assert.True(t, valid)
    }
}
```

3. **Race Condition Detection:**
```bash
go test -race ./...
```

### Integration Tests

**Location:** `test/integration/`

**Tools:**
- `testcontainers-go` for PostgreSQL
- `httptest` for HTTP testing

**Example:**
```go
func TestOrderAPI_CreateOrder_Integration(t *testing.T) {
    // Start PostgreSQL container
    ctx := context.Background()
    pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image: "postgres:16-alpine",
            Env: map[string]string{
                "POSTGRES_DB":       "testdb",
                "POSTGRES_USER":     "test",
                "POSTGRES_PASSWORD": "test",
            },
            ExposedPorts: []string{"5432/tcp"},
            WaitingFor:   wait.ForListeningPort("5432/tcp"),
        },
        Started: true,
    })
    require.NoError(t, err)
    defer pgContainer.Terminate(ctx)

    // Get connection string
    host, _ := pgContainer.Host(ctx)
    port, _ := pgContainer.MappedPort(ctx, "5432")
    connString := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, port.Port())

    // Run migrations
    runMigrations(t, connString)

    // Seed test data
    seedTestData(t, connString)

    // Create application with test config
    app := setupTestApp(t, connString)

    // Test request
    reqBody := `{"items":[{"productId":"1","quantity":2}],"couponCode":"HAPPYHRS"}`
    req := httptest.NewRequest("POST", "/order", strings.NewReader(reqBody))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("api_key", "apitest")

    w := httptest.NewRecorder()
    app.ServeHTTP(w, req)

    // Assert
    assert.Equal(t, http.StatusOK, w.Code)

    var response model.OrderResponse
    err = json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.NotEmpty(t, response.ID)
}
```

### Benchmark Tests

**Purpose:** Validate promo validation performance

**Location:** `pkg/promo/validator_bench_test.go`

```go
func BenchmarkPromoValidator_ValidatePromo(b *testing.B) {
    validator := setupBenchValidator(b) // Uses actual 10MB test files

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := validator.ValidatePromo(context.Background(), "HAPPYHRS")
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}

// Target: < 500ms per validation with 10MB files
```

---

## Observability

### Logging

**Library:** `github.com/rs/zerolog`

**Configuration:**
```go
func setupLogger(cfg config.LoggingConfig) zerolog.Logger {
    zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

    level, _ := zerolog.ParseLevel(strings.ToLower(cfg.Level))
    zerolog.SetGlobalLevel(level)

    return zerolog.New(os.Stdout).
        With().
        Timestamp().
        Str("service", "mini-kart").
        Logger()
}
```

**Structured Log Format:**
```json
{
  "time": 1701234567,
  "level": "info",
  "service": "mini-kart",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "POST",
  "path": "/order",
  "status": 200,
  "duration_ms": 523,
  "message": "request completed"
}
```

### Metrics

**Library:** `github.com/prometheus/client_golang`

**Metrics Defined:**

```go
var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )

    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
        },
        []string{"method", "path"},
    )

    promoValidationDuration = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "promo_validation_duration_seconds",
            Help:    "Promo code validation duration in seconds",
            Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
        },
    )

    promoValidationErrors = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "promo_validation_errors_total",
            Help: "Total number of promo validation errors",
        },
    )

    databaseQueriesTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "database_queries_total",
            Help: "Total number of database queries",
        },
        []string{"query_type", "status"},
    )
)
```

**Registration:**
```go
func registerMetrics() {
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpRequestDuration)
    prometheus.MustRegister(promoValidationDuration)
    prometheus.MustRegister(promoValidationErrors)
    prometheus.MustRegister(databaseQueriesTotal)
}
```

### Health Checks

**Endpoint:** GET /health

**Check Logic:**
```go
func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()

    checks := make(map[string]interface{})
    healthy := true

    // Database connectivity check
    var result int
    err := h.db.QueryRow(ctx, "SELECT 1").Scan(&result)
    if err != nil {
        checks["database"] = map[string]string{
            "status": "error",
            "error": err.Error(),
        }
        healthy = false
    } else {
        // Check connection pool stats
        stat := h.db.Stat()
        checks["database"] = map[string]interface{}{
            "status": "ok",
            "total_conns": stat.TotalConns(),
            "idle_conns": stat.IdleConns(),
            "acquired_conns": stat.AcquiredConns(),
        }

        // Warn if connection pool is exhausted
        if stat.IdleConns() == 0 && stat.AcquiredConns() == stat.MaxConns() {
            checks["database"].(map[string]interface{})["warning"] = "connection pool exhausted"
        }
    }

    // Coupon files accessibility check
    couponFiles := []string{
        h.config.Coupon.File1,
        h.config.Coupon.File2,
        h.config.Coupon.File3,
    }

    fileChecks := make([]map[string]string, 0, 3)
    for i, filepath := range couponFiles {
        fileCheck := map[string]string{
            "file": filepath,
        }

        if _, err := os.Stat(filepath); err != nil {
            fileCheck["status"] = "error"
            fileCheck["error"] = err.Error()
            healthy = false
        } else {
            fileCheck["status"] = "ok"
        }
        fileChecks = append(fileChecks, fileCheck)
    }
    checks["coupon_files"] = fileChecks

    response := map[string]interface{}{
        "status": "healthy",
        "checks": checks,
    }

    statusCode := http.StatusOK
    if !healthy {
        response["status"] = "unhealthy"
        statusCode = http.StatusServiceUnavailable
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(response)
}
```

---

## Deployment Architecture

### Docker Compose (Local Development)

**docker-compose.yml:**
```yaml
version: '3.9'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: minikart
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  server:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres:5432/minikart?sslmode=disable
      SERVER_PORT: 8080
      API_KEY: apitest
      LOG_LEVEL: DEBUG
    volumes:
      - ./data:/app/data:ro
    depends_on:
      postgres:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

volumes:
  postgres_data:
```

### Dockerfile (Multi-Stage Build)

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api cmd/api/main.go

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/server .

# Copy migrations
COPY --from=builder /build/migrations ./migrations

# Copy coupon files
COPY --from=builder /build/data ./data

# Change ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --spider -q http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/server"]
```

### .dockerignore

```
.git
.env
.env.local
*.md
docs/
test/
testdata/
.vscode/
.idea/
*.log
bin/
coverage.html
```

---

## Key Libraries and Dependencies

### Core Dependencies

```go
require (
    github.com/jackc/pgx/v5 v5.5.0
    github.com/rs/zerolog v1.31.0
    github.com/golang-migrate/migrate/v4 v4.17.0
    github.com/google/uuid v1.5.0
    github.com/prometheus/client_golang v1.18.0
    github.com/joho/godotenv v1.5.1
)
```

### Testing Dependencies

```go
require (
    github.com/stretchr/testify v1.8.4
    github.com/testcontainers/testcontainers-go v0.27.0
    github.com/testcontainers/testcontainers-go/modules/postgres v0.27.0
)
```

### Library Choices Rationale

| Library | Purpose | Rationale |
|---------|---------|-----------|
| pgx/v5 | PostgreSQL driver | Native Go driver, high performance, excellent connection pooling |
| zerolog | Structured logging | Zero-allocation, fast, JSON output, simple API |
| golang-migrate | Database migrations | Industry standard, supports up/down migrations, CLI + library |
| google/uuid | UUID generation | Standard library, UUIDv4 support |
| prometheus/client_golang | Metrics | De facto standard for Go metrics |
| godotenv | .env file loading | Simple .env file support for local development |
| testify | Testing assertions | Clean assertion syntax, mock support |
| testcontainers-go | Integration testing | Real PostgreSQL in tests, isolated test environments |

---

## Security Considerations

### Input Validation
- All request bodies validated before processing
- SQL injection prevented via parameterised queries
- No user input in file paths (coupon files configured at startup)

### Authentication
- API key required for POST /order
- No authentication for GET /product (public catalog)
- Health and metrics endpoints unauthenticated (internal)

### Error Handling
- Internal errors never exposed to clients
- Stack traces logged but not returned in responses
- Generic HTTP 500 for unexpected errors

### Data Protection
- API keys not logged
- Database credentials loaded from environment
- No sensitive data in logs (PII, credentials)

### Resource Limits
- HTTP timeouts prevent slowloris attacks
- Database connection pool limits prevent resource exhaustion
- Promo validation timeout prevents DoS via slow file operations
- Context cancellation prevents go-routine leaks

---

## Performance Targets

| Operation | Target | Measurement |
|-----------|--------|-------------|
| Promo validation | < 500ms | P95 with 1M codes per file (happy path) |
| Promo validation timeout | 5s | Maximum allowed time (prevents hung requests) |
| GET /product | < 50ms | P95 without network latency |
| POST /order (no promo) | < 100ms | P95 |
| POST /order (with promo) | < 600ms | P95 (500ms promo + 100ms order processing) |
| Concurrent requests | 100+ | Simultaneous POST /order without errors |
| Database connections | 25 max | Initial pool limit (requires load testing to optimize) |

**Performance Target Notes:**
- P95 = 95th percentile (5% of requests may be slower)
- 5s timeout is safety net, not expected normal operation
- Connection pool size (25) is provisional and requires load testing to determine optimal value based on:
  - Expected requests per second
  - Average query duration
  - Target latency under load
  - Database server capacity

---

## Outstanding Business Questions & TODOs

The following items require clarification before production deployment. They are documented here as known gaps that should be addressed based on actual requirements, load testing, and business rules.

### 1. Promo Code Business Logic (PRIORITY: HIGH)

**Current Implementation:** Promo codes are validated for existence in 2+ files and stored with orders, but no discount is applied.

**Questions Needing Answers:**
- **Usage Limits:** Are promo codes single-use, multi-use, or rate-limited?
- **Concurrent Usage:** Can multiple users use the same code simultaneously?
- **TOCTOU Race Condition:** Given that validation happens BEFORE database transaction:
  - If single-use: Need to add `promo_code_usage` tracking table OR move validation inside transaction
  - If multi-use: Current design is acceptable
  - If rate-limited: Need usage tracking with atomic counters

**Impact:** Current design has a race condition where multiple users could use a single-use code if they submit orders concurrently. This is acceptable for multi-use codes but not for single-use codes.

**Recommendation:** Clarify business rules from stakeholders, then implement appropriate tracking if needed.

### 2. Connection Pool Sizing (PRIORITY: MEDIUM)

**Current Configuration:** 25 max connections, 10 idle connections

**Questions Needing Answers:**
- What is the expected requests per second under normal load?
- What is the expected P95/P99 latency for order creation?
- What is the average database query duration?
- What is the database server's connection capacity?

**Impact:** Current pool size is a guess. If too small, requests will queue and timeout. If too large, database resources are wasted.

**Recommendation:** Conduct load testing with realistic traffic patterns to determine optimal pool size. Start with 25, monitor `dbPoolWaitCount` and `dbPoolWaitDuration` metrics, adjust based on observations.

### 3. Rate Limiting & DoS Protection (PRIORITY: HIGH for production)

**Current Implementation:** No rate limiting

**Attack Vectors:**
- Client submits thousands of requests with invalid promo codes (each times out after 5s)
- Client submits requests with hundreds of items per order
- Client submits requests with non-existent promo codes (forces full file scans)

**Impact:** Service can be trivially DoS'd by malicious or buggy clients.

**Recommendation:** Implement rate limiting before production:
- Per-IP rate limiting (e.g., 100 requests per minute)
- Per-API-key rate limiting (e.g., 1000 requests per minute)
- Request size limits (max 100 items per order, max 1MB body size)

**TODO:** Add rate limiting middleware in Phase 1.5 or accept risk for MVP.

### 4. Promo Code Validation Performance Optimization (PRIORITY: LOW)

**Current Implementation:** Files scanned on every validation (5s max)

**Potential Optimizations:**
- **In-Memory Cache:** Load valid codes into memory on startup (requires ~10MB per 1M codes)
- **Redis Cache:** Cache validation results for 5 minutes
- **Bloom Filter:** Use probabilistic data structure for fast negative lookups

**Impact:** Could reduce P95 from 500ms to <10ms, reduce I/O load, improve scalability.

**Recommendation:** Measure actual performance under load first. If promo validation is a bottleneck (P95 > 500ms or high I/O wait), implement caching. For MVP, file-based approach is acceptable.

### 5. Error Response Format for Field Validation (PRIORITY: LOW)

**Current Implementation:** Single error with message

**Gap:** When multiple validation failures occur (e.g., quantity < 0 AND invalid product ID), only one error is returned.

**Better Approach:**
```json
{
  "error": "VALIDATION_FAILED",
  "message": "Request validation failed",
  "correlation_id": "...",
  "details": [
    {
      "field": "items[0].quantity",
      "code": "INVALID_QUANTITY",
      "message": "Quantity must be greater than 0"
    },
    {
      "field": "items[1].productId",
      "code": "PRODUCT_NOT_FOUND",
      "message": "Product 'invalid-id' does not exist"
    }
  ]
}
```

**Recommendation:** Implement in Phase 1.5 if API consumers need detailed validation feedback.

### 6. Pagination Metadata (PRIORITY: LOW)

**Current Implementation:** GET /product?limit=100&offset=0 returns array of products

**Gap:** No way for client to know total product count or if there are more pages.

**Options:**
- Add `X-Total-Count` header with total products
- Add HATEOAS links (next/prev)
- Return envelope with metadata: `{data: [...], total: 1523, limit: 100, offset: 0}`

**Recommendation:** Document current behavior ("paginate until empty array") for MVP, add metadata in Phase 1.5 if needed.

### 7. Observability for Connection Pool (PRIORITY: MEDIUM)

**Current Implementation:** Metrics track HTTP requests and promo validation

**Gap:** No visibility into database connection pool behaviour under load.

**TODO:** Add Prometheus metrics:
```go
dbPoolActiveConns   // Current connections in use
dbPoolIdleConns     // Current idle connections
dbPoolWaitCount     // Total times waited for connection
dbPoolWaitDuration  // Total time waited for connection
```

**Recommendation:** Implement during initial development for production readiness.

### 8. Health Check Semantics (PRIORITY: LOW)

**Current Implementation:** Returns 503 if database is unreachable OR coupon files missing

**Question:** Should health check distinguish between:
- **Liveness:** Is the service running? (for container restart)
- **Readiness:** Is the service ready to handle traffic? (for load balancer)

**Recommendation:** Current health check is suitable for readiness. For production Kubernetes, may need separate `/live` and `/ready` endpoints.

---

## Future Enhancements (Out of Scope)

These features are not in the initial implementation but may be added later:

1. **Order Retrieval:** GET /order/{orderId} endpoint
2. **Discount Calculation:** Apply actual pricing discounts for valid promo codes
3. **Rate Limiting:** Per-client request rate limits
4. **Caching:** In-memory cache for product catalog
5. **Promo Code Cache:** Cache validation results to avoid file I/O
6. **Observability:** OpenTelemetry tracing, distributed tracing
7. **API Versioning:** /v1/product endpoints
8. **Pagination:** Cursor-based pagination for better performance
9. **Search:** Product search by name, category
10. **Admin API:** Endpoints to manage products, view orders
