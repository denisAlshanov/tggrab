package handlers

import (
	"context"
	"fmt"
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

// EventHandler handles event-related HTTP requests
type EventHandler struct {
	db *database.PostgresDB
}

// NewEventHandler creates a new event handler
func NewEventHandler(db *database.PostgresDB) *EventHandler {
	return &EventHandler{db: db}
}







// RESTful Event Management Endpoints

// UpdateEventREST handles PUT /api/v1/events/{event_id}
// @Summary Update event (RESTful)
// @Description Update an event with simplified request format and staff assignments
// @Tags events
// @Accept json
// @Produce json
// @Param event_id path string true "Event ID"
// @Param request body models.UpdateEventRequestREST true "Event update data"
// @Success 200 {object} models.EventResponseREST
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/events/{event_id} [put]
func (h *EventHandler) UpdateEventREST(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse event ID from path
	eventIDStr := c.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid event ID format", map[string]interface{}{
			"field": "event_id",
			"value": eventIDStr,
		}))
		return
	}

	var req models.UpdateEventRequestREST
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

	// Check if event exists and belongs to user
	existingEvent, err := h.db.GetEventByID(ctx, eventID)
	if err != nil {
		utils.LogError(ctx, "Failed to get event", err, utils.Fields{
			"event_id": eventID,
			"user_id":  userUUID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to retrieve event"))
		return
	}

	if existingEvent == nil {
		h.errorResponse(c, utils.NewNotFoundError("Event not found"))
		return
	}

	if existingEvent.UserID != userUUID {
		h.errorResponse(c, utils.NewForbiddenError("Access denied"))
		return
	}

	// Validate user IDs if provided
	if req.Host != nil {
		_, err := h.validateUserIDs(ctx, req.Host)
		if err != nil {
			h.errorResponse(c, utils.NewValidationError("Invalid host user IDs", map[string]interface{}{
				"field": "host",
				"error": err.Error(),
			}))
			return
		}
	}

	if req.Director != nil {
		_, err := h.validateUserIDs(ctx, req.Director)
		if err != nil {
			h.errorResponse(c, utils.NewValidationError("Invalid director user IDs", map[string]interface{}{
				"field": "director",
				"error": err.Error(),
			}))
			return
		}
	}

	if req.Producer != nil {
		_, err := h.validateUserIDs(ctx, req.Producer)
		if err != nil {
			h.errorResponse(c, utils.NewValidationError("Invalid producer user IDs", map[string]interface{}{
				"field": "producer",
				"error": err.Error(),
			}))
			return
		}
	}

	// Validate Telegram channel if provided
	if req.Telegram != nil && *req.Telegram != "" {
		if !h.isValidTelegramChannel(*req.Telegram) {
			h.errorResponse(c, utils.NewValidationError("Invalid Telegram channel format", map[string]interface{}{
				"field": "telegram",
				"value": *req.Telegram,
			}))
			return
		}
	}

	// Update event in database
	_, err = h.db.UpdateEventREST(ctx, eventID, &req)
	if err != nil {
		utils.LogError(ctx, "Failed to update event", err, utils.Fields{
			"event_id": eventID,
			"user_id":  userUUID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to update event"))
		return
	}

	// Get updated event details
	eventDetail, err := h.db.GetEventREST(ctx, eventID)
	if err != nil {
		utils.LogError(ctx, "Failed to get updated event", err, utils.Fields{
			"event_id": eventID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to retrieve updated event"))
		return
	}

	utils.LogInfo(ctx, "Event updated successfully", utils.Fields{
		"event_id": eventID,
		"user_id":  userUUID,
	})

	c.JSON(http.StatusOK, models.EventResponseREST{
		Success: true,
		Data:    eventDetail,
	})
}

// DeleteEventREST handles DELETE /api/v1/events/{event_id}
// @Summary Delete event (RESTful)
// @Description Delete an event with optional force parameter
// @Tags events
// @Produce json
// @Param event_id path string true "Event ID"
// @Param force query bool false "Force hard delete"
// @Success 200 {object} models.DeleteEventResponseREST
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/events/{event_id} [delete]
func (h *EventHandler) DeleteEventREST(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse event ID from path
	eventIDStr := c.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid event ID format", map[string]interface{}{
			"field": "event_id",
			"value": eventIDStr,
		}))
		return
	}

	// Parse force parameter
	force := c.Query("force") == "true"

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

	// Check if event exists and belongs to user
	event, err := h.db.GetEventByID(ctx, eventID)
	if err != nil {
		utils.LogError(ctx, "Failed to get event", err, utils.Fields{
			"event_id": eventID,
			"user_id":  userUUID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to retrieve event"))
		return
	}

	if event == nil {
		h.errorResponse(c, utils.NewNotFoundError("Event not found"))
		return
	}

	if event.UserID != userUUID {
		h.errorResponse(c, utils.NewForbiddenError("Access denied"))
		return
	}

	// Delete event
	err = h.db.DeleteEventREST(ctx, eventID, force)
	if err != nil {
		utils.LogError(ctx, "Failed to delete event", err, utils.Fields{
			"event_id": eventID,
			"user_id":  userUUID,
			"force":    force,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to delete event"))
		return
	}

	utils.LogInfo(ctx, "Event deleted successfully", utils.Fields{
		"event_id": eventID,
		"user_id":  userUUID,
		"force":    force,
	})

	message := "Event cancelled successfully"
	if force {
		message = "Event deleted permanently"
	}

	c.JSON(http.StatusOK, models.DeleteEventResponseREST{
		Success: true,
		Message: message,
		Data: &models.EventDeleteDataREST{
			EventID:   eventIDStr,
			DeletedAt: time.Now(),
		},
	})
}

// GetEventREST handles GET /api/v1/events/{event_id}
// @Summary Get event details (RESTful)
// @Description Get detailed information about a specific event
// @Tags events
// @Produce json
// @Param event_id path string true "Event ID"
// @Success 200 {object} models.EventResponseREST
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/events/{event_id} [get]
func (h *EventHandler) GetEventREST(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse event ID from path
	eventIDStr := c.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid event ID format", map[string]interface{}{
			"field": "event_id",
			"value": eventIDStr,
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

	// Check if event exists and belongs to user
	event, err := h.db.GetEventByID(ctx, eventID)
	if err != nil {
		utils.LogError(ctx, "Failed to get event", err, utils.Fields{
			"event_id": eventID,
			"user_id":  userUUID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to retrieve event"))
		return
	}

	if event == nil {
		h.errorResponse(c, utils.NewNotFoundError("Event not found"))
		return
	}

	if event.UserID != userUUID {
		h.errorResponse(c, utils.NewForbiddenError("Access denied"))
		return
	}

	// Get detailed event information
	eventDetail, err := h.db.GetEventREST(ctx, eventID)
	if err != nil {
		utils.LogError(ctx, "Failed to get event details", err, utils.Fields{
			"event_id": eventID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to retrieve event details"))
		return
	}

	c.JSON(http.StatusOK, models.EventResponseREST{
		Success: true,
		Data:    eventDetail,
	})
}

// ListEventsREST handles GET /api/v1/events
// @Summary List events (RESTful)
// @Description Get paginated list of events with enhanced filtering
// @Tags events
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Param event_year query int false "Filter by year (e.g., 2024)"
// @Param event_month query int false "Filter by month (1-12)"
// @Param event_week query int false "Filter by ISO week number (1-53)"
// @Param status query string false "Filter by status (scheduled, live, completed, cancelled, postponed)"
// @Param show_id query string false "Filter by show ID"
// @Param search query string false "Search in event names"
// @Param sort query string false "Sort field (event_date, event_name, created_at)"
// @Param order query string false "Sort order (asc, desc)"
// @Success 200 {object} models.EventListResponseREST
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /api/v1/events [get]
func (h *EventHandler) ListEventsREST(c *gin.Context) {
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

	// Parse pagination parameters
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

	// Parse time-based filters
	var eventYear, eventMonth, eventWeek *int

	if yearStr := c.Query("event_year"); yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil && year >= 2020 && year <= 2030 {
			eventYear = &year
		}
	}

	if monthStr := c.Query("event_month"); monthStr != "" {
		if month, err := strconv.Atoi(monthStr); err == nil && month >= 1 && month <= 12 {
			eventMonth = &month
		}
	}

	if weekStr := c.Query("event_week"); weekStr != "" {
		if week, err := strconv.Atoi(weekStr); err == nil && week >= 1 && week <= 53 {
			eventWeek = &week
		}
	}

	// Parse other filters
	status := models.EventStatus(c.Query("status"))
	showID := c.Query("show_id")
	search := c.Query("search")
	sortField := c.Query("sort")
	sortOrder := c.Query("order")

	// Validate status if provided
	if status != "" {
		validStatuses := map[models.EventStatus]bool{
			models.EventStatusScheduled: true,
			models.EventStatusLive:      true,
			models.EventStatusCompleted: true,
			models.EventStatusCancelled: true,
			models.EventStatusPostponed: true,
		}
		if !validStatuses[status] {
			h.errorResponse(c, utils.NewValidationError("Invalid status", map[string]interface{}{
				"field": "status",
				"value": status,
			}))
			return
		}
	}

	offset := (page - 1) * limit

	// Get events from database
	events, total, filters, err := h.db.ListEventsREST(ctx, userUUID, eventYear, eventMonth, eventWeek, status, showID, search, sortField, sortOrder, limit, offset)
	if err != nil {
		utils.LogError(ctx, "Failed to list events", err, utils.Fields{
			"user_id": userUUID,
		})
		h.errorResponse(c, utils.NewInternalErrorWithMessage("Failed to retrieve events"))
		return
	}

	// Calculate pagination
	totalPages := (total + limit - 1) / limit

	c.JSON(http.StatusOK, models.EventListResponseREST{
		Success: true,
		Data: &models.EventListDataREST{
			Events: events,
			Pagination: models.PaginationResponse{
				Page:       page,
				Limit:      limit,
				Total:      total,
				TotalPages: totalPages,
			},
			Filters: filters,
		},
	})
}

// Helper methods for RESTful handlers

func (h *EventHandler) errorResponse(c *gin.Context, err error) {
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

func (h *EventHandler) validateUserIDs(ctx context.Context, userIDStrings []string) ([]uuid.UUID, error) {
	userIDs := make([]uuid.UUID, len(userIDStrings))
	for i, userIDStr := range userIDStrings {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID format: %s", userIDStr)
		}
		userIDs[i] = userID
	}

	// Validate that all users exist
	summaries, err := h.db.GetUserSummaries(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to validate users: %w", err)
	}

	if len(summaries) != len(userIDs) {
		return nil, fmt.Errorf("one or more users not found")
	}

	return userIDs, nil
}

func (h *EventHandler) isValidTelegramChannel(channel string) bool {
	// Basic validation for Telegram channels
	if len(channel) == 0 || len(channel) > 50 {
		return false
	}
	// Should start with @ and contain valid characters
	if !strings.HasPrefix(channel, "@") {
		return false
	}
	// Rest of the channel name should be alphanumeric or underscores
	channelName := channel[1:]
	for _, char := range channelName {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') {
			return false
		}
	}
	return len(channelName) >= 1
}