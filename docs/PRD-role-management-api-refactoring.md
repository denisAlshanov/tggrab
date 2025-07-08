# Product Requirements Document: Role Management API Refactoring

## Document Information
- **Title**: Role Management API Refactoring
- **Version**: 1.0
- **Date**: January 8, 2025
- **Author**: API Team
- **Status**: Draft

## Executive Summary

This PRD outlines the requirements for refactoring the role management API endpoints in the St. Planer system. The goal is to align the API with RESTful best practices, simplify the request/response structures, and provide clearer separation between role management and user assignment operations.

## Background

The current role management API has several design issues:
- Non-RESTful endpoint naming (e.g., `/roles/add` instead of `/roles`)
- Inconsistent request methods and URL patterns
- Complex request payloads for simple operations
- Missing role-to-user assignment endpoints

This refactoring will improve API usability, maintainability, and adherence to REST principles.

## Objectives

1. **RESTful Design**: Align endpoints with REST conventions
2. **Separation of Concerns**: Separate role management from user assignment
3. **Simplified Payloads**: Reduce complexity of request bodies
4. **Consistent Patterns**: Use consistent URL patterns and HTTP methods
5. **Backward Compatibility**: Provide migration path for existing clients

## Scope

### In Scope
- Refactoring all role management endpoints
- Implementing new role-to-user assignment endpoints
- Updating request/response structures
- Maintaining existing business logic
- Updating API documentation

### Out of Scope
- Changes to authentication endpoints
- Modifications to user management endpoints
- Changes to underlying database schema
- Performance optimizations
- UI/Frontend changes

## Functional Requirements

### 1. Role Creation Endpoint

#### Current
- **Endpoint**: `POST /api/v1/roles/add`
- **Complex Request**: Includes metadata, status options

#### New
- **Endpoint**: `POST /api/v1/roles`
- **Request Body**:
```json
{
  "name": "string",
  "description": "string",
  "permissions": [
    "string"
  ]
}
```
- **Behavior**: 
  - Automatically set status to "active" on creation
  - Validate role name uniqueness
  - Validate permissions format
  - Return created role without user count

### 2. Role Deletion Endpoint

#### Current
- **Endpoint**: `DELETE /api/v1/roles/delete`
- **Request Body**: Contains role_id

#### New
- **Endpoint**: `DELETE /api/v1/roles/{role_id}`
- **Request Body**:
```json
{
  "force": "boolean"
}
```
- **Behavior**:
  - `force: false` (default): Soft delete (set status to inactive)
  - `force: true`: Hard delete (remove from database)
  - Prevent deletion if role has assigned users (unless force=true)
  - Return appropriate confirmation message

### 3. Role Update Endpoint

#### Current
- **Endpoint**: `PUT /api/v1/roles/update`
- **Complex Request**: Includes status, metadata

#### New
- **Endpoint**: `PUT /api/v1/roles/{role_id}`
- **Request Body**:
```json
{
  "name": "string",
  "description": "string",
  "permissions": [
    "string"
  ]
}
```
- **Behavior**:
  - Only update provided fields
  - Name validation if changed
  - Cannot update status through this endpoint
  - Return updated role without user count

### 4. Get Role Information Endpoint

#### Current
- **Endpoint**: `GET /api/v1/roles/info/{role_id}`
- **Response**: Includes user count

#### New
- **Endpoint**: `GET /api/v1/roles/{role_id}`
- **Response**: Role information without user count
- **Behavior**:
  - Return basic role information
  - Exclude user count information
  - Include timestamps and status

### 5. List Roles Endpoint

#### Current
- **Endpoint**: `POST /api/v1/roles/list`
- **Method**: POST with filters in body

#### New
- **Endpoint**: `GET /api/v1/roles`
- **Query Parameters**:
  - `page`: Page number (default: 1)
  - `limit`: Items per page (default: 20, max: 100)
  - `status`: Filter by status (active/inactive)
  - `search`: Search by name or description
  - `sort`: Sort field (name, created_at, updated_at)
  - `order`: Sort order (asc/desc)
- **Behavior**:
  - Return paginated role list
  - Support filtering and sorting
  - Exclude user count information

### 6. Add Role to User Endpoint

#### New Endpoint
- **Endpoint**: `PUT /api/v1/roles/{role_id}/users/{user_id}`
- **Request Body**: None
- **Behavior**:
  - Add specified user to role
  - Validate role and user exist
  - Prevent duplicate role assignment
  - Return success confirmation

## API Endpoint Summary

| Operation | Old Endpoint | New Endpoint | Method |
|-----------|-------------|--------------|---------| 
| Create Role | `/api/v1/roles/add` | `/api/v1/roles` | POST |
| Delete Role | `/api/v1/roles/delete` | `/api/v1/roles/{role_id}` | DELETE |
| Update Role | `/api/v1/roles/update` | `/api/v1/roles/{role_id}` | PUT |
| Get Role | `/api/v1/roles/info/{role_id}` | `/api/v1/roles/{role_id}` | GET |
| List Roles | `/api/v1/roles/list` | `/api/v1/roles` | GET |
| Add User to Role | N/A | `/api/v1/roles/{role_id}/users/{user_id}` | PUT |

## Request/Response Examples

### 1. Create Role
**Request**:
```http
POST /api/v1/roles
Content-Type: application/json
Authorization: Bearer {token}

{
  "name": "moderator",
  "description": "Content moderation role",
  "permissions": [
    "content:read",
    "content:moderate",
    "users:view"
  ]
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "moderator",
    "description": "Content moderation role",
    "permissions": [
      "content:read",
      "content:moderate", 
      "users:view"
    ],
    "status": "active",
    "created_at": "2025-01-08T10:00:00Z",
    "updated_at": "2025-01-08T10:00:00Z"
  }
}
```

### 2. Delete Role (Soft)
**Request**:
```http
DELETE /api/v1/roles/{role_id}
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
  "message": "Role deactivated successfully"
}
```

### 3. Update Role
**Request**:
```http
PUT /api/v1/roles/{role_id}
Content-Type: application/json
Authorization: Bearer {token}

{
  "name": "senior_moderator",
  "description": "Senior content moderation role",
  "permissions": [
    "content:read",
    "content:moderate",
    "content:delete",
    "users:view",
    "users:manage"
  ]
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "senior_moderator",
    "description": "Senior content moderation role",
    "permissions": [
      "content:read",
      "content:moderate",
      "content:delete",
      "users:view",
      "users:manage"
    ],
    "status": "active",
    "created_at": "2025-01-08T10:00:00Z",
    "updated_at": "2025-01-08T11:00:00Z"
  }
}
```

### 4. List Roles
**Request**:
```http
GET /api/v1/roles?page=1&limit=10&status=active&sort=name&order=asc
Authorization: Bearer {token}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "roles": [
      {
        "id": "uuid",
        "name": "admin",
        "description": "Administrator role",
        "permissions": ["*"],
        "status": "active",
        "created_at": "2025-01-08T10:00:00Z"
      },
      {
        "id": "uuid",
        "name": "viewer",
        "description": "Read-only access",
        "permissions": ["content:read"],
        "status": "active",
        "created_at": "2025-01-08T10:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total": 25,
      "total_pages": 3
    }
  }
}
```

### 5. Add User to Role
**Request**:
```http
PUT /api/v1/roles/{role_id}/users/{user_id}
Authorization: Bearer {token}
```

**Response**:
```json
{
  "success": true,
  "message": "User added to role successfully"
}
```

## Technical Requirements

### 1. URL Parameter Validation
- Role ID and User ID must be valid UUIDs
- Return 400 Bad Request for invalid formats
- Return 404 Not Found for non-existent resources

### 2. Request Validation
- Role name must be unique and 1-100 characters
- Description must be 0-500 characters
- Permissions must be valid permission strings
- All string fields must be trimmed

### 3. Error Handling
- Consistent error response format
- Appropriate HTTP status codes
- Detailed error messages for development
- Generic messages for production

### 4. Authorization
- All endpoints require JWT authentication
- Role-based access control:
  - Regular users: Cannot access role management endpoints
  - Admins: Full access to role operations except user assignment
  - Super admins: Full access to all role operations

### 5. Business Rules
- Cannot delete role if users are assigned (unless force=true)
- Cannot create role with duplicate name
- Default "viewer" role cannot be deleted
- System roles (admin, viewer) have restricted permissions updates

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
- Permission validation

### 2. Integration Tests
- Full role lifecycle (create, update, delete)
- User assignment and removal
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

1. **Permission Validation**: Ensure proper permission string validation
2. **Authorization**: Verify proper access controls
3. **Input Validation**: Prevent injection attacks
4. **Rate Limiting**: Apply appropriate limits to new endpoints
5. **Audit Logging**: Log all role management operations

## Risks and Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing clients | High | Parallel implementation with gradual migration |
| Data inconsistency | Medium | Thorough testing and validation |
| Performance degradation | Low | Load testing and optimization |
| Security vulnerabilities | High | Security review and penetration testing |
| Role permission conflicts | Medium | Comprehensive permission validation |

## Dependencies

1. **Frontend**: Updates required for new API endpoints
2. **Mobile Apps**: API client updates needed
3. **Documentation**: API docs must be updated
4. **Monitoring**: Update alerts and dashboards
5. **User Management**: Coordination with user API changes

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

## Appendix

### Permission String Format
Permissions follow the format: `resource:action`

Examples:
- `content:read` - Read content
- `content:write` - Create/update content
- `content:delete` - Delete content
- `users:view` - View user information
- `users:manage` - Manage user accounts
- `roles:manage` - Manage roles
- `*` - All permissions (admin only)

### Default Roles
- **admin**: Full system access (`*`)
- **moderator**: Content moderation (`content:*`, `users:view`)
- **viewer**: Read-only access (`content:read`)

### Error Response Format
```json
{
  "error": "ERROR_CODE",
  "message": "Human readable error message",
  "details": "Additional error details (development only)",
  "timestamp": "2025-01-08T10:00:00Z",
  "correlation_id": "uuid"
}
```