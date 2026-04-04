# Technical Design: Migrate All Integer IDs to UUIDv7

## 1. Architecture Overview

### UUIDv7 Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│  REQUEST (string ID)                                            │
│  POST /products → { "category_id": "0195c8a1-..." }             │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│  HANDLER LAYER                                                   │
│  uuid.Parse("0195c8a1-...") → uuid.UUID                         │
│  Invalid UUID → 400 Bad Request                                 │
│  Valid UUID → pass to service                                   │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│  SERVICE LAYER                                                   │
│  Business logic with uuid.UUID types                            │
│  No change in logic, only type signature                        │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│  REPOSITORY LAYER                                                │
│  GORM queries with uuid.UUID                                    │
│  New records: uuid.NewV7() before insert                        │
│  db.Where("id = ?", uuidUUID)                                   │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│  DATABASE (PostgreSQL)                                           │
│  Column type: UUID                                              │
│  No default gen_random_uuid() — Go generates via uuid.NewV7()   │
│  Indexes: B-tree on UUID columns (time-ordered = sequential)    │
└─────────────────────────────────────────────────────────────────┘
```

### Layer-by-Layer Change Summary

| Layer | Current | After | Key Change |
|-------|---------|-------|------------|
| Models | `uint` PKs | `uuid.UUID` PKs | GORM tags add `type:uuid` |
| DTOs | `uint` fields | `string` fields | JSON stays human-readable |
| Repos | `FindByID(id uint)` | `FindByID(id uuid.UUID)` | GORM queries unchanged pattern |
| Services | Pass `uint` | Pass `uuid.UUID` | Signature-only change |
| Handlers | `strconv.ParseUint` | `uuid.Parse` | 400 on invalid UUID format |
| Routes | `/:id` (same pattern) | `/:id` (same pattern) | No route structure change |
| JWT | `UserID uint` | `UserID string` | Claims store UUID string |
| Redis | `cart:%d` | `cart:%s` | Key format changes |
| Cache | `cache:product:%d` | `cache:product:%s` | Key format changes |

---

## 2. Model Design — BEFORE and AFTER

### 2.1 Products Module (5 models)

```go
// === Category ===
// BEFORE
type Category struct {
    ID            uint           `gorm:"primaryKey" json:"id"`
    ParentID      *uint          `gorm:"index" json:"parent_id"`
    Path          string         `gorm:"size:500;index" json:"path"`
    Level         int            `gorm:"default:0;index" json:"level"`
    // ...
}

// AFTER — path field ELIMINATED, recursive CTEs replace materialized path
type Category struct {
    ID            uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
    ParentID      *uuid.UUID     `gorm:"type:uuid;index" json:"parent_id,omitempty"`
    // Path field removed — use recursive CTEs for hierarchy queries
    Depth         int            `gorm:"not null;default:0" json:"depth"`
    // ...
}
```

```go
// === Product ===
// BEFORE
type Product struct {
    ID          uint      `gorm:"primaryKey" json:"id"`
    CategoryID  uint      `json:"category_id"`
    // ...
}

// AFTER
type Product struct {
    ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
    CategoryID  uuid.UUID `gorm:"type:uuid;index" json:"category_id"`
    // ...
}
```

```go
// === ProductVariant ===
// BEFORE
type ProductVariant struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    ProductID uint      `json:"product_id"`
    // ...
}

// AFTER
type ProductVariant struct {
    ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
    ProductID uuid.UUID `gorm:"type:uuid;index" json:"product_id"`
    // ...
}
```

```go
// === ProductVariantAttribute ===
// BEFORE
type ProductVariantAttribute struct {
    ID        uint   `gorm:"primaryKey" json:"id"`
    VariantID uint   `gorm:"uniqueIndex" json:"variant_id"`
    // ...
}

// AFTER
type ProductVariantAttribute struct {
    ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
    VariantID uuid.UUID `gorm:"type:uuid;uniqueIndex" json:"variant_id"`
    // ...
}
```

```go
// === ProductImage ===
// BEFORE
type ProductImage struct {
    ID        uint   `gorm:"primaryKey" json:"id"`
    ProductID uint   `json:"product_id"`
    VariantID *uint  `json:"variant_id"`
    // ...
}

// AFTER
type ProductImage struct {
    ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
    ProductID uuid.UUID  `gorm:"type:uuid;index" json:"product_id"`
    VariantID *uuid.UUID `gorm:"type:uuid;index" json:"variant_id,omitempty"`
    // ...
}
```

### 2.2 Orders Module (2 models)

```go
// === Order ===
// BEFORE
type Order struct {
    ID        uint `gorm:"primarykey" json:"id"`
    UserID    uint `gorm:"index" json:"user_id"`
    // ...
}

// AFTER
type Order struct {
    ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
    UserID    uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
    // ...
}
```

```go
// === OrderItem ===
// BEFORE
type OrderItem struct {
    ID        uint   `gorm:"primarykey" json:"id"`
    OrderID   uint   `gorm:"index" json:"order_id"`
    ProductID uint   `gorm:"index" json:"product_id"`
    VariantID *uint  `gorm:"index" json:"variant_id"`
    // ...
}

// AFTER
type OrderItem struct {
    ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
    OrderID   uuid.UUID  `gorm:"type:uuid;index" json:"order_id"`
    ProductID uuid.UUID  `gorm:"type:uuid;index" json:"product_id"`
    VariantID *uuid.UUID `gorm:"type:uuid;index" json:"variant_id,omitempty"`
    // ...
}
```

### 2.3 Users Module (1 model)

```go
// === User ===
// BEFORE
type User struct {
    ID       uint `gorm:"primarykey" json:"id"`
    // ...
}

// AFTER
type User struct {
    ID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
    // ...
}
```

### 2.4 Payments Module (2 models)

```go
// === Payment ===
// BEFORE
type Payment struct {
    ID      uint `gorm:"primaryKey" json:"id"`
    OrderID uint `gorm:"index" json:"order_id"`
    // ...
}

// AFTER
type Payment struct {
    ID      uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
    OrderID uuid.UUID `gorm:"type:uuid;index" json:"order_id"`
    // ...
}
```

```go
// === PaymentLink ===
// BEFORE
type PaymentLink struct {
    ID      uint `gorm:"primaryKey" json:"id"`
    OrderID uint `gorm:"index" json:"order_id"`
    // ...
}

// AFTER
type PaymentLink struct {
    ID      uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
    OrderID uuid.UUID `gorm:"type:uuid;index" json:"order_id"`
    // ...
}
```

### 2.5 Auth Module (1 model)

```go
// === RefreshToken ===
// BEFORE
type RefreshToken struct {
    ID     uint `gorm:"primaryKey"`
    UserID uint `gorm:"not null;index"`
    // ...
}

// AFTER
type RefreshToken struct {
    ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
    UserID uuid.UUID `gorm:"type:uuid;not null;index"`
    // ...
}
```

### 2.6 Inventory Module (1 model)

```go
// === Inventory ===
// BEFORE
type Inventory struct {
    ID        uint `gorm:"primarykey" json:"id"`
    ProductID uint `gorm:"uniqueIndex;index" json:"product_id"`
    // ...
}

// AFTER
type Inventory struct {
    ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
    ProductID uuid.UUID `gorm:"type:uuid;uniqueIndex;index" json:"product_id"`
    // ...
}
```

### 2.7 Cart Module (no PK — Redis-stored)

```go
// === Cart (Redis, no DB PK) ===
// BEFORE
type Cart struct {
    UserID    uint       `json:"user_id"`
    Items     []CartItem `json:"items"`
    // ...
}
type CartItem struct {
    VariantID uint `json:"variant_id"`
    Quantity  int  `json:"quantity"`
}

// AFTER
type Cart struct {
    UserID    string     `json:"user_id"` // UUID string for Redis key
    Items     []CartItem `json:"items"`
    // ...
}
type CartItem struct {
    VariantID string `json:"variant_id"` // UUID string
    Quantity  int    `json:"quantity"`
}
```

---

## 3. UUID Generation Strategy

### 3.1 Library

```go
// go.mod — replace google/uuid with gofrs/uuid
// github.com/google/uuid v1.6.0  → REMOVE
// github.com/gofrs/uuid v5.3.0   → ADD
```

**Why `gofrs/uuid` over `google/uuid`**: `google/uuid` does NOT support UUIDv7. `gofrs/uuid` v5+ has first-class `uuid.NewV7()` support per RFC 9562.

### 3.2 Generation Location

UUIDs are generated **in the repository layer, before insert**. This keeps generation close to persistence and avoids leaking UUID logic into handlers or services.

```go
// internal/shared/uuidutil/uuid.go
package uuidutil

import "github.com/gofrs/uuid"

// New generates a UUIDv7 (time-ordered, index-friendly)
func New() uuid.UUID {
    u, err := uuid.NewV7()
    if err != nil {
        panic("uuid: failed to generate V7: " + err.Error())
    }
    return u
}

// Parse validates and parses a UUID string
func Parse(s string) (uuid.UUID, error) {
    return uuid.FromString(s)
}

// MustParse panics on invalid UUID (use in tests only)
func MustParse(s string) uuid.UUID {
    u, err := uuid.FromString(s)
    if err != nil {
        panic("uuid: invalid UUID string: " + s)
    }
    return u
}

// IsZero checks if UUID is nil/zero
func IsZero(u uuid.UUID) bool {
    return u == uuid.Nil
}
```

### 3.3 Why UUIDv7

- **Time-ordered**: Monotonically increasing within the same millisecond → B-tree index friendly, no page splits
- **Sortable**: Lexicographic sort ≈ chronological sort
- **Standard**: RFC 9562 compliant
- **No coordination needed**: Generated client-side, no DB round-trip

---

## 4. Repository Layer Design

### 4.1 Method Signature Changes

Every repository method that accepts or returns an ID changes from `uint` to `uuid.UUID`:

```go
// BEFORE
func (r *ProductRepository) FindByID(id uint) (*Product, error)
func (r *ProductRepository) Delete(id uint) error
func (r *ProductRepository) FindByCategoryID(categoryID uint, offset, limit int) ([]Product, error)

// AFTER
func (r *ProductRepository) FindByID(id uuid.UUID) (*Product, error)
func (r *ProductRepository) Delete(id uuid.UUID) error
func (r *ProductRepository) FindByCategoryID(categoryID uuid.UUID, offset, limit int) ([]Product, error)
```

### 4.2 GORM UUID Primary Key Handling

GORM natively supports `uuid.UUID` as a primary key. The key behavior:

```go
// GORM auto-generates UUID if model has BeforeCreate hook
func (p *Product) BeforeCreate(tx *gorm.DB) error {
    if p.ID == uuid.Nil {
        p.ID = uuidutil.New()
    }
    return nil
}
```

**Every model gets a `BeforeCreate` hook** to auto-generate UUIDv7. This is cleaner than calling `uuidutil.New()` manually in every Create method.

### 4.3 Query Patterns — No Change

GORM queries remain structurally identical:

```go
// BEFORE
r.db.First(&product, id)           // WHERE id = 123
r.db.Where("category_id = ?", catID) // WHERE category_id = 5

// AFTER — same GORM API, different type
r.db.First(&product, id)           // WHERE id = '0195c8a1-...'
r.db.Where("category_id = ?", catID) // WHERE category_id = '0195c8a1-...'
```

### 4.4 Category Repository — Recursive CTEs Replace Path

The current `path`-based approach (`/1/5/12/`) is eliminated. All hierarchy queries use recursive CTEs:

```go
// FindBreadcrumbs — recursive CTE to find ancestors
func (r *CategoryRepository) FindBreadcrumbs(categoryID uuid.UUID) ([]Category, error) {
    query := `
        WITH RECURSIVE ancestors AS (
            SELECT id, name, slug, parent_id, depth, 1 as ord
            FROM categories
            WHERE id = $1 AND deleted_at IS NULL
            UNION ALL
            SELECT c.id, c.name, c.slug, c.parent_id, c.depth, a.ord + 1
            FROM categories c
            INNER JOIN ancestors a ON c.id = a.parent_id
            WHERE c.deleted_at IS NULL
        )
        SELECT id, name, slug, parent_id, depth FROM ancestors
        ORDER BY ord DESC
    `
    var categories []Category
    err := r.db.Raw(query, categoryID).Scan(&categories).Error
    return categories, err
}
```

```go
// FindChildren — direct children only (same as current, just UUID type)
func (r *CategoryRepository) FindChildren(parentID uuid.UUID) ([]Category, error) {
    var children []Category
    err := r.db.Where("parent_id = ? AND deleted_at IS NULL", parentID).
        Order("sort_order, name").Find(&children).Error
    return children, err
}
```

```go
// isDescendant — check if a category is a descendant of another
func (r *CategoryRepository) isDescendant(potentialDescendantID, ancestorID uuid.UUID) bool {
    query := `
        WITH RECURSIVE descendants AS (
            SELECT id, parent_id FROM categories WHERE id = $1 AND deleted_at IS NULL
            UNION ALL
            SELECT c.id, c.parent_id
            FROM categories c
            INNER JOIN descendants d ON c.parent_id = d.id
            WHERE c.deleted_at IS NULL
        )
        SELECT COUNT(*) FROM descendants WHERE id = $2
    `
    var count int64
    r.db.Raw(query, ancestorID, potentialDescendantID).Scan(&count)
    return count > 0
}
```

```go
// Create — no path/level calculation needed
func (r *CategoryRepository) Create(category *Category) error {
    if category.ParentID != nil && !uuidutil.IsZero(*category.ParentID) {
        var parent Category
        if err := r.db.First(&parent, *category.ParentID).Error; err != nil {
            return fmt.Errorf("parent category not found: %w", err)
        }
        category.Depth = parent.Depth + 1
    } else {
        category.ParentID = nil
        category.Depth = 0
    }
    return r.db.Create(category).Error
}
```

```go
// Update — no subtree path updates needed
func (r *CategoryRepository) Update(category *Category) error {
    var existing Category
    if err := r.db.First(&existing, category.ID).Error; err != nil {
        return err
    }

    oldParentID := existing.ParentID
    newParentID := category.ParentID

    // Circular reference check via recursive CTE
    if newParentID != nil && !uuidutil.IsZero(*newParentID) {
        if r.isDescendant(*newParentID, category.ID) {
            return errors.New("circular reference detected")
        }
    }

    // Update depth if parent changed
    if (oldParentID == nil && newParentID != nil) ||
        (oldParentID != nil && newParentID == nil) ||
        (oldParentID != nil && newParentID != nil && *oldParentID != *newParentID) {
        
        var parent Category
        if newParentID != nil {
            if err := r.db.First(&parent, *newParentID).Error; err != nil {
                return err
            }
            category.Depth = parent.Depth + 1
        } else {
            category.Depth = 0
        }
    }

    return r.db.Save(category).Error
}
```

### 4.5 Cart Repository Interface

```go
// BEFORE
type CartRepository interface {
    GetCart(userID uint) (*Cart, error)
    SaveCart(cart *Cart) error
    DeleteCart(userID uint) error
}

// AFTER
type CartRepository interface {
    GetCart(userID uuid.UUID) (*Cart, error)
    SaveCart(cart *Cart) error
    DeleteCart(userID uuid.UUID) error
}
```

---

## 5. Handler Layer Design

### 5.1 ID Parsing — uuid.Parse instead of strconv.ParseUint

```go
// BEFORE
func (h *ProductHandler) GetProduct(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 32)
    if err != nil {
        h.response.ValidationError(c, "Invalid product ID")
        return
    }
    product, err := h.productRepo.FindByID(uint(id))
    // ...
}

// AFTER
func (h *ProductHandler) GetProduct(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        h.response.ValidationError(c, "Invalid product ID: must be a valid UUID")
        return
    }
    product, err := h.productRepo.FindByID(id)
    // ...
}
```

### 5.2 Error Handling — 400 vs 404

| Scenario | Response |
|----------|----------|
| Invalid UUID format (e.g., "abc", "123") | **400** Bad Request — "Invalid ID: must be a valid UUID" |
| Valid UUID but record not found | **404** Not Found — "Product not found" |
| Valid UUID, record exists | **200** OK |

### 5.3 JWT Claims Extraction

```go
// BEFORE
userID := c.GetUint("user_id")

// AFTER
userIDStr := c.GetString("user_id")
userID, err := uuid.Parse(userIDStr)
if err != nil {
    // This should never happen if middleware is correct
    c.AbortWithStatusJSON(500, gin.H{"error": "invalid user ID in token"})
    return
}
```

### 5.4 User Handler — fmt.Sscanf → uuid.Parse

```go
// BEFORE (users/handler.go)
var targetID uint
if _, err := fmt.Sscanf(idParam, "%d", &targetID); err != nil {
    h.resp.Error(c, 400, "invalid user id")
    return
}

// AFTER
targetID, err := uuid.Parse(idParam)
if err != nil {
    h.resp.Error(c, 400, "invalid user id: must be a valid UUID")
    return
}
```

### 5.5 Order Handler — c.GetUint → c.GetString + uuid.Parse

```go
// BEFORE
userID := c.GetUint("user_id")

// AFTER
userIDStr := c.GetString("user_id")
userID, err := uuid.Parse(userIDStr)
if err != nil {
    h.resp.InternalError(c, "invalid user ID in session")
    return
}
```

---

## 6. DTO Design

### 6.1 Request DTOs — String IDs with UUID Validation

```go
// BEFORE
type CreateProductRequest struct {
    CategoryID  uint    `json:"category_id" binding:"required"`
    Name        string  `json:"name" binding:"required"`
    BasePrice   float64 `json:"base_price" binding:"required,gt=0"`
}

// AFTER
type CreateProductRequest struct {
    CategoryID  string  `json:"category_id" binding:"required,uuid"`
    Name        string  `json:"name" binding:"required,max=255"`
    BasePrice   float64 `json:"base_price" binding:"required,gt=0"`
}
```

**UUID validation**: The `binding:"uuid"` tag uses go-playground/validator's built-in UUID validator. This validates format at the binding layer, before reaching handler logic.

### 6.2 Response DTOs — String IDs in JSON

```go
// BEFORE
type ProductResponse struct {
    ID         uint      `json:"id"`
    CategoryID uint      `json:"category_id"`
    // ...
}

// AFTER
type ProductResponse struct {
    ID         string    `json:"id"`
    CategoryID string    `json:"category_id"`
    // ...
}
```

### 6.3 Mapper Functions

Every module needs mapper functions to convert `uuid.UUID` → `string`:

```go
// products/handler.go (or a new mapper.go)
func toProductResponse(p Product) ProductResponse {
    return ProductResponse{
        ID:         p.ID.String(),
        CategoryID: p.CategoryID.String(),
        Name:       p.Name,
        Slug:       p.Slug,
        Brand:      p.Brand,
        Description: p.Description,
        BasePrice:  p.BasePrice,
        IsActive:   p.IsActive,
        CreatedAt:  p.CreatedAt,
        UpdatedAt:  p.UpdatedAt,
    }
}
```

### 6.4 All DTOs Affected

| File | Types with ID fields |
|------|---------------------|
| `products/dto.go` | CreateCategoryRequest (ParentID), UpdateCategoryRequest (ParentID), CategoryResponse (ID, ParentID), CreateProductRequest (CategoryID), UpdateProductRequest (CategoryID), ProductResponse (ID, CategoryID), CreateProductVariantRequest (ProductID), ProductVariantResponse (ID, ProductID), CreateProductImageRequest (ProductID, VariantID), UpdateProductImageRequest, ProductImageResponse (ID, ProductID, VariantID) |
| `orders/model.go` | CreateOrderItemRequest (ProductID, VariantID), OrderResponse (ID, UserID), OrderItemResponse (ID, ProductID, VariantID) |
| `users/model.go` | UserResponse (ID) |
| `payments/dto.go` | CreatePaymentLinkRequest (OrderID), PaymentResponse (ID, OrderID), PaymentLinkResponse (ID, OrderID) |
| `cart/dto.go` | AddToCartRequest (VariantID), CartResponse (UserID), CartItemResponse (VariantID), CheckoutResponse (OrderID), CheckoutItemResponse (ProductID, VariantID) |
| `admin/dto.go` | UserResponse (ID) |
| `inventory/model.go` | InventoryResponse (ID, ProductID) |
| `auth/dto.go` | TokenClaims (UserID) |

---

## 7. Category Hierarchy Redesign

### 7.1 Current vs New Approach

| Aspect | Current (path-based) | New (recursive CTE) |
|--------|---------------------|---------------------|
| Path storage | `/1/5/12/` string | None — computed at query time |
| Parent change | Update entire subtree path | Update single `parent_id` + recalculate depth of subtree |
| Circular check | `strings.Contains(path, "/12/")` | Recursive CTE descendant check |
| Breadcrumbs | Parse path → IN query | Single recursive CTE query |
| Performance | O(1) read, O(n) write on move | O(depth) read, O(subtree) write on move |

### 7.2 Depth Recalculation on Parent Change

When a category's parent changes, we need to update depth for the entire subtree:

```sql
-- Update depth for the moved category and all descendants
WITH RECURSIVE subtree AS (
    SELECT id, parent_id, 0 as new_depth_offset
    FROM categories
    WHERE id = $1  -- the moved category
    UNION ALL
    SELECT c.id, c.parent_id, s.new_depth_offset + 1
    FROM categories c
    INNER JOIN subtree s ON c.parent_id = s.id
)
UPDATE categories
SET depth = depth + $2  -- $2 = new_parent_depth + 1 - old_depth
WHERE id IN (SELECT id FROM subtree);
```

### 7.3 Performance Implications

- **Read queries (breadcrumbs, tree)**: Slightly slower than path-based (recursive CTE vs simple string parse), but for typical category depths (3-5 levels) the difference is negligible (< 1ms).
- **Write queries (move category)**: Faster — no need to update every descendant's path string. Only depth recalculation needed.
- **Index**: Add composite index on `(parent_id, depth)` for efficient tree queries.

---

## 8. JWT & Auth Design

### 8.1 TokenClaims Structure

```go
// BEFORE
type TokenClaims struct {
    UserID uint   `json:"user_id"`
    Email  string `json:"email"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

// AFTER
type TokenClaims struct {
    UserID string `json:"user_id"` // UUID string
    Email  string `json:"email"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}
```

### 8.2 TokenGenerator Changes

```go
// BEFORE
func (g *TokenGenerator) GenerateAccessToken(userID uint, email, role string) (string, int64, error)

// AFTER
func (g *TokenGenerator) GenerateAccessToken(userID uuid.UUID, email, role string) (string, int64, error) {
    claims := TokenClaims{
        UserID: userID.String(), // Store as string in JWT
        Email:  email,
        Role:   role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    jwtConfig.Issuer,
        },
    }
    // ...
}
```

### 8.3 Middleware Changes

```go
// BEFORE
c.Set("user_id", claims.UserID)     // uint
// In handlers: userID := c.GetUint("user_id")

// AFTER
c.Set("user_id", claims.UserID)     // string (UUID)
// In handlers: userIDStr := c.GetString("user_id")
//              userID, _ := uuid.Parse(userIDStr)
```

### 8.4 RefreshToken Storage

```go
// BEFORE
func (g *TokenGenerator) StoreRefreshToken(token string, userID uint) error
func (g *TokenGenerator) storeRefreshTokenRedis(tokenHash string, userID uint, expiresAt time.Time) error
// Redis: "user_id": userID  (number)

// AFTER
func (g *TokenGenerator) StoreRefreshToken(token string, userID uuid.UUID) error
func (g *TokenGenerator) storeRefreshTokenRedis(tokenHash string, userID uuid.UUID, expiresAt time.Time) error
// Redis: "user_id": userID.String()  (string)
```

### 8.5 tempTokenData (in-memory 2FA)

```go
// BEFORE
type tempTokenData struct {
    UserID    uint
    ExpiresAt time.Time
}

// AFTER
type tempTokenData struct {
    UserID    uuid.UUID
    ExpiresAt time.Time
}
```

### 8.6 All AuthService Methods with userID

Every method accepting `userID uint` changes to `userID uuid.UUID`:
- `SetupTwoFactor(ctx, userID uint)` → `SetupTwoFactor(ctx, userID uuid.UUID)`
- `EnableTwoFactor(ctx, userID uint, code)` → `EnableTwoFactor(ctx, userID uuid.UUID, code)`
- `DisableTwoFactor(ctx, userID uint, code, backupCode)` → `DisableTwoFactor(ctx, userID uuid.UUID, code, backupCode)`
- `VerifyTwoFactor(ctx, userID uint, code)` → `VerifyTwoFactor(ctx, userID uuid.UUID, code)`
- `RevokeAllUserRefreshTokens(userID uint)` → `RevokeAllUserRefreshTokens(userID uuid.UUID)`

### 8.7 JWT Invalidation

**All existing JWT tokens become invalid on deployment.** The `UserID` claim changes from `uint` to `string`. Existing tokens with `user_id: 123` will fail to parse as UUID strings.

**No dual-format migration window needed** — the proposal explicitly states "Full invalidation (users must re-login)." This simplifies the implementation significantly.

---

## 9. Redis Cart Design

### 9.1 Key Format Change

```
BEFORE: cart:123
AFTER:  cart:0195c8a1-b2c3-7d4e-8f90-123456789abc
```

### 9.2 Cart Model Changes

```go
// BEFORE
type Cart struct {
    UserID    uint       `json:"user_id"`
    Items     []CartItem `json:"items"`
}
type CartItem struct {
    VariantID uint `json:"variant_id"`
    Quantity  int  `json:"quantity"`
}

// AFTER
type Cart struct {
    UserID    string     `json:"user_id"`
    Items     []CartItem `json:"items"`
}
type CartItem struct {
    VariantID string `json:"variant_id"`
    Quantity  int     `json:"quantity"`
}
```

### 9.3 Cart Repository Implementation

```go
// BEFORE (conceptual Redis implementation)
func (r *CartRedisRepository) GetCart(userID uint) (*Cart, error) {
    key := fmt.Sprintf("cart:%d", userID)
    data, err := r.redis.Get(ctx, key).Bytes()
    // ...
}

// AFTER
func (r *CartRedisRepository) GetCart(userID uuid.UUID) (*Cart, error) {
    key := fmt.Sprintf("cart:%s", userID.String())
    data, err := r.redis.Get(ctx, key).Bytes()
    // ...
}
```

### 9.4 Migration Strategy

**Clean and recreate** — no dual-read. All existing cart data is cleared on deployment. Users will see empty carts on their next visit.

```go
// On deployment, flush all cart keys
r.redis.Del(ctx, "cart:*")  // Pattern delete via SCAN
```

---

## 10. Database Migration Script

### 10.1 Strategy

GORM `AutoMigrate` cannot change column types from `integer` to `uuid`. The migration approach:

1. **Drop and recreate all tables** with UUID columns (simplest, safest for this project)
2. Run GORM `AutoMigrate` after schema change to ensure consistency

Since this is a **breaking change with full JWT invalidation**, there's no need for a zero-downtime migration. We can drop and recreate.

### 10.2 SQL Migration Script

```sql
-- ============================================================
-- UUID Migration: Drop and recreate all tables with UUID PKs
-- ============================================================
-- IMPORTANT: Run this BEFORE deploying new code
-- All data will be lost. This is a breaking change.
-- ============================================================

BEGIN;

-- 1. Drop all tables (respect FK order — children first)
DROP TABLE IF EXISTS product_variant_attributes CASCADE;
DROP TABLE IF EXISTS product_images CASCADE;
DROP TABLE IF EXISTS product_variants CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS categories CASCADE;
DROP TABLE IF EXISTS order_items CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS payments CASCADE;
DROP TABLE IF EXISTS payment_links CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS inventories CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- 2. Recreate tables with UUID primary keys
-- Order: tables with no FK dependencies first

CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    phone VARCHAR(20),
    role VARCHAR(50) DEFAULT 'customer',
    active BOOLEAN DEFAULT true,
    email_verified BOOLEAN DEFAULT false,
    verification_token VARCHAR(64),
    verification_expires TIMESTAMP,
    reset_token VARCHAR(64),
    reset_expires TIMESTAMP,
    two_fa_secret VARCHAR(255),
    two_fa_enabled BOOLEAN DEFAULT false,
    two_fa_backup_codes TEXT,
    oauth_provider VARCHAR(50),
    oauth_provider_id VARCHAR(255),
    avatar_url VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
CREATE INDEX idx_users_verification_expires ON users(verification_expires);
CREATE INDEX idx_users_reset_expires ON users(reset_expires);

CREATE TABLE categories (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    parent_id UUID,
    depth INT DEFAULT 0 NOT NULL,
    is_active BOOLEAN DEFAULT true,
    sort_order INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);
CREATE INDEX idx_categories_parent_id ON categories(parent_id);
CREATE INDEX idx_categories_depth ON categories(depth);
CREATE INDEX idx_categories_deleted_at ON categories(deleted_at);

CREATE TABLE products (
    id UUID PRIMARY KEY,
    category_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    brand VARCHAR(100),
    description TEXT,
    base_price DECIMAL(12,2) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_products_category FOREIGN KEY (category_id) REFERENCES categories(id)
);
CREATE INDEX idx_products_category_id ON products(category_id);

CREATE TABLE product_variants (
    id UUID PRIMARY KEY,
    product_id UUID NOT NULL,
    sku VARCHAR(100) UNIQUE NOT NULL,
    price DECIMAL(12,2) NOT NULL,
    stock INT DEFAULT 0,
    reserved INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_variants_product FOREIGN KEY (product_id) REFERENCES products(id)
);
CREATE INDEX idx_variants_product_id ON product_variants(product_id);

CREATE TABLE product_variant_attributes (
    id UUID PRIMARY KEY,
    variant_id UUID UNIQUE NOT NULL,
    color VARCHAR(50) NOT NULL,
    size VARCHAR(20) NOT NULL,
    weight VARCHAR(50) NOT NULL,
    CONSTRAINT fk_pva_variant FOREIGN KEY (variant_id) REFERENCES product_variants(id)
);

CREATE TABLE product_images (
    id UUID PRIMARY KEY,
    product_id UUID NOT NULL,
    variant_id UUID,
    url_image TEXT NOT NULL,
    is_main BOOLEAN DEFAULT false,
    sort_order INT DEFAULT 0,
    CONSTRAINT fk_images_product FOREIGN KEY (product_id) REFERENCES products(id),
    CONSTRAINT fk_images_variant FOREIGN KEY (variant_id) REFERENCES product_variants(id)
);
CREATE INDEX idx_product_images_product_id ON product_images(product_id);
CREATE INDEX idx_product_images_variant_id ON product_images(variant_id);

CREATE TABLE orders (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    total_price DECIMAL(10,2),
    shipping_address TEXT,
    notes TEXT,
    payment_transaction_id VARCHAR(255),
    payment_link_id VARCHAR(255),
    payment_status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_deleted_at ON orders(deleted_at);

CREATE TABLE order_items (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL,
    product_id UUID NOT NULL,
    variant_id UUID,
    quantity INT NOT NULL,
    unit_price DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_order_items_order FOREIGN KEY (order_id) REFERENCES orders(id),
    CONSTRAINT fk_order_items_product FOREIGN KEY (product_id) REFERENCES products(id),
    CONSTRAINT fk_order_items_variant FOREIGN KEY (variant_id) REFERENCES product_variants(id)
);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
CREATE INDEX idx_order_items_variant_id ON order_items(variant_id);

CREATE TABLE inventories (
    id UUID PRIMARY KEY,
    product_id UUID UNIQUE NOT NULL,
    quantity INT DEFAULT 0,
    reserved INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT fk_inventory_product FOREIGN KEY (product_id) REFERENCES products(id)
);
CREATE INDEX idx_inventories_product_id ON inventories(product_id);
CREATE INDEX idx_inventories_deleted_at ON inventories(deleted_at);

CREATE TABLE payments (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL,
    wompi_transaction_id VARCHAR(255) UNIQUE,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) DEFAULT 'COP',
    status VARCHAR(50) DEFAULT 'pending',
    payment_method VARCHAR(50),
    payment_token VARCHAR(255),
    redirect_url VARCHAR(500),
    reference VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT fk_payments_order FOREIGN KEY (order_id) REFERENCES orders(id)
);
CREATE INDEX idx_payments_order_id ON payments(order_id);
CREATE INDEX idx_payments_reference ON payments(reference);
CREATE INDEX idx_payments_deleted_at ON payments(deleted_at);

CREATE TABLE payment_links (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL,
    wompi_link_id VARCHAR(255) UNIQUE,
    url VARCHAR(500) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) DEFAULT 'COP',
    description VARCHAR(500),
    status VARCHAR(50) DEFAULT 'active',
    single_use BOOLEAN DEFAULT false,
    expires_at TIMESTAMP,
    redirect_url VARCHAR(500),
    reference VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT fk_payment_links_order FOREIGN KEY (order_id) REFERENCES orders(id)
);
CREATE INDEX idx_payment_links_order_id ON payment_links(order_id);
CREATE INDEX idx_payment_links_expires_at ON payment_links(expires_at);
CREATE INDEX idx_payment_links_reference ON payment_links(reference);
CREATE INDEX idx_payment_links_deleted_at ON payment_links(deleted_at);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY,
    token VARCHAR(255) UNIQUE NOT NULL,
    user_id UUID NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    revoked BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

COMMIT;
```

### 10.3 Table Creation Order (FK Dependencies)

```
1. users              (no FK)
2. categories         (self-referencing FK — parent_id)
3. products           (FK → categories)
4. product_variants   (FK → products)
5. product_variant_attributes (FK → product_variants)
6. product_images     (FK → products, product_variants)
7. orders             (FK → users)
8. order_items        (FK → orders, products, product_variants)
9. inventories        (FK → products)
10. payments          (FK → orders)
11. payment_links     (FK → orders)
12. refresh_tokens    (FK → users)
```

### 10.4 Post-Migration

After running the SQL migration, GORM `AutoMigrate` in `main.go` will reconcile any differences (it's idempotent and won't drop existing tables).

---

## 11. Cache Key Strategy

### 11.1 All Cache Key Patterns Affected

| Current Pattern | New Pattern | Location |
|----------------|-------------|----------|
| `cache:category:{id}` | `cache:category:{uuid}` | CategoryRepository |
| `cache:category:slug:{slug}` | unchanged | CategoryRepository |
| `cache:category:list` | unchanged | CategoryRepository |
| `cache:product:{id}` | `cache:product:{uuid}` | ProductRepository |
| `cache:product:slug:{slug}` | unchanged | ProductRepository |
| `cache:product:list:{offset}:{limit}` | unchanged | ProductRepository |
| `cache:product:list:{offset}:{limit}:{categoryID}` | `cache:product:list:{offset}:{limit}:{uuid}` | ProductRepository |
| `cache:product:search:*` | unchanged | ProductHandler |
| `cache:variant:{id}` | `cache:variant:{uuid}` | ProductVariantRepository |
| `cache:variant:product:{productID}` | `cache:variant:product:{uuid}` | ProductVariantRepository |
| `cache:image:{id}` | `cache:image:{uuid}` | ProductImageRepository |
| `cache:image:product:{productID}` | `cache:image:product:{uuid}` | ProductImageRepository |

### 11.2 Cache Key Generation

```go
// BEFORE
key := r.cache.Key("cache", "product", fmt.Sprintf("%d", id))

// AFTER
key := r.cache.Key("cache", "product", id.String())
```

### 11.3 Cache Invalidation After Migration

All existing cache entries become invalid (they use integer IDs). The cache will naturally repopulate with new UUID-based keys on first access. No explicit cache flush needed, but a flush is recommended to free memory:

```bash
redis-cli KEYS "cache:*" | xargs redis-cli DEL
```

---

## 12. Testing Strategy

### 12.1 Test UUID Generation

```go
// In test files, use uuidutil.MustParse for deterministic test data
testUUID := uuidutil.MustParse("0195c8a1-b2c3-7d4e-8f90-123456789abc")

// Or generate fresh UUIDs for each test
freshUUID := uuidutil.New()
```

### 12.2 Table-Driven Test Changes

```go
// BEFORE
tests := []struct {
    name    string
    id      uint
    wantErr bool
}{
    {"not found", 99999, false},
    {"invalid", 0, false},
}

// AFTER
tests := []struct {
    name    string
    id      uuid.UUID
    wantErr bool
}{
    {"not found", uuidutil.MustParse("00000000-0000-7000-0000-000000000001"), false},
    {"zero UUID", uuid.Nil, false},
}
```

### 12.3 Mock Repository Signature Changes

Every mock or interface that accepts `uint` IDs changes to `uuid.UUID`:

```go
// BEFORE
type VariantStockHandler interface {
    ReserveStock(id uint, quantity int) error
    ConfirmSale(id uint, quantity int) error
    ReleaseStock(id uint, quantity int) error
}

// AFTER
type VariantStockHandler interface {
    ReserveStock(id uuid.UUID, quantity int) error
    ConfirmSale(id uuid.UUID, quantity int) error
    ReleaseStock(id uuid.UUID, quantity int) error
}
```

### 12.4 Handler Test Changes

```go
// BEFORE
w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Params = []gin.Param{{Key: "id", Value: "123"}}

// AFTER
c.Params = []gin.Param{{Key: "id", Value: "0195c8a1-b2c3-7d4e-8f90-123456789abc"}}
```

---

## 13. File Change Map

### 13.1 New Files

| File | Purpose |
|------|---------|
| `internal/shared/uuidutil/uuid.go` | UUID generation, parsing, validation helpers |
| `internal/shared/uuidutil/uuid_test.go` | Tests for uuidutil |
| `migrations/001_uuid_migration.sql` | SQL migration script (Section 10) |

### 13.2 Modified Files — Model Layer

| File | Changes |
|------|---------|
| `internal/modules/products/model.go` | Category, Product, ProductVariant, ProductVariantAttribute, ProductImage: all PKs/FKs → `uuid.UUID`, remove `Path` from Category, add `Depth`, add `BeforeCreate` hooks |
| `internal/modules/orders/model.go` | Order, OrderItem: PKs/FKs → `uuid.UUID`, add `BeforeCreate` hooks |
| `internal/modules/users/model.go` | User: PK → `uuid.UUID`, add `BeforeCreate` hook |
| `internal/modules/payments/model.go` | Payment, PaymentLink: PKs/FKs → `uuid.UUID`, add `BeforeCreate` hooks |
| `internal/modules/cart/model.go` | Cart.UserID → `string`, CartItem.VariantID → `string` |
| `internal/modules/auth/model.go` | RefreshToken: PK, UserID → `uuid.UUID`, add `BeforeCreate` hook |
| `internal/modules/inventory/model.go` | Inventory: PK, ProductID → `uuid.UUID`, add `BeforeCreate` hook |

### 13.3 Modified Files — DTO Layer

| File | Changes |
|------|---------|
| `internal/modules/products/dto.go` | All ID fields: `uint` → `string`, add `binding:"uuid"` where applicable |
| `internal/modules/orders/model.go` | OrderResponse, OrderItemResponse, CreateOrderItemRequest: ID fields → `string` |
| `internal/modules/users/model.go` | UserResponse: ID → `string` |
| `internal/modules/payments/dto.go` | PaymentResponse, PaymentLinkResponse, CreatePaymentLinkRequest: ID fields → `string` |
| `internal/modules/cart/dto.go` | All ID fields → `string` |
| `internal/modules/admin/dto.go` | UserResponse: ID → `string` |
| `internal/modules/inventory/model.go` | InventoryResponse: ID, ProductID → `string` |
| `internal/modules/auth/dto.go` | TokenClaims: UserID → `string` |

### 13.4 Modified Files — Repository Layer

| File | Changes |
|------|---------|
| `internal/modules/products/repository.go` | All method signatures: `uint` → `uuid.UUID`, Category: remove path logic, add recursive CTEs, add cache key changes |
| `internal/modules/orders/repository.go` | All method signatures: `uint` → `uuid.UUID` |
| `internal/modules/users/repository.go` | All method signatures: `uint` → `uuid.UUID` |
| `internal/modules/payments/repository.go` | All method signatures: `uint` → `uuid.UUID` |
| `internal/modules/inventory/repository.go` | All method signatures: `uint` → `uuid.UUID` |
| `internal/modules/cart/repository.go` | Interface: `uint` → `uuid.UUID` |

### 13.5 Modified Files — Handler Layer

| File | Changes |
|------|---------|
| `internal/modules/products/handler.go` | All `strconv.ParseUint` → `uuid.Parse`, all cache invalidation methods: `uint` → `uuid.UUID`, all mapper functions: `uint` → `string` |
| `internal/modules/orders/handler.go` | All `strconv.ParseUint` → `uuid.Parse`, `c.GetUint` → `c.GetString` + `uuid.Parse`, interface signatures: `uint` → `uuid.UUID` |
| `internal/modules/users/handler.go` | `fmt.Sscanf` → `uuid.Parse`, `c.GetUint` → `c.GetString` + `uuid.Parse` |
| `internal/modules/payments/handler.go` | `parseUint` → `uuid.Parse`, `c.GetUint` → `c.GetString` + `uuid.Parse` |
| `internal/modules/inventory/handler.go` | `strconv.ParseUint` → `uuid.Parse` |
| `internal/modules/cart/handler.go` | `c.GetUint` → `c.GetString` + `uuid.Parse`, `parseUintParam` → `parseUUIDParam`, interface signatures: `uint` → `uuid.UUID` |
| `internal/modules/admin/handler.go` | Mapper: `uint` → `string` |

### 13.6 Modified Files — Service/Auth Layer

| File | Changes |
|------|---------|
| `internal/modules/auth/service.go` | All `userID uint` → `userID uuid.UUID`, `tempTokenData.UserID` → `uuid.UUID` |
| `internal/modules/auth/token.go` | `GenerateAccessToken(userID uint, ...)` → `GenerateAccessToken(userID uuid.UUID, ...)`, `StoreRefreshToken(token, userID uint)` → `StoreRefreshToken(token, userID uuid.UUID)`, Redis token data: `user_id` as string |
| `internal/modules/auth/middleware.go` | No direct changes (claims.UserID is already set as string in TokenClaims) |
| `internal/modules/orders/service.go` | All method signatures with `uint` IDs → `uuid.UUID` |
| `internal/modules/cart/service.go` | All method signatures with `uint` IDs → `uuid.UUID` |
| `internal/modules/payments/service.go` | All method signatures with `uint` IDs → `uuid.UUID` |

### 13.7 Modified Files — Infrastructure

| File | Changes |
|------|---------|
| `go.mod` | Replace `github.com/google/uuid` with `github.com/gofrs/uuid v5.3.0` |
| `cmd/api/main.go` | Import `uuidutil`, remove any integer ID seed data |

### 13.8 Modified Files — Tests

| File Pattern | Changes |
|-------------|---------|
| `internal/modules/*/*_test.go` | All test data: `uint` → `uuid.UUID`, test fixtures use `uuidutil.MustParse` or `uuidutil.New()` |

### 13.9 Total File Count

| Category | Count |
|----------|-------|
| New files | 3 |
| Model files | 7 |
| DTO files | 8 |
| Repository files | 6 |
| Handler files | 7 |
| Service/Auth files | 6 |
| Infrastructure files | 2 |
| Test files | ~15 (estimated) |
| **Total** | **~54 files** |

---

## 14. Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| **Data loss** | Full DB backup before migration. Migration is DROP+CREATE — test on staging first. |
| **JWT invalidation** | Communicate as "security upgrade." Users re-login. No dual-format complexity. |
| **Cart data loss** | Acceptable — carts are ephemeral. Clear Redis cart keys on deploy. |
| **Category CTE performance** | Benchmark on staging with realistic data. Add `(parent_id, depth)` composite index. |
| **UUID parsing overhead** | Negligible — `uuid.Parse` is ~50ns. No measurable impact on p95 latency. |
| **go.mod dependency conflict** | `gofrs/uuid` is well-maintained. No known conflicts with existing dependencies. |

---

## 15. Deployment Checklist

- [ ] Run SQL migration on staging database
- [ ] Deploy code to staging
- [ ] Run full test suite (`go test -v ./...`)
- [ ] Manual QA: create products, categories, orders, payments
- [ ] Load test: 1000 concurrent UUID-based requests
- [ ] Run SQL migration on production
- [ ] Flush Redis cache and cart keys
- [ ] Deploy production code
- [ ] Monitor error rates for 24 hours
- [ ] Verify Swagger docs reflect UUID format
- [ ] Remove any migration fallback code after 48 hours
