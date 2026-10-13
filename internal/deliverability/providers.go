package deliverability

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/store"
)

// providerData is the part of a provider's stored snapshot (written by the
// providers domain) that account health reads.
type providerData struct {
	Config struct {
		Region        string `json:"region"`
		MessageStream string `json:"message_stream"`
	}
	Limits map[string]struct {
		Limit  *float64 `json:"limit"`
		Review *float64 `json:"review"`
		Pause  *float64 `json:"pause"`
		Label  string   `json:"label"`
	}
	Health struct {
		BounceRate    *float64 `json:"bounce_rate"`
		ComplaintRate *float64 `json:"complaint_rate"`
		Account       string   `json:"account"`
		VDM           bool     `json:"vdm"`
		Quota         struct {
			Kind string `json:"kind"`
			Used *int64 `json:"used"`
		} `json:"quota"`
	}
	Capabilities struct {
		DedicatedIPs string `json:"dedicated_ips"`
	}
}

func readProvider(p store.HostReputeActiveProvidersRow) providerData {
	var d providerData
	_ = json.Unmarshal(p.Config, &d.Config)
	_ = json.Unmarshal(p.Limits, &d.Limits)
	_ = json.Unmarshal(p.Health, &d.Health)
	_ = json.Unmarshal(p.Capabilities, &d.Capabilities)
	return d
}

// selfHosted adapters have no account limits: ReputeCast sets the guards.
func selfHosted(adapter string) bool { return adapter == "postal" || adapter == "smtp" }

// statusLabel is the badge word of a provider status.
func statusLabel(status string) string {
	switch status {
	case "healthy":
		return "Healthy"
	case "degraded":
		return "Degraded"
	case "down":
		return "Down"
	case "sandbox":
		return "Sandbox"
	case "warming_up":
		return "Warming up"
	case "paused":
		return "Paused"
	}
	return status
}

func statusTone(status string) string {
	switch status {
	case "healthy":
		return "ok"
	case "warming_up":
		return "info"
	case "degraded", "sandbox":
		return "warn"
	case "down":
		return "bad"
	}
	return "neutral"
}

// streams returns the streams a provider is primary for.
func (st *state) primaryStreams(p store.HostReputeActiveProvidersRow) []string {
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
		d := readProvider(p)
		pool := DeliverabilityPoolHealth{ProviderID: p.ID.String(), Provider: p.Name, Short: campaigns.ShortName(p.Name, p.Adapter), Adapter: p.Adapter,
			Region: d.Config.Region, Status: p.Status, ManagedReputation: p.Adapter == "ses", GmailIPReputation: pm.IPReputation,
			BounceRate: d.Health.BounceRate, ComplaintRate: d.Health.ComplaintRate}
		if s := st.primaryStreams(p); len(s) == 1 {
			pool.OnlyStream = s[0]
		}
		pool.AccountStatus = accountWord(p, d)
		if opt, ok := strings.CutPrefix(d.Capabilities.DedicatedIPs, "optional, "); ok {
			pool.DedicatedIPOption = opt
		}
		if d.Health.Quota.Kind == "monthly_plan" && d.Health.Quota.Used != nil {
			v := *d.Health.Quota.Used
			pool.VolumeThisMonth = &v
		}
		out = append(out, pool)
	}
	return out
}

// accountWord is the provider's own word for its account standing: the
// approval state Postmark reports ("Approved"), else the status label.
func accountWord(p store.HostReputeActiveProvidersRow, d providerData) string {
	if p.Adapter == "postmark" && p.Status == "healthy" {
		if w, _, _ := strings.Cut(p.StatusDetail, " · "); w != "" {
			return w
		}
	}
	return statusLabel(p.Status)
}

// providerHealth is account health per sending provider.
func (st *state) providerHealth() []DeliverabilityProviderHealth {
	out := []DeliverabilityProviderHealth{}
	for _, p := range st.providers {
		d := readProvider(p)
		h := DeliverabilityProviderHealth{ProviderID: p.ID.String(), Name: p.Name, Adapter: p.Adapter, Status: p.Status, Tone: statusTone(p.Status)}
		if selfHosted(p.Adapter) {
			h.Name += " (self-hosted)"
		}
		h.Badge = accountWord(p, d)
		if p.Adapter == "ses" && p.Status == "healthy" {
			// SES reports enforcement: "Production · no enforcement".
			if _, enf, ok := strings.Cut(d.Health.Account, " · "); ok && enf != "" {
				h.Badge += " · " + enf
			}
		}
		complaintLabel := "Complaint rate"
		if p.Adapter == "postmark" {
			complaintLabel = "Spam complaint rate"
		}
		h.Bounce = rateLimit("Bounce rate", d.Health.BounceRate, d, "bounce", p.Adapter)
		h.Complaint = rateLimit(complaintLabel, d.Health.ComplaintRate, d, "complaint", p.Adapter)
		switch {
		case p.Adapter == "ses":
			v := "Bounce and complaint metrics only; no VDM"
			if d.Health.VDM {
				v = "Virtual Deliverability Manager on"
			}
			h.Note = &DeliverabilityProviderNote{Label: "Reputation dashboard", Value: v}
		case p.Adapter == "postmark":
			kind := "Transactional"
			if strings.Contains(strings.ToLower(d.Config.MessageStream), "broadcast") {
				kind = "Broadcast"
			}
			v := kind
			if s := st.primaryStreams(p); len(s) == 1 {
				v += " · " + streamName(s[0]) + " mail only"
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
func rateLimit(label string, value *float64, d providerData, key, adapter string) DeliverabilityRateLimit {
	out := DeliverabilityRateLimit{Label: label, Value: value, Tone: "ok"}
	l, ok := d.Limits[key]
	if !ok {
		return out
	}
	first := l.Limit
	switch {
	case l.Review != nil && l.Pause != nil:
		out.Thresholds = fmt.Sprintf("review %s, pause %s", pctText(*l.Review), pctText(*l.Pause))
		first = l.Review
	case l.Limit != nil && strings.Contains(l.Label, "guard"):
		out.Thresholds = l.Label + " " + pctText(*l.Limit)
	case l.Limit != nil && selfHosted(adapter):
		out.Thresholds = "guard " + pctText(*l.Limit)
	case l.Limit != nil:
		out.Thresholds = "limit " + pctText(*l.Limit)
	}
	if first != nil && *first > 0 && value != nil {
		out.Usage = round(*value / *first * 100, 1)
		switch {
		case out.Usage >= 100:
			out.Tone = "bad"
		case out.Usage >= 50:
			out.Tone = "warn"
		}
	}
	return out
}
