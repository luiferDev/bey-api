# Specification: User Creation and Template Method

## Domain: users/

### 1. User Creation (Regular)

#### 1.1 Create Regular User - Happy Path

**Scenario**: Successfully create a regular user with valid data

**Given** the API receives a POST request to `/api/v1/users`  
**And** the request body contains valid user data: email, password (min 8 chars), first_name, last_name  
**And** no user with the provided email exists in the database  
**When** the server processes the request  
**Then** the response status MUST be `201 Created`  
**And** the response body MUST contain the created user with id, email, first_name, last_name, role="customer", active=true  
**And** the user password MUST be hashed using bcrypt  
**And** the user MUST be persisted to the database

#### 1.2 Create Regular User - Email Already Exists

**Scenario**: Attempt to create user with duplicate email

**Given** the API receives a POST request to `/api/v1/users`  
**And** the request body contains an email that already exists in the database  
**When** the server processes the request  
**Then** the response status MUST be `400 Bad Request`  
**And** the response body MUST contain error message "email already exists"

#### 1.3 Create Regular User - Validation Errors

**Scenario**: Create user with invalid input data

**Given** the API receives a POST request to `/api/v1/users`  
**And** the request body fails validation (missing fields, invalid email, short password)  
**When** the server processes the request  
**Then** the response status MUST be `400 Bad Request`  
**And** the response body MUST contain validation error details

#### 1.4 Create Regular User - Password Hashing Failure

**Scenario**: bcrypt fails to hash password

**Given** the API receives a POST request to `/api/v1/users` with valid data  
**And** bcrypt hashing fails internally  
**When** the server processes the request  
**Then** the response status MUST be `500 Internal Server Error`  
**And** the response body MUST contain error message "failed to hash password"

#### 1.5 Create Regular User - Database Persistence Failure

**Scenario**: Database fails to save user

**Given** the API receives a POST request to `/api/v1/users` with valid data  
**And** the database fails to persist the user  
**When** the server processes the request  
**Then** the response status MUST be `500 Internal Server Error`  
**And** the response body MUST contain error message "failed to create user"

---

### 2. Template Method - User Creator Interface

#### 2.1 UserCreator Interface Definition

**Scenario**: Define the Template Method interface

**Given** a need to support multiple user creation strategies  
**When** implementing the Template Method pattern  
**Then** there MUST exist a `UserCreator` interface with the following methods:
- `ValidateInput(req interface{}) error` - Validates input data
- `HashPassword(password string) (string, error)` - Hashes password
- `SetRole(user *User)` - Sets user role (abstract method)
- `SaveUser(user *User, repo *UserRepository) error` - Persists to database
- `Create(req interface{}) (*User, error)` - Template method orchestrating the flow

#### 2.2 RegularUserCreator Implementation

**Scenario**: Create a regular user using RegularUserCreator

**Given** a RegularUserCreator instance  
**When** `SetRole(user)` is called  
**Then** the user role MUST be set to "customer"

#### 2.3 AdminUserCreator Implementation

**Scenario**: Create an admin user using AdminUserCreator

**Given** an AdminUserCreator instance  
**When** `SetRole(user)` is called  
**Then** the user role MUST be set to "admin"

#### 2.4 Template Method Execution Flow

**Scenario**: User creation through Template Method

**Given** a UserCreator implementation (Regular or Admin)  
**When** `Create(req)` is called  
**Then** the execution MUST follow this order:
1. Call `ValidateInput(req)` - fail fast on validation error
2. Extract password and call `HashPassword()` - fail on hashing error
3. Create User struct with common fields
4. Call `SetRole(user)` to set role-specific field
5. Call `SaveUser(user, repo)` to persist
6. Return created user or error

---

## Technical Implementation Notes

### File Structure Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/modules/users/creator.go` | Create | Template Method interface and implementations |
| `internal/modules/admin/handler.go` | Create | Admin-specific handlers |
| `internal/modules/admin/routes.go` | Create | Admin route registration |
| `internal/modules/admin/model.go` | Create | Admin DTOs if needed |
| `internal/modules/users/handler.go` | Modify | Refactor Create to use UserCreator |
| `cmd/api/main.go` | Modify | Register admin routes, add seed call |

### Template Method Class Diagram

```
<<interface>>
UserCreator
─────────────
+ ValidateInput(req) error
+ HashPassword(password) string
+ SetRole(user *User)
+ SaveUser(user *User, repo *UserRepository) error
+ Create(req interface{}) (*User, error)

RegularUserCreator ──► UserCreator
AdminUserCreator ──► UserCreator
```
