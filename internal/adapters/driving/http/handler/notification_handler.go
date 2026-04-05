package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
	"net/http"
	"strconv"
)

type NotificationHandler struct {
	usecase input.NotificationUseCase
}

func NewNotificationHandler(usecase input.NotificationUseCase) *NotificationHandler {
	return &NotificationHandler{usecase: usecase}
}
func (h *NotificationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	notifications := rg.Group("/notifications")
	{
		notifications.GET("", h.ListMine)
		notifications.PATCH("/:id/read", h.MarkAsRead)
	}
}
func (h *NotificationHandler) ListMine(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	resp, err := h.usecase.ListMine(c.Request.Context(), userID, limit, offset)
	if err != nil {
		writeNotificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de notificación inválido"})
		return
	}
	if err := h.usecase.MarkAsRead(c.Request.Context(), userID, notificationID); err != nil {
		writeNotificationError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "notificación marcada como leída"})
}
func writeNotificationError(c *gin.Context, err error) {
	appErr := apperrors.AsAppError(err)
	c.JSON(appErr.Code, gin.H{"error": appErr.Message})
}
