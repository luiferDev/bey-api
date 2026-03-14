package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupCSRFTest(t *testing.T) (*gin.Engine, CSRFConfig) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	config := DefaultCSRFConfig()
	return r, config
}

func TestCSRFMiddleware_GETAllowed(t *testing.T) {
	r, config := setupCSRFTest(t)

	r.GET("/protected", CSRFMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
}

func TestCSRFMiddleware_HeadAllowed(t *testing.T) {
	r, config := setupCSRFTest(t)

	r.HEAD("/protected", CSRFMiddleware(config), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodHead, "/protected", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
}

func TestCSRFMiddleware_OptionsAllowed(t *testing.T) {
	r, config := setupCSRFTest(t)

	r.OPTIONS("/protected", CSRFMiddleware(config), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/protected", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
}

func TestCSRFMiddleware_PostWithValidToken(t *testing.T) {
	r, config := setupCSRFTest(t)

	token := "test-csrf-token"

	r.POST("/protected", CSRFMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	req.Header.Set(config.HeaderName, token)
	req.AddCookie(&http.Cookie{Name: config.CookieName, Value: token})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
}

func TestCSRFMiddleware_PostWithInvalidToken(t *testing.T) {
	r, config := setupCSRFTest(t)

	r.POST("/protected", CSRFMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	req.Header.Set(config.HeaderName, "invalid-token")
	req.AddCookie(&http.Cookie{Name: config.CookieName, Value: "valid-token"})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("got status %d; want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddleware_MissingToken(t *testing.T) {
	r, config := setupCSRFTest(t)

	r.POST("/protected", CSRFMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("got status %d; want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddleware_MissingHeaderToken(t *testing.T) {
	r, config := setupCSRFTest(t)

	token := "valid-csrf-token"

	r.POST("/protected", CSRFMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: config.CookieName, Value: token})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("got status %d; want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddleware_MissingCookieToken(t *testing.T) {
	r, config := setupCSRFTest(t)

	token := "valid-csrf-token"

	r.POST("/protected", CSRFMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	req.Header.Set(config.HeaderName, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("got status %d; want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRFMiddleware_PutWithValidToken(t *testing.T) {
	r, config := setupCSRFTest(t)

	token := "test-csrf-token"

	r.PUT("/protected", CSRFMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodPut, "/protected", nil)
	req.Header.Set(config.HeaderName, token)
	req.AddCookie(&http.Cookie{Name: config.CookieName, Value: token})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
}

func TestCSRFMiddleware_DeleteWithValidToken(t *testing.T) {
	r, config := setupCSRFTest(t)

	token := "test-csrf-token"

	r.DELETE("/protected", CSRFMiddleware(config), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodDelete, "/protected", nil)
	req.Header.Set(config.HeaderName, token)
	req.AddCookie(&http.Cookie{Name: config.CookieName, Value: token})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d; want %d", w.Code, http.StatusOK)
	}
}

func TestDefaultCSRFConfig(t *testing.T) {
	config := DefaultCSRFConfig()

	if config.CookieName != "csrf_token" {
		t.Errorf("CookieName = %s; want csrf_token", config.CookieName)
	}
	if config.HeaderName != "X-CSRF-Token" {
		t.Errorf("HeaderName = %s; want X-CSRF-Token", config.HeaderName)
	}
	if config.CookieExpiry != 24*3600*1000000000 {
		t.Errorf("CookieExpiry = %v; want 24h", config.CookieExpiry)
	}
}
