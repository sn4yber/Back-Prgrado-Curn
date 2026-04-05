package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type ProjectStatus string

const (
	ProjectStatusDraft     ProjectStatus = "draft"
	ProjectStatusActive    ProjectStatus = "active"
	ProjectStatusCompleted ProjectStatus = "completed"
	ProjectStatusArchived  ProjectStatus = "archived"
)

type Project struct {
	ID          uuid.UUID
	OwnerID     uuid.UUID
	Title       string
	Description string
	Area        string
	Status      ProjectStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (p *Project) Normalize() {
	p.Title = strings.TrimSpace(p.Title)
	p.Description = strings.TrimSpace(p.Description)
	p.Area = strings.TrimSpace(strings.ToLower(p.Area))
}

func (p *Project) CanTransitionTo(next ProjectStatus) bool {
	switch p.Status {
	case ProjectStatusDraft:
		return next == ProjectStatusActive || next == ProjectStatusArchived
	case ProjectStatusActive:
		return next == ProjectStatusCompleted || next == ProjectStatusArchived
	case ProjectStatusCompleted:
		return next == ProjectStatusArchived
	case ProjectStatusArchived:
		return false
	default:
		return false
	}
}
