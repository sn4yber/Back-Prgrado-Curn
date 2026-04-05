package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

type ProjectRepository interface {
	Create(ctx context.Context, project *domain.Project) error
	FindByID(ctx context.Context, projectID uuid.UUID) (*domain.Project, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]*domain.Project, error)
	UpdateStatus(ctx context.Context, projectID uuid.UUID, status domain.ProjectStatus) error
}
