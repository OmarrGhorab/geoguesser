package games

import (
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/maps"
)

type MultiplayerStart struct {
	Game         Game
	CurrentRound Round
}

type MultiplayerGuessOutcome struct {
	Guess           Guess
	SubmittedCount  int
	EligibleCount   int
	RoundCompleted  bool
	GameCompleted   bool
	NextRoundNumber *int
}

type MultiplayerRoundState struct {
	RoundID            uuid.UUID
	RoundNumber        int
	Status             string
	StartsAt           *time.Time
	EndsAt             *time.Time
	Provider           string
	ProviderRef        string
	MediaURL           string
	Attribution        *string
	SubmittedCount     int
	EligibleCount      int
	SubmittedPlayerIDs []uuid.UUID
}

func roundsFromSelected(gameID uuid.UUID, selected []maps.SelectedLocation, count int) []Round {
	rounds := make([]Round, count)
	for i := range rounds {
		rounds[i] = Round{
			GameID:      gameID,
			LocationID:  selected[i].ID,
			RoundNumber: i + 1,
			Status:      RoundStatusPending,
		}
	}
	return rounds
}
