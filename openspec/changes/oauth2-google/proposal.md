# Proposal: OAuth2 Google Login

## Intent

Implementar autenticación con Google OAuth2 para permitir que usuarios inicien sesión con su cuenta de Google. Esto proporciona:
- Login sin contraseña (más seguro, menos fricción)
- Verificación automática de email
- Experiencia de usuario mejorada

## Scope

### In Scope
- Integración con Google OAuth2 usando `golang.org/x/oauth2`
- Creación automática de usuarios nuevos desde perfil de Google
- Guardar: email, nombre (first + last), avatar (foto de perfil)
- Configuración de URLs de callback en `config.yaml` (dev y prod)
- Endpoints: `/auth/google` (iniciar), `/auth/google/callback` (callback)
- Retorno de JWT tokens (mismo flujo que login existente)

### Out of Scope
- Vinculación de cuenta Google a cuenta existente (futuro)
- Otros providers (Facebook, GitHub, etc.)
- OAuth2 con OpenID Connect (solo OAuth2 básico)

## Approach

1. **Infraestructura**: Agregar `golang.org/x/oauth2` a `go.mod`
2. **Configuración**: Agregar `oauth.google` a `config.yaml` con client_id, client_secret, redirect_urls
3. **Modelo**: Extender User con campos OAuth (provider, provider_id, avatar_url)
4. **Servicio**: Crear `OAuthService` para flujo OAuth2 con Google
5. **Endpoints**: Crear rutas `/auth/google` y `/auth/google/callback`
6. **Callback**: Exchange code por token → obtener user info → crear/find user → retornar JWT

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `go.mod` | Modified | Agregar golang.org/x/oauth2 |
| `config.yaml` | Modified | Agregar oauth.google config |
| `internal/config/config.go` | Modified | Agregar OAuthConfig struct |
| `internal/modules/users/model.go` | Modified | Agregar OAuth campos |
| `internal/modules/auth/service.go` | Modified | Agregar OAuth login methods |
| `internal/modules/auth/routes.go` | Modified | Agregar OAuth endpoints |
| `internal/modules/auth/dto.go` | Modified | Agregar OAuth DTOs |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Token exposure | Low | Exchange OAuth token immediately, no storage |
| CSRF attacks | Med | Validar state parameter en callback |
| Email duplicate | Low | Si existe usuario con mismo email, actualizar datos |
| Scope creep | Med | Mantener solo Google OAuth2 en esta iteración |

## Rollback Plan

1. Remover `golang.org/x/oauth2` de `go.mod`
2. Remover configuración OAuth de `config.yaml`
3. Remover campos OAuth del modelo de usuario (migration necesaria)
4. Remover endpoints OAuth de `routes.go`
5. Remover métodos OAuth de `service.go`

## Dependencies

- `golang.org/x/oauth2` - Biblioteca oficial de Google
- Google Cloud Console - Crear OAuth credentials

## Success Criteria

- [ ] Usuario puede iniciar login con Google desde frontend
- [ ] Callback procesa correctamente y retorna JWT
- [ ] Usuario nuevo se crea automáticamente con datos de Google
- [ ] Usuario existente con mismo email actualiza datos
- [ ] Configuración funciona para dev y prod (diferentes redirect URLs)
- [ ] Tests unitarios cubren OAuthService
