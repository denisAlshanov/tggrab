package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/denisAlshanov/stPlaner/internal/database"
	"github.com/denisAlshanov/stPlaner/internal/services/storage"
	"github.com/denisAlshanov/stPlaner/internal/utils"
)

type HealthHandler struct {
	db      *database.PostgresDB
	storage storage.StorageInterface
}

type HealthResponse struct {
	Status     string                   `json:"status"`
	Timestamp  string                   `json:"timestamp"`
	Version    string                   `json:"version"`
	Services   map[string]ServiceHealth `json:"services"`
	Migrations MigrationStatus          `json:"migrations,omitempty"`
}

type MigrationStatus struct {
	Applied []MigrationInfo `json:"applied"`
	Count   int             `json:"count"`
}

type MigrationInfo struct {
	Version     int    `json:"version"`
	Description string `json:"description"`
}

type ServiceHealth struct {
	Status       string `json:"status"`
	ResponseTime string `json:"response_time,omitempty"`
	Error        string `json:"error,omitempty"`
}

func NewHealthHandler(db *database.PostgresDB, storage storage.StorageInterface) *HealthHandler {
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

	// Check PostgreSQL
	pgHealth := h.checkPostgreSQL(ctx)
	response.Services["postgresql"] = pgHealth

	// Check S3
	s3Health := h.checkS3(ctx)
	response.Services["s3"] = s3Health

	// Get migration status
	migrationStatus := h.getMigrationStatus(ctx)
	response.Migrations = migrationStatus

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

	// Check if PostgreSQL is accessible
	if err := h.db.Ping(ctx); err != nil {
		ready = false
		checks["postgresql"] = map[string]interface{}{
			"ready": false,
			"error": err.Error(),
		}
	} else {
		checks["postgresql"] = map[string]interface{}{
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

func (h *HealthHandler) checkPostgreSQL(ctx context.Context) ServiceHealth {
	start := time.Now()

	// Create a timeout context for the health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := h.db.Ping(checkCtx)
	responseTime := time.Since(start).String()

	if err != nil {
		utils.LogError(ctx, "PostgreSQL health check failed", err)
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

func (h *HealthHandler) getMigrationStatus(ctx context.Context) MigrationStatus {
	// Create a timeout context for migration status check
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	migrations, err := h.db.GetMigrationStatus(checkCtx)
	if err != nil {
		utils.LogError(ctx, "Failed to get migration status", err)
		return MigrationStatus{
			Applied: []MigrationInfo{},
			Count:   0,
		}
	}

	migrationInfos := make([]MigrationInfo, len(migrations))
	for i, migration := range migrations {
		migrationInfos[i] = MigrationInfo{
			Version:     migration.Version,
			Description: migration.Description,
		}
	}

	return MigrationStatus{
		Applied: migrationInfos,
		Count:   len(migrationInfos),
	}
}
