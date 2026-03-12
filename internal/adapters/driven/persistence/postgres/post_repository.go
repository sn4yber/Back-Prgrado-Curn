package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

type postRepository struct {
	pool *pgxpool.Pool
}

func NewPostRepository(pool *pgxpool.Pool) *postRepository {
	return &postRepository{pool: pool}
}

func (r *postRepository) Create(ctx context.Context, post *domain.Post, attachments []*domain.PostAttachment) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postRepository.Create begin: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO posts (
			id, author_id, declared_author_id, coauthor_ids, title, description, category,
			originality_declaration, privacy_consent, is_institutional, verified_by_faculty,
			status, moderation_notes, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15
		)
	`,
		post.ID, post.AuthorID, post.DeclaredAuthorID, post.CoAuthorIDs, post.Title, post.Description,
		string(post.Category), post.OriginalityDeclaration, post.PrivacyConsent,
		post.IsInstitutional, post.VerifiedByFaculty, string(post.Status), post.ModerationNotes,
		post.CreatedAt, post.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("postRepository.Create insert post: %w", err)
	}

	for _, a := range attachments {
		_, err = tx.Exec(ctx, `
			INSERT INTO post_attachments (
				id, post_id, file_name, file_url, file_ext, mime_type, size_bytes, uploaded_by, created_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		`,
			a.ID, a.PostID, a.FileName, a.FileURL, a.FileExt, a.MimeType, a.SizeBytes, a.UploadedBy, a.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("postRepository.Create insert attachment: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postRepository.Create commit: %w", err)
	}
	return nil
}

func (r *postRepository) FindByID(ctx context.Context, postID uuid.UUID) (*domain.Post, []*domain.PostAttachment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, author_id, declared_author_id, coauthor_ids, title, description, category,
		       originality_declaration, privacy_consent, is_institutional, verified_by_faculty,
		       status, moderation_notes, created_at, updated_at
		FROM posts
		WHERE id = $1
		LIMIT 1
	`, postID)
	if err != nil {
		return nil, nil, fmt.Errorf("postRepository.FindByID post: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil, nil
	}
	post, err := scanPost(rows)
	if err != nil {
		return nil, nil, err
	}

	attachments, err := r.listAttachmentsByPostIDs(ctx, []uuid.UUID{post.ID})
	if err != nil {
		return nil, nil, err
	}

	return post, attachments[post.ID], nil
}

func (r *postRepository) ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]*domain.Post, map[uuid.UUID][]*domain.PostAttachment, error) {
	posts, err := r.queryPosts(ctx, `
		SELECT id, author_id, declared_author_id, coauthor_ids, title, description, category,
		       originality_declaration, privacy_consent, is_institutional, verified_by_faculty,
		       status, moderation_notes, created_at, updated_at
		FROM posts
		WHERE author_id = $1
		ORDER BY created_at DESC
	`, authorID)
	if err != nil {
		return nil, nil, err
	}
	ids := collectPostIDs(posts)
	att, err := r.listAttachmentsByPostIDs(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	return posts, att, nil
}

func (r *postRepository) ListPublic(ctx context.Context) ([]*domain.Post, map[uuid.UUID][]*domain.PostAttachment, error) {
	posts, err := r.queryPosts(ctx, `
		SELECT id, author_id, declared_author_id, coauthor_ids, title, description, category,
		       originality_declaration, privacy_consent, is_institutional, verified_by_faculty,
		       status, moderation_notes, created_at, updated_at
		FROM posts
		WHERE status = 'published'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, nil, err
	}
	ids := collectPostIDs(posts)
	att, err := r.listAttachmentsByPostIDs(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	return posts, att, nil
}

func (r *postRepository) ListByStatuses(ctx context.Context, statuses []domain.PostStatus) ([]*domain.Post, map[uuid.UUID][]*domain.PostAttachment, error) {
	if len(statuses) == 0 {
		return []*domain.Post{}, map[uuid.UUID][]*domain.PostAttachment{}, nil
	}

	args := make([]any, 0, len(statuses))
	parts := make([]string, 0, len(statuses))
	for i, status := range statuses {
		args = append(args, string(status))
		parts = append(parts, fmt.Sprintf("$%d", i+1))
	}

	query := fmt.Sprintf(`
		SELECT id, author_id, declared_author_id, coauthor_ids, title, description, category,
		       originality_declaration, privacy_consent, is_institutional, verified_by_faculty,
		       status, moderation_notes, created_at, updated_at
		FROM posts
		WHERE status IN (%s)
		ORDER BY created_at DESC
	`, strings.Join(parts, ","))

	posts, err := r.queryPosts(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	ids := collectPostIDs(posts)
	att, err := r.listAttachmentsByPostIDs(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	return posts, att, nil
}

func (r *postRepository) UpdateModeration(ctx context.Context, postID uuid.UUID, status domain.PostStatus, notes *string) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE posts
		SET status = $1, moderation_notes = $2, updated_at = NOW()
		WHERE id = $3
	`, string(status), notes, postID)
	if err != nil {
		return fmt.Errorf("postRepository.UpdateModeration: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("postRepository.UpdateModeration: publicación no encontrada")
	}
	return nil
}

func (r *postRepository) queryPosts(ctx context.Context, query string, args ...any) ([]*domain.Post, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postRepository.queryPosts: %w", err)
	}
	defer rows.Close()

	posts := make([]*domain.Post, 0)
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, nil
}

func (r *postRepository) listAttachmentsByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]*domain.PostAttachment, error) {
	result := make(map[uuid.UUID][]*domain.PostAttachment)
	if len(postIDs) == 0 {
		return result, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, post_id, file_name, file_url, file_ext, mime_type, size_bytes, uploaded_by, created_at
		FROM post_attachments
		WHERE post_id = ANY($1)
		ORDER BY created_at ASC
	`, postIDs)
	if err != nil {
		return nil, fmt.Errorf("postRepository.listAttachmentsByPostIDs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var a domain.PostAttachment
		if err := rows.Scan(
			&a.ID, &a.PostID, &a.FileName, &a.FileURL, &a.FileExt,
			&a.MimeType, &a.SizeBytes, &a.UploadedBy, &a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("postRepository.listAttachmentsByPostIDs scan: %w", err)
		}
		result[a.PostID] = append(result[a.PostID], &a)
	}

	return result, nil
}

func scanPost(rows interface{ Scan(dest ...any) error }) (*domain.Post, error) {
	var p domain.Post
	var category string
	var status string
	if err := rows.Scan(
		&p.ID, &p.AuthorID, &p.DeclaredAuthorID, &p.CoAuthorIDs, &p.Title, &p.Description, &category,
		&p.OriginalityDeclaration, &p.PrivacyConsent, &p.IsInstitutional, &p.VerifiedByFaculty,
		&status, &p.ModerationNotes, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("postRepository.scanPost: %w", err)
	}
	p.Category = domain.PostCategory(category)
	p.Status = domain.PostStatus(status)
	return &p, nil
}

func collectPostIDs(posts []*domain.Post) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(posts))
	for _, post := range posts {
		ids = append(ids, post.ID)
	}
	return ids
}
