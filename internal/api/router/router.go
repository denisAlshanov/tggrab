package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/denisAlshanov/stPlaner/internal/api/handlers"
	"github.com/denisAlshanov/stPlaner/internal/api/middleware"
	"github.com/denisAlshanov/stPlaner/internal/config"
	"github.com/denisAlshanov/stPlaner/internal/services/auth"
)

type Router struct {
	engine *gin.Engine
	config *config.Config
}

func NewRouter(cfg *config.Config, postHandler *handlers.PostHandler, mediaHandler *handlers.MediaHandler, healthHandler *handlers.HealthHandler, showHandler *handlers.ShowHandler, eventHandler *handlers.EventHandler, guestHandler *handlers.GuestHandler, blockHandler *handlers.BlockHandler, userHandler *handlers.UserHandler, roleHandler *handlers.RoleHandler, authHandler *handlers.AuthHandlers, jwtService *auth.JWTService, sessionService *auth.SessionService) *Router {
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

	// Authentication endpoints (no auth required)
	authGroup := engine.Group("/api/v1/auth")
	authGroup.Use(middleware.RateLimitMiddleware(&cfg.API))
	{
		// Password authentication
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.RefreshToken)
		
		// Google OIDC authentication
		authGroup.GET("/google/login", authHandler.GoogleLogin)
		authGroup.POST("/google/callback", authHandler.GoogleCallback)
		
		// Protected authentication endpoints (require JWT)
		protected := authGroup.Group("")
		protected.Use(middleware.JWTAuthMiddleware(jwtService, sessionService))
		{
			protected.POST("/logout", authHandler.Logout)
			protected.GET("/verify", authHandler.VerifyToken)
			protected.GET("/sessions", authHandler.GetActiveSessions)
			protected.DELETE("/sessions/:session_id", authHandler.RevokeSession)
			protected.POST("/google/link", authHandler.GoogleLink)
		}
	}

	// API endpoints with JWT-only authentication and rate limiting
	api := engine.Group("/api/v1")
	api.Use(middleware.JWTOnlyMiddleware(jwtService, sessionService))
	api.Use(middleware.RateLimitMiddleware(&cfg.API))
	{
		// Media endpoints
		media := api.Group("/media")
		{
			media.POST("/grab", postHandler.AddPost)               // /api/v1/media/grab
			media.GET("/list", postHandler.GetList)                // /api/v1/media/list
			media.POST("/links", mediaHandler.GetLinkList)         // /api/v1/media/links
			media.POST("/get", mediaHandler.GetLinkMedia)          // /api/v1/media/get (download)
			media.PUT("/get", mediaHandler.UpdateLinkMedia)        // /api/v1/media/get (update)
			media.DELETE("/get", mediaHandler.DeleteLinkMedia)     // /api/v1/media/get (delete)
			media.POST("/getDirect", mediaHandler.GetLinkMediaURI) // /api/v1/media/getDirect
		}

		// Show endpoints
		show := api.Group("/show")
		{
			show.POST("/create", showHandler.CreateShow)           // /api/v1/show/create
			show.DELETE("/delete", showHandler.DeleteShow)         // /api/v1/show/delete
			show.POST("/list", showHandler.ListShows)              // /api/v1/show/list
			show.GET("/info/:show_id", showHandler.GetShowInfo)    // /api/v1/show/info/{show_id}
		}

		// Event endpoints
		event := api.Group("/event")
		{
			event.PUT("/update", eventHandler.UpdateEvent)         // /api/v1/event/update
			event.DELETE("/delete", eventHandler.DeleteEvent)      // /api/v1/event/delete
			event.POST("/list", eventHandler.ListEvents)           // /api/v1/event/list
			event.POST("/weekList", eventHandler.WeekListEvents)   // /api/v1/event/weekList
			event.POST("/monthList", eventHandler.MonthListEvents) // /api/v1/event/monthList
			event.GET("/info/:event_id", eventHandler.GetEventInfo) // /api/v1/event/info/{event_id}
		}

		// Guest endpoints
		guest := api.Group("/guest")
		{
			guest.POST("/new", guestHandler.CreateGuest)           // /api/v1/guest/new
			guest.PUT("/update", guestHandler.UpdateGuest)         // /api/v1/guest/update
			guest.POST("/list", guestHandler.ListGuests)           // /api/v1/guest/list
			guest.GET("/autocomplete", guestHandler.AutocompleteGuests) // /api/v1/guest/autocomplete
			guest.GET("/info/:guest_id", guestHandler.GetGuestInfo) // /api/v1/guest/info/{guest_id}
			guest.DELETE("/delete", guestHandler.DeleteGuest)      // /api/v1/guest/delete
		}

		// Block endpoints
		block := api.Group("/block")
		{
			block.POST("/add", blockHandler.AddBlock)              // /api/v1/block/add
			block.PUT("/update", blockHandler.UpdateBlock)         // /api/v1/block/update
			block.GET("/info/:block_id", blockHandler.GetBlockInfo) // /api/v1/block/info/{block_id}
			block.PUT("/reorder", blockHandler.ReorderBlocks)      // /api/v1/block/reorder
			block.DELETE("/delete", blockHandler.DeleteBlock)      // /api/v1/block/delete
		}

		// Event-specific block endpoints
		api.GET("/event/:event_id/blocks", blockHandler.GetEventBlocks) // /api/v1/event/{event_id}/blocks

		// User endpoints
		user := api.Group("/users")
		{
			user.POST("/add", userHandler.CreateUser)              // /api/v1/users/add
			user.DELETE("/delete", userHandler.DeleteUser)         // /api/v1/users/delete
			user.PUT("/update", userHandler.UpdateUser)            // /api/v1/users/update
			user.GET("/info/:user_id", userHandler.GetUserInfo)    // /api/v1/users/info/{user_id}
			user.POST("/list", userHandler.ListUsers)              // /api/v1/users/list
		}

		// Role endpoints
		role := api.Group("/roles")
		{
			role.POST("/add", roleHandler.CreateRole)              // /api/v1/roles/add
			role.DELETE("/delete", roleHandler.DeleteRole)         // /api/v1/roles/delete
			role.PUT("/update", roleHandler.UpdateRole)            // /api/v1/roles/update
			role.GET("/info/:role_id", roleHandler.GetRoleInfo)    // /api/v1/roles/info/{role_id}
			role.POST("/list", roleHandler.ListRoles)              // /api/v1/roles/list
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
