package middleware

import (
	"log"
	"strings"

	"github.com/gin-gonic/gin"

	"bey/internal/concurrency"
)

var allowedOrigins []string

func InitCORS(origins []string) {
	if len(origins) == 0 {
		log.Println("[WARNING] CORS: No origins configured, allowing all origins (development mode)")
		allowedOrigins = []string{"*"}
	} else {
		allowedOrigins = origins
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		if isOriginAllowed(origin) {
			if origin != "" && origin != "*" {
				c.Header("Access-Control-Allow-Origin", origin)
			} else if origin == "*" {
				c.Header("Access-Control-Allow-Origin", "*")
			}
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// isOriginAllowed checks if the given origin is in the allowed list
func isOriginAllowed(origin string) bool {
	// If "*" is in allowed origins, allow all
	for _, o := range allowedOrigins {
		if o == "*" {
			return true
		}
		if strings.EqualFold(o, origin) {
			return true
		}
	}

	// Also check without protocol for development convenience
	for _, o := range allowedOrigins {
		// Remove protocol for comparison
		originWithoutProto := strings.TrimPrefix(strings.TrimPrefix(origin, "https://"), "http://")
		oWithoutProto := strings.TrimPrefix(strings.TrimPrefix(o, "https://"), "http://")
		if strings.EqualFold(originWithoutProto, oWithoutProto) {
			return true
		}
	}

	return false
}

func LoggerMiddleware() gin.HandlerFunc {
	return gin.Logger()
}

var rateLimiter *RateLimiter

func InitRateLimiter(enabled bool, requestsPerSecond, burstCapacity int, endpointLimits map[string]int) {
	rateLimiter = NewRateLimiter(concurrency.RateLimitConfig{
		Enabled:           enabled,
		RequestsPerSecond: requestsPerSecond,
		BurstCapacity:     burstCapacity,
		EndpointLimits:    endpointLimits,
	})
}
