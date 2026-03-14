# Delta for Products

## Purpose

This spec covers consistency improvements and service layer addition for the Products module.

---

## ADDED Requirements

### Requirement: Unified Response Handler Pattern

The products handler SHALL use `response.ResponseHandler` for all responses, matching the pattern used in users and orders modules.

All product endpoints MUST return responses in the standardized format:
```json
{
  "success": true,
  "message": "optional message",
  "data": { ... },
  "error": "optional error message"
}
```

#### Scenario: Successful product creation

- GIVEN valid product data in request body
- WHEN client POSTs to `/api/v1/products`
- THEN response status is 201 Created
- AND response body follows ResponseHandler format with `success: true`

#### Scenario: Product not found

- GIVEN product with ID 999 does not exist
- WHEN client GETs `/api/v1/products/999`
- THEN response status is 404 Not Found
- AND response body follows ResponseHandler format with `success: false`

#### Scenario: Validation error

- GIVEN invalid product data (missing required fields)
- WHEN client POSTs to `/api/v1/products`
- THEN response status is 400 Bad Request
- AND response body follows ResponseHandler format with validation errors

---

### Requirement: Product Service Layer

The products module SHALL have a service layer that handles business logic, separating it from the HTTP handler.

The service layer MUST:
- Validate business rules (e.g., price must be positive)
- Coordinate between repository calls
- Return domain errors that handlers can translate to HTTP responses

#### Scenario: Create product with business validation

- GIVEN product data with negative price
- WHEN product service creates the product
- THEN service returns validation error
- AND handler returns 400 Bad Request

#### Scenario: Create product with related data

- GIVEN product with category, variants, and images
- WHEN product service creates the product
- THEN service creates product with all related data in single transaction
- AND returns complete product with relations

---

### Requirement: Product Response DTO

The system SHALL use dedicated response DTOs for API responses, excluding internal fields.

Response DTOs MUST exclude:
- `password_hash` (not applicable to products but good practice)
- Internal database IDs that should not be exposed
- Any sensitive or implementation-specific fields

#### Scenario: Product response structure

- GIVEN a product exists in database
- WHEN client requests the product
- THEN response includes only API-appropriate fields
- AND internal implementation details are hidden

---

## MODIFIED Requirements

### Requirement: Handler Response Format (Previously: Mixed gin.H and ResponseHandler)

All product endpoints SHALL use ResponseHandler consistently.

(Previously: Some endpoints used `gin.H{}` directly while others used ResponseHandler)

#### Scenario: GET products list

- GIVEN multiple products exist
- WHEN client GETs `/api/v1/products`
- THEN response uses ResponseHandler format

#### Scenario: GET product by slug

- GIVEN a product with slug "test-product" exists
- WHEN client GETs `/api/v1/products/slug/test-product`
- THEN response uses ResponseHandler format

---

## REMOVED Requirements

### Requirement: Direct gin.H Usage in Handlers

The products handler SHALL NOT use `gin.H{}` or `c.JSON()` directly for responses.

(Reason: Inconsistent with other modules, harder to maintain standardized error responses)

---

## Testing Requirements

### Requirement: Table-Driven Tests

The products module SHALL have comprehensive table-driven tests covering:
- All handler endpoints
- Success cases
- Validation errors
- Not found errors
- Edge cases (empty inputs, max length, etc.)

#### Scenario: Test coverage

- GIVEN product handler has table-driven tests
- WHEN tests run
- THEN each test case is independent
- AND test names describe the scenario clearly

---

### Requirement: Handler Unit Tests

Each product handler function MUST have corresponding unit tests.

#### Scenario: Create endpoint tests

- GIVEN Create handler
- WHEN tested with table-driven approach
- THEN covers: valid input, invalid JSON, missing required fields, duplicate slug
