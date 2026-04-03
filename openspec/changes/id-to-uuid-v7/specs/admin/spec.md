# Delta for Admin Module — UUIDv7 Migration

## MODIFIED Requirements

### Requirement: Admin Endpoint ID Parsing

All admin endpoint handlers MUST parse incoming ID parameters using `uuid.FromString()` instead of `strconv.ParseUint()`.

This applies to all admin CRUD operations for users, products, orders, categories, and any other resource managed through admin endpoints.

(Previously: Admin handlers used `strconv.ParseUint(c.Param("id"), 10, 32)`)

#### Scenario: Admin handler parses valid UUID parameter

- GIVEN a request with param `id = "01960c12-3456-7890-abcd-ef1234567890"`
- WHEN the admin handler parses the ID
- THEN `uuid.FromString()` returns a valid UUID with nil error
- AND the handler proceeds to query the repository

#### Scenario: Admin handler rejects malformed UUID parameter

- GIVEN a request with param `id = "abc123"`
- WHEN the admin handler attempts to parse the ID
- THEN `uuid.FromString()` returns an error
- AND the handler returns 400 Bad Request immediately

#### Scenario: Admin handler rejects integer ID parameter

- GIVEN a request with param `id = "123"`
- WHEN the admin handler attempts to parse the ID
- THEN `uuid.FromString()` returns an error (not a valid UUID format)
- AND the handler returns 400 Bad Request

### Requirement: Admin User Creation with UUID

Admin-created users SHALL receive UUIDv7 primary keys.

(Previously: Admin-created users received auto-incrementing integer IDs)

#### Scenario: Admin creates user with UUID ID

- GIVEN an authenticated admin user
- WHEN the admin creates a new user via POST /api/v1/admin/users
- THEN the created user receives a UUIDv7 ID
- AND the response contains the UUID string in the `id` field

### Requirement: RBAC Middleware with UUID User Identity

The RequireRole middleware SHALL work with UUID-based user identity from JWT claims.

The middleware extracts user_role from JWT claims (unchanged) but the user_id in context is now a UUID string.

(Previously: user_id in context was an integer)

#### Scenario: Admin user with UUID passes RequireRole middleware

- GIVEN a request with JWT containing user_role="admin" and user_id as UUID string
- WHEN passed through `RequireRole(RoleAdmin)` middleware
- THEN the request proceeds to the handler (c.Next() called)
- AND c.GetString("user_id") returns the UUID string

#### Scenario: Customer with UUID blocked by RequireRole

- GIVEN a request with JWT containing user_role="customer" and user_id as UUID string
- WHEN passed through `RequireRole(RoleAdmin)` middleware
- THEN the response status MUST be 403 Forbidden
- AND the request MUST be aborted

### Requirement: Admin Seed Script with UUID

The database seed script for the default admin user SHALL create a user with UUIDv7 ID.

(Previously: Seed script created user with auto-incrementing integer ID)

#### Scenario: Admin seed creates user with UUID

- GIVEN the application starts with database initialization
- WHEN the seed script executes
- THEN the admin user is created with a UUIDv7 ID
- AND the user email is "admin@bey.com"
- AND the user role is "admin"

#### Scenario: Admin seed idempotency with UUID

- GIVEN the database already contains admin@bey.com with UUID ID
- WHEN the seed script is executed again
- THEN there MUST still be exactly one user with email="admin@bey.com"
- AND no duplicate admin user is created
