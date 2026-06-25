package email

import "context"

// Message is a transactional email to be sent.
type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// Sender is the abstraction for transactional email delivery.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}
