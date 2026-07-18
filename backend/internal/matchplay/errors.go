package matchplay

import (
	"errors"
	"net/http"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// Domain errors for match snapshot, leave, chat, and collaboration operations.
var (
	ErrUnauthorized           = errors.New("matchplay unauthorized")
	ErrNotFound               = errors.New("match not found")
	ErrForbiddenOpponent      = errors.New("matchplay forbidden opponent resource")
	ErrMatchNotActive         = errors.New("match not active")
	ErrMatchAlreadyFormed     = errors.New("match already formed")
	ErrInvalidLeave           = errors.New("invalid match leave")
	ErrRoundNotActive         = errors.New("round not active")
	ErrRoundNotRevealed       = errors.New("round not revealed")
	ErrAlreadyGuessed         = errors.New("already guessed")
	ErrGuessLocked            = errors.New("guess locked")
	ErrRoundClosed            = errors.New("round closed")
	ErrSpectateForbidden      = errors.New("spectate forbidden")
	ErrTeammateRequired       = errors.New("teammate required")
	ErrChatUnavailable        = errors.New("chat unavailable")
	ErrMessageTooLong         = errors.New("message too long")
	ErrAttachmentNotReady     = errors.New("attachment not ready")
	ErrInvalidImage           = errors.New("invalid image")
	ErrUnsafeImage            = errors.New("unsafe image")
	ErrImageUnavailable       = errors.New("image service unavailable")
	ErrProgressionPending     = errors.New("progression pending")
	ErrUnavailable            = errors.New("matchplay unavailable")
	ErrInvalidRequest         = errors.New("invalid matchplay request")
	ErrInvalidJSON            = errors.New("invalid matchplay json")
	ErrIdempotencyConflict    = errors.New("matchplay idempotency conflict")
	ErrAttachmentAlreadyBound = errors.New("attachment already bound")
	ErrRateLimited            = errors.New("matchplay rate limited")
	ErrImageLimitExceeded     = errors.New("image message limit exceeded")
)

// Stable API error codes for matchplay. Shared codes reuse apphttp constants where applicable.
const (
	CodeMatchNotActive         = "match_not_active"
	CodeMatchAlreadyFormed     = "match_already_formed"
	CodeRoundNotActive         = "round_not_active"
	CodeRoundNotRevealed       = "round_not_revealed"
	CodeAlreadyGuessed         = "already_guessed"
	CodeGuessLocked            = "guess_locked"
	CodeRoundClosed            = "round_closed"
	CodeSpectateForbidden      = "spectate_forbidden"
	CodeTeammateRequired       = "teammate_required"
	CodeChatUnavailable        = "chat_unavailable"
	CodeMessageTooLong         = "message_too_long"
	CodeAttachmentNotReady     = "attachment_not_ready"
	CodeInvalidImage           = "invalid_image"
	CodeUnsafeImage            = "unsafe_image"
	CodeImageUnavailable       = "image_service_unavailable"
	CodeProgressionPending     = "progression_pending"
	CodeUnavailable            = "temporarily_unavailable"
	CodeIdempotencyConflict    = "idempotency_conflict"
	CodeAttachmentAlreadyBound = "attachment_already_bound"
	CodeImageLimitExceeded     = "image_limit_exceeded"
)

// Safe user-facing messages (frontend localizes).
const (
	MsgNotFound               = "The requested resource was not found."
	MsgForbiddenOpponent      = "The requested resource was not found."
	MsgMatchNotActive         = "This match is not currently active."
	MsgMatchAlreadyFormed     = "A match has already been formed."
	MsgInvalidLeave           = "You cannot leave the match in its current state."
	MsgRoundNotActive         = "This round is not currently active."
	MsgRoundNotRevealed       = "Round results are not available yet."
	MsgAlreadyGuessed         = "You have already submitted a guess for this round."
	MsgGuessLocked            = "Your guess is locked and cannot be changed."
	MsgRoundClosed            = "This round is closed."
	MsgSpectateForbidden      = "You cannot spectate that player right now."
	MsgTeammateRequired       = "This action requires a teammate on your team."
	MsgChatUnavailable        = "Team chat is not available right now."
	MsgMessageTooLong         = "That message is too long."
	MsgAttachmentNotReady     = "That attachment is not ready yet."
	MsgInvalidImage           = "That image is not valid."
	MsgUnsafeImage            = "That image could not be accepted."
	MsgImageUnavailable       = "Image uploads are temporarily unavailable."
	MsgProgressionPending     = "Competitive progression is still being finalized."
	MsgUnavailable            = "Match features are temporarily unavailable."
	MsgInvalidRequest         = "The match request is invalid."
	MsgIdempotencyConflict    = "This request conflicts with a previous idempotent command."
	MsgAttachmentAlreadyBound = "That attachment is already linked to another message."
	MsgImageLimitExceeded     = "Image message limit for this match has been reached."
)

// MapError converts domain errors into stable shared-envelope API errors.
// Missing and unauthorized resources both map to privacy-safe not_found.
func MapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrUnauthorized):
		return apphttp.ErrUnauthorized
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrForbiddenOpponent):
		// Privacy-safe: do not distinguish missing vs unauthorized opponent resources.
		return apphttp.ErrNotFound.WithCause(err)
	case errors.Is(err, ErrMatchNotActive):
		return apphttp.NewAPIError(http.StatusConflict, CodeMatchNotActive, MsgMatchNotActive).WithCause(err)
	case errors.Is(err, ErrMatchAlreadyFormed):
		return apphttp.NewAPIError(http.StatusConflict, CodeMatchAlreadyFormed, MsgMatchAlreadyFormed).WithCause(err)
	case errors.Is(err, ErrInvalidLeave):
		return apphttp.NewAPIError(http.StatusUnprocessableEntity, apphttp.ErrCodeUnprocessable, MsgInvalidLeave).WithCause(err)
	case errors.Is(err, ErrRoundNotActive):
		return apphttp.NewAPIError(http.StatusConflict, CodeRoundNotActive, MsgRoundNotActive).WithCause(err)
	case errors.Is(err, ErrRoundNotRevealed):
		return apphttp.NewAPIError(http.StatusConflict, CodeRoundNotRevealed, MsgRoundNotRevealed).WithCause(err)
	case errors.Is(err, ErrAlreadyGuessed):
		return apphttp.NewAPIError(http.StatusConflict, CodeAlreadyGuessed, MsgAlreadyGuessed).WithCause(err)
	case errors.Is(err, ErrGuessLocked):
		return apphttp.NewAPIError(http.StatusConflict, CodeGuessLocked, MsgGuessLocked).WithCause(err)
	case errors.Is(err, ErrRoundClosed):
		return apphttp.NewAPIError(http.StatusConflict, CodeRoundClosed, MsgRoundClosed).WithCause(err)
	case errors.Is(err, ErrSpectateForbidden):
		return apphttp.NewAPIError(http.StatusForbidden, CodeSpectateForbidden, MsgSpectateForbidden).WithCause(err)
	case errors.Is(err, ErrTeammateRequired):
		return apphttp.NewAPIError(http.StatusForbidden, CodeTeammateRequired, MsgTeammateRequired).WithCause(err)
	case errors.Is(err, ErrChatUnavailable):
		return apphttp.NewAPIError(http.StatusUnprocessableEntity, CodeChatUnavailable, MsgChatUnavailable).WithCause(err)
	case errors.Is(err, ErrMessageTooLong):
		return apphttp.NewAPIError(http.StatusUnprocessableEntity, CodeMessageTooLong, MsgMessageTooLong).WithCause(err)
	case errors.Is(err, ErrAttachmentNotReady):
		return apphttp.NewAPIError(http.StatusUnprocessableEntity, CodeAttachmentNotReady, MsgAttachmentNotReady).WithCause(err)
	case errors.Is(err, ErrInvalidImage):
		return apphttp.NewAPIError(http.StatusUnprocessableEntity, CodeInvalidImage, MsgInvalidImage).WithCause(err)
	case errors.Is(err, ErrUnsafeImage):
		return apphttp.NewAPIError(http.StatusUnprocessableEntity, CodeUnsafeImage, MsgUnsafeImage).WithCause(err)
	case errors.Is(err, ErrImageUnavailable):
		return apphttp.NewAPIError(http.StatusServiceUnavailable, CodeImageUnavailable, MsgImageUnavailable).WithCause(err)
	case errors.Is(err, ErrProgressionPending):
		return apphttp.NewAPIError(http.StatusAccepted, CodeProgressionPending, MsgProgressionPending).WithCause(err)
	case errors.Is(err, ErrUnavailable):
		return apphttp.NewAPIError(http.StatusServiceUnavailable, CodeUnavailable, MsgUnavailable).WithCause(err)
	case errors.Is(err, ErrIdempotencyConflict):
		return apphttp.NewAPIError(http.StatusConflict, CodeIdempotencyConflict, MsgIdempotencyConflict).WithCause(err)
	case errors.Is(err, ErrAttachmentAlreadyBound):
		return apphttp.NewAPIError(http.StatusConflict, CodeAttachmentAlreadyBound, MsgAttachmentAlreadyBound).WithCause(err)
	case errors.Is(err, ErrRateLimited):
		return apphttp.NewAPIError(http.StatusTooManyRequests, apphttp.ErrCodeRateLimited, apphttp.MsgRateLimited).WithCause(err)
	case errors.Is(err, ErrImageLimitExceeded):
		return apphttp.NewAPIError(http.StatusUnprocessableEntity, CodeImageLimitExceeded, MsgImageLimitExceeded).WithCause(err)
	case errors.Is(err, ErrInvalidJSON):
		return apphttp.ErrInvalidJSON
	case errors.Is(err, ErrInvalidRequest):
		return apphttp.NewAPIError(http.StatusBadRequest, apphttp.ErrCodeValidationFailed, MsgInvalidRequest).WithCause(err)
	default:
		return err
	}
}
