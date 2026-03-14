# Tasks: GitHub Actions CI/CD Workflows

## Phase 1: Infrastructure

- [x] 1.1 Create `.github/workflows/` directory structure
- [x] 1.2 Create `.golangci.yml` configuration file
- [x] 1.3 Verify golangci-lint is installed locally (in Makefile)

## Phase 2: CI Workflow Implementation

- [x] 2.1 Create `.github/workflows/ci.yml` - main CI workflow file
- [x] 2.2 Configure trigger on pull requests
- [x] 2.3 Add checkout step
- [x] 2.4 Add Go setup with caching
- [x] 2.5 Add golangci-lint step
- [x] 2.6 Add go test step with coverage
- [x] 2.7 Add gosec security scan step
- [x] 2.8 Add trivy filesystem scan step
- [x] 2.9 Add go build verification step

## Phase 3: CD Workflow Implementation

- [x] 3.1 Create `.github/workflows/cd.yml` - main CD workflow file
- [x] 3.2 Configure trigger on push to main
- [x] 3.3 Add checkout and Go setup steps
- [x] 3.4 Add test step
- [x] 3.5 Add Docker build step
- [x] 3.6 Add trivy container scan step
- [x] 3.7 Add Docker login to GHCR
- [x] 3.8 Add Docker push step
- [x] 3.9 Add metadata step (tags, labels)

## Phase 4: Testing & Verification

- [x] 4.1 Validate YAML syntax of workflows
- [x] 4.2 Verify all required actions are available
- [ ] 4.3 Test workflow locally with act (optional)
- [ ] 4.4 First manual run trigger (optional)

## Phase 5: Cleanup

- [x] 5.1 Run linter check locally
- [x] 5.2 Verify build works
- [ ] 5.3 Document workflow usage in README (optional)
