# Products Specification

## Purpose

This specification defines the async and parallel processing capabilities for the Products module, enabling bulk operations and parallel data fetching for improved performance, as well as consistency patterns and service layer architecture.

## Requirements

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

---

### Requirement: Bulk Product Operations via Async Tasks

The system SHALL support bulk product operations (bulk create, bulk update, bulk delete) submitted as background tasks. These operations SHALL NOT block the API response.

#### Scenario: Bulk product update submitted as async task

- GIVEN a bulk product update request with list of product IDs and update data
- WHEN the bulk update endpoint is called
- THEN the request SHALL be validated
- AND a task SHALL be submitted to the task queue
- AND a task ID SHALL be returned immediately to the client
- AND the actual processing SHALL happen asynchronously

#### Scenario: Task status check for bulk operation

- GIVEN a task ID from a previously submitted bulk operation
- WHEN the client calls the task status endpoint
- THEN the current status (pending, running, completed, failed) SHALL be returned
- AND if completed, the number of affected products SHALL be included

#### Scenario: Bulk operation completes successfully

- GIVEN a bulk update task processing 100 products
- WHEN all products are updated successfully
- THEN the task status SHALL be completed
- AND the result SHALL include count of updated products

#### Scenario: Bulk operation handles partial failures

- GIVEN a bulk update task processing 100 products
- WHEN 95 products update successfully and 5 fail due to validation errors
- THEN the task status SHALL be completed with errors
- AND the result SHALL include success count and failure details

### Requirement: Parallel Product Data Fetching

The system SHALL provide methods to fetch product data in parallel, retrieving product details, variants, and images concurrently.

#### Scenario: Fetch product with variants and images in parallel

- GIVEN a product ID that exists with associated variants and images
- WHEN the parallel fetch method is called
- THEN product, variants, and images SHALL be fetched concurrently
- AND the combined data SHALL be returned in a single response

#### Scenario: Parallel fetch when product has no variants

- GIVEN a product ID that exists but has no variants
- WHEN the parallel fetch method is called
- THEN product SHALL be returned with empty variants array
- AND images SHALL be fetched if any exist

#### Scenario: Parallel fetch handles missing product

- GIVEN a product ID that does not exist
- WHEN the parallel fetch method is called
- THEN the method SHALL return nil product with error
- AND variants and images SHALL not be attempted

### Requirement: Service Layer Integration with Async Tasks

The product service layer SHALL integrate with the task queue for bulk operations and SHALL provide methods to check task status.

#### Scenario: Service submits bulk operation and returns task ID

- GIVEN a BulkUpdateProductsRequest with products to update
- WHEN the service processes the request
- THEN it SHALL submit a task to the queue
- AND return the task ID to the handler

#### Scenario: Service retrieves task status

- GIVEN a task ID
- WHEN the service GetTaskStatus method is called
- THEN it SHALL delegate to the task queue's GetStatus
- AND return the task status to the handler

---

### Requirement: Variant Attribute Keys Validation

The variant attributes JSON SHALL only allow specific keys: `color`, `size`, `weight`. All other keys MUST be rejected with a 400 Bad Request response.

#### Scenario: Create variant with valid attribute keys

- GIVEN a CreateVariantRequest with attributes containing only valid keys: `color`, `size`, `weight`
- WHEN the client POSTs to `/api/v1/products/{product_id}/variants`
- THEN the variant is created successfully
- AND response status is 201 Created

#### Scenario: Create variant with invalid attribute keys

- GIVEN a CreateVariantRequest with attributes containing invalid keys (e.g., `material`, `brand`)
- WHEN the client POSTs to `/api/v1/products/{product_id}/variants`
- THEN the request is rejected
- AND response status is 400 Bad Request
- AND response includes error message listing invalid keys

#### Scenario: Create variant with mixed valid and invalid attribute keys

- GIVEN a CreateVariantRequest with attributes containing both valid (`color`) and invalid (`material`) keys
- WHEN the client POSTs to `/api/v1/products/{product_id}/variants`
- THEN the request is rejected
- AND response status is 400 Bad Request
- AND error message clearly indicates which keys are invalid

---

### Requirement: Reserved Field Exposure

The variant response SHALL expose the `reserved` field indicating the quantity reserved for pending orders.

#### Scenario: Get variant returns reserved quantity

- GIVEN a variant exists in the database with `reserved = 5`
- WHEN the client GETs `/api/v1/products/{product_id}/variants/{variant_id}`
- THEN response includes `"reserved": 5` in the variant data
- AND the value matches the database field

#### Scenario: List variants returns reserved for each

- GIVEN multiple variants exist with different reserved quantities
- WHEN the client GETs `/api/v1/products/{product_id}/variants`
- THEN each variant in the list includes its reserved quantity
- AND the values are accurate

---

### Requirement: Available Computed Field

The variant response SHALL include an `available` computed field calculated as `stock - reserved` at runtime.

#### Scenario: Available field calculated correctly

- GIVEN a variant with `stock = 100` and `reserved = 5`
- WHEN the client requests the variant
- THEN response includes `"available": 95`
- AND the value is computed, not stored in database

#### Scenario: Available is zero when reserved equals stock

- GIVEN a variant with `stock = 10` and `reserved = 10`
- WHEN the client requests the variant
- THEN response includes `"available": 0`

#### Scenario: Available is negative when reserved exceeds stock

- GIVEN a variant with `stock = 5` and `reserved = 10`
- WHEN the client requests the variant
- THEN response includes `"available": -5` (edge case handling)

#### Scenario: Available field not present in database

- GIVEN the database schema for product_variants
- WHEN inspecting the table structure
- THEN no `available` column exists
- AND the field is computed in the response DTO only

---

### Requirement: Complete Variant Response Structure

The variant response SHALL include all stock-related fields in a consistent order.

#### Scenario: Full variant response with all stock fields

- GIVEN a variant with `stock = 100`, `reserved = 5`, `price = 29.99`
- WHEN the client requests the variant
- THEN response includes:
  ```json
  {
    "product_id": 1,
    "sku": "CAMISETA-001-M",
    "price": 29.99,
    "stock": 100,
    "reserved": 5,
    "available": 95,
    "attributes": {
      "color": "azul",
      "size": "M",
      "weight": "0.5"
    }
  }
  ```
