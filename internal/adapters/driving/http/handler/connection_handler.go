package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/adapters/driving/http/middleware"
	"github.com/sn4yber/curn-networking/internal/core/domain"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	"github.com/sn4yber/curn-networking/internal/core/usecases/connection"
)

// ConnectionHandler expone los endpoints HTTP para conexiones entre usuarios.
// Depende de la interfaz input.ConnectionUseCase, no de la implementación concreta.
type ConnectionHandler struct {
	uc input.ConnectionUseCase
}

func NewConnectionHandler(uc input.ConnectionUseCase) *ConnectionHandler {
	return &ConnectionHandler{uc: uc}
}

// --- DTOs ---

type requestConnectionDTO struct {
	AddresseeID string `json:"addressee_id" binding:"required,uuid"`
}

type connectionResponse struct {
	ID          string `json:"id"`
	RequesterID string `json:"requester_id"`
	AddresseeID string `json:"addressee_id"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// --- Endpoints ---

// POST /api/v1/connections/request
// El requesterID se obtiene del JWT (inyectado por el middleware AuthRequired).
func (h *ConnectionHandler) RequestConnection(c *gin.Context) {
	requesterID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse("usuario no autenticado"))
		return
	}

	var req requestConnectionDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("datos inválidos: "+err.Error()))
		return
	}

	addresseeID, err := uuid.Parse(req.AddresseeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("addressee_id inválido"))
		return
	}

	if err := h.uc.RequestConnection(c.Request.Context(), requesterID, addresseeID); err != nil {
		c.JSON(mapDomainError(err), errorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"mensaje": "solicitud enviada correctamente"})
}

// POST /api/v1/connections/:id/accept
// El addresseeID se obtiene del JWT.
func (h *ConnectionHandler) AcceptConnection(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse("usuario no autenticado"))
		return
	}

	connID, err := parseUUIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("ID de conexión inválido"))
		return
	}

	if err := h.uc.AcceptConnection(c.Request.Context(), connID, userID); err != nil {
		c.JSON(mapDomainError(err), errorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "conexión aceptada"})
}

// POST /api/v1/connections/:id/reject
// El addresseeID se obtiene del JWT.
func (h *ConnectionHandler) RejectConnection(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse("usuario no autenticado"))
		return
	}

	connID, err := parseUUIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("ID de conexión inválido"))
		return
	}

	if err := h.uc.RejectConnection(c.Request.Context(), connID, userID); err != nil {
		c.JSON(mapDomainError(err), errorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "conexión rechazada"})
}

// POST /api/v1/connections/:id/block
// El requesterID (quien bloquea) se obtiene del JWT.
func (h *ConnectionHandler) BlockConnection(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse("usuario no autenticado"))
		return
	}

	connID, err := parseUUIDParam(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("ID de conexión inválido"))
		return
	}

	if err := h.uc.BlockConnection(c.Request.Context(), connID, userID); err != nil {
		c.JSON(mapDomainError(err), errorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "conexión bloqueada"})
}

// GET /api/v1/connections
// Retorna las conexiones del usuario autenticado.
func (h *ConnectionHandler) ListConnections(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorResponse("usuario no autenticado"))
		return
	}

	conns, err := h.uc.ListConnections(c.Request.Context(), userID)
	if err != nil {
		c.JSON(mapDomainError(err), errorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, toConnectionResponseList(conns))
}

// --- helpers privados ---

// getUserIDFromContext extrae el userID inyectado por el middleware AuthRequired.
func getUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	raw, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		return uuid.Nil, errors.New("userID no encontrado en contexto")
	}
	return uuid.Parse(raw.(string))
}

func parseUUIDParam(c *gin.Context, param string) (uuid.UUID, error) {
	return uuid.Parse(c.Param(param))
}

func errorResponse(msg string) gin.H {
	return gin.H{"error": msg}
}

// mapDomainError traduce errores de dominio a códigos HTTP apropiados
func mapDomainError(err error) int {
	switch {
	case errors.Is(err, domain.ErrSelfConnection),
		errors.Is(err, domain.ErrInvalidID),
		errors.Is(err, domain.ErrInvalidTransition),
		errors.Is(err, domain.ErrNotPending),
		errors.Is(err, domain.ErrAlreadyBlocked),
		errors.Is(err, domain.ErrConnectionAlreadyExists):
		return http.StatusUnprocessableEntity

	case errors.Is(err, connection.ErrAccionNoPermitida):
		return http.StatusForbidden

	default:
		return http.StatusInternalServerError
	}
}

func toConnectionResponse(conn *domain.Connection) connectionResponse {
	return connectionResponse{
		ID:          conn.ID.String(),
		RequesterID: conn.RequesterID.String(),
		AddresseeID: conn.AddresseeID.String(),
		Status:      string(conn.Status),
		CreatedAt:   conn.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   conn.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toConnectionResponseList(conns []*domain.Connection) []connectionResponse {
	result := make([]connectionResponse, 0, len(conns))
	for _, conn := range conns {
		result = append(result, toConnectionResponse(conn))
	}
	return result
}
