package user

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
	"github.com/sn4yber/curn-networking/pkg/logger"
	"go.uber.org/zap"
)

var documentIDRegex = regexp.MustCompile(`^[0-9]{6,20}$`)

// Service implementa input.UserUseCase.
// Solo conoce los puertos — nunca detalles de HTTP ni de PostgreSQL.
type Service struct {
	userRepo       output.UserRepository
	connectionRepo output.ConnectionRepository
	log            logger.Logger
}

// New construye el servicio con sus dependencias inyectadas.
func New(userRepo output.UserRepository, connectionRepo output.ConnectionRepository, log logger.Logger) *Service {
	return &Service{
		userRepo:       userRepo,
		connectionRepo: connectionRepo,
		log:            log,
	}
}

// ─── GetProfile ───────────────────────────────────────────────────────────────

// GetProfile busca el usuario por ID y retorna su perfil institucional completo.
// Optimización: usa FindByIDWithRoles para evitar N+1 queries (1 query en vez de 2).
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*input.ProfileResponse, error) {
	// Usar método optimizado que trae usuario + roles en una sola query
	user, err := s.userRepo.FindByIDWithRoles(ctx, userID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	return toProfileResponse(user), nil
}

// GetPublicProfile retorna datos públicos del perfil para vistas de red/inbox.
func (s *Service) GetPublicProfile(ctx context.Context, userID uuid.UUID) (*input.PublicProfileResponse, error) {
	user, err := s.userRepo.FindByIDWithRoles(ctx, userID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	resp := toPublicProfileResponse(user)
	resp.ConnectionStatus = "none"
	return resp, nil
}

func (s *Service) GetProfileByID(ctx context.Context, requesterID, userID uuid.UUID) (*input.PublicProfileResponse, error) {
	user, err := s.userRepo.FindByIDWithRoles(ctx, userID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	resp := toPublicProfileResponse(user)
	resp.ConnectionStatus = "none"

	if requesterID == userID {
		resp.ConnectionStatus = "connected"
		return resp, nil
	}

	if s.connectionRepo == nil {
		return resp, nil
	}

	status, err := s.connectionRepo.GetStatusBetween(ctx, requesterID, userID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo consultar estado de conexión", err)
	}

	if status == nil {
		return resp, nil
	}

	switch *status {
	case domain.ConnectionAccepted:
		resp.ConnectionStatus = "connected"
	case domain.ConnectionPending:
		resp.ConnectionStatus = "pending"
	default:
		resp.ConnectionStatus = "none"
	}

	return resp, nil
}

func (s *Service) GetProgramsCatalog(ctx context.Context) (*input.ProgramsCatalogResponse, error) {
	items, err := s.userRepo.ListProgramCatalog(ctx)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo consultar el catálogo de programas", err)
	}

	grouped := make(map[string][]input.FacultyProgramsItem)
	for _, item := range items {
		grouped[item.FacultyName] = append(grouped[item.FacultyName], input.FacultyProgramsItem{
			ProgramID:   item.ProgramID.String(),
			ProgramName: item.ProgramName,
			Level:       item.Level,
		})
	}

	facultyNames := make([]string, 0, len(grouped))
	for facultyName := range grouped {
		facultyNames = append(facultyNames, facultyName)
	}
	sort.Strings(facultyNames)

	response := &input.ProgramsCatalogResponse{Faculties: make([]input.FacultyProgramsGroup, 0, len(facultyNames))}
	for _, facultyName := range facultyNames {
		programs := grouped[facultyName]
		sort.Slice(programs, func(i, j int) bool {
			return programs[i].ProgramName < programs[j].ProgramName
		})
		response.Faculties = append(response.Faculties, input.FacultyProgramsGroup{
			FacultyName: facultyName,
			Programs:    programs,
		})
	}

	return response, nil
}

// ─── UpdateProfile ────────────────────────────────────────────────────────────

// UpdateProfile actualiza los datos editables del usuario autenticado.
// Solo modifica los campos que vienen en el request (patch parcial).
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, req input.UpdateProfileRequest) (*input.ProfileResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	normalizeProfileRequest(&req)
	if err := validateProfileRequest(req); err != nil {
		return nil, err
	}

	// Validar cédula duplicada si se está actualizando
	if req.DocumentID != nil {
		existing, findErr := s.userRepo.FindByDocumentID(ctx, *req.DocumentID)
		if findErr == nil && existing.ID != userID {
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
		if user.IsGraduated && req.Semester == nil {
			user.Semester = nil
		}
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

	// Recargar usuario con roles en una sola query optimizada
	user, err = s.userRepo.FindByIDWithRoles(ctx, userID)
	if err != nil {
		s.log.Error("error recargando perfil actualizado", zap.Error(err))
		return nil, apperrors.ErrInternal
	}

	s.log.Audit("perfil actualizado",
		zap.String("user_id", userID.String()),
	)

	return toProfileResponse(user), nil
}

// ─── Helper interno ───────────────────────────────────────────────────────────

// normalizeProfileRequest normaliza y valida los campos del request del perfil.
func normalizeProfileRequest(req *input.UpdateProfileRequest) {
	req.Name = strings.TrimSpace(req.Name)
	req.Bio = normalizeOptionalString(req.Bio)
	req.Phone = normalizeOptionalString(req.Phone)
	req.City = normalizeOptionalString(req.City)
	req.DocumentID = normalizeOptionalString(req.DocumentID)
	req.StudentCode = normalizeOptionalString(req.StudentCode)
	req.LinkedInURL = normalizeOptionalString(req.LinkedInURL)
	req.GitHubURL = normalizeOptionalString(req.GitHubURL)
}

// normalizeOptionalString normaliza un puntero a string opcional.
func normalizeOptionalString(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	return &trimmed
}

// validateProfileRequest valida los campos del request del perfil.
func validateProfileRequest(req input.UpdateProfileRequest) error {
	if req.Name != "" && len(req.Name) < 2 {
		return apperrors.New(400, "el nombre debe tener al menos 2 caracteres", nil)
	}
	if req.DocumentID != nil && *req.DocumentID != "" && !documentIDRegex.MatchString(*req.DocumentID) {
		return apperrors.New(400, "document_id debe tener entre 6 y 20 dígitos", nil)
	}
	if req.Semester != nil && (*req.Semester < 1 || *req.Semester > 12) {
		return apperrors.New(400, "semester debe estar entre 1 y 12", nil)
	}
	if req.GraduationYear != nil && (*req.GraduationYear < 1990 || *req.GraduationYear > 2100) {
		return apperrors.New(400, "graduation_year inválido", nil)
	}
	if req.IsGraduated != nil && *req.IsGraduated && req.Semester != nil {
		return apperrors.New(400, "un egresado no debe tener semester informado", nil)
	}
	return nil
}

// toProfileResponse mapea domain.User → input.ProfileResponse.
func toProfileResponse(u *domain.User) *input.ProfileResponse {
	var graduationDate *string
	if u.GraduationDate != nil {
		formatted := u.GraduationDate.Format("2006-01-02")
		graduationDate = &formatted
	}

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
		FacultyName:    u.FacultyName,
		ProgramName:    u.ProgramName,
		StudentCode:    u.StudentCode,
		Semester:       u.Semester,
		GraduationYear: u.GraduationYear,
		GraduationDate: graduationDate,
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
	if len(resp.Roles) > 0 {
		resp.Role = resp.Roles[0]
	}

	return resp
}

func toPublicProfileResponse(u *domain.User) *input.PublicProfileResponse {
	resp := &input.PublicProfileResponse{
		ID:          u.ID.String(),
		Name:        u.Name,
		Email:       u.Email,
		City:        u.City,
		Bio:         u.Bio,
		AvatarURL:   u.AvatarURL,
		ProgramID:   u.ProgramID.String(),
		ProgramName: u.ProgramName,
		LinkedInURL: u.LinkedInURL,
		GitHubURL:   u.GitHubURL,
		Status:      string(u.Status),
	}

	resp.Roles = make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		resp.Roles = append(resp.Roles, string(r.Name))
	}

	return resp
}
