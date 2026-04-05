package output

import (
	"context"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

type CommentRepository interface {
	Create(ctx context.Context, comment *domain.Comment) error
	ListByPostID(ctx context.Context, postID uuid.UUID) ([]*domain.Comment, error)
}
