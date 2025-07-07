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

// EventHandler handles event-related HTTP requests
type EventHandler struct {
	db *database.PostgresDB
}

// NewEventHandler creates a new event handler
func NewEventHandler(db *database.PostgresDB) *EventHandler {
	return &EventHandler{db: db}
}

// UpdateEvent handles PUT /api/v1/event/update
// @Summary Update event details
// @Description Update specific event with custom fields while preserving show inheritance
// @Tags events
// @Accept json
// @Produce json
// @Param request body models.UpdateEventRequest true "Event update data"
// @Success 200 {object} models.UpdateEventResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/event/update [put]
func (h *EventHandler) UpdateEvent(c *gin.Context) {
	var req models.UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Parse event ID
	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID format"})
		return
	}

	// Get existing event
	event, err := h.db.GetEventByID(c.Request.Context(), eventID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get event", err, utils.Fields{
			"event_id": eventID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve event"})
		return
	}

	if event == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Get the associated show for validation
	show, err := h.db.GetShowByID(c.Request.Context(), event.ShowID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get show", err, utils.Fields{
			"show_id": event.ShowID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve show"})
		return
	}

	if show == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Associated show not found"})
		return
	}

	// Update event fields
	updated := false

	if req.EventTitle != nil {
		event.EventTitle = req.EventTitle
		updated = true
	}

	if req.EventDescription != nil {
		event.EventDescription = req.EventDescription
		updated = true
	}

	if req.StartDateTime != nil {
		event.StartDateTime = *req.StartDateTime
		// Recalculate end time
		duration := show.LengthMinutes
		if event.LengthMinutes != nil {
			duration = *event.LengthMinutes
		}
		event.EndDateTime = event.StartDateTime.Add(time.Duration(duration) * time.Minute)
		updated = true
	}

	if req.LengthMinutes != nil {
		event.LengthMinutes = req.LengthMinutes
		// Recalculate end time
		event.EndDateTime = event.StartDateTime.Add(time.Duration(*req.LengthMinutes) * time.Minute)
		updated = true
	}

	if req.YouTubeKey != nil {
		event.YouTubeKey = req.YouTubeKey
		updated = true
	}

	if req.AdditionalKey != nil {
		event.AdditionalKey = req.AdditionalKey
		updated = true
	}

	if req.ZoomMeetingURL != nil {
		event.ZoomMeetingURL = req.ZoomMeetingURL
		updated = true
	}

	if req.ZoomMeetingID != nil {
		event.ZoomMeetingID = req.ZoomMeetingID
		updated = true
	}

	if req.ZoomPasscode != nil {
		event.ZoomPasscode = req.ZoomPasscode
		updated = true
	}

	if req.CustomFields != nil {
		if event.CustomFields == nil {
			event.CustomFields = make(map[string]interface{})
		}
		for key, value := range req.CustomFields {
			event.CustomFields[key] = value
		}
		updated = true
	}

	if updated {
		event.IsCustomized = true
	}

	// Validate updated event
	if err := utils.ValidateEventTiming(event, show); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Event validation failed", "details": err.Error()})
		return
	}

	// Update in database
	if err := h.db.UpdateEvent(c.Request.Context(), event); err != nil {
		utils.LogError(c.Request.Context(), "Failed to update event", err, utils.Fields{
			"event_id": eventID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update event"})
		return
	}

	utils.LogInfo(c.Request.Context(), "Event updated successfully", utils.Fields{
		"event_id":      eventID,
		"is_customized": event.IsCustomized,
	})

	c.JSON(http.StatusOK, models.UpdateEventResponse{
		Success: true,
		Data:    event,
	})
}

// DeleteEvent handles DELETE /api/v1/event/delete
// @Summary Cancel/delete event
// @Description Cancel a specific event while preserving show template
// @Tags events
// @Accept json
// @Produce json
// @Param request body models.DeleteEventRequest true "Event deletion data"
// @Success 200 {object} models.DeleteEventResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/event/delete [delete]
func (h *EventHandler) DeleteEvent(c *gin.Context) {
	var req models.DeleteEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Parse event ID
	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID format"})
		return
	}

	// Get existing event to verify it exists
	event, err := h.db.GetEventByID(c.Request.Context(), eventID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get event", err, utils.Fields{
			"event_id": eventID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve event"})
		return
	}

	if event == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Delete (cancel) the event
	if err := h.db.DeleteEvent(c.Request.Context(), eventID); err != nil {
		utils.LogError(c.Request.Context(), "Failed to delete event", err, utils.Fields{
			"event_id": eventID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel event"})
		return
	}

	utils.LogInfo(c.Request.Context(), "Event cancelled successfully", utils.Fields{
		"event_id":             eventID,
		"cancellation_reason": req.CancellationReason,
	})

	c.JSON(http.StatusOK, models.DeleteEventResponse{
		Success: true,
		Message: "Event cancelled successfully",
		Data: &models.EventDeleteData{
			EventID:     req.EventID,
			Status:      models.EventStatusCancelled,
			CancelledAt: time.Now(),
		},
	})
}

// ListEvents handles POST /api/v1/event/list
// @Summary List events with filtering
// @Description Get paginated list of events with filtering and sorting options
// @Tags events
// @Accept json
// @Produce json
// @Param request body models.EventListRequest true "Event list filters"
// @Success 200 {object} models.EventListResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/event/list [post]
func (h *EventHandler) ListEvents(c *gin.Context) {
	var req models.EventListRequest
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

	// Set defaults
	if req.Pagination.Limit <= 0 {
		req.Pagination.Limit = 20
	}
	if req.Pagination.Page <= 0 {
		req.Pagination.Page = 1
	}

	// Get events from database
	events, total, err := h.db.ListEvents(c.Request.Context(), userUUID, req.Filters, req.Pagination, req.Sort)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to list events", err, utils.Fields{
			"user_id": userUUID,
			"filters": req.Filters,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve events"})
		return
	}

	// Calculate pagination
	totalPages := (total + req.Pagination.Limit - 1) / req.Pagination.Limit

	c.JSON(http.StatusOK, models.EventListResponse{
		Success: true,
		Data: &models.EventListData{
			Events: events,
			Pagination: models.PaginationResponse{
				Page:       req.Pagination.Page,
				Limit:      req.Pagination.Limit,
				Total:      total,
				TotalPages: totalPages,
			},
		},
	})
}

// WeekListEvents handles POST /api/v1/event/weekList
// @Summary Get week view of events
// @Description Get events organized by days for a specific week
// @Tags events
// @Accept json
// @Produce json
// @Param request body models.WeekListRequest true "Week view request"
// @Success 200 {object} models.WeekListResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/event/weekList [post]
func (h *EventHandler) WeekListEvents(c *gin.Context) {
	var req models.WeekListRequest
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

	// Parse week start date
	weekStart, err := time.Parse("2006-01-02", req.WeekStart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format, expected YYYY-MM-DD"})
		return
	}

	// Ensure it's Monday (start of week)
	for weekStart.Weekday() != time.Monday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}

	// Set timezone
	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	// Get events for the week
	events, err := h.db.GetWeekEvents(c.Request.Context(), userUUID, weekStart, req.Filters)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get week events", err, utils.Fields{
			"user_id":    userUUID,
			"week_start": weekStart,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve week events"})
		return
	}

	// Organize events by day
	days := make([]models.WeekDay, 7)
	weekDays := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

	for i := 0; i < 7; i++ {
		dayDate := weekStart.AddDate(0, 0, i)
		days[i] = models.WeekDay{
			Date:    dayDate.Format("2006-01-02"),
			DayName: weekDays[i],
			Events:  []models.WeekDayEvent{},
		}
	}

	// Distribute events to days
	for _, event := range events {
		// TODO: Fix this - need to parse start time and determine day
		// For now, we'll distribute based on event start time
		_ = event // Suppress unused variable warning
	}

	weekEnd := weekStart.AddDate(0, 0, 6)

	c.JSON(http.StatusOK, models.WeekListResponse{
		Success: true,
		Data: &models.WeekListData{
			WeekStart:   weekStart.Format("2006-01-02"),
			WeekEnd:     weekEnd.Format("2006-01-02"),
			Timezone:    timezone,
			Days:        days,
			TotalEvents: len(events),
		},
	})
}

// MonthListEvents handles POST /api/v1/event/monthList
// @Summary Get month view of events
// @Description Get events organized by weeks and days for a specific month
// @Tags events
// @Accept json
// @Produce json
// @Param request body models.MonthListRequest true "Month view request"
// @Success 200 {object} models.MonthListResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/event/monthList [post]
func (h *EventHandler) MonthListEvents(c *gin.Context) {
	var req models.MonthListRequest
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

	// Set timezone
	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	// Get events for the month
	events, err := h.db.GetMonthEvents(c.Request.Context(), userUUID, req.Year, req.Month, req.Filters)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get month events", err, utils.Fields{
			"user_id": userUUID,
			"year":    req.Year,
			"month":   req.Month,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve month events"})
		return
	}

	// Create month structure
	monthNames := []string{"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}

	// Count events by status
	eventsByStatus := make(map[models.EventStatus]int)
	for _, event := range events {
		eventsByStatus[event.Status]++
	}

	// TODO: Implement proper week/day organization
	// This is a simplified version - need to properly organize events by calendar weeks
	weeks := []models.MonthWeek{
		{
			WeekNumber: 1,
			Days:       []models.MonthDay{},
		},
	}

	c.JSON(http.StatusOK, models.MonthListResponse{
		Success: true,
		Data: &models.MonthListData{
			Year:           req.Year,
			Month:          req.Month,
			MonthName:      monthNames[req.Month],
			Timezone:       timezone,
			Weeks:          weeks,
			TotalEvents:    len(events),
			EventsByStatus: eventsByStatus,
		},
	})
}

// GetEventInfo handles GET /api/v1/event/info/{event_id}
// @Summary Get event details
// @Description Get detailed information about a specific event including show details
// @Tags events
// @Produce json
// @Param event_id path string true "Event ID"
// @Success 200 {object} models.GetEventInfoResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/event/info/{event_id} [get]
func (h *EventHandler) GetEventInfo(c *gin.Context) {
	eventIDStr := c.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID format"})
		return
	}

	// Get event
	event, err := h.db.GetEventByID(c.Request.Context(), eventID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get event", err, utils.Fields{
			"event_id": eventID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve event"})
		return
	}

	if event == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Get associated show
	show, err := h.db.GetShowByID(c.Request.Context(), event.ShowID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get show", err, utils.Fields{
			"show_id": event.ShowID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve show"})
		return
	}

	if show == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Associated show not found"})
		return
	}

	showSummary := &models.ShowSummary{
		ID:            show.ID,
		ShowName:      show.ShowName,
		RepeatPattern: show.RepeatPattern,
		Status:        show.Status,
	}

	c.JSON(http.StatusOK, models.GetEventInfoResponse{
		Success: true,
		Data: &models.EventInfoData{
			Event:       event,
			ShowDetails: showSummary,
		},
	})
}