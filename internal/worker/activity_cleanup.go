package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brady1408/atlinks/internal/riverargs"
	"github.com/brady1408/atlinks/internal/store"
	"github.com/riverqueue/river"
)

// ActivityCleanupArgs is re-exported from riverargs for convenience.
type ActivityCleanupArgs = riverargs.ActivityCleanupArgs

type ActivityCleanupWorker struct {
	river.WorkerDefaults[ActivityCleanupArgs]
	ActivityStore *store.ActivityStore
}

func (w *ActivityCleanupWorker) Work(ctx context.Context, job *river.Job[ActivityCleanupArgs]) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -30)
	n, err := w.ActivityStore.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("activity cleanup: %w", err)
	}
	log.Printf("activity cleanup: deleted %d rows older than %s", n, cutoff.Format(time.DateOnly))
	return nil
}
