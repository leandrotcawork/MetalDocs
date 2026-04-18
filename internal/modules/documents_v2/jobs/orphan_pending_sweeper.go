package jobs

import (
	"context"
	"log"
	"time"

	"metaldocs/internal/modules/documents_v2/repository"
)

func StartOrphanPendingSweeper(ctx context.Context, r *repository.Repository, interval time.Duration) (stop func()) {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().Add(-24 * time.Hour)
				n, err := r.DeleteExpiredPending(ctx, cutoff)
				if err != nil {
					log.Printf("orphan_pending_sweeper error: %v", err)
					continue
				}
				if n > 0 {
					log.Printf("orphan_pending_sweeper deleted=%d", n)
				}
			}
		}
	}()
	return cancel
}
