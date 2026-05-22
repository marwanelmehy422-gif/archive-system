package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"archive-system/internal/models"
	"archive-system/internal/repository"
)

type OrgHandler struct {
	orgRepo *repository.OrgRepository
}

func NewOrgHandler(orgRepo *repository.OrgRepository) *OrgHandler {
	return &OrgHandler{orgRepo: orgRepo}
}

// POST /api/v1/organizations
func (h *OrgHandler) Create(c *gin.Context) {
	var body struct {
		Name string `json:"name" binding:"required"`
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// تأكد إن الـ code مش موجود
	existing, _ := h.orgRepo.GetByCode(c.Request.Context(), strings.ToUpper(body.Code))
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "كود الجهة موجود بالفعل"})
		return
	}

	org := &models.Organization{
		Name: body.Name,
		Code: strings.ToUpper(body.Code),
	}

	if err := h.orgRepo.Create(c.Request.Context(), org); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "حصل خطأ أثناء إنشاء الجهة"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"organization": org})
}

// GET /api/v1/organizations
func (h *OrgHandler) GetAll(c *gin.Context) {
	orgs, err := h.orgRepo.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "حصل خطأ"})
		return
	}
	if orgs == nil {
		orgs = []models.Organization{}
	}
	c.JSON(http.StatusOK, gin.H{"organizations": orgs})
}

// GET /api/v1/organizations/:code
func (h *OrgHandler) GetByCode(c *gin.Context) {
	code := c.Param("code")

	org, err := h.orgRepo.GetByCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "حصل خطأ"})
		return
	}
	if org == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "الجهة غير موجودة"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"organization": org})
}
