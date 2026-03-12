package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
)

type ConversationHandler struct {
	usecase input.ConversationUseCase
}

func NewConversationHandler(usecase input.ConversationUseCase) *ConversationHandler {
	return &ConversationHandler{usecase: usecase}
}

func (h *ConversationHandler) RegisterRoutes(rg *gin.RouterGroup) {
	conversations := rg.Group("/conversations")
	{
		conversations.POST("", h.StartConversation)
		conversations.GET("", h.ListMyConversations)
		conversations.GET("/:id", h.GetConversation)
		conversations.POST("/:id/messages", h.SendMessage)
		conversations.GET("/admin/flagged", h.AdminListFlagged)
	}
}

func (h *ConversationHandler) StartConversation(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	var req input.StartConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}
	if msg := validateStartConversationRequest(req); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	resp, err := h.usecase.StartConversation(c.Request.Context(), requesterID, req)
	if err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *ConversationHandler) SendMessage(c *gin.Context) {
	senderID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de conversación inválido"})
		return
	}

	var req input.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}
	if msg := validateSendMessageRequest(req); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	resp, err := h.usecase.SendMessage(c.Request.Context(), senderID, conversationID, req)
	if err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *ConversationHandler) GetConversation(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	conversationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de conversación inválido"})
		return
	}

	resp, err := h.usecase.GetConversation(c.Request.Context(), requesterID, conversationID)
	if err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ConversationHandler) ListMyConversations(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	resp, err := h.usecase.ListMyConversations(c.Request.Context(), requesterID)
	if err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ConversationHandler) AdminListFlagged(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	resp, err := h.usecase.AdminListFlagged(c.Request.Context(), requesterID)
	if err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func validateStartConversationRequest(req input.StartConversationRequest) string {
	if strings.TrimSpace(req.OtherUserID) == "" {
		return "other_user_id es requerido"
	}
	if strings.TrimSpace(req.SourceType) == "" {
		return "source_type es requerido"
	}
	if strings.TrimSpace(req.SourceID) == "" {
		return "source_id es requerido"
	}
	if len(strings.TrimSpace(req.FirstMessage)) > 2000 {
		return "first_message excede 2000 caracteres"
	}
	return ""
}

func validateSendMessageRequest(req input.SendMessageRequest) string {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return "content es requerido"
	}
	if len(content) > 2000 {
		return "content excede 2000 caracteres"
	}
	return ""
}
