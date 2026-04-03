# Delta for Auth Module — UUIDv7 Migration

## MODIFIED Requirements

### Requirement: JWT Claims — UserID Type

The system SHALL use UUID string for the `user_id` claim in JWT tokens instead of integer.

JWT payload structure:
```json
{
  "user_id": "01960c12-3456-7890-abcd-ef1234567890",
  "email": "user@example.com",
  "role": "customer",
  "exp": 1712345678,
  "iat": 1712344778
}
```

(Previously: `user_id` was an integer, e.g., `"user_id": 123`)

#### Scenario: JWT issued with UUID user_id

- GIVEN a user with UUID `01960c12-3456-7890-abcd-ef1234567890` logs in successfully
- WHEN the JWT access token is generated
- THEN the token payload contains `"user_id": "01960c12-3456-7890-abcd-ef1234567890"`

#### Scenario: JWT refresh token contains UUID user_id

- GIVEN a user with UUID `01960c12-3456-7890-abcd-ef1234567890`
- WHEN a refresh token is issued
- THEN the refresh token payload contains the UUID user_id

### Requirement: Auth Middleware — User ID Extraction

The auth middleware SHALL extract user_id as a string from JWT claims instead of an unsigned integer.

The middleware SHALL use `c.GetString("user_id")` instead of `c.GetUint("user_id")` to retrieve the authenticated user's ID from Gin context.

(Previously: Middleware used `c.GetUint("user_id")`)

#### Scenario: Middleware sets UUID user_id in context

- GIVEN a valid JWT with `user_id: "01960c12-3456-7890-abcd-ef1234567890"`
- WHEN the auth middleware processes the request
- THEN `c.GetString("user_id")` returns `"01960c12-3456-7890-abcd-ef1234567890"`
- AND `c.Get("user_id")` returns the UUID string

#### Scenario: Middleware rejects JWT with malformed user_id

- GIVEN a JWT with `user_id: "not-a-uuid"`
- WHEN the auth middleware processes the request
- THEN the middleware SHALL reject the token as invalid
- AND return 401 Unauthorized

### Requirement: JWT Full Invalidation on Deployment

All existing JWT tokens SHALL become invalid upon deployment of the UUIDv7 migration.

Users MUST re-authenticate to receive new tokens with UUID-based user_id claims.

Refresh tokens stored in Redis SHALL be cleared on deployment.

#### Scenario: Legacy integer-based JWT is rejected

- GIVEN a JWT token issued before the migration (with integer user_id)
- WHEN the token is validated after deployment
- THEN the token SHALL be rejected
- AND the user MUST re-authenticate

#### Scenario: User re-authenticates after migration

- GIVEN a user with valid credentials whose old JWT is now invalid
- WHEN the user logs in again
- THEN a new JWT with UUID-based user_id is issued
- AND the user can access protected resources

### Requirement: Password Reset Flow with UUID

The password reset flow SHALL work with UUID-based user identification.

(Previously: Password reset tokens were associated with integer user IDs)

#### Scenario: Password reset requested with UUID user

- GIVEN a registered user with UUID `01960c12-3456-7890-abcd-ef1234567890`
- WHEN the user calls POST /api/v1/auth/forgot-password
- THEN the system generates a reset token associated with the UUID user
- AND sends a password reset email

#### Scenario: Password reset completed with UUID user

- GIVEN a user with a valid, non-expired password reset token
- WHEN the user calls POST /api/v1/auth/reset-password
- THEN the password is updated for the UUID-identified user
- AND the reset token is invalidated

### Requirement: Email Verification with UUID

Email verification tokens SHALL be associated with UUID-based user IDs.

(Previously: Verification tokens referenced integer user IDs)

#### Scenario: Unverified UUID user cannot access protected resources

- GIVEN a user with UUID who has registered but not verified their email
- WHEN the user attempts to access a protected endpoint
- THEN the request MUST be rejected with 403 Forbidden
- AND the response MUST indicate "email not verified"
