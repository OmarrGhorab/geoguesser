package users

import (
	"time"

	"github.com/google/uuid"
)

// User is a read-only view of a registered account for the users package.
type User struct {
	ID          uuid.UUID
	Email       string
	Role        string
	Status      string
	DisplayName string
	CreatedAt   time.Time
}
