# Proposal: Dashboard UI with TDD and Swagger Documentation

## Intent

The Bey API currently lacks a web-based user interface for visualizing API data and has no formal API documentation. This change addresses three key needs:
1. **User Experience**: Provide a dashboard interface for non-technical users to view products, orders, and inventory data
2. **Developer Experience**: Add Swagger/OpenAPI documentation for all existing and future API endpoints
3. **Code Quality**: Adopt Test-Driven Development (TDD) methodology to ensure reliable, testable code from the start

## Scope

### In Scope
- Dashboard UI with HTML/CSS/JavaScript frontend served by Go backend
- Swagger/OpenAPI documentation integrated with swaggo
- TDD approach with tests written before implementation
- Dashboard views: Products, Categories, Orders, Inventory overview
- API documentation for all `/api/v1` endpoints

### Out of Scope
- User authentication/authorization for dashboard (future enhancement)
- Real-time updates via WebSocket (future enhancement)
- Mobile-responsive dashboard (basic mobile support only)
- Dashboard backend API (dashboard fetches from existing API)

## Approach

1. **Swagger Integration**:
   - Use `swaggo/swag` to generate OpenAPI 3.0 documentation
   - Add Swagger UI endpoint at `/swagger/index.html`
   - Document all existing endpoints in products, users, orders, inventory modules
   - Use Go annotations in handler files for documentation generation

2. **Dashboard UI**:
   - Create `cmd/api/static/` directory for HTML, CSS, JS files
   - Serve static files via Gin static middleware
   - Dashboard as Single Page Application (SPA) with vanilla JS
   - Fetch data from existing API endpoints (`/api/v1/products`, `/api/v1/orders`, etc.)

3. **TDD Implementation**:
   - Write Go test files (`*_test.go`) before implementing new features
   - Use table-driven tests for handler and repository layers
   - Mock database interactions where appropriate
   - Run tests: `go test -v ./...`

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/api/main.go` | Modified | Add static file serving, Swagger init, route setup |
| `cmd/api/static/` | New | Dashboard HTML, CSS, JS files |
| `cmd/api/docs/` | New | Generated Swagger files (swag init output) |
| `internal/modules/products/handler.go` | Modified | Add Swagger annotations |
| `internal/modules/users/handler.go` | Modified | Add Swagger annotations |
| `internal/modules/orders/handler.go` | Modified | Add Swagger annotations |
| `internal/modules/inventory/handler.go` | Modified | Add Swagger annotations |
| `go.mod` | Modified | Add swaggo dependency |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Static file serving exposes sensitive data | Low | Dashboard only reads, no write operations; separate API for writes |
| TDD slows initial development | Medium | Start with small scope; expand test coverage incrementally |
| Swagger generation conflicts with existing code | Low | Run `swag init` in CI before building |
| Dashboard performance with large datasets | Medium | Implement pagination in API; limit initial load |

## Rollback Plan

1. Remove `cmd/api/static/` directory
2. Remove `cmd/api/docs/` directory  
3. Remove swaggo dependency from `go.mod`
4. Revert `main.go` to previous state (remove static serving and Swagger init)
5. Run `go mod tidy` to clean up dependencies
6. Run tests to verify no regressions

## Dependencies

- **swaggo/swag** (v1.8.x) - OpenAPI documentation generator
- **golang.org/x/net** - Already available via Gin
- No new database migrations required

## Success Criteria

- [ ] Swagger UI accessible at `/swagger/index.html`
- [ ] All `/api/v1/*` endpoints documented with request/response schemas
- [ ] Dashboard loads and displays data from at least Products and Orders endpoints
- [ ] All new code has corresponding test files with >80% coverage on handler logic
- [ ] `go test ./...` passes with no failures
- [ ] `golangci-lint run` passes with no errors
