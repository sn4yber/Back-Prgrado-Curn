package input

import (
	"context"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

// ConnectionUseCase define las operaciones de negocio para conexiones entre usuarios.
// La implementación vive en usecases/connection.
type ConnectionUseCase interface {
	// RequestConnection envía una solicitud de conexión de requester a addressee.
	RequestConnection(ctx context.Context, requesterID, addresseeID uuid.UUID) error
	// AcceptConnection acepta una solicitud pendiente (solo el addressee puede hacerlo).
	AcceptConnection(ctx context.Context, connID, addresseeID uuid.UUID) error
	// RejectConnection rechaza una solicitud pendiente (solo el addressee puede hacerlo).
	RejectConnection(ctx context.Context, connID, addresseeID uuid.UUID) error
	// BlockConnection bloquea una conexión (cualquiera de los dos usuarios puede hacerlo).
	BlockConnection(ctx context.Context, connID, requesterID uuid.UUID) error
	// ListConnections retorna todas las conexiones de un usuario.
	ListConnections(ctx context.Context, userID uuid.UUID) ([]*domain.Connection, error)
	// GetConnection retorna una conexión por su ID.
	GetConnection(ctx context.Context, connID uuid.UUID) (*domain.Connection, error)
}
