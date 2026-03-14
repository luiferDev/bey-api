# Delta for Auth

## ADDED Requirements

### Requirement: Google OAuth2 Login

The system MUST allow users to authenticate using their Google account via OAuth2.

#### Scenario: New user logs in with Google

- GIVEN a user with a valid Google account
- WHEN the user initiates Google OAuth2 login
- AND the user grants permission to access their profile
- THEN a new user account MUST be created with data from Google
- AND the response MUST contain valid JWT access_token
- AND the response MUST contain valid JWT refresh_token
- AND the user's email MUST be marked as verified

#### Scenario: Existing user logs in with Google

- GIVEN a user who previously registered with email/password
- AND the user initiates Google OAuth2 login with the same email
- THEN the existing user account MUST be found
- AND the user's Google profile data (name, avatar) MUST be updated
- AND the response MUST contain valid JWT access_token
- AND the response MUST contain valid JWT refresh_token

#### Scenario: OAuth2 callback with invalid state

- GIVEN a user who initiated OAuth2 login
- WHEN the callback is received with an invalid or missing state parameter
- THEN the request MUST be rejected with 400 Bad Request
- AND no user session MUST be created

#### Scenario: OAuth2 token exchange fails

- GIVEN a user who initiated OAuth2 login
- WHEN the Google token exchange fails (invalid code, expired, etc.)
- THEN the request MUST fail with 401 Unauthorized
- AND the user MUST be redirected to login page with error

### Requirement: OAuth2 Configuration

The system MUST support different OAuth2 configurations for development and production environments.

#### Scenario: OAuth2 configured for development

- GIVEN the system is running in development mode
- WHEN the user initiates Google OAuth2 login
- THEN the redirect_url MUST use the development callback URL from config
- AND the OAuth2 client MUST use the development client_id and client_secret

#### Scenario: OAuth2 configured for production

- GIVEN the system is running in production mode
- WHEN the user initiates Google OAuth2 login
- THEN the redirect_url MUST use the production callback URL from config
- AND the OAuth2 client MUST use the production client_id and client_secret

### Requirement: OAuth2 User Data

The system MUST store and update user data from Google OAuth2 provider.

#### Scenario: User data stored from OAuth2

- GIVEN a new user logging in via Google OAuth2
- THEN the following data MUST be stored:
  - Email (from Google)
  - First Name (from Google profile)
  - Last Name (from Google profile)
  - Avatar URL (from Google profile picture)
  - OAuth Provider: "google"
  - OAuth Provider ID (Google user ID)
  - Email Verified: true

#### Scenario: User data updated on subsequent OAuth2 logins

- GIVEN an existing user who previously logged in via Google
- WHEN the user logs in again via Google OAuth2
- THEN the following data MUST be updated if changed:
  - First Name
  - Last Name
  - Avatar URL
