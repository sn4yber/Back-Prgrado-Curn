package input

import (
	"context"

	"github.com/google/uuid"
)

type CreateCommentRequest struct {
	Content string `json:"content"`
}
type CommentResponse struct {
	ID         string  `json:"id"`
	PostID     string  `json:"post_id"`
	AuthorID   string  `json:"author_id"`
	AuthorName *string `json:"author_name,omitempty"`
	Content    string  `json:"content"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}
type CommentUseCase interface {
	Create(ctx context.Context, authorID, postID uuid.UUID, req CreateCommentRequest) (*CommentResponse, error)
	ListByPost(ctx context.Context, postID uuid.UUID) ([]CommentResponse, error)
}
