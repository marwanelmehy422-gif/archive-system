package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"archive-system/internal/middleware"
	"archive-system/internal/services"
)

type FileHandler struct {
	fileService *services.FileService
}

func NewFileHandler(fileService *services.FileService) *FileHandler {
	return &FileHandler{fileService: fileService}
}

// POST /api/v1/transactions/:id/attachments
func (h *FileHandler) Upload(c *gin.Context) {
	transactionID := c.Param("id")
	orgID := middleware.GetCurrentOrgID(c)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "الملف مطلوب"})
		return
	}

	attachment, err := h.fileService.UploadFile(c.Request.Context(), transactionID, orgID, fileHeader)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrTransactionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "المعاملة غير موجودة"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "حصل خطأ أثناء رفع الملف"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"attachment":    attachment,
		"detected_type": attachment.DataType,
	})
}

// GET /api/v1/files/:attachment_id
func (h *FileHandler) Download(c *gin.Context) {
	attachmentID := c.Param("attachment_id")
	orgID := middleware.GetCurrentOrgID(c)

	storedPath, originalFilename, err := h.fileService.GetFilePath(c.Request.Context(), attachmentID, orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "الملف غير موجود"})
		return
	}

	c.FileAttachment(storedPath, originalFilename)
}
