# Design: OAuth2 Google Login

## Technical Approach

Implementar OAuth2 con Google usando `golang.org/x/oauth2`. El flujo será:
1. Frontend redirige a `/auth/google` 
2. Backend genera auth URL con state CSRF y redirige a Google
3. Usuario autoriza en Google
4. Google redirecciona a `/auth/google/callback` con code
5. Backend exchange code por token, obtiene user info
6. Backend crea/actualiza usuario, retorna JWT tokens

## Architecture Decisions

### Decision: OAuth2 Library

**Choice**: `golang.org/x/oauth2` (oficial de Google)
**Alternatives considered**: `github.com/golang/oauth2` (deprecated), `github.com/markbates/goth` (multi-provider)
**Rationale**: Biblioteca oficial, bien mantenida, soporte nativo de Google

### Decision: State Management

**Choice**: Generate random state string, store in cookie with CSRF protection
**Alternatives considered**: JWT state token, URL-encoded state
**Rationale**: Simple, stateless en servidor, protección CSRF via cookie

### Decision: User Creation Strategy

**Choice**: Create user automatically if email doesn't exist, update if exists
**Alternatives considered**: Only link to existing, reject new registrations
**Rationale**: User's requirement - smooth UX, no friction

### Decision: Token Handling

**Choice**: Exchange OAuth token immediately, return JWT, don't store OAuth token
**Alternatives considered**: Store OAuth refresh token for API access
**Rationale**: Security best practice - OAuth tokens are short-lived, we use JWT for our own auth

## Data Flow

```
┌──────────┐     /auth/google      ┌──────────┐
│ Frontend │ ─────────────────────→ │  Backend │
└──────────┘                       └────┬─────┘
                                        │ Generate state
                                        │ Redirect to Google
                                        ↓
┌──────────┐   User Consent   ┌──────────┐
│  Google  │ ←──────────────── │  Backend │
└──────────┘                   └────┬─────┘
                                        │ callback with code
                                        ↓
┌──────────┐   /callback     ┌──────────┐
│  Google  │ ───────────────── │  Backend │
└──────────┘                   └────┬─────┘
                                        │ Exchange code for token
                                        │ Get user info
                                        │ Create/Update user
                                        │ Generate JWT
                                        ↓
                                        │ Return JWT + Set cookies
                                        ↓
                                 ┌──────────┐
                                 │ Frontend │
                                 └──────────┘
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `go.mod` | Modify | Add `golang.org/x/oauth2` |
| `config.yaml` | Modify | Add `oauth.google` section |
| `internal/config/config.go` | Modify | Add `OAuthConfig` struct and getter |
| `internal/modules/users/model.go` | Modify | Add OAuth fields (Provider, ProviderID, AvatarURL) |
| `internal/modules/auth/oauth.go` | Create | OAuthService with Google OAuth logic |
| `internal/modules/auth/service.go` | Modify | Add `LoginWithGoogle()` method |
| `internal/modules/auth/routes.go` | Modify | Add `/auth/google` and `/auth/google/callback` |
| `internal/modules/auth/dto.go` | Modify | Add OAuth DTOs |

## Interfaces / Contracts

### Config Changes

```go
type Config struct {
    // ... existing fields
    OAuth OAuthConfig `yaml:"oauth"`
}

type OAuthConfig struct {
    Google GoogleOAuthConfig `yaml:"google"`
}

type GoogleOAuthConfig struct {
    ClientID     string `yaml:"client_id"`
    ClientSecret string `yaml:"client_secret"`
    RedirectURL  string `yaml:"redirect_url"`
}
```

### User Model Changes

```go
type User struct {
    // ... existing fields
    OAuthProvider  string `gorm:"size:50"` // "google", "facebook", etc.
    OAuthProviderID string `gorm:"size:255"` // User ID from provider
    AvatarURL      string `gorm:"size:500"` // Profile picture URL
}
```

### New OAuth Service

```go
type OAuthService struct {
    config        *config.Config
    oauth2Config  *oauth2.Config
}

func NewOAuthService(cfg *config.Config) *OAuthService

// GenerateAuthURL returns Google OAuth URL and state
func (s *OAuthService) GenerateAuthURL(state string) (string, error)

// HandleCallback exchanges code for token, returns user info
func (s *OAuthService) HandleCallback(ctx context.Context, code, state string) (*GoogleUserInfo, error)

type GoogleUserInfo struct {
    ID            string
    Email         string
    Name          string
    GivenName     string
    FamilyName    string
    Picture       string
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | OAuthService methods | Mock oauth2 config, test auth URL generation, callback handling |
| Unit | Config loading | Verify OAuth config parses correctly |
| Integration | Full OAuth flow | Manual test with real Google credentials |

## Migration / Rollout

1. Add `golang.org/x/oauth2` dependency
2. Add config to `config.yaml` (with placeholder values)
3. Add fields to User model (nullable, no migration needed for existing users)
4. Implement OAuthService
5. Add endpoints
6. Test with Google Cloud Console credentials

## Open Questions

- [ ] Need Google Cloud Console OAuth credentials (client_id, client_secret)
- [ ] Decide redirect URL for production (e.g., `https://api.bey.com/auth/google/callback`)
