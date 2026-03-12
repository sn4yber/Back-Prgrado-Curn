package handler

import (
	"net/http"
	"net/mail"
	"strings"

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

	if msg := validateRegisterRequest(req); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
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

	if msg := validateLoginRequest(req); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
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

	if msg := validateForgotPasswordRequest(req); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
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

	if msg := validateResetPasswordRequest(req); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	if err := h.authUseCase.ResetPassword(c.Request.Context(), req); err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "contraseña actualizada correctamente"})
}

func validateRegisterRequest(req input.RegisterRequest) string {
	if len(strings.TrimSpace(req.Name)) < 2 || len(req.Name) > 150 {
		return "name es requerido y debe tener entre 2 y 150 caracteres"
	}
	if !isValidEmail(req.Email) {
		return "email inválido"
	}
	if l := len(req.Password); l < 8 || l > 72 {
		return "password debe tener entre 8 y 72 caracteres"
	}
	if len(strings.TrimSpace(req.ProgramID)) == 0 {
		return "program_id es requerido"
	}
	return ""
}

func validateLoginRequest(req input.LoginRequest) string {
	if !isValidEmail(req.Email) {
		return "email inválido"
	}
	if l := len(req.Password); l < 1 || l > 72 {
		return "password inválido"
	}
	return ""
}

func validateForgotPasswordRequest(req input.ForgotPasswordRequest) string {
	if !isValidEmail(req.Email) {
		return "email inválido"
	}
	return ""
}

func validateResetPasswordRequest(req input.ResetPasswordRequest) string {
	if l := len(strings.TrimSpace(req.Token)); l < 10 || l > 200 {
		return "token inválido"
	}
	if l := len(req.NewPassword); l < 8 || l > 72 {
		return "new_password debe tener entre 8 y 72 caracteres"
	}
	return ""
}

func isValidEmail(value string) bool {
	email := strings.TrimSpace(value)
	if email == "" || len(email) > 255 {
		return false
	}
	_, err := mail.ParseAddress(email)
	return err == nil
}
