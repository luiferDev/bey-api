# Delta for Security

## ADDED Requirements

### Requirement: Environment Variable Configuration

The system SHALL support reading database credentials from environment variables with fallback to YAML config file.

The configuration loader MUST check for environment variables first:
- `DB_HOST` → database.host
- `DB_PORT` → database.port
- `DB_USER` → database.user
- `DB_PASSWORD` → database.password
- `DB_NAME` → database.name

If environment variable is not set, the system SHALL fallback to values in `config.yaml`.

#### Scenario: Environment variable is set

- GIVEN environment variable `DB_PASSWORD=secret123` is set
- WHEN the application starts
- THEN the database connection uses password `secret123`
- AND the value from `config.yaml` is ignored

#### Scenario: Environment variable is not set

- GIVEN environment variable `DB_PASSWORD` is not set
- WHEN the application starts
- THEN the database connection uses password from `config.yaml`

---

### Requirement: CORS Restricted Origins

The system SHALL restrict CORS to explicitly allowed origins, not allow all origins (`*`).

The CORS middleware MUST read allowed origins from configuration.

#### Scenario: Request from allowed origin

- GIVEN CORS allowed origins include `https://example.com`
- WHEN a browser makes a cross-origin request with `Origin: https://example.com`
- THEN the response includes `Access-Control-Allow-Origin: https://example.com`

#### Scenario: Request from disallowed origin

- GIVEN CORS allowed origins include only `https://example.com`
- WHEN a browser makes a cross-origin request with `Origin: https://evil.com`
- THEN the response does NOT include `Access-Control-Allow-Origin`
- AND the browser blocks the request (CORS error)

#### Scenario: No origin configured (development)

- GIVEN CORS allowed origins is empty or not configured
- WHEN a cross-origin request is made
- THEN the system SHOULD allow all origins for development compatibility
- AND log a warning about open CORS

---

### Requirement: JWT Authentication Middleware

The system SHALL provide JWT-based authentication for protected endpoints.

The auth middleware MUST:
- Validate JWT token from `Authorization: Bearer <token>` header
- Extract user ID from token claims
- Add user context to request for handlers
- Return 401 for invalid/expired tokens
- Skip validation for public endpoints

#### Scenario: Valid JWT token

- GIVEN a valid JWT token with `{"user_id": 123, "exp": 9999999999}`
- WHEN a protected endpoint receives request with `Authorization: Bearer <token>`
- THEN the request proceeds to handler
- AND handler can access user ID from context

#### Scenario: Missing JWT token

- GIVEN no JWT token in request
- WHEN accessing a protected endpoint
- THEN response is HTTP 401 Unauthorized
- AND response body includes `{"error": "missing authorization token"}`

#### Scenario: Invalid JWT token

- GIVEN an invalid or expired JWT token
- WHEN accessing a protected endpoint
- THEN response is HTTP 401 Unauthorized
- AND response body includes `{"error": "invalid or expired token"}`

#### Scenario: Public endpoint without auth

- GIVEN a public endpoint (e.g., `/api/v1/products`, `/health`)
- WHEN any request is made
- THEN the request proceeds without JWT validation
- AND no authentication error is returned

---

### Requirement: Password Hash Protection

The system MUST NOT expose password hash in JSON API responses.

User responses SHALL exclude the `password_hash` field from all endpoints.

#### Scenario: Get user returns password hidden

- GIVEN a user exists in database with password_hash
- WHEN client requests `GET /api/v1/users/1`
- THEN response does NOT include `password_hash` field
- AND all other user fields are returned normally

#### Scenario: List users returns password hidden

- GIVEN multiple users exist in database
- WHEN client requests `GET /api/v1/users`
- THEN each user in the array does NOT include `password_hash` field

---

## MODIFIED Requirements

### Requirement: Configuration Loading (Previously: Hardcoded credentials)

The system SHOULD load configuration from YAML file with environment variable overrides.

(Previously: Credentials were hardcoded in config.yaml)
