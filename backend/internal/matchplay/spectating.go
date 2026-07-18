package matchplay

import (
	"math"
	"strings"

	"github.com/google/uuid"
)

// Material-change thresholds for provider-safe scene comparison.
// Small jitter below these values is treated as unchanged (no version bump / no fanout).
const (
	SceneHeadingEpsilon = 0.5  // degrees
	ScenePitchEpsilon   = 0.5  // degrees
	SceneZoomEpsilon    = 0.05 // provider zoom units
)

// Forbidden scene field keys must never appear in view payloads or redis scene JSON.
// Kept as a pure policy list shared by HTTP/WS validation and redis write guards.
var forbiddenSceneFieldKeys = []string{
	"latitude", "longitude", "lat", "lng",
	"marker", "markers", "map", "cursor",
	"guess", "guesses", "score", "scores",
	"answer", "actual_location", "distance", "distance_meters",
	"accuracy_score", "speed_bonus", "total_score",
}

// SpectatePlayer is one roster row for pure spectating policy evaluation.
type SpectatePlayer struct {
	GamePlayerID uuid.UUID
	UserID       uuid.UUID
	TeamSlot     int
	Submitted    bool
	// Disconnected is true when the player is in reconnect-grace / offline.
	Disconnected bool
	// Abandoned is true after ranked/casual leave/forfeit.
	Abandoned bool
	// Left is true when the player permanently left the match.
	Left bool
}

// SpectatePolicyInput is the pure input for allowed-target evaluation.
type SpectatePolicyInput struct {
	// Format is solo|duo|squad (empty treated via TeamSize when needed).
	Format string
	// TeamSize is 1 for Solo, 2 for Duo, 4 for Squad; used when Format is empty.
	TeamSize int
	// RoundActive is false after round close/reveal — spectating ends.
	RoundActive bool
	Viewer      SpectatePlayer
	Players     []SpectatePlayer
}

// SceneView is the provider-safe panorama/image scene (no map/marker/guess fields).
type SceneView struct {
	PanoramaID string
	Heading    float64
	Pitch      float64
	Zoom       float64
}

// IsSoloFormat reports whether the match is 1v1 Solo.
func IsSoloFormat(format string, teamSize int) bool {
	f := strings.ToLower(strings.TrimSpace(format))
	if f == FormatSolo || f == "1v1" {
		return true
	}
	if f == FormatDuo || f == FormatSquad {
		return false
	}
	return teamSize == 1
}

// CanViewerSpectate reports whether the viewer is allowed to enter spectate mode.
// Unsubmitted viewers and closed rounds cannot spectate.
func CanViewerSpectate(viewerSubmitted, roundActive bool) bool {
	return viewerSubmitted && roundActive
}

// AllowedSpectateTargets returns game_player_ids the viewer may currently watch.
//
// Policy:
//   - Unsubmitted viewers → none
//   - Round closed → none
//   - Solo → unsubmitted, active (not disconnected/abandoned/left) opponents
//   - Duo/Squad → unsubmitted, active same-team teammates (never opponents)
//   - Self is never a target
//
// Submitted or disconnected targets are excluded so clients fall back automatically.
func AllowedSpectateTargets(in SpectatePolicyInput) []uuid.UUID {
	if !CanViewerSpectate(in.Viewer.Submitted, in.RoundActive) {
		return nil
	}
	if in.Viewer.GamePlayerID == uuid.Nil && in.Viewer.UserID == uuid.Nil {
		return nil
	}

	solo := IsSoloFormat(in.Format, in.TeamSize)
	out := make([]uuid.UUID, 0, len(in.Players))
	for _, p := range in.Players {
		if !isEligibleSpectateTarget(in.Viewer, p, solo) {
			continue
		}
		out = append(out, p.GamePlayerID)
	}
	return out
}

func isEligibleSpectateTarget(viewer, target SpectatePlayer, solo bool) bool {
	if target.GamePlayerID == uuid.Nil {
		return false
	}
	// Never spectate self.
	if viewer.GamePlayerID != uuid.Nil && target.GamePlayerID == viewer.GamePlayerID {
		return false
	}
	if viewer.UserID != uuid.Nil && target.UserID == viewer.UserID {
		return false
	}
	// Submitted targets end spectate for that target (fallback to another).
	if target.Submitted {
		return false
	}
	// Disconnected / abandoned / left are not "active" navigable targets.
	if target.Disconnected || target.Abandoned || target.Left {
		return false
	}
	if solo {
		// Solo: only the opponent (opposite team).
		return target.TeamSlot != 0 && viewer.TeamSlot != 0 && target.TeamSlot != viewer.TeamSlot
	}
	// Duo/Squad: same team only — never opponents.
	return target.TeamSlot != 0 && target.TeamSlot == viewer.TeamSlot
}

// SelectSpectateTarget picks preferred when still allowed; otherwise the first allowed
// target (automatic fallback). Returns uuid.Nil when no target is available.
// fallback is true when preferred was empty/invalid and a different target was chosen,
// or when preferred became invalid and selection moved.
func SelectSpectateTarget(allowed []uuid.UUID, preferred uuid.UUID) (selected uuid.UUID, fallback bool) {
	if len(allowed) == 0 {
		return uuid.Nil, preferred != uuid.Nil
	}
	if preferred != uuid.Nil {
		for _, id := range allowed {
			if id == preferred {
				return preferred, false
			}
		}
		// Preferred no longer allowed — fall back.
		return allowed[0], true
	}
	return allowed[0], true
}

// SpectatorsOf returns user IDs currently authorized to receive sourceUserID's view updates.
// Delivery is only to submitted viewers for whom source is an allowed target.
func SpectatorsOf(in SpectatePolicyInput, sourceUserID uuid.UUID) []uuid.UUID {
	if sourceUserID == uuid.Nil || !in.RoundActive {
		return nil
	}
	var source SpectatePlayer
	found := false
	for _, p := range in.Players {
		if p.UserID == sourceUserID {
			source = p
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	out := make([]uuid.UUID, 0)
	for _, viewer := range in.Players {
		if viewer.UserID == sourceUserID {
			continue
		}
		if !viewer.Submitted {
			continue
		}
		// Evaluate allowed targets from this viewer's perspective.
		viewerInput := in
		viewerInput.Viewer = viewer
		allowed := AllowedSpectateTargets(viewerInput)
		for _, gp := range allowed {
			if gp == source.GamePlayerID {
				out = append(out, viewer.UserID)
				break
			}
		}
	}
	return out
}

// IsForbiddenSceneField reports whether a JSON key is banned from scene state.
func IsForbiddenSceneField(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	for _, f := range forbiddenSceneFieldKeys {
		if k == f {
			return true
		}
	}
	return false
}

// ForbiddenSceneFieldKeys returns a copy of banned scene keys.
func ForbiddenSceneFieldKeys() []string {
	out := make([]string, len(forbiddenSceneFieldKeys))
	copy(out, forbiddenSceneFieldKeys)
	return out
}

// ValidateProviderSafeScene rejects forbidden keys and out-of-range heading/pitch/zoom.
// Allowed keys: panorama_id (optional string), heading, pitch, zoom, round_id (command only),
// and user_id/version when present from server storage.
func ValidateProviderSafeScene(fields map[string]any) error {
	if fields == nil {
		return ErrInvalidRequest
	}
	for key := range fields {
		k := strings.ToLower(strings.TrimSpace(key))
		switch k {
		case "panorama_id", "heading", "pitch", "zoom",
			"round_id", "user_id", "version", "v",
			"image_pan_x", "image_pan_y", "image_zoom":
			// provider-safe / command envelope fields
			continue
		default:
			// Forbidden and unknown keys are rejected to prevent private field leakage.
			return ErrInvalidRequest
		}
	}
	if h, ok := asFloat(fields["heading"]); ok {
		if h < 0 || h > 360 {
			return ErrInvalidRequest
		}
	}
	if p, ok := asFloat(fields["pitch"]); ok {
		if p < -90 || p > 90 {
			return ErrInvalidRequest
		}
	}
	if z, ok := asFloat(fields["zoom"]); ok {
		if z < 0 || z > 10 {
			return ErrInvalidRequest
		}
	}
	return nil
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	default:
		return 0, false
	}
}

// SceneMateriallyChanged reports whether next differs from prev enough to fan out.
func SceneMateriallyChanged(prev, next SceneView) bool {
	if strings.TrimSpace(prev.PanoramaID) != strings.TrimSpace(next.PanoramaID) {
		return true
	}
	if math.Abs(prev.Heading-next.Heading) >= SceneHeadingEpsilon {
		return true
	}
	if math.Abs(prev.Pitch-next.Pitch) >= ScenePitchEpsilon {
		return true
	}
	if math.Abs(prev.Zoom-next.Zoom) >= SceneZoomEpsilon {
		return true
	}
	return false
}

// BuildSpectatePolicyInput constructs policy input from a snapshot bundle for viewer.
func BuildSpectatePolicyInput(viewer MatchParticipant, bundle *SnapshotBundle, viewerSubmitted bool) SpectatePolicyInput {
	if bundle == nil {
		return SpectatePolicyInput{}
	}
	submitted := bundle.SubmittedIDs
	if submitted == nil {
		submitted = map[uuid.UUID]bool{}
	}
	playersByID := make(map[uuid.UUID]GamePlayerRow, len(bundle.Players))
	for _, gp := range bundle.Players {
		playersByID[gp.ID] = gp
	}

	roundActive := false
	if bundle.CurrentRound != nil {
		st := strings.ToLower(bundle.CurrentRound.Status)
		roundActive = st == "active" || st == "in_progress" || st == "playing"
	}

	players := make([]SpectatePlayer, 0, len(bundle.Participants))
	for _, p := range bundle.Participants {
		sp := SpectatePlayer{
			GamePlayerID: p.GamePlayerID,
			UserID:       p.UserID,
			TeamSlot:     p.TeamSlot,
			Submitted:    submitted[p.GamePlayerID],
			Abandoned:    p.AbandonedAt != nil,
		}
		if gp, ok := playersByID[p.GamePlayerID]; ok {
			status := projectPlayerStatus(p, gp)
			sp.Disconnected = status == PlayerStatusDisconnected
			sp.Left = status == PlayerStatusLeft
		} else if p.AbandonedAt != nil {
			sp.Left = true
		}
		players = append(players, sp)
	}

	return SpectatePolicyInput{
		Format:      formatOf(bundle.Match),
		TeamSize:    teamSizeOf(bundle.Match),
		RoundActive: roundActive,
		Viewer: SpectatePlayer{
			GamePlayerID: viewer.GamePlayerID,
			UserID:       viewer.UserID,
			TeamSlot:     viewer.TeamSlot,
			Submitted:    viewerSubmitted,
			Abandoned:    viewer.AbandonedAt != nil,
		},
		Players: players,
	}
}
