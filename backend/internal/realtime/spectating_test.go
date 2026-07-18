package realtime_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/realtime"
)

func TestSpectateSelectAuthorizationAndFallback(t *testing.T) {
	matchID := uuid.New()
	roundID := uuid.New()
	viewer := uuid.New()
	targetA := uuid.New()
	targetB := uuid.New()
	forbidden := uuid.New()
	team := 1

	tickets := newMemTicketStore()
	token, _, err := tickets.Issue(context.Background(), viewer, realtime.ChannelKindMatch, matchID.String(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	cmds := newMemCommands()
	cmds.roundID = roundID
	cmds.format = "duo"
	cmds.submitted[viewer] = true
	cmds.allowedByUser[viewer] = []uuid.UUID{targetA, targetB}

	h := realtime.NewMatchHandler(
		realtime.NewHubWithQueueSize(16),
		tickets,
		fixedAuthorizer{members: map[string]realtime.ChannelMembership{
			matchID.String(): {UserID: viewer, TeamSlot: &team},
		}},
		fixedSnapshot{version: 1},
		cmds,
		nil,
		realtime.MatchHandlerConfig{AllowedOrigins: []string{"http://localhost:*"}},
		nil,
		nil,
	)
	srv := startMatchServer(t, h)
	conn := dialMatch(t, srv, matchID.String(), token)
	_ = readEvent(t, conn) // snapshot

	// Preferred forbidden target → automatic fallback to first allowed.
	cmdID := uuid.NewString()
	body, _ := json.Marshal(map[string]any{
		"protocol_version": 1,
		"command_id":       cmdID,
		"type":             realtime.CommandRoundSpectateSelect,
		"payload": map[string]any{
			"target_game_player_id": forbidden.String(),
			"round_id":              roundID.String(),
		},
	})
	ctx := context.Background()
	if err := conn.Write(ctx, websocket.MessageText, body); err != nil {
		t.Fatal(err)
	}

	var accepted, targetChanged bool
	var fallback bool
	var selected string
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && (!accepted || !targetChanged) {
		evt := readEvent(t, conn)
		switch evt.Type {
		case realtime.EventCommandAccepted:
			accepted = true
			var payload map[string]any
			_ = json.Unmarshal(evt.Payload, &payload)
			if v, ok := payload["fallback"].(bool); ok {
				fallback = v
			}
			if v, ok := payload["target_game_player_id"].(string); ok {
				selected = v
			}
		case realtime.EventRoundSpectateTargetChanged:
			targetChanged = true
			var payload map[string]any
			_ = json.Unmarshal(evt.Payload, &payload)
			if v, ok := payload["fallback"].(bool); ok && v {
				fallback = true
			}
			if v, ok := payload["target_game_player_id"].(string); ok {
				selected = v
			}
			// Privacy: no guess/score fields.
			raw := string(evt.Payload)
			for _, bad := range []string{"latitude", "longitude", "guess", "score", "answer"} {
				if strings.Contains(raw, bad) {
					t.Fatalf("spectate event leaked %q: %s", bad, raw)
				}
			}
		case realtime.EventChannelError:
			t.Fatalf("unexpected error: %s", string(evt.Payload))
		}
	}
	if !accepted {
		t.Fatal("expected command.accepted")
	}
	if !fallback {
		t.Fatal("expected fallback when preferred target forbidden")
	}
	if selected != targetA.String() {
		t.Fatalf("selected = %q want %s", selected, targetA)
	}
}

func TestSpectateSelectDeniedForUnsubmittedViewer(t *testing.T) {
	matchID := uuid.New()
	viewer := uuid.New()
	team := 1
	tickets := newMemTicketStore()
	token, _, _ := tickets.Issue(context.Background(), viewer, realtime.ChannelKindMatch, matchID.String(), time.Minute)

	cmds := newMemCommands()
	cmds.submitted[viewer] = false // unsubmitted
	cmds.allowedByUser[viewer] = []uuid.UUID{uuid.New()}

	h := realtime.NewMatchHandler(
		realtime.NewHubWithQueueSize(8),
		tickets,
		fixedAuthorizer{members: map[string]realtime.ChannelMembership{
			matchID.String(): {UserID: viewer, TeamSlot: &team},
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

	body, _ := json.Marshal(map[string]any{
		"protocol_version": 1,
		"command_id":       uuid.NewString(),
		"type":             realtime.CommandRoundSpectateSelect,
		"payload":          map[string]any{"target_game_player_id": uuid.NewString()},
	})
	if err := conn.Write(context.Background(), websocket.MessageText, body); err != nil {
		t.Fatal(err)
	}
	evt := readEvent(t, conn)
	if evt.Type != realtime.EventChannelError {
		t.Fatalf("type = %s want channel.error", evt.Type)
	}
	var payload map[string]any
	_ = json.Unmarshal(evt.Payload, &payload)
	if payload["code"] != realtime.CodeSpectateForbidden {
		t.Fatalf("code = %v want spectate_forbidden", payload["code"])
	}
	// Privacy-safe message: no roster details.
	msg, _ := payload["message"].(string)
	if strings.Contains(strings.ToLower(msg), "opponent") || strings.Contains(strings.ToLower(msg), "unsubmitted") {
		t.Fatalf("message leaks policy detail: %q", msg)
	}
}

func TestViewUpdateFansOutOnlyToAuthorizedSpectators(t *testing.T) {
	matchID := uuid.New()
	roundID := uuid.New()
	source := uuid.New()   // navigating player
	viewer := uuid.New()   // submitted spectator
	outsider := uuid.New() // must not receive view
	team1, team2 := 1, 2

	tickets := newMemTicketStore()
	tokSource, _, _ := tickets.Issue(context.Background(), source, realtime.ChannelKindMatch, matchID.String(), time.Minute)
	tokViewer, _, _ := tickets.Issue(context.Background(), viewer, realtime.ChannelKindMatch, matchID.String(), time.Minute)
	tokOut, _, _ := tickets.Issue(context.Background(), outsider, realtime.ChannelKindMatch, matchID.String(), time.Minute)

	cmds := newMemCommands()
	cmds.viewRecipients[source] = []uuid.UUID{viewer} // only viewer

	hub := realtime.NewHubWithQueueSize(16)
	h := realtime.NewMatchHandler(
		hub,
		tickets,
		authByUser{slots: map[uuid.UUID]int{source: team2, viewer: team1, outsider: team1}},
		fixedSnapshot{version: 1},
		cmds,
		nil,
		realtime.MatchHandlerConfig{AllowedOrigins: []string{"http://localhost:*"}},
		nil,
		nil,
	)
	srv := startMatchServer(t, h)
	connSource := dialMatch(t, srv, matchID.String(), tokSource)
	connViewer := dialMatch(t, srv, matchID.String(), tokViewer)
	connOut := dialMatch(t, srv, matchID.String(), tokOut)
	_ = readEvent(t, connSource)
	_ = readEvent(t, connViewer)
	_ = readEvent(t, connOut)

	body, _ := json.Marshal(map[string]any{
		"protocol_version": 1,
		"command_id":       uuid.NewString(),
		"type":             realtime.CommandRoundViewUpdate,
		"payload": map[string]any{
			"round_id":    roundID.String(),
			"panorama_id": "pano-1",
			"heading":     45.0,
			"pitch":       0.0,
			"zoom":        1.0,
		},
	})
	if err := connSource.Write(context.Background(), websocket.MessageText, body); err != nil {
		t.Fatal(err)
	}

	// Source gets accepted.
	var gotAccepted bool
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !gotAccepted {
		evt := readEvent(t, connSource)
		if evt.Type == realtime.EventCommandAccepted {
			gotAccepted = true
		}
	}
	if !gotAccepted {
		t.Fatal("source missing accepted")
	}

	// Viewer receives view_changed with provider-safe fields only.
	var gotView bool
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && !gotView {
		evt := readEvent(t, connViewer)
		if evt.Type == realtime.EventRoundViewChanged {
			gotView = true
			raw := string(evt.Payload)
			if !strings.Contains(raw, "panorama_id") || !strings.Contains(raw, "heading") {
				t.Fatalf("missing scene fields: %s", raw)
			}
			for _, bad := range []string{"latitude", "longitude", "marker", "guess", "score", "answer", "cursor"} {
				if strings.Contains(raw, `"`+bad+`"`) {
					t.Fatalf("view event leaked %q: %s", bad, raw)
				}
			}
		}
	}
	if !gotView {
		t.Fatal("authorized spectator did not receive view_changed")
	}

	// Outsider must not receive view_changed.
	octx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, data, err := connOut.Read(octx)
	if err == nil {
		var evt realtime.Event
		_ = json.Unmarshal(data, &evt)
		if evt.Type == realtime.EventRoundViewChanged {
			t.Fatal("unauthorized client received view_changed")
		}
	}
}

func TestViewUpdateRejectsForbiddenFields(t *testing.T) {
	matchID := uuid.New()
	roundID := uuid.New()
	userID := uuid.New()
	team := 1
	tickets := newMemTicketStore()
	token, _, _ := tickets.Issue(context.Background(), userID, realtime.ChannelKindMatch, matchID.String(), time.Minute)
	cmds := newMemCommands()

	h := realtime.NewMatchHandler(
		realtime.NewHubWithQueueSize(4),
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

	body, _ := json.Marshal(map[string]any{
		"protocol_version": 1,
		"command_id":       uuid.NewString(),
		"type":             realtime.CommandRoundViewUpdate,
		"payload": map[string]any{
			"round_id":  roundID.String(),
			"heading":   1.0,
			"pitch":     0.0,
			"zoom":      1.0,
			"latitude":  12.3,
			"longitude": 45.6,
			"guess":     true,
		},
	})
	if err := conn.Write(context.Background(), websocket.MessageText, body); err != nil {
		t.Fatal(err)
	}
	evt := readEvent(t, conn)
	if evt.Type != realtime.EventChannelError {
		t.Fatalf("type = %s", evt.Type)
	}
	if len(cmds.views) != 0 {
		t.Fatal("forbidden view must not be stored")
	}
}

func TestSpectateSelectPreferredTarget(t *testing.T) {
	matchID := uuid.New()
	roundID := uuid.New()
	viewer := uuid.New()
	targetA := uuid.New()
	targetB := uuid.New()
	team := 1
	tickets := newMemTicketStore()
	token, _, _ := tickets.Issue(context.Background(), viewer, realtime.ChannelKindMatch, matchID.String(), time.Minute)

	cmds := newMemCommands()
	cmds.roundID = roundID
	cmds.format = "squad"
	cmds.submitted[viewer] = true
	cmds.allowedByUser[viewer] = []uuid.UUID{targetA, targetB}

	h := realtime.NewMatchHandler(
		realtime.NewHubWithQueueSize(8),
		tickets,
		fixedAuthorizer{members: map[string]realtime.ChannelMembership{
			matchID.String(): {UserID: viewer, TeamSlot: &team},
		}},
		fixedSnapshot{version: 1},
		cmds,
		nil,
		realtime.MatchHandlerConfig{AllowedOrigins: []string{"http://localhost:*"}},
		nil,
		nil,
	)
	srv := startMatchServer(t, h)
	conn := dialMatch(t, srv, matchID.String(), token)
	_ = readEvent(t, conn)

	body, _ := json.Marshal(map[string]any{
		"protocol_version": 1,
		"command_id":       uuid.NewString(),
		"type":             realtime.CommandRoundSpectateSelect,
		"payload":          map[string]any{"target_game_player_id": targetB.String()},
	})
	if err := conn.Write(context.Background(), websocket.MessageText, body); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		evt := readEvent(t, conn)
		if evt.Type == realtime.EventCommandAccepted {
			var payload map[string]any
			_ = json.Unmarshal(evt.Payload, &payload)
			if payload["target_game_player_id"] != targetB.String() {
				t.Fatalf("selected = %v want %s", payload["target_game_player_id"], targetB)
			}
			if payload["fallback"] == true {
				t.Fatal("preferred allowed target should not fallback")
			}
			return
		}
		if evt.Type == realtime.EventChannelError {
			t.Fatalf("error: %s", string(evt.Payload))
		}
	}
	t.Fatal("timeout waiting for accepted")
}

func TestViewUnchangedStillAccepted(t *testing.T) {
	matchID := uuid.New()
	roundID := uuid.New()
	userID := uuid.New()
	team := 1
	tickets := newMemTicketStore()
	token, _, _ := tickets.Issue(context.Background(), userID, realtime.ChannelKindMatch, matchID.String(), time.Minute)
	cmds := newMemCommands()
	cmds.forceUnchanged = true
	cmds.version = 7

	h := realtime.NewMatchHandler(
		realtime.NewHubWithQueueSize(4),
		tickets,
		fixedAuthorizer{members: map[string]realtime.ChannelMembership{
			matchID.String(): {UserID: userID, TeamSlot: &team},
		}},
		fixedSnapshot{version: 7},
		cmds,
		nil,
		realtime.MatchHandlerConfig{AllowedOrigins: []string{"http://localhost:*"}},
		nil,
		nil,
	)
	srv := startMatchServer(t, h)
	conn := dialMatch(t, srv, matchID.String(), token)
	_ = readEvent(t, conn)

	body, _ := json.Marshal(map[string]any{
		"protocol_version": 1,
		"command_id":       uuid.NewString(),
		"type":             realtime.CommandRoundViewUpdate,
		"payload": map[string]any{
			"round_id": roundID.String(),
			"heading":  1.0, "pitch": 0.0, "zoom": 1.0,
		},
	})
	if err := conn.Write(context.Background(), websocket.MessageText, body); err != nil {
		t.Fatal(err)
	}
	evt := readEvent(t, conn)
	if evt.Type != realtime.EventCommandAccepted {
		t.Fatalf("type = %s want accepted", evt.Type)
	}
	var payload map[string]any
	_ = json.Unmarshal(evt.Payload, &payload)
	if payload["changed"] != false {
		t.Fatalf("changed = %v", payload["changed"])
	}
}
