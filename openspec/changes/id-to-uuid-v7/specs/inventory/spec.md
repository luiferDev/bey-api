# Delta for Inventory Module — UUIDv7 Migration

## MODIFIED Requirements

### Requirement: Inventory Model Primary Key Type

The system SHALL use UUIDv7 as the primary key for all inventory-related models instead of auto-incrementing integers.

All inventory models (`StockMovement`) MUST use `uuid.UUID` as their primary key type in GORM models.

(Previously: Primary keys were `uint` with auto-increment)

#### Scenario: Stock movement created with UUIDv7 ID

- GIVEN a valid stock movement is recorded
- WHEN the inventory service creates the stock movement
- THEN the system SHALL generate a UUIDv7 ID automatically
- AND the stock movement SHALL be persisted with the UUID primary key
- AND the response SHALL include the UUID string in the `id` field

#### Scenario: Stock movement found by UUID

- GIVEN a stock movement exists with ID `01960c12-3456-7890-abcd-ef1234567890`
- WHEN the client GETs `/api/v1/inventory/movements/01960c12-3456-7890-abcd-ef1234567890`
- THEN the response status is 200 OK
- AND the response body contains the stock movement with matching UUID `id`

#### Scenario: Stock movement requested with invalid UUID format

- GIVEN a client sends a request with an invalid UUID string
- WHEN the client GETs `/api/v1/inventory/movements/not-a-uuid`
- THEN the response status is 400 Bad Request
- AND the response body indicates invalid ID format

### Requirement: Inventory Foreign Key Types

All foreign key fields in inventory-related models MUST use `uuid.UUID` type instead of `uint`.

This includes:
- `StockMovement.ProductID` → `uuid.UUID`
- `StockMovement.VariantID` → `uuid.UUID`
- `StockMovement.UserID` → `uuid.UUID` (nullable, for manual adjustments)

(Previously: Foreign keys were `uint`)

#### Scenario: Stock movement created with UUID references

- GIVEN a product with UUID `01960c12-0000-0000-0000-000000000002` and variant UUID `01960c12-0000-0000-0000-000000000003`
- WHEN a stock movement is recorded
- THEN the movement's `ProductID` and `VariantID` are set to the UUIDs
- AND the relationships are queryable

### Requirement: Inventory DTO ID Fields

All inventory-related DTOs MUST use `string` type for ID fields instead of `uint`.

(Previously: DTO ID fields were `uint`)

#### Scenario: Stock movement response contains string IDs

- GIVEN a stock movement with UUID `01960c12-3456-7890-abcd-ef1234567890` and product UUID `01960c12-0000-0000-0000-000000000002`
- WHEN the stock movement is returned via API response
- THEN the JSON contains `"id": "01960c12-3456-7890-abcd-ef1234567890"`
- AND the JSON contains `"product_id": "01960c12-0000-0000-0000-000000000002"`

### Requirement: Inventory Repository Signatures

All inventory repository methods that accept or return IDs MUST use `uuid.UUID` instead of `uint`.

This includes:
- `FindByID(id uuid.UUID) (*StockMovement, error)`
- `FindByProductID(productID uuid.UUID) ([]StockMovement, error)`
- `FindByVariantID(variantID uuid.UUID) ([]StockMovement, error)`
- All other methods accepting inventory, product, or variant IDs

(Previously: Repository methods accepted `uint` IDs)

### Requirement: Inventory Handler ID Parsing

All inventory handlers MUST parse incoming ID parameters using `uuid.FromString()` instead of `strconv.ParseUint()`.

(Previously: Handlers used `strconv.ParseUint(c.Param("id"), 10, 32)`)
