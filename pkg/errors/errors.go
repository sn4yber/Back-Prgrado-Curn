package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError es el error tipado de dominio.
// Contiene el código HTTP sugerido y un mensaje seguro para el cliente.
type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// ─── Constructor genérico ─────────────────────────────────────────────────────

func New(code int, message string, cause error) *AppError {
	return &AppError{Code: code, Message: message, Err: cause}
}

// ─── Errores de autenticación y usuario ──────────────────────────────────────

var (
	ErrInvalidCredentials = &AppError{Code: http.StatusUnauthorized, Message: "credenciales inválidas"}
	ErrUserNotFound       = &AppError{Code: http.StatusNotFound, Message: "usuario no encontrado"}
	ErrUserBanned         = &AppError{Code: http.StatusForbidden, Message: "cuenta suspendida"}
	ErrUserInactive       = &AppError{Code: http.StatusForbidden, Message: "cuenta inactiva"}
	ErrEmailAlreadyExists = &AppError{Code: http.StatusConflict, Message: "el correo ya está registrado"}
)

// ─── Errores de tokens ────────────────────────────────────────────────────────

var (
	ErrTokenExpired         = &AppError{Code: http.StatusUnauthorized, Message: "token expirado"}
	ErrTokenInvalid         = &AppError{Code: http.StatusUnauthorized, Message: "token inválido"}
	ErrRefreshTokenNotFound = &AppError{Code: http.StatusUnauthorized, Message: "refresh token no encontrado"}
	ErrResetTokenInvalid    = &AppError{Code: http.StatusBadRequest, Message: "token de recuperación inválido o expirado"}
)

// ─── Errores de validación y servidor ────────────────────────────────────────

var (
	ErrValidation  = &AppError{Code: http.StatusBadRequest, Message: "datos de entrada inválidos"}
	ErrInternal    = &AppError{Code: http.StatusInternalServerError, Message: "error interno del servidor"}
	ErrRateLimited = &AppError{Code: http.StatusTooManyRequests, Message: "demasiados intentos, espera un momento"}
)

// ─── Helper ───────────────────────────────────────────────────────────────────

// Is permite comparar errores de dominio con errors.Is()
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// AsAppError extrae un *AppError de cualquier error si es posible.
// Devuelve ErrInternal como fallback si no es un error tipado.
func AsAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return ErrInternal
}
