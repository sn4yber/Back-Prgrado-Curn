package connection

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/output"
)

// ConnectionUsecase implementa input.ConnectionUseCase.
type ConnectionUsecase struct {
	repo output.ConnectionRepository
}

func NewConnectionUsecase(repo output.ConnectionRepository) *ConnectionUsecase {
	return &ConnectionUsecase{repo: repo}
}

// RequestConnection permite solicitar una conexión entre dos usuarios.
func (uc *ConnectionUsecase) RequestConnection(ctx context.Context, requesterID, addresseeID uuid.UUID) error {
	if requesterID == addresseeID {
		return domain.ErrSelfConnection
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
func (uc *ConnectionUsecase) AcceptConnection(ctx context.Context, connID, addresseeID uuid.UUID) error {
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
func (uc *ConnectionUsecase) RejectConnection(ctx context.Context, connID, addresseeID uuid.UUID) error {
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
func (uc *ConnectionUsecase) BlockConnection(ctx context.Context, connID, requesterID uuid.UUID) error {
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
func (uc *ConnectionUsecase) ListConnections(ctx context.Context, userID uuid.UUID) ([]*domain.Connection, error) {
	return uc.repo.ListByUser(ctx, userID)
}

// GetConnection retorna una conexión por su ID.
func (uc *ConnectionUsecase) GetConnection(ctx context.Context, connID uuid.UUID) (*domain.Connection, error) {
	return uc.repo.GetByID(ctx, connID)
}

var (
	ErrAccionNoPermitida = errors.New("acción no permitida para este usuario")
)
