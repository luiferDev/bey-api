# Delta for Payments Module — UUIDv7 Migration

## MODIFIED Requirements

### Requirement: Payment Primary Key Type

The system SHALL use UUIDv7 as the primary key for all payment-related models instead of auto-incrementing integers.

All payment models (`Payment`, `WebhookLog`) MUST use `uuid.UUID` as their primary key type in GORM models.

(Previously: Primary keys were `uint` with auto-increment)

#### Scenario: Payment created with UUIDv7 ID

- GIVEN a valid payment creation request with order_id and amount
- WHEN the payment service creates the payment
- THEN the system SHALL generate a UUIDv7 ID automatically
- AND the payment SHALL be persisted with the UUID primary key
- AND the response SHALL include the UUID string in the `id` field

#### Scenario: Payment found by UUID

- GIVEN a payment exists with ID `01960c12-3456-7890-abcd-ef1234567890`
- WHEN the client GETs `/api/v1/payments/01960c12-3456-7890-abcd-ef1234567890`
- THEN the response status is 200 OK
- AND the response body contains the payment with matching UUID `id`

#### Scenario: Payment requested with invalid UUID format

- GIVEN a client sends a request with an invalid UUID string
- WHEN the client GETs `/api/v1/payments/not-a-uuid`
- THEN the response status is 400 Bad Request
- AND the response body indicates invalid ID format

### Requirement: Payment Foreign Key Types

All foreign key fields in payment-related models MUST use `uuid.UUID` type instead of `uint`.

This includes:
- `Payment.OrderID` → `uuid.UUID`
- `Payment.UserID` → `uuid.UUID`
- `WebhookLog.PaymentID` → `uuid.UUID` (nullable)

(Previously: Foreign keys were `uint`)

#### Scenario: Payment created with UUID order reference

- GIVEN an order exists with UUID `01960c12-0000-0000-0000-000000000001`
- WHEN a payment is created for this order
- THEN the payment's `OrderID` is set to the UUID
- AND the relationship is queryable via GORM preloads

### Requirement: Payment DTO ID Fields

All payment-related DTOs MUST use `string` type for ID fields instead of `uint`.

(Previously: DTO ID fields were `uint`)

#### Scenario: Payment response contains string IDs

- GIVEN a payment with UUID `01960c12-3456-7890-abcd-ef1234567890` and order UUID `01960c12-0000-0000-0000-000000000001`
- WHEN the payment is returned via API response
- THEN the JSON contains `"id": "01960c12-3456-7890-abcd-ef1234567890"`
- AND the JSON contains `"order_id": "01960c12-0000-0000-0000-000000000001"`

### Requirement: Payment Repository Signatures

All payment repository methods that accept or return IDs MUST use `uuid.UUID` instead of `uint`.

This includes:
- `FindByID(id uuid.UUID) (*Payment, error)`
- `FindByOrderID(orderID uuid.UUID) (*Payment, error)`
- All other methods accepting payment, order, or user IDs

(Previously: Repository methods accepted `uint` IDs)

### Requirement: Payment Handler ID Parsing

All payment handlers MUST parse incoming ID parameters using `uuid.FromString()` instead of `strconv.ParseUint()`.

(Previously: Handlers used `strconv.ParseUint(c.Param("id"), 10, 32)`)

### Requirement: Webhook Handling with UUID Order References

The webhook handler SHALL process Wompi webhooks that reference orders by UUID.

The webhook handler MUST:
- Accept POST requests at `/api/v1/payments/webhook`
- Verify signature using presign_key
- Look up the associated payment by its internal reference
- Update the associated order status using UUID order ID
- Implement idempotency to prevent duplicate processing

(Previously: Webhook handler looked up orders by integer ID)

#### Scenario: Process APPROVED webhook with UUID order reference

- GIVEN Wompi sends webhook for a payment associated with order UUID `01960c12-0000-0000-0000-000000000001`
- WHEN system verifies signature and processes webhook
- THEN payment status updated to APPROVED
- AND order with UUID `01960c12-0000-0000-0000-000000000001` status updated to confirmed
- AND deduplication prevents re-processing same event

### Requirement: Payment Link Creation with UUID

Payment links SHALL reference orders using UUID strings.

(Previously: Payment links referenced orders by integer ID)

#### Scenario: Create payment link for UUID order

- GIVEN authenticated user requests payment link for order UUID `01960c12-0000-0000-0000-000000000001`
- WHEN client requests `POST /api/v1/payments/links` with order reference
- THEN system creates PaymentLink with UUID order reference
- AND system calls Wompi API to create payment link
- AND response includes the payment link URL
