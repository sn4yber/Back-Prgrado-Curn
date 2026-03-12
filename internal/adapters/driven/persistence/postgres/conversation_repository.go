package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

type conversationRepository struct {
	pool *pgxpool.Pool
}

func NewConversationRepository(pool *pgxpool.Pool) *conversationRepository {
	return &conversationRepository{pool: pool}
}

func (r *conversationRepository) FindByID(ctx context.Context, conversationID uuid.UUID) (*domain.Conversation, error) {
	query := `
		SELECT id, user1_id, user2_id, source_type, source_id, status, created_at, updated_at
		FROM conversations
		WHERE id = $1
		LIMIT 1
	`

	row := r.pool.QueryRow(ctx, query, conversationID)
	conversation, err := scanConversation(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("conversationRepository.FindByID: %w", err)
	}

	return conversation, nil
}

func (r *conversationRepository) FindByParticipantsAndSource(
	ctx context.Context,
	userA uuid.UUID,
	userB uuid.UUID,
	sourceType domain.ConversationSourceType,
	sourceID uuid.UUID,
) (*domain.Conversation, error) {
	u1, u2 := orderUUIDs(userA, userB)

	query := `
		SELECT id, user1_id, user2_id, source_type, source_id, status, created_at, updated_at
		FROM conversations
		WHERE user1_id = $1 AND user2_id = $2 AND source_type = $3 AND source_id = $4
		LIMIT 1
	`

	row := r.pool.QueryRow(ctx, query, u1, u2, string(sourceType), sourceID)
	conversation, err := scanConversation(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("conversationRepository.FindByParticipantsAndSource: %w", err)
	}

	return conversation, nil
}

func (r *conversationRepository) CreateConversation(ctx context.Context, conversation *domain.Conversation) error {
	query := `
		INSERT INTO conversations (
			id, user1_id, user2_id, source_type, source_id, status, created_at, updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`

	_, err := r.pool.Exec(ctx, query,
		conversation.ID,
		conversation.User1ID,
		conversation.User2ID,
		string(conversation.SourceType),
		conversation.SourceID,
		string(conversation.Status),
		conversation.CreatedAt,
		conversation.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("conversationRepository.CreateConversation: %w", err)
	}

	return nil
}

func (r *conversationRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Conversation, error) {
	query := `
		SELECT id, user1_id, user2_id, source_type, source_id, status, created_at, updated_at
		FROM conversations
		WHERE user1_id = $1 OR user2_id = $1
		ORDER BY updated_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("conversationRepository.ListByUser: %w", err)
	}
	defer rows.Close()

	conversations := make([]*domain.Conversation, 0)
	for rows.Next() {
		conversation, err := scanConversation(rows)
		if err != nil {
			return nil, fmt.Errorf("conversationRepository.ListByUser scan: %w", err)
		}
		conversations = append(conversations, conversation)
	}

	return conversations, nil
}

func (r *conversationRepository) ListFlagged(ctx context.Context) ([]*domain.Conversation, error) {
	query := `
		SELECT id, user1_id, user2_id, source_type, source_id, status, created_at, updated_at
		FROM conversations
		WHERE status = 'flagged'
		ORDER BY updated_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("conversationRepository.ListFlagged: %w", err)
	}
	defer rows.Close()

	conversations := make([]*domain.Conversation, 0)
	for rows.Next() {
		conversation, err := scanConversation(rows)
		if err != nil {
			return nil, fmt.Errorf("conversationRepository.ListFlagged scan: %w", err)
		}
		conversations = append(conversations, conversation)
	}

	return conversations, nil
}

func (r *conversationRepository) CreateMessage(ctx context.Context, message *domain.Message) error {
	query := `
		INSERT INTO messages (id, conversation_id, sender_id, content, created_at)
		VALUES ($1,$2,$3,$4,$5)
	`

	_, err := r.pool.Exec(ctx, query,
		message.ID,
		message.ConversationID,
		message.SenderID,
		message.Content,
		message.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("conversationRepository.CreateMessage: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE conversations SET updated_at = NOW() WHERE id = $1`,
		message.ConversationID,
	)
	if err != nil {
		return fmt.Errorf("conversationRepository.CreateMessage update conversation: %w", err)
	}

	return nil
}

func (r *conversationRepository) ListMessagesByConversation(ctx context.Context, conversationID uuid.UUID) ([]*domain.Message, error) {
	query := `
		SELECT id, conversation_id, sender_id, content, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("conversationRepository.ListMessagesByConversation: %w", err)
	}
	defer rows.Close()

	messages := make([]*domain.Message, 0)
	for rows.Next() {
		var message domain.Message
		if err := rows.Scan(
			&message.ID,
			&message.ConversationID,
			&message.SenderID,
			&message.Content,
			&message.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("conversationRepository.ListMessagesByConversation scan: %w", err)
		}
		messages = append(messages, &message)
	}

	return messages, nil
}

func scanConversation(row pgx.Row) (*domain.Conversation, error) {
	var conversation domain.Conversation
	var sourceType string
	var status string

	err := row.Scan(
		&conversation.ID,
		&conversation.User1ID,
		&conversation.User2ID,
		&sourceType,
		&conversation.SourceID,
		&status,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	conversation.SourceType = domain.ConversationSourceType(sourceType)
	conversation.Status = domain.ConversationStatus(status)

	return &conversation, nil
}

func orderUUIDs(a, b uuid.UUID) (uuid.UUID, uuid.UUID) {
	if a.String() < b.String() {
		return a, b
	}
	return b, a
}
