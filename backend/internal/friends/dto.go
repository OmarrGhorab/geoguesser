package friends

import (
	"time"

	"github.com/google/uuid"

	apphttp "github.com/raven/geoguess/backend/internal/http"
)

// CreateRequestBody is the body for POST /friends/requests.
type CreateRequestBody struct {
	UserID string `json:"user_id"`
}

// PublicUserDTO is the public-safe person projection used on social surfaces.
// It intentionally excludes email, preferences, tokens, and account status.
type PublicUserDTO struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	CountryCode *string   `json:"country_code,omitempty"`
}

// FriendRequestDTO is a pending request as seen by the current user.
type FriendRequestDTO struct {
	RequestID uuid.UUID     `json:"request_id"`
	User      PublicUserDTO `json:"user"`
	CreatedAt time.Time     `json:"created_at"`
	Direction string        `json:"direction,omitempty"` // incoming | outgoing when useful
}

// FriendDTO is an accepted friend entry.
type FriendDTO struct {
	User      PublicUserDTO `json:"user"`
	Since     time.Time     `json:"since"`
	CreatedAt time.Time     `json:"created_at"`
}

// BlockedUserDTO is a user blocked by the current caller.
type BlockedUserDTO struct {
	User      PublicUserDTO `json:"user"`
	BlockedAt time.Time     `json:"blocked_at"`
}

// RequestResponse wraps a single friend request.
type RequestResponse struct {
	Data FriendRequestDTO `json:"data"`
}

// FriendshipResponse wraps an accepted friendship after accept.
type FriendshipResponse struct {
	Data FriendDTO `json:"data"`
}

// RequestListResponse is a paginated pending request list.
type RequestListResponse struct {
	Data []FriendRequestDTO `json:"data"`
	Page apphttp.PageInfo   `json:"page"`
}

// FriendListResponse is a paginated accepted friends list.
type FriendListResponse struct {
	Data []FriendDTO      `json:"data"`
	Page apphttp.PageInfo `json:"page"`
}

// BlockedListResponse is a paginated blocked-users list.
type BlockedListResponse struct {
	Data []BlockedUserDTO `json:"data"`
	Page apphttp.PageInfo `json:"page"`
}

func toPublicUserDTO(p PublicProfile) PublicUserDTO {
	return PublicUserDTO(p)
}

func toRequestDTO(row RelationshipRow, direction string) FriendRequestDTO {
	return FriendRequestDTO{
		RequestID: row.Friendship.ID,
		User:      toPublicUserDTO(row.Other),
		CreatedAt: row.Friendship.CreatedAt.UTC(),
		Direction: direction,
	}
}

func toFriendDTO(row RelationshipRow) FriendDTO {
	since := row.Friendship.CreatedAt
	if row.Friendship.AcceptedAt != nil {
		since = *row.Friendship.AcceptedAt
	}
	return FriendDTO{
		User:      toPublicUserDTO(row.Other),
		Since:     since.UTC(),
		CreatedAt: row.Friendship.CreatedAt.UTC(),
	}
}

func toBlockedDTO(row RelationshipRow) BlockedUserDTO {
	return BlockedUserDTO{
		User:      toPublicUserDTO(row.Other),
		BlockedAt: row.Friendship.UpdatedAt.UTC(),
	}
}
