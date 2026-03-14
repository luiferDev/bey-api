# Design: GitHub Actions CI/CD Workflows

## Technical Approach

Implementar dos workflows de GitHub Actions:
1. **CI Workflow** (`.github/workflows/ci.yml`): Se ejecuta en Pull Requests
2. **CD Workflow** (`.github/workflows/cd.yml`): Se ejecuta en push a main

Ambos workflows siguen los requisitos de la especificación de CI/CD.

## Architecture Decisions

### Decision: Workflow Structure

**Choice**: Dos archivos de workflow separados (ci.yml y cd.yml)
**Alternatives considered**: Un solo workflow con conditional triggers
**Rationale**: Separación clara de responsabilidades, más fácil de mantener y debuggear

### Decision: Go Version

**Choice**: Usar `actions/setup-go@v5` con matrix strategy
**Alternatives considered**: Fixed Go version, only latest
**Rationale**: Soporta Go 1.25, matrix permite testing en múltiples versiones si es necesario

### Decision: Security Scanning

**Choice**: gosec + trivy (dos herramientas separadas)
**Alternatives considered**: Solo trivy, solo gosec
**Rationale**: 
- gosec: scanner específico para código Go
- trivy: mejor para dependencias y contenedor

### Decision: Docker Registry

**Choice**: GitHub Container Registry (ghcr.io)
**Alternatives considered**: Docker Hub, AWS ECR, GCP Container Registry
**Rationale**: Integrado con GitHub, gratis para repos públicos/privados, autenticación automática

### Decision: Caching Strategy

**Choice**: GitHub Actions cache para dependencias de Go
**Alternatives considered**: No caching, third-party cache
**Rationale**: Mejora significativamente el tiempo de ejecución, configurado out-of-the-box con actions/setup-go

## Data Flow

### CI Pipeline Flow
```
PR opened/updated
       │
       ▼
┌──────────────────┐
│   Checkout Code   │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│    Setup Go      │ ──── cache restore
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│    golangci-lint │ ──── Linting check
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│   go test ./...  │ ──── Unit tests + coverage
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│   gosec scan     │ ──── Security scan
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│   trivy fs       │ ──── Dependency scan
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│   go build       │ ──── Verify build
└────────┬─────────┘
         │
         ▼
    ✅ PR Mergeable
```

### CD Pipeline Flow
```
Push to main
       │
       ▼
┌──────────────────┐
│   Checkout Code   │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│    Setup Go      │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│   go test ./...  │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Docker Build    │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Trivy Image Scan│ ──── Container scan
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│  Push to GHCR   │
└────────┬─────────┘
         │
         ▼
   ✅ Image Published
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `.github/workflows/ci.yml` | Create | CI pipeline para PRs |
| `.github/workflows/cd.yml` | Create | CD pipeline para main |
| `.golangci.yml` | Create | Configuración de linters |

## Configuration Details

### golangci-lint Configuration
```yaml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gosec
linters-settings:
  gosec:
    excludes:
      - G104  # Suppress unhandled errors
  govet:
    check-shadowing: true
issues:
  exclude-use-default: false
```

### Workflow Environment Variables
```yaml
env:
  GO_VERSION: '1.25'
  DOCKER_REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Workflow | YAML sintax | Validar con act (local) o primera ejecución |
| Jobs | Each job runs | Verificar logs de primera ejecución |
| Steps | Each step | Verificar output de cada step |

## Migration / Rollout

No se requiere migración. Los workflows se agregan y comienzan a ejecutarse automáticamente en el próximo PR/push.

## Open Questions

- [ ] ¿Qué hacer con vulnerabilidades MEDIUM? (actualmente solo FAIL en HIGH/CRITICAL)
- [ ] ¿Necesitamos más versiones de Go en matrix?
- [ ] ¿Configurar GitHub Packages visibility (public/private)?
