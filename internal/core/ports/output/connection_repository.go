package output

import (
	"context"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

type ConnectionSuggestion struct {
	UserID      uuid.UUID
	Name        string
	Role        string
	ProgramName *string
	AvatarURL   *string
}

type NetworkDiscoverParams struct {
	Query   string
	Role    string
	Program string
	City    string
	Limit   int
	Offset  int
}

type NetworkDiscoverItem struct {
	UserID                 uuid.UUID
	Name                   string
	Role                   string
	ProgramName            *string
	AvatarURL              *string
	IsConnected            bool
	PendingRequest         bool
	RecommendationScore    float64
	RecommendationReasons  []string
}

// ConnectionRepository define las operaciones de persistencia para Connection
// Permite crear, actualizar, listar y verificar conexiones entre usuarios.
type ConnectionRepository interface {
	// Create almacena una nueva conexión en la base de datos
	Create(ctx context.Context, conn *domain.Connection) error
	// UpdateStatus actualiza el estado de una conexión existente
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ConnectionStatus) error
	// ListByUser retorna todas las conexiones asociadas a un usuario
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Connection, error)
	// ListByUserFiltered retorna conexiones de un usuario con filtros opcionales.
	ListByUserFiltered(ctx context.Context, userID uuid.UUID, status *domain.ConnectionStatus, direction string) ([]*domain.Connection, error)
	// ListSuggestions retorna usuarios sugeridos para conexión.
	ListSuggestions(ctx context.Context, userID uuid.UUID, limit int) ([]ConnectionSuggestion, error)
	// DiscoverUsers lista usuarios con filtros y metadatos de recomendación.
	DiscoverUsers(ctx context.Context, userID uuid.UUID, params NetworkDiscoverParams) ([]NetworkDiscoverItem, int, error)
	// GetByID retorna una conexión por su ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Connection, error)
	// GetStatusBetween retorna el estado de conexión entre dos usuarios, o nil si no existe.
	GetStatusBetween(ctx context.Context, userA, userB uuid.UUID) (*domain.ConnectionStatus, error)
	// ExistsBetween verifica si existe una conexión entre dos usuarios
	ExistsBetween(ctx context.Context, userA, userB uuid.UUID) (bool, error)
	// ReopenRejectedByUsers reabre una conexión rechazada entre dos usuarios y la deja en pending.
	ReopenRejectedByUsers(ctx context.Context, requesterID, addresseeID uuid.UUID) (bool, error)
}
