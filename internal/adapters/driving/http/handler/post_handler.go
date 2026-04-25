package handler

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sn4yber/curn-networking/internal/core/ports/input"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
)

type PostHandler struct {
	usecase input.PostUseCase
}

const (
	maxCreatePostBodyBytes = 100 << 20 // 100 MiB
	maxCreatePostFormMem   = 16 << 20  // 16 MiB
	maxAttachmentsPerPost  = 10
	maxAttachmentBytes     = 20 << 20 // 20 MiB por archivo
)

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

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxCreatePostBodyBytes)

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

	err = c.Request.ParseMultipartForm(maxCreatePostFormMem)
	if err != nil && !errors.Is(err, http.ErrNotMultipart) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "form-data inválido"})
		return
	}

	form := c.Request.MultipartForm
	if form != nil {
		files := form.File["attachments"]
		if len(files) > maxAttachmentsPerPost {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("máximo %d adjuntos por publicación", maxAttachmentsPerPost)})
			return
		}

		req.Attachments = make([]input.AttachmentUpload, 0, len(files))
		var totalBytes int64
		for _, fh := range files {
			upload, readErr := readAttachmentUpload(fh)
			if readErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "archivo inválido"})
				return
			}

			totalBytes += int64(len(upload.Data))
			if totalBytes > maxCreatePostBodyBytes {
				c.JSON(http.StatusBadRequest, gin.H{"error": "el total de adjuntos excede el límite permitido"})
				return
			}

			req.Attachments = append(req.Attachments, upload)
		}
	}

	resp, err := h.usecase.CreatePost(c.Request.Context(), authorID, req)
	if err != nil {
		writeAppError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func readAttachmentUpload(fh *multipart.FileHeader) (input.AttachmentUpload, error) {
	if fh == nil {
		return input.AttachmentUpload{}, fmt.Errorf("archivo vacío")
	}
	if fh.Size > maxAttachmentBytes {
		return input.AttachmentUpload{}, fmt.Errorf("archivo supera tamaño máximo")
	}

	f, err := fh.Open()
	if err != nil {
		return input.AttachmentUpload{}, err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxAttachmentBytes+1))
	if err != nil {
		return input.AttachmentUpload{}, err
	}
	if int64(len(data)) > maxAttachmentBytes {
		return input.AttachmentUpload{}, fmt.Errorf("archivo supera tamaño máximo")
	}

	contentType := fh.Header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		contentType = http.DetectContentType(data)
	}

	return input.AttachmentUpload{
		FileName:    fh.Filename,
		ContentType: contentType,
		Data:        data,
	}, nil
}

func (h *PostHandler) ListMyPosts(c *gin.Context) {
	requesterID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
		return
	}

	limit, offset, err := parsePaginationParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.usecase.ListMyPosts(c.Request.Context(), requesterID, limit, offset)
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

	limit, offset, err := parsePaginationParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.usecase.ListPostsByUser(c.Request.Context(), requesterID, userID, limit, offset)
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

	limit, offset, err := parsePaginationParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.usecase.ListPublicPosts(c.Request.Context(), requesterID, limit, offset)
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

	limit, offset, err := parsePaginationParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.usecase.ListFeedPosts(c.Request.Context(), requesterID, limit, offset)
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

func parsePaginationParams(c *gin.Context) (int, int, error) {
	limit := 20
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			return 0, 0, errors.New("limit inválido")
		}
		limit = parsed
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	offset := 0
	if rawOffset := c.Query("offset"); rawOffset != "" {
		parsed, err := strconv.Atoi(rawOffset)
		if err != nil {
			return 0, 0, errors.New("offset inválido")
		}
		offset = parsed
	}
	if offset < 0 {
		offset = 0
	}

	return limit, offset, nil
}
