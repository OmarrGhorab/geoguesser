package uploads

import "time"

// nowUTC and since are thin wrappers so tests can stub time if needed later.
func nowUTC() time.Time { return time.Now().UTC() }

func since(start time.Time) time.Duration { return time.Since(start) }
