package input

import (
	"context"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
)

type ConnectionSuggestionResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Role        string  `json:"role"`
	ProgramName *string `json:"program_name"`
	AvatarURL   *string `json:"avatar_url"`
}

type NetworkDiscoverRequest struct {
	Query   string
	Role    string
	Program string
	City    string
	Limit   int
	Offset  int
}

type NetworkDiscoverItemResponse struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Role                  string   `json:"role"`
	ProgramName           *string  `json:"program_name"`
	AvatarURL             *string  `json:"avatar_url"`
	IsConnected           bool     `json:"is_connected"`
	PendingRequest        bool     `json:"pending_request"`
	RecommendationScore   float64  `json:"recommendation_score"`
	RecommendationReasons []string `json:"recommendation_reasons"`
}

type NetworkDiscoverResponse struct {
	Items  []NetworkDiscoverItemResponse `json:"items"`
	Total  int                           `json:"total"`
	Limit  int                           `json:"limit"`
	Offset int                           `json:"offset"`
}

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
	// ListConnections retorna conexiones de un usuario, con filtros opcionales.
	ListConnections(ctx context.Context, userID uuid.UUID, status string, direction string) ([]*domain.Connection, error)
	// ListSuggestions retorna usuarios sugeridos para conexión.
	ListSuggestions(ctx context.Context, userID uuid.UUID, limit int) ([]ConnectionSuggestionResponse, error)
	// DiscoverUsers lista usuarios con filtros y score de recomendación.
	DiscoverUsers(ctx context.Context, userID uuid.UUID, req NetworkDiscoverRequest) (*NetworkDiscoverResponse, error)
	// GetConnection retorna una conexión por su ID.
	GetConnection(ctx context.Context, connID uuid.UUID) (*domain.Connection, error)
}
