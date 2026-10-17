package earn

import "github.com/hostrepute/reputecast/internal/payments"

// DefaultProvider is the payment provider connections use when none is
// named: the only one available so far. Earn reaches every provider through
// internal/payments and never imports an adapter.
const DefaultProvider = "stripe"

// dashboard links to the provider's dashboard ("" for a provider without one).
func dashboard(provider string, mode payments.Mode, kind, id string) string {
	d, ok := payments.Lookup(provider)
	if !ok || d.Dashboard == nil {
		return ""
	}
	return d.Dashboard(mode, kind, id)
}

// keyHint is the part of a key that's safe to show.
func keyHint(provider, key string) string {
	if d, ok := payments.Lookup(provider); ok && d.Hint != nil {
		return d.Hint(key)
	}
	if len(key) > 4 {
		return "…" + key[len(key)-4:]
	}
	return "…"
}
