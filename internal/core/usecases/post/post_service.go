package post

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	postgresrepo "github.com/sn4yber/curn-networking/internal/adapters/driven/persistence/postgres"
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
		".pdf":  {},
		".doc":  {},
		".docx": {},
		".jpg":  {},
		".jpeg": {},
		".png":  {},
		".mp4":  {},
	},
	domain.PostCategoryTrabajo: {
		".pdf":  {},
		".doc":  {},
		".docx": {},
		".jpg":  {},
		".jpeg": {},
		".png":  {},
		".mp4":  {},
	},
	domain.PostCategoryEmprendimiento: {
		".pdf":  {},
		".doc":  {},
		".docx": {},
		".jpg":  {},
		".jpeg": {},
		".png":  {},
		".mp4":  {},
	},
}

var hardBlockedWords = []string{
	"idiota",
	"discriminacion",
	"fraude academico",
	"venta de trabajos",
	"hacer tareas por encargo",
	"plagio",
}

var warningWords = []string{
	"rifa",
	"sorteo",
	"promocion",
	"venta externa",
	"dinero facil",
	"apuesta",
}

var fraudJobWords = []string{
	"pago por inscripcion",
	"consigna para aplicar",
	"cupo limitado pagando",
}

var allowedReportReasons = map[string]struct{}{
	"fraude":           {},
	"plagio":           {},
	"ofensivo":         {},
	"datos_personales": {},
	"otro":             {},
}

var (
	documentRegex = regexp.MustCompile(`\b\d{8,12}\b`)
	phoneRegex    = regexp.MustCompile(`\b(?:\+?57)?[0-9][0-9 -]{6,}[0-9]\b`)
	emailRegex    = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
)

const reportThresholdToShadowBan = 3

const maxAttachmentPayloadBytes = 50 << 20 // 50 MiB (máximo global para videos)
const maxDocOrImageBytes = 10 << 20 // 10 MiB (máximo para pdf, doc, jpg, etc)

type policyHit struct {
	Code    string
	Message string
}

type policyResult struct {
	Block    *policyHit
	Warnings []policyHit
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

	if category == domain.PostCategoryTesis {
		if len(req.Attachments) == 0 {
			return nil, apperrors.New(400, "la categoría tesis requiere adjuntar PDF", nil)
		}
		if strings.TrimSpace(req.Faculty) == "" || strings.TrimSpace(req.AcademicProgram) == "" || strings.TrimSpace(req.Advisor) == "" {
			return nil, apperrors.New(400, "para tesis debes enviar faculty, academic_program y advisor", nil)
		}
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

	author, err := s.userRepo.FindByIDWithRoles(ctx, authorID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	if req.IsJobOffer {
		if !author.HasRole(domain.RoleEgresado) && !author.HasRole(domain.RoleAdmin) && !author.HasRole(domain.RoleAdministrativo) {
			return nil, apperrors.New(403, "solo egresados o administradores pueden publicar ofertas de empleo", nil)
		}
	}

	for _, co := range coAuthors {
		if _, err := s.userRepo.FindByID(ctx, co); err != nil {
			return nil, apperrors.New(400, "coautor no encontrado", err)
		}
	}

	attachments := make([]*domain.PostAttachment, 0, len(req.Attachments))
	attachmentURLByHash := make(map[string]string, len(req.Attachments))
	for i, raw := range req.Attachments {
		if len(raw.Data) == 0 {
			return nil, apperrors.New(400, "archivo adjunto vacío", nil)
		}
		if len(raw.Data) > maxAttachmentPayloadBytes {
			return nil, apperrors.New(400, "adjunto supera el tamaño máximo global permitido (50MB)", nil)
		}

		ext := strings.ToLower(filepath.Ext(strings.TrimSpace(raw.FileName)))
		
		// Validar pesos específicos por tipo (Fase 1)
		if ext == ".mp4" && len(raw.Data) > (50<<20) {
			return nil, apperrors.New(400, "los videos no pueden superar los 50MB", nil)
		} else if ext != ".mp4" && len(raw.Data) > maxDocOrImageBytes {
			return nil, apperrors.New(400, "las imágenes y documentos no pueden superar los 10MB", nil)
		}

		if ext == "" {
			return nil, apperrors.New(400, "archivo sin extensión", nil)
		}
		if _, blocked := blockedExtensions[ext]; blocked {
			return nil, apperrors.New(400, "tipo de archivo prohibido por seguridad", nil)
		}
		if _, allowed := allowedByCategory[category][ext]; !allowed {
			return nil, apperrors.New(400, "extensión no permitida para esta categoría", nil)
		}

		hashBytes := sha256.Sum256(raw.Data)
		hash := hex.EncodeToString(hashBytes[:])
		url, reused := attachmentURLByHash[hash]
		if !reused {
			key := fmt.Sprintf("posts/%s/%s_%d_%s", authorID.String(), hash[:16], i+1, sanitizeFileName(raw.FileName))
			url, err = s.fileStorage.Save(ctx, key, raw.ContentType, raw.Data)
			if err != nil {
				log.Printf("ERROR fileStorage.Save: author_id=%s file=%s content_type=%s err=%v", authorID.String(), raw.FileName, raw.ContentType, err)
				return nil, apperrors.ErrInternal
			}
			attachmentURLByHash[hash] = url
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
	policy := evaluatePublicationPolicy(req.Title, req.Description, req.IsJobOffer)
	if policy.Block != nil {
		return nil, apperrors.New(422, fmt.Sprintf("[%s] %s", policy.Block.Code, policy.Block.Message), nil)
	}

	var notes *string
	if len(policy.Warnings) > 0 {
		n := encodePolicyNotes(policy.Warnings)
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
		Faculty:                strings.TrimSpace(req.Faculty),
		AcademicProgram:        strings.TrimSpace(req.AcademicProgram),
		Advisor:                strings.TrimSpace(req.Advisor),
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
		log.Printf("ERROR postRepo.Create: post_id=%s author_id=%s category=%s attachments=%d err=%v", post.ID.String(), authorID.String(), post.Category, len(attachments), err)
		return nil, apperrors.New(500, "no se pudo guardar la publicación", err)
	}

	return toPostResponse(post, attachments, false, domain.PostReactionsSummary{}, nil), nil
}

func (s *Service) UpdatePost(ctx context.Context, requesterID, postID uuid.UUID, req input.UpdatePostRequest) (*input.PostResponse, error) {
	post, _, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo validar la publicación", err)
	}
	if post == nil {
		return nil, apperrors.New(404, "publicación no encontrada", nil)
	}
	if post.AuthorID != requesterID {
		return nil, apperrors.New(403, "solo el autor puede editar la publicación", nil)
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Description) == "" {
		return nil, apperrors.New(400, "title y description son requeridos", nil)
	}

	category, err := parseCategory(req.Category)
	if err != nil {
		return nil, apperrors.New(400, err.Error(), err)
	}

	if err := s.postRepo.UpdateBasic(ctx, postID, strings.TrimSpace(req.Title), strings.TrimSpace(req.Description), category); err != nil {
		if errors.Is(err, postgresrepo.ErrPostNotFound) {
			return nil, apperrors.New(404, "publicación no encontrada", nil)
		}
		return nil, apperrors.New(500, "no se pudo actualizar la publicación", err)
	}

	updated, atts, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo consultar la publicación actualizada", err)
	}
	if updated == nil {
		return nil, apperrors.New(404, "publicación no encontrada", nil)
	}

	return toPostResponse(updated, atts, false, domain.PostReactionsSummary{}, nil), nil
}

func (s *Service) DeletePost(ctx context.Context, requesterID, postID uuid.UUID) error {
	post, _, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return apperrors.New(500, "no se pudo validar la publicación", err)
	}
	if post == nil {
		return apperrors.New(404, "publicación no encontrada", nil)
	}
	if post.AuthorID != requesterID {
		return apperrors.New(403, "solo el autor puede eliminar la publicación", nil)
	}

	if err := s.postRepo.DeleteByID(ctx, postID); err != nil {
		if errors.Is(err, postgresrepo.ErrPostNotFound) {
			return apperrors.New(404, "publicación no encontrada", nil)
		}
		return apperrors.New(500, "no se pudo eliminar la publicación", err)
	}

	return nil
}

func (s *Service) ListMyPosts(ctx context.Context, authorID uuid.UUID, limit, offset int) ([]input.PostResponse, error) {
	posts, atts, err := s.postRepo.ListByAuthor(ctx, authorID, limit, offset)
	if err != nil {
		return nil, apperrors.New(500, "no se pudieron listar tus publicaciones", err)
	}
	return toPostResponseList(posts, atts, false, nil, nil), nil
}

func (s *Service) ListPublicPosts(ctx context.Context, requesterID uuid.UUID, limit, offset int) ([]input.PostResponse, error) {
	posts, atts, err := s.postRepo.ListPublic(ctx, limit, offset)
	if err != nil {
		return nil, apperrors.New(500, "no se pudieron listar publicaciones públicas", err)
	}

	postIDs := collectPostIDs(posts)
	reactionsSummary, err := s.postRepo.GetReactionsSummaryByPostIDs(ctx, postIDs)
	if err != nil {
		return nil, apperrors.New(500, "no se pudieron consultar las reacciones", err)
	}

	currentUserReactions, err := s.postRepo.GetCurrentUserReactionsByPostIDs(ctx, postIDs, requesterID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo consultar la reacción del usuario", err)
	}

	return toPostResponseList(posts, atts, true, reactionsSummary, currentUserReactions), nil
}

func (s *Service) ListFeedPosts(ctx context.Context, requesterID uuid.UUID, limit, offset int) ([]input.PostResponse, error) {
	posts, atts, err := s.postRepo.ListFeedByUser(ctx, requesterID, limit, offset)
	if err != nil {
		return nil, apperrors.New(500, "no se pudieron listar publicaciones del feed", err)
	}

	postIDs := collectPostIDs(posts)
	reactionsSummary, err := s.postRepo.GetReactionsSummaryByPostIDs(ctx, postIDs)
	if err != nil {
		return nil, apperrors.New(500, "no se pudieron consultar las reacciones", err)
	}

	currentUserReactions, err := s.postRepo.GetCurrentUserReactionsByPostIDs(ctx, postIDs, requesterID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo consultar la reacción del usuario", err)
	}

	return toPostResponseList(posts, atts, true, reactionsSummary, currentUserReactions), nil
}

func (s *Service) ListPostsByUser(ctx context.Context, requesterID, userID uuid.UUID, limit, offset int) ([]input.PostResponse, error) {
	if _, err := s.userRepo.FindByID(ctx, userID); err != nil {
		return nil, apperrors.New(404, "usuario no encontrado", err)
	}

	posts, atts, err := s.postRepo.ListPublicByAuthor(ctx, userID, limit, offset)
	if err != nil {
		return nil, apperrors.New(500, "no se pudieron listar publicaciones del usuario", err)
	}

	postIDs := collectPostIDs(posts)
	reactionsSummary, err := s.postRepo.GetReactionsSummaryByPostIDs(ctx, postIDs)
	if err != nil {
		return nil, apperrors.New(500, "no se pudieron consultar las reacciones", err)
	}

	currentUserReactions, err := s.postRepo.GetCurrentUserReactionsByPostIDs(ctx, postIDs, requesterID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo consultar la reacción del usuario", err)
	}

	return toPostResponseList(posts, atts, true, reactionsSummary, currentUserReactions), nil
}

func (s *Service) ReactToPost(ctx context.Context, requesterID, postID uuid.UUID, req input.ReactToPostRequest) (*input.ReactToPostResponse, error) {
	reactionType, err := parseReactionType(req.Type)
	if err != nil {
		return nil, apperrors.New(400, err.Error(), err)
	}

	exists, err := s.postRepo.ExistsPublishedByID(ctx, postID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo validar la publicación", err)
	}
	if !exists {
		return nil, apperrors.New(404, "publicación no encontrada", nil)
	}

	if err := s.postRepo.UpsertReaction(ctx, postID, requesterID, reactionType); err != nil {
		return nil, apperrors.New(500, "no se pudo registrar la reacción", err)
	}

	return &input.ReactToPostResponse{
		PostID: postID.String(),
		Type:   string(reactionType),
	}, nil
}

func (s *Service) ReportPost(ctx context.Context, requesterID, postID uuid.UUID, req input.ReportPostRequest) (*input.ReportPostResponse, error) {
	reason := strings.ToLower(strings.TrimSpace(req.Reason))
	if reason == "" {
		return nil, apperrors.New(400, "reason es requerido", nil)
	}
	if _, ok := allowedReportReasons[reason]; !ok {
		return nil, apperrors.New(400, "reason inválido: usa fraude|plagio|ofensivo|datos_personales|otro", nil)
	}
	if len(reason) > 500 {
		return nil, apperrors.New(400, "reason no puede superar 500 caracteres", nil)
	}

	post, _, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo validar la publicación", err)
	}
	if post == nil {
		return nil, apperrors.New(404, "publicación no encontrada", nil)
	}
	if post.AuthorID == requesterID {
		return nil, apperrors.New(400, "no puedes reportar tu propia publicación", nil)
	}

	reported, totalReports, err := s.postRepo.CreateReport(ctx, postID, requesterID, reason)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo reportar la publicación", err)
	}

	postStatus := post.Status
	moderationLevel := moderationLevelFromPost(post)
	ruleCode, ruleMessage := parsePrimaryRule(post.ModerationNotes)
	if totalReports >= reportThresholdToShadowBan && post.Status == domain.PostStatusPublished {
		note := fmt.Sprintf("[RULE_REPORT_THRESHOLD] publicación oculta automáticamente por alcanzar %d reportes", totalReports)
		if err := s.postRepo.UpdateModeration(ctx, postID, domain.PostStatusShadowBanned, &note); err != nil {
			if errors.Is(err, postgresrepo.ErrPostNotFound) {
				return nil, apperrors.New(404, "publicación no encontrada", nil)
			}
			return nil, apperrors.New(500, "no se pudo actualizar estado de moderación", err)
		}
		postStatus = domain.PostStatusShadowBanned
		moderationLevel = "rojo"
		rc := "RULE_REPORT_THRESHOLD"
		rm := "umbral de reportes alcanzado"
		ruleCode, ruleMessage = &rc, &rm
	}

	if totalReports < reportThresholdToShadowBan {
		postStatus = post.Status
	}

	return &input.ReportPostResponse{
		PostID:          postID.String(),
		Reported:        reported,
		TotalReports:    totalReports,
		PostStatus:      string(postStatus),
		ModerationLevel: moderationLevel,
		RuleCode:        ruleCode,
		RuleMessage:     ruleMessage,
	}, nil
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
		return nil, apperrors.New(500, "no se pudo consultar la cola de moderación", err)
	}
	return toPostResponseList(posts, atts, false, nil, nil), nil
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
		if errors.Is(err, postgresrepo.ErrPostNotFound) {
			return nil, apperrors.New(404, "publicación no encontrada", nil)
		}
		return nil, apperrors.New(500, "no se pudo moderar la publicación", err)
	}

	post, atts, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo consultar la publicación moderada", err)
	}
	if post == nil {
		return nil, apperrors.New(404, "publicación no encontrada", nil)
	}

	return toPostResponse(post, atts, false, domain.PostReactionsSummary{}, nil), nil
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

func parseReactionType(raw string) (domain.PostReactionType, error) {
	r := domain.PostReactionType(strings.ToLower(strings.TrimSpace(raw)))
	switch r {
	case domain.PostReactionLike, domain.PostReactionLove, domain.PostReactionDislike:
		return r, nil
	default:
		return "", fmt.Errorf("tipo de reacción inválido")
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

func collectPostIDs(posts []*domain.Post) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(posts))
	for _, p := range posts {
		ids = append(ids, p.ID)
	}
	return ids
}

func evaluatePublicationPolicy(title, description string, isJobOffer bool) policyResult {
	text := strings.ToLower(strings.TrimSpace(title + " " + description))
	result := policyResult{Warnings: make([]policyHit, 0, 3)}

	if containsAnyKeyword(text, hardBlockedWords) {
		result.Block = &policyHit{Code: "RULE_CONDUCT_HARD", Message: "la publicación incumple el reglamento de convivencia institucional"}
		return result
	}

	if isJobOffer && containsAnyKeyword(text, fraudJobWords) {
		result.Block = &policyHit{Code: "RULE_JOB_FRAUD", Message: "oferta laboral bloqueada por patrones de posible fraude"}
		return result
	}

	if containsAnyKeyword(text, warningWords) {
		result.Warnings = append(result.Warnings, policyHit{Code: "RULE_COMMERCIAL_WARNING", Message: "contenido con términos comerciales: monitoreo administrativo"})
	}

	if containsSensitiveData(text) {
		result.Warnings = append(result.Warnings, policyHit{Code: "RULE_HABEAS_DATA_WARNING", Message: "detección de posibles datos sensibles: revisar cumplimiento habeas data"})
	}

	return result
}

func encodePolicyNotes(hits []policyHit) string {
	parts := make([]string, 0, len(hits))
	for _, hit := range hits {
		parts = append(parts, fmt.Sprintf("[%s] %s", hit.Code, hit.Message))
	}
	return strings.Join(parts, " | ")
}

func moderationLevelFromPost(post *domain.Post) string {
	switch post.Status {
	case domain.PostStatusRejected, domain.PostStatusShadowBanned:
		return "rojo"
	case domain.PostStatusPendingReview, domain.PostStatusFlagged:
		return "amarillo"
	case domain.PostStatusPublished:
		if post.ModerationNotes != nil && strings.TrimSpace(*post.ModerationNotes) != "" {
			return "amarillo"
		}
		return "verde"
	default:
		return "amarillo"
	}
}

func parsePrimaryRule(notes *string) (*string, *string) {
	if notes == nil || strings.TrimSpace(*notes) == "" {
		return nil, nil
	}
	text := strings.TrimSpace(*notes)
	start := strings.Index(text, "[")
	end := strings.Index(text, "]")
	if start != -1 && end > start+1 {
		code := strings.TrimSpace(text[start+1 : end])
		message := strings.TrimSpace(text[end+1:])
		if idx := strings.Index(message, "|"); idx != -1 {
			message = strings.TrimSpace(message[:idx])
		}
		if code != "" && message != "" {
			return &code, &message
		}
	}
	message := text
	return nil, &message
}

func containsAnyKeyword(text string, words []string) bool {
	for _, w := range words {
		if strings.Contains(text, strings.ToLower(w)) {
			return true
		}
	}
	return false
}

func containsSensitiveData(text string) bool {
	if documentRegex.MatchString(text) {
		return true
	}
	if phoneRegex.MatchString(text) {
		return true
	}
	emails := emailRegex.FindAllString(text, -1)
	for _, email := range emails {
		if !strings.HasSuffix(email, "@campusuninunez.edu.co") && !strings.HasSuffix(email, "@curn.edu.co") {
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

func toPostResponseList(
	posts []*domain.Post,
	attMap map[uuid.UUID][]*domain.PostAttachment,
	publicView bool,
	reactionsMap map[uuid.UUID]domain.PostReactionsSummary,
	currentUserReactions map[uuid.UUID]domain.PostReactionType,
) []input.PostResponse {
	resp := make([]input.PostResponse, 0, len(posts))
	for _, p := range posts {
		summary := domain.PostReactionsSummary{}
		if reactionsMap != nil {
			summary = reactionsMap[p.ID]
		}

		var currentReaction *string
		if currentUserReactions != nil {
			if reactionType, ok := currentUserReactions[p.ID]; ok {
				r := string(reactionType)
				currentReaction = &r
			}
		}

		resp = append(resp, *toPostResponse(p, attMap[p.ID], publicView, summary, currentReaction))
	}
	return resp
}

func toPostResponse(
	post *domain.Post,
	attachments []*domain.PostAttachment,
	publicView bool,
	reactionsSummary domain.PostReactionsSummary,
	currentUserReaction *string,
) *input.PostResponse {
	likesCount := post.LikesCount
	if likesCount == 0 {
		likesCount = reactionsSummary.Likes + reactionsSummary.Love + reactionsSummary.Dislike
	}

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
		AuthorName:             post.AuthorName,
		AuthorAvatarURL:        post.AuthorAvatarURL,
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
		ModerationLevel:        moderationLevelFromPost(post),
		RuleCode:               func() *string { rc, _ := parsePrimaryRule(post.ModerationNotes); return rc }(),
		RuleMessage:            func() *string { _, rm := parsePrimaryRule(post.ModerationNotes); return rm }(),
		ReactionsSummary: input.PostReactionsSummaryResponse{
			Likes:   reactionsSummary.Likes,
			Love:    reactionsSummary.Love,
			Dislike: reactionsSummary.Dislike,
		},
		CurrentUserReaction: currentUserReaction,
		LikesCount:          likesCount,
		CommentsCount:       post.CommentsCount,
		Attachments:         attResp,
		CreatedAt:           post.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           post.UpdatedAt.Format(time.RFC3339),
	}
}

func redactPersonalData(value string) string {
	emailRegex := regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`)
	phoneRegex := regexp.MustCompile(`\b(?:\+?57)?[0-9][0-9 -]{6,}[0-9]\b`)
	masked := emailRegex.ReplaceAllString(value, "[email oculto]")
	masked = phoneRegex.ReplaceAllString(masked, "[telefono oculto]")
	return masked
}
