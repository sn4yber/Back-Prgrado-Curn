package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type PostCategory string

const (
	PostCategoryTesis          PostCategory = "tesis"
	PostCategoryEmprendimiento PostCategory = "emprendimiento"
	PostCategoryTrabajo        PostCategory = "trabajo"
)

type PostStatus string

const (
	PostStatusPublished     PostStatus = "published"
	PostStatusFlagged       PostStatus = "flagged"
	PostStatusPendingReview PostStatus = "pending_review"
	PostStatusShadowBanned  PostStatus = "shadow_banned"
	PostStatusRejected      PostStatus = "rejected"
)

type Post struct {
	ID uuid.UUID

	AuthorID         uuid.UUID
	DeclaredAuthorID uuid.UUID
	CoAuthorIDs      []uuid.UUID

	Title       string
	Description string
	Category    PostCategory

	OriginalityDeclaration bool
	PrivacyConsent         bool

	IsInstitutional   bool
	VerifiedByFaculty bool

	Status          PostStatus
	ModerationNotes *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type PostAttachment struct {
	ID uuid.UUID

	PostID uuid.UUID

	FileName   string
	FileURL    string
	FileExt    string
	MimeType   string
	SizeBytes  int64
	UploadedBy uuid.UUID

	CreatedAt time.Time
}

func (p *Post) Normalize() {
	p.Title = strings.TrimSpace(p.Title)
	p.Description = strings.TrimSpace(p.Description)
}
