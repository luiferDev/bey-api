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
├── .agents/                  # Project-specific AI agent skills
├── config.yaml
└── openspec/                 # SDD specifications
```

---

## Available Skills

### Project-Level Skills (`.agents/skills/`)

Estas skills están específicas para este proyecto y tienen prioridad sobre las globales:

| Skill | Path | Description |
|-------|------|-------------|
| **golang-patterns** | `.agents/skills/golang-patterns/SKILL.md` | Idiomatic Go patterns, best practices |
| **golang-testing** | `.agents/skills/golang-testing/SKILL.md` | Go testing patterns (table-driven, subtests, benchmarks) |
| **golang-concurrency-patterns** | `.agents/skills/golang-concurrency-patterns/SKILL.md` | Go concurrency patterns (goroutines, channels, sync) |
| **golang-pro** | `.agents/skills/golang-pro/SKILL.md` | Advanced Go patterns, microservices, pprof |
| **docker-expert** | `.agents/skills/docker-expert/SKILL.md` | Docker containerization, multi-stage builds, security |
| **multi-stage-dockerfile** | `.agents/skills/multi-stage-dockerfile/SKILL.md` | Optimized multi-stage Dockerfiles |
| **paypal-integration** | `.agents/skills/paypal-integration/SKILL.md` | PayPal payment processing |
| **design-patterns-expert** | `.agents/skills/design-patterns-expert/SKILL.md` | GoF design patterns, architecture decisions |

### Global Skills (`~/.opencode/skills/`)

Skills globales disponibles para cualquier proyecto:

#### Go & APIs
| Skill | Path | Description |
|-------|------|-------------|
| **golang-patterns** | `~/.opencode/skills/golang-patterns/SKILL.md` | Idiomatic Go patterns |
| **golang-testing** | `~/.opencode/skills/golang-testing/SKILL.md` | Go testing patterns |
| **golang-gin-api** | `~/.opencode/skills/golang-gin-api/golang-gin-api/SKILL.md` | Gin REST API patterns |

#### SDD (Spec-Driven Development)
| Skill | Path | Description |
|-------|------|-------------|
| **sdd-init** | `~/.opencode/skills/sdd-init/SKILL.md` | Initialize SDD structure |
| **sdd-explore** | `~/.opencode/skills/sdd-explore/SKILL.md` | Explore/investigate ideas |
| **sdd-propose** | `~/.opencode/skills/sdd-propose/SKILL.md` | Create change proposal |
| **sdd-spec** | `~/.opencode/skills/sdd-spec/SKILL.md` | Write specifications |
| **sdd-design** | `~/.opencode/skills/sdd-design/SKILL.md` | Technical design |
| **sdd-tasks** | `~/.opencode/skills/sdd-tasks/SKILL.md` | Task breakdown |
| **sdd-apply** | `~/.opencode/skills/sdd-apply/SKILL.md` | Implement tasks |
| **sdd-verify** | `~/.opencode/skills/sdd-verify/SKILL.md` | Verify implementation |
| **sdd-archive** | `~/.opencode/skills/sdd-archive/SKILL.md` | Archive completed changes |

#### Other Global Skills
| Skill | Path | Description |
|-------|------|-------------|
| **skill-creator** | `~/.opencode/skills/skill-creator/SKILL.md` | Create new AI skills |
| **github-pr** | `~/.opencode/skills/github-pr/SKILL.md` | Create pull requests |
| **jira-task** | `~/.opencode/skills/jira-task/SKILL.md` | Create Jira tasks |
| **jira-epic** | `~/.opencode/skills/jira-epic/SKILL.md` | Create Jira epics |
| **typescript** | `~/.opencode/skills/typescript/SKILL.md` | TypeScript patterns |
| **pytest** | `~/.opencode/skills/pytest/SKILL.md` | Python testing |
| **playwright** | `~/.opencode/skills/playwright/SKILL.md` | E2E testing |
| **django-drf** | `~/.opencode/skills/django-drf/SKILL.md` | Django REST Framework |
| **spring-boot-3** | `~/.opencode/skills/spring-boot-3/SKILL.md` | Spring Boot 3 |
| **java-21** | `~/.opencode/skills/java-21/SKILL.md` | Java 21 patterns |
| **react-19** | `~/.opencode/skills/react-19/SKILL.md` | React 19 patterns |
| **nextjs-15** | `~/.opencode/skills/nextjs-15/SKILL.md` | Next.js 15 patterns |
| **angular-core** | `~/.opencode/skills/angular/core/SKILL.md` | Angular core |
| **angular-architecture** | `~/.opencode/skills/angular/architecture/SKILL.md` | Angular architecture |
| **angular-forms** | `~/.opencode/skills/angular/forms/SKILL.md` | Angular forms |
| **angular-performance** | `~/.opencode/skills/angular/performance/SKILL.md` | Angular performance |

---

## How to Use Skills

### Priority Order
1. **Project-level skills** (`.agents/skills/`) - Override global if exists
2. **Global skills** (`~/.opencode/skills/`) - Fallback

### Loading a Skill
When working on a task that matches a skill:
1. Check project skills first (`.agents/skills/`)
2. If not found, check global skills (`~/.opencode/skills/`)
3. Load the skill using the `skill` tool before writing code

### Skill Loading Example
```
For Go testing → Load golang-testing from .agents/skills/ (project) or ~/.opencode/skills/
For Gin API → Load golang-gin-api from ~/.opencode/skills/
For SDD workflow → Load sdd-* from ~/.opencode/skills/
```

---

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

---

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

---

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
