# Delta for Cart Module — UUIDv7 Migration

## MODIFIED Requirements

### Requirement: Redis Cart Key Format

The system SHALL use UUID strings in Redis cart keys instead of integer user IDs.

The cart key format SHALL change from `cart:%d` (integer) to `cart:%s` (UUID string).

(Previously: Cart keys were `cart:{integer_user_id}`)

#### Scenario: Cart stored with UUID key

- GIVEN an authenticated user with UUID `01960c12-3456-7890-abcd-ef1234567890`
- WHEN the user adds an item to their cart
- THEN the Redis key SHALL be `cart:01960c12-3456-7890-abcd-ef1234567890`

#### Scenario: Cart retrieved with UUID key

- GIVEN a cart exists at Redis key `cart:01960c12-3456-7890-abcd-ef1234567890`
- WHEN the user requests their cart
- THEN the system fetches data from `cart:01960c12-3456-7890-abcd-ef1234567890`
- AND returns the cart contents

### Requirement: Redis Cart Serialization — Variant ID Type

The system SHALL store `variant_id` as a UUID string in Redis cart serialization instead of an integer.

Cart item serialization format:
```json
{
  "variant_id": "01960c12-0000-0000-0000-000000000003",
  "quantity": 2
}
```

(Previously: `variant_id` was stored as integer)

#### Scenario: Cart item stored with UUID variant_id

- GIVEN a user adds variant `01960c12-0000-0000-0000-000000000003` with quantity 2
- WHEN the cart is serialized to Redis
- THEN the stored JSON contains `"variant_id": "01960c12-0000-0000-0000-000000000003"`

#### Scenario: Cart item deserialized with UUID variant_id

- GIVEN Redis contains serialized cart with UUID variant_id
- WHEN the cart is deserialized
- THEN the variant_id is parsed as UUID string successfully
- AND the variant is enriched with current product data from database

### Requirement: Cart Expiration (Unchanged)

The system SHALL automatically expire cart data after 7 days of inactivity. Each cart operation SHALL reset the TTL.

(This requirement is unchanged — only the key format changes)

### Requirement: Cart to Order Conversion with UUIDs

The system SHALL convert cart items into an order using UUID references for all IDs.

When converting cart to order:
- Cart `variant_id` (UUID string) is used to look up the variant
- Order is created with UUID user ID
- Order items are created with UUID product_id and variant_id
- Order ID is generated as UUIDv7

(Previously: Cart conversion used integer IDs throughout)

#### Scenario: Cart converted to order with UUID references

- GIVEN a user with UUID `01960c12-0000-0000-0000-000000000001` has cart items with UUID variant IDs
- WHEN the user initiates checkout
- THEN the system creates an order with UUIDv7 ID
- AND order items reference UUID product_id and variant_id
- AND the cart is cleared from Redis

### Requirement: User Ownership with UUID

The system SHALL enforce that users can only access and modify their own cart using UUID-based user identity.

(Previously: User identity was derived from integer JWT claims)

#### Scenario: User accesses own cart with UUID identity

- GIVEN an authenticated user with JWT containing UUID `user_id`
- WHEN the user makes a request to get, update, or delete their cart
- THEN the system SHALL use the UUID user ID from the JWT token
- AND SHALL only allow operations on the cart belonging to that UUID
