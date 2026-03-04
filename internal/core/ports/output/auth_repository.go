package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

// RefreshTokenRepository define las operaciones de persistencia para refresh tokens.
type RefreshTokenRepository interface {
	Save(ctx context.Context, token *domain.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
}

// PasswordResetTokenRepository define las operaciones de persistencia para tokens de recuperación.
type PasswordResetTokenRepository interface {
	Save(ctx context.Context, token *domain.PasswordResetToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error)
	MarkAsUsed(ctx context.Context, id uuid.UUID) error
	DeleteExpiredByUserID(ctx context.Context, userID uuid.UUID) error
}
