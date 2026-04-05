package input

import (
	"context"

	"github.com/google/uuid"
)

type CreateProjectRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Area        string `json:"area"`
}

type UpdateProjectStatusRequest struct {
	Status string `json:"status"`
}

type ProjectResponse struct {
	ID          string `json:"id"`
	OwnerID     string `json:"owner_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Area        string `json:"area"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ProjectUseCase interface {
	Create(ctx context.Context, ownerID uuid.UUID, req CreateProjectRequest) (*ProjectResponse, error)
	ListMine(ctx context.Context, ownerID uuid.UUID) ([]ProjectResponse, error)
	GetByID(ctx context.Context, requesterID, projectID uuid.UUID) (*ProjectResponse, error)
	UpdateStatus(ctx context.Context, ownerID, projectID uuid.UUID, req UpdateProjectStatusRequest) (*ProjectResponse, error)
}
