# Delta for Middleware — UUIDv7 Migration

## MODIFIED Requirements

### Requirement: Auth Middleware — User ID Context Type

The authentication middleware SHALL store user_id as a string (UUID) in Gin context instead of an unsigned integer.

The middleware SHALL use `c.Set("user_id", claims.UserID)` where `claims.UserID` is a UUID string.

(Previously: Middleware used `c.Set("user_id", claims.UserID)` with uint type)

#### Scenario: Auth middleware sets UUID string in context

- GIVEN a valid JWT with `user_id: "01960c12-3456-7890-abcd-ef1234567890"`
- WHEN the auth middleware processes the request
- THEN `c.GetString("user_id")` returns `"01960c12-3456-7890-abcd-ef1234567890"`
- AND `c.GetUint("user_id")` MUST NOT be used (will panic or return zero)

#### Scenario: Downstream handlers use GetString for user_id

- GIVEN the auth middleware has set user_id in context
- WHEN a downstream handler accesses the user ID
- THEN the handler MUST use `c.GetString("user_id")`
- AND the handler MUST NOT use `c.GetUint("user_id")`

### Requirement: RequireRole Middleware with UUID Context

The RequireRole middleware SHALL work correctly when user_id is a UUID string in context.

The middleware logic for role checking is unchanged — it reads `user_role` from context (which remains a string).

(Previously: user_id in context was uint, but user_role was always string — no change to role logic)

#### Scenario: RequireRole checks role with UUID user context

- GIVEN a request with JWT containing user_role="admin" and user_id as UUID string
- WHEN passed through `RequireRole(RoleAdmin)` middleware
- THEN the middleware reads user_role from context (unchanged behavior)
- AND the request proceeds to the handler

### Requirement: Rate Limiter Middleware (Unchanged)

The rate limiting middleware is NOT affected by the UUID migration.

Rate limiter operates on client IP/request patterns, not on user IDs.

(This requirement is unchanged)

### Requirement: Bearer Token Support with UUID Claims

The system MUST support Authorization header for API clients with JWT tokens containing UUID user_id claims.

(Previously: JWT tokens contained integer user_id claims)

#### Scenario: Valid Bearer token with UUID user_id is accepted

- GIVEN a valid access token with UUID user_id claim
- WHEN the request includes Authorization: Bearer {token}
- THEN the request MUST be authenticated
- AND the user_id extracted from the token is a UUID string
