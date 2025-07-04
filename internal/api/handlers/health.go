package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tggrab/tggrab/internal/database"
	"github.com/tggrab/tggrab/internal/services/storage"
	"github.com/tggrab/tggrab/internal/utils"
)

type HealthHandler struct {
	db      *database.MongoDB
	storage storage.StorageInterface
}

type HealthResponse struct {
	Status    string                   `json:"status"`
	Timestamp string                   `json:"timestamp"`
	Version   string                   `json:"version"`
	Services  map[string]ServiceHealth `json:"services"`
}

type ServiceHealth struct {
	Status       string `json:"status"`
	ResponseTime string `json:"response_time,omitempty"`
	Error        string `json:"error,omitempty"`
}

func NewHealthHandler(db *database.MongoDB, storage storage.StorageInterface) *HealthHandler {
	return &HealthHandler{
		db:      db,
		storage: storage,
	}
}

// Health godoc
// @Summary Health check endpoint
// @Description Check the health of the service and its dependencies
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Success 503 {object} HealthResponse
// @Router /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	ctx := c.Request.Context()

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   "1.0.0",
		Services:  make(map[string]ServiceHealth),
	}

	// Check MongoDB
	mongoHealth := h.checkMongoDB(ctx)
	response.Services["mongodb"] = mongoHealth

	// Check S3
	s3Health := h.checkS3(ctx)
	response.Services["s3"] = s3Health

	// Determine overall status
	overallHealthy := true
	for _, service := range response.Services {
		if service.Status != "healthy" {
			overallHealthy = false
			break
		}
	}

	if !overallHealthy {
		response.Status = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// Readiness godoc
// @Summary Readiness check endpoint
// @Description Check if the service is ready to accept requests
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Success 503 {object} map[string]interface{}
// @Router /ready [get]
func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx := c.Request.Context()

	// Basic readiness checks
	ready := true
	checks := make(map[string]interface{})

	// Check if MongoDB is accessible
	if err := h.db.Ping(ctx); err != nil {
		ready = false
		checks["mongodb"] = map[string]interface{}{
			"ready": false,
			"error": err.Error(),
		}
	} else {
		checks["mongodb"] = map[string]interface{}{
			"ready": true,
		}
	}

	response := map[string]interface{}{
		"ready":     ready,
		"timestamp": time.Now().Format(time.RFC3339),
		"checks":    checks,
	}

	if ready {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// Liveness godoc
// @Summary Liveness check endpoint
// @Description Check if the service is alive
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /live [get]
func (h *HealthHandler) Liveness(c *gin.Context) {
	// Simple liveness check - if this endpoint responds, the service is alive
	c.JSON(http.StatusOK, map[string]interface{}{
		"alive":     true,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *HealthHandler) checkMongoDB(ctx context.Context) ServiceHealth {
	start := time.Now()

	// Create a timeout context for the health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := h.db.Ping(checkCtx)
	responseTime := time.Since(start).String()

	if err != nil {
		utils.LogError(ctx, "MongoDB health check failed", err)
		return ServiceHealth{
			Status:       "unhealthy",
			ResponseTime: responseTime,
			Error:        err.Error(),
		}
	}

	return ServiceHealth{
		Status:       "healthy",
		ResponseTime: responseTime,
	}
}

func (h *HealthHandler) checkS3(ctx context.Context) ServiceHealth {
	start := time.Now()

	// Create a timeout context for the health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try to check if a test object exists (this will verify S3 connectivity)
	_, err := h.storage.Exists(checkCtx, "health-check-test")
	responseTime := time.Since(start).String()

	if err != nil {
		utils.LogError(ctx, "S3 health check failed", err)
		return ServiceHealth{
			Status:       "unhealthy",
			ResponseTime: responseTime,
			Error:        err.Error(),
		}
	}

	return ServiceHealth{
		Status:       "healthy",
		ResponseTime: responseTime,
	}
}
