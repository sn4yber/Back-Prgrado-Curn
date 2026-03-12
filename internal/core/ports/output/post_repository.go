package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

type PostRepository interface {
	Create(ctx context.Context, post *domain.Post, attachments []*domain.PostAttachment) error
	FindByID(ctx context.Context, postID uuid.UUID) (*domain.Post, []*domain.PostAttachment, error)
	ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]*domain.Post, map[uuid.UUID][]*domain.PostAttachment, error)
	ListPublic(ctx context.Context) ([]*domain.Post, map[uuid.UUID][]*domain.PostAttachment, error)
	ListByStatuses(ctx context.Context, statuses []domain.PostStatus) ([]*domain.Post, map[uuid.UUID][]*domain.PostAttachment, error)
	UpdateModeration(ctx context.Context, postID uuid.UUID, status domain.PostStatus, notes *string) error
}

type FileStorage interface {
	Save(ctx context.Context, objectKey string, contentType string, data []byte) (string, error)
}
