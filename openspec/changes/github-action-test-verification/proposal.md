# Proposal: GitHub Actions CI/CD Workflows

## Intent

Implementar pipelines de CI/CD usando GitHub Actions para automatizar:
- Ejecución de tests antes de merge
- Verificaciones de seguridad (linting, scanning de vulnerabilidades)
- Build de Docker image
- Validación en Pull Requests y Push a main

Esto asegura que cada cambio cumple con los estándares de calidad y seguridad antes de llegar a producción.

## Scope

### In Scope
- **Workflow 1: CI (Pull Request)**
  - Linting con golangci-lint
  - Ejecución de tests con coverage
  - Escaneo de seguridad (gosec, trivy)
  - Verificación de build

- **Workflow 2: CD (Push a main)**
  - Build de Docker image
  - Escaneo de vulnerabilidades de contenedor
  - Push a container registry (GitHub Packages)

### Out of Scope
- Despliegue automático a producción
- Environments (dev/staging/prod)
- Integración con servicios externos (AWS, GCP, etc.)
- Análisis de código estático avanzado (sonarcloud)

## Approach

1. **Crear `.github/workflows/ci.yml`**:
   - Se ejecuta en Pull Requests
   - Steps: checkout → setup-go → lint → test → security-scan → build

2. **Crear `.github/workflows/cd.yml`**:
   - Se ejecuta en push a main
   - Steps: checkout → setup-go → test → build-Docker → scan → push

3. **Agregar configuración de golangci-lint** (`.golangci.yml`):
   - Habilitar linters relevantes para Go
   - Configurar reglas del proyecto

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `.github/workflows/ci.yml` | New | Pipeline de CI para PRs |
| `.github/workflows/cd.yml` | New | Pipeline de CD para main |
| `.golangci.yml` | New | Configuración de linters |
| `go.mod` | May need | Actualizar si hay nuevas dependencias |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Go 1.25 no soportado por actions | Low | Usar actions/setup-go@v5 que soporta versiones recientes |
| Tests fallan en CI | Medium | Asegurar que tests sean deterministas |
| Secrets expuestos | Low | Usar GitHub Secrets para credenciales |
| Workflow lento | Medium | Usar caching para dependencias |

## Rollback Plan

1. Eliminar `.github/workflows/ci.yml`
2. Eliminar `.github/workflows/cd.yml`
3. Eliminar `.golangci.yml`
4. Los workflows se dejan de ejecutar automáticamente

## Dependencies

- **golangci-lint**: Instalar en workflow
- **gosec**: Instalar en workflow
- **trivy**: Instalar en workflow
- **docker/build-push-action**: Para build de imagen
- **actions/checkout@v4**: Checkout de código
- **actions/setup-go@v5**: Setup de Go

## Success Criteria

- [ ] PR触发CI workflow con todos los steps pasando
- [ ] Push a main dispara CD workflow exitosamente
- [ ] golangci-lint no reporta errores
- [ ] gosec no reporta vulnerabilidades HIGH/CRITICAL
- [ ] trivy no encuentra vulnerabilidades CRITICAL en dependencies
- [ ] Docker image se builda y push correctamente
- [ ] Coverage se reporta en PR comments
