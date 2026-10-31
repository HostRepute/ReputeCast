package sparkpost

import (
	"net/http"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
)

var _ providers.IdempotentSender = (*Provider)(nil)

// IdempotencyWindow implements providers.IdempotentSender: every message is
// its own transmission carrying Idempotency-Key "rc-" + the ReputeCast
// message id, and SparkPost keeps a key for 24 hours (Transmissions API ›
// Idempotency; checked 2026-10-28). Within it the same request again is
// answered with the first one's response (X-Idempotent-Replayed: true)
// instead of a new transmission, and a request that differs answers 409
// with code 1600 (see accepted). The engine stays inside the window with a
// margin.
func (p *Provider) IdempotencyWindow() time.Duration { return 24 * time.Hour }

// SparkPost's 409 codes for a reused Idempotency-Key.
const (
	// codeKeyUsed: the key belongs to a request SparkPost already completed,
	// with another body (the message was rendered again since).
	codeKeyUsed = "1600"
	// codeKeyInProgress: the request the key belongs to is still running.
	codeKeyInProgress = "1601"
)

// accepted reports SparkPost refusing a request because its key belongs to
// one it already completed: an earlier attempt handed the message over, so
// it counts as sent. (The same body again is replayed, not refused.)
func accepted(ce *providers.CallError) bool {
	return ce.Status == http.StatusConflict && ce.Code == codeKeyUsed
}
