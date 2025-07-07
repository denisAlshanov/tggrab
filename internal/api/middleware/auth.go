package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/denisAlshanov/stPlaner/internal/config"
	"github.com/denisAlshanov/stPlaner/internal/services/auth"
	"github.com/denisAlshanov/stPlaner/internal/utils"
)

// AuthMiddleware provides authentication for API endpoints
func AuthMiddleware(cfg *config.APIConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key first (backward compatibility)
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" && apiKey == cfg.APIKey {
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

// JWTAuthMiddleware provides JWT-based authentication
func JWTAuthMiddleware(jwtService *auth.JWTService, sessionService *auth.SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.JSON(401, gin.H{
				"error":      "MISSING_TOKEN",
				"message":    "Authorization token required",
				"request_id": c.GetString("request_id"),
				"timestamp":  time.Now().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Validate access token
		claims, err := jwtService.ValidateAccessToken(token)
		if err != nil {
			c.JSON(401, gin.H{
				"error":      "INVALID_TOKEN",
				"message":    "Invalid or expired token",
				"request_id": c.GetString("request_id"),
				"timestamp":  time.Now().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Check if token is blacklisted
		blacklisted, err := sessionService.IsTokenBlacklisted(c, claims.ID)
		if err != nil {
			utils.LogError(c, "Failed to check token blacklist", err)
			c.JSON(500, gin.H{
				"error":      "TOKEN_CHECK_ERROR",
				"message":    "Failed to verify token status",
				"request_id": c.GetString("request_id"),
				"timestamp":  time.Now().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		if blacklisted {
			c.JSON(401, gin.H{
				"error":      "TOKEN_REVOKED",
				"message":    "Token has been revoked",
				"request_id": c.GetString("request_id"),
				"timestamp":  time.Now().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Update session activity if session ID is available
		if claims.SessionID != "" {
			sessionID, err := uuid.Parse(claims.SessionID)
			if err == nil {
				// Update session activity (don't fail if this fails)
				if err := sessionService.UpdateSessionActivity(c, sessionID); err != nil {
					utils.LogError(c, "Failed to update session activity", err)
				}
			}
		}

		// Store user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_roles", claims.Roles)
		c.Set("user_permissions", claims.Permissions)
		c.Set("session_id", claims.SessionID)
		c.Set("token_jti", claims.ID)

		c.Next()
	}
}

// OptionalJWTAuthMiddleware provides optional JWT authentication (doesn't fail if no token)
func OptionalJWTAuthMiddleware(jwtService *auth.JWTService, sessionService *auth.SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			// No token provided, continue without authentication
			c.Next()
			return
		}

		// Validate access token
		claims, err := jwtService.ValidateAccessToken(token)
		if err != nil {
			// Invalid token, continue without authentication
			c.Next()
			return
		}

		// Check if token is blacklisted
		blacklisted, err := sessionService.IsTokenBlacklisted(c, claims.ID)
		if err != nil || blacklisted {
			// Blacklisted or error checking, continue without authentication
			c.Next()
			return
		}

		// Valid token, store user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_roles", claims.Roles)
		c.Set("user_permissions", claims.Permissions)
		c.Set("session_id", claims.SessionID)
		c.Set("token_jti", claims.ID)

		c.Next()
	}
}

// RoleMiddleware checks if the user has any of the required roles
func RoleMiddleware(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("user_roles")
		if !exists {
			c.JSON(401, gin.H{
				"error":      "AUTHENTICATION_REQUIRED",
				"message":    "Authentication required for this endpoint",
				"request_id": c.GetString("request_id"),
				"timestamp":  time.Now().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		roles, ok := userRoles.([]string)
		if !ok {
			c.JSON(403, gin.H{
				"error":      "INVALID_ROLES",
				"message":    "Invalid user roles",
				"request_id": c.GetString("request_id"),
				"timestamp":  time.Now().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, userRole := range roles {
			for _, requiredRole := range requiredRoles {
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			c.JSON(403, gin.H{
				"error":      "INSUFFICIENT_PERMISSIONS",
				"message":    "Insufficient permissions for this endpoint",
				"request_id": c.GetString("request_id"),
				"timestamp":  time.Now().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// extractToken extracts the JWT token from the Authorization header
func extractToken(c *gin.Context) string {
	bearerToken := c.GetHeader("Authorization")
	if bearerToken == "" {
		return ""
	}

	// Remove "Bearer " prefix
	if len(bearerToken) > 7 && strings.ToLower(bearerToken[:7]) == "bearer " {
		return bearerToken[7:]
	}

	return bearerToken
}
