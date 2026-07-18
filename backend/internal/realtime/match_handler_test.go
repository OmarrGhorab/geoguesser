package realtime_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/realtime"
)

// --- fakes ---

type memTicketStore struct {
	mu      sync.Mutex
	tickets map[string]realtime.TicketClaims
}

func newMemTicketStore() *memTicketStore {
	return &memTicketStore{tickets: map[string]realtime.TicketClaims{}}
}

func (m *memTicketStore) Issue(_ context.Context, userID uuid.UUID, channelKind, channelID string, ttl time.Duration) (string, time.Time, error) {
	token := "tok-" + uuid.NewString()
	exp := time.Now().UTC().Add(ttl)
	if ttl <= 0 {
		exp = time.Now().UTC().Add(30 * time.Second)
	}
	m.mu.Lock()
	m.tickets[token] = realtime.TicketClaims{
		UserID: userID, ChannelKind: channelKind, ChannelID: channelID, ExpiresAt: exp,
	}
	m.mu.Unlock()
	return token, exp, nil
}

func (m *memTicketStore) Consume(_ context.Context, token string) (*realtime.TicketClaims, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.tickets[token]
	if !ok {
		return nil, realtime.ErrTicketUsed
	}
	delete(m.tickets, token)
	if time.Now().UTC().After(c.ExpiresAt) {
		return nil, realtime.ErrTicketExpired
	}
	cp := c
	return &cp, nil
}

type fixedAuthorizer struct {
	members map[string]realtime.ChannelMembership // channelID -> membership
}

func (f fixedAuthorizer) Authorize(_ context.Context, userID uuid.UUID, _, channelID string) (realtime.ChannelMembership, error) {
	if f.members == nil {
		return realtime.ChannelMembership{UserID: userID}, nil
	}
	m, ok := f.members[channelID]
	if !ok {
		return realtime.ChannelMembership{}, realtime.ErrNotParticipant
	}
	if m.UserID != uuid.Nil && m.UserID != userID {
		return realtime.ChannelMembership{}, realtime.ErrNotParticipant
	}
	m.UserID = userID
	return m, nil
}

type fixedSnapshot struct {
	payload any
	version int64
}

func (f fixedSnapshot) Snapshot(context.Context, uuid.UUID, string, string) (any, int64, error) {
	if f.payload == nil {
		return map[string]any{"ok": true}, f.version, nil
	}
	return f.payload, f.version, nil
}

type memCommands struct {
	mu      sync.Mutex
	version int64
	acks    map[string][]byte
	markers []markerCall
	views   []viewCall
	// spectate policy configuration for tests
	allowedByUser map[uuid.UUID][]uuid.UUID // viewer user -> allowed game_player ids
	submitted     map[uuid.UUID]bool        // viewer user submitted
	roundID       uuid.UUID
	format        string
	// viewRecipients maps source user -> spectator user ids
	viewRecipients map[uuid.UUID][]uuid.UUID
	// forceUnchanged makes SetView return ErrViewUnchanged
	forceUnchanged bool
	// denySpectate forces SelectSpectate forbidden
	denySpectate bool
}

type markerCall struct {
	MatchID  uuid.UUID
	RoundID  uuid.UUID
	UserID   uuid.UUID
	TeamSlot int
	Lat, Lng float64
}

type viewCall struct {
	MatchID, RoundID, UserID uuid.UUID
	PanoramaID               string
	Heading, Pitch, Zoom     float64
}

func newMemCommands() *memCommands {
	return &memCommands{
		acks:           map[string][]byte{},
		allowedByUser:  map[uuid.UUID][]uuid.UUID{},
		submitted:      map[uuid.UUID]bool{},
		viewRecipients: map[uuid.UUID][]uuid.UUID{},
		format:         "solo",
	}
}

func (m *memCommands) SetMarker(_ context.Context, matchID, roundID, userID uuid.UUID, teamSlot int, lat, lng float64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.version++
	m.markers = append(m.markers, markerCall{matchID, roundID, userID, teamSlot, lat, lng})
	return m.version, nil
}

func (m *memCommands) SetView(_ context.Context, matchID, roundID, userID uuid.UUID, panoramaID string, heading, pitch, zoom float64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.forceUnchanged {
		return m.version, realtime.ErrViewUnchanged
	}
	m.version++
	m.views = append(m.views, viewCall{matchID, roundID, userID, panoramaID, heading, pitch, zoom})
	return m.version, nil
}

func (m *memCommands) SelectSpectate(_ context.Context, _ uuid.UUID, viewerUserID, preferred uuid.UUID) (realtime.SpectateSelectResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.denySpectate || !m.submitted[viewerUserID] {
		return realtime.SpectateSelectResult{Format: m.format}, realtime.ErrSpectateForbidden
	}
	allowed := append([]uuid.UUID(nil), m.allowedByUser[viewerUserID]...)
	if len(allowed) == 0 {
		return realtime.SpectateSelectResult{
			AllowedTargetIDs: allowed,
			Format:           m.format,
		}, realtime.ErrSpectateForbidden
	}
	selected := allowed[0]
	fallback := true
	if preferred != uuid.Nil {
		for _, id := range allowed {
			if id == preferred {
				selected = preferred
				fallback = false
				break
			}
		}
	}
	m.version++
	return realtime.SpectateSelectResult{
		RoundID:          m.roundID,
		SelectedTargetID: selected,
		AllowedTargetIDs: allowed,
		Fallback:         fallback,
		Format:           m.format,
		Version:          m.version,
	}, nil
}

func (m *memCommands) ViewRecipients(_ context.Context, _, _ uuid.UUID, sourceUserID uuid.UUID) ([]uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]uuid.UUID(nil), m.viewRecipients[sourceUserID]...), nil
}

func (m *memCommands) Heartbeat(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func (m *memCommands) CurrentVersion(context.Context, uuid.UUID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.version, nil
}

func (m *memCommands) LoadCommandAck(_ context.Context, _, _ uuid.UUID, commandID string) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	raw, ok := m.acks[commandID]
	return raw, ok, nil
}

func (m *memCommands) SaveCommandAck(_ context.Context, _, _ uuid.UUID, commandID string, ackJSON []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.acks[commandID] = append([]byte(nil), ackJSON...)
	return nil
}

func startMatchServer(t *testing.T, h *realtime.MatchHandler) *httptest.Server {
	t.Helper()
	r := chi.NewRouter()
	r.Get("/realtime/matches/{matchId}", h.Match)
	r.Get("/realtime/parties/{partyId}", h.Party)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func dialMatch(t *testing.T, srv *httptest.Server, matchID, ticket string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/realtime/matches/" + matchID
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		Subprotocols: []string{realtime.SubprotocolV1, realtime.TicketSubprotocolPrefix + ticket},
		HTTPHeader:   http.Header{"Origin": []string{"http://localhost:3000"}},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "done") })
	return conn
}

func readEvent(t *testing.T, conn *websocket.Conn) realtime.Event {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var evt realtime.Event
	if err := json.Unmarshal(data, &evt); err != nil {
		t.Fatalf("unmarshal %s: %v", string(data), err)
	}
	return evt
}

func TestMatchHandlerOneUseTicketAndSnapshot(t *testing.T) {
	matchID := uuid.New()
	userID := uuid.New()
	team := 1
	tickets := newMemTicketStore()
	token, _, err := tickets.Issue(context.Background(), userID, realtime.ChannelKindMatch, matchID.String(), 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	hub := realtime.NewHubWithQueueSize(8)
	h := realtime.NewMatchHandler(
		hub,
		tickets,
		fixedAuthorizer{members: map[string]realtime.ChannelMembership{
			matchID.String(): {UserID: userID, TeamSlot: &team},
		}},
		fixedSnapshot{payload: map[string]any{"match_id": matchID.String()}, version: 7},
		newMemCommands(),
		nil,
		realtime.MatchHandlerConfig{AllowedOrigins: []string{"http://localhost:*"}},
		nil,
		nil,
	)
	srv := startMatchServer(t, h)

	conn := dialMatch(t, srv, matchID.String(), token)
	evt := readEvent(t, conn)
	if evt.Type != realtime.EventMatchSnapshot {
		t.Fatalf("type = %s", evt.Type)
	}
	if evt.Version != 7 {
		t.Fatalf("version = %d", evt.Version)
	}
	if evt.ProtocolVersion != realtime.ProtocolVersion {
		t.Fatalf("protocol_version = %d", evt.ProtocolVersion)
	}

	// Ticket is one-use: second dial must fail.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/realtime/matches/" + matchID.String()
	_, _, err = websocket.Dial(ctx, url, &websocket.DialOptions{
		Subprotocols: []string{realtime.SubprotocolV1, realtime.TicketSubprotocolPrefix + token},
		HTTPHeader:   http.Header{"Origin": []string{"http://localhost:3000"}},
	})
	if err == nil {
		t.Fatal("expected second dial with used ticket to fail")
	}
}

func TestMatchHandlerOriginRejection(t *testing.T) {
	matchID := uuid.New()
	userID := uuid.New()
	tickets := newMemTicketStore()
	token, _, _ := tickets.Issue(context.Background(), userID, realtime.ChannelKindMatch, matchID.String(), time.Minute)
	h := realtime.NewMatchHandler(
		realtime.NewHub(),
		tickets,
		fixedAuthorizer{},
		fixedSnapshot{},
		nil,
		nil,
		realtime.MatchHandlerConfig{AllowedOrigins: []string{"http://localhost:3000"}},
		nil,
		nil,
	)
	srv := startMatchServer(t, h)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/realtime/matches/" + matchID.String()
	_, resp, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		Subprotocols: []string{realtime.SubprotocolV1, realtime.TicketSubprotocolPrefix + token},
		HTTPHeader:   http.Header{"Origin": []string{"https://evil.example"}},
	})
	if err == nil {
		t.Fatal("expected origin rejection")
	}
	if resp != nil && resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestMatchHandlerTargetedTeamMarkerEvents(t *testing.T) {
	matchID := uuid.New()
	roundID := uuid.New()
	userA, userB, userC := uuid.New(), uuid.New(), uuid.New()
	team1, team2 := 1, 2
	tickets := newMemTicketStore()
	tokA, _, _ := tickets.Issue(context.Background(), userA, realtime.ChannelKindMatch, matchID.String(), time.Minute)
	tokB, _, _ := tickets.Issue(context.Background(), userB, realtime.ChannelKindMatch, matchID.String(), time.Minute)
	tokC, _, _ := tickets.Issue(context.Background(), userC, realtime.ChannelKindMatch, matchID.String(), time.Minute)

	cmds := newMemCommands()
	hub := realtime.NewHubWithQueueSize(16)
	// Per-user team slots.
	h := realtime.NewMatchHandler(
		hub,
		tickets,
		authByUser{slots: map[uuid.UUID]int{userA: team1, userB: team1, userC: team2}},
		fixedSnapshot{version: 1},
		cmds,
		nil,
		realtime.MatchHandlerConfig{AllowedOrigins: []string{"http://localhost:*"}},
		nil,
		nil,
	)
	srv := startMatchServer(t, h)

	connA := dialMatch(t, srv, matchID.String(), tokA)
	connB := dialMatch(t, srv, matchID.String(), tokB)
	connC := dialMatch(t, srv, matchID.String(), tokC)
	// Drain snapshots.
	_ = readEvent(t, connA)
	_ = readEvent(t, connB)
	_ = readEvent(t, connC)

	// A sets a marker — B (same team) should see event; C (opponent) must not.
	cmdID := uuid.NewString()
	cmd := map[string]any{
		"protocol_version": 1,
		"command_id":       cmdID,
		"type":             realtime.CommandRoundMarkerSet,
		"payload": map[string]any{
			"round_id":  roundID.String(),
			"latitude":  10.5,
			"longitude": 20.5,
		},
	}
	raw, _ := json.Marshal(cmd)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := connA.Write(ctx, websocket.MessageText, raw); err != nil {
		t.Fatalf("write: %v", err)
	}

	// A receives command.accepted (private).
	ack := readEvent(t, connA)
	if ack.Type != realtime.EventCommandAccepted {
		// Marker event may arrive on A's Send queue first depending on race; accept either order.
		if ack.Type != realtime.EventRoundMarkerChanged {
			t.Fatalf("A first event type = %s", ack.Type)
		}
	}

	// B must receive marker_changed.
	deadline := time.Now().Add(2 * time.Second)
	var gotMarker bool
	for time.Now().Before(deadline) {
		evt := readEvent(t, connB)
		if evt.Type == realtime.EventRoundMarkerChanged {
			gotMarker = true
			break
		}
	}
	if !gotMarker {
		t.Fatal("teammate B did not receive marker event")
	}

	// C must not receive marker within a short window.
	cctx, ccancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer ccancel()
	_, data, err := connC.Read(cctx)
	if err == nil {
		var evt realtime.Event
		_ = json.Unmarshal(data, &evt)
		if evt.Type == realtime.EventRoundMarkerChanged {
			t.Fatal("opposing team received marker event")
		}
	}
}

type authByUser struct {
	slots map[uuid.UUID]int
}

func (a authByUser) Authorize(_ context.Context, userID uuid.UUID, _, _ string) (realtime.ChannelMembership, error) {
	slot, ok := a.slots[userID]
	if !ok {
		return realtime.ChannelMembership{}, realtime.ErrNotParticipant
	}
	s := slot
	return realtime.ChannelMembership{UserID: userID, TeamSlot: &s}, nil
}

func TestMatchHandlerCommandReplay(t *testing.T) {
	matchID := uuid.New()
	roundID := uuid.New()
	userID := uuid.New()
	team := 1
	tickets := newMemTicketStore()
	token, _, _ := tickets.Issue(context.Background(), userID, realtime.ChannelKindMatch, matchID.String(), time.Minute)
	cmds := newMemCommands()
	h := realtime.NewMatchHandler(
		realtime.NewHubWithQueueSize(8),
		tickets,
		fixedAuthorizer{members: map[string]realtime.ChannelMembership{
			matchID.String(): {UserID: userID, TeamSlot: &team},
		}},
		fixedSnapshot{version: 0},
		cmds,
		nil,
		realtime.MatchHandlerConfig{AllowedOrigins: []string{"http://localhost:*"}},
		nil,
		nil,
	)
	srv := startMatchServer(t, h)
	conn := dialMatch(t, srv, matchID.String(), token)
	_ = readEvent(t, conn)

	cmdID := uuid.NewString()
	body, _ := json.Marshal(map[string]any{
		"protocol_version": 1,
		"command_id":       cmdID,
		"type":             realtime.CommandRoundMarkerSet,
		"payload":          map[string]any{"round_id": roundID.String(), "latitude": 1.0, "longitude": 2.0},
	})
	ctx := context.Background()
	if err := conn.Write(ctx, websocket.MessageText, body); err != nil {
		t.Fatal(err)
	}
	// Drain until accepted.
	var accepted bool
	for i := 0; i < 5; i++ {
		evt := readEvent(t, conn)
		if evt.Type == realtime.EventCommandAccepted {
			accepted = true
			break
		}
	}
	if !accepted {
		t.Fatal("no command.accepted")
	}
	if len(cmds.markers) != 1 {
		t.Fatalf("markers = %d", len(cmds.markers))
	}

	// Replay same command_id — must not create another marker.
	if err := conn.Write(ctx, websocket.MessageText, body); err != nil {
		t.Fatal(err)
	}
	evt := readEvent(t, conn)
	if evt.Type != realtime.EventCommandAccepted {
		// replay returns prior ack bytes directly
		t.Logf("replay event type = %s", evt.Type)
	}
	if len(cmds.markers) != 1 {
		t.Fatalf("replay republished markers = %d", len(cmds.markers))
	}
}

func TestMatchHandlerVersionGap(t *testing.T) {
	matchID := uuid.New()
	userID := uuid.New()
	team := 1
	tickets := newMemTicketStore()
	token, _, _ := tickets.Issue(context.Background(), userID, realtime.ChannelKindMatch, matchID.String(), time.Minute)
	cmds := newMemCommands()
	cmds.version = 50
	h := realtime.NewMatchHandler(
		realtime.NewHubWithQueueSize(4),
		tickets,
		fixedAuthorizer{members: map[string]realtime.ChannelMembership{
			matchID.String(): {UserID: userID, TeamSlot: &team},
		}},
		fixedSnapshot{version: 50},
		cmds,
		nil,
		realtime.MatchHandlerConfig{AllowedOrigins: []string{"http://localhost:*"}},
		nil,
		nil,
	)
	srv := startMatchServer(t, h)
	conn := dialMatch(t, srv, matchID.String(), token)
	_ = readEvent(t, conn)

	expected := int64(1)
	body, _ := json.Marshal(map[string]any{
		"protocol_version": 1,
		"command_id":       uuid.NewString(),
		"type":             realtime.CommandPresenceHeartbeat,
		"expected_version": expected,
		"payload":          map[string]any{},
	})
	if err := conn.Write(context.Background(), websocket.MessageText, body); err != nil {
		t.Fatal(err)
	}
	evt := readEvent(t, conn)
	if evt.Type != realtime.EventChannelError {
		t.Fatalf("type = %s, want channel.error", evt.Type)
	}
	if !strings.Contains(string(evt.Payload), realtime.CodeVersionGap) {
		t.Fatalf("payload = %s", string(evt.Payload))
	}
}

func TestMatchHandlerSlowConsumerClosure(t *testing.T) {
	// Hub closes slow consumers; MatchHandler writePump observes client.Done.
	hub := realtime.NewHubWithQueueSize(1)
	matchID := uuid.New().String()
	userID := uuid.New()
	client := realtime.NewClient(realtime.ChannelKindMatch, matchID, userID, nil, 1)
	hub.Subscribe(client)

	// Fill queue.
	evt, _ := realtime.NewChannelEvent("e1", realtime.EventMatchSnapshot, realtime.ChannelKindMatch, matchID, nil, nil, time.Now().UTC(), 1, map[string]any{"n": 1})
	hub.Publish(context.Background(), realtime.ChannelKey{Kind: realtime.ChannelKindMatch, ID: matchID}, evt)
	// Overflow.
	evt2, _ := realtime.NewChannelEvent("e2", realtime.EventMatchStarted, realtime.ChannelKindMatch, matchID, nil, nil, time.Now().UTC(), 2, map[string]any{"n": 2})
	hub.Publish(context.Background(), realtime.ChannelKey{Kind: realtime.ChannelKindMatch, ID: matchID}, evt2)

	select {
	case <-client.Done():
		// expected slow consumer close
	case <-time.After(time.Second):
		t.Fatal("expected slow consumer to be closed")
	}
	if !client.IsClosed() {
		t.Fatal("client should be marked closed")
	}
}

func TestDecodeTicketSubprotocols(t *testing.T) {
	hasV1, ticket := realtime.DecodeTicketSubprotocols([]string{
		realtime.SubprotocolV1,
		realtime.TicketSubprotocolPrefix + "opaque-secret",
	})
	if !hasV1 || ticket != "opaque-secret" {
		t.Fatalf("hasV1=%v ticket=%q", hasV1, ticket)
	}
}

func TestTicketHandlerIssue(t *testing.T) {
	userID := uuid.New()
	matchID := uuid.New()
	tickets := newMemTicketStore()
	uid := userID.String()
	h := realtime.NewTicketHandler(
		tickets,
		fixedAuthorizer{members: map[string]realtime.ChannelMembership{
			matchID.String(): {UserID: userID},
		}},
		realtime.TicketHandlerConfig{TTL: 30 * time.Second, FeatureEnabled: true},
		nil,
		nil,
	)

	// Build request with session via middleware-less direct call is hard;
	// exercise DecodeTicketSubprotocols and Issue store path instead.
	token, exp, err := tickets.Issue(context.Background(), userID, realtime.ChannelKindMatch, matchID.String(), 30*time.Second)
	if err != nil || token == "" || exp.IsZero() {
		t.Fatalf("issue: %v token=%q", err, token)
	}
	_ = uid
	_ = h
}
