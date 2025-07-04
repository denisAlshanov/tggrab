package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/tggrab/tggrab/internal/config"
	"github.com/tggrab/tggrab/internal/utils"
)

func AuthMiddleware(cfg *config.APIConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key first
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" && apiKey == cfg.APIKey {
			c.Next()
			return
		}

		// If JWT secret is configured, check for JWT token
		if cfg.JWTSecret != "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(401, gin.H{
					"error":      utils.NewUnauthorizedError(),
					"request_id": c.GetString("request_id"),
					"timestamp":  time.Now().Format(time.RFC3339),
				})
				c.Abort()
				return
			}

			// Extract token from Bearer scheme
			tokenString := ""
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				c.JSON(401, gin.H{
					"error":      utils.NewUnauthorizedError(),
					"request_id": c.GetString("request_id"),
					"timestamp":  time.Now().Format(time.RFC3339),
				})
				c.Abort()
				return
			}

			// Parse and validate JWT token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Validate signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(cfg.JWTSecret), nil
			})

			if err != nil || !token.Valid {
				c.JSON(401, gin.H{
					"error":      utils.NewUnauthorizedError(),
					"request_id": c.GetString("request_id"),
					"timestamp":  time.Now().Format(time.RFC3339),
				})
				c.Abort()
				return
			}

			// Extract claims if needed
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				// Store user info in context if available
				if userID, ok := claims["user_id"].(string); ok {
					c.Set("user_id", userID)
				}
			}

			c.Next()
			return
		}

		// No valid authentication found
		c.JSON(401, gin.H{
			"error":      utils.NewUnauthorizedError(),
			"request_id": c.GetString("request_id"),
			"timestamp":  time.Now().Format(time.RFC3339),
		})
		c.Abort()
	}
}
