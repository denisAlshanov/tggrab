# Product Requirements Document: CORS Configuration

## Document Information
- **Title**: CORS Configuration via Environment Variables
- **Version**: 1.0
- **Date**: January 7, 2025
- **Author**: Platform Team
- **Status**: Draft

## Executive Summary

This PRD outlines the requirements for implementing Cross-Origin Resource Sharing (CORS) configuration in the St. Planer API. The goal is to provide flexible CORS configuration through environment variables, enabling the API to be deployed in various environments with appropriate cross-origin access controls.

## Background

Cross-Origin Resource Sharing (CORS) is a security feature implemented by web browsers that restricts web pages from making requests to a different domain than the one serving the web page. For the St. Planer API to be consumed by web applications hosted on different domains, proper CORS configuration is essential.

Currently, the API lacks configurable CORS support, which limits its ability to serve web applications deployed on different domains or during development with different ports.

## Objectives

1. **Flexible Deployment**: Enable API deployment across different environments with appropriate CORS settings
2. **Development Support**: Allow local development with various frontend frameworks and ports
3. **Security**: Provide granular control over which origins can access the API
4. **Production Ready**: Support secure CORS configuration for production environments
5. **Environment-Driven**: Configure CORS entirely through environment variables

## Scope

### In Scope
- CORS middleware implementation with environment variable configuration
- Support for allowed origins, methods, headers, and credentials
- Development and production CORS profiles
- CORS configuration validation and defaults
- Documentation for CORS environment variables

### Out of Scope
- Dynamic CORS configuration during runtime
- Database-stored CORS configuration
- Per-endpoint CORS configuration
- CORS preflight request optimization

## Functional Requirements

### 1. Environment Variable Configuration

#### 1.1 Core CORS Settings
The following environment variables must be supported:

- **CORS_ENABLED**: Enable/disable CORS middleware (default: true)
- **CORS_ALLOWED_ORIGINS**: Comma-separated list of allowed origins
- **CORS_ALLOWED_METHODS**: Comma-separated list of allowed HTTP methods
- **CORS_ALLOWED_HEADERS**: Comma-separated list of allowed headers
- **CORS_EXPOSED_HEADERS**: Comma-separated list of headers exposed to client
- **CORS_ALLOW_CREDENTIALS**: Allow credentials in CORS requests (default: true)
- **CORS_MAX_AGE**: Preflight cache duration in seconds (default: 3600)

#### 1.2 Development vs Production Profiles
- **CORS_PROFILE**: Set to "development" or "production" for predefined configurations
- Development profile: Permissive settings for local development
- Production profile: Restrictive settings for production security

### 2. Default Configurations

#### 2.1 Development Profile (CORS_PROFILE=development)
```
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001,http://localhost:8080,http://127.0.0.1:3000,http://127.0.0.1:3001,http://127.0.0.1:8080
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS,PATCH
CORS_ALLOWED_HEADERS=Origin,Content-Type,Accept,Authorization,X-Requested-With,X-API-Key
CORS_EXPOSED_HEADERS=X-Total-Count,X-Page,X-Per-Page
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=86400
```

#### 2.2 Production Profile (CORS_PROFILE=production)
```
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=https://app.stplaner.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Origin,Content-Type,Accept,Authorization
CORS_EXPOSED_HEADERS=X-Total-Count
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=3600
```

#### 2.3 Custom Configuration
When no profile is set or CORS_PROFILE=custom, individual environment variables take precedence with secure defaults.

### 3. CORS Middleware Features

#### 3.1 Origin Validation
- Support for exact origin matching
- Support for wildcard origins (e.g., "*.stplaner.com")
- Support for special value "*" (all origins) - only in development
- Support for "null" origin for local file development

#### 3.2 Method Validation
- Validate allowed HTTP methods
- Support for common REST methods (GET, POST, PUT, DELETE, PATCH, OPTIONS)
- Automatic OPTIONS handling for preflight requests

#### 3.3 Header Validation
- Support for standard CORS headers
- Support for custom application headers
- Automatic handling of simple vs non-simple requests

### 4. Security Requirements

#### 4.1 Production Security
- Reject wildcard origins (*) in production profile
- Validate SSL/TLS origins in production
- Limit exposed headers to necessary ones only
- Reasonable max-age limits

#### 4.2 Configuration Validation
- Validate origin URLs format
- Warn about insecure configurations
- Fail fast on invalid configurations

## Technical Requirements

### 1. Implementation Details

#### 1.1 Configuration Structure
```go
type CORSConfig struct {
    Enabled          bool
    AllowedOrigins   []string
    AllowedMethods   []string
    AllowedHeaders   []string
    ExposedHeaders   []string
    AllowCredentials bool
    MaxAge           int
    Profile          string
}
```

#### 1.2 Middleware Integration
```go
// Add CORS middleware before authentication
engine.Use(middleware.CORSMiddleware(&cfg.CORS))
engine.Use(middleware.CorrelationIDMiddleware())
```

#### 1.3 Environment Variable Loading
```go
func LoadCORSConfig() *CORSConfig {
    profile := getEnv("CORS_PROFILE", "custom")
    
    switch profile {
    case "development":
        return getDevelopmentCORSConfig()
    case "production":
        return getProductionCORSConfig()
    default:
        return getCustomCORSConfig()
    }
}
```

### 2. Configuration Validation

#### 2.1 Origin Validation
- Validate URL format for each origin
- Check for https:// in production origins
- Warn about localhost in production
- Validate wildcard patterns

#### 2.2 Method Validation
- Validate against known HTTP methods
- Ensure OPTIONS is included when needed
- Warn about potentially dangerous methods

#### 2.3 Header Validation
- Validate header name format
- Check for security-sensitive headers
- Ensure required headers are included

### 3. Logging and Monitoring

#### 3.1 Configuration Logging
- Log active CORS configuration on startup
- Log CORS validation results
- Warn about potentially insecure settings

#### 3.2 Request Logging
- Log blocked CORS requests (debug level)
- Track CORS preflight requests
- Monitor CORS-related errors

## Environment Variables Reference

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `CORS_ENABLED` | boolean | true | Enable/disable CORS middleware |
| `CORS_PROFILE` | string | custom | Predefined configuration profile |
| `CORS_ALLOWED_ORIGINS` | string | - | Comma-separated allowed origins |
| `CORS_ALLOWED_METHODS` | string | GET,POST,PUT,DELETE,OPTIONS | Comma-separated allowed methods |
| `CORS_ALLOWED_HEADERS` | string | Origin,Content-Type,Accept,Authorization | Comma-separated allowed headers |
| `CORS_EXPOSED_HEADERS` | string | - | Comma-separated exposed headers |
| `CORS_ALLOW_CREDENTIALS` | boolean | true | Allow credentials in requests |
| `CORS_MAX_AGE` | int | 3600 | Preflight cache duration (seconds) |

## Usage Examples

### Development Environment
```bash
# Permissive development setup
CORS_PROFILE=development
```

### Production Environment
```bash
# Secure production setup
CORS_PROFILE=production
CORS_ALLOWED_ORIGINS=https://app.stplaner.com,https://admin.stplaner.com
```

### Custom Environment
```bash
# Custom configuration
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=https://myapp.com,https://staging.myapp.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE
CORS_ALLOWED_HEADERS=Origin,Content-Type,Authorization
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=7200
```

### Disabled CORS
```bash
# Disable CORS for same-origin only deployments
CORS_ENABLED=false
```

## Migration Strategy

### Phase 1: Implementation (Week 1)
1. Implement CORS middleware with environment configuration
2. Add configuration loading and validation
3. Integrate middleware into router
4. Add comprehensive logging

### Phase 2: Testing (Week 2)
1. Unit tests for CORS configuration
2. Integration tests for CORS behavior
3. Test various origin scenarios
4. Validate security configurations

### Phase 3: Documentation (Week 3)
1. Update environment variable documentation
2. Create deployment guides for different environments
3. Document common CORS scenarios
4. Update Docker and deployment configurations

### Phase 4: Deployment (Week 4)
1. Deploy to staging with CORS enabled
2. Test with frontend applications
3. Deploy to production with secure configuration
4. Monitor CORS behavior and logs

## Success Metrics

1. **Configuration Flexibility**: 100% of CORS settings configurable via environment
2. **Development Experience**: Zero CORS-related blocking in local development
3. **Security**: No wildcard origins in production deployments
4. **Performance**: < 1ms overhead for CORS processing
5. **Reliability**: Zero CORS-related service failures

## Testing Requirements

### 1. Unit Tests
- Configuration loading and validation
- Middleware behavior with different settings
- Origin, method, and header validation
- Profile-based configuration

### 2. Integration Tests
- Preflight request handling
- Simple vs complex request handling
- Credential handling
- Error responses for blocked requests

### 3. Security Tests
- Attempt access from unauthorized origins
- Test with various header combinations
- Validate credential handling
- Test production security restrictions

## Documentation Updates

1. Environment variable reference documentation
2. CORS configuration guide
3. Troubleshooting guide for CORS issues
4. Examples for common deployment scenarios
5. Security best practices for CORS

## Risks and Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Overly permissive development config | Medium | Clear documentation and warnings |
| Production misconfiguration | High | Validation and safe defaults |
| Performance impact | Low | Efficient middleware implementation |
| Breaking existing integrations | Medium | Backward compatible defaults |

## Timeline

- **Week 1**: Implementation and basic testing
- **Week 2**: Comprehensive testing and validation
- **Week 3**: Documentation and deployment preparation
- **Week 4**: Production deployment and monitoring
- **Total Duration**: 4 weeks

## Approval

- [ ] Engineering Lead
- [ ] DevOps Team
- [ ] Security Team
- [ ] Product Manager
- [ ] CTO

## Future Considerations

1. **Dynamic CORS Configuration**: Runtime configuration updates via API
2. **Per-Endpoint CORS**: Different CORS settings for different endpoints
3. **CORS Analytics**: Detailed monitoring and analytics for CORS requests
4. **Geographic Restrictions**: CORS settings based on geographic location