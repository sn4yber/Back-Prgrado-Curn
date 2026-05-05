package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
)

type ActividadHandler struct {
	usecase input.ActividadUseCase
}

func NewActividadHandler(usecase input.ActividadUseCase) *ActividadHandler {
	return &ActividadHandler{usecase: usecase}
}

func (h *ActividadHandler) RegisterRoutes(rg *gin.RouterGroup) {
	actividades := rg.Group("/actividades")
	{
		actividades.POST("", h.Crear)
		actividades.GET("", h.Listar)
		actividades.GET("/mis-inscripciones", h.MisInscripciones)
		actividades.GET("/:id", h.ObtenerPorID)
		actividades.PUT("/:id", h.Actualizar)
		actividades.DELETE("/:id", h.Eliminar)
		actividades.POST("/:id/inscribirse", h.Inscribirse)
		actividades.DELETE("/:id/inscribirse", h.CancelarInscripcion)
		actividades.GET("/:id/inscritos", h.ListarInscritos)
	}
}

func (h *ActividadHandler) Crear(c *gin.Context) {
	usuarioID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	var req input.CrearActividadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}
	resp, err := h.usecase.Crear(c.Request.Context(), usuarioID, req)
	if err != nil {
		writeActividadError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *ActividadHandler) Listar(c *gin.Context) {
	tipo := c.Query("tipo")
	estado := c.Query("estado")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	resp, err := h.usecase.Listar(c.Request.Context(), tipo, estado, limit, offset)
	if err != nil {
		writeActividadError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ActividadHandler) ObtenerPorID(c *gin.Context) {
	actividadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de actividad inválido"})
		return
	}
	resp, err := h.usecase.ObtenerPorID(c.Request.Context(), actividadID)
	if err != nil {
		writeActividadError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ActividadHandler) Actualizar(c *gin.Context) {
	usuarioID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	actividadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de actividad inválido"})
		return
	}
	var req input.ActualizarActividadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}
	resp, err := h.usecase.Actualizar(c.Request.Context(), actividadID, usuarioID, req)
	if err != nil {
		writeActividadError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ActividadHandler) Eliminar(c *gin.Context) {
	usuarioID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	actividadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de actividad inválido"})
		return
	}
	if err := h.usecase.Eliminar(c.Request.Context(), actividadID, usuarioID); err != nil {
		writeActividadError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *ActividadHandler) Inscribirse(c *gin.Context) {
	usuarioID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	actividadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de actividad inválido"})
		return
	}
	var req input.InscribirseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}
	resp, err := h.usecase.Inscribirse(c.Request.Context(), actividadID, usuarioID, req)
	if err != nil {
		writeActividadError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *ActividadHandler) CancelarInscripcion(c *gin.Context) {
	usuarioID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	actividadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de actividad inválido"})
		return
	}
	if err := h.usecase.CancelarInscripcion(c.Request.Context(), actividadID, usuarioID); err != nil {
		writeActividadError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (h *ActividadHandler) MisInscripciones(c *gin.Context) {
	usuarioID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	resp, err := h.usecase.ObtenerMisInscripciones(c.Request.Context(), usuarioID)
	if err != nil {
		writeActividadError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ActividadHandler) ListarInscritos(c *gin.Context) {
	usuarioID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	actividadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de actividad inválido"})
		return
	}
	resp, err := h.usecase.ListarInscritos(c.Request.Context(), actividadID, usuarioID)
	if err != nil {
		writeActividadError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func writeActividadError(c *gin.Context, err error) {
	appErr := apperrors.AsAppError(err)
	c.JSON(appErr.Code, gin.H{"error": appErr.Message})
}
