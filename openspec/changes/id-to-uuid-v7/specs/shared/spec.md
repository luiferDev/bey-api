# Delta for Shared Utilities — UUIDv7 Migration

## ADDED Requirements

### Requirement: UUID Utility Package

The system SHALL provide a shared UUID utility package (`internal/shared/uuidutil/`) for common UUID operations.

The package MUST provide:
- `GenerateV7() uuid.UUID` — generates a new UUIDv7
- `FromString(s string) (uuid.UUID, error)` — parses a string to UUID with consistent error handling
- `IsValid(s string) bool` — validates if a string is a valid UUID format
- `Nil() uuid.UUID` — returns the nil UUID constant

#### Scenario: GenerateV7 creates valid UUIDv7

- WHEN `uuidutil.GenerateV7()` is called
- THEN a valid UUIDv7 is returned
- AND the UUID is monotonically sortable by time
- AND the UUID is not nil

#### Scenario: FromString parses valid UUID

- GIVEN a valid UUID string `"01960c12-3456-7890-abcd-ef1234567890"`
- WHEN `uuidutil.FromString("01960c12-3456-7890-abcd-ef1234567890")` is called
- THEN a valid `uuid.UUID` is returned with nil error

#### Scenario: FromString rejects invalid UUID

- GIVEN an invalid UUID string `"not-a-uuid"`
- WHEN `uuidutil.FromString("not-a-uuid")` is called
- THEN an error is returned
- AND the error message indicates invalid UUID format

#### Scenario: IsValid validates UUID format

- GIVEN the string `"01960c12-3456-7890-abcd-ef1234567890"`
- WHEN `uuidutil.IsValid("01960c12-3456-7890-abcd-ef1234567890")` is called
- THEN `true` is returned

#### Scenario: IsValid rejects integer string

- GIVEN the string `"123"`
- WHEN `uuidutil.IsValid("123")` is called
- THEN `false` is returned

#### Scenario: IsValid rejects empty string

- GIVEN an empty string `""`
- WHEN `uuidutil.IsValid("")` is called
- THEN `false` is returned

### Requirement: Cache Key Format with UUIDs

All cache keys that previously used integer ID format (`%d`) SHALL use UUID string format (`%s`).

Cache key format changes:
- `product:%d` → `product:%s` (e.g., `product:01960c12-3456-7890-abcd-ef1234567890`)
- `category:%d` → `category:%s`
- `user:%d` → `user:%s`
- `order:%d` → `order:%s`
- `variant:%d` → `variant:%s`
- Any other cache key pattern using `%d` for IDs

(Previously: Cache keys used `fmt.Sprintf("pattern:%d", id)` with integer IDs)

#### Scenario: Product cache key uses UUID string

- GIVEN a product with UUID `01960c12-3456-7890-abcd-ef1234567890`
- WHEN the cache key is generated
- THEN the key is `product:01960c12-3456-7890-abcd-ef1234567890`

#### Scenario: Category cache key uses UUID string

- GIVEN a category with UUID `01960c12-0000-0000-0000-000000000001`
- WHEN the cache key is generated
- THEN the key is `category:01960c12-0000-0000-0000-000000000001`

#### Scenario: Cache lookup with UUID key

- GIVEN a cached value exists at key `product:01960c12-3456-7890-abcd-ef1234567890`
- WHEN the system looks up the product cache
- THEN the value is retrieved from the UUID-based key
- AND the legacy integer key is NOT checked
