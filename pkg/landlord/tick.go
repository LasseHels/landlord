package landlord

import (
	"context"
	"time"
)

// tick executes the given tick function immediately upon invocation and every interval after that.
//
// tick runs until context is done.
//
// Ticks are dropped if a tick takes longer than interval. When a slow tick (i.e., one that takes longer than interval)
// finishes, another tick runs immediately.
func tick(ctx context.Context, t func(context.Context), interval time.Duration) {
	// Ticker doesn't immediately tick, so we manually ensure an immediate tick here.
	t(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t(ctx)
		}
	}
}
