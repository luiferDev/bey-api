# Implementation Tasks: Migrate All Integer IDs to UUIDv7

## Phase 1: Setup & Foundation
> **Dependencies:** None — start here
> **Estimated:** 2-3 tasks

- [ ] 1. Add `github.com/gofrs/uuid v5.3.0` dependency to `go.mod` and remove `github.com/google/uuid` if present. Run `go mod tidy`.
  - **Files:** `go.mod`, `go.sum`
  - **Verify:** `go build ./...` succeeds

- [ ] 2. Create `internal/shared/uuidutil/uuid.go` helper package with `New()`, `Parse()`, `MustParse()`, `IsZero()` functions wrapping `gofrs/uuid`. All using UUIDv7 (`uuid.NewV7()`).
  - **Files:** `internal/shared/uuidutil/uuid.go` (NEW)
  - **Design reference:** Section 3.2 of design.md

- [ ] 3. Create `internal/shared/uuidutil/uuid_test.go` with table-driven tests for all uuidutil functions including edge cases (empty string, invalid format, nil UUID).
  - **Files:** `internal/shared/uuidutil/uuid_test.go` (NEW)
  - **Verify:** `go test -v ./internal/shared/uuidutil/...`

---

## Phase 2: Database Schema Migration
> **Dependencies:** Phase 1 complete
> **Estimated:** 1 task

- [ ] 4. Create `migrations/001_uuid_migration.sql` with the full DROP+CREATE SQL script from design.md Section 10.2. Tables must be created in FK dependency order (users → categories → products → variants → attributes → images → orders → order_items → inventories → payments → payment_links → refresh_tokens).
  - **Files:** `migrations/001_uuid_migration.sql` (NEW), `migrations/` directory (NEW)
  - **Note:** Do NOT run migration yet — script is for deployment phase

---

## Phase 3: Models (7 files, 14 models)
> **Dependencies:** Phase 1 complete
> **Estimated:** 3 tasks (grouped by module)

- [ ] 5. Update `internal/modules/products/model.go` — Change 5 models (Category, Product, ProductVariant, ProductVariantAttribute, ProductImage):
  - All PKs: `uint` → `uuid.UUID` with `gorm:"type:uuid;primaryKey"`
  - All FKs: `uint`/`*uint` → `uuid.UUID`/`*uuid.UUID` with `gorm:"type:uuid"`
  - **Category:** Remove `Path` field entirely. Change `ParentID` to `*uuid.UUID`. Add `Depth int` field if not present.
  - Add `BeforeCreate` hook to each model: `if m.ID == uuid.Nil { m.ID = uuidutil.New() }`
  - **Files:** `internal/modules/products/model.go`

- [ ] 6. Update `internal/modules/orders/model.go` — Change 2 models (Order, OrderItem):
  - All PKs/FKs: `uint` → `uuid.UUID` with `gorm:"type:uuid"`
  - Add `BeforeCreate` hooks
  - **Files:** `internal/modules/orders/model.go`

- [ ] 7. Update remaining 5 model files:
  - `internal/modules/users/model.go` — User PK → `uuid.UUID`, add `BeforeCreate`
  - `internal/modules/payments/model.go` — Payment, PaymentLink PKs/FKs → `uuid.UUID`, add `BeforeCreate`
  - `internal/modules/cart/model.go` — Cart.UserID → `string`, CartItem.VariantID → `string` (no DB PK, Redis-stored)
  - `internal/modules/auth/model.go` — RefreshToken PK/UserID → `uuid.UUID`, add `BeforeCreate`
  - `internal/modules/inventory/model.go` — Inventory PK/ProductID → `uuid.UUID`, add `BeforeCreate`
  - **Files:** 5 files listed above

---

## Phase 4: Category Hierarchy Redesign
> **Dependencies:** Phase 3 complete (Category model updated)
> **Estimated:** 2 tasks

- [ ] 8. Update `internal/modules/products/repository.go` — Category methods:
  - `Create`: Remove path/level calculation. Set `Depth = parent.Depth + 1` or `0` for root.
  - `Update`: Remove subtree path updates. Add circular reference check via `isDescendant()` recursive CTE. Recalculate depth if parent changed.
  - `FindBreadcrumbs`: Replace path-parsing with recursive CTE (design.md Section 4.4)
  - `FindChildren`: Same logic, just change `parentID uint` → `parentID uuid.UUID`
  - Add `isDescendant(potentialDescendantID, ancestorID uuid.UUID) bool` using recursive CTE
  - Remove `updateSubtreePath` method entirely (no longer needed)
  - **Files:** `internal/modules/products/repository.go`

- [ ] 9. Update `internal/modules/products/repository.go` — All other repository methods:
  - Change every method signature from `uint` → `uuid.UUID` (FindByID, Delete, FindByCategoryID, etc.)
  - Update cache key format: `fmt.Sprintf("%d", id)` → `id.String()` (design.md Section 11.2)
  - All GORM queries remain structurally identical (GORM handles uuid.UUID natively)
  - **Files:** `internal/modules/products/repository.go`

---

## Phase 5: Repositories (5 remaining files)
> **Dependencies:** Phase 3 complete
> **Estimated:** 3 tasks

- [ ] 10. Update `internal/modules/orders/repository.go`:
  - Change all method signatures: `uint` → `uuid.UUID`
  - Update GORM queries (pattern unchanged, types change)
  - Update cache keys if any: `%d` → `%s` with `.String()`
  - **Files:** `internal/modules/orders/repository.go`

- [ ] 11. Update `internal/modules/users/repository.go`:
  - Change all method signatures: `uint` → `uuid.UUID`
  - Update GORM queries
  - **Files:** `internal/modules/users/repository.go`

- [ ] 12. Update remaining 3 repository files:
  - `internal/modules/payments/repository.go` — All signatures `uint` → `uuid.UUID`
  - `internal/modules/inventory/repository.go` — All signatures `uint` → `uuid.UUID`
  - `internal/modules/cart/repository.go` — Interface: `GetCart(userID uuid.UUID)`, `DeleteCart(userID uuid.UUID)`, key format `cart:%s`
  - **Files:** 3 files listed above

---

## Phase 6: DTOs (5+ files, ~35 fields)
> **Dependencies:** Phase 3 complete (models define new types)
> **Estimated:** 2 tasks

- [ ] 13. Update `internal/modules/products/dto.go`:
  - All ID fields in Request DTOs: `uint` → `string` with `binding:"uuid"` tag
  - All ID fields in Response DTOs: `uint` → `string`
  - Types affected: CreateCategoryRequest (ParentID), UpdateCategoryRequest (ParentID), CategoryResponse (ID, ParentID), CreateProductRequest (CategoryID), UpdateProductRequest (CategoryID), ProductResponse (ID, CategoryID), CreateProductVariantRequest (ProductID), ProductVariantResponse (ID, ProductID), CreateProductImageRequest (ProductID, VariantID), ProductImageResponse (ID, ProductID, VariantID)
  - **Files:** `internal/modules/products/dto.go`

- [ ] 14. Update remaining DTO files:
  - `internal/modules/orders/model.go` — OrderResponse (ID, UserID), OrderItemResponse (ID, ProductID, VariantID), CreateOrderItemRequest (ProductID, VariantID): all ID fields → `string`
  - `internal/modules/payments/dto.go` — PaymentResponse (ID, OrderID), PaymentLinkResponse (ID, OrderID), CreatePaymentLinkRequest (OrderID): → `string`
  - `internal/modules/cart/dto.go` — AddToCartRequest (VariantID), CartResponse (UserID), CartItemResponse (VariantID), CheckoutResponse (OrderID), CheckoutItemResponse (ProductID, VariantID): → `string`
  - `internal/modules/admin/dto.go` — UserResponse (ID): → `string`
  - `internal/modules/inventory/model.go` — InventoryResponse (ID, ProductID): → `string`
  - **Files:** 5 files listed above

---

## Phase 7: Handlers (7 files, ~40 methods)
> **Dependencies:** Phase 5 (repos), Phase 6 (DTOs) complete
> **Estimated:** 3 tasks

- [ ] 15. Update `internal/modules/products/handler.go`:
  - All `strconv.ParseUint(c.Param("id"), 10, 32)` → `uuid.Parse(c.Param("id"))`
  - Invalid UUID → 400 Bad Request with message "Invalid ID: must be a valid UUID"
  - Update all mapper functions: `id.String()` for response DTOs
  - Update cache invalidation calls: `uint` → `uuid.UUID`
  - **Files:** `internal/modules/products/handler.go`

- [ ] 16. Update `internal/modules/orders/handler.go`:
  - All `strconv.ParseUint` → `uuid.Parse`
  - All `c.GetUint("user_id")` → `c.GetString("user_id")` + `uuid.Parse()`
  - Update interface method calls with UUID types
  - **Files:** `internal/modules/orders/handler.go`

- [ ] 17. Update remaining 5 handler files:
  - `internal/modules/users/handler.go` — `fmt.Sscanf` → `uuid.Parse`, `c.GetUint` → `c.GetString` + `uuid.Parse`
  - `internal/modules/payments/handler.go` — `parseUint` → `uuid.Parse`, `c.GetUint` → `c.GetString` + `uuid.Parse`
  - `internal/modules/inventory/handler.go` — `strconv.ParseUint` → `uuid.Parse`
  - `internal/modules/cart/handler.go` — `c.GetUint` → `c.GetString` + `uuid.Parse`, update `parseUUIDParam` helper
  - `internal/modules/admin/handler.go` — Update mapper functions: `uint` → `string` for ID fields
  - **Files:** 5 files listed above

---

## Phase 8: Services & Auth (6+ files)
> **Dependencies:** Phase 5 (repos), Phase 7 (handlers) complete
> **Estimated:** 3 tasks

- [ ] 18. Update `internal/modules/auth/token.go`:
  - `TokenClaims.UserID`: `uint` → `string`
  - `GenerateAccessToken(userID uint, ...)` → `GenerateAccessToken(userID uuid.UUID, ...)` — store `userID.String()` in claims
  - `StoreRefreshToken(token string, userID uint)` → `StoreRefreshToken(token string, userID uuid.UUID)`
  - `storeRefreshTokenRedis`: `user_id` value as `userID.String()` (string in JSON)
  - `tempTokenData.UserID`: `uint` → `uuid.UUID`
  - **Files:** `internal/modules/auth/token.go`

- [ ] 19. Update `internal/modules/auth/service.go`:
  - All methods with `userID uint` → `userID uuid.UUID`:
    - `SetupTwoFactor`, `EnableTwoFactor`, `DisableTwoFactor`, `VerifyTwoFactor`, `RevokeAllUserRefreshTokens`
  - Any internal struct fields holding user IDs → `uuid.UUID`
  - **Files:** `internal/modules/auth/service.go`

- [ ] 20. Update remaining service files:
  - `internal/modules/orders/service.go` — All method signatures with `uint` IDs → `uuid.UUID`
  - `internal/modules/cart/service.go` — All method signatures with `uint` IDs → `uuid.UUID`
  - `internal/modules/payments/service.go` — All method signatures with `uint` IDs → `uuid.UUID`
  - `internal/modules/products/service.go` — All method signatures with `uint` IDs → `uuid.UUID`
  - **Files:** 4 files listed above

---

## Phase 9: Middleware & Shared
> **Dependencies:** Phase 8 complete
> **Estimated:** 2 tasks

- [ ] 21. Update `internal/shared/middleware/auth.go`:
  - Change `c.GetUint("user_id")` → `c.GetString("user_id")` wherever used
  - Verify JWT claim parsing works with `UserID string` in TokenClaims
  - Update any RBAC checks that use user ID type
  - **Files:** `internal/shared/middleware/auth.go`

- [ ] 22. Update `cmd/api/main.go`:
  - Import `internal/shared/uuidutil` package
  - Remove any integer ID seed data or fixtures
  - Verify GORM AutoMigrate runs correctly with new UUID models
  - **Files:** `cmd/api/main.go`

---

## Phase 10: Redis Cart & Cache
> **Dependencies:** Phase 5 (cart repo), Phase 8 (cart service) complete
> **Estimated:** 1 task

- [ ] 23. Finalize Redis cart implementation:
  - Verify key format: `cart:{uuid-string}` (not `cart:{int}`)
  - Verify Cart model serialization uses `string` for UserID and VariantID
  - Add cart cleanup code: `SCAN` + `DEL` for `cart:*` pattern on first deploy
  - Verify cache key changes across all modules use `.String()` not `fmt.Sprintf("%d", ...)`
  - **Files:** `internal/modules/cart/repository.go`, `internal/modules/cart/model.go`, `internal/shared/cache/cache_service.go`

---

## Phase 11: Routes & Swagger
> **Dependencies:** Phase 7 (handlers) complete
> **Estimated:** 1 task

- [ ] 24. Update Swagger annotations across all handler files:
  - Change `@Param id path int true "..."` → `@Param id path string true "..."`
  - Update all `@Param` annotations that reference IDs
  - Verify route registrations still work (no route pattern changes needed, just parameter types)
  - Regenerate swagger docs: `swag init -g cmd/api/main.go -o cmd/api/docs --parseDependency --parseInternal`
  - **Files:** All handler files (7 files), `cmd/api/docs/` (regenerated)

---

## Phase 12: Tests — Foundation & Models
> **Dependencies:** Phase 3 (models), Phase 1 (uuidutil) complete
> **Estimated:** 2 tasks

- [ ] 25. Update all model-level test data:
  - Replace all `uint` test IDs with `uuid.UUID` using `uuidutil.New()` or `uuidutil.MustParse("0195c8a1-b2c3-7d4e-8f90-123456789abc")`
  - Update table-driven test structs: `id uint` → `id uuid.UUID`
  - **Files:** `internal/modules/products/model.go` (any inline tests), `internal/modules/orders/model.go`, `internal/modules/users/model.go`, `internal/modules/payments/model.go`, `internal/modules/cart/model.go`, `internal/modules/auth/model.go`, `internal/modules/inventory/model.go`

- [ ] 26. Update `internal/modules/products/repository_test.go`:
  - All test fixtures use UUID test data
  - Mock signatures updated to `uuid.UUID`
  - Test category hierarchy without path field (test recursive CTE behavior)
  - **Files:** `internal/modules/products/repository_test.go`

---

## Phase 13: Tests — Handlers & Services
> **Dependencies:** Phase 7 (handlers), Phase 8 (services), Phase 12 complete
> **Estimated:** 3 tasks

- [ ] 27. Update handler test files (6 files):
  - `internal/modules/products/handler_test.go` — Route params: `"123"` → `"0195c8a1-b2c3-7d4e-8f90-123456789abc"`, add invalid UUID test case expecting 400
  - `internal/modules/orders/handler_test.go` — Same pattern
  - `internal/modules/users/handler_test.go` — Same pattern
  - `internal/modules/payments/handler_test.go` — Same pattern
  - `internal/modules/inventory/handler_test.go` — Same pattern
  - `internal/modules/cart/handler_test.go` — Same pattern
  - **Files:** 6 handler test files

- [ ] 28. Update service test files (5 files):
  - `internal/modules/products/service_test.go` — UUID test data, mock signatures
  - `internal/modules/orders/service_test.go` — UUID test data
  - `internal/modules/cart/service_test.go` — UUID test data, Redis key format
  - `internal/modules/payments/service_test.go` — UUID test data
  - `internal/modules/auth/service_test.go` — UUID test data, TokenClaims.UserID as string
  - **Files:** 5 service test files

- [ ] 29. Update auth-specific test files (5 files):
  - `internal/modules/auth/auth_test.go` — Token generation with UUID user IDs
  - `internal/modules/auth/token_test.go` — Claims with `UserID string`
  - `internal/modules/auth/middleware_test.go` — Mock context with string user_id
  - `internal/modules/auth/twofa_test.go` — `tempTokenData.UserID` as `uuid.UUID`
  - `internal/modules/auth/rbac_integration_test.go` — UUID user IDs in RBAC
  - **Files:** 5 auth test files

---

## Phase 14: Tests — Remaining & Integration
> **Dependencies:** Phase 13 complete
> **Estimated:** 2 tasks

- [ ] 30. Update remaining test files (7 files):
  - `internal/modules/admin/admin_test.go` — UserResponse with string ID
  - `internal/modules/orders/async_order_test.go` — UUID order IDs
  - `internal/modules/products/parallel_fetch_test.go` — UUID product IDs
  - `internal/modules/payments/repository_test.go` — UUID payment IDs
  - `internal/modules/inventory/repository_test.go` — UUID inventory IDs
  - `internal/modules/users/repository_test.go` — UUID user IDs
  - `internal/modules/users/creator_test.go` — UUID user IDs
  - **Files:** 7 test files listed above

- [ ] 31. Update shared middleware and integration tests:
  - `internal/shared/middleware/auth_test.go` — String user_id in context
  - `internal/modules/auth_integration_test.go` — Full flow with UUID IDs
  - `internal/modules/email/verify_test.go` — UUID user IDs if applicable
  - `internal/modules/email/token_test.go` — UUID user IDs if applicable
  - `internal/shared/health_test.go` — Verify no ID-type dependencies
  - **Files:** 5 test files listed above

---

## Phase 15: Build & Verification
> **Dependencies:** ALL previous phases complete
> **Estimated:** 3 tasks

- [ ] 32. Full build verification:
  - Run `go build ./...` — must succeed with zero errors
  - Run `go vet ./...` — must pass
  - Run `golangci-lint run` — must pass (or fix any new lint errors introduced by migration)
  - **Files:** All files

- [ ] 33. Full test suite:
  - Run `go test -v ./...` — ALL tests must pass
  - Run `go test -race ./...` — no race conditions
  - Run `go test -v -cover ./...` — verify coverage hasn't regressed
  - **Files:** All test files

- [ ] 34. Swagger documentation verification:
  - Regenerate: `swag init -g cmd/api/main.go -o cmd/api/docs --parseDependency --parseInternal`
  - Run server: `go run ./cmd/api/`
  - Verify `http://localhost:8080/swagger/index.html` loads
  - Verify all endpoint parameters show `string` type for IDs (not `integer`)
  - **Files:** `cmd/api/docs/`

---

## Phase 16: Pre-Deployment
> **Dependencies:** Phase 15 complete
> **Estimated:** 2 tasks

- [ ] 35. Test migration script on local/staging database:
  - Run `migrations/001_uuid_migration.sql` against a test database
  - Verify all tables created with UUID columns
  - Verify foreign key constraints are correct
  - Verify GORM AutoMigrate runs without errors after migration
  - **Files:** `migrations/001_uuid_migration.sql`

- [ ] 36. Manual integration testing:
  - Create categories (root and nested) — verify hierarchy works without path field
  - Create products with categories — verify FK works with UUIDs
  - Create users, login — verify JWT with string UserID
  - Add items to cart — verify Redis key format `cart:{uuid}`
  - Create orders — verify full flow with UUIDs
  - Test invalid UUID in URL → 400 response
  - Test integer ID in URL → 400 response (UUID parse fails)
  - **Files:** N/A (manual testing)

---

## Dependency Graph

```
Phase 1 (Setup) ─────────────────────────────────────────────────┐
    ├── Phase 2 (Migration script)                                │
    ├── Phase 3 (Models) ─────────────────────────────────────┐   │
    │   ├── Phase 4 (Category hierarchy)                       │   │
    │   ├── Phase 5 (Repositories) ────────────────────────┐   │   │
    │   │   └── Phase 9 (Redis/Cart)                       │   │   │
    │   ├── Phase 6 (DTOs) ──────────────────────────────┐ │   │   │
    │   │   └── Phase 7 (Handlers) ────────────────────┐ │ │   │   │
    │   │       └── Phase 8 (Services/Auth) ────────┐  │ │ │   │   │
    │   │           └── Phase 9 (Middleware/Shared) │  │ │ │   │   │
    │   │               └── Phase 10 (Routes/Swagger)│ │ │ │   │   │
    │   │                                           │ │ │ │   │   │
    │   └── Phase 12 (Tests: Models) ───────────────┘ │ │ │   │   │
    │       └── Phase 13 (Tests: Handlers/Services)    │ │ │   │   │
    │           └── Phase 14 (Tests: Remaining)         │ │ │   │   │
    │               └── Phase 15 (Build/Test) ──────────┘ │ │   │   │
    │                   └── Phase 16 (Pre-Deploy) ────────┘ │   │   │
    └────────────────────────────────────────────────────────┘   │   │
                                                                  │   │
Parallel work possible:                                           │   │
- Phase 3 tasks can be done in parallel (different model files)    │   │
- Phase 5 tasks can be done in parallel (different repo files)     │   │
- Phase 7 tasks can be done in parallel (different handler files)  │   │
- Phase 12-14 test tasks can overlap with Phase 7-8               │   │
                                                                  │   │
Critical path: Phase 1 → Phase 3 → Phase 5 → Phase 7 → Phase 8 → │   │
Phase 9 → Phase 10 → Phase 12 → Phase 13 → Phase 14 → Phase 15 → │   │
Phase 16                                                          │   │
                                                                  │   │
Total tasks: 36                                                   │   │
Estimated commits: 20-25 (some phases can be combined)            │   │
                                                                  └───┘