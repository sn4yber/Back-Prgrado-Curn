package input

import (
	"context"

	"github.com/google/uuid"
)

// PointsResponse representa la respuesta de puntos y racha del usuario.
type PointsResponse struct {
	Points     int     `json:"points"`
	Streak     int     `json:"streak"`
	LastActive *string `json:"last_active"`
}

// CheckinResponse es la respuesta tras registrar el check-in diario.
type CheckinResponse struct {
	Points       int    `json:"points"`
	Streak       int    `json:"streak"`
	PointsEarned int    `json:"points_earned"`
	Message      string `json:"message"`
}

// AchievementResponse representa un logro desbloqueado.
type AchievementResponse struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	UnlockedAt  string `json:"unlocked_at"`
}

// PointsUseCase define los casos de uso del sistema de gamificación.
type PointsUseCase interface {
	GetMyPoints(ctx context.Context, userID uuid.UUID) (*PointsResponse, error)
	DailyCheckin(ctx context.Context, userID uuid.UUID) (*CheckinResponse, error)
	GetMyAchievements(ctx context.Context, userID uuid.UUID) ([]AchievementResponse, error)
}
