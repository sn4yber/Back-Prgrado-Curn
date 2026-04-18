package handler

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
)

type PostHandler struct {
	usecase input.PostUseCase
}

func NewPostHandler(usecase input.PostUseCase) *PostHandler {
	return &PostHandler{usecase: usecase}
}

func (h *PostHandler) RegisterRoutes(rg *gin.RouterGroup) {
	posts := rg.Group("/posts")
	{
		posts.POST("", h.CreatePost)
		posts.PUT("/:id", h.UpdatePost)
		posts.DELETE("/:id", h.DeletePost)
		posts.GET("/user/:id", h.ListPostsByUser)
		posts.GET("/mine", h.ListMyPosts)
		posts.GET("/feed", h.ListFeedPosts)
		posts.GET("/public", h.ListPublicPosts)
		posts.POST("/:id/reactions", h.ReactToPost)
		posts.POST("/:id/reports", h.ReportPost)
		posts.GET("/pending-review", h.ListPendingReview)
		posts.PATCH("/:id/moderate", h.ModeratePost)
	}
}

func (h *PostHandler) UpdatePost(c *gin.Context) {
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

	var req input.UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}

	resp, err := h.usecase.UpdatePost(c.Request.Context(), requesterID, postID, req)
	if err != nil {
		writeAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) DeletePost(c *gin.Context) {
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

	if err := h.usecase.DeletePost(c.Request.Context(), requesterID, postID); err != nil {
		writeAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "publicación eliminada"})
}

func (h *PostHandler) CreatePost(c *gin.Context) {
	authorID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	var req input.CreatePostRequest
	req.Title = c.PostForm("title")
	req.Description = c.PostForm("description")
	req.Category = c.PostForm("category")
	req.IsJobOffer = parseBool(c.PostForm("is_job_offer"))
	req.Faculty = c.PostForm("faculty")
	req.AcademicProgram = c.PostForm("academic_program")
	req.Advisor = c.PostForm("advisor")
	req.DeclaredAuthorID = c.PostForm("declared_author_id")
	req.CoAuthorIDs = parseCSV(c.PostForm("coauthor_ids"))
	req.OriginalityDeclaration = parseBool(c.PostForm("originality_declaration"))
	req.PrivacyConsent = parseBool(c.PostForm("privacy_consent"))
	req.IsInstitutional = parseBool(c.PostForm("is_institutional"))
	req.VerifiedByFaculty = parseBool(c.PostForm("verified_by_faculty"))

	form, err := c.MultipartForm()
	if err != nil && !errors.Is(err, http.ErrNotMultipart) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "form-data inválido"})
		return
	}
	if form != nil {
		files := form.File["attachments"]
		req.Attachments = make([]input.AttachmentUpload, 0, len(files))
		for _, fh := range files {
			f, err := fh.Open()
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "archivo inválido"})
				return
			}
			data, readErr := io.ReadAll(f)
			_ = f.Close()
			if readErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "no se pudo leer archivo"})
				return
			}
			req.Attachments = append(req.Attachments, input.AttachmentUpload{
				FileName:    fh.Filename,
				ContentType: fh.Header.Get("Content-Type"),
				Data:        data,
			})
		}
	}

	resp, err := h.usecase.CreatePost(c.Request.Context(), authorID, req)
	if err != nil {
		writeAppError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *PostHandler) ListMyPosts(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	resp, err := h.usecase.ListMyPosts(c.Request.Context(), requesterID)
	if err != nil {
		writeAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) ListPostsByUser(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de usuario inválido"})
		return
	}

	resp, err := h.usecase.ListPostsByUser(c.Request.Context(), requesterID, userID)
	if err != nil {
		writeAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) ListPublicPosts(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	resp, err := h.usecase.ListPublicPosts(c.Request.Context(), requesterID)
	if err != nil {
		writeAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) ListFeedPosts(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	resp, err := h.usecase.ListFeedPosts(c.Request.Context(), requesterID)
	if err != nil {
		writeAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) ReactToPost(c *gin.Context) {
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

	var req input.ReactToPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}

	resp, err := h.usecase.ReactToPost(c.Request.Context(), requesterID, postID, req)
	if err != nil {
		writeAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) ReportPost(c *gin.Context) {
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

	var req input.ReportPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}

	resp, err := h.usecase.ReportPost(c.Request.Context(), requesterID, postID, req)
	if err != nil {
		writeAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) ListPendingReview(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	resp, err := h.usecase.ListPendingReview(c.Request.Context(), requesterID)
	if err != nil {
		writeAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) ModeratePost(c *gin.Context) {
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

	var req input.ModeratePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos de entrada inválidos"})
		return
	}

	resp, err := h.usecase.ModeratePost(c.Request.Context(), requesterID, postID, req)
	if err != nil {
		writeAppError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func parseBool(v string) bool {
	return strings.EqualFold(strings.TrimSpace(v), "true") || strings.TrimSpace(v) == "1"
}

func parseCSV(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func writeAppError(c *gin.Context, err error) {
	appErr := apperrors.AsAppError(err)
	log.Printf("HTTP_ERROR method=%s path=%s status=%d message=%q err=%v", c.Request.Method, c.Request.URL.Path, appErr.Code, appErr.Message, err)
	c.JSON(appErr.Code, gin.H{"error": appErr.Message})
}
