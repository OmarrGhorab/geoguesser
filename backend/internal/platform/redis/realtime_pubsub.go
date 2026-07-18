package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Channel kinds for multi-instance realtime fanout.
const (
	ChannelKindParty = "party"
	ChannelKindMatch = "match"
)

const (
	realtimePubSubKeyPrefix = "realtime:v1:pubsub:"
	defaultSeenEventTTL     = 2 * time.Minute
	defaultSeenEventMax     = 10_000
)

// MessageHandler is invoked once per unique event_id for a subscribed channel.
// payload is the raw Redis Pub/Sub message body (typically a JSON event envelope).
type MessageHandler func(channelKind, channelID string, payload []byte)

// RealtimePubSub provides reference-counted Redis Pub/Sub subscriptions for
// party/match channels so multi-instance processes share a single SUBSCRIBE per
// locally active channel and dedupe fanout by event_id.
type RealtimePubSub struct {
	client  *redis.Client
	handler MessageHandler
	logger  *slog.Logger

	seenTTL time.Duration
	seenMax int

	mu     sync.Mutex
	refs   map[string]int // full Redis channel name -> local refcount
	pubsub *redis.PubSub
	closed bool
	done   chan struct{}
	wg     sync.WaitGroup

	seenMu sync.Mutex
	seen   map[string]time.Time // event_id -> expiry
}

// NewRealtimePubSub constructs a fanout helper. handler may be nil (messages are dropped).
func NewRealtimePubSub(client *redis.Client, handler MessageHandler) *RealtimePubSub {
	return &RealtimePubSub{
		client:  client,
		handler: handler,
		logger:  slog.Default(),
		seenTTL: defaultSeenEventTTL,
		seenMax: defaultSeenEventMax,
		refs:    make(map[string]int),
		seen:    make(map[string]time.Time),
		done:    make(chan struct{}),
	}
}

// WithLogger overrides the default logger used for receive-loop diagnostics.
func (p *RealtimePubSub) WithLogger(logger *slog.Logger) *RealtimePubSub {
	if p == nil {
		return p
	}
	if logger == nil {
		logger = slog.Default()
	}
	p.logger = logger
	return p
}

// RealtimePubSubChannel builds the Redis channel name for a kind/id pair.
func RealtimePubSubChannel(channelKind, channelID string) string {
	return realtimePubSubKeyPrefix + strings.TrimSpace(channelKind) + ":" + strings.TrimSpace(channelID)
}

// Subscribe increments the local refcount for the channel. The first reference
// issues a Redis SUBSCRIBE; subsequent calls only bump the count.
func (p *RealtimePubSub) Subscribe(ctx context.Context, channelKind, channelID string) error {
	if p == nil || p.client == nil {
		return nil
	}
	channelKind = strings.TrimSpace(channelKind)
	channelID = strings.TrimSpace(channelID)
	if channelKind == "" || channelID == "" {
		return fmt.Errorf("realtime pubsub subscribe: channel kind and id are required")
	}
	name := RealtimePubSubChannel(channelKind, channelID)

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return fmt.Errorf("realtime pubsub subscribe: shutdown")
	}

	if p.refs[name] > 0 {
		p.refs[name]++
		return nil
	}

	if p.pubsub == nil {
		// Empty subscribe establishes the connection; channels are added next.
		ps := p.client.Subscribe(ctx)
		if err := ps.Ping(ctx); err != nil {
			_ = ps.Close()
			return fmt.Errorf("realtime pubsub connect: %w", err)
		}
		p.pubsub = ps
		p.wg.Add(1)
		go p.receiveLoop()
	}

	if err := p.pubsub.Subscribe(ctx, name); err != nil {
		return fmt.Errorf("realtime pubsub subscribe %s: %w", name, err)
	}
	p.refs[name] = 1
	return nil
}

// Unsubscribe decrements the local refcount. When it reaches zero the Redis
// channel is unsubscribed. Unsubscribing an unknown channel is a no-op.
func (p *RealtimePubSub) Unsubscribe(ctx context.Context, channelKind, channelID string) error {
	if p == nil || p.client == nil {
		return nil
	}
	name := RealtimePubSubChannel(strings.TrimSpace(channelKind), strings.TrimSpace(channelID))

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	count := p.refs[name]
	if count <= 0 {
		return nil
	}
	if count > 1 {
		p.refs[name] = count - 1
		return nil
	}
	delete(p.refs, name)
	if p.pubsub != nil {
		if err := p.pubsub.Unsubscribe(ctx, name); err != nil {
			return fmt.Errorf("realtime pubsub unsubscribe %s: %w", name, err)
		}
	}
	return nil
}

// Publish sends payload to the Redis channel for the given kind/id.
func (p *RealtimePubSub) Publish(ctx context.Context, channelKind, channelID string, payload []byte) error {
	if p == nil || p.client == nil {
		return nil
	}
	name := RealtimePubSubChannel(strings.TrimSpace(channelKind), strings.TrimSpace(channelID))
	if err := p.client.Publish(ctx, name, payload).Err(); err != nil {
		return fmt.Errorf("realtime pubsub publish %s: %w", name, err)
	}
	return nil
}

// RefCount returns the local subscription reference count for tests and diagnostics.
func (p *RealtimePubSub) RefCount(channelKind, channelID string) int {
	if p == nil {
		return 0
	}
	name := RealtimePubSubChannel(strings.TrimSpace(channelKind), strings.TrimSpace(channelID))
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.refs[name]
}

// Shutdown unsubscribes all channels, closes the Pub/Sub connection, and waits
// for the receive loop to exit. It is safe to call multiple times.
func (p *RealtimePubSub) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	ps := p.pubsub
	p.pubsub = nil
	p.refs = make(map[string]int)
	// Signal receive loop before closing so it can exit cleanly.
	select {
	case <-p.done:
	default:
		close(p.done)
	}
	p.mu.Unlock()

	var closeErr error
	if ps != nil {
		if err := ps.Close(); err != nil {
			closeErr = fmt.Errorf("realtime pubsub close: %w", err)
		}
	}

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		if closeErr != nil {
			return closeErr
		}
		return ctx.Err()
	}
	return closeErr
}

func (p *RealtimePubSub) receiveLoop() {
	defer p.wg.Done()

	p.mu.Lock()
	ps := p.pubsub
	p.mu.Unlock()
	if ps == nil {
		return
	}

	ch := ps.Channel()
	for {
		select {
		case <-p.done:
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg == nil {
				continue
			}
			p.dispatch(msg.Channel, []byte(msg.Payload))
		}
	}
}

func (p *RealtimePubSub) dispatch(redisChannel string, payload []byte) {
	kind, id, ok := parseRealtimePubSubChannel(redisChannel)
	if !ok {
		return
	}
	eventID := extractEventID(payload)
	if eventID != "" && p.seenEvent(eventID) {
		return
	}
	if p.handler == nil {
		return
	}
	// Protect the process from a panicking handler.
	defer func() {
		if rec := recover(); rec != nil {
			p.logger.Error("realtime pubsub handler panic",
				slog.Any("recover", rec),
				slog.String("channel_kind", kind),
				slog.String("channel_id", id),
			)
		}
	}()
	p.handler(kind, id, payload)
}

func (p *RealtimePubSub) seenEvent(eventID string) bool {
	now := time.Now()
	p.seenMu.Lock()
	defer p.seenMu.Unlock()

	// Opportunistic expiry sweep when map grows large.
	if len(p.seen) > p.seenMax/2 {
		for id, exp := range p.seen {
			if now.After(exp) {
				delete(p.seen, id)
			}
		}
	}
	if exp, ok := p.seen[eventID]; ok && now.Before(exp) {
		return true
	}
	// Bound memory: if still over max after sweep, drop arbitrary entries.
	if len(p.seen) >= p.seenMax {
		for id := range p.seen {
			delete(p.seen, id)
			if len(p.seen) < p.seenMax {
				break
			}
		}
	}
	p.seen[eventID] = now.Add(p.seenTTL)
	return false
}

func parseRealtimePubSubChannel(name string) (kind, id string, ok bool) {
	if !strings.HasPrefix(name, realtimePubSubKeyPrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(name, realtimePubSubKeyPrefix)
	// kind:id — kind has no colons; id may be a UUID (no colons in standard form) or opaque.
	idx := strings.IndexByte(rest, ':')
	if idx <= 0 || idx == len(rest)-1 {
		return "", "", false
	}
	return rest[:idx], rest[idx+1:], true
}

func extractEventID(payload []byte) string {
	if len(payload) == 0 {
		return ""
	}
	var envelope struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return ""
	}
	return strings.TrimSpace(envelope.EventID)
}
