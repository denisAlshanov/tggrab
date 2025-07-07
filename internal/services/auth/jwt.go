package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/denisAlshanov/stPlaner/internal/models"
)

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	jwt.RegisteredClaims
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	SessionID   string   `json:"session_id"`
	TokenType   string   `json:"token_type"` // "access" or "refresh"
}

// JWTConfig represents JWT configuration
type JWTConfig struct {
	SecretKey            string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	Issuer               string
}

// JWTService handles JWT token operations
type JWTService struct {
	config   JWTConfig
	secretKey []byte
}

// NewJWTService creates a new JWT service
func NewJWTService(config JWTConfig) *JWTService {
	return &JWTService{
		config:   config,
		secretKey: []byte(config.SecretKey),
	}
}

// GenerateTokenPair generates both access and refresh tokens
func (j *JWTService) GenerateTokenPair(user *models.UserWithRoles, sessionID uuid.UUID) (*models.TokenPair, error) {
	// Extract roles and permissions
	roles := make([]string, len(user.Roles))
	
	for i, role := range user.Roles {
		roles[i] = role.Name
	}

	// Note: For full permission extraction, we'd need to query the database
	// For now, we'll use a simplified approach
	permissions := []string{} // This would be populated from user roles

	// Generate access token
	accessToken, err := j.generateToken(user, sessionID, "access", roles, permissions, j.config.AccessTokenDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := j.generateToken(user, sessionID, "refresh", roles, permissions, j.config.RefreshTokenDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(j.config.AccessTokenDuration.Seconds()),
	}, nil
}

// generateToken creates a JWT token with the specified parameters
func (j *JWTService) generateToken(user *models.UserWithRoles, sessionID uuid.UUID, tokenType string, roles, permissions []string, duration time.Duration) (string, error) {
	now := time.Now()
	jti, err := j.generateJTI()
	if err != nil {
		return "", fmt.Errorf("failed to generate JTI: %w", err)
	}

	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   user.ID.String(),
			Issuer:    j.config.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:      user.ID.String(),
		Email:       user.Email,
		Roles:       roles,
		Permissions: permissions,
		SessionID:   sessionID.String(),
		TokenType:   tokenType,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates and parses a JWT token
func (j *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate expiration
	if claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("token has expired")
	}

	return claims, nil
}

// ValidateAccessToken validates an access token specifically
func (j *JWTService) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "access" {
		return nil, fmt.Errorf("invalid token type, expected access token")
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token specifically
func (j *JWTService) ValidateRefreshToken(tokenString string) (*JWTClaims, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("invalid token type, expected refresh token")
	}

	return claims, nil
}

// ExtractTokenFromBearer extracts token from "Bearer <token>" format
func (j *JWTService) ExtractTokenFromBearer(bearerToken string) string {
	if len(bearerToken) > 7 && bearerToken[:7] == "Bearer " {
		return bearerToken[7:]
	}
	return bearerToken
}

// generateJTI generates a unique JWT ID
func (j *JWTService) generateJTI() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetTokenClaims extracts claims from a token without validation (for debugging)
func (j *JWTService) GetTokenClaims(tokenString string) (*JWTClaims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &JWTClaims{})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	return claims, nil
}

// IsTokenExpired checks if a token is expired without full validation
func (j *JWTService) IsTokenExpired(tokenString string) bool {
	claims, err := j.GetTokenClaims(tokenString)
	if err != nil {
		return true
	}
	return claims.ExpiresAt.Time.Before(time.Now())
}

// GetTokenExpiration returns the expiration time of a token
func (j *JWTService) GetTokenExpiration(tokenString string) (time.Time, error) {
	claims, err := j.GetTokenClaims(tokenString)
	if err != nil {
		return time.Time{}, err
	}
	return claims.ExpiresAt.Time, nil
}