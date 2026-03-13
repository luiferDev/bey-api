# Tasks: Improve-Opportunities

## Phase 1: Infrastructure (Security Foundation)

- [x] 1.1 Agregar `github.com/golang-jwt/jwt/v5` a `go.mod`
- [x] 1.2 Modificar `config.yaml`: agregar sección `security` con `allowed_origins`, `jwt_secret`, `jwt_expiry_hours`
- [x] 1.3 Modificar `internal/config/config.go`: agregar `SecurityConfig` struct y métodos helpers
- [x] 1.4 Crear `internal/shared/middleware/auth.go`: implementar JWT middleware con:
  - Extracción de token de header `Authorization: Bearer <token>`
  - Validación de token con jwt_secret
  - Extracción de user_id a gin.Context
  - Lista de paths públicos (sin auth)
- [x] 1.5 Modificar `internal/shared/middleware/middleware.go`: actualizar CORS para usar orígenes configurados
- [x] 1.6 Modificar `internal/modules/users/handler.go`: verificar que password_hash NO esté en UserResponse (ya está filtrado)

## Phase 2: Products Consistency

- [x] 2.1 Modificar `internal/modules/products/handler.go`: reemplazar `gin.H{}` por `response.ResponseHandler`
- [x] 2.2 Crear `internal/modules/products/service.go`: nuevo ProductService con:
  - CreateProduct() con validación de negocio
  - GetProductByID() 
  - GetProducts() con filtros
  - Errores tipados (ErrInvalidPrice, ErrNotFound)
- [x] 2.3 Modificar `internal/modules/products/dto.go`: agregar ProductResponseDTO si es necesario
- [x] 2.4 Modificar constructor de ProductHandler para incluir ProductService

## Phase 3: Health Check

- [x] 3.1 Crear `internal/shared/health.go`: función para verificar estado de DB (ping)
- [x] 3.2 Crear función para verificar estado de WorkerPool (workers activos, queue depth)
- [x] 3.3 Modificar `cmd/api/main.go`: endpoint `/health` mejorado que retorna:
  ```json
  {
    "status": "healthy",
    "timestamp": "2026-03-13T12:00:00Z",
    "dependencies": {
      "database": {"status": "healthy", "message": "connected"},
      "worker_pool": {"status": "healthy", "message": "running", "workers": 4, "queue_depth": 0}
    }
  }
  ```

## Phase 4: Testing

- [x] 4.1 Crear `internal/config/config_test.go`: tests para GetDBPassword con mock de env vars
- [x] 4.2 Crear `internal/shared/middleware/auth_test.go`: table-driven tests para JWT middleware
  - [x] 4.2.1 Token válido → 200 + user_id en context
  - [x] 4.2.2 Token expirado → 401
  - [x] 4.2.3 Token inválido → 401
  - [x] 4.2.4 Sin token → 401
  - [x] 4.2.5 Path público → sin auth
- [x] 4.3 Crear `internal/modules/products/handler_test.go`: table-driven tests para handlers
  - [x] 4.3.1 CreateProduct: valid, invalid JSON, missing required, negative price
  - [x] 4.3.2 GetProduct: found, not found
  - [x] 4.3.3 GetProducts: with filters, pagination
- [x] 4.4 Crear `internal/modules/products/service_test.go`: tests para lógica de negocio
  - [x] 4.4.1 CreateProduct con precio negativo → error
  - [x] 4.4.2 CreateProduct válido → success
- [x] 4.5 Crear `internal/shared/health_test.go`: tests para health check
  - [x] 4.5.1 DB healthy + worker pool healthy → 200
  - [x] 4.5.2 DB unhealthy → 503
  - [x] 4.5.3 Worker pool unhealthy → 503
- [x] 4.6 Crear `internal/modules/auth_integration_test.go`: integración JWT flow completo

## Phase 5: Verification & Cleanup

- [ ] 5.1 Ejecutar `go test -v ./...` y verificar que todos los tests pasen
- [ ] 5.2 Verificar con `go vet ./...` que no haya errores
- [ ] 5.3 Probar manualmente:
  - [ ] 5.3.1 CORS con origins permitidos → funciona
  - [ ] 5.3.2 CORS con origins no permitidos → rechazado
  - [ ] 5.3.3 JWT auth en endpoints protegidos
  - [ ] 5.3.4 Health endpoint retorna status correcto
- [ ] 5.4 Verificar que products NO exponga información sensible en responses
- [ ] 5.5 Actualizar `AGENTS.md` si hay nuevos patrones a documentar

---

## Implementation Order

```
1. Security Foundation (Phase 1)
   └─ Config → JWT Middleware → CORS
   
2. Products Consistency (Phase 2)
   └─ ResponseHandler → ProductService
   
3. Health Check (Phase 3)
   └─ Health functions → Main endpoint
   
4. Testing (Phase 4)
   └─ Unit tests → Integration tests
   
5. Verification (Phase 5)
   └─ Run tests → Manual verification → Cleanup
```

---

## Dependencies

```bash
# Add JWT library
go get github.com/golang-jwt/jwt/v5
```

---

## Notes

- **TDD**: Para tareas 4.2-4.5, primero escribir test que falla (RED), luego implementar (GREEN), luego refactorizar (REFACTOR)
- **Productos**: Revisar modelo en `internal/modules/products/model.go` para verificar que no haya campos sensibles
- **CORS dev**: Default origins `["http://localhost:3000", "http://localhost:8080"]` para swagger
- **JWT expiry**: 2 horas por defecto (configurable en security.jwt_expiry_hours)
