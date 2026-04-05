package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"time"
)

type notificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *notificationRepository {
	return &notificationRepository{pool: pool}
}
func (r *notificationRepository) Create(ctx context.Context, notification *domain.Notification) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO notifications (id, user_id, type, title, message, is_read, created_at, read_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
`,
		notification.ID,
		notification.UserID,
		string(notification.Type),
		notification.Title,
		notification.Message,
		notification.IsRead,
		notification.CreatedAt,
		notification.ReadAt,
	)
	if err != nil {
		return fmt.Errorf("notificationRepository.Create: %w", err)
	}
	return nil
}
func (r *notificationRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Notification, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, user_id, type, title, message, is_read, created_at, read_at
FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("notificationRepository.ListByUser: %w", err)
	}
	defer rows.Close()
	result := make([]*domain.Notification, 0)
	for rows.Next() {
		var n domain.Notification
		var nType string
		if err := rows.Scan(
			&n.ID,
			&n.UserID,
			&nType,
			&n.Title,
			&n.Message,
			&n.IsRead,
			&n.CreatedAt,
			&n.ReadAt,
		); err != nil {
			return nil, fmt.Errorf("notificationRepository.ListByUser scan: %w", err)
		}
		n.Type = domain.NotificationType(nType)
		result = append(result, &n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("notificationRepository.ListByUser rows: %w", err)
	}
	return result, nil
}
func (r *notificationRepository) MarkAsRead(ctx context.Context, notificationID, userID uuid.UUID) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx, `
UPDATE notifications
SET is_read = TRUE, read_at = $1
WHERE id = $2 AND user_id = $3
`, now, notificationID, userID)
	if err != nil {
		return fmt.Errorf("notificationRepository.MarkAsRead: %w", err)
	}
	return nil
}
