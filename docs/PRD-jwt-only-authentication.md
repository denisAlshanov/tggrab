# Product Requirements Document: JWT-Only Authentication

## Document Information
- **Title**: JWT-Only Authentication for API Endpoints
- **Version**: 1.0
- **Date**: January 7, 2025
- **Author**: Security Team
- **Status**: Draft

## Executive Summary

This PRD outlines the requirements for implementing JWT-only authentication across all API endpoints in the St. Planer system. The goal is to enhance security by removing API key authentication for all handlers except authentication endpoints, ensuring that all API access requires a valid JWT token obtained through proper authentication.

## Background

Currently, the system supports hybrid authentication (both JWT and API key). This creates potential security vulnerabilities where API keys could be used to bypass user-specific authentication and authorization checks. By enforcing JWT-only authentication, we ensure that all API access is tied to authenticated user sessions with proper role-based access control.

## Objectives

1. **Enhanced Security**: Eliminate API key access to protected endpoints
2. **Session Management**: Ensure all API access is tied to valid user sessions
3. **Audit Trail**: Enable proper user activity tracking through JWT claims
4. **Consistent Authorization**: Apply role-based access control uniformly

## Scope

### In Scope
- Remove API key authentication from all
- Implement JWT-only middleware for protected endpoints
- Update error handling for authentication failures
- Maintain backward compatibility for authentication endpoints

### Out of Scope
- Changes to the authentication flow itself
- Modifications to JWT token structure
- Updates to user roles and permissions system

## Functional Requirements

### 1. Authentication Middleware Changes

#### 1.1 JWT-Only Middleware
- **Requirement**: Create a new `JWTOnlyMiddleware` that only accepts JWT tokens
- **Details**:
  - Must validate JWT tokens in Authorization header
  - Must reject requests with API keys
  - Must return 401 Unauthorized for missing/invalid tokens
  - Must extract and validate user claims from JWT

#### 1.2 Public Endpoints
- **Requirement**: Authentication endpoints remain publicly accessible
- **Endpoints**:
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/refresh`
  - `POST /api/v1/auth/google/callback`
  - `GET /health`

### 2. Protected Endpoint Groups

All endpoints under these paths must require JWT authentication:

#### 2.1 User Management
- `/api/v1/users/*`
- Required: Valid JWT with appropriate user management permissions

#### 2.2 Media Management
- `/api/v1/media/*`
- Required: Valid JWT with media access permissions

#### 2.3 Show Management
- `/api/v1/shows/*`
- Required: Valid JWT with show management permissions

#### 2.4 Event Management
- `/api/v1/events/*`
- Required: Valid JWT with event management permissions

#### 2.5 Guest Management
- `/api/v1/guests/*`
- Required: Valid JWT with guest management permissions

#### 2.6 Block Management
- `/api/v1/blocks/*`
- Required: Valid JWT with block management permissions

#### 2.7 Role Management
- `/api/v1/roles/*`
- Required: Valid JWT with admin permissions

### 3. Error Handling

#### 3.1 Authentication Errors
- **Missing Token**: Return 401 with message "Authentication required"
- **Invalid Token**: Return 401 with message "Invalid authentication token"
- **Expired Token**: Return 401 with message "Token expired"
- **API Key Attempt**: Return 401 with message "API key authentication not allowed"

#### 3.2 Authorization Errors
- **Insufficient Permissions**: Return 403 with message "Insufficient permissions"

## Technical Requirements

### 1. Implementation Details

#### 1.1 Middleware Structure
```go
func JWTOnlyMiddleware(jwtService *auth.JWTService, sessionService *auth.SessionService) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Extract token from Authorization header
        // 2. Validate token format (Bearer scheme)
        // 3. Verify JWT signature and claims
        // 4. Check session validity
        // 5. Set user context
        // 6. Continue to handler
    }
}
```

#### 1.2 Router Configuration
```go
// Public endpoints (no auth required)
public := engine.Group("/api/v1/auth")
public.POST("/login", authHandler.Login)
public.POST("/register", authHandler.Register)
public.POST("/refresh", authHandler.RefreshToken)
public.POST("/google/callback", authHandler.GoogleCallback)

// Protected endpoints (JWT required)
api := engine.Group("/api/v1")
api.Use(middleware.JWTOnlyMiddleware(jwtService, sessionService))
api.Use(middleware.RateLimitMiddleware(&cfg.API))

// Register protected routes
api.POST("/users/add", userHandler.CreateUser)
api.PUT("/users/update", userHandler.UpdateUser)
// ... other protected endpoints
```

### 2. Security Considerations

#### 2.1 Token Validation
- Verify JWT signature using configured secret
- Check token expiration
- Validate issuer claim
- Verify session is still active

#### 2.2 Rate Limiting
- Apply rate limiting after authentication
- Track by authenticated user ID, not IP

#### 2.3 Audit Logging
- Log all authentication attempts
- Track failed authentication reasons
- Monitor for suspicious patterns

## Migration Strategy

### Phase 1: Implementation (Week 1)
1. Implement JWTOnlyMiddleware
2. Update router configuration
3. Test all endpoints

### Phase 2: Testing (Week 2)
1. Unit tests for middleware
2. Integration tests for all endpoints
3. Security testing

### Phase 3: Documentation (Week 3)
1. Update API documentation
2. Update Swagger specifications
3. Create migration guide for API consumers

### Phase 4: Deployment (Week 4)
1. Deploy to staging environment
2. Monitor for issues
3. Deploy to production with feature flag

## Rollback Plan

1. Feature flag to toggle between hybrid and JWT-only authentication
2. Ability to quickly revert to hybrid authentication if issues arise
3. Clear communication to API consumers about changes

## Success Metrics

1. **Security**: 100% of protected endpoints require JWT authentication
2. **Compatibility**: Zero breaking changes for properly authenticated requests
3. **Performance**: No degradation in API response times
4. **Reliability**: < 0.01% false-positive authentication failures

## Testing Requirements

### 1. Unit Tests
- JWT validation logic
- Middleware behavior
- Error handling

### 2. Integration Tests
- All endpoint authentication
- Token expiration handling
- Session invalidation

### 3. Security Tests
- Attempt API key access on protected endpoints
- Invalid token formats
- Expired token handling
- Replay attack prevention

## Documentation Updates

1. API documentation must clearly indicate authentication requirements
2. Swagger specs must include security schemes
3. Error response examples for authentication failures
4. Migration guide for existing API consumers

## Risks and Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing integrations | High | Phased rollout with feature flags |
| Performance impact | Medium | Optimize JWT validation, implement caching |
| Complex troubleshooting | Medium | Enhanced logging and monitoring |
| Token management overhead | Low | Clear documentation and examples |

## Timeline

- **Week 1**: Implementation
- **Week 2**: Testing
- **Week 3**: Documentation
- **Week 4**: Deployment
- **Total Duration**: 4 weeks

## Approval

- [ ] Engineering Lead
- [ ] Security Team
- [ ] Product Manager
- [ ] CTO