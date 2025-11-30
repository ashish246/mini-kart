# Mini-Kart API

A Go-based e-commerce API server with product management, order processing, and promotional code validation. The original ask and requirements are taken from this [README.md](https://github.com/oolio-group/kart-challenge/blob/advanced-challenge/backend-challenge/README.md).

## Features

- **Product Management**: Browse and retrieve product information
- **Order Processing**: Create and retrieve orders with multiple items
- **Promotional Code Validation**: Concurrent validation of promo codes across multiple sources
- **AWS S3 Integration**: Load coupon files from S3 with automatic local fallback
- **RESTful API**: Clean HTTP endpoints with proper error handling
- **Database**: PostgreSQL for persistent storage
- **Authentication**: API key-based authentication
- **Middleware**: CORS, logging, panic recovery
- **Health Checks**: Built-in health endpoint for monitoring

## Tech Stack

- **Language**: Go 1.25.4
- **Database**: PostgreSQL 16
- **Cloud Storage**: AWS S3 (optional, with local fallback)
- **Testing**: Testcontainers for integration tests
- **Logging**: Structured logging with zerolog
- **HTTP**: Standard library net/http
- **Build**: Multi-stage Docker builds

## Project Structure

```
mini-kart/
├── cmd/
│   └── api/              # Application entrypoint
├── internal/
│   ├── config/           # Configuration management
│   ├── coupon/           # Promotional code validation
│   ├── database/         # Database connection pooling
│   ├── handler/          # HTTP handlers
│   ├── middleware/       # HTTP middleware
│   ├── model/            # Domain models
│   ├── repository/       # Data access layer
│   ├── router/           # HTTP routing
│   └── service/          # Business logic
├── test/
│   └── integration/      # Integration tests
├── data/
│   └── coupons/          # Promotional code files
├── Dockerfile            # Multi-stage Docker build
├── docker-compose.yml    # Full stack orchestration
├── Makefile              # Build automation
└── .env.example          # Environment variables template

```

## Quick Start

### Prerequisites

- Go 1.25.4 or later
- Docker and Docker Compose (for containerised deployment)
- PostgreSQL 16 (if running locally)
- Make (for build automation)

### Environment Setup

1. Copy the example environment file:

```bash
cp .env.example .env
```

2. Update the `.env` file with your configuration:

```bash
# Required: Set a secure API key
API_KEY=your_secure_api_key_here

# Optional: Adjust database settings
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=minikart
```

### Running with Docker Compose (Recommended)

The easiest way to run the application is using Docker Compose, which automatically sets up the database and runs migrations:

```bash
# Start the full stack (PostgreSQL + API)
make docker-up

# View logs
docker-compose logs -f api

# Stop services
make docker-down
```

**Note:** When using Docker Compose, the API service automatically waits for PostgreSQL to be ready before starting.

### Running Locally (Without Docker)

**Option 1: Using go run (recommended for development)**

```bash
# Start only PostgreSQL database
make postgres-start

# Run database migrations
make migrate-up

# Generate sample coupon files (for testing)
make generate-coupons

# Run the application with go run (loads .env file automatically)
make run-dev
```

**Option 2: Using compiled binary**

```bash
# Start only PostgreSQL database
make postgres-start

# Run database migrations
make migrate-up

# Build the application
make build

# Run the application locally
make run-local
```

The server will start on `http://localhost:8080`

**Note:** `make run-dev` automatically loads environment variables from the `.env` file, making it ideal for local development. Ensure you have created and configured your `.env` file before running (see Environment Setup above).

## API Testing with Postman

A comprehensive Postman collection is available at `docs/postman_collection.json`.

### Importing the Collection

1. Open Postman
2. Click **Import** button
3. Select the file `docs/postman_collection.json`
4. The collection will be imported with all endpoints and tests

### Configuring Environment Variables

The collection uses two variables that you need to configure:

1. **baseUrl**: The base URL of the API (default: `http://localhost:8080`)
2. **apiKey**: Your API key for authentication (must match the `API_KEY` in your `.env` file)

To set these variables:

1. Select the **Mini-Kart API** collection in Postman
2. Click the **Variables** tab
3. Update the **Current Value** for `baseUrl` and `apiKey`
4. Click **Save**

### Collection Features

The Postman collection includes:

- **Pre-request Scripts**: Automatically adds the `X-API-Key` header to authenticated requests
- **Test Scripts**: Validates response status codes and response structure
- **Example Requests**: All endpoints with sample request bodies
- **Variable Storage**: Automatically saves product IDs and order IDs for use in subsequent requests

### Available Endpoints in Collection

- **Health**: Health check endpoint (no authentication required)
- **Products**: List all products, get product by ID
- **Orders**: Create order (with/without promo code), get order by ID
- **Authentication Tests**: Test cases for missing and invalid API keys

### Running the Collection

You can run individual requests or use the **Collection Runner** to execute all requests in sequence:

1. Right-click on the **Mini-Kart API** collection
2. Select **Run collection**
3. Click **Run Mini-Kart API**
4. View the test results

## API Endpoints

### Health Check

```bash
GET /health
```

No authentication required.

**Response:**

```json
{ "status": "healthy" }
```

### Products

#### Get All Products

```bash
GET /api/products?limit=10&offset=0
X-API-Key: your_api_key
```

**Query Parameters:**

- `limit` (optional): Number of products to return (default: 10, max: 100)
- `offset` (optional): Number of products to skip (default: 0)

**Response:**

```json
[
  {
    "id": "P001",
    "name": "Product Name",
    "price": 29.99,
    "category": "Category",
    "created_at": "2025-11-30T12:00:00Z"
  }
]
```

#### Get Product by ID

```bash
GET /api/products/{id}
X-API-Key: your_api_key
```

**Response:**

```json
{
  "id": "P001",
  "name": "Product Name",
  "price": 29.99,
  "category": "Category",
  "created_at": "2025-11-30T12:00:00Z"
}
```

### Orders

#### Create Order

```bash
POST /api/orders
X-API-Key: your_api_key
Content-Type: application/json

{
  "coupon_code": "PROMO2025",
  "items": [
    {
      "product_id": "P001",
      "quantity": 2
    },
    {
      "product_id": "P002",
      "quantity": 1
    }
  ]
}
```

**Response:**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "items": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "order_id": "550e8400-e29b-41d4-a716-446655440000",
      "product_id": "P001",
      "quantity": 2
    }
  ],
  "products": [
    {
      "id": "P001",
      "name": "Product Name",
      "price": 29.99,
      "category": "Category",
      "created_at": "2025-11-30T12:00:00Z"
    }
  ]
}
```

#### Get Order by ID

```bash
GET /api/orders/{id}
X-API-Key: your_api_key
```

**Response:** Same as Create Order response

## Development

### Running Tests

```bash
# Run unit tests only
make test

# Run integration tests
make test-integration

# Run all tests
make test-all

# Run tests with coverage
make test-coverage
```

### Database Management

```bash
# Start only PostgreSQL database
make postgres-start

# Stop PostgreSQL database
make postgres-stop

# Run database migrations
make migrate-up

# Rollback database migrations
make migrate-down

# Reset database (WARNING: Drops all tables and data!)
make db-reset
```

**Note:** The database connection string defaults to `postgres://postgres:postgres@localhost:5432/minikart?sslmode=disable`. You can override it by setting the `DB_URL` environment variable:

```bash
DB_URL=postgres://user:password@host:port/database?sslmode=disable make migrate-up
```

### Development Utilities

The project includes utility scripts in the `/scripts` directory to help with development and testing:

#### Generate Sample Coupon Files

Creates small test coupon files for local development (instead of using the large production files):

```bash
make generate-coupons
```

This creates three gzipped coupon files in `data/coupons/` with known test data:

- **Valid codes** (appear in 2+ files): `VALIDONE1`, `VALIDTWO12`, `ALLTHREE1`, `SUMMER2024`, `WINTER2024`
- **Invalid codes** (appear in only 1 file): `ONLYONE111`, `ONLYTWO222`, `ONLYTHREE3`, `SPRING2024`

#### Test Database Connection

Verify connectivity to the `minikart` database:

```bash
make test-db-connection
```

Expected output:

```
Successfully connected to database: minikart
```

#### Test PostgreSQL Server

Check PostgreSQL server connectivity and list all available databases:

```bash
make test-pg-server
```

Expected output:

```
Successfully connected to database: postgres

Available databases:
  - postgres
  - minikart
```

### Code Quality

```bash
# Format code
make format

# Run linter
make lint

# Build the application
make build
```

### Test Coverage

The project maintains high test coverage:

- **Overall Coverage**: 82.6%
- **Coupon Validation**: 95.0%
- **HTTP Handlers**: 94.9%
- **Middleware**: 97.7%
- **Service Layer**: 89.0%
- **Repository Layer**: 85.7%

## Configuration

All configuration is managed through environment variables. See `.env.example` for available options.

### Server Configuration

- `SERVER_HOST`: Server bind address (default: 0.0.0.0)
- `SERVER_PORT`: Server port (default: 8080)

### Database Configuration

- `DB_HOST`: PostgreSQL host (default: localhost)
- `DB_PORT`: PostgreSQL port (default: 5432)
- `DB_USER`: Database user (default: postgres)
- `DB_PASSWORD`: Database password (required)
- `DB_NAME`: Database name (default: minikart)
- `DB_MAX_CONNECTIONS`: Maximum connections (default: 25)
- `DB_MIN_CONNECTIONS`: Minimum connections (default: 5)
- `DB_MAX_CONN_LIFETIME`: Connection lifetime in seconds (default: 300)

### Logging Configuration

- `LOG_LEVEL`: Log level - debug, info, warn, error (default: info)
- `LOG_FORMAT`: Log format - json, console (default: json)

### Authentication

- `API_KEY`: API key for authentication (required)

### AWS S3 Configuration

The application supports loading coupon files from AWS S3 with automatic fallback to local file system. This is useful for production deployments where coupon files are stored centrally in S3.

- `S3_ENABLED`: Enable S3 for coupon files - true or false (default: false)
- `S3_BUCKET`: S3 bucket name (required when S3_ENABLED=true)
- `S3_REGION`: AWS region (default: us-east-1)
- `S3_PREFIX`: Path prefix within bucket (default: coupons/)

**How it works:**

1. When `S3_ENABLED=true`, the application first attempts to load coupon files from S3
2. S3 keys are constructed as: `S3_PREFIX + filename` (e.g., `coupons/coupon_list_1.txt.gz`)
3. If S3 loading fails (connection error, file not found, etc.), it automatically falls back to local file system
4. When `S3_ENABLED=false`, only local file system is used

**AWS Credentials:**
The application uses the AWS SDK default credential chain, which checks for credentials in this order:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM instance profile (when running on EC2)

**Example S3 setup:**

```bash
# Enable S3
S3_ENABLED=true
S3_BUCKET=my-company-coupons
S3_REGION=ap-southeast-2
S3_PREFIX=production/coupons/

# AWS credentials (if not using IAM role)
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
```

## Architecture

### Layered Architecture

The application follows a clean layered architecture:

1. **HTTP Layer** (`handler`, `middleware`, `router`): HTTP request handling and routing
2. **Service Layer** (`service`): Business logic and orchestration
3. **Repository Layer** (`repository`): Data access and persistence
4. **Domain Layer** (`model`): Core business entities

### Concurrent Promo Code Validation

The promotional code validation system uses a concurrent multi-file lookup strategy:

- Reads multiple gzipped coupon files in parallel
- Validates codes against configurable minimum match count
- Uses goroutines and channels for efficient concurrent processing
- Implements proper error handling and resource cleanup

## Deployment

### Docker

Build and run with Docker:

```bash
# Build image
docker build -t mini-kart-api .

# Run container
docker run -p 8080:8080 --env-file .env mini-kart-api
```

### Docker Compose

Full stack deployment:

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down
```

## Health Monitoring

The application includes health check endpoints:

- **HTTP Health Check**: `GET /health`
- **Docker Health Check**: Automated container health monitoring

## Performance

Refer to [performance-analysis.md](docs/performance-analysis.md) for possible recommendations to optimise the promo code validation as per the use case.

## Security

- API key authentication on all endpoints (except `/health`)
- Environment-based configuration (no hardcoded secrets)
- Input validation on all requests
- Parameterised database queries (SQL injection protection)
- CORS middleware for cross-origin requests
- Panic recovery middleware
- Non-root user in Docker containers

## License

This project is licensed under the MIT Licence.

## Contributing

This is a demonstration project. For production use, consider:

- Adding database migrations
- Implementing rate limiting
- Adding request/response encryption
- Implementing audit logging
- Adding metrics and tracing
- Setting up CI/CD pipelines
