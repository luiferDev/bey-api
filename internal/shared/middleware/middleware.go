package middleware

import (
	"github.com/gin-gonic/gin"

	"bey/internal/concurrency"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
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
