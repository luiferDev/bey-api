package users

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDBForUsersHandler(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func setupTestRouterWithUsers(t *testing.T) (*gin.Engine, *UserHandler) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForUsersHandler(t)
	handler := NewUserHandler(db)

	router := gin.New()
	return router, handler
}

func TestGetUsers_Success(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.GET("/api/v1/users", handler.List)

	req, err := http.NewRequest("GET", "/api/v1/users", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["success"] == nil {
		t.Error("Expected success field in response")
	}
}

func TestGetUsers_EmptyList(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.GET("/api/v1/users", handler.List)

	req, err := http.NewRequest("GET", "/api/v1/users", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCreateUser_InvalidBody(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.POST("/api/v1/users", handler.Register)

	req, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid body, got %d", w.Code)
	}
}

func TestCreateUser_MissingFields(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.POST("/api/v1/users", handler.Register)

	body := `{"email":"test@example.com"}`
	req, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing fields, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateUser_Success(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.POST("/api/v1/users", handler.Register)

	body := `{"email":"test@example.com","password":"password123","name":"John Doe"}`
	req, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.POST("/api/v1/users", handler.Register)

	body := `{"email":"test@example.com","password":"password123","first_name":"John","last_name":"Doe"}`
	req1, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated && w1.Code != http.StatusBadRequest {
		t.Errorf("First request: Expected status 201 or 400, got %d", w1.Code)
	}

	req2, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for duplicate email, got %d", w2.Code)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForUsersHandler(t)
	handler := NewUserHandler(db)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint(999))
		c.Set("user_role", "admin")
		c.Next()
	})
	router.GET("/api/v1/users/:id", handler.GetByID)

	req, _ := http.NewRequest("GET", "/api/v1/users/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestUpdateUser_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDBForUsersHandler(t)
	handler := NewUserHandler(db)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uint(999))
		c.Set("user_role", "admin")
		c.Next()
	})
	router.PUT("/api/v1/users/:id", handler.Update)

	body := `{"first_name":"John"}`
	req, _ := http.NewRequest("PUT", "/api/v1/users/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.DELETE("/api/v1/users/:id", handler.Delete)

	req, _ := http.NewRequest("DELETE", "/api/v1/users/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 (soft delete), got %d", w.Code)
	}
}

func TestRegisterAdmin_Success(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.POST("/api/v1/users/register-admin", handler.RegisterAdmin)

	body := `{"email":"admin@example.com","password":"password123","name":"Admin","surname":"User"}`
	req, _ := http.NewRequest("POST", "/api/v1/users/register-admin", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data field in response")
	}

	if data["role"] != "admin" {
		t.Errorf("Expected role 'admin', got '%v'", data["role"])
	}
}

func TestRegisterAdmin_DuplicateEmail(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.POST("/api/v1/users/register-admin", handler.RegisterAdmin)

	body := `{"email":"dupadmin@example.com","password":"password123","name":"Admin","surname":"User"}`

	req1, _ := http.NewRequest("POST", "/api/v1/users/register-admin", bytes.NewBufferString(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Errorf("First request: Expected status 201, got %d", w1.Code)
	}

	req2, _ := http.NewRequest("POST", "/api/v1/users/register-admin", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for duplicate email, got %d", w2.Code)
	}
}

func TestRegisterAdmin_MissingFields(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.POST("/api/v1/users/register-admin", handler.RegisterAdmin)

	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing email",
			body: `{"password":"password123","name":"Admin"}`,
		},
		{
			name: "missing password",
			body: `{"email":"admin@example.com","name":"Admin"}`,
		},
		{
			name: "missing name",
			body: `{"email":"admin@example.com","password":"password123"}`,
		},
		{
			name: "empty body",
			body: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/api/v1/users/register-admin", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
			}
		})
	}
}

func createUserForAvatarTest(t *testing.T, router *gin.Engine, handler *UserHandler, email string) uint {
	t.Helper()
	body := `{"email":"` + email + `","password":"password123","name":"Avatar","surname":"User"}`
	req, _ := http.NewRequest("POST", "/api/v1/users/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create user for avatar test, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal create response: %v", err)
	}

	data := resp["data"].(map[string]interface{})
	return uint(data["id"].(float64))
}

func TestUpdateAvatar_Success(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.POST("/api/v1/users/register", handler.Register)

	userID := createUserForAvatarTest(t, router, handler, "avatar@example.com")

	avatarRouter := gin.New()
	avatarRouter.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("user_role", "admin")
		c.Next()
	})
	avatarRouter.PUT("/api/v1/users/:id/avatar", handler.UpdateAvatar)

	body := `{"avatar_url":"https://example.com/avatar.jpg"}`
	req, _ := http.NewRequest("PUT", "/api/v1/users/1/avatar", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	avatarRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAvatar_InvalidURL(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.POST("/api/v1/users/register", handler.Register)

	userID := createUserForAvatarTest(t, router, handler, "invavatar@example.com")

	avatarRouter := gin.New()
	avatarRouter.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("user_role", "admin")
		c.Next()
	})
	avatarRouter.PUT("/api/v1/users/:id/avatar", handler.UpdateAvatar)

	body := `{"avatar_url":"not-a-url"}`
	req, _ := http.NewRequest("PUT", "/api/v1/users/1/avatar", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	avatarRouter.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid URL, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAvatar_Unauthorized(t *testing.T) {
	_, handler := setupTestRouterWithUsers(t)

	avatarRouter := gin.New()
	avatarRouter.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("user_role", "customer")
		c.Next()
	})
	avatarRouter.PUT("/api/v1/users/:id/avatar", handler.UpdateAvatar)

	body := `{"avatar_url":"https://example.com/avatar.jpg"}`
	req, _ := http.NewRequest("PUT", "/api/v1/users/999/avatar", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	avatarRouter.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for unauthorized, got %d. Body: %s", w.Code, w.Body.String())
	}
}
