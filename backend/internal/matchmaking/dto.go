package matchmaking

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidModeSelection is returned when legacy mode and playlist/format disagree.
var ErrInvalidModeSelection = errors.New("invalid mode selection")

// JoinQueueRequest is the strict POST /matchmaking/queue body.
// Unknown properties are rejected by the decoder.
// Accepts either the legacy {"mode":"..."} shape or canonical playlist/format (+ optional party_id).
type JoinQueueRequest struct {
	// Mode is the deprecated single-field mode (e.g. ranked_standard, casual_solo).
	Mode string `json:"mode,omitempty"`
	// Playlist is the canonical playlist (casual | ranked).
	Playlist string `json:"playlist,omitempty"`
	// Format is the canonical format (solo | duo | squad).
	Format string `json:"format,omitempty"`
	// PartyID is required for duo/squad; must be null/omitted for solo.
	PartyID *uuid.UUID `json:"party_id,omitempty"`
	// PartyVersion is the optimistic party version required for duo/squad queue entry.
	PartyVersion *int64 `json:"party_version,omitempty"`
}

// ResolveMode resolves legacy mode or playlist+format into ModeParts.
// Supplying both is allowed only when they are equivalent; otherwise ErrInvalidModeSelection.
func (r JoinQueueRequest) ResolveMode() (ModeParts, error) {
	legacy := strings.TrimSpace(r.Mode)
	playlist := strings.TrimSpace(r.Playlist)
	format := strings.TrimSpace(r.Format)
	hasLegacy := legacy != ""
	hasCanonical := playlist != "" || format != ""

	if !hasLegacy && !hasCanonical {
		return ModeParts{}, ErrInvalidRequest
	}

	if hasLegacy && hasCanonical {
		legacyParts, errLegacy := ParseMode(legacy)
		if errLegacy != nil {
			return ModeParts{}, errLegacy
		}
		if playlist == "" || format == "" {
			return ModeParts{}, ErrInvalidModeSelection
		}
		canonicalMode, errCanon := ModeFromParts(playlist, format)
		if errCanon != nil {
			return ModeParts{}, errCanon
		}
		if legacyParts.Canonical != canonicalMode {
			return ModeParts{}, ErrInvalidModeSelection
		}
		return legacyParts, nil
	}

	if hasLegacy {
		return ParseMode(legacy)
	}

	if playlist == "" || format == "" {
		return ModeParts{}, ErrInvalidRequest
	}
	canonicalMode, err := ModeFromParts(playlist, format)
	if err != nil {
		return ModeParts{}, err
	}
	return ParseMode(canonicalMode)
}

// StatusResponse is the public matchmaking status envelope.
// Queue is present only for searching; Match is present only for matched.
type StatusResponse struct {
	Status string        `json:"status"`
	Queue  *QueueDetails `json:"queue"`
	Match  *MatchDetails `json:"match"`
}

// QueueDetails is the public searching payload.
// Legacy fields (mode, search_started_at, lease_expires_at) are always present;
// ticket/roster fields are additive for the six-mode API.
type QueueDetails struct {
	TicketID        string        `json:"ticket_id,omitempty"`
	Playlist        string        `json:"playlist,omitempty"`
	Format          string        `json:"format,omitempty"`
	Mode            string        `json:"mode"`
	PartyID         *uuid.UUID    `json:"party_id,omitempty"`
	SearchStartedAt time.Time     `json:"search_started_at"`
	LeaseExpiresAt  time.Time     `json:"lease_expires_at"`
	RatingWindow    *RatingWindow `json:"rating_window,omitempty"`
}

// RatingWindow is the current ranked search band (absent for Casual).
type RatingWindow struct {
	Minimum int `json:"minimum"`
	Maximum int `json:"maximum"`
}

// MatchDetails is the public matched payload (no opponent data).
// Legacy fields remain; playlist/format are additive.
type MatchDetails struct {
	MatchID     uuid.UUID `json:"match_id"`
	GameID      uuid.UUID `json:"game_id"`
	Playlist    string    `json:"playlist,omitempty"`
	Format      string    `json:"format,omitempty"`
	Mode        string    `json:"mode"`
	FormedAt    time.Time `json:"formed_at"`
	Destination string    `json:"destination"`
}

// QueueEntry is the internal v1 Redis player-hash representation (pair matchmaking).
// Retained for rollout while v2 team tickets are introduced.
type QueueEntry struct {
	EntryID          string
	UserID           uuid.UUID
	Mode             string
	State            string
	EnqueuedAtMs     int64
	LeaseExpiresAtMs int64
	ClaimID          string
}

// PairClaim is the internal v1 Redis claim record used during pair formation.
// Retained for rollout while v2 team claims are introduced.
type PairClaim struct {
	ClaimID        string
	FormationKey   string
	Mode           string
	EntryIDA       string
	UserIDA        uuid.UUID
	EnqueuedAtMsA  int64
	EntryIDB       string
	UserIDB        uuid.UUID
	EnqueuedAtMsB  int64
	ClaimedAtMs    int64
	RecoverAfterMs int64
}

// QueueTicket is the internal v2 team-ticket representation.
type QueueTicket struct {
	TicketID         string
	PartyID          uuid.UUID // uuid.Nil when solo / no durable party
	Mode             string    // canonical six-mode value
	UserIDs          []uuid.UUID
	TeamSize         int
	TeamAvgRating    int
	EnqueuedAtMs     int64
	LeaseExpiresAtMs int64
	State            string
	ClaimID          string
	PartyVersion     int64
}

// TeamClaim is the internal v2 Redis claim of two equal-roster tickets.
type TeamClaim struct {
	ClaimID        string
	FormationKey   string
	Mode           string
	TicketIDA      string
	TicketIDB      string
	UserIDsA       []uuid.UUID
	UserIDsB       []uuid.UUID
	PartyIDA       uuid.UUID
	PartyIDB       uuid.UUID
	PartyVersionA  int64
	PartyVersionB  int64
	TeamSize       int
	TeamAvgRatingA int
	TeamAvgRatingB int
	EnqueuedAtMsA  int64
	EnqueuedAtMsB  int64
	ClaimedAtMs    int64
	RecoverAfterMs int64
}

// DestinationPath returns the locale-independent game destination path.
// Legacy consumers still use /games/{gameID}; match-centric clients may use /matches/{matchID}.
func DestinationPath(gameID uuid.UUID) string {
	return "/games/" + gameID.String()
}

// MatchDestinationPath returns the locale-independent match destination path.
func MatchDestinationPath(matchID uuid.UUID) string {
	return "/matches/" + matchID.String()
}

// NewNotQueuedStatus builds a not_queued public response.
func NewNotQueuedStatus() *StatusResponse {
	return &StatusResponse{
		Status: PublicStatusNotQueued,
		Queue:  nil,
		Match:  nil,
	}
}

// NewSearchingStatus builds a searching public response from a mode string.
// Playlist/format are filled when the mode is recognized (including ranked_standard).
func NewSearchingStatus(mode string, searchStartedAt, leaseExpiresAt time.Time) *StatusResponse {
	details := &QueueDetails{
		Mode:            mode,
		SearchStartedAt: searchStartedAt.UTC(),
		LeaseExpiresAt:  leaseExpiresAt.UTC(),
	}
	if parts, err := ParseMode(mode); err == nil {
		details.Playlist = parts.Playlist
		details.Format = parts.Format
		details.Mode = parts.Canonical
	}
	return &StatusResponse{
		Status: PublicStatusSearching,
		Queue:  details,
		Match:  nil,
	}
}

// NewSearchingStatusFromTicket builds a searching response from a v2 queue ticket.
func NewSearchingStatusFromTicket(ticket *QueueTicket, ratingWindow *RatingWindow) *StatusResponse {
	if ticket == nil {
		return NewNotQueuedStatus()
	}
	details := &QueueDetails{
		TicketID:        ticket.TicketID,
		Mode:            ticket.Mode,
		SearchStartedAt: time.UnixMilli(ticket.EnqueuedAtMs).UTC(),
		LeaseExpiresAt:  time.UnixMilli(ticket.LeaseExpiresAtMs).UTC(),
		RatingWindow:    ratingWindow,
	}
	if parts, err := ParseMode(ticket.Mode); err == nil {
		details.Playlist = parts.Playlist
		details.Format = parts.Format
		details.Mode = parts.Canonical
	}
	if ticket.PartyID != uuid.Nil {
		partyID := ticket.PartyID
		details.PartyID = &partyID
	}
	return &StatusResponse{
		Status: PublicStatusSearching,
		Queue:  details,
		Match:  nil,
	}
}

// NewMatchedStatus builds a matched public response.
// Playlist/format are filled when the mode is recognized.
// Legacy ranked_standard keeps /games/{id}; six-mode matches use /matches/{id}.
func NewMatchedStatus(matchID, gameID uuid.UUID, mode string, formedAt time.Time) *StatusResponse {
	destination := DestinationPath(gameID)
	if mode != ModeRankedStandard {
		destination = MatchDestinationPath(matchID)
	}
	details := &MatchDetails{
		MatchID:     matchID,
		GameID:      gameID,
		Mode:        mode,
		FormedAt:    formedAt.UTC(),
		Destination: destination,
	}
	if parts, err := ParseMode(mode); err == nil {
		details.Playlist = parts.Playlist
		details.Format = parts.Format
		details.Mode = parts.Canonical
	}
	return &StatusResponse{
		Status: PublicStatusMatched,
		Queue:  nil,
		Match:  details,
	}
}

// NewTemporarilyUnavailableStatus builds a recoverable uncertainty response.
func NewTemporarilyUnavailableStatus() *StatusResponse {
	return &StatusResponse{
		Status: PublicStatusTemporarilyUnavailable,
		Queue:  nil,
		Match:  nil,
	}
}
