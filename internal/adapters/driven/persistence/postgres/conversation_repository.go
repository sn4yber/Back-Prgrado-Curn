package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
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

	if err := r.tryRecordMessageEvent(ctx, message.ID, message.ConversationID, message.SenderID, "created", nil, &message.Content); err != nil {
		return fmt.Errorf("conversationRepository.CreateMessage event: %w", err)
	}

	return nil
}

func (r *conversationRepository) GetMessageByID(ctx context.Context, conversationID, messageID uuid.UUID) (*domain.Message, error) {
	softDeleteEnabled, err := r.supportsMessageSoftDelete(ctx)
	if err != nil {
		return nil, fmt.Errorf("conversationRepository.GetMessageByID soft-delete check: %w", err)
	}

	query := `
		SELECT id, conversation_id, sender_id, content, created_at
		FROM messages
		WHERE conversation_id = $1 AND id = $2
		LIMIT 1
	`
	if softDeleteEnabled {
		query = `
			SELECT id, conversation_id, sender_id, content, created_at, is_deleted, deleted_at, deleted_by
			FROM messages
			WHERE conversation_id = $1 AND id = $2
			LIMIT 1
		`
	}

	row := r.pool.QueryRow(ctx, query, conversationID, messageID)
	var message domain.Message
	if softDeleteEnabled {
		err = row.Scan(
			&message.ID,
			&message.ConversationID,
			&message.SenderID,
			&message.Content,
			&message.CreatedAt,
			&message.IsDeleted,
			&message.DeletedAt,
			&message.DeletedBy,
		)
	} else {
		err = row.Scan(
			&message.ID,
			&message.ConversationID,
			&message.SenderID,
			&message.Content,
			&message.CreatedAt,
		)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("conversationRepository.GetMessageByID: %w", err)
	}

	message.UpdatedAt = message.CreatedAt
	return &message, nil
}

func (r *conversationRepository) UpdateMessageContent(ctx context.Context, conversationID, messageID uuid.UUID, content string) (*domain.Message, error) {
	current, err := r.GetMessageByID(ctx, conversationID, messageID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, nil
	}
	if current.IsDeleted {
		return nil, nil
	}

	softDeleteEnabled, err := r.supportsMessageSoftDelete(ctx)
	if err != nil {
		return nil, fmt.Errorf("conversationRepository.UpdateMessageContent soft-delete check: %w", err)
	}

	query := `
		UPDATE messages
		SET content = $3
		WHERE conversation_id = $1 AND id = $2
		RETURNING id, conversation_id, sender_id, content, created_at
	`
	if softDeleteEnabled {
		query = `
			UPDATE messages
			SET content = $3
			WHERE conversation_id = $1 AND id = $2 AND is_deleted = FALSE
			RETURNING id, conversation_id, sender_id, content, created_at, is_deleted, deleted_at, deleted_by
		`
	}

	row := r.pool.QueryRow(ctx, query, conversationID, messageID, content)
	var message domain.Message
	if softDeleteEnabled {
		err = row.Scan(
			&message.ID,
			&message.ConversationID,
			&message.SenderID,
			&message.Content,
			&message.CreatedAt,
			&message.IsDeleted,
			&message.DeletedAt,
			&message.DeletedBy,
		)
	} else {
		err = row.Scan(
			&message.ID,
			&message.ConversationID,
			&message.SenderID,
			&message.Content,
			&message.CreatedAt,
		)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("conversationRepository.UpdateMessageContent: %w", err)
	}
	message.UpdatedAt = time.Now()

	_, err = r.pool.Exec(ctx,
		`UPDATE conversations SET updated_at = NOW() WHERE id = $1`,
		conversationID,
	)
	if err != nil {
		return nil, fmt.Errorf("conversationRepository.UpdateMessageContent update conversation: %w", err)
	}

	if err := r.tryRecordMessageEvent(ctx, message.ID, conversationID, message.SenderID, "edited", &current.Content, &message.Content); err != nil {
		return nil, fmt.Errorf("conversationRepository.UpdateMessageContent event: %w", err)
	}

	return &message, nil
}

func (r *conversationRepository) DeleteMessage(ctx context.Context, conversationID, messageID, deletedBy uuid.UUID) error {
	current, err := r.GetMessageByID(ctx, conversationID, messageID)
	if err != nil {
		return err
	}
	if current == nil {
		return nil
	}

	softDeleteEnabled, err := r.supportsMessageSoftDelete(ctx)
	if err != nil {
		return fmt.Errorf("conversationRepository.DeleteMessage soft-delete check: %w", err)
	}

	var cmd pgconn.CommandTag
	if softDeleteEnabled {
		cmd, err = r.pool.Exec(ctx, `
			UPDATE messages
			SET is_deleted = TRUE,
			    deleted_at = NOW(),
			    deleted_by = $3,
			    content = '[mensaje eliminado]'
			WHERE conversation_id = $1 AND id = $2 AND is_deleted = FALSE
		`, conversationID, messageID, deletedBy)
	} else {
		cmd, err = r.pool.Exec(ctx, `
			DELETE FROM messages
			WHERE conversation_id = $1 AND id = $2
		`, conversationID, messageID)
	}
	if err != nil {
		return fmt.Errorf("conversationRepository.DeleteMessage: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return nil
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE conversations SET updated_at = NOW() WHERE id = $1`,
		conversationID,
	)
	if err != nil {
		return fmt.Errorf("conversationRepository.DeleteMessage update conversation: %w", err)
	}

	redacted := "[mensaje eliminado]"
	if err := r.tryRecordMessageEvent(ctx, current.ID, conversationID, deletedBy, "deleted", &current.Content, &redacted); err != nil {
		return fmt.Errorf("conversationRepository.DeleteMessage event: %w", err)
	}

	return nil
}

func (r *conversationRepository) ListMessagesByConversation(ctx context.Context, conversationID uuid.UUID) ([]*domain.Message, error) {
	softDeleteEnabled, err := r.supportsMessageSoftDelete(ctx)
	if err != nil {
		return nil, fmt.Errorf("conversationRepository.ListMessagesByConversation soft-delete check: %w", err)
	}

	query := `
		SELECT id, conversation_id, sender_id, content, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
	`
	if softDeleteEnabled {
		query = `
			SELECT id, conversation_id, sender_id, content, created_at, is_deleted, deleted_at, deleted_by
			FROM messages
			WHERE conversation_id = $1
			ORDER BY created_at ASC
		`
	}

	rows, err := r.pool.Query(ctx, query, conversationID)
	if err != nil {
		return nil, fmt.Errorf("conversationRepository.ListMessagesByConversation: %w", err)
	}
	defer rows.Close()

	messages := make([]*domain.Message, 0)
	for rows.Next() {
		var message domain.Message
		if softDeleteEnabled {
			err = rows.Scan(
				&message.ID,
				&message.ConversationID,
				&message.SenderID,
				&message.Content,
				&message.CreatedAt,
				&message.IsDeleted,
				&message.DeletedAt,
				&message.DeletedBy,
			)
		} else {
			err = rows.Scan(
				&message.ID,
				&message.ConversationID,
				&message.SenderID,
				&message.Content,
				&message.CreatedAt,
			)
		}
		if err != nil {
			return nil, fmt.Errorf("conversationRepository.ListMessagesByConversation scan: %w", err)
		}
		message.UpdatedAt = message.CreatedAt
		messages = append(messages, &message)
	}

	return messages, nil
}

func (r *conversationRepository) supportsMessageSoftDelete(ctx context.Context) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = 'messages'
		  AND column_name IN ('is_deleted', 'deleted_at', 'deleted_by')
	`).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 3, nil
}

func (r *conversationRepository) messageEventsTableExists(ctx context.Context) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
			  AND table_name = 'message_events'
		)
	`).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (r *conversationRepository) tryRecordMessageEvent(
	ctx context.Context,
	messageID uuid.UUID,
	conversationID uuid.UUID,
	actorID uuid.UUID,
	eventType string,
	oldContent *string,
	newContent *string,
) error {
	exists, err := r.messageEventsTableExists(ctx)
	if err != nil || !exists {
		return nil
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO message_events (
			id, message_id, conversation_id, actor_id, event_type, old_content, new_content, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`, uuid.New(), messageID, conversationID, actorID, eventType, oldContent, newContent)
	if err != nil {
		return nil
	}

	return nil
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
