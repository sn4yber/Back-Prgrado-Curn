package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

type PostRepository interface {
	Create(ctx context.Context, post *domain.Post, attachments []*domain.PostAttachment) error
	FindByID(ctx context.Context, postID uuid.UUID) (*domain.Post, []*domain.PostAttachment, error)
	ExistsPublishedByID(ctx context.Context, postID uuid.UUID) (bool, error)
	UpdateBasic(ctx context.Context, postID uuid.UUID, title, description string, category domain.PostCategory) error
	DeleteByID(ctx context.Context, postID uuid.UUID) error
	ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]*domain.Post, map[uuid.UUID][]*domain.PostAttachment, error)
	ListPublic(ctx context.Context) ([]*domain.Post, map[uuid.UUID][]*domain.PostAttachment, error)
	ListByStatuses(ctx context.Context, statuses []domain.PostStatus) ([]*domain.Post, map[uuid.UUID][]*domain.PostAttachment, error)
	UpdateModeration(ctx context.Context, postID uuid.UUID, status domain.PostStatus, notes *string) error
	UpsertReaction(ctx context.Context, postID, userID uuid.UUID, reactionType domain.PostReactionType) error
	GetReactionsSummaryByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID]domain.PostReactionsSummary, error)
	GetCurrentUserReactionsByPostIDs(ctx context.Context, postIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]domain.PostReactionType, error)
	CreateReport(ctx context.Context, postID, reporterID uuid.UUID, reason string) (bool, int, error)
}

type FileStorage interface {
	Save(ctx context.Context, objectKey string, contentType string, data []byte) (string, error)
}
