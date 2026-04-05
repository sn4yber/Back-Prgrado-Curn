package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
)

type ModerationHandler struct {
	usecase input.ModerationUseCase
}

func NewModerationHandler(usecase input.ModerationUseCase) *ModerationHandler {
	return &ModerationHandler{usecase: usecase}
}

func (h *ModerationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	admin := rg.Group("/admin/moderation")
	{
		admin.GET("/posts", h.ListQueue)
		admin.PATCH("/posts/:id", h.ModeratePost)
		admin.GET("/upload-policies", h.GetUploadPolicies)
	}
}

func (h *ModerationHandler) ListQueue(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	statuses := splitQueryCSV(c.Query("statuses"))
	resp, err := h.usecase.ListQueue(c.Request.Context(), requesterID, statuses)
	if err != nil {
		writeModerationError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ModerationHandler) ModeratePost(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de publicación inválido"})
		return
	}

	var req input.ModeratePostAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload inválido"})
		return
	}

	if err := h.usecase.ModeratePost(c.Request.Context(), requesterID, postID, req); err != nil {
		writeModerationError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "moderación aplicada"})
}

func (h *ModerationHandler) GetUploadPolicies(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	resp, err := h.usecase.GetUploadPolicies(c.Request.Context(), requesterID)
	if err != nil {
		writeModerationError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func splitQueryCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func writeModerationError(c *gin.Context, err error) {
	appErr := apperrors.AsAppError(err)
	c.JSON(appErr.Code, gin.H{"error": appErr.Message})
}
