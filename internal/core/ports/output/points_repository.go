package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

// PointsRepository define las operaciones de persistencia para el sistema de puntos.
type PointsRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserPoints, error)
	Upsert(ctx context.Context, points *domain.UserPoints) error
	ListAchievements(ctx context.Context, userID uuid.UUID) ([]*domain.Achievement, error)
	UnlockAchievement(ctx context.Context, userID uuid.UUID, key string) error
	HasAchievement(ctx context.Context, userID uuid.UUID, key string) (bool, error)
}
