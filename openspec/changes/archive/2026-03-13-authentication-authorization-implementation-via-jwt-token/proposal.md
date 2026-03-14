# Proposal: JWT-based Authentication and Authorization Implementation

## Intent

Implement secure JWT-based authentication and authorization for the Bey API e-commerce platform. Current state has a basic JWT middleware that validates tokens but lacks complete auth flows: no login endpoint, no cookie handling, no refresh token mechanism, no role-based access control, and no CSRF protection.

## Scope

### In Scope
- JWT access token (15 min) with user_id and role in claims
- JWT refresh token (7 days) via secure HTTP-only cookies
- Secure cookie configuration (HttpOnly, SameSite=Strict, Secure)
- Bearer token support via Authorization header
- Refresh token rotation on each use
- CSRF protection middleware
- Role-based access control (RBAC) middleware
- Login, logout, and refresh endpoints
- Admin role enforcement

### Out of Scope
- OAuth2/Social login
- Two-factor authentication (2FA)
- Email verification flows
- Password reset functionality
- Rate limiting

## Approach

Enhance the existing JWT middleware in `internal/shared/middleware/auth.go` by extending token claims, adding cookie-based handling, implementing refresh token rotation, and creating RBAC middleware chain.

## Risks
- Token leakage via XSS → mitigated with HttpOnly cookies + rotation
- CSRF attacks → mitigated with double-submit cookie pattern
- Role escalation → mitigated with validation at each endpoint
