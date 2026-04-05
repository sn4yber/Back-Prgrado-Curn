package input

import (
	"context"

	"github.com/google/uuid"
)

type ModeratePostAdminRequest struct {
	Status string `json:"status"`
	Notes  string `json:"notes"`
}

type ModerationPostItem struct {
	ID          string  `json:"id"`
	AuthorID    string  `json:"author_id"`
	Title       string  `json:"title"`
	Category    string  `json:"category"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	ReviewNotes *string `json:"review_notes,omitempty"`
}

type ModerationQueueResponse struct {
	Posts []ModerationPostItem `json:"posts"`
	Total int                  `json:"total"`
}

type UploadPolicyCategory struct {
	Category     string   `json:"category"`
	AllowedExt   []string `json:"allowed_ext"`
	MaxFiles     int      `json:"max_files"`
	MaxSizeBytes int64    `json:"max_size_bytes"`
}

type UploadPoliciesResponse struct {
	BlockedExt []string               `json:"blocked_ext"`
	Categories []UploadPolicyCategory `json:"categories"`
}

type ModerationUseCase interface {
	ListQueue(ctx context.Context, requesterID uuid.UUID, statuses []string) (*ModerationQueueResponse, error)
	ModeratePost(ctx context.Context, requesterID, postID uuid.UUID, req ModeratePostAdminRequest) error
	GetUploadPolicies(ctx context.Context, requesterID uuid.UUID) (*UploadPoliciesResponse, error)
}

