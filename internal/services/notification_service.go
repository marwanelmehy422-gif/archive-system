package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"archive-system/internal/database"
	"archive-system/internal/models"
	ws "archive-system/internal/websocket"
)

type NotificationService struct {
	db  *database.DB
	hub *ws.Hub
}

func NewNotificationService(db *database.DB, hub *ws.Hub) *NotificationService {
	return &NotificationService{db: db, hub: hub}
}

// NotifyNewTransaction - لما مروان يبعت لمحمود
func (s *NotificationService) NotifyNewTransaction(ctx context.Context, tx *models.Transaction, senderOrgName string) {
	// جيب كل يوزرين في الجهة المستقبلة
	users, err := s.getUsersByOrg(ctx, tx.ReceiverOrgID)
	if err != nil {
		log.Printf("notify error: %v", err)
		return
	}

	title := fmt.Sprintf("معاملة جديدة من %s", senderOrgName)
	body := fmt.Sprintf("(%s) %s", tx.ReferenceNumber, tx.Title)

	for _, userID := range users {
		s.saveAndSend(ctx, userID, tx.ID, models.NotifNewTransaction, title, body)
	}
}

// NotifyAccepted - لما محمود يقبل، مروان يعرف
func (s *NotificationService) NotifyAccepted(ctx context.Context, tx *models.TransactionFull) {
	users, err := s.getUsersByOrg(ctx, tx.SenderOrgID)
	if err != nil {
		log.Printf("notify accepted error: %v", err)
		return
	}

	title := fmt.Sprintf("تم قبول معاملتك من %s", tx.ReceiverOrgName)
	body := fmt.Sprintf("(%s) %s", tx.ReferenceNumber, tx.Title)

	for _, userID := range users {
		s.saveAndSend(ctx, userID, tx.ID, models.NotifAccepted, title, body)
	}
}

// NotifyRejected - لما محمود يرفض، مروان يعرف مع السبب
func (s *NotificationService) NotifyRejected(ctx context.Context, tx *models.TransactionFull, reason string) {
	users, err := s.getUsersByOrg(ctx, tx.SenderOrgID)
	if err != nil {
		log.Printf("notify rejected error: %v", err)
		return
	}

	title := fmt.Sprintf("تم رفض معاملتك من %s", tx.ReceiverOrgName)
	body := fmt.Sprintf("(%s) %s - السبب: %s", tx.ReferenceNumber, tx.Title, reason)

	for _, userID := range users {
		s.saveAndSend(ctx, userID, tx.ID, models.NotifRejected, title, body)
	}
}

// GetUserNotifications - جيب كل إشعارات يوزر معين
func (s *NotificationService) GetUserNotifications(ctx context.Context, userID string) ([]models.Notification, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, user_id, transaction_id, type, title, body, is_read, read_at, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 50`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.TransactionID, &n.Type,
			&n.Title, &n.Body, &n.IsRead, &n.ReadAt, &n.CreatedAt,
		); err != nil {
			return nil, err
		}
		notifs = append(notifs, n)
	}
	if notifs == nil {
		notifs = []models.Notification{}
	}
	return notifs, nil
}

// MarkAllRead - اقرأ كل الإشعارات
func (s *NotificationService) MarkAllRead(ctx context.Context, userID string) error {
	_, err := s.db.Pool.Exec(ctx,
		`UPDATE notifications SET is_read = TRUE, read_at = $1 WHERE user_id = $2 AND is_read = FALSE`,
		time.Now(), userID,
	)
	return err
}

// UnreadCount - عدد الإشعارات الغير مقروءة
func (s *NotificationService) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := s.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`,
		userID,
	).Scan(&count)
	return count, err
}

// ── Private helpers ───────────────────────────────────────────

func (s *NotificationService) saveAndSend(ctx context.Context, userID, txID string, notifType models.NotificationType, title, body string) {
	// خزن في الـ DB
	var notifID string
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO notifications (user_id, transaction_id, type, title, body)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		userID, txID, notifType, title, body,
	).Scan(&notifID)
	if err != nil {
		log.Printf("save notification error: %v", err)
		return
	}

	// ابعت عبر WebSocket على طول
	msg, _ := json.Marshal(map[string]interface{}{
		"type": "notification",
		"payload": map[string]interface{}{
			"id":             notifID,
			"transaction_id": txID,
			"type":           notifType,
			"title":          title,
			"body":           body,
			"is_read":        false,
			"created_at":     time.Now(),
			"play_sound":     true,
		},
	})

	// ابعت للـ user المحدد
	s.hub.BroadcastToUser(userID, msg)
}

func (s *NotificationService) getUsersByOrg(ctx context.Context, orgID string) ([]string, error) {
	rows, err := s.db.Pool.Query(ctx,
		`SELECT id FROM users WHERE organization_id = $1 AND is_active = TRUE`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("getUsersByOrg: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, id)
	}
	return userIDs, nil
}
