# Role-Based Access Control (RBAC) Specification

## Purpose

This specification defines role-based access control for the Bey API, ensuring only authorized users can access specific resources.

## Requirements

### Requirement: Role Definitions

The system MUST define user roles with specific permissions.

#### Scenario: Admin role has full access

- GIVEN a user with role "admin"
- WHEN the user attempts any operation
- THEN the user MUST have access to all resources

#### Scenario: Customer role has limited access

- GIVEN a user with role "customer"
- WHEN the user attempts admin-only operations
- THEN the user MUST be denied access

### Requirement: Admin-Only Routes

Certain routes MUST be restricted to admin users.

#### Scenario: Admin can access admin routes

- GIVEN a user with role "admin"
- WHEN the user accesses POST /api/v1/products
- THEN the request MUST be allowed

#### Scenario: Customer cannot access admin routes

- GIVEN a user with role "customer"
- WHEN the user accesses POST /api/v1/products
- THEN the request MUST be denied with 403 Forbidden

### Requirement: Permission Matrix

The system MUST enforce the following permission matrix:

| Resource | Admin | Customer |
|----------|-------|----------|
| users:read | ✅ | ❌ |
| users:write | ✅ | ❌ |
| orders:read | ✅ | ✅ (own only) |
| orders:write | ✅ | ❌ |
| products:read | ✅ | ✅ |
| products:write | ✅ | ❌ |
| inventory:read | ✅ | ❌ |
| inventory:write | ✅ | ❌ |

### Requirement: Unauthenticated Requests

Unauthenticated requests to protected routes MUST be rejected.

#### Scenario: Unauthenticated request is denied

- GIVEN no authentication token
- WHEN the user accesses a protected route
- THEN the request MUST be denied with 401 Unauthorized
