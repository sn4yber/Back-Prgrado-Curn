package user

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
	"github.com/sn4yber/curn-networking/pkg/logger"
	"go.uber.org/zap"
)

// Service implementa input.UserUseCase.
// Solo conoce los puertos — nunca detalles de HTTP ni de PostgreSQL.
type Service struct {
	userRepo output.UserRepository
	log      logger.Logger
}

// New construye el servicio con sus dependencias inyectadas.
func New(userRepo output.UserRepository, log logger.Logger) *Service {
	return &Service{
		userRepo: userRepo,
		log:      log,
	}
}

// ─── GetProfile ───────────────────────────────────────────────────────────────

// GetProfile busca el usuario por ID y retorna su perfil institucional completo.
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*input.ProfileResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	roles, err := s.userRepo.GetRolesByUserID(ctx, userID)
	if err != nil {
		s.log.Error("error cargando roles del usuario", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	user.Roles = roles

	return toProfileResponse(user), nil
}

// ─── UpdateProfile ────────────────────────────────────────────────────────────

// UpdateProfile actualiza los datos editables del usuario autenticado.
// Solo modifica los campos que vienen en el request (patch parcial).
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, req input.UpdateProfileRequest) (*input.ProfileResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	// Validar cédula duplicada si se está actualizando
	if req.DocumentID != nil && (user.DocumentID == nil || *req.DocumentID != *user.DocumentID) {
		existing, err := s.userRepo.FindByDocumentID(ctx, *req.DocumentID)
		if err == nil && existing.ID != userID {
			return nil, apperrors.New(409, "la cédula ya está registrada por otro usuario", nil)
		}
	}

	// Aplicar solo los campos que llegan en el request
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Bio != nil {
		user.Bio = req.Bio
	}
	if req.Phone != nil {
		user.Phone = req.Phone
	}
	if req.City != nil {
		user.City = req.City
	}
	if req.DocumentID != nil {
		user.DocumentID = req.DocumentID
	}
	if req.StudentCode != nil {
		user.StudentCode = req.StudentCode
	}
	if req.Semester != nil {
		user.Semester = req.Semester
	}
	if req.GraduationYear != nil {
		user.GraduationYear = req.GraduationYear
	}
	if req.IsGraduated != nil {
		user.IsGraduated = *req.IsGraduated
	}
	if req.LinkedInURL != nil {
		user.LinkedInURL = req.LinkedInURL
	}
	if req.GitHubURL != nil {
		user.GitHubURL = req.GitHubURL
	}

	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.log.Error("error actualizando perfil", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	roles, err := s.userRepo.GetRolesByUserID(ctx, userID)
	if err != nil {
		s.log.Error("error cargando roles del usuario", zap.Error(err))
		return nil, apperrors.ErrInternal
	}
	user.Roles = roles

	s.log.Audit("perfil actualizado",
		zap.String("user_id", userID.String()),
	)

	return toProfileResponse(user), nil
}

// ─── Helper interno ───────────────────────────────────────────────────────────

// toProfileResponse mapea domain.User → input.ProfileResponse.
func toProfileResponse(u *domain.User) *input.ProfileResponse {
	resp := &input.ProfileResponse{
		ID:             u.ID.String(),
		Name:           u.Name,
		Email:          u.Email,
		DocumentID:     u.DocumentID,
		Phone:          u.Phone,
		City:           u.City,
		Bio:            u.Bio,
		AvatarURL:      u.AvatarURL,
		ProgramID:      u.ProgramID.String(),
		StudentCode:    u.StudentCode,
		Semester:       u.Semester,
		GraduationYear: u.GraduationYear,
		IsGraduated:    u.IsGraduated,
		LinkedInURL:    u.LinkedInURL,
		GitHubURL:      u.GitHubURL,
		Status:         string(u.Status),
		CreatedAt:      u.CreatedAt.Format(time.RFC3339),
	}

	resp.Roles = make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		resp.Roles = append(resp.Roles, string(r.Name))
	}

	return resp
}
