package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

// ConnectionRepository define las operaciones de persistencia para Connection
// Permite crear, actualizar, listar y verificar conexiones entre usuarios.
type ConnectionRepository interface {
	// Create almacena una nueva conexión en la base de datos
	Create(ctx context.Context, conn *domain.Connection) error
	// UpdateStatus actualiza el estado de una conexión existente
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ConnectionStatus) error
	// ListByUser retorna todas las conexiones asociadas a un usuario
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Connection, error)
	// GetByID retorna una conexión por su ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Connection, error)
	// ExistsBetween verifica si existe una conexión entre dos usuarios
	ExistsBetween(ctx context.Context, userA, userB uuid.UUID) (bool, error)
}
