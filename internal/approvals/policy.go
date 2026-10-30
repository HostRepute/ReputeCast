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

	"github.com/hostrepute/reputecast/internal/i18n"
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
	// RuleChannelCost: an SMS or WhatsApp campaign is estimated to cost at
	// least an amount (phase 8 brief §3.6; off by default).
	RuleChannelCost = "channel_cost_min"
)

// InvalidationShare is how far the audience may move (in percent) before an
// approval no longer covers it.
const InvalidationShare = 10.0

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
		ChannelCost struct {
			Enabled bool    `json:"enabled"`
			Amount  float64 `json:"amount"`
		} `json:"channel_cost_min"`
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
	// ChannelCost: an SMS or WhatsApp campaign is estimated to cost at
	// least Amount (off by default).
	ChannelCost ApprovalCostRule `json:"channelCost,omitempty"`
}

// ApprovalCostRule is the cost rule for SMS and WhatsApp campaigns.
type ApprovalCostRule struct {
	Enabled bool    `json:"enabled"`
	Amount  float64 `json:"amount" minimum:"0.01" maximum:"1000000"`
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
		ChannelCost:          ApprovalCostRule{Enabled: d.Rules.ChannelCost.Enabled, Amount: d.Rules.ChannelCost.Amount},
	}
	if p.AudienceMin.N <= 0 {
		p.AudienceMin.N = 10000
	}
	if p.ChannelCost.Amount <= 0 {
		p.ChannelCost.Amount = 50
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
	d.Rules.ChannelCost.Enabled, d.Rules.ChannelCost.Amount = p.ChannelCost.Enabled, p.ChannelCost.Amount
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
func (p ApprovalPolicy) RuleLabels() []string { return p.RuleLabelsIn(i18n.Default) }

// RuleLabelsIn are the short phrases of the enabled rules in a language.
func (p ApprovalPolicy) RuleLabelsIn(l i18n.Lang) []string {
	out := []string{}
	if p.AudienceMin.Enabled {
		out = append(out, render(l, mRuleAudienceShort.Keep(i18n.Args{"min": p.AudienceMin.N})))
	}
	if p.WarmingProvider {
		out = append(out, mRuleWarmingShort.Text(l, nil))
	}
	if p.ReadyWithWarnings {
		out = append(out, mRuleWarningsShort.Text(l, nil))
	}
	if p.AuthorLacksSend {
		out = append(out, mRuleLacksSendShort.Text(l, nil))
	}
	if p.ApprovalLists {
		out = append(out, mRuleListsShort.Text(l, nil))
	}
	if p.PostPublish {
		out = append(out, mRulePolicyPostPublish.Text(l, nil))
	}
	if p.ChannelCost.Enabled {
		out = append(out, render(l, mRuleCostPolicy.Keep(i18n.Args{"moneyMin": p.ChannelCost.Amount})))
	}
	if p.ClientSignoff.Enabled {
		out = append(out, mRuleSignoffShort.Text(l, nil))
	}
	return out
}

// costLabel writes an amount in dollars: "$50", "$19.54".
func costLabel(v float64) string {
	if v == float64(int64(v)) {
		return "$" + readiness.Count(int64(v))
	}
	return fmt.Sprintf("$%.2f", v)
}

// ApprovalRuleMatch is a rule that matched a send.
type ApprovalRuleMatch struct {
	Key string `json:"key" enum:"audience_min,warming_provider,new_provider,ready_with_warnings,author_lacks_send,approval_lists,automation_publish,transactional_publish,post_publish,channel_cost_min,client_signoff"`
	// Label is the sentence ("Audience of 47,912 is at least 10,000"),
	// Short the inbox phrase ("audience ≥ 10,000") and Badge the chip.
	Label string `json:"label"`
	Short string `json:"short"`
	Badge string `json:"badge"`
	// Author is the author's name for "author lacks send" ("You don't hold…"
	// is written by the screen when the author reads it).
	Author string `json:"author,omitempty"`
	Role   string `json:"role,omitempty"`

	// texts are the label, short phrase and badge as messages, for readers
	// in other languages; a request keeps them in its snapshot (RuleTexts).
	texts *ruleTexts
}

// ruleTexts are a matched rule's label, short phrase and badge as messages.
type ruleTexts struct {
	Label i18n.Text `json:"label,omitzero"`
	Short i18n.Text `json:"short,omitzero"`
	Badge i18n.Text `json:"badge,omitzero"`
}

// match builds a matched rule: its English, as stored and as API clients
// read it, and its messages.
func match(key string, label, short, badge i18n.Msg, args i18n.Args) ApprovalRuleMatch {
	t := &ruleTexts{Label: label.Keep(args), Short: short.Keep(args), Badge: badge.Keep(args)}
	return ApprovalRuleMatch{Key: key, Label: render(i18n.Default, t.Label), Short: render(i18n.Default, t.Short), Badge: render(i18n.Default, t.Badge), texts: t}
}

// localRules writes matched rules in a reader's language. Rules kept
// without their messages (requests made before they were kept) read as
// stored.
func localRules(l i18n.Lang, rules []ApprovalRuleMatch) []ApprovalRuleMatch {
	out := make([]ApprovalRuleMatch, len(rules))
	for i, r := range rules {
		if r.texts != nil && l.Locale != i18n.English {
			r.Label = kept(l, r.Label, r.texts.Label)
			r.Short = kept(l, r.Short, r.texts.Short)
			r.Badge = kept(l, r.Badge, r.texts.Badge)
		}
		out[i] = r
	}
	return out
}

// RuleFacts are what the rules look at.
type RuleFacts struct {
	Recipients int64
	Shares     []RuleShare
	Verdict    string
	Warnings   int
	// AuthorCanSend is whether the person sending holds campaigns:send (and
	// the channel's own send permission for SMS, push and WhatsApp).
	AuthorCanSend bool
	AuthorName    string
	AuthorRole    string
	// ApprovalLists are the audience's lists marked "Requires approval".
	ApprovalLists []string
	Now           time.Time
	// Channel is the campaign's channel ("" or "email" for email); Cost its
	// estimated cost; SendPermission the permission the author lacks
	// ("sms:send"), named by the "author lacks send" rule.
	Channel        string
	Cost           float64
	SendPermission string
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
		out = append(out, match(RuleAudienceMin, mRuleAudience, mRuleAudienceShort, mRuleAudienceShort, i18n.Args{"count": f.Recipients, "min": p.AudienceMin.N}))
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
				label, args := mRuleWarming, i18n.Args{"provider": s.Short}
				if s.WarmupDays > 0 {
					label, args["day"], args["days"] = mRuleWarmingDay, s.WarmupDay, s.WarmupDays
				}
				out = append(out, match(RuleWarmingProvider, label, mRuleWarmingShort, mRuleWarmingBadge, args))
			case !s.ConnectedAt.IsZero() && f.Now.Sub(s.ConnectedAt) < time.Duration(days)*24*time.Hour:
				ago := int(f.Now.Sub(s.ConnectedAt).Hours() / 24)
				out = append(out, match(RuleNewProvider, mRuleNewProvider, mRuleNewProviderShort, mRuleNewProviderBadge, i18n.Args{"provider": s.Short, "days": max(ago, 0)}))
			default:
				continue
			}
			break
		}
	}
	if p.ReadyWithWarnings && f.Verdict == readiness.VerdictWarnings {
		out = append(out, match(RuleReadyWithWarnings, mRuleWarnings, mRuleWarningsShort, mRuleWarningsShort, i18n.Args{"count": f.Warnings}))
	}
	if p.ChannelCost.Enabled && (f.Channel == "sms" || f.Channel == "whatsapp") && f.Cost >= p.ChannelCost.Amount {
		out = append(out, match(RuleChannelCost, mRuleCost, mRuleCostShort, mRuleCostShort, i18n.Args{"moneyCost": f.Cost, "moneyMin": p.ChannelCost.Amount}))
	}
	if p.AuthorLacksSend && !f.AuthorCanSend {
		perm := "campaigns:send"
		if f.SendPermission != "" {
			perm = f.SendPermission
		}
		label, args := mRuleLacksSend, i18n.Args{"author": f.AuthorName, "permission": perm}
		if f.AuthorRole != "" {
			label, args["role"] = mRuleLacksSendRole, f.AuthorRole
		}
		m := match(RuleAuthorLacksSend, label, mRuleLacksSendShort, mRuleLacksSendShort, args)
		m.Author, m.Role = f.AuthorName, f.AuthorRole
		out = append(out, m)
	}
	if p.ApprovalLists && len(f.ApprovalLists) > 0 {
		out = append(out, match(RuleApprovalLists, mRuleLists, mRuleListsShort, mRuleListsBadge, i18n.Args{"lists": f.ApprovalLists, "count": len(f.ApprovalLists)}))
	}
	if p.ClientSignoff.Enabled {
		out = append(out, clientSignoffMatch())
	}
	return out
}

// rulesLine is "rules: audience ≥ 10,000 · warming-up provider".
func rulesLine(rules []ApprovalRuleMatch) string { return rulesLineIn(i18n.Default, rules) }

// rulesLineIn is rulesLine in a reader's language.
func rulesLineIn(l i18n.Lang, rules []ApprovalRuleMatch) string {
	if len(rules) == 0 {
		return ""
	}
	return mRulesLine.Text(l, i18n.Args{"count": len(rules), "rules": shortsOf(localRules(l, rules))})
}

// shortsOf joins the rules' short phrases: "audience ≥ 10,000 · warming-up provider".
func shortsOf(rules []ApprovalRuleMatch) string {
	parts := make([]string, len(rules))
	for i, r := range rules {
		parts[i] = r.Short
	}
	return strings.Join(parts, " · ")
}

func ruleKeys(rules []ApprovalRuleMatch) []string {
	out := make([]string, len(rules))
	for i, r := range rules {
		out[i] = r.Key
	}
	return out
}

var (
	mRuleAudience          = msgs.M("rule.audience")
	mRuleAudienceShort     = msgs.M("rule.audienceShort")
	mRuleWarming           = msgs.M("rule.warming")
	mRuleWarmingDay        = msgs.M("rule.warmingDay")
	mRuleWarmingShort      = msgs.M("rule.warmingShort")
	mRuleWarmingBadge      = msgs.M("rule.warmingBadge")
	mRuleNewProvider       = msgs.M("rule.newProvider")
	mRuleNewProviderShort  = msgs.M("rule.newProviderShort")
	mRuleNewProviderBadge  = msgs.M("rule.newProviderBadge")
	mRuleWarnings          = msgs.M("rule.warnings")
	mRuleWarningsShort     = msgs.M("rule.warningsShort")
	mRuleCost              = msgs.M("rule.cost")
	mRuleCostShort         = msgs.M("rule.costShort")
	mRuleCostPolicy        = msgs.M("rule.costPolicy")
	mRuleLacksSend         = msgs.M("rule.lacksSend")
	mRuleLacksSendRole     = msgs.M("rule.lacksSendRole")
	mRuleLacksSendShort    = msgs.M("rule.lacksSendShort")
	mRuleLists             = msgs.M("rule.lists")
	mRuleListsShort        = msgs.M("rule.listsShort")
	mRuleListsBadge        = msgs.M("rule.listsBadge")
	mRuleSignoff           = msgs.M("rule.signoff")
	mRuleSignoffShort      = msgs.M("rule.signoffShort")
	mRuleSignoffBadge      = msgs.M("rule.signoffBadge")
	mRuleAutomation        = msgs.M("rule.automation")
	mRuleAutomationShort   = msgs.M("rule.automationShort")
	mRuleAutomationBadge   = msgs.M("rule.automationBadge")
	mRuleAutomationLacks   = msgs.M("rule.automationLacks")
	mRulePost              = msgs.M("rule.post")
	mRulePostShort         = msgs.M("rule.postShort")
	mRulePostBadge         = msgs.M("rule.postBadge")
	mRulePageShort         = msgs.M("rule.pageShort")
	mRulePageBadge         = msgs.M("rule.pageBadge")
	mRulePostLacks         = msgs.M("rule.postLacks")
	mRulePostLacksShort    = msgs.M("rule.postLacksShort")
	mRulePostLacksBadge    = msgs.M("rule.postLacksBadge")
	mRuleTemplate          = msgs.M("rule.template")
	mRuleTemplateShort     = msgs.M("rule.templateShort")
	mRuleTemplateBadge     = msgs.M("rule.templateBadge")
	mRulePolicyPostPublish = msgs.M("rule.policyPostPublish")
	mRulesLine             = msgs.M("rules.line")
)
