# Shopping Cart Specification

## Purpose

This specification defines the shopping cart functionality for authenticated users, enabling persistent cart storage using Redis with automatic expiration, stock validation, and cart-to-order conversion.

## ADDED Requirements

### Requirement: Cart Storage Structure

The system SHALL store only `variant_id` and `quantity` in Redis for each cart item. The system MUST extract `product_id`, `price`, and validate `stock` dynamically from the variant when needed.

#### Scenario: Minimal cart data storage

- GIVEN an authenticated user with a valid JWT token
- WHEN the user adds a variant to their cart
- THEN the system SHALL store only `{variant_id, quantity}` in Redis
- AND the key format SHALL be `cart:{user_id}`

#### Scenario: Cart data retrieval with dynamic enrichment

- GIVEN a user with items in their cart stored in Redis
- WHEN the user requests their cart
- THEN the system SHALL fetch `variant_id` and `quantity` from Redis
- AND SHALL dynamically enrich each item with current `product_id`, `price`, `product_name`, and `variant_name` from the database

### Requirement: Redis Configuration

The system SHALL use a dedicated Redis connection for cart storage, separate from rate-limiter Redis. The cart Redis configuration MUST be loaded from `config.yaml`.

#### Scenario: Cart uses separate Redis connection

- GIVEN the application configuration has cart-specific Redis settings
- WHEN the cart module initializes
- THEN it SHALL connect to the Redis instance specified in `config.yaml` under `cart.redis`

### Requirement: Cart Expiration

The system SHALL automatically expire cart data after 7 days of inactivity. Each cart operation SHALL reset the TTL.

#### Scenario: Cart expires after 7 days

- GIVEN a cart that was last modified 7 days ago
- WHEN any cart operation is attempted
- THEN the cart SHALL have expired and no longer be accessible
- AND the Redis key SHALL have been deleted

#### Scenario: Cart TTL resets on activity

- GIVEN a cart with items stored in Redis with less than 7 days until expiration
- WHEN the user adds, updates, or removes an item from their cart
- THEN the cart TTL SHALL be reset to 7 days

### Requirement: Stock Validation

The system SHALL validate that the requested variant has sufficient stock before adding or updating items in the cart.

#### Scenario: Adding item with sufficient stock

- GIVEN a variant with stock >= requested quantity
- WHEN a user adds this variant to their cart
- THEN the operation SHALL succeed
- AND the item SHALL be stored in the cart

#### Scenario: Adding item with insufficient stock

- GIVEN a variant with stock < requested quantity
- WHEN a user attempts to add this variant to their cart
- THEN the operation SHALL fail
- AND the system SHALL return an error indicating insufficient stock

### Requirement: Cart to Order Conversion

The system SHALL provide functionality to convert cart items into an order, with proper price capture at checkout time.

#### Scenario: Converting cart to order

- GIVEN a user with items in their cart
- WHEN the user initiates checkout
- THEN the system SHALL convert each cart item to an order item
- AND SHALL capture the current price of each variant
- AND SHALL validate stock one more time before order creation

### Requirement: User Ownership

The system SHALL enforce that users can only access and modify their own cart.

#### Scenario: User accesses own cart

- GIVEN an authenticated user with JWT containing their user ID
- WHEN the user makes a request to get, update, or delete their cart
- THEN the system SHALL use the user ID from the JWT token
- AND SHALL only allow operations on the cart belonging to that user ID

#### Scenario: User tries to access another user's cart

- GIVEN an authenticated user with JWT token
- WHEN the user attempts to access a cart belonging to a different user ID
- THEN the system SHALL reject the request
- AND SHALL return a 403 Forbidden error