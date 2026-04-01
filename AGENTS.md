# AGENTS.md - Bey API Development Guide

## Project Overview
Go REST API with Gin + GORM for e-commerce (products, categories, users, orders, inventory, cart, payments).  
**Tech Stack**: Go 1.26, Gin, GORM, PostgreSQL, Redis, JWT, OAuth2, Wompi, YAML config

```
bey_api/
├── cmd/api/main.go              # Entry point
├── internal/
│   ├── config/                  # YAML config loading
│   ├── database/                # DB connection
│   ├── concurrency/             # Worker pool, task queue
│   ├── modules/                 # Feature modules
│   │   ├── auth/               # JWT, OAuth2, 2FA
│   │   ├── users/              # User management
│   │   ├── products/           # Products, categories, variants
│   │   ├── cart/               # Shopping cart (Redis)
│   │   ├── orders/             # Order management
│   │   ├── payments/          # Wompi integration
│   │   ├── inventory/          # Stock management
│   │   └── ...
│   └── shared/                  # Middleware, helpers
├── config.yaml
└── openspec/                    # SDD specifications
```

---

## Essential Commands

### Build & Run
```bash
go run ./cmd/api/                    # Run dev server (localhost:8080)
go build -o main ./cmd/api/          # Build binary
go build -tags prod ./cmd/api/        # Build with production tag
```

### Testing - SINGLE TEST (most important)
```bash
# Run one specific test
go test -v -run TestFunctionName ./internal/modules/products/...

# Run all tests in a package
go test -v ./internal/modules/products/...

# Run all tests with coverage
go test -v -cover ./...

# Run with race detector (in development only)
go test -race ./...

# Run specific test file
go test -v ./internal/modules/products/... -run TestProductRepository_FindByID
```

### Linting & Quality
```bash
go fmt ./...                    # Format code
go vet ./...                    # Basic vet
golangci-lint run              # Full lint (config: .golangci.yml)
golangci-lint run --fast       # Fast mode (skip slow linters)
```

### Swagger Documentation
```bash
swag init -g cmd/api/main.go -o cmd/api/docs --parseDependency --parseInternal
# Access: http://localhost:8080/swagger/index.html
```

### Database
```bash
# Run migrations (auto in main.go via GORM AutoMigrate)
# Reset test database (if using testcontainers)
```

---

## Code Style Guidelines

### Naming Conventions
| Type | Convention | Example |
|------|-----------|---------|
| Files | `snake_case` | `handler.go`, `product_repository.go` |
| Types | `PascalCase` | `ProductHandler`, `ProductRepository` |
| Variables | `camelCase` | `productRepo`, `categoryID` |
| Constants | `PascalCase` or `SnakeCase` | `MaxItems`, `MAX_RETRIES` |
| Interfaces | `er` suffix | `Repository`, `Handler`, `Service` |
| Test files | `*_test.go` | `handler_test.go` |

### Imports (3 groups, blank line between)
```go
import (
    // Stdlib - standard library
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    // Third-party - external dependencies
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "gorm.io/gorm"

    // Internal - project packages
    "bey/internal/config"
    "bey/internal/modules/products"
)
```

### GORM Models
```go
type Product struct {
    ID        uint           `gorm:"primaryKey" json:"id"`
    Name      string         `gorm:"size:255;not null" json:"name"`
    Slug      string         `gorm:"size:255;uniqueIndex;not null" json:"slug"`
    BasePrice float64        `gorm:"type:decimal(12,2);not null" json:"base_price"`
    Category  Category       `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
    Images    []ProductImage `gorm:"foreignKey:ProductID" json:"images,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
}

// Use pointer for optional relations
type ProductVariant struct {
    ID        uint                   `gorm:"primaryKey" json:"id"`
    ProductID uint                   `json:"product_id"`
    SKU       string                 `gorm:"size:100;uniqueIndex;not null" json:"sku"`
    Price     float64                `gorm:"type:decimal(12,2);not null" json:"price"`
    Stock     int                    `gorm:"default:0" json:"stock"`
    Reserved  int                    `gorm:"default:0" json:"reserved"`
    Attribute *ProductVariantAttribute `gorm:"foreignKey:VariantID" json:"attribute,omitempty"`
    Images    []ProductImage          `gorm:"foreignKey:VariantID" json:"images,omitempty"`
}
```

### DTOs (Request/Response)
```go
// Request DTOs - use descriptive names
type CreateProductRequest struct {
    CategoryID  uint    `json:"category_id" binding:"required"`
    Name        string  `json:"name" binding:"required,max=255"`
    Slug        string  `json:"slug" binding:"required,max=255"`
    BasePrice   float64 `json:"base_price" binding:"required,gt=0"`
    Description string  `json:"description"`
}

// Response DTOs
type ProductResponse struct {
    ID        uint      `json:"id"`
    Name      string    `json:"name"`
    BasePrice float64   `json:"base_price"`
    Category  Category  `json:"category,omitempty"`
}

// Optional fields use pointer types
type UpdateProductRequest struct {
    Name        *string  `json:"name"`
    BasePrice   *float64  `json:"base_price"`
}
```

### Error Handling
```go
// Return errors from repo/service layers
// Use errors.Is() for GORM errors
// Return nil, nil for "not found" (not sentinel errors)

func (r *ProductRepository) FindByID(id uint) (*Product, error) {
    var product Product
    if err := r.db.First(&product, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil  // Not found - not an error
        }
        return nil, fmt.Errorf("failed to find product: %w", err)
    }
    return &product, nil
}

// Custom error types for business logic
var (
    ErrProductNotFound   = errors.New("product not found")
    ErrInsufficientStock = errors.New("insufficient stock")
    ErrUnauthorized      = errors.New("unauthorized")
)
```

### Handler Pattern
```go
type ProductHandler struct {
    productRepo *ProductRepository
    productSvc  *ProductService
}

// Constructor pattern - dependency injection
func NewProductHandler(productRepo *ProductRepository, productSvc *ProductService) *ProductHandler {
    return &ProductHandler{
        productRepo: productRepo,
        productSvc:  productSvc,
    }
}

// Handler methods - return early on errors
func (h *ProductHandler) GetProduct(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 32)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product ID"})
        return
    }

    product, err := h.productRepo.FindByID(uint(id))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    if product == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
        return
    }

    c.JSON(http.StatusOK, product)
}
```

### Routes Pattern
```go
func RegisterRoutes(router *gin.RouterGroup, handler *ProductHandler) {
    products := router.Group("/products")
    {
        products.GET("", handler.ListProducts)
        products.GET("/:id", handler.GetProduct)
        products.POST("", handler.CreateProduct)
        products.PUT("/:id", handler.UpdateProduct)
        products.DELETE("/:id", handler.DeleteProduct)
    }
}
```

### Configuration Pattern
```go
type Config struct {
    App        AppConfig        `yaml:"app"`
    Database   DatabaseConfig   `yaml:"database"`
    Redis      RedisConfig      `yaml:"redis"`
    RateLimit  RateLimitConfig  `yaml:"rate_limit"`
    Wompi      WompiConfig     `yaml:"wompi"`
}

// YAML tags with validation
type WompiConfig struct {
    Enabled      bool   `yaml:"enabled"`
    Environment  string `yaml:"environment"` // "sandbox" or "production"
    PublicKey    string `yaml:"public_key"`
    PrivateKey   string `yaml:"private_key"`
    EventKey     string `yaml:"event_key"`
    IntegrityKey string `yaml:"integrity_key"`
    BaseURL      string `yaml:"base_url"`
}
```

---

## Linter Configuration

The project uses **golangci-lint** with these linters enabled:
- `errcheck` - Check unchecked errors
- `gosimple` - Simplify code
- `govet` - Suspicious constructs
- `ineffassign` - Unused variable assignments
- `staticcheck` - Static analysis
- `unused` - Unused code
- `gosec` - Security checker
- `bodyclose` - HTTP response body closed
- `nocxt` - HTTP requests without context
- `gocritic` - Bugs and style issues

Run linting:
```bash
golangci-lint run --timeout=5m
```

---

## Module Structure

Each feature module follows this pattern:

```
internal/modules/{module}/
├── model.go         # GORM models
├── dto.go           # Request/Response DTOs
├── repository.go    # Data access layer
├── service.go       # Business logic
├── handler.go       # HTTP handlers
├── routes.go        # Route definitions
├── {module}_test.go # Tests (optional)
└── ...
```

### Adding a New Module
1. `model.go` - GORM models with proper tags
2. `dto.go` - Request/Response DTOs with binding validation
3. `repository.go` - Data access with error handling
4. `service.go` - Business logic, orchestration
5. `handler.go` - HTTP handlers, Gin context
6. `routes.go` - Route definitions
7. Register in `main.go`

---

## Testing Guidelines

### Table-Driven Tests
```go
func TestProductService_CreateProduct(t *testing.T) {
    tests := []struct {
        name        string
        input       CreateProductRequest
        wantErr     bool
        errContains string
    }{
        {
            name: "success - create product",
            input: CreateProductRequest{
                Name:      "Test Product",
                BasePrice: 100.00,
            },
            wantErr: false,
        },
        {
            name: "fail - invalid price",
            input: CreateProductRequest{
                Name:      "Test Product",
                BasePrice: -10.00,
            },
            wantErr:     true,
            errContains: "price must be greater than 0",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Handler Tests
```go
func TestProductHandler_GetProduct(t *testing.T) {
    // Use gin.SetMode(gin.TestMode)
    // Mock dependencies
    // Test HTTP responses
}
```

---

## Security Guidelines

### Never Expose Secrets
- Never log sensitive data (passwords, API keys, tokens)
- Use environment variables or secrets management
- Validate all input with binding tags

### Authentication
- All protected routes require valid JWT in cookie (`access_token`)
- Use middleware for auth checks
- Verify user ownership for sensitive operations

### Database
- Use parameterized queries (GORM does this automatically)
- Never concatenate user input into SQL
- Use transactions for multi-step operations

---

## Available Skills

### Project-Level (`.agents/skills/`)
| Skill | Description |
|-------|-------------|
| golang-patterns | Idiomatic Go patterns |
| golang-testing | Table-driven tests, subtests |
| golang-concurrency-patterns | Goroutines, channels, sync |
| golang-pro | Advanced Go, microservices, pprof |
| docker-expert | Multi-stage builds, container security |
| paypal-integration | PayPal payments |

### Global Skills (`~/.opencode/skills/`)
| Skill | Description |
|-------|-------------|
| sdd-* | SDD workflow (explore, propose, spec, design, tasks, apply, verify, archive) |
| golang-gin-api | Gin REST API patterns |

---

## Database

- Use GORM `AutoMigrate()` in `main.go`
- Models in `internal/modules/*/model.go`
- Follow migration naming: `20260101000000_create_users.go`

---

## Docker

```bash
# Development
docker-compose up -d

# Build production image
docker build -t bey-api:latest .

# Run container
docker run -p 8080:8080 bey-api:latest
```

---

*Last updated: March 2026*
