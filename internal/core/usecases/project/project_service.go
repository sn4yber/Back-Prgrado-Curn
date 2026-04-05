package project

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
	projectRepo output.ProjectRepository
	userRepo    output.UserRepository
}

func New(projectRepo output.ProjectRepository, userRepo output.UserRepository) *Service {
	return &Service{
		projectRepo: projectRepo,
		userRepo:    userRepo,
	}
}

func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, req input.CreateProjectRequest) (*input.ProjectResponse, error) {
	if _, err := s.userRepo.FindByID(ctx, ownerID); err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	title := strings.TrimSpace(req.Title)
	description := strings.TrimSpace(req.Description)
	area := strings.TrimSpace(req.Area)

	if title == "" || description == "" || area == "" {
		return nil, apperrors.New(400, "title, description y area son requeridos", nil)
	}
	if len(title) < 5 {
		return nil, apperrors.New(400, "title debe tener al menos 5 caracteres", nil)
	}

	now := time.Now()
	p := &domain.Project{
		ID:          uuid.New(),
		OwnerID:     ownerID,
		Title:       title,
		Description: description,
		Area:        area,
		Status:      domain.ProjectStatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	p.Normalize()

	if err := s.projectRepo.Create(ctx, p); err != nil {
		return nil, apperrors.New(500, "no se pudo crear el proyecto", err)
	}

	return toProjectResponse(p), nil
}

func (s *Service) ListMine(ctx context.Context, ownerID uuid.UUID) ([]input.ProjectResponse, error) {
	if _, err := s.userRepo.FindByID(ctx, ownerID); err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	projects, err := s.projectRepo.ListByOwner(ctx, ownerID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudieron listar los proyectos", err)
	}

	resp := make([]input.ProjectResponse, 0, len(projects))
	for _, p := range projects {
		resp = append(resp, *toProjectResponse(p))
	}

	return resp, nil
}

func (s *Service) GetByID(ctx context.Context, requesterID, projectID uuid.UUID) (*input.ProjectResponse, error) {
	if _, err := s.userRepo.FindByID(ctx, requesterID); err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	p, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo consultar el proyecto", err)
	}
	if p == nil {
		return nil, apperrors.New(404, "proyecto no encontrado", nil)
	}

	if p.OwnerID != requesterID {
		return nil, apperrors.New(403, "acceso denegado al proyecto", nil)
	}

	return toProjectResponse(p), nil
}

func (s *Service) UpdateStatus(ctx context.Context, ownerID, projectID uuid.UUID, req input.UpdateProjectStatusRequest) (*input.ProjectResponse, error) {
	nextStatus := domain.ProjectStatus(strings.ToLower(strings.TrimSpace(req.Status)))
	switch nextStatus {
	case domain.ProjectStatusDraft, domain.ProjectStatusActive, domain.ProjectStatusCompleted, domain.ProjectStatusArchived:
	default:
		return nil, apperrors.New(400, "status de proyecto inválido", nil)
	}

	p, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, apperrors.New(500, "no se pudo consultar el proyecto", err)
	}
	if p == nil {
		return nil, apperrors.New(404, "proyecto no encontrado", nil)
	}
	if p.OwnerID != ownerID {
		return nil, apperrors.New(403, "no puedes modificar este proyecto", nil)
	}
	if !p.CanTransitionTo(nextStatus) {
		return nil, apperrors.New(422, "transición de estado inválida", nil)
	}

	if err := s.projectRepo.UpdateStatus(ctx, p.ID, nextStatus); err != nil {
		return nil, apperrors.New(500, "no se pudo actualizar el estado del proyecto", err)
	}

	p.Status = nextStatus
	p.UpdatedAt = time.Now()

	return toProjectResponse(p), nil
}

func toProjectResponse(p *domain.Project) *input.ProjectResponse {
	return &input.ProjectResponse{
		ID:          p.ID.String(),
		OwnerID:     p.OwnerID.String(),
		Title:       p.Title,
		Description: p.Description,
		Area:        p.Area,
		Status:      string(p.Status),
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
	}
}
