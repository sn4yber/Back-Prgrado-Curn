package notification

import (
	"context"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
	"time"
)

type Service struct {
	repo output.NotificationRepository
}

func New(repo output.NotificationRepository) *Service {
	return &Service{repo: repo}
}
func (s *Service) ListMine(ctx context.Context, userID uuid.UUID, limit, offset int) ([]input.NotificationResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	items, err := s.repo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, apperrors.New(500, "no se pudieron listar notificaciones", err)
	}
	resp := make([]input.NotificationResponse, 0, len(items))
	for _, n := range items {
		resp = append(resp, *toNotificationResponse(n))
	}
	return resp, nil
}
func (s *Service) MarkAsRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	if err := s.repo.MarkAsRead(ctx, notificationID, userID); err != nil {
		return apperrors.New(500, "no se pudo marcar la notificación como leída", err)
	}
	return nil
}
func NewMentorshipNotification(toUser uuid.UUID, nType domain.NotificationType, title, message string) *domain.Notification {
	return domain.NewNotification(toUser, nType, title, message)
}
func toNotificationResponse(n *domain.Notification) *input.NotificationResponse {
	var readAt *string
	if n.ReadAt != nil {
		value := n.ReadAt.Format(time.RFC3339)
		readAt = &value
	}
	return &input.NotificationResponse{
		ID:        n.ID.String(),
		Type:      string(n.Type),
		Title:     n.Title,
		Message:   n.Message,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt.Format(time.RFC3339),
		ReadAt:    readAt,
	}
}
