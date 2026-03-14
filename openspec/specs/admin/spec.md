# Specification: Admin Endpoints and RBAC

## Domain: admin/

### 1. Admin User Creation Endpoint

#### 1.1 Create Admin User - Happy Path

**Scenario**: Admin successfully creates another admin user

**Given** the API receives a POST request to `/api/v1/admin/users`  
**And** the request includes valid admin authentication (JWT with role="admin")  
**And** the request body contains valid user data: email, password (min 8 chars), first_name, last_name  
**And** no user with the provided email exists in the database  
**When** the server processes the request  
**Then** the response status MUST be `201 Created`  
**And** the response body MUST contain the created user with role="admin"  
**And** the user MUST be persisted to the database

#### 1.2 Create Admin User - Unauthorized Request

**Scenario**: Unauthenticated user attempts to create admin

**Given** the API receives a POST request to `/api/v1/admin/users`  
**And** no authentication token is provided  
**When** the server processes the request  
**Then** the response status MUST be `401 Unauthorized`  
**And** the response body MUST contain error message "unauthorized"

#### 1.3 Create Admin User - Forbidden (Non-Admin)

**Scenario**: Regular user attempts to create admin

**Given** the API receives a POST request to `/api/v1/admin/users`  
**And** the request includes authentication token with role="customer"  
**When** the server processes the request through RequireRole middleware  
**Then** the response status MUST be `403 Forbidden`  
**And** the response body MUST contain error message "forbidden - insufficient permissions"

#### 1.4 Create Admin User - Duplicate Email

**Scenario**: Admin attempts to create user with existing email

**Given** the API receives a POST request to `/api/v1/admin/users`  
**And** the request includes valid admin authentication  
**And** the request body contains an email that already exists in the database  
**When** the server processes the request  
**Then** the response status MUST be `400 Bad Request`  
**And** the response body MUST contain error message "email already exists"

#### 1.5 Create Admin User - Validation Errors

**Scenario**: Admin submits invalid user data

**Given** the API receives a POST request to `/api/v1/admin/users`  
**And** the request includes valid admin authentication  
**And** the request body fails validation (missing fields, invalid email, short password)  
**When** the server processes the request  
**Then** the response status MUST be `400 Bad Request`  
**And** the response body MUST contain validation error details

#### 1.6 Create Admin User - Password Hashing Failure

**Scenario**: bcrypt fails during admin creation

**Given** the API receives a POST request to `/api/v1/admin/users`  
**And** the request includes valid admin authentication  
**And** the request body contains valid user data  
**And** bcrypt hashing fails  
**When** the server processes the request  
**Then** the response status MUST be `500 Internal Server Error`  
**And** the response body MUST contain error message "failed to hash password"

#### 1.7 Create Admin User - Database Failure

**Scenario**: Database fails during admin creation

**Given** the API receives a POST request to `/api/v1/admin/users`  
**And** the request includes valid admin authentication  
**And** the request body contains valid user data  
**And** the database fails to persist the user  
**When** the server processes the request  
**Then** the response status MUST be `500 Internal Server Error`  
**And** the response body MUST contain error message "failed to create user"

---

### 2. RBAC Middleware Integration

#### 2.1 RequireRole Middleware - Admin Access

**Scenario**: Admin user passes RequireRole middleware

**Given** a request with context containing user_role="admin"  
**When** passed through `RequireRole(RoleAdmin)` middleware  
**Then** the request MUST proceed to the handler (c.Next() called)

#### 2.2 RequireRole Middleware - Customer Denied

**Scenario**: Customer user blocked by RequireRole middleware

**Given** a request with context containing user_role="customer"  
**When** passed through `RequireRole(RoleAdmin)` middleware  
**Then** the response status MUST be `403 Forbidden`  
**And** the request MUST be aborted

#### 2.3 RequireRole Middleware - No Role

**Scenario**: Unauthenticated request blocked by RequireRole

**Given** a request without user_role in context  
**When** passed through `RequireRole(RoleAdmin)` middleware  
**Then** the response status MUST be `401 Unauthorized`  
**And** the request MUST be aborted

---

### 3. Database Seed - Admin User

#### 3.1 Admin User Seed - Happy Path

**Scenario**: Database seeded with default admin user

**Given** the application starts with database initialization  
**When** the seed script executes  
**Then** there MUST exist a user with email="admin@bey.com"  
**And** the user role MUST be "admin"  
**And** the user password MUST be hashed with bcrypt  
**And** the user active status MUST be true

#### 3.2 Admin User Seed - Idempotency

**Scenario**: Running seed multiple times should not duplicate admin

**Given** the database already contains admin@bey.com  
**When** the seed script is executed again  
**Then** there MUST still be exactly one user with email="admin@bey.com"  
**Or** the seed MUST use "INSERT ... ON CONFLICT DO NOTHING" or similar

#### 3.3 Admin Seed Credentials

**Scenario**: Define admin seed credentials

**Given** the need to seed an admin user  
**Then** the credentials MUST be configurable via config.yaml:
- `admin.email`: default "admin@bey.com"
- `admin.password`: default "admin123" (should be changed on first login)
- `admin.first_name`: default "Admin"
- `admin.last_name`: default "User"

---

## Technical Implementation Notes

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
