# Product Requirements Document: User and Roles Management System

## 1. Overview

This document outlines the requirements for implementing a comprehensive user and roles management system for the St. Planer service. The system will provide user account management with flexible authentication options and role-based access control (RBAC).

## 2. Objectives

- Implement user account management with profile information
- Support multiple authentication methods (password and OIDC)
- Provide role-based access control system
- Create comprehensive API endpoints for user and role management
- Ensure secure authentication and authorization

## 3. System Requirements

### 3.1 User Management

#### User Profile Requirements
- **Name**: Required field for user's first name
- **Surname**: Required field for user's last name
- **Email**: Required Gmail account for identification and communication
- **Authentication**: Support both password-based and OIDC authentication
- **Roles**: Users can have multiple roles assigned
- **Status**: Active/inactive user status management
- **Timestamps**: Creation and update tracking

#### Authentication Methods
1. **Password Authentication**
   - Secure password hashing (bcrypt)
   - Password complexity requirements
   - Password reset functionality

2. **OIDC Authentication**
   - Support for external OIDC providers (Google, Microsoft, etc.)
   - JWT token validation
   - Provider-specific user mapping

### 3.2 Role Management

#### Role Structure
- **Role Name**: Unique identifier for the role
- **Description**: Human-readable description of role purpose
- **Permissions**: Set of permissions associated with the role
- **Status**: Active/inactive role status
- **Timestamps**: Creation and update tracking

#### Permission System
- **Resource-based permissions**: Control access to specific resources
- **Action-based permissions**: Define allowed operations (create, read, update, delete)
- **Hierarchical permissions**: Support for permission inheritance

## 4. Data Models

### 4.1 User Entity

```go
type User struct {
    ID          uuid.UUID              `json:"id" db:"id"`
    Name        string                 `json:"name" db:"name"`
    Surname     string                 `json:"surname" db:"surname"`
    Email       string                 `json:"email" db:"email"`
    PasswordHash *string               `json:"-" db:"password_hash"`
    OIDCProvider *string               `json:"oidc_provider,omitempty" db:"oidc_provider"`
    OIDCSubject  *string               `json:"oidc_subject,omitempty" db:"oidc_subject"`
    Status      UserStatus             `json:"status" db:"status"`
    Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
    CreatedAt   time.Time              `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
    LastLoginAt *time.Time             `json:"last_login_at,omitempty" db:"last_login_at"`
}

type UserStatus string

const (
    UserStatusActive   UserStatus = "active"
    UserStatusInactive UserStatus = "inactive"
    UserStatusPending  UserStatus = "pending"
    UserStatusSuspended UserStatus = "suspended"
)
```

### 4.2 Role Entity

```go
type Role struct {
    ID          uuid.UUID              `json:"id" db:"id"`
    Name        string                 `json:"name" db:"name"`
    Description *string                `json:"description,omitempty" db:"description"`
    Permissions []string               `json:"permissions" db:"permissions"`
    Status      RoleStatus             `json:"status" db:"status"`
    Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
    CreatedAt   time.Time              `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

type RoleStatus string

const (
    RoleStatusActive   RoleStatus = "active"
    RoleStatusInactive RoleStatus = "inactive"
)
```

### 4.3 User-Role Association

```go
type UserRole struct {
    ID        uuid.UUID `json:"id" db:"id"`
    UserID    uuid.UUID `json:"user_id" db:"user_id"`
    RoleID    uuid.UUID `json:"role_id" db:"role_id"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
```

## 5. Database Schema

### 5.1 Users Table

```sql
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'pending', 'suspended');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    surname VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT,
    oidc_provider VARCHAR(100),
    oidc_subject VARCHAR(255),
    status user_status DEFAULT 'pending',
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT valid_email CHECK (email ~* '^[A-Za-z0-9._%+-]+@gmail\.com$'),
    CONSTRAINT valid_auth CHECK (
        (password_hash IS NOT NULL AND oidc_provider IS NULL AND oidc_subject IS NULL) OR
        (password_hash IS NULL AND oidc_provider IS NOT NULL AND oidc_subject IS NOT NULL)
    )
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_oidc ON users(oidc_provider, oidc_subject);
```

### 5.2 Roles Table

```sql
CREATE TYPE role_status AS ENUM ('active', 'inactive');

CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    status role_status DEFAULT 'active',
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_roles_name ON roles(name);
CREATE INDEX idx_roles_status ON roles(status);
```

### 5.3 User-Role Association Table

```sql
CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
```

## 6. API Endpoints

### 6.1 User Management Endpoints

#### 6.1.1 Create User - POST /api/v1/users/add

**Request Body:**
```json
{
    "name": "John",
    "surname": "Doe",
    "email": "john.doe@gmail.com",
    "password": "optional_password",
    "oidc_provider": "google",
    "oidc_subject": "google_user_id",
    "role_ids": ["role-uuid-1", "role-uuid-2"],
    "metadata": {}
}
```

**Response:**
```json
{
    "success": true,
    "data": {
        "user": {
            "id": "user-uuid",
            "name": "John",
            "surname": "Doe",
            "email": "john.doe@gmail.com",
            "status": "active",
            "roles": [
                {
                    "id": "role-uuid-1",
                    "name": "viewer",
                    "description": "Read-only access"
                }
            ],
            "created_at": "2024-01-01T00:00:00Z",
            "updated_at": "2024-01-01T00:00:00Z"
        }
    }
}
```

#### 6.1.2 Delete User - DELETE /api/v1/users/delete

**Request Body:**
```json
{
    "user_id": "user-uuid"
}
```

**Response:**
```json
{
    "success": true,
    "message": "User deleted successfully",
    "data": {
        "user_id": "user-uuid",
        "deleted_at": "2024-01-01T00:00:00Z"
    }
}
```

#### 6.1.3 Update User - PUT /api/v1/users/update

**Request Body:**
```json
{
    "user_id": "user-uuid",
    "name": "Updated Name",
    "surname": "Updated Surname",
    "email": "updated.email@gmail.com",
    "status": "active",
    "role_ids": ["role-uuid-1"],
    "metadata": {}
}
```

**Response:**
```json
{
    "success": true,
    "data": {
        "user": {
            "id": "user-uuid",
            "name": "Updated Name",
            "surname": "Updated Surname",
            "email": "updated.email@gmail.com",
            "status": "active",
            "roles": [...],
            "updated_at": "2024-01-01T00:00:00Z"
        }
    }
}
```

#### 6.1.4 Get User Info - GET /api/v1/users/info/{user_id}

**Response:**
```json
{
    "success": true,
    "data": {
        "user": {
            "id": "user-uuid",
            "name": "John",
            "surname": "Doe",
            "email": "john.doe@gmail.com",
            "status": "active",
            "oidc_provider": "google",
            "roles": [...],
            "metadata": {},
            "created_at": "2024-01-01T00:00:00Z",
            "updated_at": "2024-01-01T00:00:00Z",
            "last_login_at": "2024-01-01T00:00:00Z"
        }
    }
}
```

#### 6.1.5 List Users - POST /api/v1/users/list

**Request Body:**
```json
{
    "filters": {
        "status": ["active", "inactive"],
        "role_ids": ["role-uuid-1"],
        "search": "john",
        "oidc_provider": "google"
    },
    "sort": {
        "field": "created_at",
        "order": "desc"
    },
    "pagination": {
        "page": 1,
        "limit": 20
    }
}
```

**Response:**
```json
{
    "success": true,
    "data": {
        "users": [
            {
                "id": "user-uuid",
                "name": "John",
                "surname": "Doe",
                "email": "john.doe@gmail.com",
                "status": "active",
                "roles": [...],
                "created_at": "2024-01-01T00:00:00Z",
                "last_login_at": "2024-01-01T00:00:00Z"
            }
        ],
        "total": 1,
        "pagination": {
            "page": 1,
            "limit": 20,
            "total_pages": 1,
            "has_next": false,
            "has_prev": false
        }
    }
}
```

### 6.2 Role Management Endpoints

#### 6.2.1 Create Role - POST /api/v1/roles/add

**Request Body:**
```json
{
    "name": "content_editor",
    "description": "Can create and edit content",
    "permissions": [
        "posts:create",
        "posts:update",
        "posts:read",
        "media:create",
        "media:read"
    ],
    "metadata": {}
}
```

**Response:**
```json
{
    "success": true,
    "data": {
        "role": {
            "id": "role-uuid",
            "name": "content_editor",
            "description": "Can create and edit content",
            "permissions": [...],
            "status": "active",
            "created_at": "2024-01-01T00:00:00Z",
            "updated_at": "2024-01-01T00:00:00Z"
        }
    }
}
```

#### 6.2.2 Delete Role - DELETE /api/v1/roles/delete

**Request Body:**
```json
{
    "role_id": "role-uuid"
}
```

**Response:**
```json
{
    "success": true,
    "message": "Role deleted successfully",
    "data": {
        "role_id": "role-uuid",
        "deleted_at": "2024-01-01T00:00:00Z"
    }
}
```

#### 6.2.3 Update Role - PUT /api/v1/roles/update

**Request Body:**
```json
{
    "role_id": "role-uuid",
    "name": "updated_role",
    "description": "Updated description",
    "permissions": ["posts:read", "media:read"],
    "status": "active",
    "metadata": {}
}
```

#### 6.2.4 Get Role Info - GET /api/v1/roles/info/{role_id}

**Response:**
```json
{
    "success": true,
    "data": {
        "role": {
            "id": "role-uuid",
            "name": "content_editor",
            "description": "Can create and edit content",
            "permissions": [...],
            "status": "active",
            "user_count": 5,
            "created_at": "2024-01-01T00:00:00Z",
            "updated_at": "2024-01-01T00:00:00Z"
        }
    }
}
```

#### 6.2.5 List Roles - POST /api/v1/roles/list

**Request Body:**
```json
{
    "filters": {
        "status": ["active"],
        "search": "editor",
        "permissions": ["posts:create"]
    },
    "sort": {
        "field": "name",
        "order": "asc"
    },
    "pagination": {
        "page": 1,
        "limit": 20
    }
}
```

**Response:**
```json
{
    "success": true,
    "data": {
        "roles": [
            {
                "id": "role-uuid",
                "name": "content_editor",
                "description": "Can create and edit content",
                "permissions": [...],
                "status": "active",
                "user_count": 5,
                "created_at": "2024-01-01T00:00:00Z"
            }
        ],
        "total": 1,
        "pagination": {...}
    }
}
```

## 7. Business Logic

### 7.1 User Management Logic

1. **User Creation**
   - Validate email format (must be Gmail)
   - Ensure either password or OIDC authentication is provided
   - Hash password if provided
   - Assign default roles if none specified
   - Send welcome email notification

2. **User Updates**
   - Validate email uniqueness
   - Update password hash if password changed
   - Handle role assignments/removals
   - Track update timestamps

3. **User Deletion**
   - Soft delete approach (status change)
   - Remove from all roles
   - Cleanup associated data
   - Audit trail maintenance

### 7.2 Role Management Logic

1. **Role Creation**
   - Validate role name uniqueness
   - Validate permission format
   - Set default status as active

2. **Role Updates**
   - Validate permission changes
   - Update associated users
   - Track update timestamps

3. **Role Deletion**
   - Check for associated users
   - Remove role from all users
   - Audit trail maintenance

## 8. Security Considerations

### 8.1 Authentication
- Password hashing using bcrypt with salt
- JWT token validation for OIDC
- Session management
- Rate limiting on authentication endpoints

### 8.2 Authorization
- Role-based access control
- Permission validation for all operations
- Admin-only user management operations
- Audit logging for sensitive operations

### 8.3 Data Protection
- Email validation and sanitization
- Password complexity requirements
- Secure storage of sensitive data
- GDPR compliance considerations

## 9. Default Roles and Permissions

### 9.1 System Roles

1. **Super Admin**
   - Full system access
   - User and role management
   - All resource permissions

2. **Admin**
   - User management (limited)
   - Content management
   - System configuration

3. **Content Manager**
   - Content creation and editing
   - Media management
   - Show and event management

4. **Viewer**
   - Read-only access
   - Basic content viewing

### 9.2 Permission Categories

- **users:***  - User management permissions
- **roles:***  - Role management permissions
- **shows:***  - Show management permissions
- **events:*** - Event management permissions
- **blocks:*** - Block management permissions
- **media:***  - Media management permissions
- **posts:***  - Post management permissions

## 10. Migration Strategy

### 10.1 Database Migration
- Create new user and role tables
- Migrate existing authentication data
- Assign default roles to existing users
- Update foreign key references

### 10.2 API Migration
- Maintain backward compatibility
- Gradual rollout of new endpoints
- Update client applications
- Deprecation notices for old endpoints

## 11. Testing Requirements

### 11.1 Unit Tests
- User CRUD operations
- Role management functions
- Permission validation
- Authentication methods

### 11.2 Integration Tests
- End-to-end user workflows
- Role assignment scenarios
- Authentication flows
- API endpoint testing

### 11.3 Security Tests
- Authentication bypass attempts
- Permission escalation tests
- Input validation testing
- Session management testing

## 12. Performance Considerations

### 12.1 Database Optimization
- Proper indexing strategy
- Query optimization
- Connection pooling
- Caching frequently accessed data

### 12.2 API Performance
- Pagination for large datasets
- Rate limiting implementation
- Response caching
- Async processing for heavy operations

## 13. Monitoring and Logging

### 13.1 Audit Logging
- User creation, updates, deletions
- Role assignments and removals
- Authentication events
- Permission changes

### 13.2 Metrics
- User registration rates
- Authentication success/failure rates
- Role usage statistics
- API endpoint performance

## 14. Future Enhancements

### 14.1 Advanced Features
- Multi-factor authentication
- Single sign-on (SSO) integration
- Advanced permission hierarchies
- User groups and organizations

### 14.2 Integration Capabilities
- External identity providers
- LDAP/Active Directory integration
- Webhook notifications
- API key management

## 15. Acceptance Criteria

### 15.1 User Management
- ✅ Users can be created with name, surname, and Gmail account
- ✅ Support for both password and OIDC authentication
- ✅ Users can have multiple roles assigned
- ✅ All user CRUD operations available via API
- ✅ Proper validation and error handling

### 15.2 Role Management
- ✅ Roles can be created with name, description, and permissions
- ✅ Role-based access control implemented
- ✅ All role CRUD operations available via API
- ✅ Permission validation and enforcement

### 15.3 Security
- ✅ Secure authentication and authorization
- ✅ Password hashing and validation
- ✅ OIDC integration working
- ✅ Audit logging implemented
- ✅ Rate limiting on sensitive endpoints

This PRD provides a comprehensive foundation for implementing a robust user and roles management system that meets all specified requirements while ensuring security, scalability, and maintainability.