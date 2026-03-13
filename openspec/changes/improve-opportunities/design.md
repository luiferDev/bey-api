# Design: Improve-Opportunities

## Technical Approach

Este diseño implementa las mejoras identificadas en la propuesta, siguiendo el enfoque por fases:

1. **Fase 1 - Seguridad**: Config desde env vars, CORS restringido, JWT auth middleware
2. **Fase 2 - Consistencia**: Unificar ResponseHandler en products
3. **Fase 3 - Arquitectura**: Service layer para products, health check mejorado

---

## Architecture Decisions

### Decision: Environment Variable Configuration

**Choice**: Agregar helper `getEnvOrDefault()` que lee env vars con fallback a YAML

**Alternatives considered**: 
- Usar viper (too heavy, introduce nueva dependencia)
- Solo env vars (breaking change para desarrollo local)

**Rationale**: Mantiene backward compatibility con config.yaml existente mientras permite overrides por seguridad en producción.

### Decision: CORS Configuration

**Choice**: Agregar lista de allowed origins en config.yaml con opción de desarrollo

**Alternatives considered**:
- Hardcode lista de orígenes (difícil de mantener)
- Usar variable de entorno única (limitado para múltiples frontends)

**Rationale**: Flexible para múltiples orígenes configurables, con fallback a `*` en modo debug.

### Decision: JWT Configuration

**Choice**: JWT Secret en config.yaml, token expiry de 2 horas para desarrollo

**Alternatives considered**:
- Generar JWT secret en startup (difícil de debuggear)
- Token expiry largo (seguridad)

**Rationale**: Mantener en config facilita rotación, 2 horas es razonable para desarrollo.

### Decision: CORS for Development

**Choice**: Allowed origins configurables, con defaults para desarrollo: `localhost:3000`, `localhost:8080` (swagger)

**Alternatives considered**:
- Solo `*` en desarrollo (inseguro)
- Hardcode origins (difícil de mantener)

**Rationale**: Flexible pero con defaults útiles para desarrollo local.

### Decision: Products Security

**Choice**: Verificar que productos no expongan información sensible en responses

**Alternatives considered**:
- Asumir que no hay sensitive data (riesgoso)
- Crear DTOs para todo (overhead)

**Rationale**: Revisar modelo de productos y crear DTOs solo si es necesario.

### Decision: Products Response Handler

**Choice**: Refactor products handler para usar `response.ResponseHandler` existente

**Alternatives considered**:
- Crear nuevo response package (duplicación)
- Modificar ResponseHandler existente (podría romper users/orders)

**Rationale**: Usa código existente, mantiene consistencia, mínimo cambio.

### Decision: Product Service Layer

**Choice**: Crear servicio que encapsula lógica de negocio, handlers llaman servicios

**Alternatives considered**:
- Dejar lógica en handlers (violates thin handler principle)
- Reescribir todo a Clean Architecture (out of scope)

**Rationale**: Sigue el patrón existente en users/orders, separable para testing.

---

## Data Flow

### Auth Flow
```
Request → JWT Middleware → Validate Token → 
  ├─ Valid: Add user_id to context → Handler
  └─ Invalid: Return 401 → Client
```

### Health Check Flow
```
GET /health → Check DB Ping → Check Worker Pool → 
  ├─ All Healthy: 200 OK
  └─ Any Unhealthy: 503 Service Unavailable
```

### Product Creation Flow (with Service)
```
POST /products → Handler → Bind JSON → 
  ProductService.Create() → Validate → 
    Repository.Create() → ResponseHandler.Created()
```

---

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `config.yaml` | Modify | Agregar `allowed_origins` y `jwt_secret` |
| `internal/config/config.go` | Modify | Agregar CORS y JWT config, LoadFromEnv() |
| `internal/shared/middleware/auth.go` | Create | JWT middleware |
| `internal/shared/middleware/middleware.go` | Modify | CORS restrict a orígenes configurados |
| `internal/modules/users/handler.go` | Modify | Agregar filter de password_hash |
| `internal/modules/products/handler.go` | Modify | Usar ResponseHandler |
| `internal/modules/products/service.go` | Create | Product service layer |
| `internal/modules/products/dto.go` | Modify | Agregar ProductResponseDTO |
| `cmd/api/main.go` | Modify | Health check mejorado, inicializar auth |
| `internal/modules/*/handler_test.go` | Create | Tests unitarios |
| `internal/modules/*/integration_test.go` | Create | Tests de integración |

---

## Interfaces / Contracts

### Security Config Additions
```go
// internal/config/config.go
type Config struct {
    App         AppConfig
    Database    DatabaseConfig
    Concurrency concurrency.ConcurrencyConfig
    Security    SecurityConfig  // NEW
}

type SecurityConfig struct {
    AllowedOrigins []string `yaml:"allowed_origins"`
    JWTSecret     string   `yaml:"jwt_secret"`
    JWTExpiryHours int    `yaml:"jwt_expiry_hours"`  // Default: 2 for dev
}

// Default origins for development
func (s *SecurityConfig) GetAllowedOrigins() []string {
    if len(s.AllowedOrigins) == 0 {
        return []string{"http://localhost:3000", "http://localhost:8080"}
    }
    return s.AllowedOrigins
}

func (c *Config) GetDBPassword() string {
    // Check env first, fallback to YAML
    if p := os.Getenv("DB_PASSWORD"); p != "" {
        return p
    }
    return c.Database.Password
}
```

### JWT Middleware
```go
// internal/shared/middleware/auth.go
type AuthMiddleware struct {
    jwtSecret []byte
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
    return &AuthMiddleware{jwtSecret: []byte(secret)}
}

func (m *AuthMiddleware) Handler() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Extract token from Authorization header
        // Validate with jwt-secret
        // Set user_id in c.Set("user_id", userID)
        // c.Next() on success, c.AbortWithStatus(401) on failure
    }
}

// Public paths that skip auth
var publicPaths = []string{
    "/health",
    "/api/v1/products",
    "/api/v1/categories",
    "/api/v1/users",  // POST only (registration)
}
```

### Product Service (nuevo)
```go
// internal/modules/products/service.go
type ProductService struct {
    productRepo  *ProductRepository
    categoryRepo *CategoryRepository
}

func NewProductService(productRepo, categoryRepo *ProductRepository) *ProductService {
    return &ProductService{
        productRepo:  productRepo,
        categoryRepo: categoryRepo,
    }
}

func (s *ProductService) CreateProduct(req CreateProductRequest) (*Product, error) {
    // Business validation
    if req.BasePrice <= 0 {
        return nil, ErrInvalidPrice
    }
    // ... create logic
}

func (s *ProductService) GetProductByID(id uint) (*Product, error) {
    // ... get logic
}
```

### Health Check Response
```go
// Enhanced health response
type HealthResponse struct {
    Status       string                 `json:"status"`
    Timestamp    time.Time              `json:"timestamp"`
    Dependencies map[string]Dependency `json:"dependencies"`
}

type Dependency struct {
    Status  string `json:"status"`
    Message string `json:"message,omitempty"`
    // Optional extra info
    Workers  int `json:"workers,omitempty"`
    QueueDepth int `json:"queue_depth,omitempty"`
}
```

### User Response DTO (evita password hash)
```go
// internal/modules/users/dto.go
type UserResponse struct {
    ID        uint      `json:"id"`
    Email     string    `json:"email"`
    FirstName string    `json:"first_name"`
    LastName  string    `json:"last_name"`
    Role      string    `json:"role"`
    Active    bool      `json:"active"`
    CreatedAt time.Time `json:"created_at"`
    // PasswordHash is intentionally omitted
}
```

---

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| **Unit - Config** | LoadFromEnv, GetDBPassword | Mock os.Getenv |
| **Unit - Auth** | Valid token, Expired token, Missing token, Invalid token | Table-driven |
| **Unit - Handler** | All endpoints success/error cases | Table-driven + httptest |
| **Unit - Service** | Business logic validation | Unit tests with mock repo |
| **Unit - Response** | ResponseHandler formats | Assertion tests |
| **Integration - Auth** | Full JWT flow: login → protected endpoint | HTTP tests |
| **Integration - Health** | DB up/down, worker pool up/down | Mock/skip DB tests |

### Test Structure Example
```go
// internal/modules/products/handler_test.go
func TestProductHandler_Create(t *testing.T) {
    tests := []struct {
        name       string
        body       string
        wantStatus int
        wantErr    bool
    }{
        {"valid", `{"name":"Test","base_price":10}`, http.StatusCreated, false},
        {"invalid json", `{invalid}`, http.StatusBadRequest, true},
        {"missing required", `{}`, http.StatusBadRequest, true},
        {"negative price", `{"name":"Test","base_price":-1}`, http.StatusBadRequest, true},
    }
    
    // setup handler with mock repos
    handler := NewProductHandler(mockCategoryRepo, mockProductRepo, ...)
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(tt.body))
            req.Header.Set("Content-Type", "application/json")
            w := httptest.NewRecorder()
            
            handler.CreateProduct(w, req)
            
            if w.Code != tt.wantStatus {
                t.Errorf("got %d; want %d", w.Code, tt.wantStatus)
            }
        })
    }
}
```

---

## Migration / Rollout

**No migration required** - Estos son cambios de código, no de datos.

### Rollback Steps
1. `git checkout config.yaml` - Restaurar credenciales originales
2. `git checkout internal/shared/middleware/middleware.go` - CORS abierto
3. Eliminar auth middleware del router en main.go
4. Revertir cambios en products handler

### Feature Flags (optional para futuras fases)
- `AUTH_ENABLED` - Enable/disable auth middleware
- `STRICT_CORS` - Enable/disable origin restrictions

---

## Open Questions

- [x] **JWT Secret**: En config.yaml
- [x] **Token expiry**: 2 horas para desarrollo (configurable)
- [x] **Allowed origins**: localhost:3000, localhost:8080 (swagger) para desarrollo
- [x] **Password en responses**: Users ya filtrado, verificar products no exponga nada sensible

---

## Dependencies

```go
// go.mod additions
github.com/golang-jwt/jwt/v5 v5.2.0
```

No se necesitan más dependencias - usamos `log/slog` de stdlib para logging y `bcrypt` ya está presente.
