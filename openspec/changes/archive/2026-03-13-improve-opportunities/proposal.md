# Proposal: Improve-Opportunities - Security, Consistency & Architecture

## Intent

Este cambio aborda problemas críticos de seguridad, inconsistencias de código y gaps de arquitectura en Bey API. El proyecto tiene una base sólida pero necesita hardening para producción.

**Problemas específicos a resolver:**
1. Credenciales hardcodeadas en config.yaml
2. CORS abierto a todos los orígenes (`*`)
3. Sin middleware de autenticación
4. Inconsistencia en patrones de respuesta (products usa `gin.H`, users/orders usa `response.ResponseHandler`)
5. Falta service layer en products
6. Sin logging estructurado, métricas ni health checks

## Scope

### In Scope
- **Seguridad**:
  - Mover credenciales de DB a environment variables
  - Restringir CORS a orígenes específicos
  - Implementar auth middleware básico (JWT)
  - Ocultar password hash en respuestas de usuario
  
- **Consistencia**:
  - Unificar patrón de respuesta en products (adoptar `response.ResponseHandler`)
  - Estandarizar manejo de errores
  
- **Arquitectura**:
  - Agregar service layer a products
  - Agregar logging estructurado (slog)
  - Agregar health check completo (DB + worker pool)
  
- **Testing**:
  - Tests unitarios para handlers con table-driven tests
  - Tests de integración para auth middleware
  - Tests para health checks

### Out of Scope
- Migración completa a Clean Architecture
- Sistema de roles/permisos avanzado
- Rate limiting (ya existe deshabilitado)
- Métricas y tracing (para fase posterior)
- Pagination metadata

## Approach

### Fase 1: Seguridad Crítica
1. Crear helper para leer config desde env vars con fallback a YAML
2. Actualizar `config.yaml` con placeholders `{{.DB_PASSWORD}}`
3. Restringir CORS en middleware.go
4. Agregar JWT middleware básico
5. Filtrar password hash en responses

### Fase 2: Consistencia
1. Refactor products handler para usar `response.ResponseHandler`
2. Unificar estructura de errores

### Fase 3: Arquitectura
1. Crear service layer para products
2. Agregar structured logging con slog
3. Mejorar health check endpoint

### Testing (aplicando TDD)
- **Table-driven tests** para handlers
- **Subtests** para casos de éxito/error
- **HTTP handler testing** con `httptest`
- **Integration tests** para auth y health

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `config.yaml` | Modified | Agregar placeholders para env vars |
| `internal/config/` | Modified | Cargar config desde env vars |
| `internal/shared/middleware/middleware.go` | Modified | CORS restringido |
| `internal/shared/middleware/auth.go` | New | JWT middleware |
| `internal/modules/products/handler.go` | Modified | Usar response package |
| `internal/modules/products/service.go` | New | Service layer |
| `internal/modules/users/handler.go` | Modified | Filtrar password hash |
| `cmd/api/main.go` | Modified | Health check mejorado, DI |
| `internal/modules/*/handler_test.go` | New | Tests unitarios |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Romper API existente con cambios de response | Med | Tests antes y después, backward compatibility |
| Auth middleware rompe endpoints públicos | Med | Tests específicos por endpoint |
| Refactor sin tests rompe funcionalidad | Alta | Escribir tests primero (TDD) |

## Rollback Plan

1. Revertir cambios en `config.yaml` (volver a credenciales hardcodeadas)
2. Revertir CORS a `*` 
3. Eliminar auth middleware del router
4. Revertir cambios en products handler
5. `git checkout` para restauración completa si hay problemas

## Dependencies

- `golang-jwt/jwt` para auth middleware
- No nuevas dependencias para logging (usar `log/slog` stdlib)

## Success Criteria

- [ ] Credenciales de DB no expuestas en código fuente
- [ ] CORS restringido a orígenes configurables
- [ ] Endpoints sensibles protegidos con JWT
- [ ] Password hash nunca expuesto en JSON responses
- [ ] Products handler usa `response.ResponseHandler` igual que users/orders
- [ ] Products tiene service layer
- [ ] Health check retorna status de DB y worker pool
- [ ] Todos los handlers tienen tests unitarios con coverage > 80%
- [ ] Tests de integración para auth pasan
- [ ] API sigue funcionando sin breaking changes

---

## Testing Strategy (golang-testing skill)

### Table-Driven Tests para Handlers
```go
func TestProductHandler_Create(t *testing.T) {
    tests := []struct {
        name       string
        body       string
        wantStatus int
        wantErr    bool
    }{
        {"valid request", `{"name":"Test","price":10}`, http.StatusCreated, false},
        {"invalid json", `{invalid}`, http.StatusBadRequest, true},
        {"missing required", `{}`, http.StatusBadRequest, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### HTTP Handler Testing
```go
func TestHealthHandler(t *testing.T) {
    req := httptest.NewRequest(http.MethodGet, "/health", nil)
    w := httptest.NewRecorder()
    
    HealthHandler(w, req)
    
    if w.Code != http.StatusOK {
        t.Errorf("got %d; want %d", w.Code, http.StatusOK)
    }
}
```

### Integration Tests
- `auth_integration_test.go` - JWT flow completo
- `health_integration_test.go` - DB + worker pool checks

---

## Patrones de Gin Aplicados (golang-gin-api skill)

- **Thin Handlers**: handlers solo bindean input, llaman service, formatean response
- **Request Binding**: `ShouldBindJSON` con validation tags
- **Error Handling**: `handleServiceError` centralizado
- **Graceful Shutdown**: già implementado en main.go
- **Trusted Proxies**: configurar para producción
