package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
	"net/http"
)

type MentorshipHandler struct {
	usecase input.MentorshipUseCase
}

func NewMentorshipHandler(usecase input.MentorshipUseCase) *MentorshipHandler {
	return &MentorshipHandler{usecase: usecase}
}
func (h *MentorshipHandler) RegisterRoutes(rg *gin.RouterGroup) {
	mentorships := rg.Group("/mentorships")
	{
		mentorships.POST("/request", h.Request)
		mentorships.POST("/:id/accept", h.Accept)
		mentorships.POST("/:id/reject", h.Reject)
		mentorships.GET("/mine", h.ListMine)
	}
}
func (h *MentorshipHandler) Request(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	var req input.RequestMentorshipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}
	resp, err := h.usecase.Request(c.Request.Context(), requesterID, req)
	if err != nil {
		writeMentorshipError(c, err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}
func (h *MentorshipHandler) Accept(c *gin.Context) {
	mentorID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	mentorshipID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de mentoría inválido"})
		return
	}
	var req input.DecideMentorshipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}
	resp, err := h.usecase.Accept(c.Request.Context(), mentorID, mentorshipID, req)
	if err != nil {
		writeMentorshipError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
func (h *MentorshipHandler) Reject(c *gin.Context) {
	mentorID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	mentorshipID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de mentoría inválido"})
		return
	}
	var req input.DecideMentorshipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}
	resp, err := h.usecase.Reject(c.Request.Context(), mentorID, mentorshipID, req)
	if err != nil {
		writeMentorshipError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
func (h *MentorshipHandler) ListMine(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}
	resp, err := h.usecase.ListMine(c.Request.Context(), userID)
	if err != nil {
		writeMentorshipError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
func writeMentorshipError(c *gin.Context, err error) {
	appErr := apperrors.AsAppError(err)
	c.JSON(appErr.Code, gin.H{"error": appErr.Message})
}
