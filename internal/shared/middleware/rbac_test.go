package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRBACTest(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestRequireRole_AdminAllowed(t *testing.T) {
	r := setupRBACTest(t)

	r.GET("/admin", func(c *gin.Context) {
		c.Set("user_role", "admin")
		c.Next()
	}, RequireRole(RoleAdmin), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
}

func TestRequireRole_CustomerDenied(t *testing.T) {
	r := setupRBACTest(t)

	r.GET("/admin", func(c *gin.Context) {
		c.Set("user_role", "customer")
		c.Next()
	}, RequireRole(RoleAdmin), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("got status %d; want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireRole_NoAuth(t *testing.T) {
	r := setupRBACTest(t)

	r.GET("/admin", RequireRole(RoleAdmin), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("got status %d; want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireRole_MultipleRoles(t *testing.T) {
	r := setupRBACTest(t)

	r.GET("/admin-or-customer", func(c *gin.Context) {
		c.Set("user_role", "customer")
		c.Next()
	}, RequireRole(RoleAdmin, RoleCustomer), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "access granted"})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin-or-customer", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
}

func TestHasPermission_AdminPermissions(t *testing.T) {
	tests := []struct {
		name       string
		permission string
		want       bool
	}{
		{"users:read", PermissionUsersRead, true},
		{"users:write", PermissionUsersWrite, true},
		{"orders:read", PermissionOrdersRead, true},
		{"orders:write", PermissionOrdersWrite, true},
		{"products:read", PermissionProductsRead, true},
		{"products:write", PermissionProductsWrite, true},
		{"inventory:read", PermissionInventoryRead, true},
		{"inventory:write", PermissionInventoryWrite, true},
		{"unknown permission", "unknown:permission", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasPermission(RoleAdmin, tt.permission)
			if got != tt.want {
				t.Errorf("HasPermission(admin, %s) = %v; want %v", tt.permission, got, tt.want)
			}
		})
	}
}

func TestHasPermission_CustomerPermissions(t *testing.T) {
	tests := []struct {
		name       string
		permission string
		want       bool
	}{
		{"orders:read", PermissionOrdersRead, true},
		{"products:read", PermissionProductsRead, true},
		{"users:read", PermissionUsersRead, false},
		{"users:write", PermissionUsersWrite, false},
		{"orders:write", PermissionOrdersWrite, false},
		{"products:write", PermissionProductsWrite, false},
		{"inventory:read", PermissionInventoryRead, false},
		{"inventory:write", PermissionInventoryWrite, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasPermission(RoleCustomer, tt.permission)
			if got != tt.want {
				t.Errorf("HasPermission(customer, %s) = %v; want %v", tt.permission, got, tt.want)
			}
		})
	}
}

func TestHasPermission_UnknownRole(t *testing.T) {
	got := HasPermission(Role("unknown"), PermissionUsersRead)
	if got != false {
		t.Errorf("HasPermission(unknown, users:read) = %v; want false", got)
	}
}

func TestRequirePermission_Success(t *testing.T) {
	r := setupRBACTest(t)

	r.GET("/orders", func(c *gin.Context) {
		c.Set("user_role", "customer")
		c.Next()
	}, RequirePermission(PermissionOrdersRead), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "access granted"})
	})

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
}

func TestRequirePermission_Denied(t *testing.T) {
	r := setupRBACTest(t)

	r.GET("/users", func(c *gin.Context) {
		c.Set("user_role", "customer")
		c.Next()
	}, RequirePermission(PermissionUsersWrite), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "access granted"})
	})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("got status %d; want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequirePermission_MultiplePermissions(t *testing.T) {
	r := setupRBACTest(t)

	r.GET("/products", func(c *gin.Context) {
		c.Set("user_role", "admin")
		c.Next()
	}, RequirePermission(PermissionProductsRead, PermissionProductsWrite), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "access granted"})
	})

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
}

func TestRequirePermission_MissingOne(t *testing.T) {
	r := setupRBACTest(t)

	r.GET("/products", func(c *gin.Context) {
		c.Set("user_role", "customer")
		c.Next()
	}, RequirePermission(PermissionProductsRead, PermissionProductsWrite), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "access granted"})
	})

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("got status %d; want %d", w.Code, http.StatusForbidden)
	}
}
