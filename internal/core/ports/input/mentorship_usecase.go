package input

import (
	"context"
	"github.com/google/uuid"
)

type RequestMentorshipRequest struct {
	MentorID       string `json:"mentor_id"`
	ProjectID      string `json:"project_id"`
	RequestMessage string `json:"request_message"`
}
type DecideMentorshipRequest struct {
	Notes string `json:"notes"`
}
type MentorshipResponse struct {
	ID             string  `json:"id"`
	RequesterID    string  `json:"requester_id"`
	MentorID       string  `json:"mentor_id"`
	ProjectID      string  `json:"project_id"`
	Status         string  `json:"status"`
	RequestMessage *string `json:"request_message,omitempty"`
	DecisionNotes  *string `json:"decision_notes,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	RespondedAt    *string `json:"responded_at,omitempty"`
}
type MentorshipUseCase interface {
	Request(ctx context.Context, requesterID uuid.UUID, req RequestMentorshipRequest) (*MentorshipResponse, error)
	Accept(ctx context.Context, mentorID, mentorshipID uuid.UUID, req DecideMentorshipRequest) (*MentorshipResponse, error)
	Reject(ctx context.Context, mentorID, mentorshipID uuid.UUID, req DecideMentorshipRequest) (*MentorshipResponse, error)
	ListMine(ctx context.Context, userID uuid.UUID) ([]MentorshipResponse, error)
}
