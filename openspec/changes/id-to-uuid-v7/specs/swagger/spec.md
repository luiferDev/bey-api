# Delta for Swagger Documentation — UUIDv7 Migration

## MODIFIED Requirements

### Requirement: Swagger Parameter Types — ID Parameters

All Swagger `@Param` annotations for ID path parameters MUST use `string` type instead of `int`.

All handler annotations MUST change from:
```go
// @Param id path int true "Resource ID"
```
to:
```go
// @Param id path string true "Resource ID (UUIDv7)"
```

(Previously: ID parameters were documented as `int` type)

#### Scenario: Product endpoints document UUID string ID

- GIVEN the Swagger documentation is generated
- WHEN viewing the GET `/api/v1/products/:id` endpoint
- THEN the documentation shows `id` parameter as type `string`
- AND the description indicates it is a UUIDv7 format

#### Scenario: Order endpoints document UUID string ID

- GIVEN the Swagger documentation is generated
- WHEN viewing the GET `/api/v1/orders/:id` endpoint
- THEN the documentation shows `id` parameter as type `string`

#### Scenario: User endpoints document UUID string ID

- GIVEN the Swagger documentation is generated
- WHEN viewing the GET `/api/v1/users/:id` endpoint
- THEN the documentation shows `id` parameter as type `string`

### Requirement: Swagger Response Schemas — ID Field Types

All Swagger response schemas MUST define ID fields as `string` type instead of `integer`.

Schema definitions MUST reflect:
- `"id": { "type": "string", "format": "uuid" }`
- `"user_id": { "type": "string", "format": "uuid" }`
- `"product_id": { "type": "string", "format": "uuid" }`
- `"order_id": { "type": "string", "format": "uuid" }`
- All other `*_id` fields as string with UUID format

(Previously: ID fields were `"type": "integer"`)

#### Scenario: Product response schema shows string ID

- GIVEN the Swagger documentation is generated
- WHEN viewing the ProductResponse schema
- THEN the `id` field is defined as `string` with `format: uuid`
- AND the `category_id` field is defined as `string` with `format: uuid`

#### Scenario: Order response schema shows string IDs

- GIVEN the Swagger documentation is generated
- WHEN viewing the OrderResponse schema
- THEN the `id` field is defined as `string` with `format: uuid`
- AND the `user_id` field is defined as `string` with `format: uuid`

### Requirement: Swagger Request Schemas — ID Field Types

All Swagger request schemas that accept ID references MUST define them as `string` type.

(Previously: Request schemas accepted integer IDs)

#### Scenario: Create product request schema accepts UUID string

- GIVEN the Swagger documentation is generated
- WHEN viewing the CreateProductRequest schema
- THEN the `category_id` field is defined as `string` with `format: uuid`

#### Scenario: Create order request schema accepts UUID strings

- GIVEN the Swagger documentation is generated
- WHEN viewing the CreateOrderRequest schema
- THEN the `user_id` field is defined as `string` with `format: uuid`
- AND order item `product_id` and `variant_id` fields are defined as `string` with `format: uuid`

### Requirement: Swagger Example Values

All Swagger example values in annotations MUST use valid UUIDv7 format strings instead of integers.

Examples MUST use format like:
- `"01960c12-3456-7890-abcd-ef1234567890"` instead of `123`

(Previously: Examples used integer values like `1`, `42`, `999`)

#### Scenario: Example values use UUID format

- GIVEN the Swagger documentation is generated
- WHEN viewing any endpoint with example values
- THEN all ID example values are valid UUID strings
- AND no integer ID examples remain
