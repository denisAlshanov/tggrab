package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/denisAlshanov/stPlaner/internal/api/handlers"
	"github.com/denisAlshanov/stPlaner/internal/api/middleware"
	"github.com/denisAlshanov/stPlaner/internal/config"
)

type Router struct {
	engine *gin.Engine
	config *config.Config
}

func NewRouter(cfg *config.Config, postHandler *handlers.PostHandler, mediaHandler *handlers.MediaHandler, healthHandler *handlers.HealthHandler) *Router {
	// Set Gin mode
	if cfg.Server.Host == "0.0.0.0" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	// Add middleware
	engine.Use(gin.Recovery())
	engine.Use(middleware.CorrelationIDMiddleware())

	// Health endpoints (no auth required)
	health := engine.Group("/")
	{
		health.GET("/health", healthHandler.Health)
		health.GET("/ready", healthHandler.Readiness)
		health.GET("/live", healthHandler.Liveness)
	}

	// Swagger documentation (no auth required)
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API endpoints with authentication and rate limiting
	api := engine.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(&cfg.API))
	api.Use(middleware.RateLimitMiddleware(&cfg.API))
	{
		// Media endpoints
		media := api.Group("/media")
		{
			media.POST("/grab", postHandler.AddPost)           // /api/v1/media/grab
			media.GET("/list", postHandler.GetList)            // /api/v1/media/list
			media.POST("/links", mediaHandler.GetLinkList)     // /api/v1/media/links
			media.POST("/get", mediaHandler.GetLinkMedia)      // /api/v1/media/get
			media.POST("/getDirect", mediaHandler.GetLinkMediaURI) // /api/v1/media/getDirect
		}
	}

	return &Router{
		engine: engine,
		config: cfg,
	}
}

func (r *Router) Start() error {
	addr := r.config.Server.Host + ":" + r.config.Server.Port
	return r.engine.Run(addr)
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}
