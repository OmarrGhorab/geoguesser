package workers

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// Job is a single worker tick. It must respect ctx cancellation/timeout.
type Job func(ctx context.Context) error

// Runner executes a job on a fixed interval with panic isolation, a per-tick
// timeout, graceful stop that waits for in-flight work, and simple counters.
type Runner struct {
	name     string
	interval time.Duration
	timeout  time.Duration
	job      Job
	logger   *slog.Logger

	successCount atomic.Uint64
	failCount    atomic.Uint64
	lastSuccess  atomic.Int64 // unix nanoseconds
	lastFailure  atomic.Int64 // unix nanoseconds

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// Option configures a Runner.
type Option func(*Runner)

// WithName sets a diagnostic name used in log fields.
func WithName(name string) Option {
	return func(r *Runner) {
		r.name = name
	}
}

// WithLogger overrides the default slog logger.
func WithLogger(logger *slog.Logger) Option {
	return func(r *Runner) {
		if logger != nil {
			r.logger = logger
		}
	}
}

// NewRunner builds a periodic worker. interval and timeout must be positive;
// job must be non-nil.
func NewRunner(interval, timeout time.Duration, job Job, opts ...Option) *Runner {
	r := &Runner{
		name:     "worker",
		interval: interval,
		timeout:  timeout,
		job:      job,
		logger:   slog.Default(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
	return r
}

// Start begins the loop bound to ctx. Calling Start twice is a no-op after the
// first successful start until Stop completes.
func (r *Runner) Start(ctx context.Context) {
	if r == nil || r.job == nil {
		return
	}
	if r.interval <= 0 {
		r.interval = time.Second
	}
	if r.timeout <= 0 {
		r.timeout = r.interval
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.started = true
	r.wg.Add(1)
	go r.loop(runCtx)
}

// Stop cancels the loop and waits for any in-flight tick to finish.
// It is safe to call multiple times and before Start.
func (r *Runner) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	cancel := r.cancel
	started := r.started
	r.mu.Unlock()
	if !started {
		return
	}
	if cancel != nil {
		cancel()
	}
	r.wg.Wait()
	r.mu.Lock()
	r.started = false
	r.cancel = nil
	r.mu.Unlock()
}

// SuccessCount returns how many ticks completed without error or panic.
func (r *Runner) SuccessCount() uint64 {
	if r == nil {
		return 0
	}
	return r.successCount.Load()
}

// FailCount returns how many ticks failed or panicked.
func (r *Runner) FailCount() uint64 {
	if r == nil {
		return 0
	}
	return r.failCount.Load()
}

// LastSuccessAt returns the wall time of the last successful tick, or zero.
func (r *Runner) LastSuccessAt() time.Time {
	if r == nil {
		return time.Time{}
	}
	ns := r.lastSuccess.Load()
	if ns == 0 {
		return time.Time{}
	}
	return time.Unix(0, ns)
}

// LastFailureAt returns the wall time of the last failed/panicked tick, or zero.
func (r *Runner) LastFailureAt() time.Time {
	if r == nil {
		return time.Time{}
	}
	ns := r.lastFailure.Load()
	if ns == 0 {
		return time.Time{}
	}
	return time.Unix(0, ns)
}

func (r *Runner) loop(ctx context.Context) {
	defer r.wg.Done()

	// Run once immediately so short-lived processes and tests observe work
	// without waiting a full interval, then continue on the ticker.
	r.runOnce(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.runOnce(ctx)
		}
	}
}

func (r *Runner) runOnce(parent context.Context) {
	if parent.Err() != nil {
		return
	}

	defer func() {
		if rec := recover(); rec != nil {
			r.failCount.Add(1)
			r.lastFailure.Store(time.Now().UnixNano())
			r.logger.Error("worker panic recovered",
				slog.String("worker", r.name),
				slog.Any("recover", rec),
			)
		}
	}()

	ctx, cancel := context.WithTimeout(parent, r.timeout)
	defer cancel()

	if err := r.job(ctx); err != nil {
		r.failCount.Add(1)
		r.lastFailure.Store(time.Now().UnixNano())
		r.logger.Error("worker tick failed",
			slog.String("worker", r.name),
			slog.Any("error", err),
		)
		return
	}
	r.successCount.Add(1)
	r.lastSuccess.Store(time.Now().UnixNano())
}
