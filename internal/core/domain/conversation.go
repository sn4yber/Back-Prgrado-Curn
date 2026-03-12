package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ConversationStatus string

const (
	ConversationStatusActive   ConversationStatus = "active"
	ConversationStatusClosed   ConversationStatus = "closed"
	ConversationStatusFlagged  ConversationStatus = "flagged"
	ConversationStatusArchived ConversationStatus = "archived"
)

type ConversationSourceType string

const (
	ConversationSourcePost ConversationSourceType = "post"
)

type Conversation struct {
	ID uuid.UUID

	// Ordenados para garantizar unicidad de par (min/max UUID textual)
	User1ID uuid.UUID
	User2ID uuid.UUID

	// Contexto de negocio: chat originado por interés en publicación.
	SourceType ConversationSourceType
	SourceID   uuid.UUID // post_id

	Status ConversationStatus

	CreatedAt time.Time
	UpdatedAt time.Time
}

type Message struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	Content        string
	CreatedAt      time.Time
}

func NewConversation(userA, userB uuid.UUID, sourceType ConversationSourceType, sourceID uuid.UUID) (*Conversation, error) {
	if userA == uuid.Nil || userB == uuid.Nil {
		return nil, errors.New("usuarios inválidos")
	}
	if userA == userB {
		return nil, errors.New("no se permite conversación con uno mismo")
	}
	if sourceType == "" {
		return nil, errors.New("source_type es requerido")
	}
	if sourceID == uuid.Nil {
		return nil, errors.New("source_id es requerido")
	}

	u1, u2 := orderUUIDs(userA, userB)

	now := time.Now()
	return &Conversation{
		ID:         uuid.New(),
		User1ID:    u1,
		User2ID:    u2,
		SourceType: sourceType,
		SourceID:   sourceID,
		Status:     ConversationStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (c *Conversation) HasParticipant(userID uuid.UUID) bool {
	return c.User1ID == userID || c.User2ID == userID
}

func (c *Conversation) CanSendMessage(senderID uuid.UUID, content string) error {
	if c.Status != ConversationStatusActive {
		return errors.New("la conversación no está activa")
	}
	if !c.HasParticipant(senderID) {
		return errors.New("el remitente no pertenece a la conversación")
	}
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return errors.New("el mensaje no puede estar vacío")
	}
	if len(trimmed) > 2000 {
		return errors.New("el mensaje excede el máximo de 2000 caracteres")
	}
	return nil
}

func NewMessage(conversationID, senderID uuid.UUID, content string) (*Message, error) {
	trimmed := strings.TrimSpace(content)
	if conversationID == uuid.Nil {
		return nil, errors.New("conversation_id inválido")
	}
	if senderID == uuid.Nil {
		return nil, errors.New("sender_id inválido")
	}
	if trimmed == "" {
		return nil, errors.New("content es requerido")
	}
	if len(trimmed) > 2000 {
		return nil, errors.New("content excede 2000 caracteres")
	}

	return &Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        trimmed,
		CreatedAt:      time.Now(),
	}, nil
}

func orderUUIDs(a, b uuid.UUID) (uuid.UUID, uuid.UUID) {
	as := a.String()
	bs := b.String()
	if as < bs {
		return a, b
	}
	return b, a
}
