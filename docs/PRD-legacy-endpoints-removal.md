# Product Requirements Document: Legacy Endpoints Removal

## Document Information
- **Title**: Legacy User and Role Management Endpoints Removal
- **Version**: 1.0
- **Date**: January 8, 2025
- **Author**: API Team
- **Status**: Draft

## Executive Summary

This PRD outlines the requirements for removing legacy user and role management API endpoints from the St. Planer system. With the successful implementation of RESTful alternatives, it's time to deprecate and remove the old endpoints to maintain a clean, consistent API surface.

## Background

The St. Planer API has undergone a significant refactoring to align with RESTful principles. New endpoints have been implemented and are now stable:

### User Management Migration
- `POST /api/v1/users/add` → `POST /api/v1/users`
- `DELETE /api/v1/users/delete` → `DELETE /api/v1/users/{user_id}`
- `PUT /api/v1/users/update` → `PUT /api/v1/users/{user_id}`
- `GET /api/v1/users/info/{user_id}` → `GET /api/v1/users/{user_id}`
- `POST /api/v1/users/list` → `GET /api/v1/users`

### Role Management Migration
- `POST /api/v1/roles/add` → `POST /api/v1/roles`
- `DELETE /api/v1/roles/delete` → `DELETE /api/v1/roles/{role_id}`
- `PUT /api/v1/roles/update` → `PUT /api/v1/roles/{role_id}`
- `GET /api/v1/roles/info/{role_id}` → `GET /api/v1/roles/{role_id}`
- `POST /api/v1/roles/list` → `GET /api/v1/roles`

The legacy endpoints are no longer needed and their removal will simplify maintenance and reduce confusion.

## Objectives

1. **API Simplification**: Remove redundant endpoints to maintain a single, clear API pattern
2. **Maintenance Reduction**: Eliminate the need to maintain two sets of endpoints
3. **Documentation Clarity**: Simplify API documentation with only current endpoints
4. **Performance**: Slight performance improvement by reducing routing overhead
5. **Code Quality**: Remove deprecated code and associated tests

## Scope

### In Scope
- Complete removal of 10 legacy endpoints (5 user, 5 role)
- Removal of legacy handler methods
- Removal of legacy request/response models
- Update router configuration
- Update API documentation
- Remove associated tests

### Out of Scope
- Changes to the new RESTful endpoints
- Database schema modifications
- Authentication endpoints
- Other API endpoints (media, posts, etc.)

## Technical Requirements

### 1. Endpoints to Remove

#### User Management Endpoints
1. `POST /api/v1/users/add`
2. `DELETE /api/v1/users/delete`
3. `PUT /api/v1/users/update`
4. `GET /api/v1/users/info/{user_id}`
5. `POST /api/v1/users/list`

#### Role Management Endpoints
1. `POST /api/v1/roles/add`
2. `DELETE /api/v1/roles/delete`
3. `PUT /api/v1/roles/update`
4. `GET /api/v1/roles/info/{role_id}`
5. `POST /api/v1/roles/list`

### 2. Code Components to Remove

#### Handler Methods
**User Handler (`internal/api/handlers/users.go`)**
- `CreateUser()` - handles `/api/v1/users/add`
- `DeleteUser()` - handles `/api/v1/users/delete`
- `UpdateUser()` - handles `/api/v1/users/update`
- `GetUserInfo()` - handles `/api/v1/users/info/{user_id}`
- `ListUsers()` - handles `/api/v1/users/list`

**Role Handler (`internal/api/handlers/roles.go`)**
- `CreateRole()` - handles `/api/v1/roles/add`
- `DeleteRole()` - handles `/api/v1/roles/delete`
- `UpdateRole()` - handles `/api/v1/roles/update`
- `GetRoleInfo()` - handles `/api/v1/roles/info/{role_id}`
- `ListRoles()` - handles `/api/v1/roles/list`

#### Model Types
**User Models (`internal/models/models.go`)**
- `CreateUserRequestLegacy`
- `UpdateUserRequestLegacy`
- `DeleteUserRequestLegacy`

**Role Models (`internal/models/models.go`)**
- `CreateRoleRequestLegacy`
- `UpdateRoleRequestLegacy`
- `DeleteRoleRequestLegacy`

#### Router Configuration
**Router (`internal/api/router/router.go`)**
- Remove legacy user endpoint group and routes
- Remove legacy role endpoint group and routes
- Remove comments marking endpoints as deprecated

### 3. Documentation Updates

#### Swagger/OpenAPI
- Remove all legacy endpoint documentation
- Update generated swagger files
- Ensure only RESTful endpoints are documented

#### README/Guides
- Update any references to old endpoints
- Update migration guides to indicate completion
- Update example code snippets

### 4. Test Removal

Remove all tests associated with legacy endpoints:
- Unit tests for handler methods
- Integration tests for legacy endpoints
- Any test utilities specific to legacy endpoints

## Implementation Plan

### Phase 1: Pre-removal Verification
1. Confirm all clients have migrated to new endpoints
2. Verify no recent usage of legacy endpoints in logs
3. Ensure new endpoints have complete feature parity
4. Backup current codebase

### Phase 2: Code Removal
1. Remove handler methods from user and role handlers
2. Remove legacy model types
3. Update router configuration
4. Remove associated tests
5. Update imports and dependencies

### Phase 3: Documentation Update
1. Regenerate Swagger documentation
2. Update API reference documentation
3. Remove legacy endpoint examples
4. Update changelog

### Phase 4: Testing and Validation
1. Run all remaining tests
2. Verify new endpoints still function correctly
3. Check for any broken references
4. Performance testing

### Phase 5: Deployment
1. Deploy to staging environment
2. Run smoke tests
3. Deploy to production
4. Monitor for any issues

## Risk Assessment

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Client still using legacy endpoints | High | Low | Pre-removal verification and monitoring |
| Accidental removal of shared code | Medium | Low | Careful code review and testing |
| Documentation inconsistencies | Low | Medium | Thorough documentation review |
| Breaking imports/dependencies | Medium | Low | Comprehensive testing |

## Success Criteria

1. **Clean Removal**: All legacy endpoints return 404 Not Found
2. **No Regressions**: New RESTful endpoints continue to work correctly
3. **Test Coverage**: All remaining tests pass
4. **Documentation**: API docs only show current endpoints
5. **Performance**: No negative impact on API performance
6. **Zero Errors**: No runtime errors after deployment

## Rollback Plan

If issues are discovered after deployment:

1. **Immediate**: Revert to previous version
2. **Investigation**: Analyze what went wrong
3. **Fix**: Address the specific issue
4. **Re-deployment**: Deploy fixed version
5. **Communication**: Notify affected parties

## Timeline

- **Day 1**: Pre-removal verification and backup
- **Day 2**: Code removal and testing
- **Day 3**: Documentation updates
- **Day 4**: Staging deployment and testing
- **Day 5**: Production deployment
- **Day 6-7**: Monitoring and issue resolution

## Verification Checklist

### Pre-Removal
- [ ] Confirm zero usage of legacy endpoints in last 30 days
- [ ] All clients confirmed migrated
- [ ] New endpoints tested for feature parity
- [ ] Codebase backed up

### Code Removal
- [ ] User handler methods removed
- [ ] Role handler methods removed
- [ ] Legacy models removed
- [ ] Router configuration updated
- [ ] Tests removed
- [ ] No compilation errors
- [ ] All remaining tests pass

### Documentation
- [ ] Swagger regenerated
- [ ] API reference updated
- [ ] Examples updated
- [ ] Changelog updated

### Deployment
- [ ] Staging deployment successful
- [ ] Staging tests passed
- [ ] Production deployment successful
- [ ] Monitoring shows no errors

## Communication Plan

### Internal Communication
1. **Engineering Team**: Technical details of removal
2. **QA Team**: Testing requirements
3. **DevOps**: Deployment coordination
4. **Product Team**: Feature impact assessment

### External Communication
1. **API Consumers**: Final removal notice
2. **Documentation**: Updated API reference
3. **Support Team**: FAQ for handling questions

## Long-term Benefits

1. **Maintainability**: Single API pattern easier to maintain
2. **Clarity**: Clear, consistent API design
3. **Performance**: Reduced routing overhead
4. **Security**: Fewer endpoints to secure
5. **Documentation**: Simpler, clearer documentation
6. **Developer Experience**: Better for new developers

## Approval

- [ ] Engineering Lead
- [ ] API Team Lead
- [ ] QA Lead
- [ ] DevOps Lead
- [ ] Product Manager

## Appendix

### Legacy Endpoint Reference

For historical reference, the removed endpoints were:

#### User Management
- `POST /api/v1/users/add` - Created users with complex request
- `DELETE /api/v1/users/delete` - Deleted users with body parameter
- `PUT /api/v1/users/update` - Updated users with complex request
- `GET /api/v1/users/info/{user_id}` - Retrieved user with roles
- `POST /api/v1/users/list` - Listed users with POST method

#### Role Management
- `POST /api/v1/roles/add` - Created roles with metadata
- `DELETE /api/v1/roles/delete` - Deleted roles with body parameter
- `PUT /api/v1/roles/update` - Updated roles with complex request
- `GET /api/v1/roles/info/{role_id}` - Retrieved role with user count
- `POST /api/v1/roles/list` - Listed roles with POST method

### Migration Summary

All functionality previously available through legacy endpoints is now available through RESTful endpoints with improved design and simplified request/response formats.