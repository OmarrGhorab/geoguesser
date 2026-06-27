package games

import "errors"

var (
	// ErrInvalidGameRequest indicates invalid create/start input.
	ErrInvalidGameRequest = errors.New("invalid game request")
	// ErrInvalidGuess indicates invalid submitted coordinates.
	ErrInvalidGuess = errors.New("invalid guess")
	// ErrGameNotFound indicates the game is missing or hidden.
	ErrGameNotFound = errors.New("game not found")
	// ErrRoundNotFound indicates the round is missing or hidden.
	ErrRoundNotFound = errors.New("round not found")
	// ErrForbidden indicates the session does not own the game.
	ErrForbidden = errors.New("forbidden")
	// ErrInvalidTransition indicates the requested state transition is not allowed.
	ErrInvalidTransition = errors.New("invalid game transition")
	// ErrGameNotActive indicates the game is not currently playable.
	ErrGameNotActive = errors.New("game not active")
	// ErrRoundClosed indicates the server deadline has passed or the round is completed.
	ErrRoundClosed = errors.New("round closed")
	// ErrRoundNotCurrent indicates the round is not the current playable round.
	ErrRoundNotCurrent = errors.New("round not current")
	// ErrAlreadyGuessed indicates the player already submitted a guess for the round.
	ErrAlreadyGuessed = errors.New("already guessed")
	// ErrIdempotencyConflict indicates the same idempotency key was used for different input.
	ErrIdempotencyConflict = errors.New("idempotency conflict")
	// ErrNotEnoughLocations indicates the map cannot provide enough unique active locations.
	ErrNotEnoughLocations = errors.New("not enough locations")
	// ErrResultsNotReady indicates final results were requested before completion.
	ErrResultsNotReady = errors.New("game results not ready")
)
