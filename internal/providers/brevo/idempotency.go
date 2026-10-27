package brevo

import (
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
)

var _ providers.IdempotentSender = (*Provider)(nil)

// IdempotencyWindow implements providers.IdempotentSender: every message is
// its own call carrying IdempotencyKey = the ReputeCast message id, and
// Brevo refuses a second request with the same key for 30 minutes (see
// duplicate). The engine stays inside the window with a margin.
func (p *Provider) IdempotencyWindow() time.Duration { return 30 * time.Minute }
