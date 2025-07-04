package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/denisAlshanov/stPlaner/internal/database"
	"github.com/denisAlshanov/stPlaner/internal/models"
	"github.com/denisAlshanov/stPlaner/internal/services/downloader"
	"github.com/denisAlshanov/stPlaner/internal/utils"
)

type PostHandler struct {
	db         *database.MongoDB
	downloader *downloader.Downloader
}

func NewPostHandler(db *database.MongoDB, downloader *downloader.Downloader) *PostHandler {
	return &PostHandler{
		db:         db,
		downloader: downloader,
	}
}

// AddPost godoc
// @Summary Add a new Telegram post for processing
// @Description Add a new Telegram post link to download media
// @Tags posts
// @Accept json
// @Produce json
// @Param request body models.AddPostRequest true "Post link"
// @Success 200 {object} models.AddPostResponse
// @Success 202 {object} models.AddPostResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /add [post]
// @Security ApiKeyAuth
func (h *PostHandler) AddPost(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.AddPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, utils.NewValidationError("Invalid request body", map[string]interface{}{
			"error": err.Error(),
		}))
		return
	}

	// Process the post
	post, err := h.downloader.ProcessPost(ctx, req.Link)
	if err != nil {
		if appErr, ok := err.(*utils.AppError); ok {
			h.errorResponse(c, appErr)
		} else {
			utils.LogError(ctx, "Failed to process post", err)
			h.errorResponse(c, utils.NewInternalError())
		}
		return
	}

	response := models.AddPostResponse{
		Status:           "success",
		Message:          "Post added for processing",
		PostID:           post.PostID,
		MediaCount:       post.MediaCount,
		ProcessingStatus: post.Status,
	}

	statusCode := http.StatusOK
	if post.Status == models.PostStatusPending || post.Status == models.PostStatusProcessing {
		statusCode = http.StatusAccepted
	}

	c.JSON(statusCode, response)
}

// GetList godoc
// @Summary Get list of processed posts
// @Description Retrieve list of all previously processed Telegram links
// @Tags posts
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param sort query string false "Sort order" Enums(created_at_desc, created_at_asc)
// @Success 200 {object} models.PostListResponse
// @Failure 500 {object} map[string]interface{}
// @Router /getList [get]
// @Security ApiKeyAuth
func (h *PostHandler) GetList(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	sort := c.DefaultQuery("sort", "created_at_desc")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Prepare sort options
	sortField := "created_at"
	sortOrder := -1 // descending
	if sort == "created_at_asc" {
		sortOrder = 1
	}

	// Count total documents
	total, err := h.db.Posts().CountDocuments(ctx, bson.M{})
	if err != nil {
		utils.LogError(ctx, "Failed to count posts", err)
		h.errorResponse(c, utils.NewDatabaseError(err))
		return
	}

	// Prepare find options
	skip := (page - 1) * limit
	findOptions := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(skip)).
		SetSort(bson.D{{Key: sortField, Value: sortOrder}})

	// Find posts
	cursor, err := h.db.Posts().Find(ctx, bson.M{}, findOptions)
	if err != nil {
		utils.LogError(ctx, "Failed to find posts", err)
		h.errorResponse(c, utils.NewDatabaseError(err))
		return
	}
	defer cursor.Close(ctx)

	// Decode results
	var posts []models.Post
	if err := cursor.All(ctx, &posts); err != nil {
		utils.LogError(ctx, "Failed to decode posts", err)
		h.errorResponse(c, utils.NewDatabaseError(err))
		return
	}

	// Convert to response format
	links := make([]models.PostListItem, len(posts))
	for i, post := range posts {
		links[i] = models.PostListItem{
			PostID:     post.PostID,
			Link:       post.TelegramLink,
			AddedAt:    post.CreatedAt,
			MediaCount: post.MediaCount,
			Status:     post.Status,
		}
	}

	response := models.PostListResponse{
		Total: int(total),
		Page:  page,
		Limit: limit,
		Links: links,
	}

	c.JSON(http.StatusOK, response)
}

func (h *PostHandler) errorResponse(c *gin.Context, err *utils.AppError) {
	c.JSON(err.StatusCode, gin.H{
		"error":      err,
		"request_id": c.GetString("request_id"),
		"timestamp":  time.Now().Format(time.RFC3339),
	})
}
