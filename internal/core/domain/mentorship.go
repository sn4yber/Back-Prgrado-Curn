package domain

import (
	"errors"
	"github.com/google/uuid"
	"strings"
	"time"
)

type MentorshipStatus string

const (
	MentorshipStatusPending  MentorshipStatus = "pending"
	MentorshipStatusActive   MentorshipStatus = "active"
	MentorshipStatusRejected MentorshipStatus = "rejected"
)

var (
	ErrMentorshipInvalidID         = errors.New("mentorship id inválido")
	ErrMentorshipSelfRequest       = errors.New("no puedes solicitar mentoría para ti mismo")
	ErrMentorshipInvalidTransition = errors.New("transición de estado inválida")
)

type Mentorship struct {
	ID             uuid.UUID
	RequesterID    uuid.UUID
	MentorID       uuid.UUID
	ProjectID      uuid.UUID
	Status         MentorshipStatus
	RequestMessage *string
	DecisionNotes  *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	RespondedAt    *time.Time
}

func NewMentorship(requesterID, mentorID, projectID uuid.UUID, requestMessage *string) (*Mentorship, error) {
	if requesterID == uuid.Nil || mentorID == uuid.Nil || projectID == uuid.Nil {
		return nil, ErrMentorshipInvalidID
	}
	if requesterID == mentorID {
		return nil, ErrMentorshipSelfRequest
	}
	now := time.Now()
	m := &Mentorship{
		ID:             uuid.New(),
		RequesterID:    requesterID,
		MentorID:       mentorID,
		ProjectID:      projectID,
		Status:         MentorshipStatusPending,
		RequestMessage: normalizeOptionalMessage(requestMessage),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	return m, nil
}
func (m *Mentorship) Accept(notes *string) error {
	if m.Status != MentorshipStatusPending {
		return ErrMentorshipInvalidTransition
	}
	now := time.Now()
	m.Status = MentorshipStatusActive
	m.DecisionNotes = normalizeOptionalMessage(notes)
	m.RespondedAt = &now
	m.UpdatedAt = now
	return nil
}
func (m *Mentorship) Reject(notes *string) error {
	if m.Status != MentorshipStatusPending {
		return ErrMentorshipInvalidTransition
	}
	now := time.Now()
	m.Status = MentorshipStatusRejected
	m.DecisionNotes = normalizeOptionalMessage(notes)
	m.RespondedAt = &now
	m.UpdatedAt = now
	return nil
}
func normalizeOptionalMessage(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
