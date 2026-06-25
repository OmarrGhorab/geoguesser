package clock_test

import (
	"testing"
	"time"

	"github.com/raven/geoguess/backend/internal/platform/clock"
)

func TestSystemClock(t *testing.T) {
	c := clock.NewSystem()
	before := time.Now().UTC()
	now := c.Now()
	after := time.Now().UTC()

	if now.Before(before) || now.After(after) {
		t.Errorf("System.Now() = %v, not between %v and %v", now, before, after)
	}
}

func TestFixedClock(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := clock.Fixed(base)

	if got := c.Now(); !got.Equal(base) {
		t.Errorf("Fixed.Now() = %v, want %v", got, base)
	}

	c.Advance(time.Hour)
	want := base.Add(time.Hour)
	if got := c.Now(); !got.Equal(want) {
		t.Errorf("Fixed.Now() after advance = %v, want %v", got, want)
	}
}
