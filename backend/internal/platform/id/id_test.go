package id_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/raven/geoguess/backend/internal/platform/id"
)

func TestDefaultNewUUID(t *testing.T) {
	g := id.NewDefault()
	u, err := g.NewUUID()
	if err != nil {
		t.Fatalf("NewUUID error = %v", err)
	}
	if u == uuid.Nil {
		t.Error("NewUUID returned nil UUID")
	}
	if u.Version() != 7 {
		t.Errorf("NewUUID version = %d, want 7", u.Version())
	}
}

func TestDefaultNewEventID(t *testing.T) {
	g := id.NewDefault()
	e := g.NewEventID()
	if !strings.HasPrefix(e, "evt_") {
		t.Errorf("NewEventID = %q, want evt_ prefix", e)
	}
	if len(e) <= len("evt_") {
		t.Errorf("NewEventID too short: %q", e)
	}
}

func TestFixedGenerator(t *testing.T) {
	want := uuid.MustParse("01900000-0000-7fff-8fff-ffffffffffff")
	g := &id.Fixed{
		UUID:    want,
		EventID: "evt_test_123",
	}

	got, err := g.NewUUID()
	if err != nil {
		t.Fatalf("Fixed.NewUUID error = %v", err)
	}
	if got != want {
		t.Errorf("Fixed.NewUUID = %v, want %v", got, want)
	}
	if g.NewEventID() != "evt_test_123" {
		t.Errorf("Fixed.NewEventID = %q, want evt_test_123", g.NewEventID())
	}
}
