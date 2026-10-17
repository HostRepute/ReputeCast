package deliverability

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/providers"
	provstate "github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// Provider figures come from internal/providers/state and manage.UsageOf
// (state.usage), so Deliverability and Sending providers show the same
// numbers.

// providerConfig is the part of providers.config account health reads.
type providerConfig struct {
	Region        string `json:"region"`
	MessageStream string `json:"message_stream"`
}

func configOf(p store.Provider) providerConfig {
	var c providerConfig
	_ = json.Unmarshal(p.Config, &c)
	return c
}

// vdmOn reports whether SES's Virtual Deliverability Manager is on, when the
// providers health check records it (health.vdm); off otherwise.
func vdmOn(p store.Provider) bool {
	var h struct {
		VDM bool `json:"vdm"`
	}
	_ = json.Unmarshal(p.Health, &h)
	return h.VDM
}

// selfHosted adapters have no account limits: ReputeCast sets the guards.
func selfHosted(adapter string) bool {
	if d, ok := providers.Lookup(adapter); ok {
		return d.Kind == "self_hosted" || d.Kind == "smtp"
	}
	return adapter == "postal" || adapter == "smtp"
}

// statusLabel is the badge word of a provider status.
func statusLabel(status string) string {
	switch status {
	case provstate.Healthy:
		return "Healthy"
	case provstate.Degraded:
		return "Degraded"
	case provstate.Down:
		return "Down"
	case provstate.Sandbox:
		return "Sandbox"
	case provstate.WarmingUp:
		return "Warming up"
	case provstate.Paused:
		return "Paused"
	}
	return status
}

func statusTone(status string) string {
	switch status {
	case provstate.Healthy:
		return "ok"
	case provstate.WarmingUp:
		return "info"
	case provstate.Degraded, provstate.Sandbox:
		return "warn"
	case provstate.Down:
		return "bad"
	}
	return "neutral"
}

// primaryStreams returns the streams a provider is primary for.
func (st *state) primaryStreams(p store.Provider) []string {
	var out []string
	for _, s := range st.streams {
		if s.ProviderID == p.ID && !s.Fallback && !slices.Contains(out, s.Stream) {
			out = append(out, s.Stream)
		}
	}
	slices.Sort(out)
	return out
}

// pools lists the shared IP pools (checked at domain level).
func (st *state) pools() []DeliverabilityPoolHealth {
	out := []DeliverabilityPoolHealth{}
	domain := st.mainDomain()
	pm := st.latestPostmaster(domain)
	for _, p := range st.providers {
		if p.IpMode != "shared" {
			continue
		}
		u := st.usage[p.ID]
		pool := DeliverabilityPoolHealth{ProviderID: p.ID.String(), Provider: p.Name, Short: provstate.Short(p.Name, p.Adapter), Adapter: p.Adapter,
			Region: configOf(p).Region, Status: p.Status, ManagedReputation: p.Adapter == "ses", GmailIPReputation: pm.IPReputation,
			BounceRate: u.Bounce7d, ComplaintRate: u.Complaint7d}
		if s := st.primaryStreams(p); len(s) == 1 {
			pool.OnlyStream = s[0]
		}
		pool.AccountStatus = accountWord(p)
		if opt, ok := strings.CutPrefix(provstate.Caps(p).DedicatedIPs, "optional, "); ok {
			pool.DedicatedIPOption = opt
		}
		// Plan providers count the month's volume against the plan.
		if h := provstate.HealthOf(p); h.Quota != nil && h.Quota.Kind == "monthly_plan" {
			v := h.Quota.Used
			pool.VolumeThisMonth = &v
		}
		out = append(out, pool)
	}
	return out
}

// accountWord is the provider's own word for its account standing: the
// approval state Postmark reports ("Approved"), else the status label.
func accountWord(p store.Provider) string {
	if p.Adapter == "postmark" && p.Status == provstate.Healthy {
		standing := provstate.HealthOf(p).Account
		if standing == "" {
			standing = p.StatusDetail
		}
		if w, _, _ := strings.Cut(standing, " · "); w != "" {
			return w
		}
	}
	return statusLabel(p.Status)
}

// providerHealth is account health per sending provider.
func (st *state) providerHealth() []DeliverabilityProviderHealth {
	out := []DeliverabilityProviderHealth{}
	for _, p := range st.providers {
		u := st.usage[p.ID]
		limits := provstate.LimitsOf(p)
		h := DeliverabilityProviderHealth{ProviderID: p.ID.String(), Name: p.Name, Adapter: p.Adapter, Status: p.Status, Tone: statusTone(p.Status)}
		if selfHosted(p.Adapter) {
			h.Name += " (self-hosted)"
		}
		h.Badge = accountWord(p)
		if p.Adapter == "ses" && p.Status == provstate.Healthy {
			// SES reports enforcement: "Production · no enforcement".
			if _, enf, ok := strings.Cut(provstate.HealthOf(p).Account, " · "); ok && enf != "" {
				h.Badge += " · " + enf
			}
		}
		complaintLabel := "Complaint rate"
		if p.Adapter == "postmark" {
			complaintLabel = "Spam complaint rate"
		}
		h.Bounce = rateLimit("Bounce rate", u.Bounce7d, limits.Bounce, p.Adapter)
		h.Complaint = rateLimit(complaintLabel, u.Complaint7d, limits.Complaint, p.Adapter)
		switch {
		case p.Adapter == "ses":
			v := "Bounce and complaint metrics only; no VDM"
			if vdmOn(p) {
				v = "Virtual Deliverability Manager on"
			}
			h.Note = &DeliverabilityProviderNote{Label: "Reputation dashboard", Value: v}
		case p.Adapter == "postmark":
			kind := "Transactional"
			if strings.Contains(strings.ToLower(configOf(p).MessageStream), "broadcast") || provstate.Caps(p).BroadcastStream {
				kind = "Broadcast"
			}
			v := kind
			switch s := st.primaryStreams(p); len(s) {
			case 1:
				v += " · " + streamName(s[0]) + " mail only"
			case 2:
				// Postmark often carries System and Transactional mail.
				v += " · " + streamName(s[0]) + " and " + streamName(s[1]) + " mail"
			}
			h.Note = &DeliverabilityProviderNote{Label: "Stream", Value: v}
		case selfHosted(p.Adapter):
			h.Note = &DeliverabilityProviderNote{Label: "Thresholds", Value: "Set by ReputeCast; " + p.Name + " has no account limits"}
		}
		out = append(out, h)
	}
	return out
}

func streamName(s string) string {
	switch s {
	case "broadcast":
		return "Broadcast"
	case "automation":
		return "Automation"
	case "system":
		return "System"
	case "transactional":
		return "Transactional"
	}
	return s
}

// pctText formats a threshold: 5 → "5%", 0.1 → "0.1%".
func pctText(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64) + "%"
}

// rateLimit describes a rate against the provider's thresholds: SES has a
// review and a pause level, Postmark a limit, self-hosted servers the guard
// ReputeCast sets (the warm-up guard while warming up).
func rateLimit(label string, value *float64, l provstate.Limit, adapter string) DeliverabilityRateLimit {
	out := DeliverabilityRateLimit{Label: label, Value: value, Tone: "ok"}
	first := l.Limit
	switch {
	case l.Review > 0 && l.Pause > 0:
		out.Thresholds = fmt.Sprintf("review %s, pause %s", pctText(l.Review), pctText(l.Pause))
		first = l.Review
	case strings.Contains(l.Label, "guard"):
		out.Thresholds = l.Label + " " + pctText(l.Limit)
	case selfHosted(adapter):
		out.Thresholds = "guard " + pctText(l.Limit)
	default:
		out.Thresholds = "limit " + pctText(l.Limit)
	}
	if first > 0 && value != nil {
		out.Usage = round(*value/first*100, 1)
		switch {
		case out.Usage >= 100:
			out.Tone = "bad"
		case out.Usage >= 50:
			out.Tone = "warn"
		}
	}
	return out
}
