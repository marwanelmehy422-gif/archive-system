package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"archive-system/internal/middleware"
	"archive-system/internal/repository"
	"archive-system/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
	userRepo    *repository.UserRepository
}

func NewAuthHandler(authService *services.AuthService, userRepo *repository.UserRepository) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userRepo:    userRepo,
	}
}

// Login godoc
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{"error": "اسم المستخدم أو كلمة المرور غلط"})
		case errors.Is(err, services.ErrUserNotActive):
			c.JSON(http.StatusForbidden, gin.H{"error": "الحساب موقوف"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "حصل خطأ، حاول تاني"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      resp.Token,
		"expires_at": resp.ExpiresAt,
		"user":       resp.User,
	})
}

// Register godoc
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req services.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrOrgNotFound):
			c.JSON(http.StatusBadRequest, gin.H{"error": "كود الجهة غلط أو غير موجود"})
		case errors.Is(err, services.ErrUsernameExists):
			c.JSON(http.StatusConflict, gin.H{"error": "اسم المستخدم موجود بالفعل"})
		case errors.Is(err, services.ErrEmailExists):
			c.JSON(http.StatusConflict, gin.H{"error": "البريد الإلكتروني موجود بالفعل"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "حصل خطأ، حاول تاني"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token":      resp.Token,
		"expires_at": resp.ExpiresAt,
		"user":       resp.User,
	})
}

// Me godoc
// GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "المستخدم مش موجود"})
		return
	}

	user.PasswordHash = ""
	c.JSON(http.StatusOK, gin.H{"user": user})
}
