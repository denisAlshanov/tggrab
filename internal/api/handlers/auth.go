package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/denisAlshanov/stPlaner/internal/database"
	"github.com/denisAlshanov/stPlaner/internal/models"
	"github.com/denisAlshanov/stPlaner/internal/services/auth"
	"github.com/denisAlshanov/stPlaner/internal/utils"
)

type AuthHandlers struct {
	db             *database.PostgresDB
	jwtService     *auth.JWTService
	sessionService *auth.SessionService
	googleService  *auth.GoogleOIDCService
}

func NewAuthHandlers(db *database.PostgresDB, jwtService *auth.JWTService, sessionService *auth.SessionService, googleService *auth.GoogleOIDCService) *AuthHandlers {
	return &AuthHandlers{
		db:             db,
		jwtService:     jwtService,
		sessionService: sessionService,
		googleService:  googleService,
	}
}

// Login godoc
// @Summary User login with password
// @Description Authenticate user with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.LoginResponse
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Failure 429 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/auth/login [post]
func (h *AuthHandlers) Login(c *gin.Context) {
	utils.LogInfo(c, "Login attempt")

	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(c, "Invalid login request", err)
		c.JSON(http.StatusBadRequest, models.APIError{
			Error:   "INVALID_REQUEST",
			Message: "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// Validate email format
	if !utils.IsValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, models.APIError{
			Error:   "INVALID_EMAIL",
			Message: "Invalid email format",
		})
		return
	}

	// Get user by email
	user, err := h.db.GetUserByEmail(c, req.Email)
	if err != nil {
		utils.LogError(c, "Failed to get user by email", err)
		c.JSON(http.StatusInternalServerError, models.APIError{
			Error:   "DATABASE_ERROR",
			Message: "Failed to authenticate user",
		})
		return
	}

	if user == nil {
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "INVALID_CREDENTIALS",
			Message: "Invalid email or password",
		})
		return
	}

	// Check if user has password set (not OIDC-only user)
	if user.PasswordHash == nil {
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "PASSWORD_NOT_SET",
			Message: "This account uses OAuth authentication. Please login with Google.",
		})
		return
	}

	// Verify password
	if err := utils.VerifyPassword(req.Password, *user.PasswordHash); err != nil {
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "INVALID_CREDENTIALS",
			Message: "Invalid email or password",
		})
		return
	}

	// Update last login
	if err := h.db.UpdateUserLoginTime(c, user.ID); err != nil {
		utils.LogError(c, "Failed to update last login", err)
		// Don't fail the login for this
	}

	// Extract device info from request
	deviceInfo := extractDeviceInfo(c, req.DeviceInfo)

	// Create session and generate tokens
	tokenPair, err := h.sessionService.CreateSession(c, user, deviceInfo)
	if err != nil {
		utils.LogError(c, "Failed to create session", err)
		c.JSON(http.StatusInternalServerError, models.APIError{
			Error:   "SESSION_ERROR",
			Message: "Failed to create user session",
		})
		return
	}

	utils.LogInfo(c, fmt.Sprintf("User %s logged in successfully", user.Email))

	// Create login response data
	responseData := &models.LoginResponseData{
		TokenPair: *tokenPair,
		User:      user,
	}

	c.JSON(http.StatusOK, models.LoginResponse{
		Success: true,
		Data:    responseData,
	})
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Exchange refresh token for new access token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} models.RefreshTokenResponse
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandlers) RefreshToken(c *gin.Context) {
	utils.LogInfo(c, "Token refresh attempt")

	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{
			Error:   "INVALID_REQUEST",
			Message: "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// Refresh the session
	tokenPair, err := h.sessionService.RefreshSession(c, req.RefreshToken)
	if err != nil {
		utils.LogError(c, "Failed to refresh session", err)
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "INVALID_REFRESH_TOKEN",
			Message: "Invalid or expired refresh token",
		})
		return
	}

	utils.LogInfo(c, "Token refreshed successfully")

	c.JSON(http.StatusOK, models.RefreshTokenResponse{
		Success: true,
		Data:    tokenPair,
	})
}

// Logout godoc
// @Summary User logout
// @Description Logout user and invalidate tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.LogoutRequest true "Logout request"
// @Success 200 {object} models.LogoutResponse
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/auth/logout [post]
func (h *AuthHandlers) Logout(c *gin.Context) {
	utils.LogInfo(c, "Logout attempt")

	var req models.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{
			Error:   "INVALID_REQUEST",
			Message: "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// Get current session from token
	token := extractTokenFromHeader(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "MISSING_TOKEN",
			Message: "Authorization token required",
		})
		return
	}

	session, err := h.sessionService.GetSessionFromToken(c, token)
	if err != nil {
		utils.LogError(c, "Failed to get session from token", err)
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "INVALID_TOKEN",
			Message: "Invalid authorization token",
		})
		return
	}

	// Validate refresh token if provided
	if req.RefreshToken != nil && *req.RefreshToken != "" {
		if session.RefreshToken != *req.RefreshToken {
			c.JSON(http.StatusBadRequest, models.APIError{
				Error:   "TOKEN_MISMATCH",
				Message: "Refresh token does not match current session",
			})
			return
		}
	}

	// Get token claims for blacklisting
	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		utils.LogError(c, "Failed to validate token for logout", err)
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "INVALID_TOKEN",
			Message: "Invalid authorization token",
		})
		return
	}

	// Blacklist current access token
	if err := h.sessionService.BlacklistToken(c, claims.ID, session.UserID, "user_logout"); err != nil {
		utils.LogError(c, "Failed to blacklist token", err)
		// Continue with logout even if blacklisting fails
	}

	if req.LogoutAllDevices {
		// Revoke all user sessions
		if err := h.sessionService.RevokeUserSessions(c, session.UserID); err != nil {
			utils.LogError(c, "Failed to revoke all user sessions", err)
			c.JSON(http.StatusInternalServerError, models.APIError{
				Error:   "LOGOUT_ERROR",
				Message: "Failed to logout from all devices",
			})
			return
		}
		utils.LogInfo(c, "User logged out from all devices")
	} else {
		// Revoke current session only
		if err := h.sessionService.RevokeSession(c, session.ID); err != nil {
			utils.LogError(c, "Failed to revoke session", err)
			c.JSON(http.StatusInternalServerError, models.APIError{
				Error:   "LOGOUT_ERROR",
				Message: "Failed to logout",
			})
			return
		}
		utils.LogInfo(c, "User logged out from current device")
	}

	c.JSON(http.StatusOK, models.LogoutResponse{
		Success: true,
		Message: "Successfully logged out",
	})
}

// VerifyToken godoc
// @Summary Verify access token
// @Description Verify if the provided access token is valid
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.VerifyTokenResponse
// @Failure 401 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/auth/verify [get]
func (h *AuthHandlers) VerifyToken(c *gin.Context) {

	token := extractTokenFromHeader(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "MISSING_TOKEN",
			Message: "Authorization token required",
		})
		return
	}

	// Validate token
	claims, err := h.jwtService.ValidateAccessToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "INVALID_TOKEN",
			Message: "Invalid or expired token",
		})
		return
	}

	// Check if token is blacklisted
	blacklisted, err := h.sessionService.IsTokenBlacklisted(c, claims.ID)
	if err != nil {
		utils.LogError(c, "Failed to check token blacklist", err)
		c.JSON(http.StatusInternalServerError, models.APIError{
			Error:   "TOKEN_CHECK_ERROR",
			Message: "Failed to verify token status",
		})
		return
	}

	if blacklisted {
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "TOKEN_REVOKED",
			Message: "Token has been revoked",
		})
		return
	}

	responseData := &models.TokenVerificationData{
		Valid:     true,
		UserID:    claims.UserID,
		Email:     claims.Email,
		Roles:     claims.Roles,
		ExpiresAt: claims.ExpiresAt.Time,
	}

	c.JSON(http.StatusOK, models.VerifyTokenResponse{
		Success: true,
		Data:    responseData,
	})
}

// GetActiveSessions godoc
// @Summary Get user's active sessions
// @Description Retrieve all active sessions for the authenticated user
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.SessionListResponse
// @Failure 401 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/auth/sessions [get]
func (h *AuthHandlers) GetActiveSessions(c *gin.Context) {

	token := extractTokenFromHeader(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "MISSING_TOKEN",
			Message: "Authorization token required",
		})
		return
	}

	// Get current session
	currentSession, err := h.sessionService.GetSessionFromToken(c, token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "INVALID_TOKEN",
			Message: "Invalid authorization token",
		})
		return
	}

	// Get all user sessions
	sessions, err := h.sessionService.GetUserSessions(c, currentSession.UserID)
	if err != nil {
		utils.LogError(c, "Failed to get user sessions", err)
		c.JSON(http.StatusInternalServerError, models.APIError{
			Error:   "SESSION_ERROR",
			Message: "Failed to retrieve sessions",
		})
		return
	}

	// Mark current session
	for i := range sessions {
		if sessions[i].ID == currentSession.ID.String() {
			sessions[i].IsCurrent = true
			break
		}
	}

	responseData := &models.SessionListData{
		Sessions: sessions,
	}

	c.JSON(http.StatusOK, models.SessionListResponse{
		Success: true,
		Data:    responseData,
	})
}

// RevokeSession godoc
// @Summary Revoke a specific session
// @Description Revoke a specific session by session ID
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Param session_id path string true "Session ID"
// @Success 200 {object} models.RevokeSessionResponse
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Failure 403 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/auth/sessions/{session_id} [delete]
func (h *AuthHandlers) RevokeSession(c *gin.Context) {

	sessionIDStr := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{
			Error:   "INVALID_SESSION_ID",
			Message: "Invalid session ID format",
		})
		return
	}

	token := extractTokenFromHeader(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "MISSING_TOKEN",
			Message: "Authorization token required",
		})
		return
	}

	// Get current session
	currentSession, err := h.sessionService.GetSessionFromToken(c, token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "INVALID_TOKEN",
			Message: "Invalid authorization token",
		})
		return
	}

	// Get target session to verify ownership
	targetSession, err := h.sessionService.ValidateSession(c, sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{
			Error:   "SESSION_NOT_FOUND",
			Message: "Session not found or expired",
		})
		return
	}

	// Verify user owns this session
	if targetSession.UserID != currentSession.UserID {
		c.JSON(http.StatusForbidden, models.APIError{
			Error:   "SESSION_ACCESS_DENIED",
			Message: "You can only revoke your own sessions",
		})
		return
	}

	// Revoke the session
	if err := h.sessionService.RevokeSession(c, sessionID); err != nil {
		utils.LogError(c, "Failed to revoke session", err)
		c.JSON(http.StatusInternalServerError, models.APIError{
			Error:   "SESSION_REVOKE_ERROR",
			Message: "Failed to revoke session",
		})
		return
	}

	utils.LogInfo(c, fmt.Sprintf("Session %s revoked", sessionID.String()))

	c.JSON(http.StatusOK, models.RevokeSessionResponse{
		Success: true,
		Message: "Session revoked successfully",
	})
}

// Google OIDC Handlers

// GoogleLogin godoc
// @Summary Initiate Google OAuth login
// @Description Generate Google OAuth authorization URL
// @Tags Authentication
// @Produce json
// @Param redirect_uri query string false "Redirect URI after Google auth"
// @Success 200 {object} models.GoogleLoginResponse
// @Failure 500 {object} models.APIError
// @Router /api/v1/auth/google/login [get]
func (h *AuthHandlers) GoogleLogin(c *gin.Context) {
	utils.LogInfo(c, "Google OAuth login initiation")

	// Generate Google OAuth URL
	authURL, state, err := h.googleService.GenerateAuthURL(c)
	if err != nil {
		utils.LogError(c, "Failed to generate Google auth URL", err)
		c.JSON(http.StatusInternalServerError, models.APIError{
			Error:   "OAUTH_ERROR",
			Message: "Failed to initialize Google authentication",
		})
		return
	}

	responseData := &models.GoogleLoginResponseData{
		AuthURL: authURL,
		State:   state,
	}

	c.JSON(http.StatusOK, models.GoogleLoginResponse{
		Success: true,
		Data:    responseData,
	})
}

// GoogleCallback godoc
// @Summary Handle Google OAuth callback
// @Description Process Google OAuth callback and create user session
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.GoogleCallbackRequest true "Google callback data"
// @Success 200 {object} models.GoogleCallbackResponse
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/auth/google/callback [post]
func (h *AuthHandlers) GoogleCallback(c *gin.Context) {
	utils.LogInfo(c, "Google OAuth callback processing")

	var req models.GoogleCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{
			Error:   "INVALID_REQUEST",
			Message: "Invalid callback request format",
			Details: err.Error(),
		})
		return
	}

	// Extract device info from request
	deviceInfo := extractDeviceInfo(c, req.DeviceInfo)

	// Handle Google OAuth callback
	tokenPair, user, isNewUser, err := h.googleService.HandleCallback(c, req.Code, req.State, deviceInfo)
	if err != nil {
		utils.LogError(c, "Failed to handle Google callback", err)
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "OAUTH_CALLBACK_ERROR",
			Message: "Failed to process Google authentication",
			Details: err.Error(),
		})
		return
	}

	utils.LogInfo(c, fmt.Sprintf("User %s authenticated via Google (new_user: %v)", user.Email, isNewUser))

	responseData := &models.GoogleCallbackResponseData{
		TokenPair: *tokenPair,
		User:      user,
		IsNewUser: isNewUser,
	}

	c.JSON(http.StatusOK, models.GoogleCallbackResponse{
		Success: true,
		Data:    responseData,
	})
}

// GoogleLink godoc
// @Summary Link Google account to existing user
// @Description Link a Google account to the currently authenticated user
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.GoogleLinkRequest true "Google ID token"
// @Success 200 {object} models.GoogleLinkResponse
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Failure 409 {object} models.APIError
// @Failure 500 {object} models.APIError
// @Router /api/v1/auth/google/link [post]
func (h *AuthHandlers) GoogleLink(c *gin.Context) {
	utils.LogInfo(c, "Google account linking attempt")

	var req models.GoogleLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{
			Error:   "INVALID_REQUEST",
			Message: "Invalid link request format",
			Details: err.Error(),
		})
		return
	}

	token := extractTokenFromHeader(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "MISSING_TOKEN",
			Message: "Authorization token required",
		})
		return
	}

	// Get current user from token
	claims, err := h.jwtService.ValidateAccessToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "INVALID_TOKEN",
			Message: "Invalid authorization token",
		})
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.APIError{
			Error:   "INVALID_USER_ID",
			Message: "Invalid user ID in token",
		})
		return
	}

	// Link Google account
	if err := h.googleService.LinkGoogleAccount(c, userID, req.IDToken); err != nil {
		utils.LogError(c, "Failed to link Google account", err)
		
		if strings.Contains(err.Error(), "already linked") {
			c.JSON(http.StatusConflict, models.APIError{
				Error:   "ACCOUNT_ALREADY_LINKED",
				Message: "Google account is already linked to another user",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.APIError{
			Error:   "LINK_ERROR",
			Message: "Failed to link Google account",
			Details: err.Error(),
		})
		return
	}

	utils.LogInfo(c, fmt.Sprintf("Google account linked to user %s", claims.UserID))

	c.JSON(http.StatusOK, models.GoogleLinkResponse{
		Success: true,
		Message: "Google account successfully linked",
	})
}

// Helper functions

func extractTokenFromHeader(c *gin.Context) string {
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

func extractDeviceInfo(c *gin.Context, requestDeviceInfo *models.DeviceInfo) *models.DeviceInfo {
	deviceInfo := &models.DeviceInfo{}

	if requestDeviceInfo != nil {
		deviceInfo.DeviceName = requestDeviceInfo.DeviceName
		deviceInfo.DeviceType = requestDeviceInfo.DeviceType
	}

	// Extract from headers if not provided in request
	if deviceInfo.IPAddress == "" {
		deviceInfo.IPAddress = c.ClientIP()
	}
	if deviceInfo.UserAgent == "" {
		deviceInfo.UserAgent = c.GetHeader("User-Agent")
	}

	return deviceInfo
}