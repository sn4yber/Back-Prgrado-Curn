package input

import (
	"context"

	"github.com/google/uuid"
)

// UpdateProfileRequest representa los campos editables del formulario de perfil.
// Es patch parcial: solo se actualiza lo que venga en el payload.
type UpdateProfileRequest struct {
	// Datos personales
	Name       string  `json:"name" validate:"omitempty,min=2,max=150"`
	Bio        *string `json:"bio" validate:"omitempty,max=500"`
	Phone      *string `json:"phone" validate:"omitempty,max=20"`
	City       *string `json:"city" validate:"omitempty,max=100"`
	DocumentID *string `json:"document_id" validate:"omitempty,numeric,min=6,max=20"`

	// Datos academicos
	StudentCode    *string `json:"student_code" validate:"omitempty,max=20"`
	Semester       *int    `json:"semester" validate:"omitempty,min=1,max=12"`
	GraduationYear *int    `json:"graduation_year" validate:"omitempty,min=1990,max=2100"`
	IsGraduated    *bool   `json:"is_graduated"`

	// Redes profesionales
	LinkedInURL *string `json:"linkedin_url" validate:"omitempty,url,max=255"`
	GitHubURL   *string `json:"github_url" validate:"omitempty,url,max=255"`
}

type ProfileResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`

	// Datos personales
	Name       string  `json:"name"`
	DocumentID *string `json:"document_id"`
	Phone      *string `json:"phone"`
	City       *string `json:"city"`
	Bio        *string `json:"bio"`
	AvatarURL  *string `json:"avatar_url"`

	// Datos academicos
	ProgramID      string   `json:"program_id"`
	ProgramName    *string  `json:"program_name"`
	StudentCode    *string  `json:"student_code"`
	Semester       *int     `json:"semester"`
	GraduationYear *int     `json:"graduation_year"`
	IsGraduated    bool     `json:"is_graduated"`
	Roles          []string `json:"roles"`

	// Redes profesionales
	LinkedInURL *string `json:"linkedin_url"`
	GitHubURL   *string `json:"github_url"`

	// Metadata
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type PublicProfileResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	City        *string  `json:"city"`
	Bio         *string  `json:"bio"`
	AvatarURL   *string  `json:"avatar_url"`
	ProgramID   string   `json:"program_id"`
	Roles       []string `json:"roles"`
	LinkedInURL *string  `json:"linkedin_url"`
	GitHubURL   *string  `json:"github_url"`
	Status      string   `json:"status"`
}

type UserUseCase interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*ProfileResponse, error)
	GetPublicProfile(ctx context.Context, userID uuid.UUID) (*PublicProfileResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileRequest) (*ProfileResponse, error)
}
