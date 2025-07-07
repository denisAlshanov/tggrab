package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/denisAlshanov/stPlaner/internal/database"
	"github.com/denisAlshanov/stPlaner/internal/models"
	"github.com/denisAlshanov/stPlaner/internal/utils"
)

// GoogleOIDCConfig represents Google OAuth 2.0 configuration
type GoogleOIDCConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
}

// GoogleOIDCService handles Google OAuth 2.0 authentication
type GoogleOIDCService struct {
	config         GoogleOIDCConfig
	db             *database.PostgresDB
	sessionService *SessionService
	httpClient     *http.Client
}

// GoogleUserInfo represents user information from Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// GoogleTokenResponse represents Google's token response
type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
}

// NewGoogleOIDCService creates a new Google OIDC service
func NewGoogleOIDCService(config GoogleOIDCConfig, db *database.PostgresDB, sessionService *SessionService) *GoogleOIDCService {
	return &GoogleOIDCService{
		config:         config,
		db:             db,
		sessionService: sessionService,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

// GenerateAuthURL generates the Google OAuth authorization URL
func (g *GoogleOIDCService) GenerateAuthURL(ctx context.Context) (string, string, error) {
	// Generate state for CSRF protection
	state, err := g.generateState()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state in database with expiration
	oauthState := &models.OAuthState{
		State:     state,
		ExpiresAt: time.Now().Add(10 * time.Minute), // 10 minutes expiration
	}

	if err := g.db.CreateOAuthState(ctx, oauthState); err != nil {
		return "", "", fmt.Errorf("failed to store OAuth state: %w", err)
	}

	// Build authorization URL
	authURL := fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?%s",
		url.Values{
			"client_id":     {g.config.ClientID},
			"redirect_uri":  {g.config.RedirectURI},
			"scope":         {strings.Join(g.config.Scopes, " ")},
			"response_type": {"code"},
			"state":         {state},
			"access_type":   {"offline"}, // Request refresh token
			"prompt":        {"consent"},  // Force consent to get refresh token
		}.Encode())

	return authURL, state, nil
}

// HandleCallback processes the OAuth callback from Google
func (g *GoogleOIDCService) HandleCallback(ctx context.Context, code, state string, deviceInfo *models.DeviceInfo) (*models.TokenPair, *models.UserWithRoles, bool, error) {
	// Validate state
	if err := g.validateState(ctx, state); err != nil {
		return nil, nil, false, fmt.Errorf("invalid state: %w", err)
	}

	// Exchange code for tokens
	tokens, err := g.exchangeCodeForTokens(ctx, code)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to exchange code for tokens: %w", err)
	}

	// Get user info from Google
	userInfo, err := g.getUserInfo(ctx, tokens.AccessToken)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to get user info: %w", err)
	}

	// Validate email is verified
	if !userInfo.VerifiedEmail {
		return nil, nil, false, errors.New("email not verified with Google")
	}

	// Check if user exists
	existingUser, err := g.db.GetUserByEmail(ctx, userInfo.Email)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to check existing user: %w", err)
	}

	var user *models.UserWithRoles
	isNewUser := false

	if existingUser == nil {
		// Create new user
		newUser := &models.User{
			ID:           uuid.New(),
			Name:         userInfo.GivenName,
			Surname:      userInfo.FamilyName,
			Email:        userInfo.Email,
			OIDCProvider: utils.StringPtr("google"),
			OIDCSubject:  utils.StringPtr(userInfo.ID),
			Status:       models.UserStatusActive,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Assign default role (e.g., "user")
		defaultRoleID, err := g.getDefaultRoleID(ctx)
		if err != nil {
			return nil, nil, false, fmt.Errorf("failed to get default role: %w", err)
		}

		if err := g.db.CreateUser(ctx, newUser, []uuid.UUID{defaultRoleID}); err != nil {
			return nil, nil, false, fmt.Errorf("failed to create user: %w", err)
		}

		// Get user with roles
		user, err = g.db.GetUserByID(ctx, newUser.ID)
		if err != nil {
			return nil, nil, false, fmt.Errorf("failed to get created user: %w", err)
		}
		isNewUser = true
	} else {
		// Update existing user's OIDC info if not set
		if existingUser.OIDCProvider == nil || existingUser.OIDCSubject == nil {
			existingUser.OIDCProvider = utils.StringPtr("google")
			existingUser.OIDCSubject = utils.StringPtr(userInfo.ID)
			existingUser.UpdatedAt = time.Now()

			if err := g.db.UpdateUser(ctx, &existingUser.User, nil); err != nil {
				return nil, nil, false, fmt.Errorf("failed to update user OIDC info: %w", err)
			}
		}

		// Update last login
		if err := g.db.UpdateUserLoginTime(ctx, existingUser.ID); err != nil {
			// Don't fail for this
		}

		user = existingUser
	}

	// Create session and generate JWT tokens
	tokenPair, err := g.sessionService.CreateSession(ctx, user, deviceInfo)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to create session: %w", err)
	}

	return tokenPair, user, isNewUser, nil
}

// LinkGoogleAccount links a Google account to an existing user
func (g *GoogleOIDCService) LinkGoogleAccount(ctx context.Context, userID uuid.UUID, idToken string) error {
	// Validate ID token and extract claims
	userInfo, err := g.validateIDToken(ctx, idToken)
	if err != nil {
		return fmt.Errorf("invalid ID token: %w", err)
	}

	// Get user to verify they exist
	user, err := g.db.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return errors.New("user not found")
	}

	// Check if Google account is already linked to another user
	existingUser, err := g.db.GetUserByOIDC(ctx, "google", userInfo.ID)
	if err != nil {
		return fmt.Errorf("failed to check existing Google account: %w", err)
	}

	if existingUser != nil && existingUser.ID != userID {
		return errors.New("Google account is already linked to another user")
	}

	// Update user with Google OIDC info
	user.OIDCProvider = utils.StringPtr("google")
	user.OIDCSubject = utils.StringPtr(userInfo.ID)
	user.UpdatedAt = time.Now()

	if err := g.db.UpdateUser(ctx, &user.User, nil); err != nil {
		return fmt.Errorf("failed to link Google account: %w", err)
	}

	return nil
}

// validateState validates the OAuth state parameter
func (g *GoogleOIDCService) validateState(ctx context.Context, state string) error {
	// Get state from database
	storedState, err := g.db.GetOAuthState(ctx, state)
	if err != nil {
		return fmt.Errorf("failed to get OAuth state: %w", err)
	}

	if storedState == nil {
		return errors.New("invalid or expired state")
	}

	// Check if state has expired
	if storedState.ExpiresAt.Before(time.Now()) {
		// Clean up expired state
		g.db.DeleteOAuthState(ctx, state)
		return errors.New("state has expired")
	}

	// Clean up used state
	if err := g.db.DeleteOAuthState(ctx, state); err != nil {
		// Log error but don't fail
		utils.LogError(ctx, "Failed to delete OAuth state", err)
	}

	return nil
}

// exchangeCodeForTokens exchanges authorization code for access tokens
func (g *GoogleOIDCService) exchangeCodeForTokens(ctx context.Context, code string) (*GoogleTokenResponse, error) {
	data := url.Values{
		"client_id":     {g.config.ClientID},
		"client_secret": {g.config.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {g.config.RedirectURI},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google token exchange failed with status %d", resp.StatusCode)
	}

	var tokens GoogleTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, err
	}

	return &tokens, nil
}

// getUserInfo retrieves user information from Google
func (g *GoogleOIDCService) getUserInfo(ctx context.Context, accessToken string) (*GoogleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google userinfo request failed with status %d", resp.StatusCode)
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// validateIDToken validates a Google ID token (simplified implementation)
func (g *GoogleOIDCService) validateIDToken(ctx context.Context, idToken string) (*GoogleUserInfo, error) {
	// In production, you should validate the ID token signature using Google's public keys
	// For now, we'll use Google's tokeninfo endpoint (not recommended for production)
	req, err := http.NewRequestWithContext(ctx, "GET", 
		"https://oauth2.googleapis.com/tokeninfo?id_token="+idToken, nil)
	if err != nil {
		return nil, err
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ID token validation failed with status %d", resp.StatusCode)
	}

	var tokenInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, err
	}

	// Validate audience
	if aud, ok := tokenInfo["aud"].(string); !ok || aud != g.config.ClientID {
		return nil, errors.New("invalid token audience")
	}

	// Extract user info
	userInfo := &GoogleUserInfo{
		ID:            tokenInfo["sub"].(string),
		Email:         tokenInfo["email"].(string),
		VerifiedEmail: tokenInfo["email_verified"].(bool),
		Name:          tokenInfo["name"].(string),
		GivenName:     tokenInfo["given_name"].(string),
		FamilyName:    tokenInfo["family_name"].(string),
	}

	return userInfo, nil
}

// getDefaultRoleID gets the default role ID for new users
func (g *GoogleOIDCService) getDefaultRoleID(ctx context.Context) (uuid.UUID, error) {
	// Try to get "user" role first
	role, err := g.db.GetRoleByName(ctx, "user")
	if err == nil && role != nil {
		return role.ID, nil
	}

	// If "user" role doesn't exist, get any role or create a default one
	roles, _, err := g.db.ListRoles(ctx, nil, nil, &models.PaginationOptions{Limit: 1})
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get roles: %w", err)
	}

	if len(roles) == 0 {
		return uuid.Nil, errors.New("no roles available for new users")
	}

	return roles[0].ID, nil
}

// generateState generates a random state string for CSRF protection
func (g *GoogleOIDCService) generateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}