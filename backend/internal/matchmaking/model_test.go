package matchmaking_test

import (
	"errors"
	"testing"

	"github.com/raven/geoguess/backend/internal/matchmaking"
)

func TestSupportedModeCanonicalAndLegacy(t *testing.T) {
	t.Parallel()

	for _, mode := range matchmaking.CanonicalModes() {
		if !matchmaking.SupportedMode(mode) {
			t.Fatalf("canonical mode %q should be supported", mode)
		}
	}
	if !matchmaking.SupportedMode(matchmaking.ModeRankedStandard) {
		t.Fatal("ranked_standard legacy alias should be supported")
	}
	if matchmaking.SupportedMode("") {
		t.Fatal("empty mode must not be supported")
	}
	if matchmaking.SupportedMode("ranked_season") {
		t.Fatal("unknown mode must not be supported")
	}
	if matchmaking.SupportedMode("solo") {
		t.Fatal("bare format must not be supported as mode")
	}
}

func TestNormalizeModeLegacyAlias(t *testing.T) {
	t.Parallel()

	if got := matchmaking.NormalizeMode(matchmaking.ModeRankedStandard); got != matchmaking.ModeRankedSolo {
		t.Fatalf("NormalizeMode(ranked_standard) = %q, want %q", got, matchmaking.ModeRankedSolo)
	}
	if got := matchmaking.NormalizeMode(matchmaking.ModeCasualDuo); got != matchmaking.ModeCasualDuo {
		t.Fatalf("NormalizeMode(casual_duo) = %q, want self", got)
	}
	if got := matchmaking.NormalizeMode("nope"); got != "" {
		t.Fatalf("NormalizeMode(unknown) = %q, want empty", got)
	}
}

func TestParseModeAndParts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		mode     string
		playlist string
		format   string
		teamSize int
		canon    string
	}{
		{matchmaking.ModeCasualSolo, matchmaking.PlaylistCasual, matchmaking.FormatSolo, 1, matchmaking.ModeCasualSolo},
		{matchmaking.ModeCasualDuo, matchmaking.PlaylistCasual, matchmaking.FormatDuo, 2, matchmaking.ModeCasualDuo},
		{matchmaking.ModeCasualSquad, matchmaking.PlaylistCasual, matchmaking.FormatSquad, 4, matchmaking.ModeCasualSquad},
		{matchmaking.ModeRankedSolo, matchmaking.PlaylistRanked, matchmaking.FormatSolo, 1, matchmaking.ModeRankedSolo},
		{matchmaking.ModeRankedDuo, matchmaking.PlaylistRanked, matchmaking.FormatDuo, 2, matchmaking.ModeRankedDuo},
		{matchmaking.ModeRankedSquad, matchmaking.PlaylistRanked, matchmaking.FormatSquad, 4, matchmaking.ModeRankedSquad},
		{matchmaking.ModeRankedStandard, matchmaking.PlaylistRanked, matchmaking.FormatSolo, 1, matchmaking.ModeRankedSolo},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.mode, func(t *testing.T) {
			t.Parallel()
			parts, err := matchmaking.ParseMode(tc.mode)
			if err != nil {
				t.Fatalf("ParseMode: %v", err)
			}
			if parts.Playlist != tc.playlist || parts.Format != tc.format || parts.TeamSize != tc.teamSize {
				t.Fatalf("parts = %+v, want playlist=%s format=%s team=%d", parts, tc.playlist, tc.format, tc.teamSize)
			}
			if parts.Canonical != tc.canon {
				t.Fatalf("canonical = %q, want %q", parts.Canonical, tc.canon)
			}
			if parts.Raw != tc.mode {
				t.Fatalf("raw = %q, want %q", parts.Raw, tc.mode)
			}

			fromParts, err := matchmaking.ModeFromParts(tc.playlist, tc.format)
			if err != nil {
				t.Fatalf("ModeFromParts: %v", err)
			}
			if fromParts != tc.canon {
				t.Fatalf("ModeFromParts = %q, want %q", fromParts, tc.canon)
			}
		})
	}
}

func TestParseModeRejectsUnknown(t *testing.T) {
	t.Parallel()

	_, err := matchmaking.ParseMode("ranked_season")
	if !errors.Is(err, matchmaking.ErrUnsupportedMode) {
		t.Fatalf("err = %v, want ErrUnsupportedMode", err)
	}
}

func TestTeamSizeValidation(t *testing.T) {
	t.Parallel()

	if size, err := matchmaking.TeamSizeForFormat(matchmaking.FormatDuo); err != nil || size != 2 {
		t.Fatalf("duo size = %d, %v", size, err)
	}
	if err := matchmaking.ValidateTeamSize(matchmaking.FormatSquad, 4); err != nil {
		t.Fatalf("valid squad size: %v", err)
	}
	if err := matchmaking.ValidateTeamSize(matchmaking.FormatSolo, 2); err == nil {
		t.Fatal("expected team size mismatch error")
	}
	if _, err := matchmaking.TeamSizeForFormat("party"); err == nil {
		t.Fatal("expected unknown format error")
	}
}

func TestModeClassificationHelpers(t *testing.T) {
	t.Parallel()

	if !matchmaking.IsCasual(matchmaking.ModeCasualSquad) {
		t.Fatal("casual_squad should be casual")
	}
	if matchmaking.IsRanked(matchmaking.ModeCasualSolo) {
		t.Fatal("casual_solo should not be ranked")
	}
	if !matchmaking.IsRanked(matchmaking.ModeRankedStandard) {
		t.Fatal("ranked_standard should be ranked")
	}
	if !matchmaking.IsTeamMode(matchmaking.ModeRankedDuo) {
		t.Fatal("ranked_duo should be team mode")
	}
	if matchmaking.IsTeamMode(matchmaking.ModeCasualSolo) {
		t.Fatal("casual_solo should not be team mode")
	}
	if !matchmaking.IsPlaylist(matchmaking.PlaylistRanked) || !matchmaking.IsFormat(matchmaking.FormatSquad) {
		t.Fatal("playlist/format helpers failed")
	}
}

func TestMatchResultAndAbandonEnums(t *testing.T) {
	t.Parallel()

	for _, result := range []string{
		matchmaking.MatchResultTeamOneWin,
		matchmaking.MatchResultTeamTwoWin,
		matchmaking.MatchResultDraw,
		matchmaking.MatchResultForfeit,
		matchmaking.MatchResultAbandoned,
		matchmaking.MatchResultCancelled,
	} {
		if !matchmaking.IsTerminalMatchResult(result) {
			t.Fatalf("result %q should be recognized", result)
		}
	}
	if matchmaking.IsTerminalMatchResult("win") {
		t.Fatal("unknown result must be rejected")
	}

	for _, reason := range []string{
		matchmaking.AbandonReasonExplicitLeave,
		matchmaking.AbandonReasonDisconnectTimeout,
		matchmaking.AbandonReasonAccountIneligible,
	} {
		if !matchmaking.IsAbandonReason(reason) {
			t.Fatalf("reason %q should be recognized", reason)
		}
	}
	if matchmaking.IsAbandonReason("rage_quit") {
		t.Fatal("unknown abandon reason must be rejected")
	}
}

func TestIsTerminalMatchPreserved(t *testing.T) {
	t.Parallel()

	if !matchmaking.IsTerminalMatch(matchmaking.MatchStatusCompleted) {
		t.Fatal("completed should be terminal")
	}
	if matchmaking.IsTerminalMatch(matchmaking.MatchStatusActive) {
		t.Fatal("active should not be terminal")
	}
}
