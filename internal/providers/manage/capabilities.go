package manage

import (
	"strconv"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// Badges turns a capability snapshot into the chips of spec 5.2, in the
// order the design shows them: bulk terms, webhooks, domain API, suppression
// sync, batching, dedicated IPs, sandbox, quota discovery, pricing.
func Badges(c state.Snapshot, dedicatedIP bool) []CapabilityBadge {
	out := []CapabilityBadge{}
	add := func(kind, label string) { out = append(out, CapabilityBadge{Label: label, Kind: kind}) }
	switch c.Bulk {
	case providers.BulkAllowed:
		add("yes", "Bulk marketing allowed")
	case providers.BulkBroadcastStreams:
		if c.BroadcastStream {
			add("yes", "Bulk marketing · Broadcast stream")
		} else {
			add("partial", "Bulk marketing · Broadcast streams only")
		}
	case providers.BulkYourPolicy:
		add("yes", "Bulk marketing · your own policy")
	default:
		add("no", "Bulk marketing")
	}
	switch c.Webhooks {
	case "":
		add("no", "Bounce & complaint webhooks")
	case "Bounce mailbox":
		add("yes", "Bounce mailbox")
	default:
		add("yes", "Bounce & complaint webhooks")
	}
	switch c.DomainAPI {
	case "full":
		add("yes", "Domain setup via API")
	case "dkim_only":
		add("partial", "Domain setup via API · DKIM only")
	default:
		add("no", "Domain setup via API")
	}
	if c.SuppressionSync {
		add("yes", "Suppression-list sync")
	} else {
		add("no", "Suppression-list sync")
	}
	if c.BatchSize > 1 {
		add("yes", "Batch sending · "+strconv.Itoa(c.BatchSize)+" per call")
	} else {
		add("no", "Batch sending")
	}
	switch {
	case dedicatedIP:
		add("yes", "Dedicated IP")
	case c.DedicatedIPs != "":
		add("partial", "Dedicated IPs · "+c.DedicatedIPs)
	default:
		add("no", "Dedicated IPs")
	}
	switch c.Sandbox {
	case "detection":
		add("yes", "Sandbox detection")
	case "test token":
		add("yes", "Sandbox · test token")
	case "mode":
		add("yes", "Sandbox mode")
	default:
		add("no", "Sandbox mode")
	}
	switch {
	case c.QuotaDiscovery && c.RateDiscovery:
		add("yes", "Quota & rate discovery")
	case c.QuotaDiscovery:
		add("yes", "Quota discovery")
	default:
		add("partial", "Quota & rate · set manually")
	}
	if c.Pricing != "" {
		add("info", "Pricing · "+c.Pricing)
	}
	return out
}

// providerBadges is Badges for a connected provider.
func providerBadges(p store.Provider) []CapabilityBadge {
	return Badges(state.Caps(p), p.IpMode == "dedicated")
}
