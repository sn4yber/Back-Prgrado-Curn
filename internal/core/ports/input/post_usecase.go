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
	IsJobOffer  bool   `json:"is_job_offer"`

	// Metadatos académicos obligatorios para categoría tesis.
	Faculty         string `json:"faculty"`
	AcademicProgram string `json:"academic_program"`
	Advisor         string `json:"advisor"`

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

type UpdatePostRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

type ReactToPostRequest struct {
	Type string `json:"type"`
}

type ReactToPostResponse struct {
	PostID string `json:"post_id"`
	Type   string `json:"type"`
}

type ReportPostRequest struct {
	Reason string `json:"reason"`
}

type ReportPostResponse struct {
	PostID          string  `json:"post_id"`
	Reported        bool    `json:"reported"`
	TotalReports    int     `json:"total_reports"`
	PostStatus      string  `json:"post_status"`
	ModerationLevel string  `json:"moderation_level"`
	RuleCode        *string `json:"rule_code,omitempty"`
	RuleMessage     *string `json:"rule_message,omitempty"`
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
	AuthorName       *string  `json:"author_name,omitempty"`
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
	ModerationLevel string  `json:"moderation_level"`
	RuleCode        *string `json:"rule_code,omitempty"`
	RuleMessage     *string `json:"rule_message,omitempty"`

	ReactionsSummary    PostReactionsSummaryResponse `json:"reactions_summary"`
	CurrentUserReaction *string                      `json:"current_user_reaction"`
	LikesCount          int                          `json:"likes_count"`
	CommentsCount       int                          `json:"comments_count"`

	Attachments []PostAttachmentResponse `json:"attachments"`
	CreatedAt   string                   `json:"created_at"`
	UpdatedAt   string                   `json:"updated_at"`
}

type PostReactionsSummaryResponse struct {
	Likes   int `json:"likes"`
	Love    int `json:"love"`
	Dislike int `json:"dislike"`
}

type PostUseCase interface {
	CreatePost(ctx context.Context, authorID uuid.UUID, req CreatePostRequest) (*PostResponse, error)
	UpdatePost(ctx context.Context, requesterID, postID uuid.UUID, req UpdatePostRequest) (*PostResponse, error)
	DeletePost(ctx context.Context, requesterID, postID uuid.UUID) error
	ListMyPosts(ctx context.Context, authorID uuid.UUID) ([]PostResponse, error)
	ListPublicPosts(ctx context.Context, requesterID uuid.UUID) ([]PostResponse, error)
	ListFeedPosts(ctx context.Context, requesterID uuid.UUID) ([]PostResponse, error)
	ListPostsByUser(ctx context.Context, requesterID, userID uuid.UUID) ([]PostResponse, error)
	ReactToPost(ctx context.Context, requesterID, postID uuid.UUID, req ReactToPostRequest) (*ReactToPostResponse, error)
	ReportPost(ctx context.Context, requesterID, postID uuid.UUID, req ReportPostRequest) (*ReportPostResponse, error)
	ListPendingReview(ctx context.Context, requesterID uuid.UUID) ([]PostResponse, error)
	ModeratePost(ctx context.Context, requesterID, postID uuid.UUID, req ModeratePostRequest) (*PostResponse, error)
}
