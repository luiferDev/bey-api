# AGENTS.md - Bey API Development Guide

Go 1.26 REST API with Gin + GORM for e-commerce (products, categories, users, orders, inventory, cart, payments).
**Stack**: Go 1.26, Gin, GORM, PostgreSQL 17, Redis 8, JWT, OAuth2, Wompi, YAML config

## Essential Commands

```bash
go run ./cmd/api/                          # Dev server (localhost:8080)
go build -o main ./cmd/api/                # Build binary
go test -v -run TestName ./internal/...    # Single test (MOST IMPORTANT)
go test -race ./...                        # Race detector (dev only)
go test -v -cover ./...                    # All tests with coverage
golangci-lint run --timeout=5m             # Lint (v2.11+, config: .golangci.yml)
go fmt ./... && go vet ./...               # Format + basic checks
swag init -g cmd/api/main.go -o cmd/api/docs --parseDependency --parseInternal
docker compose up -d                       # Start all services
```

## Code Style

### Imports (3 groups, blank line between)
```go
import (
    "encoding/json"  // Stdlib
    "github.com/gin-gonic/gin"  // Third-party
    "bey/internal/config"  // Internal
)
```

### Naming
| Type | Convention | Example |
|------|-----------|---------|
| Files | `snake_case` | `handler.go`, `product_repository.go` |
| Types | `PascalCase` | `ProductHandler`, `ProductRepository` |
| Variables | `camelCase` | `productRepo`, `categoryID` |
| Interfaces | `er` suffix | `Repository`, `Handler`, `Service` |
| Test files | `*_test.go` | `handler_test.go` |

### GORM Models — use `uuid.UUID` (gofrs/uuid/v5), NOT uint
```go
type Product struct {
    ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
    Name      string    `gorm:"size:255;not null" json:"name"`
    Slug      string    `gorm:"size:255;uniqueIndex;not null" json:"slug"`
    Category  Category  `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}
```

### DTOs — Request/Response, never raw GORM models in handlers
```go
type CreateProductRequest struct {
    CategoryID string  `json:"category_id" binding:"required"`
    Name       string  `json:"name" binding:"required,max=255"`
    BasePrice  float64 `json:"base_price" binding:"required,gt=0"`
}
type UpdateProductRequest struct {
    Name      *string  `json:"name"`      // Pointer = optional (PATCH)
    BasePrice *float64 `json:"base_price"`
}
```

### Error Handling — return nil, nil for "not found"
```go
func (r *ProductRepository) FindByID(id uuid.UUID) (*Product, error) {
    var product Product
    if err := r.db.First(&product, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil  // Not found — NOT an error
        }
        return nil, fmt.Errorf("failed to find product: %w", err)
    }
    return &product, nil
}
```

### Handler Pattern — use ResponseHandler, return early on errors
```go
func (h *ProductHandler) GetProduct(c *gin.Context) {
    id, err := uuid.FromString(c.Param("id"))
    if err != nil {
        h.resp.ValidationError(c, "invalid product ID format")
        return
    }
    product, err := h.repo.FindByID(id)
    if err != nil { h.resp.InternalError(c, "failed to get product"); return }
    if product == nil { h.resp.NotFound(c, "product not found"); return }
    h.resp.Success(c, toProductResponse(product))  // DTO, never raw model
}
```

### Module Structure
```
internal/modules/{module}/
├── model.go         # GORM models (uuid.UUID, gorm tags)
├── dto.go           # Request/Response DTOs (binding tags)
├── repository.go    # Data access (nil, nil for not found)
├── service.go       # Business logic
├── handler.go       # HTTP handlers (ResponseHandler, DTOs only)
├── routes.go        # Route definitions
└── *_test.go        # Table-driven tests
```

## Critical Rules

1. **NEVER return raw GORM models in HTTP responses** — always use DTOs. Bidirectional relations (Category↔Product) cause infinite JSON recursion.
2. **ALWAYS use thread-safe Task methods** — `task.SetStatus()`, `task.SetError()`, `task.SetResult()`, `task.SetUpdatedAt()`. NEVER write `task.Status = ...` directly (data race).
3. **`product_variants` has NO `deleted_at` column** — never query `WHERE deleted_at IS NULL` on this table.
4. **Inventory source of truth is `product_variants`** — the `inventories` table is legacy. Sum `stock`/`reserved` from variants.
5. **Never use empty strings for `time.Duration` in config.yaml** — YAML parser crashes. Remove the field; defaults are in code.
6. **Type assertions must use comma-ok**: `val, ok := iface.(Type)` — never `iface.(Type)` directly (errcheck lint).
7. **Use `http.NewRequestWithContext()`** — never `http.NewRequest()` (noctx lint).
8. **Docker secrets via `env_file: - .env`** — never hardcode in docker-compose.yml.

## Security

- Load `go-gin-security` skill before any security-related work
- Derive user identity from JWT claims, NEVER from request body
- All DTOs have `binding` tags with `max` length limits
- Generic error messages to clients; full errors logged server-side
- JWT secret must be ≥32 characters (validated at startup)

## Linter (golangci-lint v2.11+)

Enabled: `errcheck`, `staticcheck` (includes gosimple), `govet`, `ineffassign`, `unused`, `gosec`, `bodyclose`, `noctx`, `gocritic`.

## Database

- PostgreSQL 17, GORM AutoMigrate in `main.go`
- UUID v7 for all primary keys (`uuidutil.GenerateV7()` in BeforeCreate)
- Parameterized queries only (GORM handles this)
- Use transactions for multi-step operations

## Docker

```bash
docker compose up -d           # Start API + PostgreSQL + Redis
docker compose watch           # Dev watch mode (auto-restart on .go changes)
docker compose down -v         # Stop + delete volumes (⚠️ data loss)
```
