package router

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sn4yber/curn-networking/internal/adapters/driving/http/handler"
	"github.com/sn4yber/curn-networking/internal/adapters/driving/http/middleware"
	"github.com/sn4yber/curn-networking/pkg/logger"
)

func Setup(
	authHandler *handler.AuthHandler,
	connectionHandler *handler.ConnectionHandler,
	userHandler *handler.UserHandler,
	conversationHandler *handler.ConversationHandler,
	postHandler *handler.PostHandler,
	jwtSecret string,
	maxReqs int,
	window time.Duration,
	log logger.Logger,
	pool *pgxpool.Pool,
) *gin.Engine {
	engine := gin.New()

	// Middlewares globales
	engine.Use(gin.Recovery())
	engine.Use(middleware.CORS())

	// Health check mejorado con verificación de dependencias
	engine.GET("/health", func(c *gin.Context) {
		// Verificar conectividad a la base de datos
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			log.Error("health check falló: base de datos no responde")
			c.JSON(503, gin.H{
				"status":   "unhealthy",
				"database": "down",
				"error":    "database unavailable",
			})
			return
		}

		c.JSON(200, gin.H{
			"status":   "healthy",
			"database": "up",
		})
	})

	// Archivos de adjuntos publicados localmente.
	engine.Static("/uploads", "./uploads")

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

		// Conversaciones 1:1 por contexto (inbox)
		conversationHandler.RegisterRoutes(protected)

		// Publicaciones con moderación institucional
		postHandler.RegisterRoutes(protected)
	}

	return engine
}
