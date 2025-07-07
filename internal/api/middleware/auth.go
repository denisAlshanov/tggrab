package middleware

import (
	"fmt"
	"net/url"
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

// HybridAuthMiddleware provides both JWT and API key authentication
func HybridAuthMiddleware(cfg *config.APIConfig, jwtService *auth.JWTService, sessionService *auth.SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for API key first (for initial setup and admin operations)
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" && apiKey == cfg.APIKey {
			c.Set("auth_method", "api_key")
			c.Next()
			return
		}

		// Check for JWT token
		token := extractToken(c)
		if token != "" {
			// Validate access token
			claims, err := jwtService.ValidateAccessToken(token)
			if err == nil {
				// Check if token is blacklisted
				blacklisted, err := sessionService.IsTokenBlacklisted(c, claims.ID)
				if err == nil && !blacklisted {
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
					c.Set("auth_method", "jwt")
					c.Next()
					return
				}
			}
		}

		// No valid authentication found
		c.JSON(401, gin.H{
			"error":      "AUTHENTICATION_REQUIRED",
			"message":    "Valid JWT token or API key required",
			"request_id": c.GetString("request_id"),
			"timestamp":  time.Now().Format(time.RFC3339),
		})
		c.Abort()
	}
}

// JWTOnlyMiddleware provides JWT-only authentication and rejects API key attempts
func JWTOnlyMiddleware(jwtService *auth.JWTService, sessionService *auth.SessionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if API key is being attempted
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			c.JSON(401, gin.H{
				"error":      "API_KEY_NOT_ALLOWED",
				"message":    "API key authentication not allowed",
				"request_id": c.GetString("request_id"),
				"timestamp":  time.Now().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Extract JWT token
		token := extractToken(c)
		if token == "" {
			c.JSON(401, gin.H{
				"error":      "AUTHENTICATION_REQUIRED",
				"message":    "Authentication required",
				"request_id": c.GetString("request_id"),
				"timestamp":  time.Now().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Validate access token
		claims, err := jwtService.ValidateAccessToken(token)
		if err != nil {
			errorMessage := "Invalid authentication token"
			if strings.Contains(err.Error(), "expired") {
				errorMessage = "Token expired"
			}
			c.JSON(401, gin.H{
				"error":      "INVALID_TOKEN",
				"message":    errorMessage,
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

		// Validate session is still active
		if claims.SessionID != "" {
			sessionID, err := uuid.Parse(claims.SessionID)
			if err == nil {
				session, err := sessionService.ValidateSession(c, sessionID)
				if err == nil && session != nil {
					// Update session activity only if session is valid
					if err := sessionService.UpdateSessionActivity(c, session.ID); err != nil {
						utils.LogError(c, "Failed to update session activity", err)
					}
				} else {
					// Log the error but don't fail the request - the JWT is still valid
					utils.LogError(c, "Session validation failed", err)
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
		c.Set("auth_method", "jwt")

		c.Next()
	}
}

// CORSMiddleware provides Cross-Origin Resource Sharing support
func CORSMiddleware(cfg *config.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CORS if disabled
		if !cfg.Enabled {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		method := c.Request.Method

		// Log CORS configuration on first request (debug level)
		utils.LogDebug(c, "CORS Request", utils.Fields{
			"origin": origin,
			"method": method,
			"profile": cfg.Profile,
		})

		// Handle preflight OPTIONS request
		if method == "OPTIONS" {
			handlePreflightRequest(c, cfg, origin)
			return
		}

		// Handle actual request
		handleActualRequest(c, cfg, origin)
		c.Next()
	}
}

// handlePreflightRequest handles CORS preflight OPTIONS requests
func handlePreflightRequest(c *gin.Context, cfg *config.CORSConfig, origin string) {
	// Check if origin is allowed
	if !isOriginAllowed(origin, cfg.AllowedOrigins) {
		utils.LogDebug(c, "CORS preflight rejected", utils.Fields{
			"origin": origin,
			"reason": "origin not allowed",
		})
		c.AbortWithStatus(403)
		return
	}

	// Set CORS headers for preflight
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
	c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
	
	if cfg.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}
	
	if cfg.MaxAge > 0 {
		c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
	}

	utils.LogDebug(c, "CORS preflight approved", utils.Fields{
		"origin": origin,
		"methods": strings.Join(cfg.AllowedMethods, ", "),
	})

	c.AbortWithStatus(204) // No Content
}

// handleActualRequest handles actual CORS requests
func handleActualRequest(c *gin.Context, cfg *config.CORSConfig, origin string) {
	// Check if origin is allowed
	if !isOriginAllowed(origin, cfg.AllowedOrigins) {
		utils.LogDebug(c, "CORS request rejected", utils.Fields{
			"origin": origin,
			"reason": "origin not allowed",
		})
		return
	}

	// Set CORS headers for actual request
	c.Header("Access-Control-Allow-Origin", origin)
	
	if len(cfg.ExposedHeaders) > 0 {
		c.Header("Access-Control-Expose-Headers", strings.Join(cfg.ExposedHeaders, ", "))
	}
	
	if cfg.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	utils.LogDebug(c, "CORS request approved", utils.Fields{
		"origin": origin,
	})
}

// isOriginAllowed checks if the given origin is allowed based on the configuration
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}

	for _, allowedOrigin := range allowedOrigins {
		if matchOrigin(origin, allowedOrigin) {
			return true
		}
	}

	return false
}

// matchOrigin checks if an origin matches an allowed origin pattern
func matchOrigin(origin, pattern string) bool {
	// Exact match
	if origin == pattern {
		return true
	}

	// Wildcard match (e.g., *.example.com)
	if strings.HasPrefix(pattern, "*.") {
		domain := pattern[2:] // Remove "*."
		return strings.HasSuffix(origin, "."+domain) || origin == domain
	}

	// Special case for "*" (all origins) - should only be used in development
	if pattern == "*" {
		return true
	}

	return false
}

// ValidateCORSConfig validates CORS configuration and logs warnings for insecure settings
func ValidateCORSConfig(cfg *config.CORSConfig) {
	if !cfg.Enabled {
		fmt.Println("CORS is disabled")
		return
	}

	fmt.Printf("CORS Configuration loaded (profile: %s)\n", cfg.Profile)
	fmt.Printf("  - Allowed Origins: %v\n", cfg.AllowedOrigins)
	fmt.Printf("  - Allowed Methods: %v\n", cfg.AllowedMethods)
	fmt.Printf("  - Allowed Headers: %v\n", cfg.AllowedHeaders)
	fmt.Printf("  - Exposed Headers: %v\n", cfg.ExposedHeaders)
	fmt.Printf("  - Allow Credentials: %t\n", cfg.AllowCredentials)
	fmt.Printf("  - Max Age: %d seconds\n", cfg.MaxAge)

	// Security validations
	for _, origin := range cfg.AllowedOrigins {
		if origin == "*" && cfg.Profile == "production" {
			fmt.Println("WARNING: Wildcard origin (*) should not be used in production")
		}
		
		if strings.HasPrefix(origin, "http://") && cfg.Profile == "production" {
			fmt.Printf("WARNING: Non-HTTPS origin '%s' in production configuration\n", origin)
		}
		
		if strings.Contains(origin, "localhost") && cfg.Profile == "production" {
			fmt.Printf("WARNING: Localhost origin '%s' in production configuration\n", origin)
		}

		// Validate URL format
		if origin != "*" && !strings.HasPrefix(origin, "*.") {
			if _, err := url.Parse(origin); err != nil {
				fmt.Printf("WARNING: Invalid origin URL format: %s\n", origin)
			}
		}
	}

	// Check for dangerous methods in production
	if cfg.Profile == "production" {
		dangerousMethods := []string{"TRACE", "CONNECT"}
		for _, method := range cfg.AllowedMethods {
			for _, dangerous := range dangerousMethods {
				if method == dangerous {
					fmt.Printf("WARNING: Potentially dangerous method '%s' allowed in production\n", method)
				}
			}
		}
	}
}
