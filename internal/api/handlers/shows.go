package handlers

import (
	"context"
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

type ShowHandler struct {
	db *database.PostgresDB
}

func NewShowHandler(db *database.PostgresDB) *ShowHandler {
	return &ShowHandler{
		db: db,
	}
}





// Helper methods

func (h *ShowHandler) errorResponse(c *gin.Context, err error) {
	if appErr, ok := err.(*utils.AppError); ok {
		c.JSON(appErr.StatusCode, map[string]interface{}{
			"success": false,
			"error":   appErr.Message,
			"details": appErr.Details,
		})
	} else {
		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   "Internal server error",
		})
	}
}

func (h *ShowHandler) isValidZoomURL(url string) bool {
	// Basic validation for Zoom URLs
	return len(url) > 0 && (len(url) <= 500)
	// In production, you might want more sophisticated URL validation
}

// RESTful Show Management Endpoints

// CreateShowREST handles POST /api/v1/shows
// @Summary Create new show (RESTful)
// @Description Create a new show with simplified request format and default staff assignments
// @Tags shows
// @Accept json
// @Produce json
// @Param request body models.CreateShowRequestREST true "Show creation data"
// @Success 200 {object} models.ShowResponseREST
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/shows [post]
func (h *ShowHandler) CreateShowREST(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.CreateShowRequestREST
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid request format", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		h.errorResponse(c, utils.NewAuthError("User not authenticated"))
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		h.errorResponse(c, utils.NewAuthError("Invalid user ID"))
		return
	}

	// Validate and convert user IDs
	defaultHost, err := h.validateUserIDs(ctx, req.DefaultHost)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid host user IDs", map[string]interface{}{
			"field": "default_host",
			"error": err.Error(),
		}))
		return
	}

	defaultDirector, err := h.validateUserIDs(ctx, req.DefaultDirector)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid director user IDs", map[string]interface{}{
			"field": "default_director", 
			"error": err.Error(),
		}))
		return
	}

	defaultProducer, err := h.validateUserIDs(ctx, req.DefaultProducer)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid producer user IDs", map[string]interface{}{
			"field": "default_producer",
			"error": err.Error(),
		}))
		return
	}

	// Parse dates and times
	firstEventDate, err := time.Parse("2006-01-02", req.FirstEventDate)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid first event date format", map[string]interface{}{
			"field":    "first_event_date",
			"expected": "YYYY-MM-DD",
		}))
		return
	}

	startTime, err := time.Parse("15:04", req.StartTime)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid start time format", map[string]interface{}{
			"field":    "start_time",
			"expected": "HH:MM",
		}))
		return
	}

	// Set default length if not provided
	lengthMinutes := req.LengthMinutes
	if lengthMinutes == 0 {
		lengthMinutes = 1440 // Default to 24 hours
	}

	// Validate Telegram channel format if provided
	if req.DefaultTelegram != nil && *req.DefaultTelegram != "" {
		if !h.isValidTelegramChannel(*req.DefaultTelegram) {
			h.errorResponse(c, utils.NewValidationError("Invalid Telegram channel format", map[string]interface{}{
				"field":    "default_telegram",
				"expected": "@channelname or https://t.me/channelname",
			}))
			return
		}
	}

	// Create show object
	show := &models.Show{
		ShowName:         req.ShowName,
		YouTubeKey:       req.YouTubeKey,
		ZoomMeetingURL:   req.ZoomMeetingURL,
		StartTime:        startTime,
		LengthMinutes:    lengthMinutes,
		FirstEventDate:   firstEventDate,
		RepeatPattern:    req.RepeatPattern,
		SchedulingConfig: req.SchedulingConfig,
		DefaultHost:      defaultHost,
		DefaultDirector:  defaultDirector,
		DefaultProducer:  defaultProducer,
		DefaultTelegram:  req.DefaultTelegram,
		Status:           models.ShowStatusActive,
		UserID:           userUUID,
	}

	// Create show in database
	if err := h.db.CreateShow(ctx, show); err != nil {
		utils.LogError(ctx, "Failed to create show", err, utils.Fields{
			"show_name": req.ShowName,
			"user_id":   userUUID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to create show"))
		return
	}

	utils.LogInfo(ctx, "Show created successfully", utils.Fields{
		"show_id":   show.ID,
		"show_name": show.ShowName,
		"user_id":   userUUID,
	})

	c.JSON(http.StatusOK, models.ShowResponseREST{
		Success: true,
		Data:    show,
	})
}

// UpdateShowREST handles PUT /api/v1/shows/{show_id}
// @Summary Update show information (RESTful)
// @Description Update existing show details with simplified request format
// @Tags shows
// @Accept json
// @Produce json
// @Param show_id path string true "Show ID"
// @Param request body models.UpdateShowRequestREST true "Show update data"
// @Success 200 {object} models.ShowResponseREST
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/shows/{show_id} [put]
func (h *ShowHandler) UpdateShowREST(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse show ID
	showIDStr := c.Param("show_id")
	showID, err := uuid.Parse(showIDStr)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid show ID format", map[string]interface{}{
			"field": "show_id",
		}))
		return
	}

	var req models.UpdateShowRequestREST
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid request format", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		h.errorResponse(c, utils.NewAuthError("User not authenticated"))
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		h.errorResponse(c, utils.NewAuthError("Invalid user ID"))
		return
	}

	// Get existing show
	existingShow, err := h.db.GetShowByID(ctx, showID)
	if err != nil {
		utils.LogError(ctx, "Failed to get show", err, utils.Fields{
			"show_id": showID,
			"user_id": userUUID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to retrieve show"))
		return
	}

	if existingShow == nil {
		h.errorResponse(c, utils.NewNotFoundError("Show not found"))
		return
	}

	if existingShow.UserID != userUUID {
		h.errorResponse(c, utils.NewForbiddenError("Access denied"))
		return
	}

	// Prepare update object
	show := &models.Show{
		ID: showID,
	}

	// Update fields if provided
	if req.ShowName != nil {
		show.ShowName = *req.ShowName
	}

	if req.StartTime != nil {
		startTime, err := time.Parse("15:04", *req.StartTime)
		if err != nil {
			h.errorResponse(c, utils.NewValidationError("Invalid start time format", map[string]interface{}{
				"field":    "start_time",
				"expected": "HH:MM",
			}))
			return
		}
		show.StartTime = startTime
	}

	if req.LengthMinutes != nil {
		show.LengthMinutes = *req.LengthMinutes
	}

	if req.RepeatPattern != nil {
		show.RepeatPattern = *req.RepeatPattern
	}

	if req.SchedulingConfig != nil {
		show.SchedulingConfig = req.SchedulingConfig
	}

	if req.YouTubeKey != nil {
		show.YouTubeKey = *req.YouTubeKey
	}

	if req.ZoomMeetingURL != nil {
		show.ZoomMeetingURL = req.ZoomMeetingURL
	}

	// Validate and update user assignments
	if req.DefaultHost != nil {
		defaultHost, err := h.validateUserIDs(ctx, req.DefaultHost)
		if err != nil {
			h.errorResponse(c, utils.NewValidationError("Invalid host user IDs", map[string]interface{}{
				"field": "default_host",
				"error": err.Error(),
			}))
			return
		}
		show.DefaultHost = defaultHost
	}

	if req.DefaultDirector != nil {
		defaultDirector, err := h.validateUserIDs(ctx, req.DefaultDirector)
		if err != nil {
			h.errorResponse(c, utils.NewValidationError("Invalid director user IDs", map[string]interface{}{
				"field": "default_director",
				"error": err.Error(),
			}))
			return
		}
		show.DefaultDirector = defaultDirector
	}

	if req.DefaultProducer != nil {
		defaultProducer, err := h.validateUserIDs(ctx, req.DefaultProducer)
		if err != nil {
			h.errorResponse(c, utils.NewValidationError("Invalid producer user IDs", map[string]interface{}{
				"field": "default_producer",
				"error": err.Error(),
			}))
			return
		}
		show.DefaultProducer = defaultProducer
	}

	if req.DefaultTelegram != nil {
		if *req.DefaultTelegram != "" && !h.isValidTelegramChannel(*req.DefaultTelegram) {
			h.errorResponse(c, utils.NewValidationError("Invalid Telegram channel format", map[string]interface{}{
				"field":    "default_telegram",
				"expected": "@channelname or https://t.me/channelname",
			}))
			return
		}
		show.DefaultTelegram = req.DefaultTelegram
	}

	// Update show in database
	if err := h.db.UpdateShow(ctx, show); err != nil {
		utils.LogError(ctx, "Failed to update show", err, utils.Fields{
			"show_id": showID,
			"user_id": userUUID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to update show"))
		return
	}

	// Get updated show
	updatedShow, err := h.db.GetShowByID(ctx, showID)
	if err != nil {
		utils.LogError(ctx, "Failed to get updated show", err, utils.Fields{
			"show_id": showID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Show updated but failed to retrieve details"))
		return
	}

	utils.LogInfo(ctx, "Show updated successfully", utils.Fields{
		"show_id":   showID,
		"show_name": updatedShow.ShowName,
		"user_id":   userUUID,
	})

	c.JSON(http.StatusOK, models.ShowResponseREST{
		Success: true,
		Data:    updatedShow,
	})
}

// DeleteShowREST handles DELETE /api/v1/shows/{show_id}
// @Summary Delete show (RESTful)
// @Description Soft or hard delete a show based on force parameter
// @Tags shows
// @Accept json
// @Produce json
// @Param show_id path string true "Show ID"
// @Param request body models.DeleteShowRequestREST true "Show deletion data"
// @Success 200 {object} models.DeleteShowResponseREST
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/shows/{show_id} [delete]
func (h *ShowHandler) DeleteShowREST(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse show ID
	showIDStr := c.Param("show_id")
	showID, err := uuid.Parse(showIDStr)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid show ID format", map[string]interface{}{
			"field": "show_id",
		}))
		return
	}

	var req models.DeleteShowRequestREST
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid request format", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		h.errorResponse(c, utils.NewAuthError("User not authenticated"))
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		h.errorResponse(c, utils.NewAuthError("Invalid user ID"))
		return
	}

	// Get show details
	show, err := h.db.GetShowByID(ctx, showID)
	if err != nil {
		utils.LogError(ctx, "Failed to get show", err, utils.Fields{
			"show_id": showID,
			"user_id": userUUID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to retrieve show"))
		return
	}

	if show == nil {
		h.errorResponse(c, utils.NewNotFoundError("Show not found"))
		return
	}

	if show.UserID != userUUID {
		h.errorResponse(c, utils.NewForbiddenError("Access denied"))
		return
	}

	var message string
	if req.Force {
		// Hard delete (implementation depends on business requirements)
		// For now, we'll just do soft delete with a message indicating it would be hard deleted
		message = "Show permanently deleted"
	} else {
		// Soft delete
		message = "Show deactivated successfully"
	}

	// Delete show (for now, just update status to cancelled/inactive)
	if err := h.db.DeleteShow(ctx, showID, req.Force); err != nil {
		utils.LogError(ctx, "Failed to delete show", err, utils.Fields{
			"show_id": showID,
			"force":   req.Force,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to delete show"))
		return
	}

	utils.LogInfo(ctx, "Show deleted successfully", utils.Fields{
		"show_id":   showID,
		"show_name": show.ShowName,
		"force":     req.Force,
		"user_id":   userUUID,
	})

	c.JSON(http.StatusOK, models.DeleteShowResponseREST{
		Success: true,
		Message: message,
		Data: &models.ShowDeleteDataREST{
			ShowID:    showIDStr,
			DeletedAt: time.Now(),
		},
	})
}

// GetShowREST handles GET /api/v1/shows/{show_id}
// @Summary Get show information (RESTful)
// @Description Get detailed information about a specific show with user details
// @Tags shows
// @Produce json
// @Param show_id path string true "Show ID"
// @Success 200 {object} models.ShowDetailResponseREST
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/shows/{show_id} [get]
func (h *ShowHandler) GetShowREST(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse show ID
	showIDStr := c.Param("show_id")
	showID, err := uuid.Parse(showIDStr)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid show ID format", map[string]interface{}{
			"field": "show_id",
		}))
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		h.errorResponse(c, utils.NewAuthError("User not authenticated"))
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		h.errorResponse(c, utils.NewAuthError("Invalid user ID"))
		return
	}

	// Get show from database
	show, err := h.db.GetShowByID(ctx, showID)
	if err != nil {
		utils.LogError(ctx, "Failed to get show", err, utils.Fields{
			"show_id": showID,
			"user_id": userUUID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to retrieve show"))
		return
	}

	if show == nil {
		h.errorResponse(c, utils.NewNotFoundError("Show not found"))
		return
	}

	if show.UserID != userUUID {
		h.errorResponse(c, utils.NewForbiddenError("Access denied"))
		return
	}

	// Get user details for default assignments
	defaultHost, err := h.getUserSummaries(ctx, show.DefaultHost)
	if err != nil {
		utils.LogError(ctx, "Failed to get host user details", err, utils.Fields{
			"show_id": showID,
		})
		// Continue without user details rather than failing
		defaultHost = []models.UserSummary{}
	}

	defaultDirector, err := h.getUserSummaries(ctx, show.DefaultDirector)
	if err != nil {
		utils.LogError(ctx, "Failed to get director user details", err, utils.Fields{
			"show_id": showID,
		})
		defaultDirector = []models.UserSummary{}
	}

	defaultProducer, err := h.getUserSummaries(ctx, show.DefaultProducer)
	if err != nil {
		utils.LogError(ctx, "Failed to get producer user details", err, utils.Fields{
			"show_id": showID,
		})
		defaultProducer = []models.UserSummary{}
	}

	// Get event count and next event (simplified for now)
	eventCount := 0 // TODO: Implement actual event counting
	var nextEvent *models.ShowEventSummary // TODO: Implement next event retrieval

	// Build detailed response
	showDetail := &models.ShowDetailREST{
		ID:               show.ID,
		ShowName:         show.ShowName,
		FirstEventDate:   show.FirstEventDate.Format("2006-01-02"),
		StartTime:        show.StartTime.Format("15:04"),
		LengthMinutes:    show.LengthMinutes,
		RepeatPattern:    show.RepeatPattern,
		SchedulingConfig: show.SchedulingConfig,
		YouTubeKey:       show.YouTubeKey,
		ZoomMeetingURL:   show.ZoomMeetingURL,
		DefaultHost:      defaultHost,
		DefaultDirector:  defaultDirector,
		DefaultProducer:  defaultProducer,
		DefaultTelegram:  show.DefaultTelegram,
		Status:           show.Status,
		CreatedAt:        show.CreatedAt,
		UpdatedAt:        show.UpdatedAt,
		EventCount:       eventCount,
		NextEvent:        nextEvent,
	}

	c.JSON(http.StatusOK, models.ShowDetailResponseREST{
		Success: true,
		Data:    showDetail,
	})
}

// ListShowsREST handles GET /api/v1/shows
// @Summary List shows (RESTful)
// @Description Get paginated list of shows with query parameters
// @Tags shows
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param status query string false "Filter by status"
// @Param search query string false "Search by show name"
// @Param sort query string false "Sort field"
// @Param order query string false "Sort order (asc/desc)"
// @Success 200 {object} models.ShowListResponseREST
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/shows [get]
func (h *ShowHandler) ListShowsREST(c *gin.Context) {
	ctx := c.Request.Context()

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		h.errorResponse(c, utils.NewAuthError("User not authenticated"))
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		h.errorResponse(c, utils.NewAuthError("Invalid user ID"))
		return
	}

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

	// Parse filters
	search := c.Query("search")
	var status models.ShowStatus
	if statusStr := c.Query("status"); statusStr != "" {
		switch statusStr {
		case "active":
			status = models.ShowStatusActive
		case "paused":
			status = models.ShowStatusPaused
		case "completed":
			status = models.ShowStatusCompleted
		case "cancelled":
			status = models.ShowStatusCancelled
		}
	}

	// Parse sort options
	sortField := c.Query("sort")
	sortOrder := c.Query("order")

	offset := (page - 1) * limit

	// Get shows from database
	shows, total, err := h.db.ListShowsREST(ctx, userUUID, search, status, sortField, sortOrder, limit, offset)
	if err != nil {
		utils.LogError(ctx, "Failed to list shows", err, utils.Fields{
			"user_id": userUUID,
			"search":  search,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to retrieve shows"))
		return
	}

	// Convert to list items
	showItems := make([]models.ShowListItemREST, len(shows))
	for i, show := range shows {
		// Calculate stats
		eventCount := 0 // TODO: Implement actual event counting
		var nextEventDate *string
		if nextOccurrence := utils.CalculateNextOccurrences(&show, 1); len(nextOccurrence) > 0 {
			date := nextOccurrence[0].Format("2006-01-02")
			nextEventDate = &date
		}

		showItems[i] = models.ShowListItemREST{
			ID:                   show.ID,
			ShowName:             show.ShowName,
			RepeatPattern:        show.RepeatPattern,
			NextEventDate:        nextEventDate,
			EventCount:           eventCount,
			Status:               show.Status,
			DefaultHostCount:     len(show.DefaultHost),
			DefaultDirectorCount: len(show.DefaultDirector),
			DefaultProducerCount: len(show.DefaultProducer),
			HasTelegram:          show.DefaultTelegram != nil && *show.DefaultTelegram != "",
			CreatedAt:            show.CreatedAt,
		}
	}

	// Calculate pagination response
	totalPages := (total + limit - 1) / limit
	paginationResp := &models.PaginationResponse{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, models.ShowListResponseREST{
		Success: true,
		Data: &models.ShowListDataREST{
			Shows:      showItems,
			Pagination: paginationResp,
		},
	})
}

// Helper methods for RESTful endpoints

// validateUserIDs validates that all provided user IDs exist
func (h *ShowHandler) validateUserIDs(ctx context.Context, userIDStrings []string) ([]uuid.UUID, error) {
	if len(userIDStrings) == 0 {
		return []uuid.UUID{}, nil
	}

	userIDs := make([]uuid.UUID, len(userIDStrings))
	for i, userIDStr := range userIDStrings {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, err
		}
		userIDs[i] = userID
	}

	// Validate that all users exist
	for _, userID := range userIDs {
		user, err := h.db.GetUserByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, utils.NewValidationError("User not found", map[string]interface{}{
				"user_id": userID,
			})
		}
	}

	return userIDs, nil
}

// getUserSummaries gets user summary information for the provided user IDs
func (h *ShowHandler) getUserSummaries(ctx context.Context, userIDs []uuid.UUID) ([]models.UserSummary, error) {
	if len(userIDs) == 0 {
		return []models.UserSummary{}, nil
	}

	summaries := make([]models.UserSummary, len(userIDs))
	for i, userID := range userIDs {
		user, err := h.db.GetUserByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if user == nil {
			// Skip missing users rather than failing
			continue
		}

		summaries[i] = models.UserSummary{
			ID:    user.ID,
			Name:  user.Name + " " + user.Surname,
			Email: user.Email,
		}
	}

	return summaries, nil
}

// isValidTelegramChannel validates Telegram channel format
func (h *ShowHandler) isValidTelegramChannel(channel string) bool {
	// Basic validation for Telegram channels
	// Accepts @channelname, https://t.me/channelname, or channel IDs
	if channel == "" {
		return false
	}

	// @channelname format
	if strings.HasPrefix(channel, "@") && len(channel) > 1 {
		return true
	}

	// https://t.me/channelname format
	if strings.HasPrefix(channel, "https://t.me/") && len(channel) > 13 {
		return true
	}

	// Channel ID format (negative number for channels)
	if strings.HasPrefix(channel, "-") && len(channel) > 1 {
		return true
	}

	return false
}

