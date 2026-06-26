package maps

import (
	"time"

	"github.com/google/uuid"
)

// Map is a playable location pool.
type Map struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Slug            string     `gorm:"type:text;not null;uniqueIndex:maps_slug_key"`
	Name            string     `gorm:"type:text;not null"`
	Description     *string    `gorm:"type:text"`
	Visibility      string     `gorm:"type:text;not null;default:'public'"`
	AccessTier      string     `gorm:"type:text;not null;default:'free'"`
	Difficulty      string     `gorm:"type:text;not null;default:'mixed'"`
	Status          string     `gorm:"type:text;not null;default:'draft'"`
	CreatedByUserID *uuid.UUID `gorm:"type:uuid"`
	CreatedAt       time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt       time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the table name for the model.
func (Map) TableName() string {
	return "maps"
}

// MapLocation links a location to a map with a selection weight.
type MapLocation struct {
	MapID           uuid.UUID `gorm:"type:uuid;primary_key"`
	LocationID      uuid.UUID `gorm:"type:uuid;primary_key"`
	SelectionWeight int       `gorm:"type:int;not null;default:1"`
	CreatedAt       time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the table name for the model.
func (MapLocation) TableName() string {
	return "map_locations"
}

// SelectedLocation is a gameplay-ready location with true coordinates kept for
// server-side round creation only. Media provider refs are resolved separately.
type SelectedLocation struct {
	ID          uuid.UUID
	Latitude    float64
	Longitude   float64
	CountryCode string
	Region      *string
	Locality    *string
	Difficulty  string
	Provider    string
	Attribution *string
}
