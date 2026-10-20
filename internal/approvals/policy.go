// Package approvals implements send approvals (add-on 3.8, 5.12) and the
// Review & send flow around them (spec 7.5): the workspace policy and its
// rules, approval requests with a snapshot of what they cover (content and
// subject, audience definition, route, schedule), invalidation with the exact
// diff when any of it changes or the audience moves more than 10%,
// comments, decisions, the send-time recheck, the Owners' emergency bypass,
// and scheduling or starting the send through the sending engine.
package approvals

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/readiness"
)

// Rule keys, in the order of the policy screen.
const (
	RuleAudienceMin       = "audience_min"
	RuleWarmingProvider   = "warming_provider"
	RuleNewProvider       = "new_provider"
	RuleReadyWithWarnings = "ready_with_warnings"
	RuleAuthorLacksSend   = "author_lacks_send"
	RuleApprovalLists     = "approval_lists"
	RuleAutomationPublish = "automation_publish"
	// RuleTransactionalPublish: publishing a transactional template needs
	// approval (add-on transactional §3.6; off by default).
	RuleTransactionalPublish = "transactional_publish"
	// RulePostPublish: publishing a post of the publication website (or a
	// page) needs approval (design expansion phase 7; off by default).
	RulePostPublish = "post_publish"
)

// InvalidationShare is how far the audience may move (in percent) before an
// approval no longer covers it.
const InvalidationShare = 10.0

// LargeSend is the audience size from which sending asks to type the count.
const LargeSend = 1000

// policyDoc is workspaces.approval_policy (the shape the workspace settings
// summary reads too).
type policyDoc struct {
	Rules struct {
		AudienceMin struct {
			Enabled bool  `json:"enabled"`
			N       int64 `json:"n"`
		} `json:"audience_min"`
		Warming struct {
			Enabled bool `json:"enabled"`
		} `json:"warming_provider"`
		NewProvider struct {
			Enabled bool `json:"enabled"`
			Days    int  `json:"days"`
		} `json:"new_provider"`
		Warnings struct {
			Enabled bool `json:"enabled"`
		} `json:"ready_with_warnings"`
		LacksSend struct {
			Enabled bool `json:"enabled"`
		} `json:"author_lacks_send"`
		Lists struct {
			Enabled bool `json:"enabled"`
		} `json:"approval_lists"`
	} `json:"rules"`
	ApprovalsRequired    int      `json:"approvals_required"`
	ApproverRoles        []string `json:"approver_roles,omitempty"`
	AutomationPublish    bool     `json:"automation_publish"`
	TransactionalPublish bool     `json:"transactional_publish"`
	PostPublish          bool     `json:"post_publish"`
	EmergencyBypass      *bool    `json:"emergency_bypass"`
	// ClientSignoff is client workspaces' rule (agencies, clientsignoff.go).
	ClientSignoff *clientSignoffDoc `json:"client_signoff,omitempty"`
}

// ApprovalAudienceRule is the audience-size rule.
type ApprovalAudienceRule struct {
	Enabled bool  `json:"enabled"`
	N       int64 `json:"n" minimum:"1" maximum:"100000000"`
}

// ApprovalPolicy is the workspace's approval policy.
type ApprovalPolicy struct {
	AudienceMin ApprovalAudienceRule `json:"audienceMin"`
	// WarmingProvider: the route includes a provider that is warming up or
	// was connected in the last NewProviderDays days.
	WarmingProvider   bool `json:"warmingProvider"`
	NewProviderDays   int  `json:"newProviderDays" minimum:"1" maximum:"90"`
	ReadyWithWarnings bool `json:"readyWithWarnings"`
	AuthorLacksSend   bool `json:"authorLacksSend"`
	ApprovalLists     bool `json:"approvalLists"`
	ApprovalsRequired int  `json:"approvalsRequired" minimum:"1" maximum:"2"`
	AutomationPublish bool `json:"automationPublish"`
	// TransactionalPublish: publishing a transactional template needs an
	// approver other than the publisher (off by default).
	TransactionalPublish bool `json:"transactionalPublish"`
	// PostPublish: publishing a post or page of the website needs an
	// approver other than the publisher (off by default). "The author
	// lacks the send permission" covers website:publish too.
	PostPublish     bool `json:"postPublish"`
	EmergencyBypass bool `json:"emergencyBypass"`
	// ClientSignoff: in a client workspace, every send needs one approval
	// from a Client approver; approvals from agency staff don't count.
	// Leaving it out of a save turns it off.
	ClientSignoff ApprovalClientSignoff `json:"clientSignoff,omitempty"`
}

// ParsePolicy reads the stored policy. An empty policy is the default of a
// new workspace (A5 §3.8).
func ParsePolicy(raw json.RawMessage) ApprovalPolicy {
	var d policyDoc
	if len(raw) == 0 || string(raw) == "{}" || json.Unmarshal(raw, &d) != nil {
		return ApprovalPolicy{AudienceMin: ApprovalAudienceRule{Enabled: true, N: 10000}, WarmingProvider: true, NewProviderDays: 14,
			AuthorLacksSend: true, ApprovalsRequired: 1, EmergencyBypass: true}
	}
	p := ApprovalPolicy{
		AudienceMin:          ApprovalAudienceRule{Enabled: d.Rules.AudienceMin.Enabled, N: d.Rules.AudienceMin.N},
		WarmingProvider:      d.Rules.Warming.Enabled || d.Rules.NewProvider.Enabled,
		NewProviderDays:      d.Rules.NewProvider.Days,
		ReadyWithWarnings:    d.Rules.Warnings.Enabled,
		AuthorLacksSend:      d.Rules.LacksSend.Enabled,
		ApprovalLists:        d.Rules.Lists.Enabled,
		ApprovalsRequired:    d.ApprovalsRequired,
		AutomationPublish:    d.AutomationPublish,
		TransactionalPublish: d.TransactionalPublish,
		PostPublish:          d.PostPublish,
		EmergencyBypass:      d.EmergencyBypass == nil || *d.EmergencyBypass,
		ClientSignoff:        d.ClientSignoff.policy(),
	}
	if p.AudienceMin.N <= 0 {
		p.AudienceMin.N = 10000
	}
	if p.NewProviderDays <= 0 {
		p.NewProviderDays = 14
	}
	if p.ApprovalsRequired < 1 {
		p.ApprovalsRequired = 1
	}
	if p.ApprovalsRequired > 2 {
		p.ApprovalsRequired = 2
	}
	return p
}

// toDB writes the policy in the stored shape, keeping other keys of prev.
// The warming rule covers new providers too (one rule in A5 §3.8), so the
// separate new_provider switch is folded into it.
func (p ApprovalPolicy) toDB(prev json.RawMessage) json.RawMessage {
	var d policyDoc
	_ = json.Unmarshal(prev, &d)
	d.Rules.AudienceMin.Enabled, d.Rules.AudienceMin.N = p.AudienceMin.Enabled, p.AudienceMin.N
	d.Rules.Warming.Enabled = p.WarmingProvider
	d.Rules.NewProvider.Enabled, d.Rules.NewProvider.Days = false, p.NewProviderDays
	d.Rules.Warnings.Enabled = p.ReadyWithWarnings
	d.Rules.LacksSend.Enabled = p.AuthorLacksSend
	d.Rules.Lists.Enabled = p.ApprovalLists
	d.ApprovalsRequired = p.ApprovalsRequired
	d.AutomationPublish = p.AutomationPublish
	d.TransactionalPublish = p.TransactionalPublish
	d.PostPublish = p.PostPublish
	b := p.EmergencyBypass
	d.EmergencyBypass = &b
	d.ClientSignoff = p.ClientSignoff.doc()
	if len(d.ApproverRoles) == 0 {
		d.ApproverRoles = []string{"owner", "admin"}
	}
	out, _ := json.Marshal(d)
	return out
}

// RuleLabels are the short phrases of the enabled rules ("audience ≥ 10,000").
func (p ApprovalPolicy) RuleLabels() []string {
	out := []string{}
	if p.AudienceMin.Enabled {
		out = append(out, "audience ≥ "+readiness.Count(p.AudienceMin.N))
	}
	if p.WarmingProvider {
		out = append(out, "warming-up provider")
	}
	if p.ReadyWithWarnings {
		out = append(out, "Ready with warnings")
	}
	if p.AuthorLacksSend {
		out = append(out, "author lacks send")
	}
	if p.ApprovalLists {
		out = append(out, "list requires approval")
	}
	if p.PostPublish {
		out = append(out, "post or page publish")
	}
	if p.ClientSignoff.Enabled {
		out = append(out, "client sign-off")
	}
	return out
}

// ApprovalRuleMatch is a rule that matched a send.
type ApprovalRuleMatch struct {
	Key string `json:"key" enum:"audience_min,warming_provider,new_provider,ready_with_warnings,author_lacks_send,approval_lists,automation_publish,transactional_publish,post_publish,client_signoff"`
	// Label is the sentence ("Audience of 47,912 is at least 10,000"),
	// Short the inbox phrase ("audience ≥ 10,000") and Badge the chip.
	Label string `json:"label"`
	Short string `json:"short"`
	Badge string `json:"badge"`
	// Author is the author's name for "author lacks send" ("You don't hold…"
	// is written by the screen when the author reads it).
	Author string `json:"author,omitempty"`
	Role   string `json:"role,omitempty"`
}

// RuleFacts are what the rules look at.
type RuleFacts struct {
	Recipients int64
	Shares     []RuleShare
	Verdict    string
	Warnings   int
	// AuthorCanSend is whether the person sending holds campaigns:send.
	AuthorCanSend bool
	AuthorName    string
	AuthorRole    string
	// ApprovalLists are the audience's lists marked "Requires approval".
	ApprovalLists []string
	Now           time.Time
}

// RuleShare is a provider of the route.
type RuleShare struct {
	Short       string
	Recipients  int64
	WarmingUp   bool
	WarmupDay   int
	WarmupDays  int
	ConnectedAt time.Time
}

// Evaluate returns the rules that match a send. No match: no approval needed.
func (p ApprovalPolicy) Evaluate(f RuleFacts) []ApprovalRuleMatch {
	out := []ApprovalRuleMatch{}
	if p.AudienceMin.Enabled && f.Recipients >= p.AudienceMin.N {
		short := "audience ≥ " + readiness.Count(p.AudienceMin.N)
		out = append(out, ApprovalRuleMatch{Key: RuleAudienceMin, Short: short, Badge: short,
			Label: fmt.Sprintf("Audience of %s is at least %s", readiness.Count(f.Recipients), readiness.Count(p.AudienceMin.N))})
	}
	if p.WarmingProvider {
		for _, s := range f.Shares {
			if s.Recipients == 0 && len(f.Shares) > 1 {
				continue
			}
			days := p.NewProviderDays
			if days <= 0 {
				days = 14
			}
			switch {
			case s.WarmingUp:
				label := fmt.Sprintf("Route includes %s, which is warming up", s.Short)
				if s.WarmupDays > 0 {
					label += fmt.Sprintf(" (day %d of %d)", s.WarmupDay, s.WarmupDays)
				}
				out = append(out, ApprovalRuleMatch{Key: RuleWarmingProvider, Label: label, Short: "warming-up provider", Badge: "Warming-up provider"})
			case !s.ConnectedAt.IsZero() && f.Now.Sub(s.ConnectedAt) < time.Duration(days)*24*time.Hour:
				ago := int(f.Now.Sub(s.ConnectedAt).Hours() / 24)
				when := fmt.Sprintf("%d days ago", ago)
				if ago < 1 {
					when = "today"
				} else if ago == 1 {
					when = "yesterday"
				}
				out = append(out, ApprovalRuleMatch{Key: RuleNewProvider, Label: fmt.Sprintf("Route includes %s, connected %s", s.Short, when), Short: "new provider", Badge: "New provider"})
			default:
				continue
			}
			break
		}
	}
	if p.ReadyWithWarnings && f.Verdict == readiness.VerdictWarnings {
		out = append(out, ApprovalRuleMatch{Key: RuleReadyWithWarnings, Short: "Ready with warnings", Badge: "Ready with warnings",
			Label: fmt.Sprintf("Send Readiness is Ready with warnings (%d %s)", f.Warnings, readiness.Plural(int64(f.Warnings), "warning", "warnings"))})
	}
	if p.AuthorLacksSend && !f.AuthorCanSend {
		label := f.AuthorName + " doesn't hold campaigns:send"
		if f.AuthorRole != "" {
			label += " · " + f.AuthorRole
		}
		out = append(out, ApprovalRuleMatch{Key: RuleAuthorLacksSend, Label: label, Short: "author lacks send", Badge: "author lacks send", Author: f.AuthorName, Role: f.AuthorRole})
	}
	if p.ApprovalLists && len(f.ApprovalLists) > 0 {
		out = append(out, ApprovalRuleMatch{Key: RuleApprovalLists, Short: "list requires approval", Badge: "List requires approval",
			Label: fmt.Sprintf("The audience includes %s, which %s approval", readiness.JoinAnd(f.ApprovalLists), readiness.Plural(int64(len(f.ApprovalLists)), "requires", "require"))})
	}
	if p.ClientSignoff.Enabled {
		out = append(out, clientSignoffMatch())
	}
	return out
}

// rulesLine is "rules: audience ≥ 10,000 · warming-up provider".
func rulesLine(rules []ApprovalRuleMatch) string {
	if len(rules) == 0 {
		return ""
	}
	parts := make([]string, len(rules))
	for i, r := range rules {
		parts[i] = r.Short
	}
	word := "rules"
	if len(rules) == 1 {
		word = "rule"
	}
	return word + ": " + strings.Join(parts, " · ")
}

func ruleKeys(rules []ApprovalRuleMatch) []string {
	out := make([]string, len(rules))
	for i, r := range rules {
		out[i] = r.Key
	}
	return out
}
