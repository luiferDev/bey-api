# Delta for Products Module — UUIDv7 Migration

## MODIFIED Requirements

### Requirement: Product Primary Key Type

The system SHALL use UUIDv7 as the primary key for all product-related models instead of auto-incrementing integers.

All product models (`Product`, `Category`, `ProductVariant`, `ProductImage`, `ProductVariantAttribute`) MUST use `uuid.UUID` as their primary key type in GORM models.

(Previously: Primary keys were `uint` with auto-increment)

#### Scenario: Product created with UUIDv7 ID

- GIVEN a valid CreateProductRequest with category_id (UUID string), name, slug, and base_price
- WHEN the product service creates the product
- THEN the system SHALL generate a UUIDv7 ID automatically
- AND the product SHALL be persisted with the UUID primary key
- AND the response SHALL include the UUID string in the `id` field

#### Scenario: Product found by UUID

- GIVEN a product exists with ID `01960c12-3456-7890-abcd-ef1234567890`
- WHEN the client GETs `/api/v1/products/01960c12-3456-7890-abcd-ef1234567890`
- THEN the response status is 200 OK
- AND the response body contains the product with matching UUID `id`

#### Scenario: Product not found by valid UUID

- GIVEN no product exists with ID `01960c12-3456-7890-abcd-ef9999999999`
- WHEN the client GETs `/api/v1/products/01960c12-3456-7890-abcd-ef9999999999`
- THEN the response status is 404 Not Found
- AND the response body indicates product not found

#### Scenario: Product requested with invalid UUID format

- GIVEN a client sends a request with an invalid UUID string
- WHEN the client GETs `/api/v1/products/not-a-uuid`
- THEN the response status is 400 Bad Request
- AND the response body indicates invalid ID format
- AND the system MUST NOT return 500 Internal Server Error

#### Scenario: Product requested with integer ID (legacy format)

- GIVEN a client sends a request with an integer ID
- WHEN the client GETs `/api/v1/products/123`
- THEN the response status is 400 Bad Request
- AND the response body indicates invalid ID format

### Requirement: Product Foreign Key Types

All foreign key fields in product-related models MUST use `uuid.UUID` type instead of `uint`.

This includes:
- `Product.CategoryID` → `uuid.UUID`
- `ProductVariant.ProductID` → `uuid.UUID`
- `ProductImage.ProductID` → `uuid.UUID`
- `ProductImage.VariantID` → `uuid.UUID` (nullable)
- `ProductVariantAttribute.VariantID` → `uuid.UUID`

(Previously: Foreign keys were `uint`)

#### Scenario: Product created with UUID category reference

- GIVEN a category exists with UUID `01960c12-0000-0000-0000-000000000001`
- WHEN a product is created with `category_id: "01960c12-0000-0000-0000-000000000001"`
- THEN the product is created with the correct foreign key reference
- AND the relationship is queryable via GORM preloads

#### Scenario: Product created with non-existent category UUID

- GIVEN no category exists with UUID `01960c12-0000-0000-0000-999999999999`
- WHEN a product is created with `category_id: "01960c12-0000-0000-0000-999999999999"`
- THEN the system returns 400 Bad Request with foreign key constraint error

### Requirement: Product DTO ID Fields

All product-related DTOs (request and response) MUST use `string` type for ID fields instead of `uint`.

Response DTOs MUST convert `uuid.UUID` to string using `.String()` method.
Request DTOs MUST accept UUID strings and parse them using `uuid.FromString()`.

(Previously: DTO ID fields were `uint`)

#### Scenario: Product response contains string IDs

- GIVEN a product with UUID `01960c12-3456-7890-abcd-ef1234567890` and category UUID `01960c12-0000-0000-0000-000000000001`
- WHEN the product is returned via API response
- THEN the JSON contains `"id": "01960c12-3456-7890-abcd-ef1234567890"`
- AND the JSON contains `"category_id": "01960c12-0000-0000-0000-000000000001"`

#### Scenario: Create product request accepts UUID string category_id

- GIVEN a client sends `POST /api/v1/products` with `{"category_id": "01960c12-0000-0000-0000-000000000001", "name": "Test", ...}`
- WHEN the request is processed
- THEN the category_id is parsed as UUID successfully
- AND the product is created with the correct category reference

#### Scenario: Create product request rejects invalid UUID string

- GIVEN a client sends `POST /api/v1/products` with `{"category_id": "not-a-uuid", "name": "Test", ...}`
- WHEN the request is processed
- THEN the response status is 400 Bad Request
- AND the response indicates invalid category_id format

### Requirement: Product Repository Signatures

All product repository methods that accept or return IDs MUST use `uuid.UUID` instead of `uint`.

This includes:
- `FindByID(id uuid.UUID) (*Product, error)`
- `Delete(id uuid.UUID) error`
- `FindByCategoryID(categoryID uuid.UUID) ([]Product, error)`
- All other methods accepting product, category, or variant IDs

(Previously: Repository methods accepted `uint` IDs)

#### Scenario: Repository FindByID with valid UUID

- GIVEN a product exists with UUID `01960c12-3456-7890-abcd-ef1234567890`
- WHEN `FindByID(uuid.FromString("01960c12-3456-7890-abcd-ef1234567890"))` is called
- THEN the method returns the product with nil error

#### Scenario: Repository FindByID with nil UUID

- GIVEN no product exists
- WHEN `FindByID(uuid.Nil)` is called
- THEN the method returns nil, nil (not found, not an error)

### Requirement: Product Handler ID Parsing

All product handlers MUST parse incoming ID parameters using `uuid.FromString()` instead of `strconv.ParseUint()`.

(Previously: Handlers used `strconv.ParseUint(c.Param("id"), 10, 32)`)

#### Scenario: Handler parses valid UUID parameter

- GIVEN a request with param `id = "01960c12-3456-7890-abcd-ef1234567890"`
- WHEN the handler parses the ID
- THEN `uuid.FromString()` returns a valid UUID with nil error
- AND the handler proceeds to query the repository

#### Scenario: Handler rejects malformed UUID parameter

- GIVEN a request with param `id = "abc123"`
- WHEN the handler attempts to parse the ID
- THEN `uuid.FromString()` returns an error
- AND the handler returns 400 Bad Request immediately

### Requirement: Product Route Parameters

All product route parameters accepting `:id` MUST accept UUID strings instead of integers.

Route patterns remain structurally the same (e.g., `/products/:id`) but the handler logic changes to expect UUID format.

(Previously: Route parameters were parsed as integers)

#### Scenario: Route accepts UUID string parameter

- GIVEN the route `GET /api/v1/products/:id`
- WHEN a request is made to `/api/v1/products/01960c12-3456-7890-abcd-ef1234567890`
- THEN the route matches and the handler receives the UUID string as `:id`

### Requirement: Variant Attribute ID References

Product variant attributes MUST reference their parent variant using `uuid.UUID` instead of `uint`.

(Previously: `ProductVariantAttribute.VariantID` was `uint`)

#### Scenario: Variant attribute created with UUID variant reference

- GIVEN a variant exists with UUID `01960c12-0000-0000-0000-000000000002`
- WHEN a variant attribute is created for this variant
- THEN the attribute's `VariantID` is set to the UUID
- AND the relationship is queryable
