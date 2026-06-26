package maps

import (
	"time"

	"github.com/google/uuid"
)

// MapDTO is the public map shape returned by the API.
type MapDTO struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Visibility  string    `json:"visibility"`
	AccessTier  string    `json:"access_tier"`
	Difficulty  string    `json:"difficulty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MapListResponse is the response for GET /maps.
type MapListResponse struct {
	Data []MapDTO `json:"data"`
	Page PageInfo `json:"page"`
}

// MapResponse is the response for GET /maps/{mapId}.
type MapResponse struct {
	Map MapDTO `json:"map"`
}

// PageInfo carries cursor pagination metadata.
type PageInfo struct {
	Limit      int     `json:"limit"`
	NextCursor *string `json:"next_cursor,omitempty"`
}
