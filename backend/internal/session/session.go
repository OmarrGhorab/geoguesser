package session

// Kind indicates the type of session attached to a request.
type Kind string

const (
	KindAnonymous Kind = "anonymous"
	KindGuest     Kind = "guest"
	KindUser      Kind = "user"
)

// Context holds resolved session information for a request.
type Context struct {
	Kind      Kind
	UserID    *string
	GuestID   *string
	SessionID *string
	Role      string
}

// IsRegistered returns true for registered user sessions.
func (s Context) IsRegistered() bool {
	return s.Kind == KindUser && s.UserID != nil
}

// IsGuest returns true for guest sessions.
func (s Context) IsGuest() bool {
	return s.Kind == KindGuest && s.GuestID != nil
}

// IsAdmin returns true for admin sessions.
func (s Context) IsAdmin() bool {
	return s.IsRegistered() && s.Role == "admin"
}
