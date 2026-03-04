package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/adapters/driving/http/middleware"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
)

// UserHandler maneja las rutas HTTP del perfil de usuario.
type UserHandler struct {
	usecase input.UserUseCase
}

// NewUserHandler construye el handler con su dependencia inyectada.
func NewUserHandler(usecase input.UserUseCase) *UserHandler {
	return &UserHandler{usecase: usecase}
}

// ─── RegisterRoutes ───────────────────────────────────────────────────────────

// RegisterRoutes registra las rutas de usuario en el grupo protegido.
func (h *UserHandler) RegisterRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	{
		users.GET("/me", h.GetProfile)
		users.PUT("/me", h.UpdateProfile)
	}
}

// ─── GetProfile ───────────────────────────────────────────────────────────────

// GetProfile retorna el perfil completo del usuario autenticado.
//
//	GET /api/v1/users/me
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	profile, err := h.usecase.GetProfile(c.Request.Context(), userID)
	if err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// ─── UpdateProfile ────────────────────────────────────────────────────────────

// UpdateProfile actualiza los datos editables del usuario autenticado.
//
//	PUT /api/v1/users/me
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	var req input.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}

	profile, err := h.usecase.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// ─── Helper ───────────────────────────────────────────────────────────────────

// extractUserID obtiene el UUID del usuario desde el contexto JWT.
// El middleware AuthRequired inyecta el userID como string.
func extractUserID(c *gin.Context) (uuid.UUID, error) {
	raw, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		return uuid.Nil, apperrors.ErrUserNotFound
	}
	id, err := uuid.Parse(raw.(string))
	if err != nil {
		return uuid.Nil, apperrors.ErrUserNotFound
	}
	return id, nil
}

// silenceUnusedImport evita error de compilación mientras errors se usa internamente.
var _ = errors.New
