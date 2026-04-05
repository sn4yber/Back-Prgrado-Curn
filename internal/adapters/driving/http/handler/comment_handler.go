package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
	"net/http"
)

type CommentHandler struct {
	usecase input.CommentUseCase
}

func NewCommentHandler(usecase input.CommentUseCase) *CommentHandler {
	return &CommentHandler{usecase: usecase}
}
func (h *CommentHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/posts/:id/comments", h.Create)
	rg.GET("/posts/:id/comments", h.ListByPost)
}
func (h *CommentHandler) Create(c *gin.Context) {
	authorID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de publicación inválido"})
		return
	}
	var req input.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}
	resp, err := h.usecase.Create(c.Request.Context(), authorID, postID, req)
	if err != nil {
		writeCommentError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}
func (h *CommentHandler) ListByPost(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de publicación inválido"})
		return
	}
	resp, err := h.usecase.ListByPost(c.Request.Context(), postID)
	if err != nil {
		writeCommentError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
func writeCommentError(c *gin.Context, err error) {
	appErr := apperrors.AsAppError(err)
	c.JSON(appErr.Code, gin.H{"error": appErr.Message})
}
