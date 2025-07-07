package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/denisAlshanov/stPlaner/internal/database"
	"github.com/denisAlshanov/stPlaner/internal/models"
	"github.com/denisAlshanov/stPlaner/internal/utils"
)

// RoleHandler handles role-related HTTP requests
type RoleHandler struct {
	db *database.PostgresDB
}

// NewRoleHandler creates a new role handler
func NewRoleHandler(db *database.PostgresDB) *RoleHandler {
	return &RoleHandler{db: db}
}

// CreateRole handles POST /api/v1/roles/add
// @Summary Create new role
// @Description Create a new role with specified permissions
// @Tags roles
// @Accept json
// @Produce json
// @Param request body models.CreateRoleRequest true "Role creation data"
// @Success 200 {object} models.CreateRoleResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/roles/add [post]
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req models.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Normalize role name
	roleName := strings.ToLower(strings.TrimSpace(req.Name))
	
	// Check if role already exists
	existingRole, err := h.db.GetRoleByName(c.Request.Context(), roleName)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to check existing role", err, utils.Fields{
			"name": roleName,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing role"})
		return
	}

	if existingRole != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Role with this name already exists"})
		return
	}

	// Validate permissions format
	validPermissions := make([]string, 0)
	for _, perm := range req.Permissions {
		perm = strings.TrimSpace(perm)
		if perm != "" {
			// Validate permission format (resource:action)
			parts := strings.Split(perm, ":")
			if len(parts) != 2 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":      "Invalid permission format",
					"permission": perm,
					"expected":   "resource:action",
				})
				return
			}
			validPermissions = append(validPermissions, perm)
		}
	}

	if len(validPermissions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one permission is required"})
		return
	}

	// Create role object
	role := &models.Role{
		Name:        roleName,
		Description: req.Description,
		Permissions: validPermissions,
		Status:      models.RoleStatusActive,
		Metadata:    req.Metadata,
	}

	// Create role in database
	if err := h.db.CreateRole(c.Request.Context(), role); err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			c.JSON(http.StatusConflict, gin.H{"error": "Role already exists"})
			return
		}
		utils.LogError(c.Request.Context(), "Failed to create role", err, utils.Fields{
			"name": role.Name,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create role"})
		return
	}

	utils.LogInfo(c.Request.Context(), "Role created successfully", utils.Fields{
		"role_id": role.ID,
		"name":    role.Name,
	})

	c.JSON(http.StatusOK, models.CreateRoleResponse{
		Success: true,
		Data:    role,
	})
}

// UpdateRole handles PUT /api/v1/roles/update
// @Summary Update role information
// @Description Update existing role details and permissions
// @Tags roles
// @Accept json
// @Produce json
// @Param request body models.UpdateRoleRequest true "Role update data"
// @Success 200 {object} models.UpdateRoleResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/roles/update [put]
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	var req models.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Parse role ID
	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID format"})
		return
	}

	// Get existing role
	existingRole, err := h.db.GetRoleByID(c.Request.Context(), roleID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get role", err, utils.Fields{
			"role_id": roleID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve role"})
		return
	}

	if existingRole == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	// Prevent modification of system roles
	systemRoles := []string{"super_admin", "admin", "content_manager", "viewer"}
	for _, sysRole := range systemRoles {
		if existingRole.Name == sysRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot modify system role"})
			return
		}
	}

	// Prepare update
	role := &models.Role{
		ID: roleID,
	}

	// Check name uniqueness if being updated
	if req.Name != nil {
		normalizedName := strings.ToLower(strings.TrimSpace(*req.Name))
		if normalizedName != existingRole.Name {
			nameRole, err := h.db.GetRoleByName(c.Request.Context(), normalizedName)
			if err != nil {
				utils.LogError(c.Request.Context(), "Failed to check role name", err, utils.Fields{
					"name": normalizedName,
				})
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check name availability"})
				return
			}
			if nameRole != nil {
				c.JSON(http.StatusConflict, gin.H{"error": "Role name already in use"})
				return
			}
			role.Name = normalizedName
		}
	}

	// Update fields
	if req.Description != nil {
		role.Description = req.Description
	}
	if req.Status != nil {
		role.Status = *req.Status
	}
	if req.Metadata != nil {
		role.Metadata = req.Metadata
	}

	// Validate and update permissions
	if req.Permissions != nil {
		validPermissions := make([]string, 0)
		for _, perm := range req.Permissions {
			perm = strings.TrimSpace(perm)
			if perm != "" {
				// Validate permission format (resource:action)
				parts := strings.Split(perm, ":")
				if len(parts) != 2 {
					c.JSON(http.StatusBadRequest, gin.H{
						"error":      "Invalid permission format",
						"permission": perm,
						"expected":   "resource:action",
					})
					return
				}
				validPermissions = append(validPermissions, perm)
			}
		}

		if len(validPermissions) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "At least one permission is required"})
			return
		}

		role.Permissions = validPermissions
	}

	// Update role in database
	if err := h.db.UpdateRole(c.Request.Context(), role); err != nil {
		utils.LogError(c.Request.Context(), "Failed to update role", err, utils.Fields{
			"role_id": roleID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
		return
	}

	// Get updated role
	updatedRole, err := h.db.GetRoleByID(c.Request.Context(), roleID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get updated role", err, utils.Fields{
			"role_id": roleID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Role updated but failed to retrieve details"})
		return
	}

	utils.LogInfo(c.Request.Context(), "Role updated successfully", utils.Fields{
		"role_id": roleID,
		"name":    updatedRole.Name,
	})

	c.JSON(http.StatusOK, models.UpdateRoleResponse{
		Success: true,
		Data:    &updatedRole.Role,
	})
}

// DeleteRole handles DELETE /api/v1/roles/delete
// @Summary Delete role
// @Description Delete a role if it has no associated users
// @Tags roles
// @Accept json
// @Produce json
// @Param request body models.DeleteRoleRequest true "Role deletion data"
// @Success 200 {object} models.DeleteRoleResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/roles/delete [delete]
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	var req models.DeleteRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Parse role ID
	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID format"})
		return
	}

	// Check if role exists
	role, err := h.db.GetRoleByID(c.Request.Context(), roleID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get role", err, utils.Fields{
			"role_id": roleID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve role"})
		return
	}

	if role == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	// Prevent deletion of system roles
	systemRoles := []string{"super_admin", "admin", "content_manager", "viewer"}
	for _, sysRole := range systemRoles {
		if role.Name == sysRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete system role"})
			return
		}
	}

	// Check if role has users
	if role.UserCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":      "Cannot delete role with associated users",
			"user_count": role.UserCount,
		})
		return
	}

	// Delete role
	if err := h.db.DeleteRole(c.Request.Context(), roleID); err != nil {
		if strings.Contains(err.Error(), "associated users") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		utils.LogError(c.Request.Context(), "Failed to delete role", err, utils.Fields{
			"role_id": roleID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete role"})
		return
	}

	utils.LogInfo(c.Request.Context(), "Role deleted successfully", utils.Fields{
		"role_id": roleID,
		"name":    role.Name,
	})

	c.JSON(http.StatusOK, models.DeleteRoleResponse{
		Success: true,
		Message: "Role deleted successfully",
		Data: &models.RoleDeleteData{
			RoleID:    req.RoleID,
			DeletedAt: time.Now(),
		},
	})
}

// GetRoleInfo handles GET /api/v1/roles/info/{role_id}
// @Summary Get role information
// @Description Get detailed information about a specific role including user count
// @Tags roles
// @Produce json
// @Param role_id path string true "Role ID"
// @Success 200 {object} models.GetRoleInfoResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/roles/info/{role_id} [get]
func (h *RoleHandler) GetRoleInfo(c *gin.Context) {
	roleIDStr := c.Param("role_id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID format"})
		return
	}

	// Get role with user count
	role, err := h.db.GetRoleByID(c.Request.Context(), roleID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get role", err, utils.Fields{
			"role_id": roleID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve role"})
		return
	}

	if role == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	c.JSON(http.StatusOK, models.GetRoleInfoResponse{
		Success: true,
		Data:    role,
	})
}

// ListRoles handles POST /api/v1/roles/list
// @Summary List roles
// @Description Get paginated list of roles with filters
// @Tags roles
// @Accept json
// @Produce json
// @Param request body models.ListRolesRequest true "List filters and options"
// @Success 200 {object} models.ListRolesResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/roles/list [post]
func (h *RoleHandler) ListRoles(c *gin.Context) {
	var req models.ListRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Set default pagination if not provided
	if req.Pagination == nil {
		req.Pagination = &models.PaginationOptions{
			Page:  1,
			Limit: 20,
		}
	}

	// Get roles from database
	roles, total, err := h.db.ListRoles(c.Request.Context(), req.Filters, req.Sort, req.Pagination)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to list roles", err, utils.Fields{
			"filters": req.Filters,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve roles"})
		return
	}

	// Calculate pagination response
	totalPages := (total + req.Pagination.Limit - 1) / req.Pagination.Limit
	paginationResp := &models.PaginationResponse{
		Page:       req.Pagination.Page,
		Limit:      req.Pagination.Limit,
		Total:      total,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, models.ListRolesResponse{
		Success: true,
		Data: &models.ListRolesData{
			Roles:      roles,
			Total:      total,
			Pagination: paginationResp,
		},
	})
}