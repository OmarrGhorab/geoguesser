package realtime

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// RedisTicketStore is satisfied by platform/redis.RealtimeStore methods used here.
// Defined as a narrow interface so realtime does not import platform packages in tests.
type RedisTicketBackend interface {
	IssueTicket(ctx context.Context, userID uuid.UUID, channelKind, channelID string, ttl time.Duration) (string, error)
	ConsumeTicket(ctx context.Context, opaqueToken string) (userID uuid.UUID, channelKind, channelID string, expiresAt time.Time, err error)
}

// RealtimeTicketAdapter adapts a Redis-backed store to TicketStore.
// The concrete backend is typically *redis.RealtimeStore wrapped at wiring time.
type RealtimeTicketAdapter struct {
	IssueFn   func(ctx context.Context, userID uuid.UUID, channelKind, channelID string, ttl time.Duration) (string, error)
	ConsumeFn func(ctx context.Context, token string) (TicketClaims, error)
}

// Issue implements TicketIssuer.
func (a RealtimeTicketAdapter) Issue(ctx context.Context, userID uuid.UUID, channelKind, channelID string, ttl time.Duration) (string, time.Time, error) {
	if a.IssueFn == nil {
		return "", time.Time{}, errors.New("ticket issuer unavailable")
	}
	token, err := a.IssueFn(ctx, userID, channelKind, channelID, ttl)
	if err != nil {
		return "", time.Time{}, err
	}
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return token, time.Now().UTC().Add(ttl), nil
}

// Consume implements TicketValidator.
func (a RealtimeTicketAdapter) Consume(ctx context.Context, token string) (*TicketClaims, error) {
	if a.ConsumeFn == nil {
		return nil, errors.New("ticket consumer unavailable")
	}
	claims, err := a.ConsumeFn(ctx, token)
	if err != nil {
		return nil, err
	}
	return &claims, nil
}
