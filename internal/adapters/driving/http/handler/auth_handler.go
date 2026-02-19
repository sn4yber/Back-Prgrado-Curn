package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
)

type AuthHandler struct {
	authUseCase input.AuthUseCase
}

func NewAuthHandler(authUseCase input.AuthUseCase) *AuthHandler {
	return &AuthHandler{authUseCase: authUseCase}
}

// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req input.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}

	resp, err := h.authUseCase.Register(c.Request.Context(), req)
	if err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req input.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}

	resp, err := h.authUseCase.Login(c.Request.Context(), req)
	if err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// POST /api/v1/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token requerido"})
		return
	}

	resp, err := h.authUseCase.RefreshToken(c.Request.Context(), body.RefreshToken)
	if err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// POST /api/v1/auth/forgot-password
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req input.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}

	// Siempre responde 200 aunque el email no exista — evita enumeración
	_ = h.authUseCase.ForgotPassword(c.Request.Context(), req)
	c.JSON(http.StatusOK, gin.H{"message": "si el correo está registrado, recibirás instrucciones"})
}

// POST /api/v1/auth/reset-password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req input.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}

	if err := h.authUseCase.ResetPassword(c.Request.Context(), req); err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "contraseña actualizada correctamente"})
}
