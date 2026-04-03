# Proposal: Migrate All Integer IDs to UUIDv7

## Intent

Replace all auto-incrementing integer primary keys (`uint`) across the entire bey_api codebase with UUIDv7 identifiers. This migration addresses security concerns (enumeration attacks, predictable IDs), improves data distribution in indexes, enables offline ID generation, and prepares the system for distributed/multi-tenant architectures where centralized ID generation becomes a bottleneck.

## Scope

### In Scope
- All 14 model primary keys across 7 model files (products, orders, users, payments, cart, inventory, auth, admin)
- All 50+ repository method signatures accepting/returning IDs
- All 40+ handler methods using `strconv.ParseUint` for ID parsing
- All 35+ DTO fields using `uint` for ID references
- All 38 route patterns accepting integer ID parameters
- Category tree materialized path migration from `/1/5/12/` to UUID-based path
- GORM model definitions updated to use UUIDv7 as primary key
- JWT claims updated to use UUID string for `user_id`
- Redis cart key format updated from `cart:%d` to `cart:%s`
- Database migration script for existing data transformation
- All affected tests updated

### Out of Scope
- API versioning (v1 → v2) — this is a breaking change within the same version
- External webhook URL changes (Wompi webhooks reference internal IDs)
- Third-party integrations that may cache integer IDs
- Admin dashboard UI changes (handled by frontend team)
- Audit log historical data migration (existing audit logs keep integer references)

## Decisions

### D1: UUID Library — `github.com/gofrs/uuid`

**Decision**: Use `github.com/gofrs/uuid` (v5+) for UUID generation and parsing.

**Rationale**: The standard `google/uuid` library does not support UUIDv7. The `gofrs/uuid` library provides first-class UUIDv7 support with `uuid.NewV7()`, is well-maintained by the UUID working group, and integrates cleanly with GORM via the `uuid.NullUUID` type for optional fields.

**Alternative considered**: `github.com/lithammer/shortuuid` — rejected due to lack of UUIDv7 support and non-standard format.

### D2: Go ID Type — `string` in DTOs, `uuid.UUID` in models

**Decision**: 
- **GORM models**: Use `uuid.UUID` type for primary keys and foreign keys
- **DTOs (request/response)**: Use `string` type for all ID fields
- **Repository layer**: Accept `uuid.UUID` in method signatures
- **Handler layer**: Parse incoming string IDs to `uuid.UUID` using `uuid.FromString()`

**Rationale**: Using `string` in DTOs keeps the JSON API clean and avoids custom JSON marshaling. Using `uuid.UUID` in models provides type safety and leverages the library's validation. This avoids the overhead of custom GORM value transformers.

```go
// Model
type Product struct {
    ID        uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    CategoryID uuid.UUID   `gorm:"type:uuid;not null;index" json:"category_id"`
    Name      string       `gorm:"size:255;not null" json:"name"`
    // ...
}

// DTO
type ProductResponse struct {
    ID         string  `json:"id"`
    CategoryID string  `json:"category_id"`
    Name       string  `json:"name"`
    // ...
}

// Mapper
func ToProductResponse(p Product) ProductResponse {
    return ProductResponse{
        ID:         p.ID.String(),
        CategoryID: p.CategoryID.String(),
        Name:       p.Name,
    }
}
```

### D3: GORM AutoMigrate Limitation — Manual SQL Migration

**Decision**: GORM `AutoMigrate` cannot change an existing column type from `integer` to `uuid`. A manual SQL migration script must be executed BEFORE deploying the new code.

**Migration strategy**:
```sql
-- Step 1: Add new UUID columns (nullable)
ALTER TABLE products ADD COLUMN uuid_id UUID DEFAULT gen_random_uuid();
ALTER TABLE categories ADD COLUMN uuid_id UUID DEFAULT gen_random_uuid();
-- ... repeat for all 14 tables

-- Step 2: Update foreign key references
-- For each table with FKs, populate uuid_id by joining to the new UUID columns

-- Step 3: Create mapping table for data transformation
CREATE TABLE id_mapping (
    table_name TEXT NOT NULL,
    old_id BIGINT NOT NULL,
    new_id UUID NOT NULL,
    PRIMARY KEY (table_name, old_id)
);

-- Step 4: Populate mapping table with existing data
INSERT INTO id_mapping (table_name, old_id, new_id)
SELECT 'products', id, uuid_id FROM products;

-- Step 5: Drop old PKs, rename columns, add constraints
-- (Detailed per-table migration in migration script)
```

**Deployment**: Migration runs as a pre-deployment step. The application starts with a feature flag `UUID_MIGRATION_COMPLETE` that controls which ID type is used.

### D4: Category Tree Path — UUID-based materialized path

**Decision**: Replace the integer-based materialized path (`/1/5/12/`) with a UUID-based path using the first 8 characters of the UUID for readability.

**Rationale**: Full UUID paths like `/a1b2c3d4-e5f6-7890-abcd-ef1234567890/...` are excessively long and hurt index performance. Using the first 8 hex characters provides sufficient uniqueness for path segments while keeping the column size manageable.

```go
type Category struct {
    ID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
    ParentID *uuid.UUID `gorm:"type:uuid;index" json:"parent_id,omitempty"`
    Path     string    `gorm:"size:500;not null" json:"path"` // /a1b2c3d4/e5f67890/...
    Depth    int       `gorm:"not null;default:0" json:"depth"`
}

// Path generation
func (c *Category) BuildPath(parent *Category) string {
    shortID := c.ID.String()[:8]
    if parent == nil {
        return "/" + shortID + "/"
    }
    return parent.Path + shortID + "/"
}
```

**Alternative considered**: Keep integer `path` column alongside UUID PK — rejected due to data duplication and synchronization complexity.

### D5: JWT Invalidation — Hard break with migration window

**Decision**: All existing JWT tokens become invalid upon deployment. Implement a 24-hour migration window where both integer and UUID user lookups are supported during the transition.

**Implementation**:
```go
type JWTClaims struct {
    UserID string `json:"user_id"` // UUID string instead of uint
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

// During migration window, support both formats
func ExtractUserID(claims JWTClaims) (uuid.UUID, error) {
    // Try UUID first
    if uid, err := uuid.FromString(claims.UserID); err == nil {
        return uid, nil
    }
    // Fallback: legacy integer ID → lookup mapping
    return legacyIDToUUID(claims.UserID)
}
```

**User impact**: All users must re-authenticate within 24 hours. Refresh tokens stored in Redis are cleared on deployment. Communicate this as a "security upgrade" to users.

### D6: Redis Cart Migration — Key format change with dual-read

**Decision**: Change Redis cart keys from `cart:%d` to `cart:%s` (UUID string). During migration, check both key formats.

```go
func (r *CartRepository) GetCart(userID uuid.UUID) (*Cart, error) {
    // New format
    key := fmt.Sprintf("cart:%s", userID.String())
    data, err := r.redis.Get(ctx, key).Bytes()
    if err == nil {
        return deserializeCart(data)
    }
    
    // Fallback: check legacy integer key (during migration window)
    legacyKey := fmt.Sprintf("cart:%d", legacyUserID)
    data, err = r.redis.Get(ctx, legacyKey).Bytes()
    if err == nil {
        // Migrate to new format
        r.redis.Set(ctx, key, data, ttl)
        r.redis.Del(ctx, legacyKey)
        return deserializeCart(data)
    }
    
    return nil, ErrCartNotFound
}
```

### D7: API Versioning — No version bump (breaking change in-place)

**Decision**: Do NOT create a v2 API. This is a breaking change deployed within the same version with a migration window.

**Rationale**: Maintaining two API versions doubles the testing surface and creates long-term maintenance burden. The migration window (24-48 hours) is sufficient for frontend clients to update. External API consumers receive advance notice.

## Approach

### Phase 1: Foundation (Days 1-2)
1. Add `github.com/gofrs/uuid` dependency
2. Create UUID migration helper package (`internal/shared/uuidutil/`)
3. Write database migration script for all 14 tables
4. Add feature flag `UUID_MIGRATION_COMPLETE` to config
5. Create ID mapping table and populate with existing data

### Phase 2: Model Layer (Days 3-4)
1. Update all 7 model files to use `uuid.UUID` for PKs and FKs
2. Add `To{Model}Response()` mapper functions for each model
3. Update GORM tags with `type:uuid`
4. Run migration script against staging database
5. Verify model tests pass

### Phase 3: Repository Layer (Days 5-6)
1. Update all 50+ repository method signatures from `uint` to `uuid.UUID`
2. Update all GORM queries to use UUID parameters
3. Add UUID validation in repository entry points
4. Update repository tests with UUID test data

### Phase 4: Service Layer (Days 7-8)
1. Update service method signatures
2. Update business logic that depends on integer ID behavior
3. Update category tree path logic
4. Update service tests

### Phase 5: Handler & DTO Layer (Days 9-10)
1. Update all 35+ DTO fields from `uint` to `string`
2. Replace `strconv.ParseUint` with `uuid.FromString()` in all 40+ handlers
3. Add UUID validation middleware
4. Update handler tests

### Phase 6: Infrastructure (Days 11-12)
1. Update JWT claims and token generation
2. Update Redis cart key format with dual-read support
3. Update all 38 route patterns (no structural change needed, just handler logic)
4. Update Swagger documentation

### Phase 7: Testing & Validation (Days 13-14)
1. Run full test suite with UUID data
2. Integration tests against migrated staging database
3. Load testing to verify UUID index performance
4. Security review of new endpoints

### Phase 8: Deployment (Day 15)
1. Execute database migration script
2. Deploy application with feature flag enabled
3. Clear all existing JWT tokens (force re-auth)
4. Monitor error rates for 24 hours
5. Remove migration fallback code after 48 hours

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/modules/products/model.go` | Modified | Product, Category, ProductVariant, ProductImage, ProductVariantAttribute PKs |
| `internal/modules/orders/model.go` | Modified | Order, OrderItem PKs and FKs |
| `internal/modules/users/model.go` | Modified | User, Address PKs |
| `internal/modules/payments/model.go` | Modified | Payment, WebhookLog PKs and FKs |
| `internal/modules/cart/repository.go` | Modified | Redis key format |
| `internal/modules/inventory/model.go` | Modified | StockMovement PK and FKs |
| `internal/modules/auth/middleware.go` | Modified | JWT claims parsing |
| `internal/modules/admin/handler.go` | Modified | Admin endpoint ID parsing |
| `internal/shared/uuidutil/` | New | UUID helper utilities |
| `cmd/api/main.go` | Modified | Feature flag, migration check |
| All `*_test.go` files | Modified | Test data updated to UUIDs |
| `go.mod` | Modified | Add `github.com/gofrs/uuid` dependency |

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| **Data loss during migration** | Medium | Critical | Full database backup before migration. Test migration script on staging copy. Idempotent migration with rollback script. |
| **JWT invalidation causes user churn** | High | High | 24-hour migration window with dual ID support. Communicate as security upgrade. Auto-refresh on next login. |
| **Category path performance degradation** | Medium | Medium | Benchmark UUID path queries vs integer paths. Add composite index on `(path, depth)`. Monitor query times post-deployment. |
| **Redis cart data loss** | Low | Medium | Dual-read during migration window. Existing carts automatically migrated on first access. |
| **Third-party webhook failures** | Medium | High | Wompi webhooks reference order IDs — update webhook handler to support both formats during migration. Notify Wompi of ID format change if they validate IDs. |

## Rollback Plan

1. **Immediate rollback** (within 1 hour): Revert to previous deployment binary. Database remains migrated — application supports both ID formats via feature flag.
2. **Database rollback** (within 24 hours): Run reverse migration script that restores integer PKs from `id_mapping` table. This script is tested alongside the forward migration.
3. **Full rollback** (after 24 hours): Restore from pre-migration database backup. Deploy previous application version.

```bash
# Rollback commands
git revert <migration-commit-hash>
go build -o main ./cmd/api/
./main --rollback-migration  # Runs reverse migration
```

## Dependencies

- `github.com/gofrs/uuid` v5+ (UUIDv7 support)
- Database backup before migration execution
- Staging environment with production data copy for migration testing
- Frontend team coordination for JWT re-auth flow
- Communication plan for external API consumers

## Success Criteria

- [ ] All 14 models use `uuid.UUID` as primary key type
- [ ] All 73 affected files compile without errors
- [ ] Full test suite passes with UUID test data (`go test -v ./...`)
- [ ] Database migration script runs successfully on staging (zero data loss)
- [ ] API endpoints return UUID strings in all ID fields
- [ ] Category tree operations (create, move, query) work with UUID paths
- [ ] JWT authentication works with UUID user IDs
- [ ] Redis cart operations work with UUID keys
- [ ] No performance regression on ID-based queries (< 5% increase in p95 latency)
- [ ] Swagger documentation reflects UUID ID format
- [ ] Zero data loss in production migration
- [ ] All existing users can re-authenticate within 24-hour window
