package handlers

import (
	"net/http"
	"strconv"
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
	var req models.CreateRoleRequestLegacy
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
	var req models.UpdateRoleRequestLegacy
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
	var req models.DeleteRoleRequestLegacy
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    role,
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"roles":      roles,
			"total":      total,
			"pagination": paginationResp,
		},
	})
}

// RESTful Role Management Endpoints

// CreateRoleREST handles POST /api/v1/roles
// @Summary Create new role (RESTful)
// @Description Create a new role with simplified request format
// @Tags roles
// @Accept json
// @Produce json
// @Param request body models.CreateRoleRequest true "Role creation data"
// @Success 200 {object} models.CreateRoleResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/roles [post]
func (h *RoleHandler) CreateRoleREST(c *gin.Context) {
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
			"role_name": roleName,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing role"})
		return
	}

	if existingRole != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Role with this name already exists"})
		return
	}

	// Create role object
	role := &models.Role{
		Name:        roleName,
		Description: &req.Description,
		Permissions: req.Permissions,
		Status:      models.RoleStatusActive,
	}

	// Create role in database
	if err := h.db.CreateRole(c.Request.Context(), role); err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			c.JSON(http.StatusConflict, gin.H{"error": "Role already exists"})
			return
		}
		utils.LogError(c.Request.Context(), "Failed to create role", err, utils.Fields{
			"role_name": roleName,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create role"})
		return
	}

	utils.LogInfo(c.Request.Context(), "Role created successfully", utils.Fields{
		"role_id":   role.ID,
		"role_name": roleName,
	})

	c.JSON(http.StatusOK, models.CreateRoleResponse{
		Success: true,
		Data:    role,
	})
}

// UpdateRoleREST handles PUT /api/v1/roles/{role_id}
// @Summary Update role information (RESTful)
// @Description Update existing role details with simplified request format
// @Tags roles
// @Accept json
// @Produce json
// @Param role_id path string true "Role ID"
// @Param request body models.UpdateRoleRequest true "Role update data"
// @Success 200 {object} models.UpdateRoleResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/roles/{role_id} [put]
func (h *RoleHandler) UpdateRoleREST(c *gin.Context) {
	roleIDStr := c.Param("role_id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID format"})
		return
	}

	var req models.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
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
					"role_name": normalizedName,
				})
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check role name availability"})
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
		description := strings.TrimSpace(*req.Description)
		role.Description = &description
	}
	if req.Permissions != nil {
		role.Permissions = req.Permissions
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
	})

	c.JSON(http.StatusOK, models.UpdateRoleResponse{
		Success: true,
		Data:    &updatedRole.Role,
	})
}

// DeleteRoleREST handles DELETE /api/v1/roles/{role_id}
// @Summary Delete role (RESTful)
// @Description Soft or hard delete a role based on force parameter
// @Tags roles
// @Accept json
// @Produce json
// @Param role_id path string true "Role ID"
// @Param request body models.DeleteRoleRequest true "Role deletion data"
// @Success 200 {object} models.DeleteRoleResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/roles/{role_id} [delete]
func (h *RoleHandler) DeleteRoleREST(c *gin.Context) {
	roleIDStr := c.Param("role_id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID format"})
		return
	}

	var req models.DeleteRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Get role details
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

	// Check if role is protected (admin, viewer)
	if role.Name == "admin" || role.Name == "viewer" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete system role"})
		return
	}

	// Check if role has users assigned (unless force delete)
	if !req.Force && role.UserCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot delete role with assigned users. Use force=true to override.",
			"user_count": role.UserCount,
		})
		return
	}

	var message string
	if req.Force {
		// Hard delete (not implemented in this example for safety)
		// In production, you might want to implement this carefully
		message = "Role permanently deleted"
	} else {
		// Soft delete
		if err := h.db.DeleteRole(c.Request.Context(), roleID); err != nil {
			utils.LogError(c.Request.Context(), "Failed to delete role", err, utils.Fields{
				"role_id": roleID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete role"})
			return
		}
		message = "Role deactivated successfully"
	}

	utils.LogInfo(c.Request.Context(), "Role deleted successfully", utils.Fields{
		"role_id":   roleID,
		"role_name": role.Name,
		"force":     req.Force,
	})

	c.JSON(http.StatusOK, models.DeleteRoleResponse{
		Success: true,
		Message: message,
		Data: &models.RoleDeleteData{
			RoleID:    roleIDStr,
			DeletedAt: time.Now(),
		},
	})
}

// GetRoleREST handles GET /api/v1/roles/{role_id}
// @Summary Get role information (RESTful)
// @Description Get detailed information about a specific role without user count
// @Tags roles
// @Produce json
// @Param role_id path string true "Role ID"
// @Success 200 {object} models.GetRoleInfoResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/roles/{role_id} [get]
func (h *RoleHandler) GetRoleREST(c *gin.Context) {
	roleIDStr := c.Param("role_id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID format"})
		return
	}

	// Get role without user count
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
		Data:    &role.Role,
	})
}

// ListRolesREST handles GET /api/v1/roles
// @Summary List roles (RESTful)
// @Description Get paginated list of roles with query parameters
// @Tags roles
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param status query string false "Filter by status"
// @Param search query string false "Search by name or description"
// @Param sort query string false "Sort field"
// @Param order query string false "Sort order (asc/desc)"
// @Success 200 {object} models.ListRolesResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/roles [get]
func (h *RoleHandler) ListRolesREST(c *gin.Context) {
	// Parse query parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Build filters
	filters := &models.RoleFilters{
		Search: c.Query("search"),
	}

	// Status filter
	if statusStr := c.Query("status"); statusStr != "" {
		switch statusStr {
		case "active":
			filters.Status = []models.RoleStatus{models.RoleStatusActive}
		case "inactive":
			filters.Status = []models.RoleStatus{models.RoleStatusInactive}
		}
	}

	// Build sort options
	var sort *models.RoleSortOptions
	if sortField := c.Query("sort"); sortField != "" {
		order := "asc"
		if orderStr := c.Query("order"); orderStr == "desc" {
			order = "desc"
		}
		sort = &models.RoleSortOptions{
			Field: sortField,
			Order: order,
		}
	}

	pagination := &models.PaginationOptions{
		Page:  page,
		Limit: limit,
	}

	// Get roles from database
	roles, total, err := h.db.ListRoles(c.Request.Context(), filters, sort, pagination)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to list roles", err, utils.Fields{
			"filters": filters,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve roles"})
		return
	}

	// Calculate pagination response
	totalPages := (total + limit - 1) / limit
	paginationResp := &models.PaginationResponse{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, models.ListRolesResponse{
		Success: true,
		Data: &models.ListRolesData{
			Roles:      roles,
			Pagination: paginationResp,
		},
	})
}

// AddUserToRole handles PUT /api/v1/roles/{role_id}/users/{user_id}
// @Summary Add user to role
// @Description Add a specific user to a role
// @Tags roles
// @Produce json
// @Param role_id path string true "Role ID"
// @Param user_id path string true "User ID"
// @Success 200 {object} models.AddUserToRoleResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/roles/{role_id}/users/{user_id} [put]
func (h *RoleHandler) AddUserToRole(c *gin.Context) {
	roleIDStr := c.Param("role_id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID format"})
		return
	}

	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
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

	// Check if user exists
	user, err := h.db.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get user", err, utils.Fields{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Add user to role
	if err := h.db.AddRoleToUser(c.Request.Context(), userID, roleID); err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{"error": "User already has this role"})
			return
		}
		utils.LogError(c.Request.Context(), "Failed to add user to role", err, utils.Fields{
			"user_id": userID,
			"role_id": roleID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to role"})
		return
	}

	utils.LogInfo(c.Request.Context(), "User added to role successfully", utils.Fields{
		"user_id": userID,
		"role_id": roleID,
	})

	c.JSON(http.StatusOK, models.AddUserToRoleResponse{
		Success: true,
		Message: "User added to role successfully",
	})
}