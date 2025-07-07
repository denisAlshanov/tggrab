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

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	db *database.PostgresDB
}

// NewUserHandler creates a new user handler
func NewUserHandler(db *database.PostgresDB) *UserHandler {
	return &UserHandler{db: db}
}

// CreateUser handles POST /api/v1/users/add
// @Summary Create new user
// @Description Create a new user with password or OIDC authentication
// @Tags users
// @Accept json
// @Produce json
// @Param request body models.CreateUserRequest true "User creation data"
// @Success 200 {object} models.CreateUserResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/users/add [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Check if user is admin (normally would check JWT claims)
	// For now, we'll check if the requesting user has user:create permission
	// This is a placeholder - implement proper auth middleware
	
	// Validate email is Gmail
	if !strings.HasSuffix(strings.ToLower(req.Email), "@gmail.com") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email must be a Gmail account"})
		return
	}

	// Check if user already exists
	existingUser, err := h.db.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to check existing user", err, utils.Fields{
			"email": req.Email,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing user"})
		return
	}

	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		return
	}

	// Validate authentication method
	if req.Password == nil && (req.OIDCProvider == nil || req.OIDCSubject == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either password or OIDC credentials must be provided"})
		return
	}

	if req.Password != nil && (req.OIDCProvider != nil || req.OIDCSubject != nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot use both password and OIDC authentication"})
		return
	}

	// Create user object
	user := &models.User{
		Name:         strings.TrimSpace(req.Name),
		Surname:      strings.TrimSpace(req.Surname),
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		OIDCProvider: req.OIDCProvider,
		OIDCSubject:  req.OIDCSubject,
		Status:       models.UserStatusActive,
		Metadata:     req.Metadata,
	}

	// Hash password if provided
	if req.Password != nil {
		hashedPassword, err := utils.ValidateAndHashPassword(*req.Password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid password", "details": err.Error()})
			return
		}
		user.PasswordHash = &hashedPassword
	}

	// Parse role IDs
	var roleIDs []uuid.UUID
	if len(req.RoleIDs) == 0 {
		// Assign default viewer role
		viewerRole, err := h.db.GetRoleByName(c.Request.Context(), "viewer")
		if err == nil && viewerRole != nil {
			roleIDs = append(roleIDs, viewerRole.ID)
		}
	} else {
		for _, roleIDStr := range req.RoleIDs {
			roleID, err := uuid.Parse(roleIDStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID format", "role_id": roleIDStr})
				return
			}
			
			// Verify role exists
			role, err := h.db.GetRoleByID(c.Request.Context(), roleID)
			if err != nil {
				utils.LogError(c.Request.Context(), "Failed to get role", err, utils.Fields{
					"role_id": roleID,
				})
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify role"})
				return
			}
			if role == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Role not found", "role_id": roleIDStr})
				return
			}
			
			roleIDs = append(roleIDs, roleID)
		}
	}

	// Create user in database
	if err := h.db.CreateUser(c.Request.Context(), user, roleIDs); err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			return
		}
		utils.LogError(c.Request.Context(), "Failed to create user", err, utils.Fields{
			"email": user.Email,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Get the created user with roles
	userWithRoles, err := h.db.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get created user", err, utils.Fields{
			"user_id": user.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User created but failed to retrieve details"})
		return
	}

	utils.LogInfo(c.Request.Context(), "User created successfully", utils.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	})

	c.JSON(http.StatusOK, models.CreateUserResponse{
		Success: true,
		Data:    userWithRoles,
	})
}

// UpdateUser handles PUT /api/v1/users/update
// @Summary Update user information
// @Description Update existing user details and roles
// @Tags users
// @Accept json
// @Produce json
// @Param request body models.UpdateUserRequest true "User update data"
// @Success 200 {object} models.UpdateUserResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/users/update [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Parse user ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Get existing user
	existingUser, err := h.db.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get user", err, utils.Fields{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	if existingUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Prepare update
	user := &models.User{
		ID: userID,
	}

	// Check email uniqueness if being updated
	if req.Email != nil && *req.Email != existingUser.Email {
		if !strings.HasSuffix(strings.ToLower(*req.Email), "@gmail.com") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email must be a Gmail account"})
			return
		}

		emailUser, err := h.db.GetUserByEmail(c.Request.Context(), *req.Email)
		if err != nil {
			utils.LogError(c.Request.Context(), "Failed to check email", err, utils.Fields{
				"email": *req.Email,
			})
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check email availability"})
			return
		}
		if emailUser != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already in use"})
			return
		}
		user.Email = strings.ToLower(strings.TrimSpace(*req.Email))
	}

	// Update fields
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		user.Name = name
	}
	if req.Surname != nil {
		surname := strings.TrimSpace(*req.Surname)
		user.Surname = surname
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	if req.Metadata != nil {
		user.Metadata = req.Metadata
	}

	// Hash new password if provided
	if req.Password != nil {
		hashedPassword, err := utils.ValidateAndHashPassword(*req.Password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid password", "details": err.Error()})
			return
		}
		user.PasswordHash = &hashedPassword
	}

	// Parse and validate role IDs
	var roleIDs []uuid.UUID
	if req.RoleIDs != nil {
		for _, roleIDStr := range req.RoleIDs {
			roleID, err := uuid.Parse(roleIDStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID format", "role_id": roleIDStr})
				return
			}
			
			// Verify role exists
			role, err := h.db.GetRoleByID(c.Request.Context(), roleID)
			if err != nil {
				utils.LogError(c.Request.Context(), "Failed to get role", err, utils.Fields{
					"role_id": roleID,
				})
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify role"})
				return
			}
			if role == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Role not found", "role_id": roleIDStr})
				return
			}
			
			roleIDs = append(roleIDs, roleID)
		}
	}

	// Update user in database
	if err := h.db.UpdateUser(c.Request.Context(), user, roleIDs); err != nil {
		utils.LogError(c.Request.Context(), "Failed to update user", err, utils.Fields{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	// Get updated user
	updatedUser, err := h.db.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get updated user", err, utils.Fields{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User updated but failed to retrieve details"})
		return
	}

	utils.LogInfo(c.Request.Context(), "User updated successfully", utils.Fields{
		"user_id": userID,
	})

	c.JSON(http.StatusOK, models.UpdateUserResponse{
		Success: true,
		Data:    updatedUser,
	})
}

// DeleteUser handles DELETE /api/v1/users/delete
// @Summary Delete user
// @Description Soft delete a user by setting status to inactive
// @Tags users
// @Accept json
// @Produce json
// @Param request body models.DeleteUserRequest true "User deletion data"
// @Success 200 {object} models.DeleteUserResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/users/delete [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	var req models.DeleteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Parse user ID
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
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

	// Soft delete user
	if err := h.db.DeleteUser(c.Request.Context(), userID); err != nil {
		utils.LogError(c.Request.Context(), "Failed to delete user", err, utils.Fields{
			"user_id": userID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	utils.LogInfo(c.Request.Context(), "User deleted successfully", utils.Fields{
		"user_id": userID,
		"email":   user.Email,
	})

	c.JSON(http.StatusOK, models.DeleteUserResponse{
		Success: true,
		Message: "User deleted successfully",
		Data: &models.UserDeleteData{
			UserID:    req.UserID,
			DeletedAt: time.Now(),
		},
	})
}

// GetUserInfo handles GET /api/v1/users/info/{user_id}
// @Summary Get user information
// @Description Get detailed information about a specific user including roles
// @Tags users
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} models.GetUserInfoResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/users/info/{user_id} [get]
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Get user with roles
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

	c.JSON(http.StatusOK, models.GetUserInfoResponse{
		Success: true,
		Data:    user,
	})
}

// ListUsers handles POST /api/v1/users/list
// @Summary List users
// @Description Get paginated list of users with filters
// @Tags users
// @Accept json
// @Produce json
// @Param request body models.ListUsersRequest true "List filters and options"
// @Success 200 {object} models.ListUsersResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/users/list [post]
func (h *UserHandler) ListUsers(c *gin.Context) {
	var req models.ListUsersRequest
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

	// Get users from database
	users, total, err := h.db.ListUsers(c.Request.Context(), req.Filters, req.Sort, req.Pagination)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to list users", err, utils.Fields{
			"filters": req.Filters,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
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

	c.JSON(http.StatusOK, models.ListUsersResponse{
		Success: true,
		Data: &models.ListUsersData{
			Users:      users,
			Total:      total,
			Pagination: paginationResp,
		},
	})
}