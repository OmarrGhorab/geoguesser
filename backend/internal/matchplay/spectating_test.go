package matchplay_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/raven/geoguess/backend/internal/matchplay"
)

func TestAllowedSpectateTargetsTable(t *testing.T) {
	t.Parallel()

	viewerID := uuid.MustParse("00000000-0000-4000-8000-0000000000a1")
	viewerGP := uuid.MustParse("00000000-0000-4000-8000-0000000000b1")
	oppID := uuid.MustParse("00000000-0000-4000-8000-0000000000a2")
	oppGP := uuid.MustParse("00000000-0000-4000-8000-0000000000b2")
	mateID := uuid.MustParse("00000000-0000-4000-8000-0000000000a3")
	mateGP := uuid.MustParse("00000000-0000-4000-8000-0000000000b3")
	foeID := uuid.MustParse("00000000-0000-4000-8000-0000000000a4")
	foeGP := uuid.MustParse("00000000-0000-4000-8000-0000000000b4")

	viewer := matchplay.SpectatePlayer{
		GamePlayerID: viewerGP, UserID: viewerID, TeamSlot: 1, Submitted: true,
	}
	opponent := matchplay.SpectatePlayer{
		GamePlayerID: oppGP, UserID: oppID, TeamSlot: 2, Submitted: false,
	}
	teammate := matchplay.SpectatePlayer{
		GamePlayerID: mateGP, UserID: mateID, TeamSlot: 1, Submitted: false,
	}
	foe := matchplay.SpectatePlayer{
		GamePlayerID: foeGP, UserID: foeID, TeamSlot: 2, Submitted: false,
	}

	tests := []struct {
		name    string
		input   matchplay.SpectatePolicyInput
		wantLen int
		wantIDs []uuid.UUID
	}{
		{
			name: "solo submitted viewer may spectate unsubmitted opponent",
			input: matchplay.SpectatePolicyInput{
				Format: matchplay.FormatSolo, TeamSize: 1, RoundActive: true,
				Viewer:  viewer,
				Players: []matchplay.SpectatePlayer{viewer, opponent},
			},
			wantLen: 1,
			wantIDs: []uuid.UUID{oppGP},
		},
		{
			name: "duo submitted viewer may spectate unsubmitted teammate only",
			input: matchplay.SpectatePolicyInput{
				Format: matchplay.FormatDuo, TeamSize: 2, RoundActive: true,
				Viewer: viewer,
				Players: []matchplay.SpectatePlayer{viewer, teammate, foe, {
					GamePlayerID: uuid.New(), UserID: uuid.New(), TeamSlot: 2, Submitted: false,
				}},
			},
			wantLen: 1,
			wantIDs: []uuid.UUID{mateGP},
		},
		{
			name: "squad submitted viewer may spectate multiple unsubmitted teammates not opponents",
			input: matchplay.SpectatePolicyInput{
				Format: matchplay.FormatSquad, TeamSize: 4, RoundActive: true,
				Viewer: viewer,
				Players: []matchplay.SpectatePlayer{
					viewer, teammate,
					{GamePlayerID: uuid.MustParse("00000000-0000-4000-8000-0000000000b5"), UserID: uuid.New(), TeamSlot: 1, Submitted: false},
					foe,
					{GamePlayerID: uuid.New(), UserID: uuid.New(), TeamSlot: 2, Submitted: false},
				},
			},
			wantLen: 2,
			wantIDs: []uuid.UUID{mateGP, uuid.MustParse("00000000-0000-4000-8000-0000000000b5")},
		},
		{
			name: "unsubmitted viewer cannot spectate",
			input: matchplay.SpectatePolicyInput{
				Format: matchplay.FormatSolo, TeamSize: 1, RoundActive: true,
				Viewer: matchplay.SpectatePlayer{
					GamePlayerID: viewerGP, UserID: viewerID, TeamSlot: 1, Submitted: false,
				},
				Players: []matchplay.SpectatePlayer{
					{GamePlayerID: viewerGP, UserID: viewerID, TeamSlot: 1, Submitted: false},
					opponent,
				},
			},
			wantLen: 0,
		},
		{
			name: "submitted target excluded for solo",
			input: matchplay.SpectatePolicyInput{
				Format: matchplay.FormatSolo, TeamSize: 1, RoundActive: true,
				Viewer: viewer,
				Players: []matchplay.SpectatePlayer{viewer, {
					GamePlayerID: oppGP, UserID: oppID, TeamSlot: 2, Submitted: true,
				}},
			},
			wantLen: 0,
		},
		{
			name: "disconnected target excluded",
			input: matchplay.SpectatePolicyInput{
				Format: matchplay.FormatDuo, TeamSize: 2, RoundActive: true,
				Viewer: viewer,
				Players: []matchplay.SpectatePlayer{viewer, {
					GamePlayerID: mateGP, UserID: mateID, TeamSlot: 1, Submitted: false, Disconnected: true,
				}},
			},
			wantLen: 0,
		},
		{
			name: "abandoned target excluded",
			input: matchplay.SpectatePolicyInput{
				Format: matchplay.FormatDuo, TeamSize: 2, RoundActive: true,
				Viewer: viewer,
				Players: []matchplay.SpectatePlayer{viewer, {
					GamePlayerID: mateGP, UserID: mateID, TeamSlot: 1, Submitted: false, Abandoned: true,
				}},
			},
			wantLen: 0,
		},
		{
			name: "round close ends spectate",
			input: matchplay.SpectatePolicyInput{
				Format: matchplay.FormatSolo, TeamSize: 1, RoundActive: false,
				Viewer:  viewer,
				Players: []matchplay.SpectatePlayer{viewer, opponent},
			},
			wantLen: 0,
		},
		{
			name: "team forbidden from spectating opponents",
			input: matchplay.SpectatePolicyInput{
				Format: matchplay.FormatDuo, TeamSize: 2, RoundActive: true,
				Viewer:  viewer,
				Players: []matchplay.SpectatePlayer{viewer, foe},
			},
			wantLen: 0,
		},
		{
			name: "self never allowed",
			input: matchplay.SpectatePolicyInput{
				Format: matchplay.FormatSolo, TeamSize: 1, RoundActive: true,
				Viewer:  viewer,
				Players: []matchplay.SpectatePlayer{viewer},
			},
			wantLen: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := matchplay.AllowedSpectateTargets(tc.input)
			if len(got) != tc.wantLen {
				t.Fatalf("len = %d want %d; got %v", len(got), tc.wantLen, got)
			}
			if tc.wantIDs != nil {
				set := map[uuid.UUID]bool{}
				for _, id := range got {
					set[id] = true
				}
				for _, id := range tc.wantIDs {
					if !set[id] {
						t.Fatalf("missing target %s in %v", id, got)
					}
				}
			}
		})
	}
}

func TestSelectSpectateTargetFallback(t *testing.T) {
	t.Parallel()
	a := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	b := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	c := uuid.MustParse("00000000-0000-4000-8000-000000000003")

	selected, fallback := matchplay.SelectSpectateTarget([]uuid.UUID{a, b}, a)
	if selected != a || fallback {
		t.Fatalf("preferred kept: selected=%s fallback=%v", selected, fallback)
	}

	selected, fallback = matchplay.SelectSpectateTarget([]uuid.UUID{a, b}, c)
	if selected != a || !fallback {
		t.Fatalf("invalid preferred fallback: selected=%s fallback=%v", selected, fallback)
	}

	selected, fallback = matchplay.SelectSpectateTarget([]uuid.UUID{a, b}, uuid.Nil)
	if selected != a || !fallback {
		t.Fatalf("empty preferred fallback: selected=%s fallback=%v", selected, fallback)
	}

	selected, fallback = matchplay.SelectSpectateTarget(nil, a)
	if selected != uuid.Nil || !fallback {
		t.Fatalf("no allowed: selected=%s fallback=%v", selected, fallback)
	}
}

func TestSpectatorsOfOnlyAuthorizedSubmittedViewers(t *testing.T) {
	t.Parallel()
	source := matchplay.SpectatePlayer{
		GamePlayerID: uuid.MustParse("00000000-0000-4000-8000-0000000000c1"),
		UserID:       uuid.MustParse("00000000-0000-4000-8000-0000000000d1"),
		TeamSlot:     2, Submitted: false,
	}
	// Submitted solo opponent may receive source's view.
	viewer := matchplay.SpectatePlayer{
		GamePlayerID: uuid.MustParse("00000000-0000-4000-8000-0000000000c2"),
		UserID:       uuid.MustParse("00000000-0000-4000-8000-0000000000d2"),
		TeamSlot:     1, Submitted: true,
	}
	// Unsubmitted third party must not receive.
	other := matchplay.SpectatePlayer{
		GamePlayerID: uuid.MustParse("00000000-0000-4000-8000-0000000000c3"),
		UserID:       uuid.MustParse("00000000-0000-4000-8000-0000000000d3"),
		TeamSlot:     1, Submitted: false,
	}

	in := matchplay.SpectatePolicyInput{
		Format: matchplay.FormatSolo, TeamSize: 1, RoundActive: true,
		// Viewer field unused by SpectatorsOf (recomputed per player).
		Players: []matchplay.SpectatePlayer{source, viewer, other},
	}
	got := matchplay.SpectatorsOf(in, source.UserID)
	if len(got) != 1 || got[0] != viewer.UserID {
		t.Fatalf("spectators = %v, want [%s]", got, viewer.UserID)
	}

	// Duo: teammate submitted receives teammate source; opponent submitted does not.
	mateSource := matchplay.SpectatePlayer{
		GamePlayerID: uuid.New(), UserID: uuid.New(), TeamSlot: 1, Submitted: false,
	}
	mateViewer := matchplay.SpectatePlayer{
		GamePlayerID: uuid.New(), UserID: uuid.New(), TeamSlot: 1, Submitted: true,
	}
	oppViewer := matchplay.SpectatePlayer{
		GamePlayerID: uuid.New(), UserID: uuid.New(), TeamSlot: 2, Submitted: true,
	}
	duo := matchplay.SpectatePolicyInput{
		Format: matchplay.FormatDuo, TeamSize: 2, RoundActive: true,
		Players: []matchplay.SpectatePlayer{mateSource, mateViewer, oppViewer},
	}
	got = matchplay.SpectatorsOf(duo, mateSource.UserID)
	if len(got) != 1 || got[0] != mateViewer.UserID {
		t.Fatalf("duo spectators = %v", got)
	}
}

func TestValidateProviderSafeScene(t *testing.T) {
	t.Parallel()

	if err := matchplay.ValidateProviderSafeScene(map[string]any{
		"panorama_id": "pano-1",
		"heading":     90.0,
		"pitch":       0.0,
		"zoom":        1.0,
	}); err != nil {
		t.Fatalf("safe scene: %v", err)
	}

	for _, key := range []string{"latitude", "longitude", "marker", "map", "cursor", "guess", "score", "answer"} {
		err := matchplay.ValidateProviderSafeScene(map[string]any{
			"heading": 1.0, "pitch": 0.0, "zoom": 1.0,
			key: 12.5,
		})
		if err == nil {
			t.Fatalf("expected reject for key %q", key)
		}
	}

	if err := matchplay.ValidateProviderSafeScene(map[string]any{
		"heading": 400.0, "pitch": 0.0, "zoom": 1.0,
	}); err == nil {
		t.Fatal("expected heading range reject")
	}
}

func TestSceneMateriallyChanged(t *testing.T) {
	t.Parallel()
	base := matchplay.SceneView{PanoramaID: "p", Heading: 10, Pitch: 0, Zoom: 1}
	if matchplay.SceneMateriallyChanged(base, matchplay.SceneView{PanoramaID: "p", Heading: 10.2, Pitch: 0, Zoom: 1}) {
		t.Fatal("sub-threshold heading should not be material")
	}
	if !matchplay.SceneMateriallyChanged(base, matchplay.SceneView{PanoramaID: "p", Heading: 11, Pitch: 0, Zoom: 1}) {
		t.Fatal("heading delta should be material")
	}
	if !matchplay.SceneMateriallyChanged(base, matchplay.SceneView{PanoramaID: "other", Heading: 10, Pitch: 0, Zoom: 1}) {
		t.Fatal("panorama change should be material")
	}
}

func TestBuildSpectatePolicyInputFromBundle(t *testing.T) {
	t.Parallel()
	viewerUID := uuid.New()
	viewerGP := uuid.New()
	mateUID := uuid.New()
	mateGP := uuid.New()
	oppUID := uuid.New()
	oppGP := uuid.New()

	bundle := &matchplay.SnapshotBundle{
		Match: matchplay.Match{
			Playlist: matchplay.PlaylistCasual,
			Format:   matchplay.FormatDuo,
			TeamSize: 2,
			Status:   matchplay.MatchStatusActive,
		},
		CurrentRound: &matchplay.RoundRow{Status: "active"},
		SubmittedIDs: map[uuid.UUID]bool{viewerGP: true},
		Participants: []matchplay.MatchParticipant{
			{UserID: viewerUID, GamePlayerID: viewerGP, TeamSlot: 1, Status: matchplay.ParticipantStatusActive},
			{UserID: mateUID, GamePlayerID: mateGP, TeamSlot: 1, Status: matchplay.ParticipantStatusActive},
			{UserID: oppUID, GamePlayerID: oppGP, TeamSlot: 2, Status: matchplay.ParticipantStatusActive},
		},
		Players: []matchplay.GamePlayerRow{
			{ID: viewerGP, Status: matchplay.PlayerStatusActive},
			{ID: mateGP, Status: matchplay.PlayerStatusActive},
			{ID: oppGP, Status: matchplay.PlayerStatusActive},
		},
	}
	in := matchplay.BuildSpectatePolicyInput(
		matchplay.MatchParticipant{UserID: viewerUID, GamePlayerID: viewerGP, TeamSlot: 1},
		bundle,
		true,
	)
	got := matchplay.AllowedSpectateTargets(in)
	if len(got) != 1 || got[0] != mateGP {
		t.Fatalf("allowed = %v want [%s]", got, mateGP)
	}

	// Round close via completed status.
	bundle.CurrentRound.Status = "completed"
	in = matchplay.BuildSpectatePolicyInput(
		matchplay.MatchParticipant{UserID: viewerUID, GamePlayerID: viewerGP, TeamSlot: 1},
		bundle,
		true,
	)
	if got := matchplay.AllowedSpectateTargets(in); len(got) != 0 {
		t.Fatalf("closed round allowed = %v", got)
	}

	// Disconnected mate excluded.
	bundle.CurrentRound.Status = "active"
	bundle.Players[1].Status = matchplay.PlayerStatusDisconnected
	in = matchplay.BuildSpectatePolicyInput(
		matchplay.MatchParticipant{UserID: viewerUID, GamePlayerID: viewerGP, TeamSlot: 1},
		bundle,
		true,
	)
	if got := matchplay.AllowedSpectateTargets(in); len(got) != 0 {
		t.Fatalf("disconnected allowed = %v", got)
	}
}
