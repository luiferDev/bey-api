# Specification: Admin Endpoint and Template Method Refactoring

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

## Domain: admin/

### 3. Admin User Creation Endpoint

#### 3.1 Create Admin User - Happy Path

**Scenario**: Admin successfully creates another admin user

**Given** the API receives a POST request to `/api/v1/admin/users`  
**And** the request includes valid admin authentication (JWT with role="admin")  
**And** the request body contains valid user data: email, password (min 8 chars), first_name, last_name  
**And** no user with the provided email exists in the database  
**When** the server processes the request  
**Then** the response status MUST be `201 Created`  
**And** the response body MUST contain the created user with role="admin"  
**And** the user MUST be persisted to the database

#### 3.2 Create Admin User - Unauthorized Request

**Scenario**: Unauthenticated user attempts to create admin

**Given** the API receives a POST request to `/api/v1/admin/users`  
**And** no authentication token is provided  
**When** the server processes the request  
**Then** the response status MUST be `401 Unauthorized`  
**And** the response body MUST contain error message "unauthorized"

#### 3.3 Create Admin User - Forbidden (Non-Admin)

**Scenario**: Regular user attempts to create admin

**Given** the API receives a POST request to `/api/v1/admin/users`  
**And** the request includes authentication token with role="customer"  
**When** the server processes the request through RequireRole middleware  
**Then** the response status MUST be `403 Forbidden`  
**And** the response body MUST contain error message "forbidden - insufficient permissions"

#### 3.4 Create Admin User - Duplicate Email

**Scenario**: Admin attempts to create user with existing email

**Given** the API receives a POST request to `/api/v1/admin/users`  
**And** the request includes valid admin authentication  
**And** the request body contains an email that already exists in the database  
**When** the server processes the request  
**Then** the response status MUST be `400 Bad Request`  
**And** the response body MUST contain error message "email already exists"

#### 3.5 Create Admin User - Validation Errors

**Scenario**: Admin submits invalid user data

**Given** the API receives a POST request to `/api/v1/admin/users`  
**And** the request includes valid admin authentication  
**And** the request body fails validation (missing fields, invalid email, short password)  
**When** the server processes the request  
**Then** the response status MUST be `400 Bad Request`  
**And** the response body MUST contain validation error details

#### 3.6 Create Admin User - Password Hashing Failure

**Scenario**: bcrypt fails during admin creation

**Given** the API receives a POST request to `/api/v1/admin/users`  
**And** the request includes valid admin authentication  
**And** the request body contains valid user data  
**And** bcrypt hashing fails  
**When** the server processes the request  
**Then** the response status MUST be `500 Internal Server Error`  
**And** the response body MUST contain error message "failed to hash password"

#### 3.7 Create Admin User - Database Failure

**Scenario**: Database fails during admin creation

**Given** the API receives a POST request to `/api/v1/admin/users`  
**And** the request includes valid admin authentication  
**And** the request body contains valid user data  
**And** the database fails to persist the user  
**When** the server processes the request  
**Then** the response status MUST be `500 Internal Server Error`  
**And** the response body MUST contain error message "failed to create user"

---

### 4. RBAC Middleware Integration

#### 4.1 RequireRole Middleware - Admin Access

**Scenario**: Admin user passes RequireRole middleware

**Given** a request with context containing user_role="admin"  
**When** passed through `RequireRole(RoleAdmin)` middleware  
**Then** the request MUST proceed to the handler (c.Next() called)

#### 4.2 RequireRole Middleware - Customer Denied

**Scenario**: Customer user blocked by RequireRole middleware

**Given** a request with context containing user_role="customer"  
**When** passed through `RequireRole(RoleAdmin)` middleware  
**Then** the response status MUST be `403 Forbidden`  
**And** the request MUST be aborted

#### 4.3 RequireRole Middleware - No Role

**Scenario**: Unauthenticated request blocked by RequireRole

**Given** a request without user_role in context  
**When** passed through `RequireRole(RoleAdmin)` middleware  
**Then** the response status MUST be `401 Unauthorized`  
**And** the request MUST be aborted

---

### 5. Database Seed - Admin User

#### 5.1 Admin User Seed - Happy Path

**Scenario**: Database seeded with default admin user

**Given** the application starts with database initialization  
**When** the seed script executes  
**Then** there MUST exist a user with email="admin@bey.com"  
**And** the user role MUST be "admin"  
**And** the user password MUST be hashed with bcrypt  
**And** the user active status MUST be true

#### 5.2 Admin User Seed - Idempotency

**Scenario**: Running seed multiple times should not duplicate admin

**Given** the database already contains admin@bey.com  
**When** the seed script is executed again  
**Then** there MUST still be exactly one user with email="admin@bey.com"  
**Or** the seed MUST use "INSERT ... ON CONFLICT DO NOTHING" or similar

#### 5.3 Admin Seed Credentials

**Scenario**: Define admin seed credentials

**Given** the need to seed an admin user  
**Then** the credentials MUST be configurable via config.yaml:
- `admin.email`: default "admin@bey.com"
- `admin.password`: default "admin123" (should be changed on first login)
- `admin.first_name`: default "Admin"
- `admin.last_name`: default "User"

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

### Route Registration

```go
// Main router setup
admin := rg.Group("/admin")
admin.Use(authMiddleware)
admin.Use(RequireRole(RoleAdmin))
{
    admin.POST("/users", adminHandler.CreateUser)
}
```

### Configuration Addition

```yaml
admin:
  email: "admin@bey.com"
  password: "admin123"
  first_name: "Admin"
  last_name: "User"
```
