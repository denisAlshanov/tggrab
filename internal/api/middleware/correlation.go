package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/tggrab/tggrab/internal/utils"
)

func CorrelationIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if correlation ID exists in header
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = utils.GenerateCorrelationID()
		}

		// Generate request ID
		requestID := utils.GenerateRequestID()

		// Store in context
		c.Set("correlation_id", correlationID)
		c.Set("request_id", requestID)

		// Add to response headers
		c.Header("X-Correlation-ID", correlationID)
		c.Header("X-Request-ID", requestID)

		// Create context with IDs for logging
		ctx := c.Request.Context()
		ctx = utils.WithCorrelationID(ctx, correlationID)
		ctx = utils.WithRequestID(ctx, requestID)
		c.Request = c.Request.WithContext(ctx)

		// Log request
		utils.LogInfo(ctx, "Incoming request", utils.Fields{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"ip":     c.ClientIP(),
		})

		c.Next()

		// Log response
		utils.LogInfo(ctx, "Request completed", utils.Fields{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"status": c.Writer.Status(),
		})
	}
}
