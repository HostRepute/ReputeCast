// Package routing decides which sending provider delivers each email: rules
// per stream (Broadcasts, Automations, System) with conditions on the list,
// the sender domain and the recipient's mailbox provider, a deterministic
// weighted split for migrations and warm-up, a fallback per stream and the
// failover policy.
//
// The decision functions here are pure; service.go stores and validates
// rules and policies and serves the routing screen and the route tester.
package routing

import (
	"crypto/sha256"
	"encoding/binary"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
)

// Policy is a stream's failover policy.
type Policy string

// Failover policies.
const (
	// Automatic switches to the fallback when the primary fails.
	Automatic Policy = "automatic"
	// PauseAndAsk holds the remaining recipients and asks a person.
	PauseAndAsk Policy = "pause_and_ask"
	// WarmOnly fails over automatically, but only to a provider that is not
	// warming up; otherwise it behaves like PauseAndAsk.
	WarmOnly Policy = "warm_only"
)

// Valid reports whether p is a known policy.
func (p Policy) Valid() bool { return p == Automatic || p == PauseAndAsk || p == WarmOnly }

// DefaultPolicy is the failover policy of a stream nobody configured:
// pause-and-ask for Broadcasts (moving a bulk send to a cold provider
// mid-send can hurt reputation), automatic for System and Automations.
func DefaultPolicy(s providers.Stream) Policy {
	if s == providers.StreamBroadcast {
		return PauseAndAsk
	}
	return Automatic
}

// StreamTransactional is the fourth stream (design expansion phase 1):
// mail from the workspace's own product, sent through the transactional
// API and the SMTP relay. It fails over automatically, bulk-marketing terms
// don't apply to it, and on a provider it shares with Broadcasts it goes
// first within the provider's rate.
const StreamTransactional providers.Stream = "transactional"

// Streams in display order.
var Streams = []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem, StreamTransactional}

// Conditions restrict a rule; an empty set is the stream's default rule.
// Values within a field are OR-ed, fields are AND-ed. Categories (ids of
// transactional categories) only apply to the Transactional stream.
type Conditions struct {
	Lists            []string `json:"lists,omitempty"`
	SenderDomains    []string `json:"sender_domains,omitempty"`
	MailboxProviders []string `json:"mailbox_providers,omitempty"`
	Categories       []string `json:"categories,omitempty"`
}

// Empty reports whether the conditions match everything.
func (c Conditions) Empty() bool {
	return len(c.Lists) == 0 && len(c.SenderDomains) == 0 && len(c.MailboxProviders) == 0 && len(c.Categories) == 0
}

// Split is one provider's share of a rule, in percent.
type Split struct {
	ProviderID uuid.UUID `json:"provider_id"`
	Weight     int       `json:"weight"`
}

// Rule is one routing rule of a stream.
type Rule struct {
	ID         uuid.UUID
	Stream     providers.Stream
	Priority   int
	Name       string
	Conditions Conditions
	Splits     []Split
	Fallback   *uuid.UUID
	Enabled    bool
}

// Set is a workspace's routing: rules by priority and a policy per stream.
type Set struct {
	Rules    []Rule
	Policies map[providers.Stream]Policy
}

// Policy returns the failover policy of a stream.
func (s Set) Policy(stream providers.Stream) Policy {
	if p, ok := s.Policies[stream]; ok && p.Valid() {
		return p
	}
	return DefaultPolicy(stream)
}

// StreamRules returns a stream's enabled rules in evaluation order:
// conditional rules by priority, then the default rule.
func (s Set) StreamRules(stream providers.Stream) []Rule {
	var out []Rule
	for _, r := range s.Rules {
		if r.Stream == stream && r.Enabled {
			out = append(out, r)
		}
	}
	slices.SortStableFunc(out, func(a, b Rule) int {
		ae, be := a.Conditions.Empty(), b.Conditions.Empty()
		if ae != be {
			if ae {
				return 1
			}
			return -1
		}
		return a.Priority - b.Priority
	})
	return out
}

// Default returns the stream's default rule (no conditions), if any.
func (s Set) Default(stream providers.Stream) (Rule, bool) {
	for _, r := range s.StreamRules(stream) {
		if r.Conditions.Empty() {
			return r, true
		}
	}
	return Rule{}, false
}

// Recipient describes the email being routed.
type Recipient struct {
	Stream          providers.Stream
	Email           string
	Lists           []string
	SenderDomain    string
	MailboxProvider string // gmail, outlook, yahoo, icloud, other; derived from Email when empty
	// Category is the transactional category of the message (its id).
	Category string
}

// Decision is where a recipient goes and why.
type Decision struct {
	ProviderID uuid.UUID
	Rule       Rule
	// Matched is the condition field that decided ("mailbox_provider",
	// "list", "sender_domain"), empty for the default rule.
	Matched      string
	MatchedValue string
	// Bucket is the recipient's hash bucket (0–99) when a weighted split decided.
	Bucket   int
	Split    bool
	Fallback *uuid.UUID
	Policy   Policy
}

// Resolve applies the stream's rules to a recipient. ok is false when the
// stream has no rule (no provider is set up for it).
func (s Set) Resolve(r Recipient) (Decision, bool) {
	mbp := r.MailboxProvider
	if mbp == "" {
		mbp = MailboxProvider(r.Email)
	}
	for _, rule := range s.StreamRules(r.Stream) {
		field, value, ok := match(rule.Conditions, r, mbp)
		if !ok || len(rule.Splits) == 0 {
			continue
		}
		d := Decision{Rule: rule, Matched: field, MatchedValue: value, Fallback: rule.Fallback, Policy: s.Policy(r.Stream)}
		if d.Fallback == nil && !rule.Conditions.Empty() {
			if def, ok := s.Default(r.Stream); ok {
				d.Fallback = def.Fallback
			}
		}
		d.ProviderID, d.Bucket, d.Split = pick(rule.Splits, r.Email)
		return d, true
	}
	return Decision{}, false
}

func match(c Conditions, r Recipient, mbp string) (field, value string, ok bool) {
	if c.Empty() {
		return "", "", true
	}
	if len(c.MailboxProviders) > 0 {
		if !slices.Contains(c.MailboxProviders, mbp) {
			return "", "", false
		}
		field, value = "mailbox_provider", mbp
	}
	if len(c.Lists) > 0 {
		hit := ""
		for _, l := range r.Lists {
			if slices.Contains(c.Lists, l) {
				hit = l
				break
			}
		}
		if hit == "" {
			return "", "", false
		}
		if field == "" {
			field, value = "list", hit
		}
	}
	if len(c.Categories) > 0 {
		if r.Category == "" || !slices.Contains(c.Categories, r.Category) {
			return "", "", false
		}
		if field == "" {
			field, value = "category", r.Category
		}
	}
	if len(c.SenderDomains) > 0 {
		d := strings.ToLower(r.SenderDomain)
		if !slices.ContainsFunc(c.SenderDomains, func(x string) bool { return strings.EqualFold(x, d) }) {
			return "", "", false
		}
		if field == "" {
			field, value = "sender_domain", d
		}
	}
	return field, value, true
}

// pick chooses a provider from a weighted split. The bucket comes from a
// hash of the address, so a recipient always lands on the same provider
// (important while a provider warms up) and the split is exact over many
// recipients.
func pick(splits []Split, email string) (uuid.UUID, int, bool) {
	if len(splits) == 1 {
		return splits[0].ProviderID, 0, false
	}
	total := 0
	for _, s := range splits {
		total += max(s.Weight, 0)
	}
	b := Bucket(email)
	if total <= 0 {
		return splits[0].ProviderID, b, true
	}
	// Scale the bucket to the total weight (weights normally sum to 100).
	at := b * total / 100
	acc := 0
	for _, s := range splits {
		acc += max(s.Weight, 0)
		if at < acc {
			return s.ProviderID, b, true
		}
	}
	return splits[len(splits)-1].ProviderID, b, true
}

// Bucket maps an address to 0–99, stable across runs and machines.
func Bucket(email string) int {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return int(binary.BigEndian.Uint64(sum[:8]) % 100)
}

// Mailbox providers.
const (
	Gmail   = "gmail"
	Outlook = "outlook"
	Yahoo   = "yahoo"
	ICloud  = "icloud"
	Other   = "other"
)

// MailboxProviders in display order.
var MailboxProviders = []string{Gmail, Outlook, Yahoo, ICloud, Other}

// MailboxProvider guesses the mailbox provider from an address's domain.
// Contacts carry a better answer (MX-based) when they exist.
func MailboxProvider(email string) string {
	d := strings.ToLower(email)
	if i := strings.LastIndexByte(d, '@'); i >= 0 {
		d = d[i+1:]
	}
	switch {
	case d == "gmail.com" || d == "googlemail.com":
		return Gmail
	case d == "outlook.com" || d == "hotmail.com" || d == "live.com" || d == "msn.com" ||
		strings.HasPrefix(d, "outlook.") || strings.HasPrefix(d, "hotmail.") || strings.HasPrefix(d, "live."):
		return Outlook
	case d == "yahoo.com" || d == "ymail.com" || d == "aol.com" || d == "rocketmail.com" ||
		strings.HasPrefix(d, "yahoo.") || strings.HasPrefix(d, "aol."):
		return Yahoo
	case d == "icloud.com" || d == "me.com" || d == "mac.com":
		return ICloud
	}
	return Other
}

// MailboxName is the display name of a mailbox provider.
func MailboxName(k string) string {
	switch k {
	case Gmail:
		return "Gmail"
	case Outlook:
		return "Outlook"
	case Yahoo:
		return "Yahoo"
	case ICloud:
		return "iCloud"
	}
	return "Other"
}

// ProviderState is what failover needs to know about a provider.
type ProviderState struct {
	Available bool // healthy enough to send (not down, webhooks not silent)
	WarmingUp bool
}

// Failover returns where a stream's mail goes when its provider is not
// available: the fallback (moved) or nowhere (held for a decision).
func Failover(policy Policy, fallback *uuid.UUID, state func(uuid.UUID) ProviderState) (target *uuid.UUID, hold bool) {
	if fallback == nil {
		return nil, true
	}
	st := state(*fallback)
	if !st.Available {
		return nil, true
	}
	switch policy {
	case Automatic:
		return fallback, false
	case WarmOnly:
		if st.WarmingUp {
			return nil, true
		}
		return fallback, false
	}
	return nil, true
}
