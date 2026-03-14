# Delta for email/

## NEW Specification

### Requirement: SMTP Configuration

The system MUST support SMTP configuration for sending emails.

#### Scenario: Email service configured with valid SMTP settings

- GIVEN valid SMTP host, port, username, password
- WHEN the email service is initialized
- THEN it MUST establish a connection to the SMTP server
- AND be ready to send emails

### Requirement: Token Generation

The system MUST generate secure random tokens for email verification and password reset.

#### Scenario: Token generation produces unique tokens

- GIVEN a request for a new token
- WHEN the token is generated
- THEN it MUST be 32 bytes of cryptographically secure random data
- AND be encoded as a 64-character hex string

### Requirement: Token Hashing

The system MUST hash tokens before storage to prevent token leakage.

#### Scenario: Token hashing produces consistent hashes

- GIVEN a token
- WHEN the token is hashed using SHA-256
- THEN the hash MUST be deterministic (same input = same output)
- AND the hash MUST be different from the original token

### Requirement: Token Validation

The system MUST validate tokens securely.

#### Scenario: Valid token passes validation

- GIVEN a valid (non-expired) token
- WHEN the token is validated
- THEN the validation MUST succeed
- AND return the associated user

#### Scenario: Invalid token fails validation

- GIVEN an invalid or expired token
- WHEN the token is validated
- THEN the validation MUST fail
- AND return an appropriate error

### Requirement: Email Sending

The system MUST send verification and password reset emails.

#### Scenario: Verification email sent successfully

- GIVEN a user with valid email address
- WHEN a verification email is requested
- THEN an email MUST be sent to the user
- AND contain a verification link with token

#### Scenario: Password reset email sent successfully

- GIVEN a user with valid email address
- WHEN a password reset is requested
- THEN a password reset email MUST be sent
- AND contain a reset link with token
