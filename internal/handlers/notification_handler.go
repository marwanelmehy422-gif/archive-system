package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"archive-system/internal/middleware"
	"archive-system/internal/services"
)

type NotificationHandler struct {
	notifService *services.NotificationService
}

func NewNotificationHandler(notifService *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifService: notifService}
}

// GET /api/v1/notifications
func (h *NotificationHandler) GetAll(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)

	notifs, err := h.notifService.GetUserNotifications(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "حصل خطأ"})
		return
	}

	count, _ := h.notifService.UnreadCount(c.Request.Context(), userID)

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifs,
		"unread_count":  count,
	})
}

// PUT /api/v1/notifications/read-all
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)

	if err := h.notifService.MarkAllRead(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "حصل خطأ"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "تم تحديد كل الإشعارات كمقروءة"})
}
