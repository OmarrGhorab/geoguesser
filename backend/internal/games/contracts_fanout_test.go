package games

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type recordingOutcomeSink struct {
	calls int
	err   error
}

func (s *recordingOutcomeSink) PublishMultiplayerOutcome(context.Context, uuid.UUID, MultiplayerGuessOutcome) error {
	s.calls++
	return s.err
}

func TestMultiplayerEventFanoutAttemptsEverySink(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("degraded sink")
	first := &recordingOutcomeSink{err: wantErr}
	second := &recordingOutcomeSink{}
	err := (MultiplayerEventFanout{first, second}).PublishMultiplayerOutcome(context.Background(), uuid.New(), MultiplayerGuessOutcome{RoundCompleted: true})
	if !errors.Is(err, wantErr) || first.calls != 1 || second.calls != 1 {
		t.Fatalf("err=%v first=%d second=%d", err, first.calls, second.calls)
	}
}
