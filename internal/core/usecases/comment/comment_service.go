package comment

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
)

type Service struct {
	commentRepo output.CommentRepository
	userRepo    output.UserRepository
	postRepo    output.PostRepository
}

func New(commentRepo output.CommentRepository, userRepo output.UserRepository, postRepo output.PostRepository) *Service {
	return &Service{commentRepo: commentRepo, userRepo: userRepo, postRepo: postRepo}
}
func (s *Service) Create(ctx context.Context, authorID, postID uuid.UUID, req input.CreateCommentRequest) (*input.CommentResponse, error) {
	if _, err := s.userRepo.FindByID(ctx, authorID); err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	post, _, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo validar la publicación", err)
	}
	if post == nil {
		return nil, apperrors.New(404, "publicación no encontrada", nil)
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, apperrors.New(400, "content es requerido", nil)
	}
	if len(content) > 1000 {
		return nil, apperrors.New(400, "content excede 1000 caracteres", nil)
	}
	now := time.Now()
	c := &domain.Comment{
		ID:        uuid.New(),
		PostID:    postID,
		AuthorID:  authorID,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
	c.Normalize()
	if err := s.commentRepo.Create(ctx, c); err != nil {
		return nil, apperrors.New(500, "no se pudo crear el comentario", err)
	}
	return toCommentResponse(c), nil
}
func (s *Service) ListByPost(ctx context.Context, postID uuid.UUID) ([]input.CommentResponse, error) {
	post, _, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo validar la publicación", err)
	}
	if post == nil {
		return nil, apperrors.New(404, "publicación no encontrada", nil)
	}
	comments, err := s.commentRepo.ListByPostID(ctx, postID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudieron listar los comentarios", err)
	}
	resp := make([]input.CommentResponse, 0, len(comments))
	for _, c := range comments {
		resp = append(resp, *toCommentResponse(c))
	}
	return resp, nil
}
func toCommentResponse(c *domain.Comment) *input.CommentResponse {
	return &input.CommentResponse{
		ID:         c.ID.String(),
		PostID:     c.PostID.String(),
		AuthorID:   c.AuthorID.String(),
		AuthorName: c.AuthorName,
		Content:    c.Content,
		CreatedAt:  c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  c.UpdatedAt.Format(time.RFC3339),
	}
}
