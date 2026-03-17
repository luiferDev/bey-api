# Tasks: change-products-variants-information

## Phase 1: Foundation

### 1.1 Create custom validator function for attributes validation
- **File**: `internal/modules/products/dto.go`
- **Action**: Add `validateAttributes` function that checks each key in the JSONMap is not empty and keys are valid (alphanumeric + underscore)
- **Implementation**: Create function `func validateAttributesfl(fl validator.FieldLevel) bool` that casts `fl.Field().Interface()` to `datatypes.JSONMap`, iterates keys, validates non-empty keys and valid key format

### 1.2 Register validator in dto.go init()
- **File**: `internal/modules/products/dto.go`
- **Action**: Add `init()` function that calls `binding.Validator.RegisterValidation("validAttributes", validateAttributesfl)`
- **Placement**: After imports, before type definitions

## Phase 2: Core Implementation

### 2.1 Add Reserved field to ProductVariantResponse
- **File**: `internal/modules/products/dto.go`
- **Action**: Add `Reserved int json:"reserved"` to `ProductVariantResponse` struct (line 86-95)
- **Position**: After `Stock` field

### 2.2 Add Available computed field to ProductVariantResponse
- **File**: `internal/modules/products/dto.go`
- **Action**: Add `Available int json:"available"` to `ProductVariantResponse` struct
- **Position**: After `Reserved` field

### 2.3 Update CreateProductVariantRequest binding
- **File**: `internal/modules/products/dto.go`
- **Action**: Change `binding:"required"` to `binding:"required,validAttributes"` on line 76

### 2.4 Update UpdateProductVariantRequest binding
- **File**: `internal/modules/products/dto.go`
- **Action**: Add `validAttributes` to the binding tag for `Attributes` field on line 83
- **Note**: Only apply when `Attributes` is not nil (Gin evaluates binding even for nil values)

## Phase 3: Integration

### 3.1 Update handler to calculate available before response
- **File**: `internal/modules/products/handler.go`
- **Action**: Find functions that return `ProductVariantResponse` and calculate `available = stock - reserved`, ensure result is >= 0
- **Functions to check**: `GetVariantsByProductID`, `GetVariantByID`, `CreateVariant`, `UpdateVariant`

### 3.2 Ensure reserved is included in response
- **File**: `internal/modules/products/handler.go`
- **Action**: Map the `Reserved` field from the model to the response DTO in all variant response construction points

## Phase 4: Testing

### 4.1 Unit tests for validator - valid keys
- **File**: `internal/modules/products/dto_validator_test.go` (new file)
- **Action**: Test validator accepts valid attribute maps like `{"color": "red", "size": "M"}`

### 4.2 Unit tests for validator - invalid keys
- **File**: `internal/modules/products/dto_validator_test.go`
- **Action**: Test validator rejects keys with special characters like `{"bad-key": "value", "key with spaces": "value"}`

### 4.3 Unit tests for validator - empty
- **File**: `internal/modules/products/dto_validator_test.go`
- **Action**: Test validator rejects empty keys like `{"": "value"}`

### 4.4 Integration test: POST with invalid key returns 400
- **File**: `internal/modules/products/handler_integration_test.go` (or existing test file)
- **Action**: POST to `/api/v1/products/:id/variants` with `{"attributes": {"invalid-key": "value"}}` and assert 400 response

### 4.5 Integration test: GET variant returns reserved and available
- **File**: `internal/modules/products/handler_integration_test.go`
- **Action**: GET variant and assert response contains `reserved` and `available` fields with correct values
