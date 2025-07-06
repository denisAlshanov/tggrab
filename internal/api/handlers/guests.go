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

// GuestHandler handles guest-related HTTP requests
type GuestHandler struct {
	db *database.PostgresDB
}

// NewGuestHandler creates a new guest handler
func NewGuestHandler(db *database.PostgresDB) *GuestHandler {
	return &GuestHandler{db: db}
}

// CreateGuest handles POST /api/v1/guest/new
// @Summary Create new guest
// @Description Create a new guest with contact information and notes
// @Tags guests
// @Accept json
// @Produce json
// @Param request body models.CreateGuestRequest true "Guest creation data"
// @Success 200 {object} models.CreateGuestResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /api/v1/guest/new [post]
func (h *GuestHandler) CreateGuest(c *gin.Context) {
	var req models.CreateGuestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Validate contacts
	if err := validateGuestContacts(req.Contacts); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contact information", "details": err.Error()})
		return
	}

	// Create guest object
	guest := &models.Guest{
		UserID:    userUUID,
		Name:      strings.TrimSpace(req.Name),
		Surname:   strings.TrimSpace(req.Surname),
		ShortName: req.ShortName,
		Contacts:  req.Contacts,
		Notes:     req.Notes,
		Avatar:    req.Avatar,
		Tags:      req.Tags,
		Metadata:  req.Metadata,
	}

	// Validate short name if provided
	if guest.ShortName != nil {
		trimmed := strings.TrimSpace(*guest.ShortName)
		if trimmed == "" {
			guest.ShortName = nil
		} else {
			guest.ShortName = &trimmed
		}
	}

	// Create guest in database
	if err := h.db.CreateGuest(c.Request.Context(), guest); err != nil {
		if strings.Contains(err.Error(), "unique_user_guest") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Guest with this name already exists"})
			return
		}
		utils.LogError(c.Request.Context(), "Failed to create guest", err, utils.Fields{
			"user_id":  userUUID,
			"name":     guest.Name,
			"surname":  guest.Surname,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create guest"})
		return
	}

	utils.LogInfo(c.Request.Context(), "Guest created successfully", utils.Fields{
		"guest_id": guest.ID,
		"user_id":  userUUID,
		"name":     guest.Name,
		"surname":  guest.Surname,
	})

	c.JSON(http.StatusOK, models.CreateGuestResponse{
		Success: true,
		Data:    guest,
	})
}

// UpdateGuest handles PUT /api/v1/guest/update
// @Summary Update guest information
// @Description Update existing guest with new information
// @Tags guests
// @Accept json
// @Produce json
// @Param request body models.UpdateGuestRequest true "Guest update data"
// @Success 200 {object} models.UpdateGuestResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /api/v1/guest/update [put]
func (h *GuestHandler) UpdateGuest(c *gin.Context) {
	var req models.UpdateGuestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Parse guest ID
	guestID, err := uuid.Parse(req.GuestID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid guest ID format"})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Get existing guest
	guest, err := h.db.GetGuestByID(c.Request.Context(), guestID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get guest", err, utils.Fields{
			"guest_id": guestID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve guest"})
		return
	}

	if guest == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guest not found"})
		return
	}

	// Verify ownership
	if guest.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Update fields
	updated := false

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed != "" && trimmed != guest.Name {
			guest.Name = trimmed
			updated = true
		}
	}

	if req.Surname != nil {
		trimmed := strings.TrimSpace(*req.Surname)
		if trimmed != "" && trimmed != guest.Surname {
			guest.Surname = trimmed
			updated = true
		}
	}

	if req.ShortName != nil {
		trimmed := strings.TrimSpace(*req.ShortName)
		if trimmed == "" {
			if guest.ShortName != nil {
				guest.ShortName = nil
				updated = true
			}
		} else {
			if guest.ShortName == nil || *guest.ShortName != trimmed {
				guest.ShortName = &trimmed
				updated = true
			}
		}
	}

	if req.Contacts != nil {
		if err := validateGuestContacts(req.Contacts); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contact information", "details": err.Error()})
			return
		}
		guest.Contacts = req.Contacts
		updated = true
	}

	if req.Notes != nil {
		guest.Notes = req.Notes
		updated = true
	}

	if req.Avatar != nil {
		guest.Avatar = req.Avatar
		updated = true
	}

	if req.Tags != nil {
		guest.Tags = req.Tags
		updated = true
	}

	if req.Metadata != nil {
		guest.Metadata = req.Metadata
		updated = true
	}

	if !updated {
		c.JSON(http.StatusOK, models.UpdateGuestResponse{
			Success: true,
			Data:    guest,
		})
		return
	}

	// Update in database
	if err := h.db.UpdateGuest(c.Request.Context(), guest); err != nil {
		if strings.Contains(err.Error(), "unique_user_guest") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Guest with this name already exists"})
			return
		}
		utils.LogError(c.Request.Context(), "Failed to update guest", err, utils.Fields{
			"guest_id": guestID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update guest"})
		return
	}

	utils.LogInfo(c.Request.Context(), "Guest updated successfully", utils.Fields{
		"guest_id": guestID,
		"user_id":  userUUID,
	})

	c.JSON(http.StatusOK, models.UpdateGuestResponse{
		Success: true,
		Data:    guest,
	})
}

// ListGuests handles POST /api/v1/guest/list
// @Summary List guests with filtering
// @Description Get paginated list of guests with filtering and sorting options
// @Tags guests
// @Accept json
// @Produce json
// @Param request body models.ListGuestsRequest true "Guest list filters"
// @Success 200 {object} models.ListGuestsResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /api/v1/guest/list [post]
func (h *GuestHandler) ListGuests(c *gin.Context) {
	var req models.ListGuestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Set defaults
	if req.Pagination.Limit <= 0 {
		req.Pagination.Limit = 20
	}
	if req.Pagination.Page <= 0 {
		req.Pagination.Page = 1
	}
	if req.Sort.Field == "" {
		req.Sort.Field = "name"
	}
	if req.Sort.Order == "" {
		req.Sort.Order = "asc"
	}

	// Get guests from database
	guests, total, err := h.db.ListGuests(c.Request.Context(), userUUID, req.Filters, req.Pagination, req.Sort)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to list guests", err, utils.Fields{
			"user_id": userUUID,
			"filters": req.Filters,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve guests"})
		return
	}

	// Calculate pagination
	totalPages := (total + req.Pagination.Limit - 1) / req.Pagination.Limit

	c.JSON(http.StatusOK, models.ListGuestsResponse{
		Success: true,
		Data: &models.ListGuestsData{
			Guests: guests,
			Pagination: models.PaginationResponse{
				Page:       req.Pagination.Page,
				Limit:      req.Pagination.Limit,
				Total:      total,
				TotalPages: totalPages,
			},
		},
	})
}

// AutocompleteGuests handles GET /api/v1/guest/autocomplete
// @Summary Guest autocomplete search
// @Description Get guest suggestions for autocomplete functionality
// @Tags guests
// @Produce json
// @Param q query string true "Search query (minimum 2 characters)"
// @Param limit query int false "Maximum results (default: 10, max: 50)"
// @Success 200 {object} models.AutocompleteResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /api/v1/guest/autocomplete [get]
func (h *GuestHandler) AutocompleteGuests(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if len(query) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query must be at least 2 characters long"})
		return
	}

	// Parse limit with defaults
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			if parsedLimit > 0 && parsedLimit <= 50 {
				limit = parsedLimit
			}
		}
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Search guests
	suggestions, err := h.db.SearchGuests(c.Request.Context(), userUUID, query, limit)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to search guests", err, utils.Fields{
			"user_id": userUUID,
			"query":   query,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search guests"})
		return
	}

	c.JSON(http.StatusOK, models.AutocompleteResponse{
		Success: true,
		Data: &models.AutocompleteData{
			Suggestions:  suggestions,
			Query:        query,
			TotalMatches: len(suggestions),
		},
	})
}

// GetGuestInfo handles GET /api/v1/guest/info/{guest_id}
// @Summary Get guest details
// @Description Get detailed information about a specific guest
// @Tags guests
// @Produce json
// @Param guest_id path string true "Guest ID"
// @Success 200 {object} models.GetGuestInfoResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /api/v1/guest/info/{guest_id} [get]
func (h *GuestHandler) GetGuestInfo(c *gin.Context) {
	guestIDStr := c.Param("guest_id")
	guestID, err := uuid.Parse(guestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid guest ID format"})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Get guest
	guest, err := h.db.GetGuestByID(c.Request.Context(), guestID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get guest", err, utils.Fields{
			"guest_id": guestID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve guest"})
		return
	}

	if guest == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guest not found"})
		return
	}

	// Verify ownership
	if guest.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// TODO: Implement guest stats (total shows, last appearance, upcoming shows)
	// This would require additional database queries to events/shows tables
	stats := &models.GuestStats{
		TotalShows:     0,
		LastAppearance: nil,
		UpcomingShows:  0,
	}

	c.JSON(http.StatusOK, models.GetGuestInfoResponse{
		Success: true,
		Data: &models.GuestInfoData{
			Guest: guest,
			Stats: stats,
		},
	})
}

// DeleteGuest handles DELETE /api/v1/guest/delete
// @Summary Delete guest
// @Description Delete a guest from the system
// @Tags guests
// @Accept json
// @Produce json
// @Param request body models.DeleteGuestRequest true "Guest deletion data"
// @Success 200 {object} models.DeleteGuestResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /api/v1/guest/delete [delete]
func (h *GuestHandler) DeleteGuest(c *gin.Context) {
	var req models.DeleteGuestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Parse guest ID
	guestID, err := uuid.Parse(req.GuestID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid guest ID format"})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Get existing guest to verify ownership
	guest, err := h.db.GetGuestByID(c.Request.Context(), guestID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get guest", err, utils.Fields{
			"guest_id": guestID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve guest"})
		return
	}

	if guest == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Guest not found"})
		return
	}

	// Verify ownership
	if guest.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Delete guest
	if err := h.db.DeleteGuest(c.Request.Context(), guestID); err != nil {
		utils.LogError(c.Request.Context(), "Failed to delete guest", err, utils.Fields{
			"guest_id": guestID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete guest"})
		return
	}

	utils.LogInfo(c.Request.Context(), "Guest deleted successfully", utils.Fields{
		"guest_id": guestID,
		"user_id":  userUUID,
	})

	c.JSON(http.StatusOK, models.DeleteGuestResponse{
		Success: true,
		Message: "Guest deleted successfully",
		Data: &models.GuestDeleteData{
			GuestID:   req.GuestID,
			DeletedAt: time.Now(),
		},
	})
}

// Helper functions

// validateGuestContacts validates a slice of guest contacts
func validateGuestContacts(contacts []models.GuestContact) error {
	if len(contacts) == 0 {
		return nil // Contacts are optional
	}

	primaryCount := 0
	for i, contact := range contacts {
		// Validate contact type
		if !isValidContactType(contact.Type) {
			return utils.NewValidationError("invalid contact type", map[string]interface{}{
				"index": i,
				"type":  contact.Type,
			})
		}

		// Validate contact value
		if strings.TrimSpace(contact.Value) == "" {
			return utils.NewValidationError("contact value cannot be empty", map[string]interface{}{
				"index": i,
				"type":  contact.Type,
			})
		}

		// Count primary contacts
		if contact.IsPrimary {
			primaryCount++
		}
	}

	// Only one primary contact allowed
	if primaryCount > 1 {
		return utils.NewValidationError("only one primary contact allowed", map[string]interface{}{
			"primary_count": primaryCount,
		})
	}

	return nil
}

// isValidContactType checks if a contact type is valid
func isValidContactType(contactType models.ContactType) bool {
	switch contactType {
	case models.ContactTypeEmail,
		models.ContactTypePhone,
		models.ContactTypeTelegram,
		models.ContactTypeDiscord,
		models.ContactTypeTwitter,
		models.ContactTypeLinkedIn,
		models.ContactTypeInstagram,
		models.ContactTypeWebsite,
		models.ContactTypeOther:
		return true
	default:
		return false
	}
}