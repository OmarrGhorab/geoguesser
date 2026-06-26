package maps_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/maps"
	"github.com/raven/geoguess/backend/internal/platform/observability"
	"github.com/raven/geoguess/backend/internal/platform/postgres"
	"gorm.io/gorm"
)

func setupMapsTest(t *testing.T) (*maps.Handler, *gorm.DB) {
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

	obs, err := observability.New("geoguess-test", "0.0.0")
	if err != nil {
		t.Fatalf("observability setup failed: %v", err)
	}

	repo := maps.NewRepository(db)
	service := maps.NewService(repo)
	handler := maps.NewHandler(service, obs.Logger)

	// Clean any leftover from a previous aborted run.
	_ = db.WithContext(ctx).Exec("DELETE FROM maps WHERE slug = 'test-world'")

	return handler, db
}

func TestListAndGetMapsIntegration(t *testing.T) {
	handler, db := setupMapsTest(t)
	ctx := context.Background()

	mapID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	if err := db.WithContext(ctx).Exec(`
		INSERT INTO maps (id, slug, name, visibility, access_tier, difficulty, status)
		VALUES (?, 'test-world', 'Test World', 'public', 'free', 'mixed', 'active')
	`, mapID).Error; err != nil {
		t.Fatalf("seed map failed: %v", err)
	}
	t.Cleanup(func() {
		_ = db.WithContext(ctx).Exec("DELETE FROM maps WHERE id = ?", mapID)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/maps?limit=5", nil)
	handler.ListMaps(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var list maps.MapListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	if list.Page.Limit != 5 {
		t.Fatalf("expected limit 5, got %d", list.Page.Limit)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/maps/"+mapID.String(), nil)
	handler.GetMap(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var detail maps.MapResponse
	if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatalf("failed to decode detail response: %v", err)
	}
	if detail.Map.ID != mapID {
		t.Fatalf("expected map id %v, got %v", mapID, detail.Map.ID)
	}
}
