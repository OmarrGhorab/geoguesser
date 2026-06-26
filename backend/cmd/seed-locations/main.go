package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/locations"
	"github.com/raven/geoguess/backend/internal/platform/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// sourceLocation matches the shape of the imported JSON file.
type sourceLocation struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Heading int     `json:"heading"`
	Extra   struct {
		Tags []string `json:"tags"`
	} `json:"extra"`
	PanoID string `json:"panoId"`
}

func main() {
	var (
		file       = flag.String("file", "", "path to the locations JSON file (required)")
		mapSlug    = flag.String("map-slug", "world", "slug of the map to attach locations to")
		mapName    = flag.String("map-name", "World", "display name for the map if it is created")
		dryRun     = flag.Bool("dry-run", false, "print counts without writing to the database")
		difficulty = flag.String("difficulty", "medium", "default difficulty for imported locations")
	)
	flag.Parse()

	logger := slog.Default()

	if *file == "" {
		logger.Error("-file is required")
		os.Exit(1)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	f, err := os.Open(*file)
	if err != nil {
		logger.Error("failed to open locations file", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() { _ = f.Close() }()

	var locations []sourceLocation
	if err := json.NewDecoder(f).Decode(&locations); err != nil {
		logger.Error("failed to parse locations file", slog.Any("error", err))
		os.Exit(1)
	}

	if len(locations) == 0 {
		logger.Info("no locations found in file")
		return
	}

	logger.Info("parsed locations", slog.Int("count", len(locations)))

	if *dryRun {
		for i, loc := range locations {
			if i >= 5 {
				break
			}
			logger.Info("sample",
				slog.String("pano_id", loc.PanoID),
				slog.Float64("lat", loc.Lat),
				slog.Float64("lng", loc.Lng),
				slog.Int("heading", loc.Heading),
				slog.Any("tags", loc.Extra.Tags),
			)
		}
		return
	}

	db, err := postgres.Open(databaseURL)
	if err != nil {
		logger.Error("failed to connect to postgres", slog.Any("error", err))
		os.Exit(1)
	}
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("failed to get sql db", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := importLocations(db, *mapSlug, *mapName, *difficulty, locations); err != nil {
		logger.Error("failed to import locations", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("import complete")
}

func importLocations(db *gorm.DB, mapSlug, mapName, difficulty string, src []sourceLocation) error {
	return db.Transaction(func(tx *gorm.DB) error {
		mapID, err := ensureMap(tx, mapSlug, mapName)
		if err != nil {
			return err
		}

		provider := "google_street_view"
		attribution := "Google Street View"

		models := make([]locations.Location, len(src))
		for i, loc := range src {
			country, region := extractCodes(loc.Extra.Tags)
			heading := loc.Heading
			models[i] = locations.Location{
				Latitude:    loc.Lat,
				Longitude:   loc.Lng,
				CountryCode: country,
				Region:      strPtr(region),
				Difficulty:  difficulty,
				Provider:    provider,
				ProviderRef: loc.PanoID,
				Attribution: strPtr(attribution),
				Heading:     &heading,
				Status:      "active",
			}
		}

		start := time.Now()
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "provider"}, {Name: "provider_ref"}},
			DoNothing: true,
		}).CreateInBatches(models, 500).Error; err != nil {
			return fmt.Errorf("failed to insert locations: %w", err)
		}
		slog.Default().Info("inserted locations", slog.Int("count", len(models)), slog.Duration("duration", time.Since(start)))

		start = time.Now()
		var locationIDStrings []string
		if err := tx.Raw(`
			SELECT id::text FROM locations
			WHERE provider = ?
		`, provider).Scan(&locationIDStrings).Error; err != nil {
			return fmt.Errorf("failed to resolve location ids: %w", err)
		}

		links := make([]mapLocation, len(locationIDStrings))
		for i, idStr := range locationIDStrings {
			locationID, err := uuid.Parse(idStr)
			if err != nil {
				return fmt.Errorf("invalid location id %q: %w", idStr, err)
			}
			links[i] = mapLocation{MapID: mapID, LocationID: locationID}
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "map_id"}, {Name: "location_id"}},
			DoNothing: true,
		}).CreateInBatches(links, 500).Error; err != nil {
			return fmt.Errorf("failed to insert map_locations: %w", err)
		}

		slog.Default().Info("linked locations to map",
			slog.Int("parsed", len(src)),
			slog.Int("resolved", len(locationIDStrings)),
			slog.Int("linked", len(links)),
			slog.Duration("duration", time.Since(start)),
		)

		return nil
	})
}

type mapLocation struct {
	MapID      uuid.UUID `gorm:"column:map_id"`
	LocationID uuid.UUID `gorm:"column:location_id"`
}

func (mapLocation) TableName() string {
	return "map_locations"
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ensureMap(tx *gorm.DB, slug, name string) (uuid.UUID, error) {
	var idStr string
	if err := tx.Raw("SELECT id::text FROM maps WHERE slug = ?", slug).Scan(&idStr).Error; err != nil {
		return uuid.Nil, fmt.Errorf("failed to look up map: %w", err)
	}
	if idStr != "" {
		return uuid.Parse(idStr)
	}

	if err := tx.Exec(`
		INSERT INTO maps (slug, name, visibility, access_tier, difficulty, status)
		VALUES (?, ?, 'public', 'free', 'mixed', 'active')
	`, slug, name).Error; err != nil {
		return uuid.Nil, fmt.Errorf("failed to create map: %w", err)
	}

	if err := tx.Raw("SELECT id::text FROM maps WHERE slug = ?", slug).Scan(&idStr).Error; err != nil {
		return uuid.Nil, fmt.Errorf("failed to read created map id: %w", err)
	}
	return uuid.Parse(idStr)
}

func extractCodes(tags []string) (country, region string) {
	if len(tags) > 0 {
		country = tags[0]
	}
	if len(tags) > 1 {
		region = tags[1]
	}
	return
}
