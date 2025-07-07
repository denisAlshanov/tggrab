# Product Requirements Document: User Authentication System

## 1. Overview

This document outlines the requirements for implementing a comprehensive authentication system for the St. Planer service. The system will provide JWT-based authentication with support for both password-based login and Google OIDC (OpenID Connect) authentication.

## 2. Objectives

- Implement secure JWT-based authentication for API access
- Provide password-based login functionality
- Support Google OIDC authentication flow
- Implement logout mechanism with token invalidation
- Enable seamless authentication experience across different login methods
- Ensure secure token management and refresh capabilities

## 3. System Requirements

### 3.1 JWT Token Management

#### Token Structure
- **Access Token**: Short-lived JWT for API authentication (15-30 minutes)
- **Refresh Token**: Long-lived token for obtaining new access tokens (7-30 days)
- **Token Claims**: User ID, email, roles, permissions, issued at, expiration
- **Token Storage**: Client-side storage with security best practices

#### Security Requirements
- Signed with HS256 or RS256 algorithm
- Secure secret key management
- Token rotation on refresh
- Blacklist mechanism for revoked tokens

### 3.2 Authentication Methods

#### Password Authentication
- Validate user credentials against stored bcrypt hash
- Generate JWT tokens on successful authentication
- Track login attempts and implement rate limiting
- Update last login timestamp

#### Google OIDC Authentication
- OAuth 2.0 authorization code flow
- Validate Google ID tokens
- Auto-register new users or link to existing accounts
- Extract user profile information from Google

### 3.3 Session Management

#### Active Sessions
- Track active user sessions
- Support multiple concurrent sessions per user
- Ability to list and revoke active sessions
- Session metadata (device, IP, location)

#### Token Lifecycle
- Access token expiration and refresh
- Refresh token rotation
- Grace period for expired tokens
- Automatic cleanup of expired sessions

## 4. API Endpoints

### 4.1 Authentication Endpoints

#### 4.1.1 Password Login - POST /api/v1/auth/login

**Request Body:**
```json
{
    "email": "user@gmail.com",
    "password": "userpassword",
    "device_info": {
        "device_name": "Chrome on Windows",
        "device_type": "web",
        "ip_address": "192.168.1.1"
    }
}
```

**Response:**
```json
{
    "success": true,
    "data": {
        "access_token": "eyJhbGciOiJIUzI1NiIs...",
        "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
        "token_type": "Bearer",
        "expires_in": 1800,
        "user": {
            "id": "user-uuid",
            "email": "user@gmail.com",
            "name": "John",
            "surname": "Doe",
            "roles": [
                {
                    "id": "role-uuid",
                    "name": "admin",
                    "permissions": ["users:read", "users:write"]
                }
            ]
        }
    }
}
```

#### 4.1.2 Refresh Token - POST /api/v1/auth/refresh

**Request Body:**
```json
{
    "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response:**
```json
{
    "success": true,
    "data": {
        "access_token": "eyJhbGciOiJIUzI1NiIs...",
        "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
        "token_type": "Bearer",
        "expires_in": 1800
    }
}
```

#### 4.1.3 Logout - POST /api/v1/auth/logout

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "logout_all_devices": false
}
```

**Response:**
```json
{
    "success": true,
    "message": "Successfully logged out"
}
```

#### 4.1.4 Verify Token - GET /api/v1/auth/verify

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:**
```json
{
    "success": true,
    "data": {
        "valid": true,
        "user_id": "user-uuid",
        "email": "user@gmail.com",
        "roles": ["admin"],
        "expires_at": "2024-01-07T12:00:00Z"
    }
}
```

### 4.2 Google OIDC Endpoints

#### 4.2.1 Initiate Google Login - GET /api/v1/auth/google/login

**Query Parameters:**
- `redirect_uri`: Where to redirect after Google auth
- `state`: CSRF protection token

**Response:**
```json
{
    "success": true,
    "data": {
        "auth_url": "https://accounts.google.com/o/oauth2/v2/auth?client_id=...&redirect_uri=...&state=...",
        "state": "random-state-token"
    }
}
```

#### 4.2.2 Google Callback - POST /api/v1/auth/google/callback

**Request Body:**
```json
{
    "code": "authorization-code-from-google",
    "state": "random-state-token",
    "device_info": {
        "device_name": "Chrome on Windows",
        "device_type": "web"
    }
}
```

**Response:**
```json
{
    "success": true,
    "data": {
        "access_token": "eyJhbGciOiJIUzI1NiIs...",
        "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
        "token_type": "Bearer",
        "expires_in": 1800,
        "user": {
            "id": "user-uuid",
            "email": "user@gmail.com",
            "name": "John",
            "surname": "Doe",
            "oidc_provider": "google",
            "roles": [...]
        },
        "is_new_user": false
    }
}
```

#### 4.2.3 Link Google Account - POST /api/v1/auth/google/link

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request Body:**
```json
{
    "id_token": "google-id-token"
}
```

**Response:**
```json
{
    "success": true,
    "message": "Google account successfully linked"
}
```

### 4.3 Session Management Endpoints

#### 4.3.1 List Active Sessions - GET /api/v1/auth/sessions

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:**
```json
{
    "success": true,
    "data": {
        "sessions": [
            {
                "id": "session-uuid",
                "device_name": "Chrome on Windows",
                "device_type": "web",
                "ip_address": "192.168.1.1",
                "location": "New York, US",
                "created_at": "2024-01-07T10:00:00Z",
                "last_activity": "2024-01-07T11:30:00Z",
                "is_current": true
            }
        ]
    }
}
```

#### 4.3.2 Revoke Session - DELETE /api/v1/auth/sessions/{session_id}

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:**
```json
{
    "success": true,
    "message": "Session revoked successfully"
}
```

## 5. Data Models

### 5.1 Session Entity

```go
type Session struct {
    ID           uuid.UUID              `json:"id" db:"id"`
    UserID       uuid.UUID              `json:"user_id" db:"user_id"`
    RefreshToken string                 `json:"-" db:"refresh_token"`
    DeviceName   string                 `json:"device_name" db:"device_name"`
    DeviceType   string                 `json:"device_type" db:"device_type"`
    IPAddress    string                 `json:"ip_address" db:"ip_address"`
    UserAgent    string                 `json:"user_agent" db:"user_agent"`
    IsActive     bool                   `json:"is_active" db:"is_active"`
    ExpiresAt    time.Time              `json:"expires_at" db:"expires_at"`
    CreatedAt    time.Time              `json:"created_at" db:"created_at"`
    LastActivity time.Time              `json:"last_activity" db:"last_activity"`
    Metadata     map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}
```

### 5.2 Token Blacklist Entity

```go
type TokenBlacklist struct {
    ID        uuid.UUID `json:"id" db:"id"`
    TokenJTI  string    `json:"token_jti" db:"token_jti"`
    UserID    uuid.UUID `json:"user_id" db:"user_id"`
    ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    Reason    string    `json:"reason" db:"reason"`
}
```

### 5.3 JWT Claims Structure

```go
type JWTClaims struct {
    jwt.RegisteredClaims
    UserID      string   `json:"user_id"`
    Email       string   `json:"email"`
    Roles       []string `json:"roles"`
    Permissions []string `json:"permissions"`
    SessionID   string   `json:"session_id"`
    TokenType   string   `json:"token_type"` // "access" or "refresh"
}
```

## 6. Database Schema

### 6.1 Sessions Table

```sql
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token TEXT UNIQUE NOT NULL,
    device_name VARCHAR(255),
    device_type VARCHAR(50),
    ip_address INET,
    user_agent TEXT,
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB,
    INDEX idx_sessions_user_id (user_id),
    INDEX idx_sessions_refresh_token (refresh_token),
    INDEX idx_sessions_expires_at (expires_at)
);
```

### 6.2 Token Blacklist Table

```sql
CREATE TABLE token_blacklist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_jti VARCHAR(255) UNIQUE NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    reason VARCHAR(255),
    INDEX idx_token_blacklist_jti (token_jti),
    INDEX idx_token_blacklist_expires_at (expires_at)
);
```

## 7. Google OIDC Configuration

### 7.1 Required Google OAuth 2.0 Setup

1. **Google Cloud Console Configuration**
   - Create OAuth 2.0 Client ID
   - Configure authorized redirect URIs
   - Enable Google+ API

2. **Required Scopes**
   - `openid` - OpenID Connect authentication
   - `email` - User's email address
   - `profile` - User's basic profile info

3. **Environment Variables**
   ```
   GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
   GOOGLE_CLIENT_SECRET=your-client-secret
   GOOGLE_REDIRECT_URI=http://localhost:8080/api/v1/auth/google/callback
   ```

### 7.2 OIDC Flow Implementation

1. **Authorization Flow**
   - Generate state token for CSRF protection
   - Redirect to Google authorization endpoint
   - Handle callback with authorization code
   - Exchange code for tokens

2. **Token Validation**
   - Verify ID token signature
   - Check token audience and issuer
   - Validate token expiration
   - Extract user claims

3. **User Account Handling**
   - Check if user exists by email
   - Auto-create account if new user
   - Link Google account to existing user
   - Update profile information from Google

## 8. Security Considerations

### 8.1 Token Security

1. **JWT Security**
   - Use strong secret keys (minimum 256 bits)
   - Implement key rotation strategy
   - Short access token lifetime (15-30 minutes)
   - Secure token transmission (HTTPS only)

2. **Refresh Token Security**
   - One-time use refresh tokens
   - Refresh token rotation
   - Detect and prevent token replay attacks
   - Secure storage recommendations for clients

### 8.2 Authentication Security

1. **Rate Limiting**
   - Login attempt rate limiting per IP
   - Progressive delays on failed attempts
   - Account lockout after threshold
   - CAPTCHA integration for suspicious activity

2. **Session Security**
   - Secure session token generation
   - IP address validation (optional)
   - Device fingerprinting
   - Suspicious activity detection

### 8.3 OIDC Security

1. **State Parameter**
   - CSRF protection with state parameter
   - State expiration (5-10 minutes)
   - One-time use validation

2. **Token Validation**
   - Verify Google's public keys
   - Check token signatures
   - Validate all required claims
   - Prevent token substitution attacks

## 9. Implementation Details

### 9.1 Middleware Updates

```go
// Updated auth middleware to support JWT
func JWTAuthMiddleware(jwtService *JWTService) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractToken(c)
        if token == "" {
            c.JSON(401, gin.H{"error": "Missing authorization token"})
            c.Abort()
            return
        }
        
        claims, err := jwtService.ValidateToken(token)
        if err != nil {
            c.JSON(401, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }
        
        // Check blacklist
        if jwtService.IsTokenBlacklisted(claims.JTI) {
            c.JSON(401, gin.H{"error": "Token has been revoked"})
            c.Abort()
            return
        }
        
        c.Set("user_id", claims.UserID)
        c.Set("user_email", claims.Email)
        c.Set("user_roles", claims.Roles)
        c.Set("user_permissions", claims.Permissions)
        c.Next()
    }
}
```

### 9.2 Token Service Interface

```go
type TokenService interface {
    GenerateTokenPair(user *models.User) (*TokenPair, error)
    ValidateAccessToken(token string) (*JWTClaims, error)
    ValidateRefreshToken(token string) (*JWTClaims, error)
    RefreshTokenPair(refreshToken string) (*TokenPair, error)
    RevokeToken(token string, reason string) error
    RevokeAllUserTokens(userID uuid.UUID) error
    IsTokenBlacklisted(jti string) bool
}
```

## 10. Migration Strategy

### 10.1 Database Migration

```sql
-- Migration 11: Add authentication tables
CREATE TABLE sessions (...);
CREATE TABLE token_blacklist (...);

-- Add indexes for performance
CREATE INDEX idx_sessions_user_active ON sessions(user_id, is_active);
CREATE INDEX idx_token_blacklist_cleanup ON token_blacklist(expires_at);
```

### 10.2 API Migration

1. **Phase 1**: Implement JWT authentication alongside existing API key auth
2. **Phase 2**: Migrate existing integrations to JWT
3. **Phase 3**: Deprecate API key authentication (optional)

## 11. Testing Requirements

### 11.1 Unit Tests

- JWT token generation and validation
- Password authentication flow
- Google OIDC token validation
- Session management operations
- Token blacklist functionality

### 11.2 Integration Tests

- Complete login/logout flow
- Token refresh mechanism
- Google OIDC authentication flow
- Session revocation
- Rate limiting behavior

### 11.3 Security Tests

- Token expiration handling
- Invalid token rejection
- Blacklisted token blocking
- CSRF protection validation
- SQL injection prevention

## 12. Performance Considerations

### 12.1 Caching Strategy

- Cache user permissions in Redis
- Cache blacklisted tokens
- Session data caching
- Google public keys caching

### 12.2 Database Optimization

- Indexed queries for token lookup
- Periodic cleanup of expired sessions
- Efficient blacklist checking
- Connection pooling for auth queries

## 13. Monitoring and Logging

### 13.1 Authentication Metrics

- Login success/failure rates
- Token generation rate
- Session duration statistics
- Authentication method distribution

### 13.2 Security Monitoring

- Failed login attempts
- Suspicious login patterns
- Token abuse detection
- Geographic anomaly detection

## 14. Error Handling

### 14.1 Authentication Errors

- `401 Unauthorized`: Invalid credentials
- `403 Forbidden`: Account locked/suspended
- `429 Too Many Requests`: Rate limit exceeded
- `400 Bad Request`: Invalid request format

### 14.2 Token Errors

- `TOKEN_EXPIRED`: Access token expired
- `INVALID_TOKEN`: Malformed or invalid token
- `TOKEN_REVOKED`: Token has been blacklisted
- `INVALID_REFRESH`: Refresh token invalid/expired

## 15. Client Implementation Guide

### 15.1 Token Storage

```javascript
// Recommended client-side storage
const tokenStorage = {
    setTokens: (accessToken, refreshToken) => {
        // Store access token in memory
        sessionStorage.setItem('access_token', accessToken);
        // Store refresh token in httpOnly cookie (server-set)
        // or secure localStorage with encryption
    },
    
    getAccessToken: () => {
        return sessionStorage.getItem('access_token');
    },
    
    clearTokens: () => {
        sessionStorage.removeItem('access_token');
        // Clear refresh token cookie
    }
};
```

### 15.2 Auto-Refresh Implementation

```javascript
// Axios interceptor for token refresh
axios.interceptors.response.use(
    response => response,
    async error => {
        if (error.response?.status === 401) {
            const newTokens = await refreshAccessToken();
            if (newTokens) {
                // Retry original request
                return axios(error.config);
            }
        }
        return Promise.reject(error);
    }
);
```

## 16. Acceptance Criteria

### 16.1 Authentication Requirements
- ✅ Users can login with email/password
- ✅ Users receive JWT tokens on successful login
- ✅ Access tokens expire and can be refreshed
- ✅ Users can logout and invalidate tokens
- ✅ Multiple sessions per user supported

### 16.2 Google OIDC Requirements
- ✅ Users can login via Google
- ✅ New users auto-registered from Google
- ✅ Existing users can link Google accounts
- ✅ Profile synced from Google

### 16.3 Security Requirements
- ✅ Tokens properly signed and validated
- ✅ Refresh token rotation implemented
- ✅ Token blacklist functioning
- ✅ Rate limiting on login endpoints
- ✅ CSRF protection for OIDC flow

This PRD provides a comprehensive foundation for implementing a secure and scalable authentication system with both password and Google OIDC support.