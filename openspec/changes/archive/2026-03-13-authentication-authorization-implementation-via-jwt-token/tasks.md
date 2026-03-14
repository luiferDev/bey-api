# Tasks: JWT Authentication & Authorization

## Phase 1: Infrastructure ✅

- [x] 1.1 Add JWT config to config.yaml (jwt_access_expiry, jwt_refresh_expiry, jwt_secret_key, jwt_issuer)
- [x] 1.2 Add JWT config struct to internal/config/config.go (JWTConfig with AccessExpiry, RefreshExpiry, SecretKey, Issuer)
- [x] 1.3 Create RefreshToken model in internal/modules/auth/model.go
- [x] 1.4 Add RefreshToken to AutoMigrate in main.go
- [x] 1.5 Create TokenClaims struct in internal/modules/auth/dto.go

## Phase 2: Core Implementation ✅

- [x] 2.1 Create AuthService interface in internal/modules/auth/service.go
- [x] 2.2 Implement TokenGenerator in internal/modules/auth/token.go
- [x] 2.3 Implement AuthService with Login
- [x] 2.4 Implement AuthService with Refresh (with rotation)
- [x] 2.5 Implement AuthService with Logout
- [x] 2.6 Implement AuthService with Validate
- [x] 2.7 Create AuthMiddleware in internal/modules/auth/middleware.go
- [x] 2.8 Create CookieHandler in internal/modules/auth/cookie.go

## Phase 3: RBAC Middleware ✅

- [x] 3.1 Create RBAC model in internal/shared/middleware/rbac.go
- [x] 3.2 Add Role field to User model (already exists)
- [x] 3.3 Create RBACMiddleware (RequireRole)
- [x] 3.4 Create RequirePermission helper

## Phase 4: Routes & Integration ✅

- [x] 4.1 Create auth routes (login, refresh, logout, validate)
- [x] 4.2 Wire auth service in main.go
- [x] 4.3 Add middleware chain to routes
- [x] 4.4 Protect admin routes with RequireRole
- [x] 4.5 CSRF middleware setup

## Phase 5: Testing ✅

- [x] 5.1 Unit tests para TokenGenerator
- [x] 5.2 Unit tests para AuthService
- [x] 5.3 Unit tests para AuthMiddleware
- [x] 5.4 Unit tests para RBAC middleware
- [x] 5.5 Unit tests para CSRF middleware
- [x] 5.6 Integration tests para auth flow completo
- [x] 5.7 Integration tests para RBAC protection

## Phase 6: Verification ✅

- [x] 6.1 Verify login flow end-to-end
- [x] 6.2 Verify refresh token rotation
- [x] 6.3 Verify RBAC protection
- [x] 6.4 Verify CSRF protection
- [x] 6.5 Verify cookie security flags
- [x] 6.6 Verify all existing tests pass

---

**Total: 34/34 tasks complete**
