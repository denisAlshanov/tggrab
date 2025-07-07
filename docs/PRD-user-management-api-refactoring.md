# Product Requirements Document: User Management API Refactoring

## Document Information
- **Title**: User Management API Refactoring
- **Version**: 1.0
- **Date**: January 7, 2025
- **Author**: API Team
- **Status**: Draft

## Executive Summary

This PRD outlines the requirements for refactoring the user management API endpoints in the St. Planer system. The goal is to align the API with RESTful best practices, simplify the request/response structures, and provide clearer separation between user management and role assignment operations.

## Background

The current user management API has several design issues:
- Non-RESTful endpoint naming (e.g., `/users/add` instead of `/users`)
- Mixed concerns in endpoints (user creation includes role assignment)
- Inconsistent request methods and URL patterns
- Overly complex request payloads for simple operations

This refactoring will improve API usability, maintainability, and adherence to REST principles.

## Objectives

1. **RESTful Design**: Align endpoints with REST conventions
2. **Separation of Concerns**: Separate user management from role management
3. **Simplified Payloads**: Reduce complexity of request bodies
4. **Consistent Patterns**: Use consistent URL patterns and HTTP methods
5. **Backward Compatibility**: Provide migration path for existing clients

## Scope

### In Scope
- Refactoring all user management endpoints
- Implementing new role assignment endpoints
- Updating request/response structures
- Maintaining existing business logic
- Updating API documentation

### Out of Scope
- Changes to authentication endpoints
- Modifications to role management endpoints
- Changes to underlying database schema
- Performance optimizations
- UI/Frontend changes

## Functional Requirements

### 1. User Creation Endpoint

#### Current
- **Endpoint**: `POST /api/v1/users/add`
- **Complex Request**: Includes roles, metadata, OIDC options

#### New
- **Endpoint**: `POST /api/v1/users`
- **Request Body**:
```json
{
  "email": "string",
  "name": "string",
  "surname": "string",
  "password": "string"
}
```
- **Behavior**: 
  - Automatically assign "viewer" role on creation
  - Validate email format (must be Gmail)
  - Validate password strength
  - Return created user without roles

### 2. User Deletion Endpoint

#### Current
- **Endpoint**: `DELETE /api/v1/users/delete`
- **Request Body**: Contains user_id

#### New
- **Endpoint**: `DELETE /api/v1/users/{user_id}`
- **Request Body**:
```json
{
  "force": "boolean"
}
```
- **Behavior**:
  - `force: false` (default): Soft delete (set status to inactive)
  - `force: true`: Hard delete (remove from database)
  - Return appropriate confirmation message

### 3. User Update Endpoint

#### Current
- **Endpoint**: `PUT /api/v1/users/update`
- **Complex Request**: Includes roles, status, metadata

#### New
- **Endpoint**: `PUT /api/v1/users/{user_id}`
- **Request Body**:
```json
{
  "email": "string",
  "name": "string",
  "surname": "string"
}
```
- **Behavior**:
  - Only update provided fields
  - Email validation if changed
  - Cannot update roles through this endpoint
  - Return updated user without roles

### 4. Get User Information Endpoint

#### Current
- **Endpoint**: `GET /api/v1/users/info/{user_id}`
- **Response**: Includes roles

#### New
- **Endpoint**: `GET /api/v1/users/{user_id}`
- **Response**: User information without roles
- **Behavior**:
  - Return basic user information
  - Exclude role information
  - Include timestamps and status

### 5. List Users Endpoint

#### Current
- **Endpoint**: `POST /api/v1/users/list`
- **Method**: POST with filters in body

#### New
- **Endpoint**: `GET /api/v1/users`
- **Query Parameters**:
  - `page`: Page number (default: 1)
  - `limit`: Items per page (default: 20, max: 100)
  - `status`: Filter by status (active/inactive)
  - `search`: Search by name or email
  - `sort`: Sort field (name, email, created_at)
  - `order`: Sort order (asc/desc)
- **Behavior**:
  - Return paginated user list
  - Support filtering and sorting
  - Exclude role information

### 6. Add Role to User Endpoint

#### New Endpoint
- **Endpoint**: `PUT /api/v1/users/{user_id}/roles/{role_id}`
- **Request Body**: None
- **Behavior**:
  - Add specified role to user
  - Validate role exists
  - Prevent duplicate role assignment
  - Return success confirmation

### 7. Remove Role from User Endpoint

#### New Endpoint
- **Endpoint**: `DELETE /api/v1/users/{user_id}/roles/{role_id}`
- **Request Body**: None
- **Behavior**:
  - Remove specified role from user
  - Validate user has the role
  - Prevent removing last role if required
  - Return success confirmation

## API Endpoint Summary

| Operation | Old Endpoint | New Endpoint | Method |
|-----------|-------------|--------------|---------|
| Create User | `/api/v1/users/add` | `/api/v1/users` | POST |
| Delete User | `/api/v1/users/delete` | `/api/v1/users/{user_id}` | DELETE |
| Update User | `/api/v1/users/update` | `/api/v1/users/{user_id}` | PUT |
| Get User | `/api/v1/users/info/{user_id}` | `/api/v1/users/{user_id}` | GET |
| List Users | `/api/v1/users/list` | `/api/v1/users` | GET |
| Add Role | N/A | `/api/v1/users/{user_id}/roles/{role_id}` | PUT |
| Remove Role | N/A | `/api/v1/users/{user_id}/roles/{role_id}` | DELETE |

## Request/Response Examples

### 1. Create User
**Request**:
```http
POST /api/v1/users
Content-Type: application/json
Authorization: Bearer {token}

{
  "email": "user@gmail.com",
  "name": "John",
  "surname": "Doe",
  "password": "SecurePass123!"
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "email": "user@gmail.com",
    "name": "John",
    "surname": "Doe",
    "status": "active",
    "created_at": "2025-01-07T10:00:00Z",
    "updated_at": "2025-01-07T10:00:00Z"
  }
}
```

### 2. Delete User (Soft)
**Request**:
```http
DELETE /api/v1/users/{user_id}
Content-Type: application/json
Authorization: Bearer {token}

{
  "force": false
}
```

**Response**:
```json
{
  "success": true,
  "message": "User deactivated successfully"
}
```

### 3. Update User
**Request**:
```http
PUT /api/v1/users/{user_id}
Content-Type: application/json
Authorization: Bearer {token}

{
  "name": "Jane",
  "surname": "Smith"
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "email": "user@gmail.com",
    "name": "Jane",
    "surname": "Smith",
    "status": "active",
    "created_at": "2025-01-07T10:00:00Z",
    "updated_at": "2025-01-07T11:00:00Z"
  }
}
```

### 4. List Users
**Request**:
```http
GET /api/v1/users?page=1&limit=10&status=active&sort=name&order=asc
Authorization: Bearer {token}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "users": [
      {
        "id": "uuid",
        "email": "user@gmail.com",
        "name": "John",
        "surname": "Doe",
        "status": "active",
        "created_at": "2025-01-07T10:00:00Z",
        "last_login_at": "2025-01-07T15:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 50,
      "total_pages": 5
    }
  }
}
```

### 5. Add Role to User
**Request**:
```http
PUT /api/v1/users/{user_id}/roles/{role_id}
Authorization: Bearer {token}
```

**Response**:
```json
{
  "success": true,
  "message": "Role added successfully"
}
```

## Technical Requirements

### 1. URL Parameter Validation
- User ID and Role ID must be valid UUIDs
- Return 400 Bad Request for invalid formats
- Return 404 Not Found for non-existent resources

### 2. Request Validation
- Email must be valid Gmail address
- Password must meet security requirements (min 8 chars, etc.)
- Name and surname must be 1-100 characters
- All string fields must be trimmed

### 3. Error Handling
- Consistent error response format
- Appropriate HTTP status codes
- Detailed error messages for development
- Generic messages for production

### 4. Authorization
- All endpoints require JWT authentication
- Role-based access control:
  - Regular users: Can only view/update own profile
  - Admins: Full access to all user operations
  - Super admins: Can manage roles

## Migration Strategy

### Phase 1: Parallel Implementation (Week 1-2)
1. Implement new endpoints alongside existing ones
2. Mark old endpoints as deprecated in documentation
3. Add deprecation headers to old endpoint responses
4. Log usage of deprecated endpoints

### Phase 2: Client Migration (Week 3-4)
1. Update internal clients to use new endpoints
2. Notify external API consumers
3. Provide migration guide with examples
4. Monitor adoption metrics

### Phase 3: Deprecation (Week 5-6)
1. Add warning responses to old endpoints
2. Set sunset date for old endpoints
3. Increase deprecation warning severity
4. Prepare for final removal

### Phase 4: Removal (Week 8)
1. Remove old endpoint implementations
2. Return 410 Gone for old endpoints
3. Update all documentation
4. Final communication to clients

## Success Metrics

1. **API Usage**: 100% migration to new endpoints within 8 weeks
2. **Error Rate**: < 0.1% error rate on new endpoints
3. **Performance**: No degradation in response times
4. **Client Satisfaction**: Positive feedback on simplified API

## Testing Requirements

### 1. Unit Tests
- Input validation for all endpoints
- Business logic for role assignment
- Error handling scenarios

### 2. Integration Tests
- Full user lifecycle (create, update, delete)
- Role assignment and removal
- Pagination and filtering
- Authorization checks

### 3. Migration Tests
- Parallel endpoint functionality
- Deprecation warnings
- Client compatibility

## Documentation Updates

1. **API Reference**: Update all endpoint documentation
2. **Migration Guide**: Step-by-step migration instructions
3. **Examples**: Updated code examples for all languages
4. **Changelog**: Detailed list of changes
5. **Deprecation Timeline**: Clear communication of sunset dates

## Security Considerations

1. **Password Handling**: Ensure passwords are properly hashed
2. **Authorization**: Verify proper access controls
3. **Input Validation**: Prevent injection attacks
4. **Rate Limiting**: Apply appropriate limits to new endpoints
5. **Audit Logging**: Log all user management operations

## Risks and Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing clients | High | Parallel implementation with gradual migration |
| Data inconsistency | Medium | Thorough testing and validation |
| Performance degradation | Low | Load testing and optimization |
| Security vulnerabilities | High | Security review and penetration testing |

## Dependencies

1. **Frontend**: Updates required for new API endpoints
2. **Mobile Apps**: API client updates needed
3. **Documentation**: API docs must be updated
4. **Monitoring**: Update alerts and dashboards

## Timeline

- **Week 1-2**: Implementation of new endpoints
- **Week 3**: Testing and documentation
- **Week 4**: Internal migration
- **Week 5-6**: External client migration
- **Week 7**: Monitoring and fixes
- **Week 8**: Old endpoint removal

## Approval

- [ ] Engineering Lead
- [ ] API Team Lead
- [ ] Security Team
- [ ] Product Manager
- [ ] CTO