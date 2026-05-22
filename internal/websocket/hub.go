package websocket

import (
	"log"
	"sync"
)

// Client - يمثل connection واحد
type Client struct {
	UserID string
	OrgID  string
	Send   chan []byte
	Hub    *Hub
}

// Hub - بيدير كل الـ connections
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]bool // orgID -> clients
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[client.OrgID] == nil {
		h.clients[client.OrgID] = make(map[*Client]bool)
	}
	h.clients[client.OrgID][client] = true
	log.Printf("✅ WS: user %s connected (org: %s)", client.UserID, client.OrgID)
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.OrgID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.Send)
			log.Printf("❌ WS: user %s disconnected (org: %s)", client.UserID, client.OrgID)
		}
	}
}

// BroadcastToOrg - ابعت لكل يوزرين في جهة
func (h *Hub) BroadcastToOrg(orgID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients[orgID] {
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			delete(h.clients[orgID], client)
		}
	}
}

// BroadcastToUser - ابعت ليوزر معين بالـ ID
func (h *Hub) BroadcastToUser(userID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.clients {
		for client := range clients {
			if client.UserID == userID {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(clients, client)
				}
			}
		}
	}
}

func (h *Hub) OnlineCount(orgID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[orgID])
}
