package router

import (
	"github.com/gin-gonic/gin"
	"github.com/sn4yber/curn-networking/internal/adapters/driving/http/handler"
	"github.com/sn4yber/curn-networking/internal/adapters/driving/http/middleware"
	"github.com/sn4yber/curn-networking/pkg/logger"
	"time"
)

func Setup(
	authHandler *handler.AuthHandler,
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

	//grupo versionado

	v1 := engine.Group("/api/v1")

	// Rutas de autenticación
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", middleware.RateLimitLogin(maxReqs, window, log), authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/reset-password", authHandler.ResetPassword)
	}

	return engine
}
