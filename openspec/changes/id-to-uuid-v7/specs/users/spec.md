# Delta for Users Module — UUIDv7 Migration

## MODIFIED Requirements

### Requirement: User Primary Key Type

The system SHALL use UUIDv7 as the primary key for all user-related models instead of auto-incrementing integers.

All user models (`User`, `Address`) MUST use `uuid.UUID` as their primary key type in GORM models.

(Previously: Primary keys were `uint` with auto-increment)

#### Scenario: User created with UUIDv7 ID

- GIVEN a valid user registration request with email, password, first_name, and last_name
- WHEN the user service creates the user
- THEN the system SHALL generate a UUIDv7 ID automatically
- AND the user SHALL be persisted with the UUID primary key
- AND the response SHALL include the UUID string in the `id` field

#### Scenario: User found by UUID

- GIVEN a user exists with ID `01960c12-3456-7890-abcd-ef1234567890`
- WHEN the client GETs `/api/v1/users/01960c12-3456-7890-abcd-ef1234567890`
- THEN the response status is 200 OK
- AND the response body contains the user with matching UUID `id`

#### Scenario: User not found by valid UUID

- GIVEN no user exists with ID `01960c12-3456-7890-abcd-ef9999999999`
- WHEN the client GETs `/api/v1/users/01960c12-3456-7890-abcd-ef9999999999`
- THEN the response status is 404 Not Found

#### Scenario: User requested with invalid UUID format

- GIVEN a client sends a request with an invalid UUID string
- WHEN the client GETs `/api/v1/users/not-a-uuid`
- THEN the response status is 400 Bad Request
- AND the response body indicates invalid ID format
- AND the system MUST NOT return 500 Internal Server Error

#### Scenario: User requested with integer ID (legacy format)

- GIVEN a client sends a request with an integer ID
- WHEN the client GETs `/api/v1/users/123`
- THEN the response status is 400 Bad Request
- AND the response body indicates invalid ID format

### Requirement: User Foreign Key Types

All foreign key fields in user-related models MUST use `uuid.UUID` type instead of `uint`.

This includes:
- `Address.UserID` → `uuid.UUID`

(Previously: Foreign keys were `uint`)

#### Scenario: Address created with UUID user reference

- GIVEN a user exists with UUID `01960c12-0000-0000-0000-000000000001`
- WHEN an address is created for this user
- THEN the address's `UserID` is set to the UUID
- AND the relationship is queryable via GORM preloads

### Requirement: User DTO ID Fields

All user-related DTOs MUST use `string` type for ID fields instead of `uint`.

Response DTOs MUST convert `uuid.UUID` to string using `.String()` method.

(Previously: DTO ID fields were `uint`)

#### Scenario: User response contains string ID

- GIVEN a user with UUID `01960c12-3456-7890-abcd-ef1234567890`
- WHEN the user is returned via API response
- THEN the JSON contains `"id": "01960c12-3456-7890-abcd-ef1234567890"`

#### Scenario: Address response contains UUID user reference

- GIVEN an address with user UUID `01960c12-0000-0000-0000-000000000001`
- WHEN the address is returned via API response
- THEN the JSON contains `"user_id": "01960c12-0000-0000-0000-000000000001"`

### Requirement: User Repository Signatures

All user repository methods that accept or return IDs MUST use `uuid.UUID` instead of `uint`.

This includes:
- `FindByID(id uuid.UUID) (*User, error)`
- `FindByEmail(email string) (*User, error)` (unchanged — email lookup)
- `Delete(id uuid.UUID) error`
- All other methods accepting user or address IDs

(Previously: Repository methods accepted `uint` IDs)

### Requirement: User Handler ID Parsing

All user handlers MUST parse incoming ID parameters using `uuid.FromString()` instead of `strconv.ParseUint()`.

(Previously: Handlers used `strconv.ParseUint(c.Param("id"), 10, 32)`)

#### Scenario: User handler rejects malformed UUID

- GIVEN a request with param `id = "abc123"`
- WHEN the user handler attempts to parse the ID
- THEN `uuid.FromString()` returns an error
- AND the handler returns 400 Bad Request immediately

### Requirement: Template Method User Creator with UUID

The UserCreator interface and implementations (RegularUserCreator, AdminUserCreator) SHALL work with UUID-based user models.

(Previously: User models used `uint` primary keys)

#### Scenario: RegularUserCreator creates user with UUID

- GIVEN a RegularUserCreator instance
- WHEN `Create(req)` is called with valid data
- THEN the created user has a UUIDv7 ID
- AND the user role is set to "customer"

#### Scenario: AdminUserCreator creates user with UUID

- GIVEN an AdminUserCreator instance
- WHEN `Create(req)` is called with valid data
- THEN the created user has a UUIDv7 ID
- AND the user role is set to "admin"
