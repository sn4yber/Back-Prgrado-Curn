package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

// ConversationRepository define persistencia para conversaciones y mensajes 1:1.
// Nota: cuando no existe registro, los métodos Find* retornan (nil, nil).
type ConversationRepository interface {
	// FindByID busca una conversación por su ID.
	FindByID(ctx context.Context, conversationID uuid.UUID) (*domain.Conversation, error)

	// FindByParticipantsAndSource busca conversación existente por par de usuarios + contexto.
	FindByParticipantsAndSource(
		ctx context.Context,
		userA uuid.UUID,
		userB uuid.UUID,
		sourceType domain.ConversationSourceType,
		sourceID uuid.UUID,
	) (*domain.Conversation, error)

	// CreateConversation persiste una conversación nueva.
	CreateConversation(ctx context.Context, conversation *domain.Conversation) error

	// ListByUser retorna las conversaciones de un usuario participante.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Conversation, error)

	// ListFlagged retorna conversaciones marcadas para revisión institucional.
	ListFlagged(ctx context.Context) ([]*domain.Conversation, error)

	// CreateMessage persiste un mensaje dentro de una conversación.
	CreateMessage(ctx context.Context, message *domain.Message) error

	// ListMessagesByConversation retorna historial cronológico de mensajes.
	ListMessagesByConversation(ctx context.Context, conversationID uuid.UUID) ([]*domain.Message, error)
}
