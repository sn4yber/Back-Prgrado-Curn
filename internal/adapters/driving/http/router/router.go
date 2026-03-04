package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sn4yber/curn-networking/internal/adapters/driving/http/handler"
	"github.com/sn4yber/curn-networking/internal/adapters/driving/http/middleware"
	"github.com/sn4yber/curn-networking/pkg/logger"
)

func Setup(
	authHandler *handler.AuthHandler,
	connectionHandler *handler.ConnectionHandler,
	userHandler *handler.UserHandler,
	jwtSecret string,
	maxReqs int,
	window time.Duration,
	log logger.Logger,
) *gin.Engine {
	engine := gin.New()

	// Middlewares globales
	engine.Use(gin.Recovery())
	engine.Use(middleware.CORS())

	// Health check
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Grupo versionado
	v1 := engine.Group("/api/v1")

	// ── Rutas públicas de autenticación ───────────────────────────────────────
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", middleware.RateLimitLogin(maxReqs, window, log), authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/reset-password", authHandler.ResetPassword)
	}

	// ── Rutas protegidas con JWT ───────────────────────────────────────────────
	protected := v1.Group("")
	protected.Use(middleware.AuthRequired(jwtSecret))
	{
		// Perfil de usuario
		userHandler.RegisterRoutes(protected)

		// Conexiones entre usuarios
		connections := protected.Group("/connections")
		{
			connections.POST("/request", connectionHandler.RequestConnection)
			connections.POST("/:id/accept", connectionHandler.AcceptConnection)
			connections.POST("/:id/reject", connectionHandler.RejectConnection)
			connections.POST("/:id/block", connectionHandler.BlockConnection)
			connections.GET("", connectionHandler.ListConnections)
		}
	}

	return engine
}
