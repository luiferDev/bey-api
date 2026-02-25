# AGENTS.md - Bey API Development Guide

## Project Overview

Bey API is a Go REST API built with Gin web framework and GORM ORM. It provides e-commerce functionality including products, categories, users, orders, and inventory management.

## Tech Stack
- **Language**: Go 1.25+
- **Web Framework**: Gin
- **ORM**: GORM
- **Databases**: PostgreSQL (production), SQLite (development)
- **Configuration**: YAML-based

## Project Structure
```
bey_api/
├── cmd/api/main.go          # Application entry point
├── internal/
│   ├── config/              # Configuration loading
│   ├── database/            # Database connection (postgres.go)
│   ├── concurrency/        # Worker pool, task queue, task types
│   ├── modules/             # Feature modules
│   │   ├── products/        # Products, categories, variants, images
│   │   ├── users/           # User management
│   │   ├── orders/          # Order processing
│   │   └── inventory/       # Inventory tracking
│   └── shared/
│       ├── middleware/      # CORS, logging, rate limiting middleware
│       └── response/        # Response formatting helpers
├── config.yaml              # Application configuration
└── test_api.sh             # Manual API test script
```

## Build & Run Commands

### Development
```bash
# Run the application
go run ./cmd/api/

# Build binary
go build -o main ./cmd/api/

# Run with custom config
go run ./cmd/api/ --config=/path/to/config.yaml
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for specific package
go test -v ./internal/modules/products/...

# Run specific test function
go test -v -run TestFunctionName ./internal/modules/products/...

# Run tests with coverage
go test -cover ./...
```

### Manual API Testing
```bash
# Ensure server is running on localhost:8080, then:
./test_api.sh
```

### Linting & Code Quality
```bash
# Install golangci-lint (if not present)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run

# Format code
go fmt ./...

# Vet code
go vet ./...
```

### Concurrency & Profiling
```bash
# Run with pprof enabled (starts on main port + 1, e.g., 8081 if API is on 8080)
go run ./cmd/api/

# Analyze goroutine heap dump
go tool pprof http://localhost:8081/debug/pprof/heap

# View current goroutines
go tool pprof http://localhost:8081/debug/pprof/goroutine

# Get goroutine profile (30 seconds)
curl -o profile.pb http://localhost:8081/debug/pprof/profile?seconds=30

# View blocking profile
go tool pprof http://localhost:8081/debug/pprof/block

# View mutex profile
go tool pprof http://localhost:8081/debug/pprof/mutex
```

## Code Style Guidelines

### Package Organization
- Handlers, repositories, models, and DTOs live in the same package per module
- Use meaningful package names (e.g., `products`, `users`, `orders`)
- Shared utilities go in `internal/shared/`

### Naming Conventions
- **Files**: snake_case (e.g., `handler.go`, `repository.go`, `model.go`)
- **Types/Structs**: PascalCase (e.g., `ProductHandler`, `ProductRepository`)
- **Functions/Methods**: PascalCase (e.g., `CreateProduct`, `FindByID`)
- **Variables**: camelCase (e.g., `productRepo`, `categoryID`)
- **Constants**: PascalCase or SCREAMING_SNAKE_CASE for config values
- **Interfaces**: PascalCase with `er` suffix when appropriate (e.g., `Repository`)

### Import Organization
Standard Go import grouping:
```go
import (
    // Standard library
    "fmt"
    "net/http"
    "strconv"
    "time"

    // Third-party packages
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"

    // Internal packages
    "bey/internal/config"
    "bey/internal/modules/products"
)
```

### Error Handling
- Return errors from repository and service layers
- Handle errors in handlers with appropriate HTTP status codes
- Use `errors.Is()` for error comparison (especially GORM errors)
- Return `nil, nil` for "not found" cases (avoid wrapping in sentinel errors)
- Log errors appropriately (use `log.Printf` or structured logging)

```go
// Repository example
func (r *ProductRepository) FindByID(id uint) (*Product, error) {
    var product Product
    if err := r.db.First(&product, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, err
    }
    return &product, nil
}

// Handler example
func (h *ProductHandler) GetProduct(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
        return
    }

    product, err := h.productRepo.FindByID(uint(id))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get product"})
        return
    }
    if product == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
        return
    }

    c.JSON(http.StatusOK, product)
}
```

### GORM Model Conventions
- Use `gorm:"primaryKey"` for primary keys
- Use `gorm:"size:X"` for string size limits
- Use `gorm:"uniqueIndex"` or `gorm:"index"` for indexes
- Use `gorm:"default:VALUE"` for defaults
- Use `gorm:"foreignKey:Name"` for relationships
- Use `gorm:"type:DECIMAL(12,2)"` for precise decimals

```go
type Product struct {
    ID          uint           `gorm:"primaryKey" json:"id"`
    CategoryID  uint           `json:"category_id"`
    Name        string         `gorm:"size:255;not null" json:"name"`
    Slug        string         `gorm:"size:255;uniqueIndex;not null" json:"slug"`
    BasePrice   float64        `gorm:"type:decimal(12,2);not null" json:"base_price"`
    IsActive    bool           `gorm:"default:true" json:"is_active"`
    Category    Category       `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}
```

### DTO Conventions
- Request DTOs: `CreateXxxRequest`, `UpdateXxxRequest`
- Response DTOs: `XxxResponse`
- Use pointer types (`*string`, `*int`) for optional fields in requests
- Use struct tags: `json:"field_name"`, `binding:"required"`
- Validation: Use Gin's binding tags (e.g., `binding:"required"`, `binding:"gt=0"`)

```go
type CreateProductRequest struct {
    CategoryID  uint    `json:"category_id" binding:"required"`
    Name        string  `json:"name" binding:"required"`
    BasePrice   float64 `json:"base_price" binding:"required,gt=0"`
    IsActive    *bool   `json:"is_active"`
}
```

### Handler Patterns
- One handler struct per feature module
- Pass dependencies via constructor
- Use pointer receiver methods
- Return early on errors
- Use consistent response patterns via `response.Success()` or direct `c.JSON()`

```go
type ProductHandler struct {
    productRepo *ProductRepository
}

func NewProductHandler(productRepo *ProductRepository) *ProductHandler {
    return &ProductHandler{
        productRepo: productRepo,
    }
}
```

### Route Registration
- Group routes under `/api/v1` prefix
- Use meaningful HTTP methods (GET, POST, PUT, DELETE)
- Follow REST conventions: `/resources`, `/resources/:id`, `/resources/:id/subresource`

### Configuration
- Configuration via YAML file (`config.yaml`)
- Use `.env.example` to document required environment variables
- Load config in `main.go` before database initialization

## Database

### Migrations
- Use GORM's `AutoMigrate()` in `main.go`
- Models are defined in `internal/modules/*/model.go`

### Connections
- Database connection managed in `internal/database/postgres.go`
- Supports PostgreSQL (production) and SQLite (development)

## Common Tasks

### Adding a New Module
1. Create `internal/modules/<module>/model.go` - GORM models
2. Create `internal/modules/<module>/repository.go` - Data access layer
3. Create `internal/modules/<module>/handler.go` - HTTP handlers
4. Create `internal/modules/<module>/dto.go` - Request/Response DTOs
5. Create `internal/modules/<module>/routes.go` - Route definitions
6. Register routes in `main.go`

### Adding a New Endpoint
1. Add DTO in `dto.go` if needed
2. Add repository method in `repository.go`
3. Add handler method in `handler.go`
4. Register route in `routes.go`

### Database Changes
1. Modify model in `model.go`
2. Run AutoMigrate or create manual migration
3. Test with SQLite for development
