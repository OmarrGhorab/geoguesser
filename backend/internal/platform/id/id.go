package id

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

// Generator produces identifiers. The interface isolates feature code from the
// underlying ID scheme and enables deterministic generators in tests.
type Generator interface {
	NewUUID() (uuid.UUID, error)
	NewEventID() string
}

// Default is the production ID generator.
type Default struct{}

// NewDefault returns a production ID generator.
func NewDefault() *Default {
	return &Default{}
}

// NewUUID returns a new UUID v7, which is sortable by creation time.
func (d *Default) NewUUID() (uuid.UUID, error) {
	return uuid.NewV7()
}

// NewEventID returns a unique realtime event identifier with a stable prefix.
func (d *Default) NewEventID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// rand.Read is documented to never return an error for the default source,
		// but defensively panic because an ID generator cannot continue otherwise.
		panic(fmt.Sprintf("id: failed to read random bytes: %v", err))
	}
	return "evt_" + hex.EncodeToString(b)
}

// Fixed is a deterministic generator for tests.
type Fixed struct {
	UUID      uuid.UUID
	EventID   string
	UUIDError error
}

// NewUUID returns the configured fixed UUID or error.
func (f *Fixed) NewUUID() (uuid.UUID, error) {
	return f.UUID, f.UUIDError
}

// NewEventID returns the configured fixed event ID.
func (f *Fixed) NewEventID() string {
	return f.EventID
}
