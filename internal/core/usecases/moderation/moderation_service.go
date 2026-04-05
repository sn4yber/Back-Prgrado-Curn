package moderation

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
)

type Service struct {
	postRepo output.PostRepository
	userRepo output.UserRepository
}

func New(postRepo output.PostRepository, userRepo output.UserRepository) *Service {
	return &Service{postRepo: postRepo, userRepo: userRepo}
}

func (s *Service) ListQueue(ctx context.Context, requesterID uuid.UUID, statuses []string) (*input.ModerationQueueResponse, error) {
	if err := s.ensureAdmin(ctx, requesterID); err != nil {
		return nil, err
	}

	mappedStatuses := normalizeStatuses(statuses)
	posts, _, err := s.postRepo.ListByStatuses(ctx, mappedStatuses)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo consultar la cola de moderación", err)
	}

	items := make([]input.ModerationPostItem, 0, len(posts))
	for _, p := range posts {
		ruleCode, ruleMessage := parseModerationRule(p.ModerationNotes)
		items = append(items, input.ModerationPostItem{
			ID:              p.ID.String(),
			AuthorID:        p.AuthorID.String(),
			Title:           p.Title,
			Category:        string(p.Category),
			Status:          string(p.Status),
			ModerationLevel: moderationLevelFromStatus(p.Status, p.ModerationNotes),
			RuleCode:        ruleCode,
			RuleMessage:     ruleMessage,
			CreatedAt:       p.CreatedAt.Format(time.RFC3339),
			UpdatedAt:       p.UpdatedAt.Format(time.RFC3339),
			ReviewNotes:     p.ModerationNotes,
		})
	}

	return &input.ModerationQueueResponse{Posts: items, Total: len(items)}, nil
}

func (s *Service) ModeratePost(ctx context.Context, requesterID, postID uuid.UUID, req input.ModeratePostAdminRequest) error {
	if err := s.ensureAdmin(ctx, requesterID); err != nil {
		return err
	}

	status := domain.PostStatus(strings.ToLower(strings.TrimSpace(req.Status)))
	switch status {
	case domain.PostStatusPublished, domain.PostStatusFlagged, domain.PostStatusPendingReview, domain.PostStatusShadowBanned, domain.PostStatusRejected:
	default:
		return apperrors.New(400, "status de moderación inválido", nil)
	}

	var notes *string
	if trimmed := strings.TrimSpace(req.Notes); trimmed != "" {
		notes = &trimmed
	}

	post, _, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return apperrors.New(500, "no se pudo consultar la publicación", err)
	}
	if post == nil {
		return apperrors.New(404, "publicación no encontrada", nil)
	}

	if err := s.postRepo.UpdateModeration(ctx, postID, status, notes); err != nil {
		return apperrors.New(500, "no se pudo moderar la publicación", err)
	}
	return nil
}

func (s *Service) GetUploadPolicies(ctx context.Context, requesterID uuid.UUID) (*input.UploadPoliciesResponse, error) {
	if err := s.ensureAdmin(ctx, requesterID); err != nil {
		return nil, err
	}

	return &input.UploadPoliciesResponse{
		BlockedExt: []string{".exe", ".js", ".py", ".sh", ".bat"},
		Categories: []input.UploadPolicyCategory{
			{Category: "tesis", AllowedExt: []string{".pdf"}, MaxFiles: 3, MaxSizeBytes: 15 * 1024 * 1024},
			{Category: "trabajo", AllowedExt: []string{".pdf"}, MaxFiles: 3, MaxSizeBytes: 15 * 1024 * 1024},
			{Category: "emprendimiento", AllowedExt: []string{".pdf", ".jpg", ".jpeg", ".png"}, MaxFiles: 6, MaxSizeBytes: 20 * 1024 * 1024},
		},
	}, nil
}

func (s *Service) ensureAdmin(ctx context.Context, requesterID uuid.UUID) error {
	user, err := s.userRepo.FindByIDWithRoles(ctx, requesterID)
	if err != nil {
		return apperrors.ErrUserNotFound
	}
	if !user.HasRole(domain.RoleAdmin) && !user.HasRole(domain.RoleAdministrativo) {
		return apperrors.New(403, "acceso denegado", nil)
	}
	return nil
}

func normalizeStatuses(raw []string) []domain.PostStatus {
	if len(raw) == 0 {
		return []domain.PostStatus{domain.PostStatusPendingReview, domain.PostStatusFlagged, domain.PostStatusShadowBanned}
	}

	allowed := map[domain.PostStatus]struct{}{
		domain.PostStatusPendingReview: {},
		domain.PostStatusFlagged:       {},
		domain.PostStatusShadowBanned:  {},
		domain.PostStatusRejected:      {},
		domain.PostStatusPublished:     {},
	}

	uniq := map[domain.PostStatus]struct{}{}
	out := make([]domain.PostStatus, 0, len(raw))
	for _, item := range raw {
		status := domain.PostStatus(strings.ToLower(strings.TrimSpace(item)))
		if _, ok := allowed[status]; !ok {
			continue
		}
		if _, exists := uniq[status]; exists {
			continue
		}
		uniq[status] = struct{}{}
		out = append(out, status)
	}

	if len(out) == 0 {
		return []domain.PostStatus{domain.PostStatusPendingReview, domain.PostStatusFlagged, domain.PostStatusShadowBanned}
	}

	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func moderationLevelFromStatus(status domain.PostStatus, notes *string) string {
	switch status {
	case domain.PostStatusRejected, domain.PostStatusShadowBanned:
		return "rojo"
	case domain.PostStatusPendingReview, domain.PostStatusFlagged:
		return "amarillo"
	case domain.PostStatusPublished:
		if notes != nil && strings.TrimSpace(*notes) != "" {
			return "amarillo"
		}
		return "verde"
	default:
		return "amarillo"
	}
}

func parseModerationRule(notes *string) (*string, *string) {
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
