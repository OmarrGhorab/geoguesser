package locations

import (
	"time"

	"github.com/google/uuid"
)

// Location is a curated playable location with hidden coordinates.
type Location struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Latitude    float64   `gorm:"type:numeric(9,6);not null"`
	Longitude   float64   `gorm:"type:numeric(9,6);not null"`
	CountryCode string    `gorm:"type:text;not null"`
	Region      *string   `gorm:"type:text"`
	Locality    *string   `gorm:"type:text"`
	Difficulty  string    `gorm:"type:text;not null"`
	Provider    string    `gorm:"type:text;not null"`
	ProviderRef string    `gorm:"type:text;not null"`
	Attribution *string   `gorm:"type:text"`
	Heading     *int      `gorm:"type:int"`
	Status      string    `gorm:"type:text;not null;default:'active'"`
	RandomKey   float64   `gorm:"type:numeric;not null;default:random()"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the table name for the model.
func (Location) TableName() string {
	return "locations"
}

// mapAccess holds the map metadata needed to authorize location media access.
type mapAccess struct {
	MapID          uuid.UUID
	Visibility     string
	AccessTier     string
	Status         string
	LocationStatus string
}
