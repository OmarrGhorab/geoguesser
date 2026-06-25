package clock

import "time"

// Clock provides a testable abstraction over the system clock.
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Until(t time.Time) time.Duration
}

// System uses the host system clock. Use this in production.
type System struct{}

// NewSystem returns a production system clock.
func NewSystem() System {
	return System{}
}

// Now returns the current UTC time.
func (System) Now() time.Time {
	return time.Now().UTC()
}

// Since returns the time elapsed since t.
func (System) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// Until returns the duration until t.
func (System) Until(t time.Time) time.Duration {
	return time.Until(t)
}

// Fixed returns a clock that always reports the supplied time. Useful in tests.
func Fixed(t time.Time) *FixedClock {
	return &FixedClock{now: t}
}

// FixedClock is a deterministic clock for tests.
type FixedClock struct {
	now time.Time
}

// Now returns the fixed time.
func (f *FixedClock) Now() time.Time {
	return f.now
}

// Since returns the duration since t relative to the fixed time.
func (f *FixedClock) Since(t time.Time) time.Duration {
	return f.now.Sub(t)
}

// Until returns the duration until t relative to the fixed time.
func (f *FixedClock) Until(t time.Time) time.Duration {
	return t.Sub(f.now)
}

// Advance moves the fixed clock forward by d.
func (f *FixedClock) Advance(d time.Duration) {
	f.now = f.now.Add(d)
}
