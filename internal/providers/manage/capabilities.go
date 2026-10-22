package manage

import (
	"strconv"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// Badges turns a capability snapshot into the chips of spec 5.2, in the
// order the design shows them (bulk terms, webhooks, domain API, suppression
// sync, batching, dedicated IPs, sandbox, quota discovery, pricing), in the
// reader's language.
func Badges(l i18n.Lang, c state.Snapshot, dedicatedIP bool) []ProviderCapabilityBadge {
	out := []ProviderCapabilityBadge{}
	add := func(kind string, m i18n.Msg, args i18n.Args) {
		out = append(out, ProviderCapabilityBadge{Label: m.Text(l, args), Kind: kind})
	}
	switch c.Bulk {
	case providers.BulkAllowed:
		add("yes", mBadgeBulkAllowed, nil)
	case providers.BulkBroadcastStreams:
		if c.BroadcastStream {
			add("yes", mBadgeBulkStream, nil)
		} else {
			add("partial", mBadgeBulkStreamsOnly, nil)
		}
	case providers.BulkYourPolicy:
		add("yes", mBadgeBulkYourPolicy, nil)
	default:
		add("no", mBadgeBulk, nil)
	}
	switch c.Webhooks {
	case "":
		add("no", mBadgeWebhooks, nil)
	case "Bounce mailbox":
		add("yes", mBadgeBounceMailbox, nil)
	default:
		add("yes", mBadgeWebhooks, nil)
	}
	switch c.DomainAPI {
	case "full":
		add("yes", mBadgeDomainAPI, nil)
	case "dkim_only":
		add("partial", mBadgeDomainDKIM, nil)
	default:
		add("no", mBadgeDomainAPI, nil)
	}
	if c.SuppressionSync {
		add("yes", mBadgeSuppressionSync, nil)
	} else {
		add("no", mBadgeSuppressionSync, nil)
	}
	if c.BatchSize > 1 {
		// English keeps its digits as they were ("1000 per call").
		var n any = c.BatchSize
		if l.Locale == i18n.English {
			n = strconv.Itoa(c.BatchSize)
		}
		add("yes", mBadgeBatchPer, i18n.Args{"count": n})
	} else {
		add("no", mBadgeBatch, nil)
	}
	switch {
	case dedicatedIP:
		add("yes", mBadgeDedicatedIP, nil)
	case c.DedicatedIPs != "":
		add("partial", mBadgeDedicatedOption, i18n.Args{"option": providers.TextIn(c.DedicatedIPs, c.DedicatedIPsText, l)})
	default:
		add("no", mBadgeDedicatedIPs, nil)
	}
	switch c.Sandbox {
	case "detection":
		add("yes", mBadgeSandboxDetection, nil)
	case "test token":
		add("yes", mBadgeSandboxToken, nil)
	case "mode":
		add("yes", mBadgeSandboxMode, nil)
	case "test addresses":
		add("yes", mBadgeSandboxAddresses, nil)
	default:
		add("no", mBadgeSandboxMode, nil)
	}
	switch {
	case c.QuotaDiscovery && c.RateDiscovery:
		add("yes", mBadgeQuotaRate, nil)
	case c.QuotaDiscovery:
		add("yes", mBadgeQuota, nil)
	default:
		add("partial", mBadgeQuotaManual, nil)
	}
	if c.Pricing != "" {
		add("info", mBadgePricing, i18n.Args{"pricing": providers.TextIn(c.Pricing, c.PricingText, l)})
	}
	return out
}

// providerBadges is Badges for a connected provider.
func providerBadges(l i18n.Lang, p store.Provider) []ProviderCapabilityBadge {
	return Badges(l, state.Caps(p), p.IpMode == "dedicated")
}
