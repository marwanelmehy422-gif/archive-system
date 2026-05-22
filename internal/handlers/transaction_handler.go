package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"archive-system/internal/middleware"
	"archive-system/internal/services"
)

type TransactionHandler struct {
	txService   *services.TransactionService
	fileService *services.FileService
}

func NewTransactionHandler(txService *services.TransactionService, fileService *services.FileService) *TransactionHandler {
	return &TransactionHandler{txService: txService, fileService: fileService}
}

// POST /api/v1/transactions
func (h *TransactionHandler) Create(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	orgID := middleware.GetCurrentOrgID(c)

	// اقرأ الـ form fields
	req := services.CreateTransactionRequest{
		Title:           c.PostForm("title"),
		Description:     c.PostForm("description"),
		ReceiverOrgCode: c.PostForm("receiver_org_code"),
		Priority:        c.PostForm("priority"),
	}

	if req.Title == "" || req.ReceiverOrgCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title و receiver_org_code مطلوبين"})
		return
	}

	// عمل الـ transaction
	tx, err := h.txService.Create(c.Request.Context(), &req, userID, orgID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrOrgNotFound):
			c.JSON(http.StatusBadRequest, gin.H{"error": "الجهة المستقبلة غير موجودة"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "حصل خطأ أثناء إرسال المعاملة"})
		}
		return
	}

	// رفع الملفات لو موجودة
	form, _ := c.MultipartForm()
	var attachments []interface{}

	if form != nil && form.File["files"] != nil {
		for _, fileHeader := range form.File["files"] {
			attachment, err := h.fileService.UploadFile(c.Request.Context(), tx.ID, orgID, fileHeader)
			if err != nil {
				// لو ملف واحد فشل، مش هنوقف الباقي
				attachments = append(attachments, gin.H{
					"filename": fileHeader.Filename,
					"error":    "فشل رفع الملف",
				})
				continue
			}
			attachments = append(attachments, attachment)
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"transaction": tx,
		"attachments": attachments,
	})
}

// GET /api/v1/transactions
func (h *TransactionHandler) GetAll(c *gin.Context) {
	orgID := middleware.GetCurrentOrgID(c)

	txs, err := h.txService.GetAll(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "حصل خطأ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": txs})
}

// GET /api/v1/transactions/:id
func (h *TransactionHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	orgID := middleware.GetCurrentOrgID(c)

	tx, attachments, history, err := h.txService.GetByID(c.Request.Context(), id, orgID)
	if err != nil {
		if errors.Is(err, services.ErrTransactionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "المعاملة غير موجودة"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "حصل خطأ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction": tx,
		"attachments": attachments,
		"history":     history,
	})
}

// PUT /api/v1/transactions/:id/accept
func (h *TransactionHandler) Accept(c *gin.Context) {
	id := c.Param("id")
	orgID := middleware.GetCurrentOrgID(c)

	if err := h.txService.Accept(c.Request.Context(), id, orgID); err != nil {
		switch {
		case errors.Is(err, services.ErrTransactionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "المعاملة غير موجودة"})
		case errors.Is(err, services.ErrNotReceiver):
			c.JSON(http.StatusForbidden, gin.H{"error": "مش من حقك تقبل المعاملة دي"})
		case errors.Is(err, services.ErrAlreadyResponded):
			c.JSON(http.StatusBadRequest, gin.H{"error": "تم الرد على المعاملة دي من قبل"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "حصل خطأ"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "تم قبول المعاملة بنجاح"})
}

// PUT /api/v1/transactions/:id/reject
func (h *TransactionHandler) Reject(c *gin.Context) {
	id := c.Param("id")
	orgID := middleware.GetCurrentOrgID(c)

	var body struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "سبب الرفض مطلوب"})
		return
	}

	if err := h.txService.Reject(c.Request.Context(), id, orgID, body.Reason); err != nil {
		switch {
		case errors.Is(err, services.ErrTransactionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "المعاملة غير موجودة"})
		case errors.Is(err, services.ErrNotReceiver):
			c.JSON(http.StatusForbidden, gin.H{"error": "مش من حقك ترفض المعاملة دي"})
		case errors.Is(err, services.ErrAlreadyResponded):
			c.JSON(http.StatusBadRequest, gin.H{"error": "تم الرد على المعاملة دي من قبل"})
		case errors.Is(err, services.ErrRejectionReasonNeeded):
			c.JSON(http.StatusBadRequest, gin.H{"error": "سبب الرفض مطلوب"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "حصل خطأ"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "تم رفض المعاملة"})
}
