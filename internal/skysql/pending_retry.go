package skysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Defaults for waiting out a transient "service is in a pending state"
// rejection. The backend (dbprovision-service ensureThatServiceStatusAllowsUpdates)
// returns HTTP 400 with the message "service modification not allowed while
// service is in a pending state" whenever a mutating operation reaches a service
// while an earlier scale/config/power operation is still in flight. The HTTP
// layer does not retry 400s, so without this the later operation fails outright
// (MCDEV-3899). The rejection clears once the in-flight operation finishes, so
// we poll until the service is modifiable again.
const (
	defaultPendingRetryInterval = 10 * time.Second
	defaultPendingRetryTimeout  = 30 * time.Minute
)

// pendingStateMarker is the stable, distinctive fragment of the backend's
// legacy 400 pending-state rejection message. handleError uses it to classify
// that form (newer DPS returns a 409 Conflict instead) into
// ErrorServiceInPendingState.
const pendingStateMarker = "pending state"

// isPendingStateError reports whether err is the backend's rejection of a
// mutating operation because the target service is currently in a pending_*
// state. handleError classifies both the legacy 400 message and the newer 409
// Conflict into ErrorServiceInPendingState; the error resolves once the
// in-flight operation completes, so retrying is safe and desirable.
func isPendingStateError(err error) bool {
	return errors.Is(err, ErrorServiceInPendingState)
}

// doWithPendingRetry runs fn and, when it fails only because the service is in a
// pending state, waits and retries until the operation is accepted, fn fails for
// another reason, ctx is cancelled, or the configured timeout elapses. This
// serializes operations issued close together (for example a config change and a
// scaling operation) instead of failing the later one.
func (c *Client) doWithPendingRetry(ctx context.Context, fn func() error) error {
	interval := c.pendingRetryInterval
	if interval <= 0 {
		interval = defaultPendingRetryInterval
	}
	timeout := c.pendingRetryTimeout
	if timeout <= 0 {
		timeout = defaultPendingRetryTimeout
	}

	deadline := time.Now().Add(timeout)
	for attempt := 1; ; attempt++ {
		err := fn()
		if !isPendingStateError(err) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("service was still in a pending state after %s: %w", timeout, err)
		}

		tflog.Debug(ctx, "service is in a pending state, waiting before retrying the operation", map[string]interface{}{
			"attempt":  attempt,
			"retry_in": interval.String(),
		})

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}
