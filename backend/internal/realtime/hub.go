package realtime

import (
	"context"
	"sync"
)

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]struct{}
}

type Client struct {
	RoomCode string
	Send     chan Event
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[string]map[*Client]struct{})}
}

func (h *Hub) Add(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[client.RoomCode] == nil {
		h.rooms[client.RoomCode] = make(map[*Client]struct{})
	}
	h.rooms[client.RoomCode][client] = struct{}{}
}

func (h *Hub) Remove(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients := h.rooms[client.RoomCode]
	delete(clients, client)
	if len(clients) == 0 {
		delete(h.rooms, client.RoomCode)
	}
}

func (h *Hub) Broadcast(_ context.Context, roomCode string, event Event) {
	h.mu.RLock()
	clients := h.rooms[roomCode]
	for client := range clients {
		select {
		case client.Send <- event:
		default:
		}
	}
	h.mu.RUnlock()
}
