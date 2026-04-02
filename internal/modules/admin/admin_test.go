package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bey/internal/modules/users"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(&users.User{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func setupAdminTestRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestAdminCreateUser_Success(t *testing.T) {
	db := setupAdminTestDB(t)
	r := setupAdminTestRouter(t, db)

	handler := NewAdminHandler(db)

	r.POST("/api/v1/admin/users", func(c *gin.Context) {
		c.Set("user_role", "admin")
		c.Next()
	}, handler.CreateUser)

	body := CreateUserRequest{
		Email:    "newadmin@test.com",
		Password: "Password123",
		Name:     "New Admin",
		Role:     "admin",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Status = %d; want %d", w.Code, http.StatusCreated)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Failed to extract data from response")
	}
	if data["email"] != "newadmin@test.com" {
		t.Errorf("Email = %v; want newadmin@test.com", data["email"])
	}
	if data["role"] != "admin" {
		t.Errorf("Role = %v; want admin", data["role"])
	}
}

func TestAdminCreateUser_Unauthorized(t *testing.T) {
	db := setupAdminTestDB(t)
	r := setupAdminTestRouter(t, db)

	handler := NewAdminHandler(db)

	authMiddleware := func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization token"})
			return
		}
		c.Next()
	}

	r.POST("/api/v1/admin/users", authMiddleware, handler.CreateUser)

	body := CreateUserRequest{
		Email:    "test@test.com",
		Password: "Password123",
		Name:     "Test User",
		Role:     "admin",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %d; want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAdminCreateUser_Forbidden(t *testing.T) {
	db := setupAdminTestDB(t)
	r := setupAdminTestRouter(t, db)

	handler := NewAdminHandler(db)

	authMiddleware := func(c *gin.Context) {
		c.Set("user_role", "customer")
		c.Next()
	}

	adminMiddleware := func(c *gin.Context) {
		role, _ := c.Get("user_role")
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}

	r.POST("/api/v1/admin/users", authMiddleware, adminMiddleware, handler.CreateUser)

	body := CreateUserRequest{
		Email:    "test@test.com",
		Password: "Password123",
		Name:     "Test User",
		Role:     "admin",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Status = %d; want %d", w.Code, http.StatusForbidden)
	}
}
