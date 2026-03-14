package middleware

import (
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type CSRFConfig struct {
	CookieName   string
	HeaderName   string
	CookieExpiry time.Duration
}

func DefaultCSRFConfig() CSRFConfig {
	return CSRFConfig{
		CookieName:   "csrf_token",
		HeaderName:   "X-CSRF-Token",
		CookieExpiry: 24 * time.Hour,
	}
}

func CSRFMiddleware(config CSRFConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		headerToken := c.GetHeader(config.HeaderName)
		if headerToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "csrf token required",
			})
			return
		}

		cookieToken, err := c.Cookie(config.CookieName)
		if err != nil || cookieToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "csrf token required",
			})
			return
		}

		if subtle.ConstantTimeCompare([]byte(headerToken), []byte(cookieToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "invalid csrf token",
			})
			return
		}

		c.Next()
	}
}
