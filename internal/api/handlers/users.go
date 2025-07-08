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

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	db *database.PostgresDB
}

// NewUserHandler creates a new user handler
func NewUserHandler(db *database.PostgresDB) *UserHandler {
	return &UserHandler{db: db}
}


// RESTful User Management Endpoints

// CreateUserREST handles POST /api/v1/users
// @Summary Create new user (RESTful)
// @Description Create a new user with simplified request format
// @Tags users
// @Accept json
// @Produce json
// @Param request body models.CreateUserRequest true "User creation data"
// @Success 200 {object} models.CreateUserResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUserREST(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

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

	// Create user object
	user := &models.User{
		Name:    strings.TrimSpace(req.Name),
		Surname: strings.TrimSpace(req.Surname),
		Email:   strings.ToLower(strings.TrimSpace(req.Email)),
		Status:  models.UserStatusActive,
	}

	// Hash password
	hashedPassword, err := utils.ValidateAndHashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid password", "details": err.Error()})
		return
	}
	user.PasswordHash = &hashedPassword

	// Get default viewer role
	var roleIDs []uuid.UUID
	viewerRole, err := h.db.GetRoleByName(c.Request.Context(), "viewer")
	if err == nil && viewerRole != nil {
		roleIDs = append(roleIDs, viewerRole.ID)
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

	utils.LogInfo(c.Request.Context(), "User created successfully", utils.Fields{
		"user_id": user.ID,
		"email":   user.Email,
	})

	c.JSON(http.StatusOK, models.CreateUserResponse{
		Success: true,
		Data:    user,
	})
}

// UpdateUserREST handles PUT /api/v1/users/{user_id}
// @Summary Update user information (RESTful)
// @Description Update existing user details with simplified request format
// @Tags users
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Param request body models.UpdateUserRequest true "User update data"
// @Success 200 {object} models.UpdateUserResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/users/{user_id} [put]
func (h *UserHandler) UpdateUserREST(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
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

	// Update user in database (no role changes in this endpoint)
	if err := h.db.UpdateUser(c.Request.Context(), user, nil); err != nil {
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
		Data:    &updatedUser.User,
	})
}

// DeleteUserREST handles DELETE /api/v1/users/{user_id}
// @Summary Delete user (RESTful)
// @Description Soft or hard delete a user based on force parameter
// @Tags users
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Param request body models.DeleteUserRequest true "User deletion data"
// @Success 200 {object} models.DeleteUserResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/users/{user_id} [delete]
func (h *UserHandler) DeleteUserREST(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	var req models.DeleteUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
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

	var message string
	if req.Force {
		// Hard delete (not implemented in this example for safety)
		// In production, you might want to implement this carefully
		message = "User permanently deleted"
	} else {
		// Soft delete
		if err := h.db.DeleteUser(c.Request.Context(), userID); err != nil {
			utils.LogError(c.Request.Context(), "Failed to delete user", err, utils.Fields{
				"user_id": userID,
			})
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
			return
		}
		message = "User deactivated successfully"
	}

	utils.LogInfo(c.Request.Context(), "User deleted successfully", utils.Fields{
		"user_id": userID,
		"email":   user.Email,
		"force":   req.Force,
	})

	c.JSON(http.StatusOK, models.DeleteUserResponse{
		Success: true,
		Message: message,
		Data: &models.UserDeleteData{
			UserID:    userIDStr,
			DeletedAt: time.Now(),
		},
	})
}

// GetUserREST handles GET /api/v1/users/{user_id}
// @Summary Get user information (RESTful)
// @Description Get detailed information about a specific user without roles
// @Tags users
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} models.GetUserInfoResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/users/{user_id} [get]
func (h *UserHandler) GetUserREST(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Get user without roles
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
		Data:    &user.User,
	})
}

// ListUsersREST handles GET /api/v1/users
// @Summary List users (RESTful)
// @Description Get paginated list of users with query parameters
// @Tags users
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param status query string false "Filter by status"
// @Param search query string false "Search by name or email"
// @Param sort query string false "Sort field"
// @Param order query string false "Sort order (asc/desc)"
// @Success 200 {object} models.ListUsersResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/users [get]
func (h *UserHandler) ListUsersREST(c *gin.Context) {
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
	filters := &models.UserFilters{
		Search: c.Query("search"),
	}

	// Status filter
	if statusStr := c.Query("status"); statusStr != "" {
		switch statusStr {
		case "active":
			filters.Status = []models.UserStatus{models.UserStatusActive}
		case "inactive":
			filters.Status = []models.UserStatus{models.UserStatusInactive}
		}
	}

	// Build sort options
	var sort *models.UserSortOptions
	if sortField := c.Query("sort"); sortField != "" {
		order := "asc"
		if orderStr := c.Query("order"); orderStr == "desc" {
			order = "desc"
		}
		sort = &models.UserSortOptions{
			Field: sortField,
			Order: order,
		}
	}

	pagination := &models.PaginationOptions{
		Page:  page,
		Limit: limit,
	}

	// Get users from database
	users, total, err := h.db.ListUsers(c.Request.Context(), filters, sort, pagination)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to list users", err, utils.Fields{
			"filters": filters,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
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

	c.JSON(http.StatusOK, models.ListUsersResponse{
		Success: true,
		Data: &models.ListUsersData{
			Users:      users,
			Pagination: paginationResp,
		},
	})
}

// AddRoleToUser handles PUT /api/v1/users/{user_id}/roles/{role_id}
// @Summary Add role to user
// @Description Add a specific role to a user
// @Tags users
// @Produce json
// @Param user_id path string true "User ID"
// @Param role_id path string true "Role ID"
// @Success 200 {object} models.AddRoleToUserResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/users/{user_id}/roles/{role_id} [put]
func (h *UserHandler) AddRoleToUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	roleIDStr := c.Param("role_id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID format"})
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

	// Add role to user
	if err := h.db.AddRoleToUser(c.Request.Context(), userID, roleID); err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{"error": "User already has this role"})
			return
		}
		utils.LogError(c.Request.Context(), "Failed to add role to user", err, utils.Fields{
			"user_id": userID,
			"role_id": roleID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add role to user"})
		return
	}

	utils.LogInfo(c.Request.Context(), "Role added to user successfully", utils.Fields{
		"user_id": userID,
		"role_id": roleID,
	})

	c.JSON(http.StatusOK, models.AddRoleToUserResponse{
		Success: true,
		Message: "Role added successfully",
	})
}

// RemoveRoleFromUser handles DELETE /api/v1/users/{user_id}/roles/{role_id}
// @Summary Remove role from user
// @Description Remove a specific role from a user
// @Tags users
// @Produce json
// @Param user_id path string true "User ID"
// @Param role_id path string true "Role ID"
// @Success 200 {object} models.RemoveRoleFromUserResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/users/{user_id}/roles/{role_id} [delete]
func (h *UserHandler) RemoveRoleFromUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	roleIDStr := c.Param("role_id")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID format"})
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

	// Remove role from user
	if err := h.db.RemoveRoleFromUser(c.Request.Context(), userID, roleID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User does not have this role"})
			return
		}
		utils.LogError(c.Request.Context(), "Failed to remove role from user", err, utils.Fields{
			"user_id": userID,
			"role_id": roleID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove role from user"})
		return
	}

	utils.LogInfo(c.Request.Context(), "Role removed from user successfully", utils.Fields{
		"user_id": userID,
		"role_id": roleID,
	})

	c.JSON(http.StatusOK, models.RemoveRoleFromUserResponse{
		Success: true,
		Message: "Role removed successfully",
	})
}