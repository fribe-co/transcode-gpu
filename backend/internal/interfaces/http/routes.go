package http

import (
	"os"
	"time"

	"github.com/cashbacktv/backend/internal/domain"
	"github.com/cashbacktv/backend/internal/interfaces/http/handlers"
	"github.com/cashbacktv/backend/internal/interfaces/http/middleware"
	"github.com/cashbacktv/backend/internal/pkg/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Router holds all handlers and middleware
type Router struct {
	app            *fiber.App
	authHandler    *handlers.AuthHandler
	channelHandler *handlers.ChannelHandler
	uploadHandler  *handlers.UploadHandler
	settingsHandler *handlers.SettingsHandler
	systemHandler  *handlers.SystemHandler
	authMiddleware *middleware.AuthMiddleware
	logoPath       string
	hlsPath        string
}

// NewRouter creates a new router
func NewRouter(
	authHandler *handlers.AuthHandler,
	channelHandler *handlers.ChannelHandler,
	uploadHandler *handlers.UploadHandler,
	settingsHandler *handlers.SettingsHandler,
	authMiddleware *middleware.AuthMiddleware,
	logoPath string,
	hlsPath string,
	serverConfig *config.ServerConfig,
) *Router {
	// Check if running in production (prefork mode for performance)
	isProd := os.Getenv("ENV") == "production" || os.Getenv("ENVIRONMENT") == "production"
	
	app := fiber.New(fiber.Config{
		ErrorHandler:    customErrorHandler,
		BodyLimit:       10 * 1024 * 1024, // 10MB for file uploads
		ReadTimeout:     time.Duration(serverConfig.ReadTimeout) * time.Second,
		WriteTimeout:    time.Duration(serverConfig.WriteTimeout) * time.Second,
		IdleTimeout:     time.Duration(serverConfig.IdleTimeout) * time.Second,
		ReadBufferSize:  4096,  // 4KB read buffer for better performance
		WriteBufferSize: 4096,  // 4KB write buffer for better performance
		Concurrency:     256 * 1024, // Maximum number of concurrent connections
		Prefork:         false, // Disable prefork for now (can enable if needed)
		ServerHeader:    "CashbackTV",
		AppName:         "CashbackTV API",
	})

	// Global middleware - order matters!
	app.Use(recover.New())
	
	// Response compression (gzip) - should be early in the chain
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed, // Fastest compression for better response time
	}))
	
	// Logger middleware - disable or make less verbose in production
	if !isProd {
		app.Use(logger.New(logger.Config{
			Format: "[${time}] ${status} - ${method} ${path} - ${latency}\n",
		}))
	} else {
		// Production: minimal logging for performance
		app.Use(logger.New(logger.Config{
			Format:     "${status} ${method} ${path} ${latency}\n",
			TimeFormat: "15:04:05",
			Output:     os.Stdout,
		}))
	}
	
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: false,
		MaxAge:           86400, // Cache preflight requests for 24 hours
	}))

	return &Router{
		app:            app,
		authHandler:    authHandler,
		channelHandler: channelHandler,
		uploadHandler:  uploadHandler,
		settingsHandler: settingsHandler,
		systemHandler:  handlers.NewSystemHandler(),
		authMiddleware: authMiddleware,
		logoPath:       logoPath,
		hlsPath:        hlsPath,
	}
}

// SetupRoutes configures all routes
func (r *Router) SetupRoutes() {
	// Static file serving for logos
	r.app.Static("/logos", r.logoPath)
	
	// Custom stream handler for /streams/:channelId/index.m3u8
	// This must come BEFORE static serving to intercept m3u8 requests
	r.app.Get("/streams/:channelId/index.m3u8", r.channelHandler.ServeStream)
	
	// Static file serving for HLS streams (segments, etc.)
	// Note: This will handle all other /streams/* requests except /streams/:channelId/index.m3u8
	r.app.Static("/streams", r.hlsPath)

	api := r.app.Group("/api/v1")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
		})
	})

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/login", r.authHandler.Login)
	auth.Post("/logout", r.authHandler.Logout)
	auth.Post("/refresh", r.authHandler.Refresh)

	// Protected routes
	protected := api.Group("")
	protected.Use(r.authMiddleware.Authenticate())

	// Auth (protected)
	protected.Get("/auth/me", r.authHandler.Me)

	// Channels
	channels := protected.Group("/channels")
	channels.Get("/", r.channelHandler.List)
	
	// Batch operations must be defined BEFORE /:id routes to avoid route conflicts
	// Operator+ only
	channels.Post("/batch/start", r.authMiddleware.RequireRole(domain.UserRoleOperator), r.channelHandler.BatchStart)
	channels.Post("/batch/stop", r.authMiddleware.RequireRole(domain.UserRoleOperator), r.channelHandler.BatchStop)
	channels.Post("/batch/restart", r.authMiddleware.RequireRole(domain.UserRoleOperator), r.channelHandler.BatchRestart)
	
	// Admin only
	channels.Post("/batch/delete", r.authMiddleware.RequireRole(domain.UserRoleAdmin), r.channelHandler.BatchDelete)
	
	// Batch metrics endpoint (must come before /:id routes to avoid route conflicts)
	channels.Get("/metrics", r.channelHandler.AllMetrics)
	
	// Individual channel routes (must come after batch routes)
	channels.Get("/:id", r.channelHandler.Get)
	channels.Get("/:id/metrics", r.channelHandler.Metrics)
	channels.Get("/:id/logs", r.channelHandler.Logs)

	// Operator+ only
	channels.Post("/", r.authMiddleware.RequireRole(domain.UserRoleOperator), r.channelHandler.Create)
	channels.Put("/:id", r.authMiddleware.RequireRole(domain.UserRoleOperator), r.channelHandler.Update)
	channels.Post("/:id/start", r.authMiddleware.RequireRole(domain.UserRoleOperator), r.channelHandler.Start)
	channels.Post("/:id/stop", r.authMiddleware.RequireRole(domain.UserRoleOperator), r.channelHandler.Stop)
	channels.Post("/:id/restart", r.authMiddleware.RequireRole(domain.UserRoleOperator), r.channelHandler.Restart)

	// Admin only
	channels.Delete("/:id", r.authMiddleware.RequireRole(domain.UserRoleAdmin), r.channelHandler.Delete)

	// Upload routes (Operator+ only)
	uploads := protected.Group("/uploads")
	uploads.Post("/logo", r.authMiddleware.RequireRole(domain.UserRoleOperator), r.uploadHandler.UploadLogo)
	uploads.Delete("/logo/:filename", r.authMiddleware.RequireRole(domain.UserRoleOperator), r.uploadHandler.DeleteLogo)

	// Settings routes (Admin only)
	settings := protected.Group("/settings")
	settings.Get("/", r.authMiddleware.RequireRole(domain.UserRoleAdmin), r.settingsHandler.Get)
	settings.Put("/", r.authMiddleware.RequireRole(domain.UserRoleAdmin), r.settingsHandler.Update)

	// System info routes (all authenticated users)
	protected.Get("/system/info", r.systemHandler.GetSystemInfo)
}

// Start starts the HTTP server
func (r *Router) Start(addr string) error {
	return r.app.Listen(addr)
}

// Shutdown gracefully shuts down the server
func (r *Router) Shutdown() error {
	return r.app.Shutdown()
}

// customErrorHandler handles errors globally
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
	})
}

