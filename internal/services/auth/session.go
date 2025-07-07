package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/denisAlshanov/stPlaner/internal/database"
	"github.com/denisAlshanov/stPlaner/internal/models"
)

// SessionService handles session management operations
type SessionService struct {
	db         *database.PostgresDB
	jwtService *JWTService
}

// NewSessionService creates a new session service
func NewSessionService(db *database.PostgresDB, jwtService *JWTService) *SessionService {
	return &SessionService{
		db:         db,
		jwtService: jwtService,
	}
}

// CreateSession creates a new user session
func (s *SessionService) CreateSession(ctx context.Context, user *models.UserWithRoles, deviceInfo *models.DeviceInfo) (*models.TokenPair, error) {
	// Generate session ID
	sessionID := uuid.New()

	// Generate JWT token pair first
	tokenPair, err := s.jwtService.GenerateTokenPair(user, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Create session record with refresh token
	session := &models.Session{
		ID:           sessionID,
		UserID:       user.ID,
		RefreshToken: tokenPair.RefreshToken,
		IsActive:     true,
		ExpiresAt:    time.Now().Add(s.jwtService.config.RefreshTokenDuration),
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	// Set device info if provided
	if deviceInfo != nil {
		if deviceInfo.DeviceName != "" {
			session.DeviceName = &deviceInfo.DeviceName
		}
		if deviceInfo.DeviceType != "" {
			session.DeviceType = &deviceInfo.DeviceType
		}
		if deviceInfo.IPAddress != "" {
			session.IPAddress = &deviceInfo.IPAddress
		}
		if deviceInfo.UserAgent != "" {
			session.UserAgent = &deviceInfo.UserAgent
		}
	}

	// Save session to database
	if err := s.db.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return tokenPair, nil
}

// RefreshSession refreshes an existing session and generates new tokens
func (s *SessionService) RefreshSession(ctx context.Context, refreshToken string) (*models.TokenPair, error) {
	// Validate refresh token
	claims, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Parse session ID
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID in token: %w", err)
	}

	// Get session from database
	session, err := s.db.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	if !session.IsActive {
		return nil, fmt.Errorf("session is not active")
	}

	if session.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("session has expired")
	}

	if session.RefreshToken != refreshToken {
		return nil, fmt.Errorf("refresh token mismatch")
	}

	// Get user with roles
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token: %w", err)
	}

	user, err := s.db.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Blacklist old tokens
	if err := s.BlacklistToken(ctx, claims.ID, userID, "token_refresh"); err != nil {
		return nil, fmt.Errorf("failed to blacklist old token: %w", err)
	}

	// Generate new token pair
	tokenPair, err := s.jwtService.GenerateTokenPair(user, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new tokens: %w", err)
	}

	// Update session with new refresh token and activity
	session.RefreshToken = tokenPair.RefreshToken
	session.LastActivity = time.Now()
	if err := s.db.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return tokenPair, nil
}

// RevokeSession revokes a user session
func (s *SessionService) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	return s.db.DeleteSession(ctx, sessionID)
}

// RevokeUserSessions revokes all sessions for a user
func (s *SessionService) RevokeUserSessions(ctx context.Context, userID uuid.UUID) error {
	return s.db.DeleteUserSessions(ctx, userID)
}

// GetUserSessions retrieves all active sessions for a user
func (s *SessionService) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]models.SessionInfo, error) {
	sessions, err := s.db.GetUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]models.SessionInfo, len(sessions))
	for i, session := range sessions {
		result[i] = models.SessionInfo{
			ID:           session.ID.String(),
			DeviceName:   session.DeviceName,
			DeviceType:   session.DeviceType,
			IPAddress:    session.IPAddress,
			CreatedAt:    session.CreatedAt,
			LastActivity: session.LastActivity,
			IsCurrent:    false, // This would be determined by comparing with current session
		}
	}

	return result, nil
}

// ValidateSession validates if a session is active and valid
func (s *SessionService) ValidateSession(ctx context.Context, sessionID uuid.UUID) (*models.Session, error) {
	session, err := s.db.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	if !session.IsActive {
		return nil, fmt.Errorf("session is not active")
	}

	if session.ExpiresAt.Before(time.Now()) {
		// Clean up expired session
		s.db.DeleteSession(ctx, sessionID)
		return nil, fmt.Errorf("session has expired")
	}

	return session, nil
}

// BlacklistToken adds a token to the blacklist
func (s *SessionService) BlacklistToken(ctx context.Context, jti string, userID uuid.UUID, reason string) error {
	blacklist := &models.TokenBlacklist{
		ID:        uuid.New(),
		TokenJTI:  jti,
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.jwtService.config.AccessTokenDuration), // Only need to blacklist until normal expiry
		CreatedAt: time.Now(),
		Reason:    reason,
	}

	return s.db.CreateTokenBlacklist(ctx, blacklist)
}

// IsTokenBlacklisted checks if a token is blacklisted
func (s *SessionService) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	return s.db.IsTokenBlacklisted(ctx, jti)
}

// CleanupExpiredSessions removes expired sessions and blacklisted tokens
func (s *SessionService) CleanupExpiredSessions(ctx context.Context) error {
	return s.db.CleanupExpiredAuthData(ctx)
}

// UpdateSessionActivity updates the last activity time for a session
func (s *SessionService) UpdateSessionActivity(ctx context.Context, sessionID uuid.UUID) error {
	return s.db.UpdateSessionActivity(ctx, sessionID)
}

// generateSecureToken generates a cryptographically secure random token
func (s *SessionService) generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetSessionFromToken extracts session information from a JWT token
func (s *SessionService) GetSessionFromToken(ctx context.Context, token string) (*models.Session, error) {
	claims, err := s.jwtService.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID in token: %w", err)
	}

	return s.ValidateSession(ctx, sessionID)
}