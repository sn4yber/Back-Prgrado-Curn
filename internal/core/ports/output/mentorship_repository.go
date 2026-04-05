package output

import (
	"context"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

type MentorshipRepository interface {
	Create(ctx context.Context, mentorship *domain.Mentorship) error
	FindByID(ctx context.Context, mentorshipID uuid.UUID) (*domain.Mentorship, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Mentorship, error)
	UpdateDecision(ctx context.Context, mentorshipID uuid.UUID, status domain.MentorshipStatus, notes *string) error
	ExistsPendingByProjectAndMentor(ctx context.Context, projectID, mentorID uuid.UUID) (bool, error)
}
