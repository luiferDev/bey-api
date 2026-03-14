# Authentication Specification

## Purpose

This specification defines the authentication flow for the Bey API e-commerce platform.

## Requirements

### Requirement: JWT Access Token

The system MUST issue JWT access tokens with a 15-minute expiration time.

#### Scenario: Valid login returns access token

- GIVEN a user with valid credentials
- WHEN the user submits POST /api/v1/auth/login
- THEN the response MUST contain a valid access_token
- AND the token MUST include user_id, email, and role in the payload

#### Scenario: Access token expires after 15 minutes

- GIVEN an access token that was issued 15 minutes ago
- WHEN the token is validated
- THEN the validation MUST fail with an "expired" error

### Requirement: JWT Refresh Token

The system MUST issue refresh tokens with a 7-day expiration time.

#### Scenario: Refresh token grants new access token

- GIVEN a valid refresh token
- WHEN the user calls POST /api/v1/auth/refresh
- THEN the response MUST contain a new access_token
- AND the refresh token MUST be rotated (new token issued)

#### Scenario: Expired refresh token is rejected

- GIVEN a refresh token that expired 8 days ago
- WHEN the user calls POST /api/v1/auth/refresh
- THEN the request MUST fail with 401 Unauthorized

### Requirement: Cookie-Based Token Storage

The system MUST store tokens in HTTP-only cookies.

#### Scenario: Tokens are stored in secure cookies

- GIVEN a successful login
- WHEN cookies are inspected
- THEN refresh_token cookie MUST have HttpOnly=true
- AND refresh_token cookie MUST have Secure=true in production
- AND refresh_token cookie MUST have SameSite=Strict

### Requirement: Bearer Token Support

The system MUST support Authorization header for API clients.

#### Scenario: Valid Bearer token is accepted

- GIVEN a valid access token
- WHEN the request includes Authorization: Bearer {token}
- THEN the request MUST be authenticated

#### Scenario: Bearer token in query string is rejected

- GIVEN a token in the URL query string
- WHEN the request is made
- THEN the request MUST be rejected for security reasons

### Requirement: Email Verification Required

The system MUST require email verification before allowing full account access.

#### Scenario: Unverified user cannot access protected resources

- GIVEN a user who has registered but not verified their email
- WHEN the user attempts to access a protected endpoint
- THEN the request MUST be rejected with 403 Forbidden
- AND the response MUST indicate "email not verified"

#### Scenario: Verified user can access protected resources

- GIVEN a user who has verified their email
- WHEN the user accesses a protected endpoint with valid credentials
- THEN the request MUST be allowed

### Requirement: Password Reset Flow

The system MUST provide a secure password reset flow.

#### Scenario: User requests password reset

- GIVEN a registered user with a valid email
- WHEN the user calls POST /api/v1/auth/forgot-password with valid email
- THEN the system MUST send a password reset email
- AND the response MUST return 200 OK (without confirming email existence)

#### Scenario: User resets password with valid token

- GIVEN a user with a valid, non-expired password reset token
- WHEN the user calls POST /api/v1/auth/reset-password with valid token and new password
- THEN the password MUST be updated
- AND the reset token MUST be invalidated
- AND the user MUST be able to login with the new password

#### Scenario: User cannot reset password with expired token

- GIVEN a user with an expired password reset token
- WHEN the user calls POST /api/v1/auth/reset-password
- THEN the request MUST fail with 400 Bad Request
- AND the response MUST indicate "token expired"
