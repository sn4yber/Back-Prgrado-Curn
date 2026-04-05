package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
	"net/http"
)

type ProjectHandler struct {
	usecase input.ProjectUseCase
}

func NewProjectHandler(usecase input.ProjectUseCase) *ProjectHandler {
	return &ProjectHandler{usecase: usecase}
}
func (h *ProjectHandler) RegisterRoutes(rg *gin.RouterGroup) {
	projects := rg.Group("/projects")
	{
		projects.POST("", h.Create)
		projects.GET("/mine", h.ListMine)
		projects.GET("/:id", h.GetByID)
		projects.PATCH("/:id/status", h.UpdateStatus)
	}
}
func (h *ProjectHandler) Create(c *gin.Context) {
	ownerID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	var req input.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}
	resp, err := h.usecase.Create(c.Request.Context(), ownerID, req)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}
func (h *ProjectHandler) ListMine(c *gin.Context) {
	ownerID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	resp, err := h.usecase.ListMine(c.Request.Context(), ownerID)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
func (h *ProjectHandler) GetByID(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de proyecto inválido"})
		return
	}
	resp, err := h.usecase.GetByID(c.Request.Context(), requesterID, projectID)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
func (h *ProjectHandler) UpdateStatus(c *gin.Context) {
	ownerID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de proyecto inválido"})
		return
	}
	var req input.UpdateProjectStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}
	resp, err := h.usecase.UpdateStatus(c.Request.Context(), ownerID, projectID, req)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
func writeProjectError(c *gin.Context, err error) {
	appErr := apperrors.AsAppError(err)
	c.JSON(appErr.Code, gin.H{"error": appErr.Message})
}
