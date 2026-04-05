package input

import (
	"context"
	"github.com/google/uuid"
)

type NotificationResponse struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Title     string  `json:"title"`
	Message   string  `json:"message"`
	IsRead    bool    `json:"is_read"`
	CreatedAt string  `json:"created_at"`
	ReadAt    *string `json:"read_at,omitempty"`
}
type NotificationUseCase interface {
	ListMine(ctx context.Context, userID uuid.UUID, limit, offset int) ([]NotificationResponse, error)
	MarkAsRead(ctx context.Context, userID, notificationID uuid.UUID) error
}
