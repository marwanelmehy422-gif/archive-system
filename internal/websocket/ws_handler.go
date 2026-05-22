package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	gws "github.com/gorilla/websocket"
)

var upgrader = gws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	hub       *Hub
	jwtSecret string
}

func NewWSHandler(hub *Hub, jwtSecret string) *WSHandler {
	return &WSHandler{hub: hub, jwtSecret: jwtSecret}
}

// GET /ws?token=<JWT>
func (h *WSHandler) Handle(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token مطلوب"})
		return
	}

	// تحقق من الـ token مباشرة بدون AuthService
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !parsed.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token غير صالح"})
		return
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token غير صالح"})
		return
	}

	userID := claims["user_id"].(string)
	orgID := claims["org_id"].(string)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}

	client := &Client{
		UserID: userID,
		OrgID:  orgID,
		Send:   make(chan []byte, 256),
		Hub:    h.hub,
	}

	h.hub.Register(client)

	welcome, _ := json.Marshal(map[string]interface{}{
		"type": "connected",
		"payload": map[string]string{
			"message": "متصل بنجاح",
			"user_id": userID,
			"org_id":  orgID,
		},
	})
	client.Send <- welcome

	go writePump(client, conn)
	go readPump(client, conn)
}

func writePump(client *Client, conn *gws.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		conn.Close()
		client.Hub.Unregister(client)
	}()

	for {
		select {
		case message, ok := <-client.Send:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				conn.WriteMessage(gws.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(gws.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(gws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func readPump(client *Client, conn *gws.Conn) {
	defer func() {
		client.Hub.Unregister(client)
		conn.Close()
	}()

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
