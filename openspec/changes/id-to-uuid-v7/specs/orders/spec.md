# Delta for Orders Module — UUIDv7 Migration

## MODIFIED Requirements

### Requirement: Order Primary Key Type

The system SHALL use UUIDv7 as the primary key for all order-related models instead of auto-incrementing integers.

All order models (`Order`, `OrderItem`) MUST use `uuid.UUID` as their primary key type in GORM models.

(Previously: Primary keys were `uint` with auto-increment)

#### Scenario: Order created with UUIDv7 ID

- GIVEN a valid order creation request with items, shipping address, and payment info
- WHEN the order service creates the order
- THEN the system SHALL generate a UUIDv7 ID automatically
- AND the order SHALL be persisted with the UUID primary key
- AND the response SHALL include the UUID string in the `id` field

#### Scenario: Order found by UUID

- GIVEN an order exists with ID `01960c12-3456-7890-abcd-ef1234567890`
- WHEN the client GETs `/api/v1/orders/01960c12-3456-7890-abcd-ef1234567890`
- THEN the response status is 200 OK
- AND the response body contains the order with matching UUID `id`

#### Scenario: Order not found by valid UUID

- GIVEN no order exists with ID `01960c12-3456-7890-abcd-ef9999999999`
- WHEN the client GETs `/api/v1/orders/01960c12-3456-7890-abcd-ef9999999999`
- THEN the response status is 404 Not Found

#### Scenario: Order requested with invalid UUID format

- GIVEN a client sends a request with an invalid UUID string
- WHEN the client GETs `/api/v1/orders/not-a-uuid`
- THEN the response status is 400 Bad Request
- AND the response body indicates invalid ID format
- AND the system MUST NOT return 500 Internal Server Error

### Requirement: Order Foreign Key Types

All foreign key fields in order-related models MUST use `uuid.UUID` type instead of `uint`.

This includes:
- `Order.UserID` → `uuid.UUID`
- `OrderItem.OrderID` → `uuid.UUID`
- `OrderItem.ProductID` → `uuid.UUID`
- `OrderItem.VariantID` → `uuid.UUID`

(Previously: Foreign keys were `uint`)

#### Scenario: Order created with UUID user reference

- GIVEN a user exists with UUID `01960c12-0000-0000-0000-000000000001`
- WHEN an order is created for this user
- THEN the order's `UserID` is set to the UUID
- AND the relationship is queryable via GORM preloads

#### Scenario: Order item created with UUID product/variant references

- GIVEN a product with UUID `01960c12-0000-0000-0000-000000000002` and variant UUID `01960c12-0000-0000-0000-000000000003`
- WHEN an order item is created
- THEN the order item's `ProductID` and `VariantID` are set to the UUIDs
- AND the relationships are queryable

### Requirement: Order DTO ID Fields

All order-related DTOs MUST use `string` type for ID fields instead of `uint`.

Response DTOs MUST convert `uuid.UUID` to string using `.String()` method.
Request DTOs MUST accept UUID strings and parse them using `uuid.FromString()`.

(Previously: DTO ID fields were `uint`)

#### Scenario: Order response contains string IDs

- GIVEN an order with UUID `01960c12-3456-7890-abcd-ef1234567890` and user UUID `01960c12-0000-0000-0000-000000000001`
- WHEN the order is returned via API response
- THEN the JSON contains `"id": "01960c12-3456-7890-abcd-ef1234567890"`
- AND the JSON contains `"user_id": "01960c12-0000-0000-0000-000000000001"`

#### Scenario: Order item response contains UUID references

- GIVEN an order item with product UUID `01960c12-0000-0000-0000-000000000002` and variant UUID `01960c12-0000-0000-0000-000000000003`
- WHEN the order item is returned via API response
- THEN the JSON contains `"product_id": "01960c12-0000-0000-0000-000000000002"`
- AND the JSON contains `"variant_id": "01960c12-0000-0000-0000-000000000003"`

### Requirement: Order Repository Signatures

All order repository methods that accept or return IDs MUST use `uuid.UUID` instead of `uint`.

This includes:
- `FindByID(id uuid.UUID) (*Order, error)`
- `FindByUserID(userID uuid.UUID) ([]Order, error)`
- `Delete(id uuid.UUID) error`
- All other methods accepting order, user, product, or variant IDs

(Previously: Repository methods accepted `uint` IDs)

### Requirement: Order Handler ID Parsing

All order handlers MUST parse incoming ID parameters using `uuid.FromString()` instead of `strconv.ParseUint()`.

(Previously: Handlers used `strconv.ParseUint(c.Param("id"), 10, 32)`)

#### Scenario: Order handler rejects malformed UUID

- GIVEN a request with param `id = "abc123"`
- WHEN the order handler attempts to parse the ID
- THEN `uuid.FromString()` returns an error
- AND the handler returns 400 Bad Request immediately

### Requirement: Order Status Task Tracking with UUID

The system SHALL provide task status tracking for order processing, allowing clients to check the status of their order creation using UUID order IDs.

(Previously: Task tracking referenced integer order IDs)

#### Scenario: Order task includes UUID order ID on completion

- GIVEN a successfully processed order task
- WHEN the task status is retrieved
- THEN the result SHALL include the created order UUID string
- AND SHALL include the final order status

### Requirement: Order Status Update on Payment with UUID

When a payment is approved, the associated order MUST be updated to "confirmed" status using UUID references.

(Previously: Order lookup used integer ID from payment reference)

#### Scenario: Payment approved via webhook with UUID order reference

- GIVEN order has status "pending" with UUID payment reference
- WHEN payment webhook received with status APPROVED
- THEN order status updated to "confirmed" using UUID lookup
- AND order.payment_status updated to "completed"
