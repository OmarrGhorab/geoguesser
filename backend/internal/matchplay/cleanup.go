package matchplay

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// ObjectDeleter deletes private storage objects (sanitized derivatives or raw uploads).
// Implementations wrap platform/storage.Provider.
type ObjectDeleter interface {
	DeleteObject(ctx context.Context, key string) error
}

// CleanupConfig controls bounded chat/raw retention sweeps.
type CleanupConfig struct {
	BatchSize     int
	RawMaxAge     time.Duration
	DeleteRetries int
	RetryBackoff  time.Duration
}

// CleanupStats summarizes one cleanup pass (for tests/metrics; no PII).
type CleanupStats struct {
	MessagesSelected int
	MessagesDeleted  int
	FilesDeleted     int
	UploadsDeleted   int
	ObjectDeletes    int
	ObjectFailures   int
	SkippedProtected int
}

// Cleaner runs bounded retention cleanup for team chat and raw uploads.
type Cleaner struct {
	store   ChatStore
	objects ObjectDeleter
	logger  *slog.Logger
	metrics MetricsRecorder
	cfg     CleanupConfig
	clock   func() time.Time
}

// NewCleaner constructs a chat retention cleaner.
func NewCleaner(store ChatStore, objects ObjectDeleter, logger *slog.Logger, metrics MetricsRecorder, cfg CleanupConfig) *Cleaner {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	if cfg.BatchSize < 1 {
		cfg.BatchSize = DefaultCleanupBatchSize
	}
	if cfg.RawMaxAge <= 0 {
		cfg.RawMaxAge = RawUploadRetention
	}
	if cfg.DeleteRetries < 1 {
		cfg.DeleteRetries = DefaultCleanupDeleteRetries
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 50 * time.Millisecond
	}
	return &Cleaner{
		store:   store,
		objects: objects,
		logger:  logger,
		metrics: metrics,
		cfg:     cfg,
		clock:   func() time.Time { return time.Now().UTC() },
	}
}

// WithClock overrides wall time (tests).
func (c *Cleaner) WithClock(now func() time.Time) *Cleaner {
	if c != nil && now != nil {
		c.clock = now
	}
	return c
}

// RunOnce performs one bounded cleanup pass: expired chat rows then raw uploads.
func (c *Cleaner) RunOnce(ctx context.Context) (CleanupStats, error) {
	var stats CleanupStats
	if c == nil || c.store == nil {
		return stats, ErrUnavailable
	}
	now := c.clock()

	if err := c.cleanupMessages(ctx, now, &stats); err != nil {
		c.metrics.ObserveCleanup("messages", "error")
		return stats, err
	}
	if err := c.cleanupRawUploads(ctx, now, &stats); err != nil {
		c.metrics.ObserveCleanup("raw_uploads", "error")
		return stats, err
	}
	c.metrics.ObserveCleanup("messages", "ok")
	c.metrics.ObserveCleanup("raw_uploads", "ok")
	c.metrics.ObserveCleanupDeleted("messages", stats.MessagesDeleted)
	c.metrics.ObserveCleanupDeleted("files", stats.FilesDeleted)
	c.metrics.ObserveCleanupDeleted("uploads", stats.UploadsDeleted)
	c.metrics.ObserveCleanupDeleted("objects", stats.ObjectDeletes)
	return stats, nil
}

func (c *Cleaner) cleanupMessages(ctx context.Context, now time.Time, stats *CleanupStats) error {
	candidates, err := c.store.ListCleanupCandidates(ctx, now, c.cfg.BatchSize)
	if err != nil {
		return err
	}
	stats.MessagesSelected = len(candidates)
	if len(candidates) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(candidates))
	matchSet := make(map[uuid.UUID]struct{})
	for _, m := range candidates {
		ids = append(ids, m.ID)
		matchSet[m.MatchID] = struct{}{}
	}

	// Defense-in-depth: re-check legal hold / retention protection.
	protected, err := c.store.ListProtectedMessageIDs(ctx, ids, now)
	if err != nil {
		return err
	}
	eligible := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := protected[id]; ok {
			stats.SkippedProtected++
			continue
		}
		eligible = append(eligible, id)
	}
	if len(eligible) == 0 {
		return nil
	}

	// Load file metadata before deleting message rows so storage keys survive.
	fileIDSet := make(map[uuid.UUID]struct{})
	for _, m := range candidates {
		if _, skip := protected[m.ID]; skip {
			continue
		}
		if m.FileID != nil {
			fileIDSet[*m.FileID] = struct{}{}
		}
	}
	fileIDs := make([]uuid.UUID, 0, len(fileIDSet))
	for id := range fileIDSet {
		fileIDs = append(fileIDs, id)
	}
	files, err := c.store.GetChatFiles(ctx, fileIDs)
	if err != nil {
		return err
	}

	if _, err := c.store.DeleteMessages(ctx, eligible); err != nil {
		return err
	}
	stats.MessagesDeleted = len(eligible)

	if len(fileIDs) > 0 {
		keys := make([]string, 0, len(files)*2)
		for _, f := range files {
			if f.StorageKey != "" {
				keys = append(keys, f.StorageKey)
			}
			if f.RawStorageKey != nil && *f.RawStorageKey != "" {
				keys = append(keys, *f.RawStorageKey)
			}
		}
		ok, fail := c.deleteObjectsWithRetry(ctx, keys)
		stats.ObjectDeletes += ok
		stats.ObjectFailures += fail
		// Only drop file rows when all associated objects deleted successfully for that batch.
		if fail == 0 {
			if err := c.store.DeleteFiles(ctx, fileIDs); err != nil {
				c.logger.WarnContext(ctx, "chat cleanup delete file rows failed", slog.Any("error", err))
			} else {
				stats.FilesDeleted += len(fileIDs)
			}
		} else {
			c.metrics.ObserveDependencyFailure("storage")
		}
	}

	// Best-effort mute cleanup for matches that had expired chat.
	if len(matchSet) > 0 {
		matchIDs := make([]uuid.UUID, 0, len(matchSet))
		for id := range matchSet {
			matchIDs = append(matchIDs, id)
		}
		if err := c.store.DeleteMutesForMatches(ctx, matchIDs); err != nil {
			c.logger.WarnContext(ctx, "chat cleanup delete mutes failed", slog.Any("error", err))
		}
	}
	return nil
}

func (c *Cleaner) cleanupRawUploads(ctx context.Context, now time.Time, stats *CleanupStats) error {
	cutoff := now.Add(-c.cfg.RawMaxAge)
	uploads, err := c.store.ListRawUploadsForCleanup(ctx, cutoff, c.cfg.BatchSize)
	if err != nil {
		return err
	}
	if len(uploads) == 0 {
		return nil
	}

	keys := make([]string, 0, len(uploads)*2)
	deleteIDs := make([]uuid.UUID, 0, len(uploads))
	clearRawIDs := make([]uuid.UUID, 0, len(uploads))
	for _, u := range uploads {
		if u.Status == UploadStatusCompleted {
			clearRawIDs = append(clearRawIDs, u.ID)
		} else {
			deleteIDs = append(deleteIDs, u.ID)
		}
		if u.Status != UploadStatusCompleted && u.StorageKey != "" {
			keys = append(keys, u.StorageKey)
		}
		if u.RawStorageKey != nil && *u.RawStorageKey != "" {
			keys = append(keys, *u.RawStorageKey)
		}
	}
	ok, fail := c.deleteObjectsWithRetry(ctx, keys)
	stats.ObjectDeletes += ok
	stats.ObjectFailures += fail
	if fail > 0 {
		c.metrics.ObserveDependencyFailure("storage")
		// Still attempt row cleanup for successful deletes; keep rows when objects remain.
		// For simplicity and safety, only delete upload rows when every object delete succeeded.
		return nil
	}
	if err := c.store.ClearUploadRawStorageKeys(ctx, clearRawIDs); err != nil {
		return err
	}
	if err := c.store.DeleteUploads(ctx, deleteIDs); err != nil {
		return err
	}
	stats.UploadsDeleted += len(deleteIDs)
	return nil
}

func (c *Cleaner) deleteObjectsWithRetry(ctx context.Context, keys []string) (ok, fail int) {
	if c.objects == nil {
		// No storage configured: count as failures so callers retain DB rows when needed.
		return 0, len(keys)
	}
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		if err := c.deleteOne(ctx, key); err != nil {
			c.logger.WarnContext(ctx, "chat cleanup object delete failed",
				slog.String("key_prefix", redactStorageKey(key)),
				slog.Any("error", err),
			)
			fail++
			continue
		}
		ok++
	}
	return ok, fail
}

func (c *Cleaner) deleteOne(ctx context.Context, key string) error {
	var last error
	for attempt := 0; attempt < c.cfg.DeleteRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		last = c.objects.DeleteObject(ctx, key)
		if last == nil {
			return nil
		}
		// Brief backoff between retries (tests may use zero clock advancement).
		if c.cfg.RetryBackoff > 0 && attempt+1 < c.cfg.DeleteRetries {
			timer := time.NewTimer(c.cfg.RetryBackoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return last
}

// redactStorageKey keeps logs free of full object keys while remaining debuggable.
func redactStorageKey(key string) string {
	if len(key) <= 12 {
		return "***"
	}
	return key[:8] + "…"
}
