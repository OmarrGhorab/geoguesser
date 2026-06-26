package locations_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/locations"
	"github.com/raven/geoguess/backend/internal/platform/postgres"
	"github.com/raven/geoguess/backend/internal/session"
	"gorm.io/gorm"
)

func setupLocationsService(t *testing.T) (*locations.Service, seedData) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL required for integration tests")
	}

	ctx := context.Background()
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatalf("postgres connection failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	repo := locations.NewRepository(db)
	service := locations.NewService(repo, locations.StaticProvider{})

	data := seed(ctx, t, db)
	return service, data
}

type seedData struct {
	FreeMapID         uuid.UUID
	PremiumMapID      uuid.UUID
	FreeLocationID    uuid.UUID
	PremiumLocationID uuid.UUID
}

func seed(ctx context.Context, t *testing.T, db *gorm.DB) seedData {
	t.Helper()

	freeMap := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	premiumMap := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	freeLoc := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	premiumLoc := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")

	clean := func() {
		_ = db.WithContext(ctx).Exec("DELETE FROM map_locations WHERE map_id IN (?, ?)", freeMap, premiumMap)
		_ = db.WithContext(ctx).Exec("DELETE FROM locations WHERE id IN (?, ?)", freeLoc, premiumLoc)
		_ = db.WithContext(ctx).Exec("DELETE FROM maps WHERE id IN (?, ?)", freeMap, premiumMap)
	}
	clean()
	t.Cleanup(clean)

	if err := db.WithContext(ctx).Exec(`
		INSERT INTO maps (id, slug, name, visibility, access_tier, difficulty, status)
		VALUES (?, 'test-free', 'Free', 'public', 'free', 'mixed', 'active'),
		       (?, 'test-premium', 'Premium', 'public', 'premium', 'mixed', 'active')
	`, freeMap, premiumMap).Error; err != nil {
		t.Fatalf("seed maps failed: %v", err)
	}

	if err := db.WithContext(ctx).Exec(`
		INSERT INTO locations (id, latitude, longitude, country_code, difficulty, provider, provider_ref, status)
		VALUES (?, 0.0, 0.0, 'US', 'easy', 'image', 'https://example.com/free.jpg', 'active'),
		       (?, 10.0, 10.0, 'FR', 'medium', 'streetview', 'https://example.com/premium.jpg', 'active')
	`, freeLoc, premiumLoc).Error; err != nil {
		t.Fatalf("seed locations failed: %v", err)
	}

	if err := db.WithContext(ctx).Exec(`
		INSERT INTO map_locations (map_id, location_id)
		VALUES (?, ?), (?, ?)
	`, freeMap, freeLoc, premiumMap, premiumLoc).Error; err != nil {
		t.Fatalf("seed map_locations failed: %v", err)
	}

	return seedData{
		FreeMapID:         freeMap,
		PremiumMapID:      premiumMap,
		FreeLocationID:    freeLoc,
		PremiumLocationID: premiumLoc,
	}
}

func TestGetLocationMedia_Free_AllowsAnonymous(t *testing.T) {
	service, data := setupLocationsService(t)

	resp, err := service.GetLocationMedia(context.Background(), &session.Context{Kind: session.KindAnonymous}, data.FreeLocationID.String())
	if err != nil {
		t.Fatalf("expected free media to be allowed, got %v", err)
	}
	if resp.Media.URL != "https://example.com/free.jpg" {
		t.Fatalf("unexpected url: %s", resp.Media.URL)
	}
	if resp.Media.Type != locations.MediaTypeImage {
		t.Fatalf("expected image type, got %s", resp.Media.Type)
	}
}

func TestGetLocationMedia_Premium_RequiresRegistered(t *testing.T) {
	service, data := setupLocationsService(t)

	_, err := service.GetLocationMedia(context.Background(), &session.Context{Kind: session.KindAnonymous}, data.PremiumLocationID.String())
	if err == nil {
		t.Fatal("expected premium media to be denied for anonymous")
	}
	if err != locations.ErrMediaAccessDenied {
		t.Fatalf("expected ErrMediaAccessDenied, got %v", err)
	}

	userID := "0197a000-0000-7000-8000-000000000001"
	_, err = service.GetLocationMedia(context.Background(), &session.Context{Kind: session.KindUser, UserID: &userID}, data.PremiumLocationID.String())
	if err != nil {
		t.Fatalf("expected premium media to be allowed for registered user, got %v", err)
	}
}

func TestGetLocationMedia_StreetView_ReturnsPanorama(t *testing.T) {
	service, data := setupLocationsService(t)

	resp, err := service.GetLocationMedia(context.Background(), &session.Context{Kind: session.KindAnonymous}, data.FreeLocationID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Media.Type != locations.MediaTypeImage {
		t.Fatalf("expected image, got %s", resp.Media.Type)
	}

	resp, err = service.GetLocationMedia(context.Background(), &session.Context{Kind: session.KindUser, UserID: strPtr("0197a000-0000-7000-8000-000000000001")}, data.PremiumLocationID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Media.Type != locations.MediaTypePanorama {
		t.Fatalf("expected panorama, got %s", resp.Media.Type)
	}
}

func strPtr(s string) *string {
	return &s
}
