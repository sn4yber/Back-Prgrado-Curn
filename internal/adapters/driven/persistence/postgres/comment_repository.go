package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

type commentRepository struct {
	pool *pgxpool.Pool
}

func NewCommentRepository(pool *pgxpool.Pool) *commentRepository {
	return &commentRepository{pool: pool}
}
func (r *commentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO comments (id, post_id, author_id, content, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
`,
		comment.ID,
		comment.PostID,
		comment.AuthorID,
		comment.Content,
		comment.CreatedAt,
		comment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("commentRepository.Create: %w", err)
	}
	return nil
}
func (r *commentRepository) ListByPostID(ctx context.Context, postID uuid.UUID) ([]*domain.Comment, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, post_id, author_id, content, created_at, updated_at
FROM comments
WHERE post_id = $1
ORDER BY created_at ASC
`, postID)
	if err != nil {
		return nil, fmt.Errorf("commentRepository.ListByPostID: %w", err)
	}
	defer rows.Close()
	comments := make([]*domain.Comment, 0)
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(
			&c.ID,
			&c.PostID,
			&c.AuthorID,
			&c.Content,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("commentRepository.ListByPostID scan: %w", err)
		}
		comments = append(comments, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("commentRepository.ListByPostID rows: %w", err)
	}
	return comments, nil
}
