package mentorship

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	postgresrepo "github.com/sn4yber/curn-networking/internal/adapters/driven/persistence/postgres"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
)

type Service struct {
	mentorshipRepo   output.MentorshipRepository
	userRepo         output.UserRepository
	projectRepo      output.ProjectRepository
	notificationRepo output.NotificationRepository
}

func New(mentorshipRepo output.MentorshipRepository, userRepo output.UserRepository, projectRepo output.ProjectRepository, notificationRepo output.NotificationRepository) *Service {
	return &Service{mentorshipRepo: mentorshipRepo, userRepo: userRepo, projectRepo: projectRepo, notificationRepo: notificationRepo}
}
func (s *Service) Request(ctx context.Context, requesterID uuid.UUID, req input.RequestMentorshipRequest) (*input.MentorshipResponse, error) {
	mentorID, err := uuid.Parse(strings.TrimSpace(req.MentorID))
	if err != nil {
		return nil, apperrors.New(400, "mentor_id inválido", err)
	}
	projectID, err := uuid.Parse(strings.TrimSpace(req.ProjectID))
	if err != nil {
		return nil, apperrors.New(400, "project_id inválido", err)
	}
	if _, err := s.userRepo.FindByID(ctx, requesterID); err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	mentor, err := s.userRepo.FindByIDWithRoles(ctx, mentorID)
	if err != nil {
		return nil, apperrors.New(404, "mentor no encontrado", nil)
	}
	if !mentor.HasRole(domain.RoleEgresado) {
		return nil, apperrors.New(422, "el usuario seleccionado no tiene rol de mentor (egresado)", nil)
	}
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo validar el proyecto", err)
	}
	if project == nil {
		return nil, apperrors.New(404, "proyecto no encontrado", nil)
	}
	if project.OwnerID != requesterID {
		return nil, apperrors.New(403, "solo el dueño del proyecto puede solicitar mentoría", nil)
	}
	exists, err := s.mentorshipRepo.ExistsPendingByProjectAndMentor(ctx, projectID, mentorID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo validar solicitudes existentes", err)
	}
	if exists {
		return nil, apperrors.New(409, "ya existe una solicitud pendiente para este mentor y proyecto", nil)
	}
	message := strings.TrimSpace(req.RequestMessage)
	var msgPtr *string
	if message != "" {
		msgPtr = &message
	}
	m, err := domain.NewMentorship(requesterID, mentorID, projectID, msgPtr)
	if err != nil {
		if errors.Is(err, domain.ErrMentorshipSelfRequest) {
			return nil, apperrors.New(422, err.Error(), nil)
		}
		return nil, apperrors.New(400, "datos de mentoría inválidos", err)
	}
	if err := s.mentorshipRepo.Create(ctx, m); err != nil {
		return nil, apperrors.New(500, "no se pudo crear la solicitud de mentoría", err)
	}

	// Notifica al mentor que recibió una nueva solicitud.
	if s.notificationRepo != nil {
		n := domain.NewNotification(
			mentorID,
			domain.NotificationTypeMentorshipRequest,
			"Nueva solicitud de mentoría",
			"Tienes una solicitud de mentoría pendiente para revisar.",
		)
		_ = s.notificationRepo.Create(ctx, n)
	}
	return toMentorshipResponse(m), nil
}
func (s *Service) Accept(ctx context.Context, mentorID, mentorshipID uuid.UUID, req input.DecideMentorshipRequest) (*input.MentorshipResponse, error) {
	m, err := s.mentorshipRepo.FindByID(ctx, mentorshipID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo consultar la mentoría", err)
	}
	if m == nil {
		return nil, apperrors.New(404, "solicitud de mentoría no encontrada", nil)
	}
	if m.MentorID != mentorID {
		return nil, apperrors.New(403, "solo el mentor destino puede aceptar", nil)
	}
	notes := strings.TrimSpace(req.Notes)
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}
	if err := m.Accept(notesPtr); err != nil {
		if errors.Is(err, domain.ErrMentorshipInvalidTransition) {
			return nil, apperrors.New(422, err.Error(), nil)
		}
		return nil, apperrors.New(400, "no se pudo aceptar la mentoría", err)
	}
	if err := s.mentorshipRepo.UpdateDecision(ctx, m.ID, m.Status, m.DecisionNotes); err != nil {
		if errors.Is(err, postgresrepo.ErrMentorshipNotFound) {
			return nil, apperrors.New(404, "solicitud de mentoría no encontrada", nil)
		}
		return nil, apperrors.New(500, "no se pudo actualizar la mentoría", err)
	}

	if s.notificationRepo != nil {
		n := domain.NewNotification(
			m.RequesterID,
			domain.NotificationTypeMentorshipAccepted,
			"Mentoría aceptada",
			"Tu solicitud de mentoría fue aceptada.",
		)
		_ = s.notificationRepo.Create(ctx, n)
	}

	project, err := s.projectRepo.FindByID(ctx, m.ProjectID)
	if err == nil && project != nil && project.Status == domain.ProjectStatusDraft {
		_ = s.projectRepo.UpdateStatus(ctx, project.ID, domain.ProjectStatusActive)
	}
	return toMentorshipResponse(m), nil
}
func (s *Service) Reject(ctx context.Context, mentorID, mentorshipID uuid.UUID, req input.DecideMentorshipRequest) (*input.MentorshipResponse, error) {
	m, err := s.mentorshipRepo.FindByID(ctx, mentorshipID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo consultar la mentoría", err)
	}
	if m == nil {
		return nil, apperrors.New(404, "solicitud de mentoría no encontrada", nil)
	}
	if m.MentorID != mentorID {
		return nil, apperrors.New(403, "solo el mentor destino puede rechazar", nil)
	}
	notes := strings.TrimSpace(req.Notes)
	var notesPtr *string
	if notes != "" {
		notesPtr = &notes
	}
	if err := m.Reject(notesPtr); err != nil {
		if errors.Is(err, domain.ErrMentorshipInvalidTransition) {
			return nil, apperrors.New(422, err.Error(), nil)
		}
		return nil, apperrors.New(400, "no se pudo rechazar la mentoría", err)
	}
	if err := s.mentorshipRepo.UpdateDecision(ctx, m.ID, m.Status, m.DecisionNotes); err != nil {
		if errors.Is(err, postgresrepo.ErrMentorshipNotFound) {
			return nil, apperrors.New(404, "solicitud de mentoría no encontrada", nil)
		}
		return nil, apperrors.New(500, "no se pudo actualizar la mentoría", err)
	}

	if s.notificationRepo != nil {
		n := domain.NewNotification(
			m.RequesterID,
			domain.NotificationTypeMentorshipRejected,
			"Mentoría rechazada",
			"Tu solicitud de mentoría fue rechazada.",
		)
		_ = s.notificationRepo.Create(ctx, n)
	}
	return toMentorshipResponse(m), nil
}
func (s *Service) ListMine(ctx context.Context, userID uuid.UUID) ([]input.MentorshipResponse, error) {
	items, err := s.mentorshipRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo listar mentorías", err)
	}
	resp := make([]input.MentorshipResponse, 0, len(items))
	for _, m := range items {
		resp = append(resp, *toMentorshipResponse(m))
	}
	return resp, nil
}
func toMentorshipResponse(m *domain.Mentorship) *input.MentorshipResponse {
	var respondedAt *string
	if m.RespondedAt != nil {
		value := m.RespondedAt.Format(time.RFC3339)
		respondedAt = &value
	}
	return &input.MentorshipResponse{
		ID:             m.ID.String(),
		RequesterID:    m.RequesterID.String(),
		MentorID:       m.MentorID.String(),
		ProjectID:      m.ProjectID.String(),
		Status:         string(m.Status),
		RequestMessage: m.RequestMessage,
		DecisionNotes:  m.DecisionNotes,
		CreatedAt:      m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      m.UpdatedAt.Format(time.RFC3339),
		RespondedAt:    respondedAt,
	}
}
