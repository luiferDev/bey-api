package middleware

import (
	"github.com/gin-gonic/gin"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleCustomer Role = "customer"
)

const (
	PermissionUsersRead      = "users:read"
	PermissionUsersWrite     = "users:write"
	PermissionOrdersRead     = "orders:read"
	PermissionOrdersWrite    = "orders:write"
	PermissionProductsRead   = "products:read"
	PermissionProductsWrite  = "products:write"
	PermissionInventoryRead  = "inventory:read"
	PermissionInventoryWrite = "inventory:write"
)

var RolePermissions = map[Role][]string{
	RoleAdmin: {
		PermissionUsersRead,
		PermissionUsersWrite,
		PermissionOrdersRead,
		PermissionOrdersWrite,
		PermissionProductsRead,
		PermissionProductsWrite,
		PermissionInventoryRead,
		PermissionInventoryWrite,
	},
	RoleCustomer: {
		PermissionOrdersRead,
		PermissionProductsRead,
	},
}

func HasPermission(role Role, permission string) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

func RequireRole(allowedRoles ...Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleStr, exists := c.Get("user_role")
		if !exists {
			c.JSON(401, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		userRoleStr, ok := roleStr.(string)
		if !ok {
			c.JSON(401, gin.H{"error": "invalid role format"})
			c.Abort()
			return
		}
		userRole := Role(userRoleStr)

		for _, role := range allowedRoles {
			if userRole == role {
				c.Next()
				return
			}
		}

		c.JSON(403, gin.H{"error": "forbidden - insufficient permissions"})
		c.Abort()
	}
}

func RequirePermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleStr, exists := c.Get("user_role")
		if !exists {
			c.JSON(401, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		userRoleStr, ok := roleStr.(string)
		if !ok {
			c.JSON(401, gin.H{"error": "invalid role format"})
			c.Abort()
			return
		}
		userRole := Role(userRoleStr)

		for _, perm := range permissions {
			if !HasPermission(userRole, perm) {
				c.JSON(403, gin.H{"error": "forbidden - missing permission: " + perm})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
