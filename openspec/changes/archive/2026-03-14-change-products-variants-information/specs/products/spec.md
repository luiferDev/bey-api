# Products Variants Information - Delta Spec

## Purpose

This spec adds attribute validation, exposes the `reserved` field, and introduces a computed `available` field for product variants.

## Requirements

### Requirement: Attribute Keys Validation

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

---

## Acceptance Criteria

1. Creating a variant with invalid attribute keys returns 400 Bad Request with clear error message
2. The `reserved` field is exposed in all variant responses (single and list)
3. The `available` field is computed as `stock - reserved` at runtime
4. The `available` field is NOT stored in the database
5. Edge case where reserved > stock returns negative available (0 or negative)
6. Response DTO includes all three fields: `stock`, `reserved`, `available`
