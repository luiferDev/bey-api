# Tasks: OAuth2 Google Login

## Phase 1: Infrastructure

- [x] 1.1 Add `golang.org/x/oauth2` to `go.mod`
- [x] 1.2 Add `oauth.google` section to `config.yaml` with client_id, client_secret, redirect_url
- [x] 1.3 Add `OAuthConfig` struct to `internal/config/config.go`
- [x] 1.4 Add `GetOAuthConfig()` method to Config
- [x] 1.5 Add OAuth fields to User model: `OAuthProvider`, `OAuthProviderID`, `AvatarURL`

## Phase 2: Core Implementation

- [x] 2.1 Create `internal/modules/auth/oauth.go` - OAuthService struct and methods
- [x] 2.2 Implement `NewOAuthService()` constructor
- [x] 2.3 Implement `GenerateAuthURL(state string)` method
- [x] 2.4 Implement `HandleCallback(ctx, code, state)` method
- [x] 2.5 Add `GoogleUserInfo` struct with all fields (ID, Email, Name, GivenName, FamilyName, Picture)
- [x] 2.6 Add `LoginWithGoogle(ctx, googleUser *GoogleUserInfo)` method to AuthService
- [x] 2.7 Add OAuth DTOs to `internal/modules/auth/dto.go`

## Phase 3: Integration

- [x] 3.1 Add `/auth/google` endpoint to `routes.go` - initiates OAuth flow
- [x] 3.2 Add `/auth/google/callback` endpoint to `routes.go` - handles OAuth callback
- [x] 3.3 Implement state parameter generation and validation (CSRF protection)
- [x] 3.4 Wire OAuthService in route registration

## Phase 4: Testing

- [x] 4.1 Write unit tests for OAuthService.GenerateAuthURL
- [x] 4.2 Write unit tests for OAuthService.HandleCallback (mock Google)
- [x] 4.3 Write unit tests for AuthService.LoginWithGoogle
- [x] 4.4 Test: New user creation from Google profile
- [x] 4.5 Test: Existing user update from Google profile

## Phase 5: Cleanup

- [x] 5.1 Run `go build` to verify compilation
- [x] 5.2 Run `go vet` to check for issues
- [x] 5.3 Run all tests to ensure nothing is broken
- [ ] 5.4 Update documentation (if needed)
