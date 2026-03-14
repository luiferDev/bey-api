# Design: JWT Authentication & Authorization

## Technical Approach

Implement secure JWT-based authentication with dual tokens (access + refresh), cookie-based storage, RBAC, and CSRF protection.

## Architecture Decisions

### Decision: Dual Token Strategy

**Choice**: Access token (15 min) + Refresh token (7 days)
**Alternatives**: Single token, longer-lived access token
**Rationale**: Short access token reduces exposure window; refresh token allows persistent sessions

### Decision: Cookie-Based Storage

**Choice**: HTTP-only cookies for tokens
**Alternatives**: LocalStorage, SessionStorage
**Rationale**: HttpOnly prevents XSS token theft

### Decision: Refresh Token Rotation

**Choice**: New refresh token on each use
**Alternatives**: Static refresh token
**Rationale**: Rotation prevents token replay attacks

### Decision: Middleware Chain Order

**Choice**: CORS → Logger → RateLimit → Auth → RBAC → Handler
**Rationale**: Auth must run before RBAC to populate user context

## Data Flow

```
User Login
    ↓
POST /api/v1/auth/login
    ↓
AuthService.Login → Validate credentials
    ↓
GenerateAccessToken + GenerateRefreshToken
    ↓
Store RefreshToken (hashed) in DB
    ↓
Set cookies (HttpOnly, Secure, SameSite)
    ↓
Return access_token + expires_in

Protected Request
    ↓
AuthMiddleware → Extract token (Bearer header or cookie)
    ↓
Validate token → Extract claims (user_id, email, role)
    ↓
Set in gin.Context
    ↓
RBACMiddleware → Check role permissions
    ↓
Handler
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/modules/auth/model.go` | Created | RefreshToken model |
| `internal/modules/auth/dto.go` | Created | TokenClaims, DTOs |
| `internal/modules/auth/service.go` | Created | AuthService |
| `internal/modules/auth/token.go` | Created | TokenGenerator |
| `internal/modules/auth/middleware.go` | Created | AuthMiddleware |
| `internal/modules/auth/cookie.go` | Created | CookieHandler |
| `internal/modules/auth/routes.go` | Created | Auth routes |
| `internal/shared/middleware/rbac.go` | Created | RBAC middleware |
| `internal/shared/middleware/csrf.go` | Created | CSRF middleware |
| `config.yaml` | Modified | JWT config |
| `internal/config/config.go` | Modified | JWTConfig struct |
| `cmd/api/main.go` | Modified | Wire auth |

## Testing Strategy

| Layer | Coverage |
|-------|----------|
| Unit | TokenGenerator, AuthService, Middleware |
| Integration | Full auth flow, RBAC protection |
| Total Tests | 150+ |

## Migration / Rollout

No migration required - fresh feature.
