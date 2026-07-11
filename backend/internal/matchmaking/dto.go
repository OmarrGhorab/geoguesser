package matchmaking

import (
	"time"

	"github.com/google/uuid"
)

// JoinQueueRequest is the strict POST /matchmaking/queue body.
// Unknown properties are rejected by the decoder.
type JoinQueueRequest struct {
	Mode string `json:"mode"`
}

// StatusResponse is the public matchmaking status envelope.
// Queue is present only for searching; Match is present only for matched.
type StatusResponse struct {
	Status string        `json:"status"`
	Queue  *QueueDetails `json:"queue"`
	Match  *MatchDetails `json:"match"`
}

// QueueDetails is the public searching payload.
type QueueDetails struct {
	Mode            string    `json:"mode"`
	SearchStartedAt time.Time `json:"search_started_at"`
	LeaseExpiresAt  time.Time `json:"lease_expires_at"`
}

// MatchDetails is the public matched payload (no opponent data).
type MatchDetails struct {
	MatchID     uuid.UUID `json:"match_id"`
	GameID      uuid.UUID `json:"game_id"`
	Mode        string    `json:"mode"`
	FormedAt    time.Time `json:"formed_at"`
	Destination string    `json:"destination"`
}

// QueueEntry is the internal Redis player-hash representation.
type QueueEntry struct {
	EntryID          string
	UserID           uuid.UUID
	Mode             string
	State            string
	EnqueuedAtMs     int64
	LeaseExpiresAtMs int64
	ClaimID          string
}

// PairClaim is the internal Redis claim record used during formation.
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

// DestinationPath returns the locale-independent game destination path.
func DestinationPath(gameID uuid.UUID) string {
	return "/games/" + gameID.String()
}

// NewNotQueuedStatus builds a not_queued public response.
func NewNotQueuedStatus() *StatusResponse {
	return &StatusResponse{
		Status: PublicStatusNotQueued,
		Queue:  nil,
		Match:  nil,
	}
}

// NewSearchingStatus builds a searching public response.
func NewSearchingStatus(mode string, searchStartedAt, leaseExpiresAt time.Time) *StatusResponse {
	return &StatusResponse{
		Status: PublicStatusSearching,
		Queue: &QueueDetails{
			Mode:            mode,
			SearchStartedAt: searchStartedAt.UTC(),
			LeaseExpiresAt:  leaseExpiresAt.UTC(),
		},
		Match: nil,
	}
}

// NewMatchedStatus builds a matched public response.
func NewMatchedStatus(matchID, gameID uuid.UUID, mode string, formedAt time.Time) *StatusResponse {
	return &StatusResponse{
		Status: PublicStatusMatched,
		Queue:  nil,
		Match: &MatchDetails{
			MatchID:     matchID,
			GameID:      gameID,
			Mode:        mode,
			FormedAt:    formedAt.UTC(),
			Destination: DestinationPath(gameID),
		},
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
