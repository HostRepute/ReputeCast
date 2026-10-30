package providers

import "time"

// IdempotentSender is implemented by adapters whose provider recognizes a
// message it already took when the same message is sent again: each call
// carries an idempotency key derived from the ReputeCast message id alone
// (never from a batch), and the provider keeps the keys for a while.
//
// The sending engine relies on it after a send whose answer was lost (the
// worker stopped in the middle of the call, the result couldn't be
// recorded): within the window it sends such a message again, one per call,
// and the provider drops the copy if it took the first one. Without it, or
// after the window, the message is "in doubt" and a person decides, because
// sending it again could deliver it twice (review GS-1).
type IdempotentSender interface {
	// IdempotencyWindow is how long the provider keeps a key: a message sent
	// again within it is never delivered twice.
	IdempotencyWindow() time.Duration
}

// IdempotencyWindow is p's idempotency window, 0 when p doesn't dedupe.
func IdempotencyWindow(p Provider) time.Duration {
	if s, ok := p.(IdempotentSender); ok {
		return max(s.IdempotencyWindow(), 0)
	}
	return 0
}
