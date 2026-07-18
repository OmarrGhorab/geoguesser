package realtime

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

// DefaultOutboundQueueSize is the default per-client outbound buffer.
// Production wiring can pass config.RealtimeOutboundQueueSize (default 128).
const DefaultOutboundQueueSize = 128

// ChannelKey uniquely identifies a realtime fanout channel.
type ChannelKey struct {
	Kind string
	ID   string
}

// Client is a single subscriber on a channel. UserID and TeamSlot are optional
// server-side audience attributes; clients never supply delivery targets.
type Client struct {
	RoomCode    string
	ChannelKind string
	ChannelID   string
	UserID      uuid.UUID
	TeamSlot    *int
	Send        chan Event

	closed atomic.Bool
	done   chan struct{}
}

// NewClient constructs a subscriber with a bounded outbound queue.
// queueSize <= 0 uses DefaultOutboundQueueSize.
func NewClient(channelKind, channelID string, userID uuid.UUID, teamSlot *int, queueSize int) *Client {
	if queueSize < 1 {
		queueSize = DefaultOutboundQueueSize
	}
	return &Client{
		ChannelKind: channelKind,
		ChannelID:   channelID,
		UserID:      userID,
		TeamSlot:    teamSlot,
		Send:        make(chan Event, queueSize),
		done:        make(chan struct{}),
	}
}

// NewRoomClient constructs a legacy room subscriber.
func NewRoomClient(roomCode string, queueSize int) *Client {
	c := NewClient(ChannelKindRoom, roomCode, uuid.Nil, nil, queueSize)
	c.RoomCode = roomCode
	return c
}

// Done is closed when the hub drops the client (e.g. slow consumer).
func (c *Client) Done() <-chan struct{} {
	if c == nil || c.done == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return c.done
}

// IsClosed reports whether the client was dropped or unsubscribed as closed.
func (c *Client) IsClosed() bool {
	return c != nil && c.closed.Load()
}

func (c *Client) markClosed() {
	if c == nil {
		return
	}
	if c.closed.CompareAndSwap(false, true) {
		if c.done != nil {
			close(c.done)
		}
	}
}

func (c *Client) channelKey() ChannelKey {
	if c.ChannelKind != "" && c.ChannelID != "" {
		return ChannelKey{Kind: c.ChannelKind, ID: c.ChannelID}
	}
	if c.RoomCode != "" {
		return ChannelKey{Kind: ChannelKindRoom, ID: c.RoomCode}
	}
	return ChannelKey{Kind: c.ChannelKind, ID: c.ChannelID}
}

// Hub is an in-process fanout registry for room, party, and match channels.
type Hub struct {
	mu          sync.RWMutex
	channels    map[ChannelKey]map[*Client]struct{}
	lastVersion map[ChannelKey]int64
	queueSize   int
}

// NewHub creates a hub with the default outbound queue size.
func NewHub() *Hub {
	return NewHubWithQueueSize(DefaultOutboundQueueSize)
}

// NewHubWithQueueSize creates a hub; queueSize is used when Subscribe allocates Send.
func NewHubWithQueueSize(queueSize int) *Hub {
	if queueSize < 1 {
		queueSize = DefaultOutboundQueueSize
	}
	return &Hub{
		channels:    make(map[ChannelKey]map[*Client]struct{}),
		lastVersion: make(map[ChannelKey]int64),
		queueSize:   queueSize,
	}
}

// QueueSize returns the hub's configured outbound buffer size.
func (h *Hub) QueueSize() int {
	return h.queueSize
}

// Subscribe registers a client on its channel. Initializes Send/done when missing.
func (h *Hub) Subscribe(client *Client) {
	if client == nil {
		return
	}
	h.normalizeClient(client)
	key := client.channelKey()
	if key.Kind == "" || key.ID == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.channels[key] == nil {
		h.channels[key] = make(map[*Client]struct{})
	}
	h.channels[key][client] = struct{}{}
}

// Unsubscribe removes a client from its channel without marking it closed.
func (h *Hub) Unsubscribe(client *Client) {
	if client == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.removeLocked(client)
}

// Add is the legacy room subscription API. Prefer Subscribe for party/match.
func (h *Hub) Add(client *Client) {
	if client == nil {
		return
	}
	if client.ChannelKind == "" && client.RoomCode != "" {
		client.ChannelKind = ChannelKindRoom
		client.ChannelID = client.RoomCode
	}
	h.Subscribe(client)
}

// Remove is the legacy room unsubscription API.
func (h *Hub) Remove(client *Client) {
	h.Unsubscribe(client)
}

// Broadcast publishes to all subscribers of a room code (legacy API).
func (h *Hub) Broadcast(ctx context.Context, roomCode string, event Event) {
	h.Publish(ctx, ChannelKey{Kind: ChannelKindRoom, ID: roomCode}, event)
}

// Publish delivers event to every subscriber on the channel.
func (h *Hub) Publish(ctx context.Context, channel ChannelKey, event Event) {
	h.publishFiltered(ctx, channel, event, nil)
}

// PublishToUsers delivers event only to subscribers whose UserID is in userIDs.
func (h *Hub) PublishToUsers(ctx context.Context, channel ChannelKey, userIDs []uuid.UUID, event Event) {
	if len(userIDs) == 0 {
		return
	}
	wanted := make(map[uuid.UUID]struct{}, len(userIDs))
	for _, id := range userIDs {
		if id == uuid.Nil {
			continue
		}
		wanted[id] = struct{}{}
	}
	if len(wanted) == 0 {
		return
	}
	h.publishFiltered(ctx, channel, event, func(c *Client) bool {
		_, ok := wanted[c.UserID]
		return ok
	})
}

// PublishToTeam delivers event only to subscribers with the given team slot.
func (h *Hub) PublishToTeam(ctx context.Context, channel ChannelKey, teamSlot int, event Event) {
	h.publishFiltered(ctx, channel, event, func(c *Client) bool {
		return c.TeamSlot != nil && *c.TeamSlot == teamSlot
	})
}

// LastVersion returns the highest non-decreasing version observed for a channel.
func (h *Hub) LastVersion(channel ChannelKey) int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastVersion[channel]
}

// SubscriberCount returns the number of active clients on a channel.
func (h *Hub) SubscriberCount(channel ChannelKey) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.channels[channel])
}

func (h *Hub) normalizeClient(client *Client) {
	if client.done == nil {
		client.done = make(chan struct{})
	}
	if client.Send == nil {
		client.Send = make(chan Event, h.queueSize)
	}
	if client.ChannelKind == "" && client.RoomCode != "" {
		client.ChannelKind = ChannelKindRoom
		client.ChannelID = client.RoomCode
	}
	if client.RoomCode == "" && client.ChannelKind == ChannelKindRoom {
		client.RoomCode = client.ChannelID
	}
}

func (h *Hub) publishFiltered(_ context.Context, channel ChannelKey, event Event, filter func(*Client) bool) {
	if channel.Kind == "" || channel.ID == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Track non-decreasing channel versions for reconnect/gap detection.
	if event.Version > 0 {
		if prev, ok := h.lastVersion[channel]; !ok || event.Version >= prev {
			h.lastVersion[channel] = event.Version
		}
	}

	clients := h.channels[channel]
	if len(clients) == 0 {
		return
	}

	slow := make([]*Client, 0)
	for client := range clients {
		if client.IsClosed() {
			continue
		}
		if filter != nil && !filter(client) {
			continue
		}
		select {
		case client.Send <- event:
		default:
			// Bounded queue full: drop slow consumer instead of blocking.
			slow = append(slow, client)
		}
	}
	for _, client := range slow {
		h.removeLocked(client)
		client.markClosed()
	}
}

func (h *Hub) removeLocked(client *Client) {
	key := client.channelKey()
	set := h.channels[key]
	if set == nil {
		return
	}
	delete(set, client)
	if len(set) == 0 {
		delete(h.channels, key)
	}
}
