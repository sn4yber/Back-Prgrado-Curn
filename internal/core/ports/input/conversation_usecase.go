package input

import (
	"context"

	"github.com/google/uuid"
)

type StartConversationRequest struct {
	OtherUserID  string `json:"other_user_id"` // (autor de la publicación o interesado)
	SourceType   string `json:"source_type"`   // "post"
	SourceID     string `json:"source_id"`     // post_id
	FirstMessage string `json:"first_message"`
}

type SendMessageRequest struct {
	Content     string             `json:"content"`
	Attachments []AttachmentUpload `json:"-"`
}

type UpdateMessageRequest struct {
	Content string `json:"content"`
}

type ConversationItemResponse struct {
	ID         string `json:"id"`
	User1ID    string `json:"user1_id"`
	User2ID    string `json:"user2_id"`
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type MessageAttachmentResponse struct {
	ID        string `json:"id"`
	FileName  string `json:"file_name"`
	FileURL   string `json:"file_url"`
	FileExt   string `json:"file_ext"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	CreatedAt string `json:"created_at"`
}

type MessageResponse struct {
	ID             string                      `json:"id"`
	ConversationID string                      `json:"conversation_id"`
	SenderID       string                      `json:"sender_id"`
	Content        string                      `json:"content"`
	IsDeleted      bool                        `json:"is_deleted"`
	DeletedAt      *string                     `json:"deleted_at,omitempty"`
	CreatedAt      string                      `json:"created_at"`
	UpdatedAt      string                      `json:"updated_at"`
	Attachments    []MessageAttachmentResponse `json:"attachments"`
}

type ConversationDetailResponse struct {
	Conversation ConversationItemResponse `json:"conversation"`
	Messages     []MessageResponse        `json:"messages"`
}

type ConversationUseCase interface {
	StartConversation(ctx context.Context, requesterID uuid.UUID, req StartConversationRequest) (*ConversationDetailResponse, error)
	SendMessage(ctx context.Context, senderID uuid.UUID, conversationID uuid.UUID, req SendMessageRequest) (*MessageResponse, error)
	UpdateMessage(ctx context.Context, requesterID, conversationID, messageID uuid.UUID, req UpdateMessageRequest) (*MessageResponse, error)
	DeleteMessage(ctx context.Context, requesterID, conversationID, messageID uuid.UUID) error
	GetConversation(ctx context.Context, requesterID uuid.UUID, conversationID uuid.UUID) (*ConversationDetailResponse, error)
	ListMyConversations(ctx context.Context, requesterID uuid.UUID) ([]ConversationItemResponse, error)

	// Lectura institucional para retroalimentación y moderación (sin editar mensajes).
	AdminListFlagged(ctx context.Context, requesterID uuid.UUID) ([]ConversationItemResponse, error)
}
