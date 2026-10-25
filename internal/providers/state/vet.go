package state

import (
	"encoding/json"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Target vetting (review GS-5, PR-7): mail only moves to a provider that
// can take it. A person's move, an automatic failover, a reroute while an
// audience is built and the alternatives an outage card offers all ask Vet,
// so they agree on what "can take it" means.

// Vetting checks, in the order screens list them.
const (
	VetAvailable = "available"  // connected, not down or paused, events arriving
	VetBulkTerms = "bulk_terms" // its terms allow Broadcasts
	VetSandbox   = "sandbox"    // the account may send to anyone
	VetDomain    = "domain"     // the sender is verified there
	VetQuota     = "quota"      // what is left of its quota covers the mail
	VetWarm      = "warm"       // not warming up, or today's cap covers the mail
)

// Check results.
const (
	VetPass = "pass"
	// VetWarn is possible at a cost a person chooses: a warming IP takes
	// mail its plan didn't count on.
	VetWarn = "warn"
	VetFail = "fail"
	// VetUnknown is what ReputeCast can't tell: a sender on a provider
	// without a domain API, a quota not read yet.
	VetUnknown = "unknown"
)

// Load is what would move to a target.
type Load struct {
	// Broadcast is true for Broadcasts, to which bulk terms apply.
	Broadcast bool
	// Messages is how many messages would move.
	Messages int64
	// Sender is the sender's verification per provider ID
	// (senders.verification); nil when the mail has no stored sender.
	Sender map[string]string
	// Warmup is the target's warm-up today, nil when no plan is running.
	Warmup *WarmupToday
}

// WarmupToday is a warming provider's allowance today.
type WarmupToday struct {
	Cap    int64
	Sent   int64
	Paused bool
}

// VetCheck is one finding about a target.
type VetCheck struct {
	Key    string `json:"key"`
	Result string `json:"result"`
	// Left and Need are what a quota or warm-up check compares: what the
	// provider can still take and what would move.
	Left *int64 `json:"left,omitempty"`
	Need *int64 `json:"need,omitempty"`
	// Detail qualifies the result: the provider's status, the sender's
	// verification there ("pending", "not_set_up"), the quota's kind.
	Detail string `json:"detail,omitempty"`
	// Text is the finding in the reader's language (Vetting.In), empty
	// for a passed check.
	Text string `json:"text,omitempty"`
}

// Vetting is what Vet found about one target.
type Vetting struct {
	Checks []VetCheck `json:"checks"`
}

// Failed is the first failed check.
func (v Vetting) Failed() (VetCheck, bool) {
	for _, c := range v.Checks {
		if c.Result == VetFail {
			return c, true
		}
	}
	return VetCheck{}, false
}

// Allowed reports whether a person may move mail there: no check failed.
func (v Vetting) Allowed() bool {
	_, failed := v.Failed()
	return !failed
}

// Automatic reports whether mail may move there without asking anyone:
// every check passed, or only the warm-up warned and the failover policy
// accepts warming providers.
func (v Vetting) Automatic(acceptWarming bool) bool {
	for _, c := range v.Checks {
		if c.Result != VetPass && (!acceptWarming || c.Key != VetWarm || c.Result != VetWarn) {
			return false
		}
	}
	return true
}

// Warming reports whether the target is warming up.
func (v Vetting) Warming() bool {
	for _, c := range v.Checks {
		if c.Key == VetWarm {
			return c.Result != VetPass
		}
	}
	return false
}

// In returns the checks with their sentences about the provider, in a
// reader's language.
func (v Vetting) In(l i18n.Lang, provider string) Vetting {
	out := Vetting{Checks: make([]VetCheck, len(v.Checks))}
	for i, c := range v.Checks {
		c.Text = c.In(l, provider)
		out.Checks[i] = c
	}
	return out
}

// SenderVerification decodes a sender's verification per provider ID
// (senders.verification).
func SenderVerification(raw json.RawMessage) map[string]string {
	v := map[string]string{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &v)
	}
	return v
}

// Vet checks whether a provider can take a load.
func Vet(p store.Provider, l Load) Vetting {
	caps, h := Caps(p), HealthOf(p)
	need := l.Messages
	out := Vetting{Checks: make([]VetCheck, 0, 6)}
	add := func(key, result, detail string) *VetCheck {
		out.Checks = append(out.Checks, VetCheck{Key: key, Result: result, Detail: detail})
		return &out.Checks[len(out.Checks)-1]
	}

	switch {
	case p.DisconnectedAt != nil:
		add(VetAvailable, VetFail, "disconnected")
	case !Available(p):
		add(VetAvailable, VetFail, p.Status)
	default:
		add(VetAvailable, VetPass, "")
	}
	if l.Broadcast {
		if caps.BulkAllowed() {
			add(VetBulkTerms, VetPass, "")
		} else {
			add(VetBulkTerms, VetFail, string(caps.Bulk))
		}
	}
	if p.Status == Sandbox || h.Sandbox {
		add(VetSandbox, VetFail, "")
	} else {
		add(VetSandbox, VetPass, "")
	}
	switch v := l.Sender[p.ID.String()]; {
	case l.Sender == nil:
		add(VetDomain, VetUnknown, "no_sender")
	case v == "verified":
		add(VetDomain, VetPass, "")
	case (v == "" || v == "not_set_up") && caps.DomainAPI == "":
		// Without a domain API nobody asked the provider to sign for the
		// domain, and nothing tells whether it does.
		add(VetDomain, VetUnknown, "not_set_up")
	case v == "":
		add(VetDomain, VetFail, "not_set_up")
	default:
		add(VetDomain, VetFail, v)
	}
	switch q := h.Quota; {
	case q != nil && q.Max > 0 && q.Kind != "warmup_daily":
		left := max(q.Max-q.Used, 0)
		c := add(VetQuota, VetPass, q.Kind)
		if left < need {
			c.Result = VetFail
		}
		c.Left, c.Need = &left, &need
	case q == nil && caps.QuotaDiscovery:
		add(VetQuota, VetUnknown, "not_read")
	default:
		// The provider sets no quota (self-hosted, plans without one).
		add(VetQuota, VetPass, "")
	}
	switch w := l.Warmup; {
	case w != nil:
		left := max(w.Cap-w.Sent, 0)
		c := add(VetWarm, VetWarn, "")
		switch {
		case w.Paused:
			c.Result, c.Detail = VetFail, "paused"
		case left < need:
			c.Result = VetFail
		}
		c.Left, c.Need = &left, &need
	case p.Status == WarmingUp:
		add(VetWarm, VetUnknown, "")
	default:
		add(VetWarm, VetPass, "")
	}
	return out
}

var vetTexts = map[string]i18n.Msg{
	VetAvailable: msgs.M("vet.available"),
	VetBulkTerms: msgs.M("vet.bulkTerms"),
	VetSandbox:   msgs.M("vet.sandbox"),
	VetDomain:    msgs.M("vet.domain"),
	VetQuota:     msgs.M("vet.quota"),
	VetWarm:      msgs.M("vet.warm"),
}

// Keep is the check's sentence about a provider, kept as a message so a
// stored decision reads in each reader's language. Passed checks have none.
func (c VetCheck) Keep(provider string) i18n.Text {
	m, ok := vetTexts[c.Key]
	if !ok || c.Result == VetPass {
		return i18n.Text{}
	}
	args := i18n.Args{"provider": provider, "result": c.Result, "detail": c.Detail}
	if c.Left != nil && c.Need != nil {
		args["left"], args["need"] = *c.Left, *c.Need
	}
	return m.Keep(args)
}

// In is the check's sentence about a provider for a reader ("" when it
// passed).
func (c VetCheck) In(l i18n.Lang, provider string) string {
	t := c.Keep(provider)
	if t.IsZero() {
		return ""
	}
	return t.In(l)
}
