package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sn4yber/curn-networking/internal/core/domain"
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

	queryListByUser = `
		SELECT id, requester_id, addressee_id, status, created_at, updated_at
		FROM connections
		WHERE requester_id = $1 OR addressee_id = $1`

	queryExistsBetween = `
		SELECT EXISTS (
			SELECT 1 FROM connections
			WHERE (requester_id = $1 AND addressee_id = $2)
			   OR (requester_id = $2 AND addressee_id = $1)
		)`
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
	rows, err := r.pool.Query(ctx, queryListByUser, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.Connection
	for rows.Next() {
		var id, requesterID, addresseeID uuid.UUID
		var status domain.ConnectionStatus
		var createdAt, updatedAt time.Time
		err := rows.Scan(&id, &requesterID, &addresseeID, &status, &createdAt, &updatedAt)
		if err != nil {
			return nil, err
		}
		conn := domain.Reconstitute(id, requesterID, addresseeID, status, createdAt, updatedAt)
		result = append(result, conn)
	}
	return result, nil
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
