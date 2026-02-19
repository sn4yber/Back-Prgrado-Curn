package domain

import (
	"github.com/google/uuid"
	"time"
)

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}

func (p *PasswordResetToken) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)

}
