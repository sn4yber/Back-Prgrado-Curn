package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sn4yber/curn-networking/internal/adapters/driving/http/middleware"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"

	"github.com/google/uuid"
)

// PointsHandler expone los endpoints HTTP del sistema de gamificación.
type PointsHandler struct {
	usecase input.PointsUseCase
}

func NewPointsHandler(usecase input.PointsUseCase) *PointsHandler {
	return &PointsHandler{usecase: usecase}
}

// RegisterRoutes registra las rutas de puntos en el grupo protegido.
func (h *PointsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	{
		users.GET("/me/points", h.GetMyPoints)
		users.POST("/me/daily-checkin", h.DailyCheckin)
		users.GET("/me/achievements", h.GetMyAchievements)
	}
}

// GetMyPoints devuelve los puntos y racha del usuario autenticado.
//
//	GET /api/v1/users/me/points
func (h *PointsHandler) GetMyPoints(c *gin.Context) {
	userID, err := extractPointsUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	resp, err := h.usecase.GetMyPoints(c.Request.Context(), userID)
	if err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DailyCheckin registra el check-in diario del usuario.
//
//	POST /api/v1/users/me/daily-checkin
func (h *PointsHandler) DailyCheckin(c *gin.Context) {
	userID, err := extractPointsUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	resp, err := h.usecase.DailyCheckin(c.Request.Context(), userID)
	if err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetMyAchievements devuelve la lista de logros desbloqueados.
//
//	GET /api/v1/users/me/achievements
func (h *PointsHandler) GetMyAchievements(c *gin.Context) {
	userID, err := extractPointsUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	resp, err := h.usecase.GetMyAchievements(c.Request.Context(), userID)
	if err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func extractPointsUserID(c *gin.Context) (uuid.UUID, error) {
	raw, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		return uuid.Nil, apperrors.ErrUserNotFound
	}
	return uuid.Parse(raw.(string))
}
