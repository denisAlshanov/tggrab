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

// BlockHandler handles block-related HTTP requests
type BlockHandler struct {
	db *database.PostgresDB
}

// NewBlockHandler creates a new block handler
func NewBlockHandler(db *database.PostgresDB) *BlockHandler {
	return &BlockHandler{db: db}
}

// AddBlock handles POST /api/v1/block/add
// @Summary Add new block to event
// @Description Create a new block with guests and media attachments
// @Tags blocks
// @Accept json
// @Produce json
// @Param request body models.AddBlockRequest true "Block creation data"
// @Success 200 {object} models.AddBlockResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /api/v1/block/add [post]
func (h *BlockHandler) AddBlock(c *gin.Context) {
	var req models.AddBlockRequest
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

	// Parse event ID
	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID format"})
		return
	}

	// Verify event exists and belongs to user
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

	if event.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Parse guest IDs
	var guestIDs []uuid.UUID
	for _, guestIDStr := range req.GuestIDs {
		guestID, err := uuid.Parse(guestIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid guest ID format", "guest_id": guestIDStr})
			return
		}
		guestIDs = append(guestIDs, guestID)
	}

	// Validate block type
	if req.BlockType == "" {
		req.BlockType = models.BlockTypeCustom
	}
	if !isValidBlockType(req.BlockType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid block type", "block_type": req.BlockType})
		return
	}

	// Create block object
	block := &models.Block{
		EventID:         eventID,
		UserID:          userUUID,
		Title:           strings.TrimSpace(req.Title),
		Description:     req.Description,
		Topic:           req.Topic,
		EstimatedLength: req.EstimatedLength,
		OrderIndex:      req.OrderIndex,
		BlockType:       req.BlockType,
		Status:          models.BlockStatusPlanned,
		Metadata:        req.Metadata,
	}

	// Create block in database
	if err := h.db.CreateBlock(c.Request.Context(), block, guestIDs, req.Media); err != nil {
		if strings.Contains(err.Error(), "unique_event_order") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Order index already exists for this event"})
			return
		}
		utils.LogError(c.Request.Context(), "Failed to create block", err, utils.Fields{
			"event_id": eventID,
			"user_id":  userUUID,
			"title":    block.Title,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create block"})
		return
	}

	// Get the full block details for response
	blockDetail, err := h.db.GetBlockByID(c.Request.Context(), block.ID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get created block", err, utils.Fields{
			"block_id": block.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Block created but failed to retrieve details"})
		return
	}

	utils.LogInfo(c.Request.Context(), "Block created successfully", utils.Fields{
		"block_id": block.ID,
		"event_id": eventID,
		"user_id":  userUUID,
		"title":    block.Title,
	})

	c.JSON(http.StatusOK, models.AddBlockResponse{
		Success: true,
		Data:    blockDetail,
	})
}

// UpdateBlock handles PUT /api/v1/block/update
// @Summary Update block information
// @Description Update existing block with new information
// @Tags blocks
// @Accept json
// @Produce json
// @Param request body models.UpdateBlockRequest true "Block update data"
// @Success 200 {object} models.UpdateBlockResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /api/v1/block/update [put]
func (h *BlockHandler) UpdateBlock(c *gin.Context) {
	var req models.UpdateBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Parse block ID
	blockID, err := uuid.Parse(req.BlockID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid block ID format"})
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

	// Get existing block
	blockDetail, err := h.db.GetBlockByID(c.Request.Context(), blockID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get block", err, utils.Fields{
			"block_id": blockID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve block"})
		return
	}

	if blockDetail == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Block not found"})
		return
	}

	// Verify ownership
	if blockDetail.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Update fields
	block := &blockDetail.Block
	updated := false

	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed != "" && trimmed != block.Title {
			block.Title = trimmed
			updated = true
		}
	}

	if req.Description != nil {
		block.Description = req.Description
		updated = true
	}

	if req.Topic != nil {
		block.Topic = req.Topic
		updated = true
	}

	if req.EstimatedLength != nil && *req.EstimatedLength > 0 {
		block.EstimatedLength = *req.EstimatedLength
		updated = true
	}

	if req.ActualLength != nil {
		block.ActualLength = req.ActualLength
		updated = true
	}

	if req.BlockType != nil && isValidBlockType(*req.BlockType) {
		block.BlockType = *req.BlockType
		updated = true
	}

	if req.Status != nil && isValidBlockStatus(*req.Status) {
		block.Status = *req.Status
		updated = true
	}

	if req.Metadata != nil {
		block.Metadata = req.Metadata
		updated = true
	}

	// Parse guest IDs
	var guestIDs []uuid.UUID
	if req.GuestIDs != nil {
		for _, guestIDStr := range req.GuestIDs {
			guestID, err := uuid.Parse(guestIDStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid guest ID format", "guest_id": guestIDStr})
				return
			}
			guestIDs = append(guestIDs, guestID)
		}
		updated = true
	}

	// Prepare media
	var media []models.BlockMediaInput
	if req.Media != nil {
		media = req.Media
		updated = true
	}

	if !updated {
		c.JSON(http.StatusOK, models.UpdateBlockResponse{
			Success: true,
			Data:    blockDetail,
		})
		return
	}

	// Update in database
	if err := h.db.UpdateBlock(c.Request.Context(), block, guestIDs, media); err != nil {
		utils.LogError(c.Request.Context(), "Failed to update block", err, utils.Fields{
			"block_id": blockID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update block"})
		return
	}

	// Get updated block details
	updatedBlockDetail, err := h.db.GetBlockByID(c.Request.Context(), blockID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get updated block", err, utils.Fields{
			"block_id": blockID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Block updated but failed to retrieve details"})
		return
	}

	utils.LogInfo(c.Request.Context(), "Block updated successfully", utils.Fields{
		"block_id": blockID,
		"user_id":  userUUID,
	})

	c.JSON(http.StatusOK, models.UpdateBlockResponse{
		Success: true,
		Data:    updatedBlockDetail,
	})
}

// GetBlockInfo handles GET /api/v1/block/info/{block_id}
// @Summary Get block details
// @Description Get detailed information about a specific block including guests and media
// @Tags blocks
// @Produce json
// @Param block_id path string true "Block ID"
// @Success 200 {object} models.GetBlockInfoResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /api/v1/block/info/{block_id} [get]
func (h *BlockHandler) GetBlockInfo(c *gin.Context) {
	blockIDStr := c.Param("block_id")
	blockID, err := uuid.Parse(blockIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid block ID format"})
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

	// Get block
	blockDetail, err := h.db.GetBlockByID(c.Request.Context(), blockID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get block", err, utils.Fields{
			"block_id": blockID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve block"})
		return
	}

	if blockDetail == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Block not found"})
		return
	}

	// Verify ownership
	if blockDetail.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get event summary
	eventSummary, err := h.db.GetEventSummary(c.Request.Context(), blockDetail.EventID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get event summary", err, utils.Fields{
			"event_id": blockDetail.EventID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve event information"})
		return
	}

	c.JSON(http.StatusOK, models.GetBlockInfoResponse{
		Success: true,
		Data: &models.BlockInfoData{
			Block:     blockDetail,
			EventInfo: eventSummary,
		},
	})
}

// ReorderBlocks handles PUT /api/v1/block/reorder
// @Summary Reorder blocks within an event
// @Description Change the order of blocks within a specific event
// @Tags blocks
// @Accept json
// @Produce json
// @Param request body models.ReorderBlocksRequest true "Block reordering data"
// @Success 200 {object} models.ReorderBlocksResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /api/v1/block/reorder [put]
func (h *BlockHandler) ReorderBlocks(c *gin.Context) {
	var req models.ReorderBlocksRequest
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

	// Verify event exists and belongs to user
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

	if event.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Validate order continuity
	if err := validateBlockOrdering(req.BlockOrders); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid block ordering", "details": err.Error()})
		return
	}

	// Reorder blocks in database
	if err := h.db.ReorderBlocks(c.Request.Context(), eventID, req.BlockOrders); err != nil {
		utils.LogError(c.Request.Context(), "Failed to reorder blocks", err, utils.Fields{
			"event_id": eventID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder blocks"})
		return
	}

	// Get updated block list
	blocks, err := h.db.GetEventBlocks(c.Request.Context(), eventID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get updated blocks", err, utils.Fields{
			"event_id": eventID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Blocks reordered but failed to retrieve updated list"})
		return
	}

	// Calculate total estimated time
	var totalEstimatedTime int
	var blockSummaries []models.BlockOrderSummary
	for _, block := range blocks {
		totalEstimatedTime += block.EstimatedLength
		blockSummaries = append(blockSummaries, models.BlockOrderSummary{
			BlockID:    block.ID.String(),
			Title:      block.Title,
			OrderIndex: block.OrderIndex,
		})
	}

	utils.LogInfo(c.Request.Context(), "Blocks reordered successfully", utils.Fields{
		"event_id":    eventID,
		"user_id":     userUUID,
		"block_count": len(req.BlockOrders),
	})

	c.JSON(http.StatusOK, models.ReorderBlocksResponse{
		Success: true,
		Data: &models.ReorderBlocksData{
			EventID:            req.EventID,
			Blocks:             blockSummaries,
			TotalEstimatedTime: totalEstimatedTime,
		},
	})
}

// DeleteBlock handles DELETE /api/v1/block/delete
// @Summary Delete block from event
// @Description Remove a block and optionally reorder remaining blocks
// @Tags blocks
// @Accept json
// @Produce json
// @Param request body models.DeleteBlockRequest true "Block deletion data"
// @Success 200 {object} models.DeleteBlockResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /api/v1/block/delete [delete]
func (h *BlockHandler) DeleteBlock(c *gin.Context) {
	var req models.DeleteBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Parse block ID
	blockID, err := uuid.Parse(req.BlockID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid block ID format"})
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

	// Get existing block to verify ownership
	blockDetail, err := h.db.GetBlockByID(c.Request.Context(), blockID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get block", err, utils.Fields{
			"block_id": blockID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve block"})
		return
	}

	if blockDetail == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Block not found"})
		return
	}

	// Verify ownership
	if blockDetail.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Delete block
	if err := h.db.DeleteBlock(c.Request.Context(), blockID, req.ReorderRemaining); err != nil {
		utils.LogError(c.Request.Context(), "Failed to delete block", err, utils.Fields{
			"block_id": blockID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete block"})
		return
	}

	utils.LogInfo(c.Request.Context(), "Block deleted successfully", utils.Fields{
		"block_id":          blockID,
		"user_id":           userUUID,
		"reorder_remaining": req.ReorderRemaining,
	})

	c.JSON(http.StatusOK, models.DeleteBlockResponse{
		Success: true,
		Message: "Block deleted successfully",
		Data: &models.DeleteBlockData{
			BlockID:                  req.BlockID,
			DeletedAt:                time.Now(),
			RemainingBlocksReordered: req.ReorderRemaining,
		},
	})
}

// GetEventBlocks handles GET /api/v1/event/{event_id}/blocks
// @Summary List all blocks for an event
// @Description Get ordered list of blocks for a specific event
// @Tags blocks
// @Produce json
// @Param event_id path string true "Event ID"
// @Success 200 {object} models.EventBlocksResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security ApiKeyAuth
// @Router /api/v1/event/{event_id}/blocks [get]
func (h *BlockHandler) GetEventBlocks(c *gin.Context) {
	eventIDStr := c.Param("event_id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event ID format"})
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

	// Verify event exists and belongs to user
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

	if event.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get blocks
	blocks, err := h.db.GetEventBlocks(c.Request.Context(), eventID)
	if err != nil {
		utils.LogError(c.Request.Context(), "Failed to get event blocks", err, utils.Fields{
			"event_id": eventID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve blocks"})
		return
	}

	// Calculate totals
	var totalEstimatedTime, totalActualTime int
	for _, block := range blocks {
		totalEstimatedTime += block.EstimatedLength
		if block.ActualLength != nil {
			totalActualTime += *block.ActualLength
		}
	}

	c.JSON(http.StatusOK, models.EventBlocksResponse{
		Success: true,
		Data: &models.EventBlocksData{
			EventID:            eventIDStr,
			Blocks:             blocks,
			TotalBlocks:        len(blocks),
			TotalEstimatedTime: totalEstimatedTime,
			TotalActualTime:    totalActualTime,
		},
	})
}

// Helper functions

// isValidBlockType checks if a block type is valid
func isValidBlockType(blockType models.BlockType) bool {
	switch blockType {
	case models.BlockTypeIntro,
		models.BlockTypeMain,
		models.BlockTypeInterview,
		models.BlockTypeQA,
		models.BlockTypeBreak,
		models.BlockTypeOutro,
		models.BlockTypeCustom:
		return true
	default:
		return false
	}
}

// isValidBlockStatus checks if a block status is valid
func isValidBlockStatus(status models.BlockStatus) bool {
	switch status {
	case models.BlockStatusPlanned,
		models.BlockStatusReady,
		models.BlockStatusInProgress,
		models.BlockStatusCompleted,
		models.BlockStatusSkipped:
		return true
	default:
		return false
	}
}

// validateBlockOrdering validates the continuity of block ordering
func validateBlockOrdering(blockOrders []models.BlockOrder) error {
	// Check for gaps and duplicates in order
	orderMap := make(map[int]bool)
	for _, bo := range blockOrders {
		if orderMap[bo.OrderIndex] {
			return utils.NewValidationError("duplicate order index", map[string]interface{}{
				"order_index": bo.OrderIndex,
			})
		}
		orderMap[bo.OrderIndex] = true
	}

	// Ensure continuous ordering starting from 0
	for i := 0; i < len(blockOrders); i++ {
		if !orderMap[i] {
			return utils.NewValidationError("missing order index", map[string]interface{}{
				"missing_index": i,
				"expected_range": "0 to " + string(rune(len(blockOrders)-1)),
			})
		}
	}

	return nil
}