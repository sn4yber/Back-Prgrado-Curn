package handler

import (
	"io"
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
		posts.GET("/mine", h.ListMyPosts)
		posts.GET("/public", h.ListPublicPosts)
		posts.GET("/pending-review", h.ListPendingReview)
		posts.PATCH("/:id/moderate", h.ModeratePost)
	}
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
	req.DeclaredAuthorID = c.PostForm("declared_author_id")
	req.CoAuthorIDs = parseCSV(c.PostForm("coauthor_ids"))
	req.OriginalityDeclaration = parseBool(c.PostForm("originality_declaration"))
	req.PrivacyConsent = parseBool(c.PostForm("privacy_consent"))
	req.IsInstitutional = parseBool(c.PostForm("is_institutional"))
	req.VerifiedByFaculty = parseBool(c.PostForm("verified_by_faculty"))

	form, err := c.MultipartForm()
	if err == nil && form != nil {
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
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
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
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *PostHandler) ListPublicPosts(c *gin.Context) {
	resp, err := h.usecase.ListPublicPosts(c.Request.Context())
	if err != nil {
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
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
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
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
		appErr := apperrors.AsAppError(err)
		c.JSON(appErr.Code, gin.H{"error": appErr.Message})
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
