# Delta for auth/

## ADDED Requirements

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
