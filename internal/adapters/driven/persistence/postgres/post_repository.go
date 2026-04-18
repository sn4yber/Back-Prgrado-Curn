package postgres

import (
	"context"
	"errors"
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

var ErrPostNotFound = errors.New("post no encontrado")

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
			$1,$2,$3,$4::uuid[],$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15
		)
	`,
		post.ID, post.AuthorID, post.DeclaredAuthorID, uuidSliceToStringSlice(post.CoAuthorIDs), post.Title, post.Description,
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
		SELECT id, author_id, declared_author_id, COALESCE(coauthor_ids, '{}'::uuid[])::text[], title, description, category,
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

func (r *postRepository) ExistsPublishedByID(ctx context.Context, postID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM posts
			WHERE id = $1 AND status = 'published'
		)
	`, postID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("postRepository.ExistsPublishedByID: %w", err)
	}
	return exists, nil
}

func (r *postRepository) UpdateBasic(ctx context.Context, postID uuid.UUID, title, description string, category domain.PostCategory) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE posts
		SET title = $1,
		    description = $2,
		    category = $3,
		    updated_at = NOW()
		WHERE id = $4
	`, title, description, string(category), postID)
	if err != nil {
		return fmt.Errorf("postRepository.UpdateBasic: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrPostNotFound
	}
	return nil
}

func (r *postRepository) DeleteByID(ctx context.Context, postID uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM posts WHERE id = $1`, postID)
	if err != nil {
		return fmt.Errorf("postRepository.DeleteByID: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrPostNotFound
	}
	return nil
}

func (r *postRepository) ListByAuthor(ctx context.Context, authorID uuid.UUID) ([]*domain.Post, map[uuid.UUID][]*domain.PostAttachment, error) {
	posts, err := r.queryPosts(ctx, `
		SELECT id, author_id, declared_author_id, COALESCE(coauthor_ids, '{}'::uuid[])::text[], title, description, category,
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
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.author_id, u.name AS author_name, p.declared_author_id, COALESCE(p.coauthor_ids, '{}'::uuid[])::text[],
		       p.title, p.description, p.category, p.originality_declaration, p.privacy_consent,
		       p.is_institutional, p.verified_by_faculty, p.status, p.moderation_notes,
		       COALESCE(pr.likes_count, 0) AS likes_count,
		       COALESCE(cm.comments_count, 0) AS comments_count,
		       p.created_at, p.updated_at
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN (
			SELECT post_id, COUNT(*)::int AS likes_count
			FROM post_reactions
			GROUP BY post_id
		) pr ON pr.post_id = p.id
		LEFT JOIN (
			SELECT post_id, COUNT(*)::int AS comments_count
			FROM comments
			GROUP BY post_id
		) cm ON cm.post_id = p.id
		WHERE p.status = 'published'
		ORDER BY p.created_at DESC
	`)
	if err != nil {
		return nil, nil, fmt.Errorf("postRepository.ListPublic: %w", err)
	}
	defer rows.Close()

	posts := make([]*domain.Post, 0)
	for rows.Next() {
		p, scanErr := scanPublicPost(rows)
		if scanErr != nil {
			return nil, nil, scanErr
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("postRepository.ListPublic rows: %w", err)
	}

	ids := collectPostIDs(posts)
	att, err := r.listAttachmentsByPostIDs(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	return posts, att, nil
}

func (r *postRepository) ListFeedByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Post, map[uuid.UUID][]*domain.PostAttachment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.author_id, u.name AS author_name, p.declared_author_id, COALESCE(p.coauthor_ids, '{}'::uuid[])::text[],
		       p.title, p.description, p.category, p.originality_declaration, p.privacy_consent,
		       p.is_institutional, p.verified_by_faculty, p.status, p.moderation_notes,
		       COALESCE(pr.likes_count, 0) AS likes_count,
		       COALESCE(cm.comments_count, 0) AS comments_count,
		       p.created_at, p.updated_at
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN (
			SELECT post_id, COUNT(*)::int AS likes_count
			FROM post_reactions
			GROUP BY post_id
		) pr ON pr.post_id = p.id
		LEFT JOIN (
			SELECT post_id, COUNT(*)::int AS comments_count
			FROM comments
			GROUP BY post_id
		) cm ON cm.post_id = p.id
		WHERE p.status = 'published'
		ORDER BY
			CASE
				WHEN p.author_id = $1 THEN 0
				WHEN EXISTS (
					SELECT 1
					FROM connections c
					WHERE c.status = 'accepted'
					  AND (
						(c.requester_id = $1 AND c.addressee_id = p.author_id)
						OR
						(c.addressee_id = $1 AND c.requester_id = p.author_id)
					  )
				) THEN 1
				ELSE 2
			END ASC,
			(COALESCE(pr.likes_count, 0) + COALESCE(cm.comments_count, 0)) DESC,
			p.created_at DESC
	`, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("postRepository.ListFeedByUser: %w", err)
	}
	defer rows.Close()

	posts := make([]*domain.Post, 0)
	for rows.Next() {
		p, scanErr := scanPublicPost(rows)
		if scanErr != nil {
			return nil, nil, scanErr
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("postRepository.ListFeedByUser rows: %w", err)
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
		SELECT id, author_id, declared_author_id, COALESCE(coauthor_ids, '{}'::uuid[])::text[], title, description, category,
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
		return ErrPostNotFound
	}
	return nil
}

func (r *postRepository) UpsertReaction(ctx context.Context, postID, userID uuid.UUID, reactionType domain.PostReactionType) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO post_reactions (post_id, user_id, reaction_type, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (post_id, user_id)
		DO UPDATE SET reaction_type = EXCLUDED.reaction_type, updated_at = NOW()
	`, postID, userID, string(reactionType))
	if err != nil {
		return fmt.Errorf("postRepository.UpsertReaction: %w", err)
	}
	return nil
}

func (r *postRepository) GetReactionsSummaryByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID]domain.PostReactionsSummary, error) {
	result := make(map[uuid.UUID]domain.PostReactionsSummary)
	if len(postIDs) == 0 {
		return result, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT post_id,
		       COUNT(*) FILTER (WHERE reaction_type = 'like')::int    AS likes,
		       COUNT(*) FILTER (WHERE reaction_type = 'love')::int    AS love,
		       COUNT(*) FILTER (WHERE reaction_type = 'dislike')::int AS dislike
		FROM post_reactions
		WHERE post_id = ANY($1)
		GROUP BY post_id
	`, postIDs)
	if err != nil {
		return nil, fmt.Errorf("postRepository.GetReactionsSummaryByPostIDs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var postID uuid.UUID
		var summary domain.PostReactionsSummary
		if err := rows.Scan(&postID, &summary.Likes, &summary.Love, &summary.Dislike); err != nil {
			return nil, fmt.Errorf("postRepository.GetReactionsSummaryByPostIDs scan: %w", err)
		}
		result[postID] = summary
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postRepository.GetReactionsSummaryByPostIDs rows: %w", err)
	}

	return result, nil
}

func (r *postRepository) GetCurrentUserReactionsByPostIDs(ctx context.Context, postIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]domain.PostReactionType, error) {
	result := make(map[uuid.UUID]domain.PostReactionType)
	if len(postIDs) == 0 {
		return result, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT post_id, reaction_type
		FROM post_reactions
		WHERE user_id = $1
		  AND post_id = ANY($2)
	`, userID, postIDs)
	if err != nil {
		return nil, fmt.Errorf("postRepository.GetCurrentUserReactionsByPostIDs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var postID uuid.UUID
		var reactionType string
		if err := rows.Scan(&postID, &reactionType); err != nil {
			return nil, fmt.Errorf("postRepository.GetCurrentUserReactionsByPostIDs scan: %w", err)
		}
		result[postID] = domain.PostReactionType(reactionType)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postRepository.GetCurrentUserReactionsByPostIDs rows: %w", err)
	}

	return result, nil
}

func (r *postRepository) CreateReport(ctx context.Context, postID, reporterID uuid.UUID, reason string) (bool, int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, 0, fmt.Errorf("postRepository.CreateReport begin: %w", err)
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `
		INSERT INTO post_reports (post_id, reporter_id, reason, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (post_id, reporter_id) DO NOTHING
	`, postID, reporterID, reason)
	if err != nil {
		return false, 0, fmt.Errorf("postRepository.CreateReport insert: %w", err)
	}

	var total int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM post_reports
		WHERE post_id = $1
	`, postID).Scan(&total); err != nil {
		return false, 0, fmt.Errorf("postRepository.CreateReport count: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, 0, fmt.Errorf("postRepository.CreateReport commit: %w", err)
	}

	return cmd.RowsAffected() > 0, total, nil
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postRepository.queryPosts rows: %w", err)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postRepository.listAttachmentsByPostIDs rows: %w", err)
	}

	return result, nil
}

func scanPost(rows interface{ Scan(dest ...any) error }) (*domain.Post, error) {
	var p domain.Post
	var category string
	var status string
	var coauthorIDs []string
	if err := rows.Scan(
		&p.ID, &p.AuthorID, &p.DeclaredAuthorID, &coauthorIDs, &p.Title, &p.Description, &category,
		&p.OriginalityDeclaration, &p.PrivacyConsent, &p.IsInstitutional, &p.VerifiedByFaculty,
		&status, &p.ModerationNotes, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("postRepository.scanPost: %w", err)
	}
	parsed, err := stringSliceToUUIDSlice(coauthorIDs)
	if err != nil {
		return nil, fmt.Errorf("postRepository.scanPost parse coauthor_ids: %w", err)
	}
	p.CoAuthorIDs = parsed
	p.Category = domain.PostCategory(category)
	p.Status = domain.PostStatus(status)
	return &p, nil
}

func scanPublicPost(rows interface{ Scan(dest ...any) error }) (*domain.Post, error) {
	var p domain.Post
	var category string
	var status string
	var coauthorIDs []string
	if err := rows.Scan(
		&p.ID, &p.AuthorID, &p.AuthorName, &p.DeclaredAuthorID, &coauthorIDs, &p.Title, &p.Description, &category,
		&p.OriginalityDeclaration, &p.PrivacyConsent, &p.IsInstitutional, &p.VerifiedByFaculty,
		&status, &p.ModerationNotes, &p.LikesCount, &p.CommentsCount, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("postRepository.scanPublicPost: %w", err)
	}
	parsed, err := stringSliceToUUIDSlice(coauthorIDs)
	if err != nil {
		return nil, fmt.Errorf("postRepository.scanPublicPost parse coauthor_ids: %w", err)
	}
	p.CoAuthorIDs = parsed
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

func uuidSliceToStringSlice(values []uuid.UUID) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, v.String())
	}
	return out
}

func stringSliceToUUIDSlice(values []string) ([]uuid.UUID, error) {
	if len(values) == 0 {
		return []uuid.UUID{}, nil
	}
	out := make([]uuid.UUID, 0, len(values))
	for _, v := range values {
		id, err := uuid.Parse(v)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}
