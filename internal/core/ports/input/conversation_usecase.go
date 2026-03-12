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

type MessageResponse struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	Content        string `json:"content"`
	CreatedAt      string `json:"created_at"`
}

type ConversationDetailResponse struct {
	Conversation ConversationItemResponse `json:"conversation"`
	Messages     []MessageResponse        `json:"messages"`
}

type ConversationUseCase interface {
	StartConversation(ctx context.Context, requesterID uuid.UUID, req StartConversationRequest) (*ConversationDetailResponse, error)
	SendMessage(ctx context.Context, senderID uuid.UUID, conversationID uuid.UUID, req SendMessageRequest) (*MessageResponse, error)
	GetConversation(ctx context.Context, requesterID uuid.UUID, conversationID uuid.UUID) (*ConversationDetailResponse, error)
	ListMyConversations(ctx context.Context, requesterID uuid.UUID) ([]ConversationItemResponse, error)

	// Lectura institucional para retroalimentación y moderación (sin editar mensajes).
	AdminListFlagged(ctx context.Context, requesterID uuid.UUID) ([]ConversationItemResponse, error)
}
