package handlers

import (
	"net/http"
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

// CreateShow godoc
// @Summary Create a new show
// @Description Create a new show with YouTube and Zoom integration for recurring streams
// @Tags shows
// @Accept json
// @Produce json
// @Param request body models.CreateShowRequest true "Show details"
// @Success 200 {object} models.CreateShowResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/show/create [post]
// @Security ApiKeyAuth
func (h *ShowHandler) CreateShow(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.CreateShowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Get user ID from context (set by auth middleware)
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

	// Parse start time
	startTime, err := time.Parse("15:04:05", req.StartTime)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid start time format. Use HH:MM:SS", map[string]interface{}{
			"field": "start_time",
			"value": req.StartTime,
		}))
		return
	}

	// Parse first event date
	firstEventDate, err := time.Parse("2006-01-02", req.FirstEventDate)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid first event date format. Use YYYY-MM-DD", map[string]interface{}{
			"field": "first_event_date",
			"value": req.FirstEventDate,
		}))
		return
	}

	// Validate first event date is not in the past
	if firstEventDate.Before(time.Now().Truncate(24 * time.Hour)) {
		h.errorResponse(c, utils.NewValidationError("First event date cannot be in the past", map[string]interface{}{
			"field": "first_event_date",
			"value": req.FirstEventDate,
		}))
		return
	}

	// Validate scheduling configuration
	if err := utils.ValidateSchedulingConfig(req.RepeatPattern, req.SchedulingConfig); err != nil {
		h.errorResponse(c, err)
		return
	}

	// Validate Zoom URL if provided
	if req.ZoomMeetingURL != nil && *req.ZoomMeetingURL != "" {
		if !h.isValidZoomURL(*req.ZoomMeetingURL) {
			h.errorResponse(c, utils.NewValidationError("Invalid Zoom meeting URL format", map[string]interface{}{
				"field": "zoom_meeting_url",
				"value": *req.ZoomMeetingURL,
			}))
			return
		}
	}

	// Create show model
	show := &models.Show{
		ShowName:         req.ShowName,
		YouTubeKey:       req.YouTubeKey,
		AdditionalKey:    req.AdditionalKey,
		ZoomMeetingURL:   req.ZoomMeetingURL,
		ZoomMeetingID:    req.ZoomMeetingID,
		ZoomPasscode:     req.ZoomPasscode,
		StartTime:        startTime,
		LengthMinutes:    req.LengthMinutes,
		FirstEventDate:   firstEventDate,
		RepeatPattern:    req.RepeatPattern,
		SchedulingConfig: req.SchedulingConfig,
		Status:           models.ShowStatusActive,
		UserID:           userUUID,
		Metadata:         req.Metadata,
	}

	// Save to database
	if err := h.db.CreateShow(ctx, show); err != nil {
		utils.LogError(ctx, "Failed to create show", err, utils.Fields{
			"user_id":   userUUID,
			"show_name": req.ShowName,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to create show"))
		return
	}

	utils.LogInfo(ctx, "Show created successfully", utils.Fields{
		"show_id":   show.ID,
		"user_id":   userUUID,
		"show_name": req.ShowName,
	})

	c.JSON(http.StatusOK, models.CreateShowResponse{
		Success: true,
		Data:    show,
	})
}

// DeleteShow godoc
// @Summary Delete a show
// @Description Soft delete a show by setting its status to cancelled
// @Tags shows
// @Accept json
// @Produce json
// @Param request body models.DeleteShowRequest true "Show ID to delete"
// @Success 200 {object} models.DeleteShowResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/show/delete [delete]
// @Security ApiKeyAuth
func (h *ShowHandler) DeleteShow(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.DeleteShowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Parse show ID
	showID, err := uuid.Parse(req.ShowID)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid show ID format", map[string]interface{}{
			"field": "show_id",
			"value": req.ShowID,
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

	// Check if show exists and belongs to user
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

	// Delete show (soft delete)
	if err := h.db.DeleteShow(ctx, showID); err != nil {
		utils.LogError(ctx, "Failed to delete show", err, utils.Fields{
			"show_id": showID,
			"user_id": userUUID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to delete show"))
		return
	}

	utils.LogInfo(ctx, "Show deleted successfully", utils.Fields{
		"show_id": showID,
		"user_id": userUUID,
	})

	c.JSON(http.StatusOK, models.DeleteShowResponse{
		Success: true,
		Message: "Show deleted successfully",
	})
}

// ListShows godoc
// @Summary List shows
// @Description Get a paginated list of shows with filtering and sorting options
// @Tags shows
// @Accept json
// @Produce json
// @Param request body models.ListShowsRequest false "List options"
// @Success 200 {object} models.ListShowsResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/show/list [post]
// @Security ApiKeyAuth
func (h *ShowHandler) ListShows(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.ListShowsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use default values if no body provided
		req = models.ListShowsRequest{
			Pagination: models.PaginationOptions{
				Page:  1,
				Limit: 20,
			},
		}
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

	// Get shows from database
	shows, total, err := h.db.ListShows(ctx, userUUID, req.Filters, req.Pagination, req.Sort)
	if err != nil {
		utils.LogError(ctx, "Failed to list shows", err, utils.Fields{
			"user_id": userUUID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to retrieve shows"))
		return
	}

	// Convert to list items
	showItems := make([]models.ShowListItem, len(shows))
	for i, show := range shows {
		nextOccurrences := utils.CalculateNextOccurrences(&show, 3) // Get next 3 occurrences
		var nextOccurrence *time.Time
		if len(nextOccurrences) > 0 {
			nextOccurrence = &nextOccurrences[0]
		}
		
		showItems[i] = models.ShowListItem{
			ID:               show.ID,
			ShowName:         show.ShowName,
			StartTime:        show.StartTime,
			LengthMinutes:    show.LengthMinutes,
			FirstEventDate:   show.FirstEventDate,
			RepeatPattern:    show.RepeatPattern,
			SchedulingConfig: show.SchedulingConfig,
			Status:           show.Status,
			HasZoomMeeting:   show.ZoomMeetingURL != nil && *show.ZoomMeetingURL != "",
			NextOccurrence:   nextOccurrence,
			NextOccurrences:  nextOccurrences,
		}
	}

	// Calculate pagination
	totalPages := (total + req.Pagination.Limit - 1) / req.Pagination.Limit

	c.JSON(http.StatusOK, models.ListShowsResponse{
		Success: true,
		Data: &models.ListShowsData{
			Shows: showItems,
			Pagination: models.PaginationResponse{
				Page:       req.Pagination.Page,
				Limit:      req.Pagination.Limit,
				Total:      total,
				TotalPages: totalPages,
			},
		},
	})
}

// GetShowInfo godoc
// @Summary Get show information
// @Description Get detailed information about a specific show including upcoming events
// @Tags shows
// @Produce json
// @Param show_id path string true "Show ID"
// @Success 200 {object} models.GetShowInfoResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/show/info/{show_id} [get]
// @Security ApiKeyAuth
func (h *ShowHandler) GetShowInfo(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse show ID from path
	showIDParam := c.Param("show_id")
	showID, err := uuid.Parse(showIDParam)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid show ID format", map[string]interface{}{
			"field": "show_id",
			"value": showIDParam,
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

	// Calculate upcoming events using advanced scheduling
	upcomingOccurrences := utils.CalculateNextOccurrences(show, 5)
	upcomingEvents := make([]models.ShowEvent, len(upcomingOccurrences))
	for i, occurrence := range upcomingOccurrences {
		endTime := occurrence.Add(time.Duration(show.LengthMinutes) * time.Minute)
		upcomingEvents[i] = models.ShowEvent{
			Date:          occurrence.Format("2006-01-02"),
			StartDateTime: occurrence,
			EndDateTime:   endTime,
		}
	}

	c.JSON(http.StatusOK, models.GetShowInfoResponse{
		Success: true,
		Data: &models.ShowInfoData{
			Show:           show,
			UpcomingEvents: upcomingEvents,
		},
	})
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

