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

	router.POST("/api/v1/users", handler.Create)

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

	router.POST("/api/v1/users", handler.Create)

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

	router.POST("/api/v1/users", handler.Create)

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

	router.POST("/api/v1/users", handler.Create)

	body := `{"email":"test@example.com","password":"password123","first_name":"John","last_name":"Doe"}`
	req1, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	req2, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for duplicate email, got %d", w2.Code)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.GET("/api/v1/users/:id", handler.GetByID)

	req, _ := http.NewRequest("GET", "/api/v1/users/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 404 or 400, got %d", w.Code)
	}
}

func TestUpdateUser_NotFound(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.PUT("/api/v1/users/:id", handler.Update)

	body := `{"first_name":"John"}`
	req, _ := http.NewRequest("PUT", "/api/v1/users/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 404 or 400, got %d", w.Code)
	}
}

func TestDeleteUser_NotFound(t *testing.T) {
	router, handler := setupTestRouterWithUsers(t)

	router.DELETE("/api/v1/users/:id", handler.Delete)

	req, _ := http.NewRequest("DELETE", "/api/v1/users/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest && w.Code != http.StatusOK {
		t.Errorf("Expected status 404, 400 or 200, got %d", w.Code)
	}
}
