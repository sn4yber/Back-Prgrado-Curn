package input

import (
	"context"

	"github.com/google/uuid"
)

type AttachmentUpload struct {
	FileName    string
	ContentType string
	Data        []byte
}

type CreatePostRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`

	DeclaredAuthorID string   `json:"declared_author_id"`
	CoAuthorIDs      []string `json:"coauthor_ids"`

	OriginalityDeclaration bool `json:"originality_declaration"`
	PrivacyConsent         bool `json:"privacy_consent"`

	IsInstitutional   bool `json:"is_institutional"`
	VerifiedByFaculty bool `json:"verified_by_faculty"`

	Attachments []AttachmentUpload `json:"-"`
}

type ModeratePostRequest struct {
	Status string `json:"status"`
	Notes  string `json:"notes"`
}

type PostAttachmentResponse struct {
	ID       string `json:"id"`
	FileName string `json:"file_name"`
	FileURL  string `json:"file_url"`
	FileExt  string `json:"file_ext"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size_bytes"`
}

type PostResponse struct {
	ID string `json:"id"`

	AuthorID         string   `json:"author_id"`
	DeclaredAuthorID string   `json:"declared_author_id"`
	CoAuthorIDs      []string `json:"coauthor_ids"`

	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`

	OriginalityDeclaration bool `json:"originality_declaration"`
	PrivacyConsent         bool `json:"privacy_consent"`
	IsInstitutional        bool `json:"is_institutional"`
	VerifiedByFaculty      bool `json:"verified_by_faculty"`

	Status          string  `json:"status"`
	ModerationNotes *string `json:"moderation_notes,omitempty"`

	Attachments []PostAttachmentResponse `json:"attachments"`
	CreatedAt   string                   `json:"created_at"`
	UpdatedAt   string                   `json:"updated_at"`
}

type PostUseCase interface {
	CreatePost(ctx context.Context, authorID uuid.UUID, req CreatePostRequest) (*PostResponse, error)
	ListMyPosts(ctx context.Context, authorID uuid.UUID) ([]PostResponse, error)
	ListPublicPosts(ctx context.Context) ([]PostResponse, error)
	ListPendingReview(ctx context.Context, requesterID uuid.UUID) ([]PostResponse, error)
	ModeratePost(ctx context.Context, requesterID, postID uuid.UUID, req ModeratePostRequest) (*PostResponse, error)
}
