package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ConnectionStatus representa el estado de una conexión entre dos usuarios
type ConnectionStatus string

const (
	ConnectionPending  ConnectionStatus = "pending"
	ConnectionAccepted ConnectionStatus = "accepted"
	ConnectionRejected ConnectionStatus = "rejected"
	ConnectionBlocked  ConnectionStatus = "blocked"
)

// Errores de dominio
var (
	ErrSelfConnection          = errors.New("no puedes conectar contigo mismo")
	ErrInvalidID               = errors.New("ID de usuario inválido")
	ErrInvalidTransition       = errors.New("transición de estado inválida")
	ErrNotPending              = errors.New("la conexión no está en estado pendiente")
	ErrAlreadyBlocked          = errors.New("la conexión ya está bloqueada")
	ErrConnectionAlreadyExists = errors.New("ya existe una conexión entre estos usuarios")
)

// Connection representa una solicitud de relación entre dos usuarios
type Connection struct {
	ID          uuid.UUID
	RequesterID uuid.UUID
	AddresseeID uuid.UUID
	Status      ConnectionStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewConnection crea una nueva conexión en estado pendiente con validaciones básicas
func NewConnection(requesterID, addresseeID uuid.UUID) (*Connection, error) {
	if requesterID == uuid.Nil || addresseeID == uuid.Nil {
		return nil, ErrInvalidID
	}
	if requesterID == addresseeID {
		return nil, ErrSelfConnection
	}

	now := time.Now()
	return &Connection{
		ID:          uuid.New(),
		RequesterID: requesterID,
		AddresseeID: addresseeID,
		Status:      ConnectionPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Reconstitute reconstruye una Connection desde almacenamiento persistente sin validaciones
func Reconstitute(
	id, requesterID, addresseeID uuid.UUID,
	status ConnectionStatus,
	createdAt, updatedAt time.Time,
) *Connection {
	return &Connection{
		ID:          id,
		RequesterID: requesterID,
		AddresseeID: addresseeID,
		Status:      status,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

// --- Transiciones de estado ---

// Accept mueve la conexión de pendiente a aceptada
func (c *Connection) Accept() error {
	if c.Status != ConnectionPending {
		return ErrNotPending
	}
	c.Status = ConnectionAccepted
	c.UpdatedAt = time.Now()
	return nil
}

// Reject mueve la conexión de pendiente a rechazada
func (c *Connection) Reject() error {
	if c.Status != ConnectionPending {
		return ErrNotPending
	}
	c.Status = ConnectionRejected
	c.UpdatedAt = time.Now()
	return nil
}

// Block puede aplicarse desde cualquier estado excepto cuando ya está bloqueada
func (c *Connection) Block() error {
	if c.Status == ConnectionBlocked {
		return ErrAlreadyBlocked
	}
	c.Status = ConnectionBlocked
	c.UpdatedAt = time.Now()
	return nil
}

// --- Consultas ---

// IsPending retorna true si la conexión aún espera una respuesta
func (c *Connection) IsPending() bool {
	return c.Status == ConnectionPending
}

// IsActive retorna true si la conexión fue aceptada
func (c *Connection) IsActive() bool {
	return c.Status == ConnectionAccepted
}

// IsBlocked retorna true si alguno de los usuarios bloqueó al otro
func (c *Connection) IsBlocked() bool {
	return c.Status == ConnectionBlocked
}

// IsBetween retorna true si la conexión involucra a ambos usuarios, sin importar la dirección
func (c *Connection) IsBetween(userA, userB uuid.UUID) bool {
	return (c.RequesterID == userA && c.AddresseeID == userB) ||
		(c.RequesterID == userB && c.AddresseeID == userA)
}

// InvolvesUser retorna true si el usuario dado forma parte de esta conexión
func (c *Connection) InvolvesUser(userID uuid.UUID) bool {
	return c.RequesterID == userID || c.AddresseeID == userID
}
