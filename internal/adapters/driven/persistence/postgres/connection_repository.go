package postgres

import (
	"context"
	"fmt"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
)

const (
	queryInsertConnection = `
		INSERT INTO connections (id, requester_id, addressee_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	queryUpdateStatusConnection = `
		UPDATE connections
		SET status = $1, updated_at = $2
		WHERE id = $3`

	queryGetByID = `
		SELECT id, requester_id, addressee_id, status, created_at, updated_at
		FROM connections
		WHERE id = $1`

	queryListSuggestions = `
		SELECT
			u.id,
			u.name,
			COALESCE(MAX(r.name)::text, 'estudiante') AS role_name,
			p.name AS program_name,
			u.avatar_url
		FROM users u
		LEFT JOIN programs p ON p.id = u.program_id
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN roles r ON r.id = ur.role_id
		WHERE u.id <> $1
		  AND NOT EXISTS (
			SELECT 1
			FROM connections c
			WHERE (
				(c.requester_id = $1 AND c.addressee_id = u.id)
				OR
				(c.requester_id = u.id AND c.addressee_id = $1)
			)
			AND c.status IN ('pending', 'accepted', 'blocked')
		  )
		GROUP BY u.id, u.name, p.name, u.avatar_url
		ORDER BY u.name ASC
		LIMIT $2`

	queryDiscoverUsers = `
		WITH candidates AS (
			SELECT
				u.id,
				u.name,
				COALESCE(MAX(r.name)::text, 'estudiante') AS role_name,
				p.name AS program_name,
				u.avatar_url,
				u.city,
				u.program_id
			FROM users u
			LEFT JOIN programs p ON p.id = u.program_id
			LEFT JOIN user_roles ur ON ur.user_id = u.id
			LEFT JOIN roles r ON r.id = ur.role_id
			WHERE u.id <> $1
			GROUP BY u.id, u.name, p.name, u.avatar_url, u.city, u.program_id
		),
		me AS (
			SELECT id, program_id, city
			FROM users
			WHERE id = $1
		)
		SELECT
			c.id,
			c.name,
			c.role_name,
			c.program_name,
			c.avatar_url,
			(rel.status = 'accepted') AS is_connected,
			(rel.status = 'pending') AS pending_request,
			ROUND((
				CASE WHEN rel.status = 'accepted' THEN 0.15 ELSE 0.0 END +
				CASE WHEN m.program_id IS NOT NULL AND c.program_id = m.program_id THEN 0.35 ELSE 0.0 END +
				CASE WHEN m.city IS NOT NULL AND c.city IS NOT NULL AND LOWER(m.city) = LOWER(c.city) THEN 0.20 ELSE 0.0 END +
				CASE WHEN inter.has_interaction THEN 0.30 ELSE 0.0 END
			)::numeric, 2) AS recommendation_score,
			ARRAY_REMOVE(ARRAY[
				CASE WHEN rel.status = 'accepted' THEN 'already_connected' END,
				CASE WHEN rel.status = 'pending' THEN 'pending_request' END,
				CASE WHEN m.program_id IS NOT NULL AND c.program_id = m.program_id THEN 'same_program' END,
				CASE WHEN m.city IS NOT NULL AND c.city IS NOT NULL AND LOWER(m.city) = LOWER(c.city) THEN 'same_city' END,
				CASE WHEN inter.has_interaction THEN 'high_interaction' END
			], NULL)::text[] AS recommendation_reasons,
			COUNT(*) OVER()::int AS total
		FROM candidates c
		CROSS JOIN me m
		LEFT JOIN LATERAL (
			SELECT status
			FROM connections con
			WHERE (
				(con.requester_id = $1 AND con.addressee_id = c.id)
				OR
				(con.requester_id = c.id AND con.addressee_id = $1)
			)
			ORDER BY con.updated_at DESC
			LIMIT 1
		) rel ON true
		LEFT JOIN LATERAL (
			SELECT EXISTS (
				SELECT 1
				FROM posts p
				JOIN post_reactions pr ON pr.post_id = p.id
				WHERE (p.author_id = c.id AND pr.user_id = $1)
				   OR (p.author_id = $1 AND pr.user_id = c.id)
				UNION ALL
				SELECT 1
				FROM posts p
				JOIN comments cm ON cm.post_id = p.id
				WHERE (p.author_id = c.id AND cm.author_id = $1)
				   OR (p.author_id = $1 AND cm.author_id = c.id)
				LIMIT 1
			) AS has_interaction
		) inter ON true
		WHERE ($2 = '' OR c.name ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR c.role_name ILIKE $3)
		  AND ($4 = '' OR c.program_name ILIKE '%' || $4 || '%')
		  AND ($5 = '' OR c.city ILIKE '%' || $5 || '%')
		ORDER BY recommendation_score DESC, c.name ASC
		LIMIT $6 OFFSET $7`

	queryExistsBetween = `
		SELECT EXISTS (
			SELECT 1 FROM connections
			WHERE (requester_id = $1 AND addressee_id = $2)
			   OR (requester_id = $2 AND addressee_id = $1)
		)`

	queryReopenRejectedByUsers = `
		UPDATE connections
		SET requester_id = $1,
		    addressee_id = $2,
		    status = 'pending',
		    updated_at = NOW()
		WHERE id = (
			SELECT id
			FROM connections
			WHERE (
				(requester_id = $1 AND addressee_id = $2)
				OR
				(requester_id = $2 AND addressee_id = $1)
			)
			  AND status = 'rejected'
			ORDER BY updated_at DESC
			LIMIT 1
		)`

	queryStatusBetween = `
		SELECT status
		FROM connections
		WHERE (requester_id = $1 AND addressee_id = $2)
		   OR (requester_id = $2 AND addressee_id = $1)
		ORDER BY updated_at DESC
		LIMIT 1`
)

// connectionRepositoryPostgres implementa ConnectionRepository usando PostgreSQL.
type connectionRepositoryPostgres struct {
	pool *pgxpool.Pool
}

func NewConnectionRepositoryPostgres(pool *pgxpool.Pool) *connectionRepositoryPostgres {
	return &connectionRepositoryPostgres{pool: pool}
}

// Create almacena una nueva conexión en la base de datos
func (r *connectionRepositoryPostgres) Create(ctx context.Context, conn *domain.Connection) error {
	_, err := r.pool.Exec(ctx, queryInsertConnection,
		conn.ID,
		conn.RequesterID,
		conn.AddresseeID,
		conn.Status,
		conn.CreatedAt,
		conn.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("error al crear la conexión: %w", err)
	}
	return nil
}

// UpdateStatus actualiza el estado de una conexión existente
func (r *connectionRepositoryPostgres) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ConnectionStatus) error {
	_, err := r.pool.Exec(ctx, queryUpdateStatusConnection,
		status,
		time.Now(),
		id,
	)
	return err
}

// GetByID retorna una conexión por su ID
func (r *connectionRepositoryPostgres) GetByID(ctx context.Context, id uuid.UUID) (*domain.Connection, error) {
	row := r.pool.QueryRow(ctx, queryGetByID, id)
	var connID, requesterID, addresseeID uuid.UUID
	var status domain.ConnectionStatus
	var createdAt, updatedAt time.Time
	if err := row.Scan(&connID, &requesterID, &addresseeID, &status, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	return domain.Reconstitute(connID, requesterID, addresseeID, status, createdAt, updatedAt), nil
}

// ListByUser retorna todas las conexiones en las que participa un usuario
func (r *connectionRepositoryPostgres) ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Connection, error) {
	return r.ListByUserFiltered(ctx, userID, nil, "all")
}

func (r *connectionRepositoryPostgres) ListByUserFiltered(ctx context.Context, userID uuid.UUID, status *domain.ConnectionStatus, direction string) ([]*domain.Connection, error) {
	base := `
		SELECT id, requester_id, addressee_id, status, created_at, updated_at
		FROM connections
		WHERE `

	args := []any{userID}
	argPos := 2

	direction = strings.ToLower(strings.TrimSpace(direction))
	switch direction {
	case "incoming":
		base += "addressee_id = $1"
	case "outgoing":
		base += "requester_id = $1"
	default:
		base += "(requester_id = $1 OR addressee_id = $1)"
	}

	if status != nil {
		base += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *status)
	}

	base += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.Connection
	for rows.Next() {
		var id, requesterID, addresseeID uuid.UUID
		var connStatus domain.ConnectionStatus
		var createdAt, updatedAt time.Time
		err := rows.Scan(&id, &requesterID, &addresseeID, &connStatus, &createdAt, &updatedAt)
		if err != nil {
			return nil, err
		}
		conn := domain.Reconstitute(id, requesterID, addresseeID, connStatus, createdAt, updatedAt)
		result = append(result, conn)
	}
	return result, nil
}

func (r *connectionRepositoryPostgres) ListSuggestions(ctx context.Context, userID uuid.UUID, limit int) ([]output.ConnectionSuggestion, error) {
	rows, err := r.pool.Query(ctx, queryListSuggestions, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("error listando sugerencias de conexión: %w", err)
	}
	defer rows.Close()

	result := make([]output.ConnectionSuggestion, 0, limit)
	for rows.Next() {
		var item output.ConnectionSuggestion
		if err := rows.Scan(&item.UserID, &item.Name, &item.Role, &item.ProgramName, &item.AvatarURL); err != nil {
			return nil, fmt.Errorf("error escaneando sugerencias de conexión: %w", err)
		}
		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando sugerencias de conexión: %w", err)
	}

	return result, nil
}

func (r *connectionRepositoryPostgres) DiscoverUsers(ctx context.Context, userID uuid.UUID, params output.NetworkDiscoverParams) ([]output.NetworkDiscoverItem, int, error) {
	rows, err := r.pool.Query(ctx, queryDiscoverUsers,
		userID,
		params.Query,
		params.Role,
		params.Program,
		params.City,
		params.Limit,
		params.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error listando discover users: %w", err)
	}
	defer rows.Close()

	items := make([]output.NetworkDiscoverItem, 0, params.Limit)
	total := 0
	for rows.Next() {
		var item output.NetworkDiscoverItem
		var rowTotal int
		if err := rows.Scan(
			&item.UserID,
			&item.Name,
			&item.Role,
			&item.ProgramName,
			&item.AvatarURL,
			&item.IsConnected,
			&item.PendingRequest,
			&item.RecommendationScore,
			&item.RecommendationReasons,
			&rowTotal,
		); err != nil {
			return nil, 0, fmt.Errorf("error escaneando discover users: %w", err)
		}
		total = rowTotal
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterando discover users: %w", err)
	}

	return items, total, nil
}

// ExistsBetween verifica si ya existe una conexión entre dos usuarios
func (r *connectionRepositoryPostgres) ExistsBetween(ctx context.Context, userA, userB uuid.UUID) (bool, error) {
	row := r.pool.QueryRow(ctx, queryExistsBetween, userA, userB)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *connectionRepositoryPostgres) ReopenRejectedByUsers(ctx context.Context, requesterID, addresseeID uuid.UUID) (bool, error) {
	cmd, err := r.pool.Exec(ctx, queryReopenRejectedByUsers, requesterID, addresseeID)
	if err != nil {
		return false, fmt.Errorf("error reabriendo conexión rechazada: %w", err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (r *connectionRepositoryPostgres) GetStatusBetween(ctx context.Context, userA, userB uuid.UUID) (*domain.ConnectionStatus, error) {
	row := r.pool.QueryRow(ctx, queryStatusBetween, userA, userB)
	var status domain.ConnectionStatus
	if err := row.Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &status, nil
}

