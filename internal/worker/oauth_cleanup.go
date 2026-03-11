package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/brady1408/atlinks/internal/riverargs"
	"github.com/brady1408/atlinks/internal/store"
	"github.com/riverqueue/river"
)

// OAuthCleanupArgs is re-exported from riverargs for convenience.
type OAuthCleanupArgs = riverargs.OAuthCleanupArgs

type OAuthCleanupWorker struct {
	river.WorkerDefaults[OAuthCleanupArgs]
	DeviceCodeStore   *store.DeviceCodeStore
	RefreshTokenStore *store.RefreshTokenStore
}

func (w *OAuthCleanupWorker) Work(ctx context.Context, job *river.Job[OAuthCleanupArgs]) error {
	dc, err := w.DeviceCodeStore.CleanupExpired(ctx)
	if err != nil {
		return fmt.Errorf("oauth cleanup device codes: %w", err)
	}
	rt, err := w.RefreshTokenStore.CleanupExpired(ctx)
	if err != nil {
		return fmt.Errorf("oauth cleanup refresh tokens: %w", err)
	}
	log.Printf("oauth cleanup: deleted %d device codes, %d refresh tokens", dc, rt)
	return nil
}
