# AGENTS.md - Bey API Development Guide

## Project Overview
Bey API is a Go REST API built with Gin + GORM. Provides e-commerce: products, categories, users, orders, inventory.

**Tech Stack**: Go 1.25+, Gin, GORM, PostgreSQL/SQLite, YAML config

## Project Structure
```
bey_api/
├── cmd/api/main.go           # Entry point
├── internal/
│   ├── config/               # YAML config loading
│   ├── database/             # DB connection
│   ├── concurrency/          # Worker pool, task queue
│   ├── modules/              # Feature modules (products, users, orders, inventory)
│   └── shared/               # Middleware, response helpers
├── config.yaml
└── openspec/                 # SDD specifications
```

## Essential Commands

### Build & Run
```bash
go run ./cmd/api/                    # Run dev server
go build -o main ./cmd/api/          # Build binary
```

### Testing (SINGLE TEST - most important)
```bash
go test -v -run TestFunctionName ./internal/modules/products/...  # Run one test
go test -v ./internal/modules/products/...                          # Package tests
go test -v ./...                                                   # All tests
go test -cover ./...                                              # With coverage
```

### Linting & Quality
```bash
go fmt ./...           # Format code
go vet ./...           # Vet
golangci-lint run      # Lint (install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
```

### Swagger
```bash
swag init -g cmd/api/main.go -o cmd/api/docs --parseDependency --parseInternal  # Generate docs
# Access: http://localhost:8080/swagger/index.html
```

## Code Style Guidelines

### Naming
- Files: `snake_case` (handler.go, model.go)
- Types: `PascalCase` (ProductHandler, ProductRepository)
- Variables: `camelCase` (productRepo, categoryID)
- Interfaces: `er` suffix (Repository, Handler)

### Imports (3 groups, blank line between)
```go
import (
    // Stdlib
    "fmt"
    "net/http"
    
    // Third-party
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
    
    // Internal
    "bey/internal/config"
)
```

### Error Handling
- Return errors from repo/service layers
- Use `errors.Is()` for GORM errors
- Return `nil, nil` for "not found" (not sentinel errors)
```go
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
```

### GORM Models
```go
type Product struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    Name      string    `gorm:"size:255;not null" json:"name"`
    Slug      string    `gorm:"size:255;uniqueIndex;not null" json:"slug"`
    BasePrice float64   `gorm:"type:decimal(12,2);not null" json:"base_price"`
    Category  Category  `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}
```

### DTOs
- Request: `CreateXxxRequest`, `UpdateXxxRequest`
- Response: `XxxResponse`
- Optional fields: pointer types (`*string`, `*int`)
```go
type CreateProductRequest struct {
    CategoryID  uint    `json:"category_id" binding:"required"`
    Name        string  `json:"name" binding:"required"`
    BasePrice   float64 `json:"base_price" binding:"required,gt=0"`
}
```

### Handlers
- One struct per module, dependencies via constructor
- Return early on errors, use `response.Success()` or `c.JSON()`
```go
type ProductHandler struct {
    productRepo *ProductRepository
}

func NewProductHandler(productRepo *ProductRepository) *ProductHandler {
    return &ProductHandler{productRepo: productRepo}
}
```

### Routes
- Group under `/api/v1`
- REST conventions: `/resources`, `/resources/:id`

### Configuration
- YAML in `config.yaml`
- Load in `main.go` before DB init

## Adding a New Module
1. `model.go` - GORM models
2. `repository.go` - Data access
3. `handler.go` - HTTP handlers  
4. `dto.go` - Request/Response DTOs
5. `routes.go` - Route definitions
6. Register in `main.go`

## Database
- Use GORM `AutoMigrate()` in main.go
- Models in `internal/modules/*/model.go`
