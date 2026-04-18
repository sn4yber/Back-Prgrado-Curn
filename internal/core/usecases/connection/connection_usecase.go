package connection

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
)

// ConnectionUseCase implementa input.ConnectionUseCase.
type ConnectionUseCase struct {
	repo output.ConnectionRepository
}

func NewConnectionUseCase(repo output.ConnectionRepository) *ConnectionUseCase {
	return &ConnectionUseCase{repo: repo}
}

// RequestConnection permite solicitar una conexión entre dos usuarios.
func (uc *ConnectionUseCase) RequestConnection(ctx context.Context, requesterID, addresseeID uuid.UUID) error {
	if requesterID == addresseeID {
		return domain.ErrSelfConnection
	}

	// Si había una relación rechazada entre ambos, se reactiva como pending.
	reopened, err := uc.repo.ReopenRejectedByUsers(ctx, requesterID, addresseeID)
	if err != nil {
		return err
	}
	if reopened {
		return nil
	}

	// Verifica si ya existe una conexión
	exists, err := uc.repo.ExistsBetween(ctx, requesterID, addresseeID)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrConnectionAlreadyExists
	}
	conn, err := domain.NewConnection(requesterID, addresseeID)
	if err != nil {
		return err
	}
	return uc.repo.Create(ctx, conn)
}

// AcceptConnection mueve la conexión a estado aceptado.
func (uc *ConnectionUseCase) AcceptConnection(ctx context.Context, connID, addresseeID uuid.UUID) error {
	conn, err := uc.repo.GetByID(ctx, connID)
	if err != nil {
		return err
	}
	if conn.AddresseeID != addresseeID {
		return ErrAccionNoPermitida
	}
	if err := conn.Accept(); err != nil {
		return err
	}
	return uc.repo.UpdateStatus(ctx, connID, conn.Status)
}

// RejectConnection mueve la conexión a estado rechazado.
func (uc *ConnectionUseCase) RejectConnection(ctx context.Context, connID, addresseeID uuid.UUID) error {
	conn, err := uc.repo.GetByID(ctx, connID)
	if err != nil {
		return err
	}
	if conn.AddresseeID != addresseeID {
		return ErrAccionNoPermitida
	}
	if err := conn.Reject(); err != nil {
		return err
	}
	return uc.repo.UpdateStatus(ctx, connID, conn.Status)
}

// BlockConnection mueve la conexión a estado bloqueado.
func (uc *ConnectionUseCase) BlockConnection(ctx context.Context, connID, requesterID uuid.UUID) error {
	conn, err := uc.repo.GetByID(ctx, connID)
	if err != nil {
		return err
	}
	if conn.RequesterID != requesterID && conn.AddresseeID != requesterID {
		return ErrAccionNoPermitida
	}
	if err := conn.Block(); err != nil {
		return err
	}
	return uc.repo.UpdateStatus(ctx, connID, conn.Status)
}

// ListConnections retorna todas las conexiones de un usuario.
func (uc *ConnectionUseCase) ListConnections(ctx context.Context, userID uuid.UUID, status string, direction string) ([]*domain.Connection, error) {
	parsedStatus, err := parseConnectionStatus(status)
	if err != nil {
		return nil, err
	}

	parsedDirection, err := parseDirection(direction)
	if err != nil {
		return nil, err
	}

	return uc.repo.ListByUserFiltered(ctx, userID, parsedStatus, parsedDirection)
}

func (uc *ConnectionUseCase) ListSuggestions(ctx context.Context, userID uuid.UUID, limit int) ([]input.ConnectionSuggestionResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	items, err := uc.repo.ListSuggestions(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	resp := make([]input.ConnectionSuggestionResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, input.ConnectionSuggestionResponse{
			ID:          item.UserID.String(),
			Name:        item.Name,
			Role:        item.Role,
			ProgramName: item.ProgramName,
			AvatarURL:   item.AvatarURL,
		})
	}

	return resp, nil
}

func (uc *ConnectionUseCase) DiscoverUsers(ctx context.Context, userID uuid.UUID, req input.NetworkDiscoverRequest) (*input.NetworkDiscoverResponse, error) {
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 50 {
		req.Limit = 50
	}
	if req.Offset < 0 {
		return nil, ErrFiltroInvalido
	}

	items, total, err := uc.repo.DiscoverUsers(ctx, userID, output.NetworkDiscoverParams{
		Query:   strings.TrimSpace(req.Query),
		Role:    strings.TrimSpace(req.Role),
		Program: strings.TrimSpace(req.Program),
		City:    strings.TrimSpace(req.City),
		Limit:   req.Limit,
		Offset:  req.Offset,
	})
	if err != nil {
		return nil, err
	}

	respItems := make([]input.NetworkDiscoverItemResponse, 0, len(items))
	for _, item := range items {
		respItems = append(respItems, input.NetworkDiscoverItemResponse{
			ID:                    item.UserID.String(),
			Name:                  item.Name,
			Role:                  item.Role,
			ProgramName:           item.ProgramName,
			AvatarURL:             item.AvatarURL,
			IsConnected:           item.IsConnected,
			PendingRequest:        item.PendingRequest,
			RecommendationScore:   item.RecommendationScore,
			RecommendationReasons: item.RecommendationReasons,
		})
	}

	return &input.NetworkDiscoverResponse{
		Items:  respItems,
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
	}, nil
}

// GetConnection retorna una conexión por su ID.
func (uc *ConnectionUseCase) GetConnection(ctx context.Context, connID uuid.UUID) (*domain.Connection, error) {
	return uc.repo.GetByID(ctx, connID)
}

var (
	ErrAccionNoPermitida = errors.New("acción no permitida para este usuario")
	ErrFiltroInvalido    = errors.New("filtro inválido")
)

func parseConnectionStatus(raw string) (*domain.ConnectionStatus, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return nil, nil
	}

	status := domain.ConnectionStatus(raw)
	switch status {
	case domain.ConnectionPending, domain.ConnectionAccepted, domain.ConnectionRejected, domain.ConnectionBlocked:
		return &status, nil
	default:
		return nil, ErrFiltroInvalido
	}
}

func parseDirection(raw string) (string, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return "all", nil
	}

	switch raw {
	case "all", "incoming", "outgoing":
		return raw, nil
	default:
		return "", ErrFiltroInvalido
	}
}

