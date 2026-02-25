# Tasks: Dashboard UI with TDD and Swagger Documentation

## Phase 1: Infrastructure (Dependencies & Config)

- [x] 1.1 Add swaggo/swag dependency to `go.mod` — run `go get -u github.com/swaggo/swag/cmd/swag@v1.8.12`
- [x] 1.2 Install swag CLI — `go install github.com/swaggo/swag/cmd/swag@latest`
- [x] 1.3 Update `config.yaml` — add `app.static_path: "./cmd/api/static"` and `app.swagger_enabled: true`
- [x] 1.4 Modify `cmd/api/main.go` — add static file serving with Gin `Static()` middleware and Swagger middleware init
- [x] 1.5 Create directory `cmd/api/static/` — for dashboard files

## Phase 2: Dashboard UI Implementation

- [x] 2.1 Create `cmd/api/static/index.html` — dashboard SPA entry point with navigation and content container
- [x] 2.2 Create `cmd/api/static/css/styles.css` — dashboard styling for cards, tables, navigation
- [x] 2.3 Create `cmd/api/static/js/app.js` — SPA routing with hash-based navigation (#products, #orders, #inventory), API fetch functions, view rendering functions
- [x] 2.4 Create `cmd/api/static/js/api.js` — HTTP client wrapper with error handling for GET /api/v1/products, /api/v1/orders, /api/v1/inventory
- [x] 2.5 Create `cmd/api/static/js/views/products.js` — render products table with name, price, category, status
- [x] 2.6 Create `cmd/api/static/js/views/orders.js` — render orders table with ID, user, total, status, date
- [x] 2.7 Create `cmd/api/static/js/views/inventory.js` — render inventory summary with total items, low stock alerts
- [x] 2.8 Add refresh button functionality to each view in `app.js`

## Phase 3: Swagger Documentation

- [x] 3.1 Add swaggo annotations to `internal/modules/products/handler.go` — @Summary, @Description, @Tags, @Param, @Success, @Router for all endpoints
- [x] 3.2 Add swaggo annotations to `internal/modules/users/handler.go` — same structure as products
- [x] 3.3 Add swaggo annotations to `internal/modules/orders/handler.go` — same structure
- [x] 3.4 Add swaggo annotations to `internal/modules/inventory/handler.go` — same structure
- [x] 3.5 Run `swag init -g cmd/api/main.go -o cmd/api/docs` — generate docs.go and swagger.yaml
- [x] 3.6 Verify `/swagger/index.html` accessible — test with curl or browser

## Phase 4: Testing (TDD Red/Green/Refactor)

### 4.1 Products Handler Tests (TDD)

- [x] 4.1.1 RED: Write failing test in `internal/modules/products/handler_test.go` — TestGetProducts_Success verifies 200 response with product list
- [x] 4.1.2 RED: Write failing test — TestGetProducts_InvalidPagination verifies 400 on invalid offset/limit
- [x] 4.1.3 GREEN: Implement pagination validation in products handler to make tests pass
- [x] 4.1.4 RED: Write failing test — TestGetProductByID_Success verifies 200 with single product
- [x] 4.1.5 GREEN: Implement GetProductByID handler logic if missing
- [x] 4.1.6 REFACTOR: Clean up handler, ensure >80% coverage on product handler

### 4.2 Orders Handler Tests (TDD)

- [x] 4.2.1 RED: Write failing test in `internal/modules/orders/handler_test.go` — TestGetOrders_Success verifies 200 response
- [x] 4.2.2 GREEN: Implement order handler logic to pass test
- [x] 4.2.3 RED: Write failing test — TestCreateOrder_InvalidBody verifies 400 on missing fields
- [x] 4.2.4 GREEN: Implement validation in CreateOrder handler
- [x] 4.2.5 REFACTOR: Clean up order handler tests

### 4.3 Inventory Handler Tests (TDD)

- [x] 4.3.1 RED: Write failing test in `internal/modules/inventory/handler_test.go` — TestGetInventory_Success verifies 200 response
- [x] 4.3.2 GREEN: Implement inventory handler if missing
- [x] 4.3.3 REFACTOR: Clean up inventory handler tests

### 4.4 Users Handler Tests (TDD)

- [x] 4.4.1 RED: Write failing test in `internal/modules/users/handler_test.go` — TestGetUsers_Success verifies 200 response
- [x] 4.4.2 GREEN: Implement user handler if missing
- [x] 4.4.3 REFACTOR: Clean up user handler tests

### 4.5 Static File Serving Tests

- [x] 4.5.1 RED: Write failing test in `internal/shared/middleware/middleware_test.go` — TestStaticFileServing verifies index.html served at root
- [x] 4.5.2 GREEN: Implement static serving logic to pass test
- [x] 4.5.3 RED: Write failing test — TestStaticFileNotFound verifies 404 for non-existent files
- [x] 4.5.4 GREEN: Implement 404 handling for static files

### 4.6 Swagger Integration Tests

- [x] 4.6.1 RED: Write failing test — TestSwaggerEndpoint verifies /swagger/index.html returns 200
- [x] 4.6.2 GREEN: Ensure Swagger middleware wired in main.go

## Phase 5: Integration & Cleanup

- [x] 5.1 Run `go test -v ./...` — verify all tests pass with no failures
- [x] 5.2 Run `golangci-lint run` — fix any lint errors (handler.go fix applied, pre-existing test issues remain)
- [x] 5.3 Run `go test -cover ./...` — verify >80% coverage on handler packages
- [ ] 5.4 Verify dashboard loads in browser at `http://localhost:8080/` — products, orders, inventory views render correctly
- [ ] 5.5 Verify Swagger UI at `http://localhost:8080/swagger/index.html` — all endpoints documented
- [ ] 5.6 Test SPA navigation — clicking nav links updates content without page reload
- [ ] 5.7 Test API error handling — dashboard shows error message when API returns error
- [ ] 5.8 Test data refresh — refresh button re-fetches and updates displayed data

## Implementation Order

1. Phase 1 first — dependencies and config needed before anything else
2. Phase 3 second — Swagger annotations added to handlers (no functional changes)
3. Phase 2 third — static files independent of handler tests
4. Phase 4 fourth — TDD tests written after handlers have annotations
5. Phase 5 last — verification and cleanup

Rationale: Infrastructure must be ready before dashboard can serve files. Swagger annotations colocated with handlers should be done early. Dashboard static files are independent. TDD tests depend on handler structure. Final integration verifies everything works together.
