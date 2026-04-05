package output

import (
	"context"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *domain.Notification) error
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Notification, error)
	MarkAsRead(ctx context.Context, notificationID, userID uuid.UUID) error
}
