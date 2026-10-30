package deliverability

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
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

// statusLabel is the badge word of a provider status, in a reader's language.
func statusLabel(l i18n.Lang, status string) string {
	return text(l, "status", i18n.Args{"status": status, "raw": status})
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
		pool.AccountStatus = accountWord(st.l, p)
		if caps := provstate.Caps(p); strings.HasPrefix(caps.DedicatedIPs, "optional, ") {
			pool.DedicatedIPOption = strings.TrimPrefix(caps.DedicatedIPs, "optional, ")
			// The adapter's own words in the reader's language, when the snapshot kept them.
			if !caps.DedicatedIPsText.IsZero() && st.l.Locale != "" && st.l.Locale != i18n.English {
				pool.DedicatedIPOption = caps.In(st.l).DedicatedIPs
			}
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
// approval state Postmark reports ("Approved"), else the status label. The
// standing reads in the reader's language when the provider kept its
// message; a status_detail (older connections) stays as it was written.
func accountWord(l i18n.Lang, p store.Provider) string {
	if p.Adapter == "postmark" && p.Status == provstate.Healthy {
		h := provstate.HealthOf(p)
		standing := providers.TextIn(h.Account, h.AccountText, l)
		if standing == "" {
			standing = p.StatusDetail
		}
		if w, _, _ := strings.Cut(standing, " · "); w != "" {
			return w
		}
	}
	return statusLabel(l, p.Status)
}

// providerHealth is account health per sending provider.
func (st *state) providerHealth() []DeliverabilityProviderHealth {
	out := []DeliverabilityProviderHealth{}
	for _, p := range st.providers {
		u := st.usage[p.ID]
		limits := provstate.LimitsOf(p)
		l := st.l
		h := DeliverabilityProviderHealth{ProviderID: p.ID.String(), Name: p.Name, Adapter: p.Adapter, Status: p.Status, Tone: statusTone(p.Status)}
		if selfHosted(p.Adapter) {
			h.Name = text(l, "name.selfHosted", i18n.Args{"name": p.Name})
		}
		h.Badge = accountWord(l, p)
		if p.Adapter == "ses" && p.Status == provstate.Healthy {
			// SES reports enforcement: "Production · no enforcement".
			ph := provstate.HealthOf(p)
			if _, enf, ok := strings.Cut(providers.TextIn(ph.Account, ph.AccountText, l), " · "); ok && enf != "" {
				h.Badge += " · " + enf
			}
		}
		complaintLabel := "rate.complaint"
		if p.Adapter == "postmark" {
			complaintLabel = "rate.spamComplaint"
		}
		h.Bounce = rateLimit(l, text(l, "rate.bounce", nil), u.Bounce7d, limits.Bounce, p.Adapter)
		h.Complaint = rateLimit(l, text(l, complaintLabel, nil), u.Complaint7d, limits.Complaint, p.Adapter)
		if u.TooFewToJudge() {
			// Too little mail to judge the rates (the rule Send Readiness
			// uses): they show without a tone, and the card says why.
			h.Bounce.Tone, h.Bounce.Usage = "ok", 0
			h.Complaint.Tone, h.Complaint.Usage = "ok", 0
			h.NotJudged = text(l, "rate.tooFew", i18n.Args{"count": u.Sent7d, "min": manage.MinRateVolume})
		}
		switch {
		case p.Adapter == "ses":
			v := text(l, "note.vdmOff", nil)
			if vdmOn(p) {
				v = text(l, "note.vdmOn", nil)
			}
			h.Note = &DeliverabilityProviderNote{Label: text(l, "note.reputationDashboard", nil), Value: v}
		case p.Adapter == "postmark":
			kind := "Transactional"
			if strings.Contains(strings.ToLower(configOf(p).MessageStream), "broadcast") || provstate.Caps(p).BroadcastStream {
				kind = "Broadcast"
			}
			v := text(l, "note.streamNone", i18n.Args{"kind": kind})
			switch s := st.primaryStreams(p); len(s) {
			case 1:
				v = text(l, "note.streamOnly", i18n.Args{"kind": kind, "stream": streamName(l, s[0])})
			case 2:
				// Postmark often carries System and Transactional mail.
				v = text(l, "note.streamTwo", i18n.Args{"kind": kind, "first": streamName(l, s[0]), "second": streamName(l, s[1])})
			}
			h.Note = &DeliverabilityProviderNote{Label: text(l, "note.stream", nil), Value: v}
		case selfHosted(p.Adapter):
			h.Note = &DeliverabilityProviderNote{Label: text(l, "note.thresholds", nil), Value: text(l, "note.thresholdsValue", i18n.Args{"name": p.Name})}
		}
		out = append(out, h)
	}
	return out
}

// streamName is one of ReputeCast's streams in a provider's note.
func streamName(l i18n.Lang, s string) string {
	return text(l, "stream.name", i18n.Args{"stream": s, "raw": s})
}

// rateLimit describes a rate against the provider's thresholds: SES has a
// review and a pause level, Postmark a limit, self-hosted servers the guard
// ReputeCast sets (the warm-up guard while warming up).
func rateLimit(lang i18n.Lang, label string, value *float64, l provstate.Limit, adapter string) DeliverabilityRateLimit {
	out := DeliverabilityRateLimit{Label: label, Value: value, Tone: "ok"}
	first := l.Limit
	switch {
	case l.Review > 0 && l.Pause > 0:
		out.Thresholds = text(lang, "threshold.reviewPause", i18n.Args{"review": pct(lang, l.Review), "pause": pct(lang, l.Pause)})
		first = l.Review
	case l.Label == "warm-up guard":
		out.Thresholds = text(lang, "threshold.warmupGuard", i18n.Args{"limit": pct(lang, l.Limit)})
	case strings.Contains(l.Label, "guard"):
		out.Thresholds = l.Label + " " + pct(lang, l.Limit)
	case selfHosted(adapter):
		out.Thresholds = text(lang, "threshold.guard", i18n.Args{"limit": pct(lang, l.Limit)})
	default:
		out.Thresholds = text(lang, "threshold.limit", i18n.Args{"limit": pct(lang, l.Limit)})
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
