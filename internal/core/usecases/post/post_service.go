package post

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
)

var blockedExtensions = map[string]struct{}{
	".exe": {},
	".js":  {},
	".py":  {},
	".sh":  {},
	".bat": {},
}

var allowedByCategory = map[domain.PostCategory]map[string]struct{}{
	domain.PostCategoryTesis: {
		".pdf": {},
	},
	domain.PostCategoryTrabajo: {
		".pdf": {},
	},
	domain.PostCategoryEmprendimiento: {
		".pdf":  {},
		".jpg":  {},
		".jpeg": {},
		".png":  {},
	},
}

var prohibitedWords = []string{
	"idiota",
	"discriminacion",
	"fraude academico",
	"venta de trabajos",
	"hacer tareas por encargo",
	"plagio",
}

type Service struct {
	postRepo    output.PostRepository
	userRepo    output.UserRepository
	fileStorage output.FileStorage
}

func New(postRepo output.PostRepository, userRepo output.UserRepository, fileStorage output.FileStorage) *Service {
	return &Service{postRepo: postRepo, userRepo: userRepo, fileStorage: fileStorage}
}

func (s *Service) CreatePost(ctx context.Context, authorID uuid.UUID, req input.CreatePostRequest) (*input.PostResponse, error) {
	category, err := parseCategory(req.Category)
	if err != nil {
		return nil, apperrors.New(400, err.Error(), err)
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Description) == "" {
		return nil, apperrors.New(400, "title y description son requeridos", nil)
	}

	if !req.OriginalityDeclaration {
		return nil, apperrors.New(400, "debes aceptar la declaración de originalidad", nil)
	}

	if req.IsInstitutional && !req.VerifiedByFaculty {
		return nil, apperrors.New(400, "publicación institucional requiere verified_by_faculty=true", nil)
	}

	declaredAuthorID := authorID
	if strings.TrimSpace(req.DeclaredAuthorID) != "" {
		declaredAuthorID, err = uuid.Parse(req.DeclaredAuthorID)
		if err != nil {
			return nil, apperrors.New(400, "declared_author_id inválido", err)
		}
	}

	coAuthors, err := parseUUIDList(req.CoAuthorIDs)
	if err != nil {
		return nil, apperrors.New(400, "coauthor_ids inválidos", err)
	}

	if declaredAuthorID != authorID && !containsUUID(coAuthors, declaredAuthorID) {
		return nil, apperrors.New(400, "si declared_author_id no coincide contigo, debes declararlo en coauthor_ids", nil)
	}

	if _, err := s.userRepo.FindByID(ctx, authorID); err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	for _, co := range coAuthors {
		if _, err := s.userRepo.FindByID(ctx, co); err != nil {
			return nil, apperrors.New(400, "coautor no encontrado", err)
		}
	}

	attachments := make([]*domain.PostAttachment, 0, len(req.Attachments))
	for i, raw := range req.Attachments {
		ext := strings.ToLower(filepath.Ext(strings.TrimSpace(raw.FileName)))
		if ext == "" {
			return nil, apperrors.New(400, "archivo sin extensión", nil)
		}
		if _, blocked := blockedExtensions[ext]; blocked {
			return nil, apperrors.New(400, "tipo de archivo prohibido por seguridad", nil)
		}
		if _, allowed := allowedByCategory[category][ext]; !allowed {
			return nil, apperrors.New(400, "extensión no permitida para esta categoría", nil)
		}

		key := fmt.Sprintf("posts/%s/%d_%s", authorID.String(), i+1, sanitizeFileName(raw.FileName))
		url, err := s.fileStorage.Save(ctx, key, raw.ContentType, raw.Data)
		if err != nil {
			return nil, apperrors.ErrInternal
		}

		attachments = append(attachments, &domain.PostAttachment{
			ID:         uuid.New(),
			FileName:   raw.FileName,
			FileURL:    url,
			FileExt:    ext,
			MimeType:   raw.ContentType,
			SizeBytes:  int64(len(raw.Data)),
			UploadedBy: authorID,
			CreatedAt:  time.Now(),
		})
	}

	status := domain.PostStatusPublished
	var notes *string
	if containsProhibitedContent(req.Title + " " + req.Description) {
		status = domain.PostStatusPendingReview
		n := "Contenido en revisión automática por reglas de ética institucional"
		notes = &n
	}

	now := time.Now()
	post := &domain.Post{
		ID:                     uuid.New(),
		AuthorID:               authorID,
		DeclaredAuthorID:       declaredAuthorID,
		CoAuthorIDs:            coAuthors,
		Title:                  req.Title,
		Description:            req.Description,
		Category:               category,
		OriginalityDeclaration: req.OriginalityDeclaration,
		PrivacyConsent:         req.PrivacyConsent,
		IsInstitutional:        req.IsInstitutional,
		VerifiedByFaculty:      req.VerifiedByFaculty,
		Status:                 status,
		ModerationNotes:        notes,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	post.Normalize()

	for _, a := range attachments {
		a.PostID = post.ID
	}

	if err := s.postRepo.Create(ctx, post, attachments); err != nil {
		return nil, apperrors.ErrInternal
	}

	return toPostResponse(post, attachments, false), nil
}

func (s *Service) ListMyPosts(ctx context.Context, authorID uuid.UUID) ([]input.PostResponse, error) {
	posts, atts, err := s.postRepo.ListByAuthor(ctx, authorID)
	if err != nil {
		return nil, apperrors.ErrInternal
	}
	return toPostResponseList(posts, atts, false), nil
}

func (s *Service) ListPublicPosts(ctx context.Context) ([]input.PostResponse, error) {
	posts, atts, err := s.postRepo.ListPublic(ctx)
	if err != nil {
		return nil, apperrors.ErrInternal
	}
	return toPostResponseList(posts, atts, true), nil
}

func (s *Service) ListPendingReview(ctx context.Context, requesterID uuid.UUID) ([]input.PostResponse, error) {
	user, err := s.userRepo.FindByIDWithRoles(ctx, requesterID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	if !user.HasRole(domain.RoleAdmin) && !user.HasRole(domain.RoleAdministrativo) {
		return nil, apperrors.New(403, "acceso denegado", nil)
	}

	posts, atts, err := s.postRepo.ListByStatuses(ctx, []domain.PostStatus{domain.PostStatusPendingReview, domain.PostStatusFlagged})
	if err != nil {
		return nil, apperrors.ErrInternal
	}
	return toPostResponseList(posts, atts, false), nil
}

func (s *Service) ModeratePost(ctx context.Context, requesterID, postID uuid.UUID, req input.ModeratePostRequest) (*input.PostResponse, error) {
	user, err := s.userRepo.FindByIDWithRoles(ctx, requesterID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	if !user.HasRole(domain.RoleAdmin) && !user.HasRole(domain.RoleAdministrativo) {
		return nil, apperrors.New(403, "acceso denegado", nil)
	}

	status := domain.PostStatus(strings.ToLower(strings.TrimSpace(req.Status)))
	switch status {
	case domain.PostStatusPublished, domain.PostStatusFlagged, domain.PostStatusPendingReview, domain.PostStatusShadowBanned, domain.PostStatusRejected:
	default:
		return nil, apperrors.New(400, "status de moderación inválido", nil)
	}

	var notes *string
	if trimmed := strings.TrimSpace(req.Notes); trimmed != "" {
		notes = &trimmed
	}

	if err := s.postRepo.UpdateModeration(ctx, postID, status, notes); err != nil {
		return nil, apperrors.ErrInternal
	}

	post, atts, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, apperrors.ErrInternal
	}
	if post == nil {
		return nil, apperrors.New(404, "publicación no encontrada", nil)
	}

	return toPostResponse(post, atts, false), nil
}

func parseCategory(raw string) (domain.PostCategory, error) {
	c := domain.PostCategory(strings.ToLower(strings.TrimSpace(raw)))
	switch c {
	case domain.PostCategoryTesis, domain.PostCategoryEmprendimiento, domain.PostCategoryTrabajo:
		return c, nil
	default:
		return "", fmt.Errorf("categoría inválida")
	}
}

func parseUUIDList(values []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		id, err := uuid.Parse(trimmed)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, nil
}

func containsUUID(items []uuid.UUID, target uuid.UUID) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func containsProhibitedContent(text string) bool {
	normalized := strings.ToLower(text)
	for _, word := range prohibitedWords {
		if strings.Contains(normalized, word) {
			return true
		}
	}
	return false
}

func sanitizeFileName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	clean := re.ReplaceAllString(strings.TrimSpace(name), "_")
	if clean == "" {
		return "file"
	}
	return clean
}

func toPostResponseList(posts []*domain.Post, attMap map[uuid.UUID][]*domain.PostAttachment, publicView bool) []input.PostResponse {
	resp := make([]input.PostResponse, 0, len(posts))
	for _, p := range posts {
		resp = append(resp, *toPostResponse(p, attMap[p.ID], publicView))
	}
	return resp
}

func toPostResponse(post *domain.Post, attachments []*domain.PostAttachment, publicView bool) *input.PostResponse {
	description := post.Description
	if publicView && !post.PrivacyConsent {
		description = redactPersonalData(description)
	}

	coauthors := make([]string, 0, len(post.CoAuthorIDs))
	for _, id := range post.CoAuthorIDs {
		coauthors = append(coauthors, id.String())
	}

	attResp := make([]input.PostAttachmentResponse, 0, len(attachments))
	for _, a := range attachments {
		attResp = append(attResp, input.PostAttachmentResponse{
			ID:       a.ID.String(),
			FileName: a.FileName,
			FileURL:  a.FileURL,
			FileExt:  a.FileExt,
			MimeType: a.MimeType,
			Size:     a.SizeBytes,
		})
	}

	return &input.PostResponse{
		ID:                     post.ID.String(),
		AuthorID:               post.AuthorID.String(),
		DeclaredAuthorID:       post.DeclaredAuthorID.String(),
		CoAuthorIDs:            coauthors,
		Title:                  post.Title,
		Description:            description,
		Category:               string(post.Category),
		OriginalityDeclaration: post.OriginalityDeclaration,
		PrivacyConsent:         post.PrivacyConsent,
		IsInstitutional:        post.IsInstitutional,
		VerifiedByFaculty:      post.VerifiedByFaculty,
		Status:                 string(post.Status),
		ModerationNotes:        post.ModerationNotes,
		Attachments:            attResp,
		CreatedAt:              post.CreatedAt.Format(time.RFC3339),
		UpdatedAt:              post.UpdatedAt.Format(time.RFC3339),
	}
}

func redactPersonalData(value string) string {
	emailRegex := regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	phoneRegex := regexp.MustCompile(`\b(?:\+?57)?[0-9][0-9 -]{6,}[0-9]\b`)
	masked := emailRegex.ReplaceAllString(value, "[email oculto]")
	masked = phoneRegex.ReplaceAllString(masked, "[telefono oculto]")
	return masked
}
