package matchmaking

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Canonical competitive modes for new writes.
const (
	ModeCasualSolo  = "casual_solo"
	ModeCasualDuo   = "casual_duo"
	ModeCasualSquad = "casual_squad"
	ModeRankedSolo  = "ranked_solo"
	ModeRankedDuo   = "ranked_duo"
	ModeRankedSquad = "ranked_squad"

	// ModeRankedStandard is a legacy alias for ranked solo (playlist ranked, format solo, team_size 1).
	// New writes should use ModeRankedSolo; reads continue to accept this value.
	ModeRankedStandard = "ranked_standard"
)

// Playlists partition casual vs competitive queues.
const (
	PlaylistCasual = "casual"
	PlaylistRanked = "ranked"
)

// Formats partition team sizes.
const (
	FormatSolo  = "solo"
	FormatDuo   = "duo"
	FormatSquad = "squad"
)

// Team sizes that agree with format.
const (
	TeamSizeSolo  = 1
	TeamSizeDuo   = 2
	TeamSizeSquad = 4
)

// Match lifecycle statuses (matches.status).
const (
	MatchStatusMatched       = "matched"
	MatchStatusActive        = "active"
	MatchStatusCompleted     = "completed"
	MatchStatusCancelled     = "cancelled"
	MatchStatusFailedToStart = "failed_to_start"
)

// Match result values (matches.result) set when the match is terminal.
const (
	MatchResultTeamOneWin = "team_one_win"
	MatchResultTeamTwoWin = "team_two_win"
	MatchResultDraw       = "draw"
	MatchResultForfeit    = "forfeit"
	MatchResultAbandoned  = "abandoned"
	MatchResultCancelled  = "cancelled"
)

// Abandon reasons (match_players.abandon_reason).
const (
	AbandonReasonExplicitLeave     = "explicit_leave"
	AbandonReasonDisconnectTimeout = "disconnect_timeout"
	AbandonReasonAccountIneligible = "account_ineligible"
)

// Match participant statuses (match_players.status).
const (
	ParticipantStatusAssigned  = "assigned"
	ParticipantStatusActive    = "active"
	ParticipantStatusCompleted = "completed"
	ParticipantStatusCancelled = "cancelled"
	ParticipantStatusFailed    = "failed"
)

// Ephemeral queue entry states stored in Redis.
const (
	QueueStateSearching = "searching"
	QueueStateClaimed   = "claimed"
)

// Public status values returned to clients.
const (
	PublicStatusNotQueued              = "not_queued"
	PublicStatusSearching              = "searching"
	PublicStatusMatched                = "matched"
	PublicStatusTemporarilyUnavailable = "temporarily_unavailable"
)

// ModeParts is the decomposed playlist/format/team-size view of a mode string.
type ModeParts struct {
	Playlist string
	Format   string
	TeamSize int
	// Canonical is the six-mode value (never ranked_standard).
	Canonical string
	// Raw is the input as accepted (may be ranked_standard).
	Raw string
}

// Match is the durable ranked/casual match record backed by matches.
type Match struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	FormationKey           string     `gorm:"type:text;not null;uniqueIndex"`
	GameID                 uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	Mode                   string     `gorm:"type:text;not null"`
	Playlist               string     `gorm:"type:text;not null"`
	Format                 string     `gorm:"type:text;not null"`
	TeamSize               int        `gorm:"type:smallint;not null"`
	SeasonID               *uuid.UUID `gorm:"type:uuid"`
	TeamOneScore           int        `gorm:"type:int;not null;default:0"`
	TeamTwoScore           int        `gorm:"type:int;not null;default:0"`
	WinnerTeamSlot         *int       `gorm:"type:smallint"`
	Result                 *string    `gorm:"type:text"`
	Status                 string     `gorm:"type:text;not null"`
	MatchedAt              time.Time  `gorm:"type:timestamptz;not null"`
	StartedAt              *time.Time `gorm:"type:timestamptz"`
	CompletedAt            *time.Time `gorm:"type:timestamptz"`
	ClosedAt               *time.Time `gorm:"type:timestamptz"`
	FailureCode            *string    `gorm:"type:text"`
	LastActivityAt         time.Time  `gorm:"type:timestamptz;not null"`
	ChatAccessUntil        *time.Time `gorm:"type:timestamptz"`
	ProgressionFinalizedAt *time.Time `gorm:"type:timestamptz"`
	CreatedAt              time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt              time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the database table name.
func (Match) TableName() string {
	return "matches"
}

// MatchPlayer is a durable match participant backed by match_players.
type MatchPlayer struct {
	MatchID       uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	GamePlayerID  uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	TeamSlot      int        `gorm:"type:smallint;not null"`
	PartyID       *uuid.UUID `gorm:"type:uuid"`
	Status        string     `gorm:"type:text;not null"`
	AssignedAt    time.Time  `gorm:"type:timestamptz;not null"`
	CompletedAt   *time.Time `gorm:"type:timestamptz"`
	ClosedAt      *time.Time `gorm:"type:timestamptz"`
	AbandonedAt   *time.Time `gorm:"type:timestamptz"`
	AbandonReason *string    `gorm:"type:text"`
}

// TableName returns the database table name.
func (MatchPlayer) TableName() string {
	return "match_players"
}

// ActiveAssignment is a privacy-safe durable assignment used for status recovery.
type ActiveAssignment struct {
	MatchID   uuid.UUID
	GameID    uuid.UUID
	Mode      string
	Status    string
	MatchedAt time.Time
	UserID    uuid.UUID
}

// CanonicalModes returns the six modes used for new writes.
func CanonicalModes() []string {
	return []string{
		ModeCasualSolo,
		ModeCasualDuo,
		ModeCasualSquad,
		ModeRankedSolo,
		ModeRankedDuo,
		ModeRankedSquad,
	}
}

// SupportedModes returns all accepted mode strings including the legacy alias.
func SupportedModes() []string {
	return append(CanonicalModes(), ModeRankedStandard)
}

// IsTerminalMatch reports whether a match status is immutable.
func IsTerminalMatch(status string) bool {
	switch status {
	case MatchStatusCompleted, MatchStatusCancelled, MatchStatusFailedToStart:
		return true
	default:
		return false
	}
}

// IsActiveParticipant reports whether a participant still holds an active assignment.
func IsActiveParticipant(status string) bool {
	return status == ParticipantStatusAssigned || status == ParticipantStatusActive
}

// IsTerminalMatchResult reports whether a match result value is recognized.
func IsTerminalMatchResult(result string) bool {
	switch result {
	case MatchResultTeamOneWin, MatchResultTeamTwoWin, MatchResultDraw,
		MatchResultForfeit, MatchResultAbandoned, MatchResultCancelled:
		return true
	default:
		return false
	}
}

// IsAbandonReason reports whether reason is a known abandon reason.
func IsAbandonReason(reason string) bool {
	switch reason {
	case AbandonReasonExplicitLeave, AbandonReasonDisconnectTimeout, AbandonReasonAccountIneligible:
		return true
	default:
		return false
	}
}

// SupportedMode reports whether mode is approved for queue/formation entry.
// Accepts the six canonical modes plus the legacy ranked_standard alias.
func SupportedMode(mode string) bool {
	switch NormalizeMode(mode) {
	case ModeCasualSolo, ModeCasualDuo, ModeCasualSquad,
		ModeRankedSolo, ModeRankedDuo, ModeRankedSquad:
		// NormalizeMode returns "" for unknown inputs; also reject empty.
		return mode != ""
	default:
		return false
	}
}

// NormalizeMode maps accepted mode strings to their canonical six-mode value.
// ranked_standard becomes ranked_solo. Unknown inputs return "".
func NormalizeMode(mode string) string {
	mode = strings.TrimSpace(mode)
	switch mode {
	case ModeRankedStandard:
		return ModeRankedSolo
	case ModeCasualSolo, ModeCasualDuo, ModeCasualSquad,
		ModeRankedSolo, ModeRankedDuo, ModeRankedSquad:
		return mode
	default:
		return ""
	}
}

// ParseMode validates and decomposes a mode string (canonical or legacy alias).
func ParseMode(mode string) (ModeParts, error) {
	raw := strings.TrimSpace(mode)
	canonical := NormalizeMode(raw)
	if canonical == "" {
		return ModeParts{}, fmt.Errorf("%w: %q", ErrUnsupportedMode, mode)
	}
	parts, err := PartsFromMode(canonical)
	if err != nil {
		return ModeParts{}, err
	}
	parts.Raw = raw
	return parts, nil
}

// ModeFromParts builds the canonical mode string from playlist and format.
func ModeFromParts(playlist, format string) (string, error) {
	playlist = strings.TrimSpace(playlist)
	format = strings.TrimSpace(format)
	switch playlist {
	case PlaylistCasual, PlaylistRanked:
	default:
		return "", fmt.Errorf("%w: playlist %q", ErrUnsupportedMode, playlist)
	}
	switch format {
	case FormatSolo, FormatDuo, FormatSquad:
	default:
		return "", fmt.Errorf("%w: format %q", ErrUnsupportedMode, format)
	}
	return playlist + "_" + format, nil
}

// PartsFromMode decomposes a canonical mode (or legacy alias) into playlist/format/team size.
func PartsFromMode(mode string) (ModeParts, error) {
	raw := strings.TrimSpace(mode)
	canonical := NormalizeMode(raw)
	if canonical == "" {
		return ModeParts{}, fmt.Errorf("%w: %q", ErrUnsupportedMode, mode)
	}
	playlist, format, ok := splitMode(canonical)
	if !ok {
		return ModeParts{}, fmt.Errorf("%w: %q", ErrUnsupportedMode, mode)
	}
	teamSize, err := TeamSizeForFormat(format)
	if err != nil {
		return ModeParts{}, err
	}
	return ModeParts{
		Playlist:  playlist,
		Format:    format,
		TeamSize:  teamSize,
		Canonical: canonical,
		Raw:       raw,
	}, nil
}

func splitMode(canonical string) (playlist, format string, ok bool) {
	switch canonical {
	case ModeCasualSolo:
		return PlaylistCasual, FormatSolo, true
	case ModeCasualDuo:
		return PlaylistCasual, FormatDuo, true
	case ModeCasualSquad:
		return PlaylistCasual, FormatSquad, true
	case ModeRankedSolo:
		return PlaylistRanked, FormatSolo, true
	case ModeRankedDuo:
		return PlaylistRanked, FormatDuo, true
	case ModeRankedSquad:
		return PlaylistRanked, FormatSquad, true
	default:
		return "", "", false
	}
}

// TeamSizeForFormat returns the roster size required per team for a format.
func TeamSizeForFormat(format string) (int, error) {
	switch strings.TrimSpace(format) {
	case FormatSolo:
		return TeamSizeSolo, nil
	case FormatDuo:
		return TeamSizeDuo, nil
	case FormatSquad:
		return TeamSizeSquad, nil
	default:
		return 0, fmt.Errorf("%w: format %q", ErrUnsupportedMode, format)
	}
}

// ValidateTeamSize reports whether teamSize agrees with format.
func ValidateTeamSize(format string, teamSize int) error {
	expected, err := TeamSizeForFormat(format)
	if err != nil {
		return err
	}
	if teamSize != expected {
		return fmt.Errorf("%w: team_size %d does not match format %q", ErrInvalidRequest, teamSize, format)
	}
	return nil
}

// IsCasual reports whether mode is a casual playlist mode (including after alias normalization).
func IsCasual(mode string) bool {
	parts, err := PartsFromMode(mode)
	return err == nil && parts.Playlist == PlaylistCasual
}

// IsRanked reports whether mode is a ranked playlist mode (including ranked_standard).
func IsRanked(mode string) bool {
	parts, err := PartsFromMode(mode)
	return err == nil && parts.Playlist == PlaylistRanked
}

// IsTeamMode reports whether mode requires a multi-player premade party (duo or squad).
func IsTeamMode(mode string) bool {
	parts, err := PartsFromMode(mode)
	return err == nil && (parts.Format == FormatDuo || parts.Format == FormatSquad)
}

// IsPlaylist reports whether value is a known playlist.
func IsPlaylist(playlist string) bool {
	switch strings.TrimSpace(playlist) {
	case PlaylistCasual, PlaylistRanked:
		return true
	default:
		return false
	}
}

// IsFormat reports whether value is a known format.
func IsFormat(format string) bool {
	switch strings.TrimSpace(format) {
	case FormatSolo, FormatDuo, FormatSquad:
		return true
	default:
		return false
	}
}
