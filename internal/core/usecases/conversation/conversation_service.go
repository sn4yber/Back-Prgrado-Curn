package conversation

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
)

// Service implementa input.ConversationUseCase.
type Service struct {
	conversationRepo output.ConversationRepository
	userRepo         output.UserRepository
}

func New(conversationRepo output.ConversationRepository, userRepo output.UserRepository) *Service {
	return &Service{
		conversationRepo: conversationRepo,
		userRepo:         userRepo,
	}
}

// StartConversation crea o reutiliza conversación 1:1 contextual (por publicación).
func (s *Service) StartConversation(ctx context.Context, requesterID uuid.UUID, req input.StartConversationRequest) (*input.ConversationDetailResponse, error) {
	otherUserID, err := uuid.Parse(strings.TrimSpace(req.OtherUserID))
	if err != nil {
		return nil, apperrors.ErrValidation
	}

	sourceID, err := uuid.Parse(strings.TrimSpace(req.SourceID))
	if err != nil {
		return nil, apperrors.ErrValidation
	}

	sourceType := domain.ConversationSourceType(strings.ToLower(strings.TrimSpace(req.SourceType)))
	if sourceType != domain.ConversationSourcePost {
		return nil, apperrors.New(400, "source_type inválido: solo se permite 'post'", nil)
	}

	if requesterID == otherUserID {
		return nil, apperrors.New(400, "no puedes iniciar conversación contigo mismo", nil)
	}

	if _, err := s.userRepo.FindByID(ctx, requesterID); err != nil {
		return nil, apperrors.ErrUserNotFound
	}
	if _, err := s.userRepo.FindByID(ctx, otherUserID); err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	conversation, err := s.conversationRepo.FindByParticipantsAndSource(ctx, requesterID, otherUserID, sourceType, sourceID)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	if conversation == nil {
		conversation, err = domain.NewConversation(requesterID, otherUserID, sourceType, sourceID)
		if err != nil {
			return nil, apperrors.New(400, err.Error(), err)
		}
		if err := s.conversationRepo.CreateConversation(ctx, conversation); err != nil {
			return nil, apperrors.ErrInternal
		}
	}

	firstMessage := strings.TrimSpace(req.FirstMessage)
	if firstMessage != "" {
		if err := conversation.CanSendMessage(requesterID, firstMessage); err != nil {
			return nil, apperrors.New(400, err.Error(), err)
		}

		message, err := domain.NewMessage(conversation.ID, requesterID, firstMessage)
		if err != nil {
			return nil, apperrors.New(400, err.Error(), err)
		}
		if err := s.conversationRepo.CreateMessage(ctx, message); err != nil {
			return nil, apperrors.ErrInternal
		}
	}

	messages, err := s.conversationRepo.ListMessagesByConversation(ctx, conversation.ID)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	return toDetailResponse(conversation, messages), nil
}

// SendMessage envía un mensaje en una conversación existente.
func (s *Service) SendMessage(ctx context.Context, senderID uuid.UUID, conversationID uuid.UUID, req input.SendMessageRequest) (*input.MessageResponse, error) {
	conversation, err := s.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		return nil, apperrors.ErrInternal
	}
	if conversation == nil {
		return nil, apperrors.New(404, "conversación no encontrada", nil)
	}

	if err := conversation.CanSendMessage(senderID, req.Content); err != nil {
		return nil, apperrors.New(403, err.Error(), err)
	}

	message, err := domain.NewMessage(conversation.ID, senderID, req.Content)
	if err != nil {
		return nil, apperrors.New(400, err.Error(), err)
	}

	if err := s.conversationRepo.CreateMessage(ctx, message); err != nil {
		return nil, apperrors.ErrInternal
	}

	return toMessageResponse(message), nil
}

// GetConversation retorna detalle de conversación e historial para un participante.
func (s *Service) GetConversation(ctx context.Context, requesterID uuid.UUID, conversationID uuid.UUID) (*input.ConversationDetailResponse, error) {
	conversation, err := s.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		return nil, apperrors.ErrInternal
	}
	if conversation == nil {
		return nil, apperrors.New(404, "conversación no encontrada", nil)
	}
	if !conversation.HasParticipant(requesterID) {
		return nil, apperrors.New(403, "no tienes acceso a esta conversación", nil)
	}

	messages, err := s.conversationRepo.ListMessagesByConversation(ctx, conversation.ID)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	return toDetailResponse(conversation, messages), nil
}

// ListMyConversations retorna conversaciones donde participa el usuario autenticado.
func (s *Service) ListMyConversations(ctx context.Context, requesterID uuid.UUID) ([]input.ConversationItemResponse, error) {
	conversations, err := s.conversationRepo.ListByUser(ctx, requesterID)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	items := make([]input.ConversationItemResponse, 0, len(conversations))
	for _, c := range conversations {
		items = append(items, toConversationItemResponse(c))
	}

	return items, nil
}

// AdminListFlagged retorna conversaciones marcadas para revisión institucional.
func (s *Service) AdminListFlagged(ctx context.Context, requesterID uuid.UUID) ([]input.ConversationItemResponse, error) {
	requester, err := s.userRepo.FindByIDWithRoles(ctx, requesterID)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	if !requester.HasRole(domain.RoleAdmin) && !requester.HasRole(domain.RoleAdministrativo) {
		return nil, apperrors.New(403, "acceso denegado", nil)
	}

	conversations, err := s.conversationRepo.ListFlagged(ctx)
	if err != nil {
		return nil, apperrors.ErrInternal
	}

	items := make([]input.ConversationItemResponse, 0, len(conversations))
	for _, c := range conversations {
		items = append(items, toConversationItemResponse(c))
	}

	return items, nil
}

func toDetailResponse(conversation *domain.Conversation, messages []*domain.Message) *input.ConversationDetailResponse {
	resp := &input.ConversationDetailResponse{
		Conversation: toConversationItemResponse(conversation),
		Messages:     make([]input.MessageResponse, 0, len(messages)),
	}

	for _, m := range messages {
		resp.Messages = append(resp.Messages, input.MessageResponse{
			ID:             m.ID.String(),
			ConversationID: m.ConversationID.String(),
			SenderID:       m.SenderID.String(),
			Content:        m.Content,
			CreatedAt:      m.CreatedAt.Format(time.RFC3339),
		})
	}

	return resp
}

func toConversationItemResponse(c *domain.Conversation) input.ConversationItemResponse {
	return input.ConversationItemResponse{
		ID:         c.ID.String(),
		User1ID:    c.User1ID.String(),
		User2ID:    c.User2ID.String(),
		SourceType: string(c.SourceType),
		SourceID:   c.SourceID.String(),
		Status:     string(c.Status),
		CreatedAt:  c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  c.UpdatedAt.Format(time.RFC3339),
	}
}

func toMessageResponse(m *domain.Message) *input.MessageResponse {
	return &input.MessageResponse{
		ID:             m.ID.String(),
		ConversationID: m.ConversationID.String(),
		SenderID:       m.SenderID.String(),
		Content:        m.Content,
		CreatedAt:      m.CreatedAt.Format(time.RFC3339),
	}
}
