package readiness

import (
	"cmp"
	"context"
	"errors"
	"html"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Workspace sending rules (SR-10): checks a workspace adds to Send
// Readiness on every send in their scope, next to the built-in checks,
// with Warn (the sender confirms it) or Block (sending stays disabled until
// it's fixed); and the audience defaults (Settings › Sending rules).

// GroupRules is the Workspace rules group's key.
const GroupRules = "rules"

// What a rule checks, and how.
const (
	RuleLinkParam     = "link_param"
	RuleLinkDomain    = "link_domain"
	RuleSubjectLength = "subject_length"
	RuleContainsText  = "contains_text"
)

// ruleOperators are the operators of each kind of rule.
var ruleOperators = map[string][]string{
	RuleLinkParam:     {"on_every", "on_none"},
	RuleLinkDomain:    {"not_one_of", "one_of"},
	RuleSubjectLength: {"at_most", "at_least"},
	RuleContainsText:  {"footer_contains", "footer_lacks", "body_contains", "subject_contains"},
}

// Shorteners is the value of a link-domain rule that names the URL
// shortener domains (ShortenerDomains).
const Shorteners = "shorteners"

// ShortenerDomains are the URL shorteners a "No URL shorteners" rule keeps
// out of links: mailbox providers distrust them, and a shortener's domain
// on a blocklist takes every email that uses it along.
var ShortenerDomains = []string{"bit.ly", "t.co", "tinyurl.com", "ow.ly", "buff.ly", "goo.gl", "is.gd", "rebrand.ly", "cutt.ly", "shorturl.at",
	"t.ly", "tiny.cc", "bl.ink", "rb.gy", "lnkd.in", "s.id", "v.gd", "soo.gd", "clck.ru", "qr.ae", "adf.ly", "bit.do", "mcaf.ee", "su.pr", "po.st",
	"x.co", "db.tt", "trib.al", "dlvr.it", "fb.me", "amzn.to", "wp.me", "ift.tt", "shorte.st", "tr.im", "cli.gs", "snip.ly", "hubs.ly", "mzl.la", "j.mp"}

// SendingRule is a workspace sending rule.
type SendingRule struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Kind is what the rule checks: link_param (a query parameter on the
	// links), link_domain (where links point), subject_length or
	// contains_text; Operator how (ruleOperators); Value the parameter,
	// the domains (comma-separated, or "shorteners"), the length or the
	// text.
	Kind      string `json:"kind" enum:"link_param,link_domain,subject_length,contains_text"`
	Operator  string `json:"operator" enum:"on_every,on_none,not_one_of,one_of,at_most,at_least,footer_contains,footer_lacks,body_contains,subject_contains"`
	Value     string `json:"value"`
	Severity  string `json:"severity" enum:"warn,block"`
	AppliesTo string `json:"appliesTo" enum:"broadcast,automation,both"`
	Enabled   bool   `json:"enabled"`
	// Message is shown to senders next to the check.
	Message string `json:"message"`
	// Description says what the rule checks, in the reader's language
	// ("Every link has a utm_campaign query parameter").
	Description string    `json:"description"`
	AddedBy     string    `json:"addedBy"`
	AddedAt     time.Time `json:"addedAt"`
	// Triggered is the last send a run found breaking the rule.
	Triggered *SendingRuleTrigger `json:"triggered,omitempty"`
}

// SendingRuleTrigger is the last send a run found breaking a rule.
type SendingRuleTrigger struct {
	At         time.Time `json:"at"`
	CampaignID string    `json:"campaignId,omitempty"`
	Campaign   string    `json:"campaign"`
	// Count is how many links broke it (0 for a subject or a text).
	Count int `json:"count"`
	// Fixed is true when a later run of that send found it kept.
	Fixed bool `json:"fixed"`
}

// SendingRuleInput adds or changes a rule.
type SendingRuleInput struct {
	Name      string `json:"name" minLength:"1" maxLength:"120"`
	Kind      string `json:"kind" enum:"link_param,link_domain,subject_length,contains_text"`
	Operator  string `json:"operator" enum:"on_every,on_none,not_one_of,one_of,at_most,at_least,footer_contains,footer_lacks,body_contains,subject_contains"`
	Value     string `json:"value" maxLength:"2000"`
	Severity  string `json:"severity" enum:"warn,block"`
	AppliesTo string `json:"appliesTo" enum:"broadcast,automation,both"`
	Enabled   bool   `json:"enabled"`
	Message   string `json:"message,omitempty" maxLength:"500"`
}

// SendDefaults are how a workspace's sends count and confirm their
// audience: the unengaged (the sunset tier) are left out of every send
// (ExcludeUnengaged), and a sender types the recipient count when the
// audience is more than TypedMultiple times the usual one.
type SendDefaults struct {
	ExcludeUnengaged bool    `json:"excludeUnengaged"`
	TypedMultiple    float64 `json:"typedMultiple" minimum:"1.1" maximum:"20"`
}

// SendingRulesView is Settings › Sending rules.
type SendingRulesView struct {
	Rules    []SendingRule `json:"rules"`
	Defaults SendDefaults  `json:"defaults"`
	// SunsetDays is the sunset policy's (contacts unengaged that long are
	// left out).
	SunsetDays int `json:"sunsetDays"`
	// CanManage is true for members who can change the rules; Managers
	// names up to two of them ("Linh Nguyen and Marcus Chen"), and Reason
	// says why the reader can't.
	CanManage bool     `json:"canManage"`
	Managers  []string `json:"managers"`
	Reason    string   `json:"reason,omitempty"`
}

// RuleFacts is a rule as a run checks it.
type RuleFacts struct {
	ID, Name, Kind, Operator, Value, Severity, AppliesTo, Message, AddedBy string
}

// SendingRules is Settings › Sending rules for a reader.
func (s *Service) SendingRules(ctx context.Context, ws *store.Workspace, p *authz.Principal) (SendingRulesView, error) {
	out := SendingRulesView{Rules: []SendingRule{}, SunsetDays: sunsetDays(ws.SunsetPolicy), CanManage: p.Can(authz.WorkspaceManage),
		Managers: access.Helpers(ctx, s.Q, ws.ID, authz.WorkspaceManage)}
	if out.Managers == nil {
		out.Managers = []string{}
	}
	if !out.CanManage {
		// Who can, by name, never a permission key.
		names := "none"
		if len(out.Managers) > 0 {
			names = i18n.OrList(i18n.LangFrom(ctx).Locale, out.Managers)
		}
		out.Reason = say(ctx, "rules.whoCanChange", A{"names": names})
	}
	d, err := s.Q.SendDefaults(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Defaults = SendDefaults{ExcludeUnengaged: d.ExcludeUnengaged, TypedMultiple: d.TypedMultiple}
	rows, err := s.Q.SendingRulesList(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Rules = append(out.Rules, ruleView(ctx, r.SendingRule, r.CreatedByName))
	}
	return out, nil
}

func ruleView(ctx context.Context, r store.SendingRule, by string) SendingRule {
	out := SendingRule{ID: r.ID.String(), Name: r.Name, Kind: r.Kind, Operator: r.Operator, Value: r.Value, Severity: r.Severity, AppliesTo: r.AppliesTo,
		Enabled: r.Enabled, Message: r.Message, AddedBy: by, AddedAt: r.CreatedAt}
	out.Description = ruleDescription(ruleFacts(r, by)).In(i18n.LangFrom(ctx))
	if r.TriggeredAt != nil {
		t := &SendingRuleTrigger{At: *r.TriggeredAt, Campaign: r.TriggeredName, Count: int(r.TriggeredCount), Fixed: r.TriggeredFixed}
		if r.TriggeredCampaign != nil {
			t.CampaignID = r.TriggeredCampaign.String()
		}
		out.Triggered = t
	}
	return out
}

func ruleFacts(r store.SendingRule, by string) RuleFacts {
	return RuleFacts{ID: r.ID.String(), Name: r.Name, Kind: r.Kind, Operator: r.Operator, Value: r.Value, Severity: r.Severity, AppliesTo: r.AppliesTo,
		Message: r.Message, AddedBy: by}
}

// validRule checks a rule's input and returns it tidied: the value of a
// domain list is lower-case domains, a length is a number.
func validRule(ctx context.Context, in SendingRuleInput) (SendingRuleInput, error) {
	in.Name, in.Value, in.Message = strings.TrimSpace(in.Name), strings.TrimSpace(in.Value), strings.TrimSpace(in.Message)
	if in.Name == "" {
		return in, api.Invalid("body.name", say(ctx, "problem.ruleName", nil))
	}
	if !slices.Contains(ruleOperators[in.Kind], in.Operator) {
		return in, api.Invalid("body.operator", say(ctx, "problem.ruleOperator", nil))
	}
	switch in.Kind {
	case RuleLinkParam:
		if in.Value == "" || strings.ContainsAny(in.Value, "=&?# ") {
			return in, api.Invalid("body.value", say(ctx, "problem.ruleParam", nil))
		}
	case RuleLinkDomain:
		if strings.EqualFold(in.Value, Shorteners) {
			in.Value = Shorteners
			break
		}
		domains := ruleDomains(in.Value)
		if len(domains) == 0 {
			return in, api.Invalid("body.value", say(ctx, "problem.ruleDomains", nil))
		}
		for _, d := range domains {
			if !strings.Contains(d, ".") || strings.ContainsAny(d, "/:@ ") {
				return in, api.Invalid("body.value", say(ctx, "problem.ruleDomain", A{"domain": d}))
			}
		}
		in.Value = strings.Join(domains, ", ")
	case RuleSubjectLength:
		n, err := strconv.Atoi(in.Value)
		if err != nil || n < 1 || n > 998 {
			return in, api.Invalid("body.value", say(ctx, "problem.ruleLength", nil))
		}
		in.Value = strconv.Itoa(n)
	case RuleContainsText:
		if in.Value == "" {
			return in, api.Invalid("body.value", say(ctx, "problem.ruleText", nil))
		}
	}
	return in, nil
}

// ruleDomains reads a domain list: comma-, space- or line-separated, lower
// case, without a scheme or "www.".
func ruleDomains(v string) []string {
	out := []string{}
	for _, d := range strings.FieldsFunc(strings.ToLower(v), func(r rune) bool { return r == ',' || r == ' ' || r == '\n' || r == ';' }) {
		d = strings.TrimPrefix(strings.TrimPrefix(d, "https://"), "http://")
		d = strings.TrimSuffix(strings.TrimPrefix(d, "www."), "/")
		if d != "" && !slices.Contains(out, d) {
			out = append(out, d)
		}
	}
	return out
}

// CreateSendingRule adds a rule (workspace:manage, audited).
func (s *Service) CreateSendingRule(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SendingRuleInput) (SendingRule, error) {
	in, err := validRule(ctx, in)
	if err != nil {
		return SendingRule{}, err
	}
	id, now := uuid.Must(uuid.NewV7()), s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SendingRuleInsert(ctx, store.SendingRuleInsertParams{ID: id, WorkspaceID: ws.ID, Name: in.Name, Kind: in.Kind, Operator: in.Operator, Value: in.Value,
			Severity: in.Severity, AppliesTo: in.AppliesTo, Enabled: in.Enabled, Message: in.Message, CreatedBy: userOf(p), Now: now}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sending_rule.created", TargetType: "sending_rule", TargetID: id.String(), TargetName: in.Name,
			After: ruleAudit(in)})
	})
	if err != nil {
		return SendingRule{}, err
	}
	return s.sendingRule(ctx, ws, id)
}

// UpdateSendingRule changes a rule, its severity, scope or status
// (workspace:manage, audited).
func (s *Service) UpdateSendingRule(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in SendingRuleInput) (SendingRule, error) {
	in, err := validRule(ctx, in)
	if err != nil {
		return SendingRule{}, err
	}
	before, err := s.Q.SendingRuleGet(ctx, store.SendingRuleGetParams{WorkspaceID: ws.ID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return SendingRule{}, ruleGone(ctx)
	}
	if err != nil {
		return SendingRule{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.SendingRuleUpdate(ctx, store.SendingRuleUpdateParams{Name: in.Name, Kind: in.Kind, Operator: in.Operator, Value: in.Value, Severity: in.Severity,
			AppliesTo: in.AppliesTo, Enabled: in.Enabled, Message: in.Message, Now: s.now(), WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sending_rule.updated", TargetType: "sending_rule", TargetID: id.String(), TargetName: in.Name,
			Before: ruleAudit(SendingRuleInput{Name: before.Name, Kind: before.Kind, Operator: before.Operator, Value: before.Value, Severity: before.Severity,
				AppliesTo: before.AppliesTo, Enabled: before.Enabled, Message: before.Message}), After: ruleAudit(in)})
	})
	if err != nil {
		return SendingRule{}, err
	}
	return s.sendingRule(ctx, ws, id)
}

// DeleteSendingRule removes a rule (workspace:manage, audited).
func (s *Service) DeleteSendingRule(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	before, err := s.Q.SendingRuleGet(ctx, store.SendingRuleGetParams{WorkspaceID: ws.ID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return ruleGone(ctx)
	}
	if err != nil {
		return err
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.SendingRuleDelete(ctx, store.SendingRuleDeleteParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "sending_rule.deleted", TargetType: "sending_rule", TargetID: id.String(), TargetName: before.Name,
			Before: ruleAudit(SendingRuleInput{Name: before.Name, Kind: before.Kind, Operator: before.Operator, Value: before.Value, Severity: before.Severity,
				AppliesTo: before.AppliesTo, Enabled: before.Enabled, Message: before.Message})})
	})
}

// SaveSendDefaults changes the audience defaults (workspace:manage,
// audited).
func (s *Service) SaveSendDefaults(ctx context.Context, ws *store.Workspace, p *authz.Principal, in SendDefaults) (SendingRulesView, error) {
	if in.TypedMultiple < 1.1 || in.TypedMultiple > 20 {
		return SendingRulesView{}, api.Invalid("body.typedMultiple", say(ctx, "problem.typedMultiple", nil))
	}
	in.TypedMultiple = math.Round(in.TypedMultiple*10) / 10
	before, err := s.Q.SendDefaults(ctx, ws.ID)
	if err != nil {
		return SendingRulesView{}, err
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SendDefaultsSave(ctx, store.SendDefaultsSaveParams{WorkspaceID: ws.ID, ExcludeUnengaged: in.ExcludeUnengaged,
			TypedMultiple: in.TypedMultiple, Now: s.now(), UpdatedBy: userOf(p)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "send_defaults.updated", TargetType: "workspace", TargetID: ws.ID.String(), TargetName: ws.Name,
			Before: map[string]any{"exclude_unengaged": before.ExcludeUnengaged, "typed_multiple": before.TypedMultiple},
			After:  map[string]any{"exclude_unengaged": in.ExcludeUnengaged, "typed_multiple": in.TypedMultiple}})
	})
	if err != nil {
		return SendingRulesView{}, err
	}
	return s.SendingRules(ctx, ws, p)
}

func (s *Service) sendingRule(ctx context.Context, ws *store.Workspace, id uuid.UUID) (SendingRule, error) {
	rows, err := s.Q.SendingRulesList(ctx, ws.ID)
	if err != nil {
		return SendingRule{}, err
	}
	for _, r := range rows {
		if r.SendingRule.ID == id {
			return ruleView(ctx, r.SendingRule, r.CreatedByName), nil
		}
	}
	return SendingRule{}, ruleGone(ctx)
}

func ruleGone(ctx context.Context) error {
	return api.Err(http.StatusNotFound, api.CodeNotFound, say(ctx, "problem.ruleGone", nil))
}

func ruleAudit(in SendingRuleInput) map[string]any {
	return map[string]any{"name": in.Name, "kind": in.Kind, "operator": in.Operator, "value": in.Value, "severity": in.Severity, "applies_to": in.AppliesTo,
		"enabled": in.Enabled, "message": in.Message}
}

func userOf(p *authz.Principal) *uuid.UUID {
	if p == nil || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

// rulesFor are the rules a send in stream checks, as a run reads them.
func (s *Service) rulesFor(ctx context.Context, wsID uuid.UUID, stream string) []RuleFacts {
	rows, err := s.Q.ReadinessSendingRules(ctx, store.ReadinessSendingRulesParams{WorkspaceID: wsID, Stream: stream})
	if err != nil {
		return nil
	}
	out := make([]RuleFacts, 0, len(rows))
	for _, r := range rows {
		out = append(out, ruleFacts(r.SendingRule, r.CreatedByName))
	}
	return out
}

// noteRules records on each rule the send a run found breaking it, and
// marks a rule kept when a later run of that send finds it fixed.
func noteRules(ctx context.Context, q *store.Queries, c store.ReadinessCampaignRow, res ReadinessResult, at time.Time) error {
	for _, ch := range res.Checks {
		key, ok := strings.CutPrefix(ch.Key, "rules.")
		if !ok {
			continue
		}
		id, err := uuid.Parse(key)
		if err != nil {
			continue
		}
		if ch.Status == Pass {
			if err := q.SendingRuleFixed(ctx, store.SendingRuleFixedParams{ID: id, CampaignID: &c.ID}); err != nil {
				return err
			}
			continue
		}
		// The links that broke it are the raw lines of its details.
		count := int32(len(ch.Raw))
		if err := q.SendingRuleTriggered(ctx, store.SendingRuleTriggeredParams{ID: id, At: &at, CampaignID: &c.ID, CampaignName: c.Name, Count: count}); err != nil {
			return err
		}
	}
	return nil
}

// ruleLink is a link of the email a rule reads: its address and where it
// points.
type ruleLink struct {
	URL, Host string
	Query     url.Values
}

// ruleLinks are the email's links a rule reads: the web links it points
// readers to (not mailto:, merge tags or the unsubscribe links ReputeCast
// adds), each once.
func (b *builder) ruleLinks() []ruleLink {
	out := []ruleLink{}
	seen := map[string]bool{}
	for _, l := range b.f.Content.Report.LinkList {
		u, err := url.Parse(strings.TrimSpace(l.URL))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || seen[l.URL] {
			continue
		}
		seen[l.URL] = true
		out = append(out, ruleLink{URL: l.URL, Host: strings.TrimPrefix(strings.ToLower(u.Hostname()), "www."), Query: u.Query()})
	}
	return out
}

// onDomain reports whether host is domain or one of its subdomains.
func onDomain(host, domain string) bool { return host == domain || strings.HasSuffix(host, "."+domain) }

// ruleDescription says what a rule checks.
func ruleDescription(r RuleFacts) Text {
	switch r.Kind {
	case RuleLinkParam:
		if r.Operator == "on_none" {
			return tx("rules.desc.paramNone", A{"param": r.Value})
		}
		return tx("rules.desc.paramEvery", A{"param": r.Value})
	case RuleLinkDomain:
		if r.Value == Shorteners {
			return tx("rules.desc.shorteners", A{"first": argJoin(", ", ShortenerDomains[:3]), "more": len(ShortenerDomains) - 3})
		}
		domains := ruleDomains(r.Value)
		if r.Operator == "one_of" {
			return tx("rules.desc.oneOf", A{"domains": ArgOr(domains)})
		}
		return tx("rules.desc.notOneOf", A{"domains": ArgOr(domains)})
	case RuleSubjectLength:
		n, _ := strconv.Atoi(r.Value)
		if r.Operator == "at_least" {
			return tx("rules.desc.atLeast", A{"n": n})
		}
		return tx("rules.desc.atMost", A{"n": n})
	case RuleContainsText:
		return tx("rules.desc."+r.Operator, A{"text": r.Value})
	}
	return RawText(r.Name)
}

// ruleFinding is what a run found for a rule.
type ruleFinding struct {
	broken  bool
	finding Text
	// count of links breaking it, of total; bad are their addresses.
	count, total int
	bad          []string
	// what names the links in the technical details ("Links without it").
	what Text
}

func (b *builder) evalRule(r RuleFacts) ruleFinding {
	links := b.ruleLinks()
	out := ruleFinding{total: len(links)}
	linkRule := func(bad func(ruleLink) bool) {
		for _, l := range links {
			if bad(l) {
				out.count++
				out.bad = append(out.bad, l.URL)
			}
		}
		out.broken = out.count > 0
	}
	switch r.Kind {
	case RuleLinkParam:
		if r.Operator == "on_none" {
			linkRule(func(l ruleLink) bool { return l.Query.Has(r.Value) })
			out.what = tx("tech.linksWithParam", nil)
			if out.broken {
				out.finding = tx("rules.hasParam", A{"count": out.count, "total": out.total, "param": r.Value})
			} else {
				out.finding = tx("rules.noneParam", A{"total": out.total, "param": r.Value})
			}
			break
		}
		linkRule(func(l ruleLink) bool { return !l.Query.Has(r.Value) })
		out.what = tx("tech.linksWithoutParam", nil)
		if out.broken {
			out.finding = tx("rules.missingParam", A{"count": out.count, "total": out.total, "param": r.Value})
		} else {
			out.finding = tx("rules.allParam", A{"total": out.total, "param": r.Value})
		}
	case RuleLinkDomain:
		domains := ShortenerDomains
		if r.Value != Shorteners {
			domains = ruleDomains(r.Value)
		}
		listed := func(l ruleLink) bool {
			return slices.ContainsFunc(domains, func(d string) bool { return onDomain(l.Host, d) })
		}
		found := []string{}
		note := func(l ruleLink) {
			if !slices.Contains(found, l.Host) {
				found = append(found, l.Host)
			}
		}
		if r.Operator == "one_of" {
			linkRule(func(l ruleLink) bool {
				if listed(l) {
					return false
				}
				note(l)
				return true
			})
			out.what = tx("tech.linksElsewhere", nil)
			if out.broken {
				out.finding = tx("rules.elsewhere", A{"count": out.count, "total": out.total, "domains": ArgAnd(found)})
			} else {
				out.finding = tx("rules.allListed", A{"total": out.total})
			}
			break
		}
		linkRule(func(l ruleLink) bool {
			if !listed(l) {
				return false
			}
			note(l)
			return true
		})
		out.what = tx("tech.linksListed", nil)
		switch {
		case out.broken:
			out.finding = tx("rules.listed", A{"count": out.count, "total": out.total, "domains": ArgAnd(found)})
		case r.Value == Shorteners:
			out.finding = tx("rules.noShorteners", A{"total": out.total})
		default:
			out.finding = tx("rules.noneListed", A{"total": out.total})
		}
	case RuleSubjectLength:
		limit, _ := strconv.Atoi(r.Value)
		n := utf8.RuneCountInString(b.f.Campaign.Subject)
		switch {
		case r.Operator == "at_most" && n > limit:
			out.broken, out.finding = true, tx("rules.subjectLong", A{"n": n, "limit": limit})
		case r.Operator == "at_least" && n < limit:
			out.broken, out.finding = true, tx("rules.subjectShort", A{"n": n, "limit": limit})
		default:
			out.finding = tx("rules.subjectLength", A{"n": n})
		}
		return out
	case RuleContainsText:
		text := strings.ToLower(r.Value)
		has := func(s string) bool { return strings.Contains(strings.ToLower(collapseSpace(s)), collapseSpace(text)) }
		switch r.Operator {
		case "footer_contains":
			out.broken = !has(b.f.Content.FooterText)
		case "footer_lacks":
			out.broken = has(b.f.Content.FooterText)
		case "body_contains":
			out.broken = !has(b.f.Content.Report.PlainText)
		case "subject_contains":
			out.broken = !has(b.f.Campaign.Subject)
		}
		key := "rules.kept." + r.Operator
		if out.broken {
			key = "rules.broken." + r.Operator
		}
		out.finding = tx(key, A{"text": r.Value})
		return out
	}
	if out.total == 0 {
		out.finding = tx("rules.noLinks", nil)
	}
	return out
}

func collapseSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

// rules adds a check per workspace rule: kept, or broken with the rule's
// severity, its findings and their fix (SR-10).
func (b *builder) rules() {
	for _, r := range b.f.Rules {
		res := b.evalRule(r)
		name := RawText(r.Name)
		c := check{Key: "rules." + r.ID, Group: GroupRules, Status: Pass, Label: name, Short: name}
		scope := tx("rules.scope", A{"severity": r.Severity, "scope": r.AppliesTo})
		if !res.broken {
			c.Title = tx("rules.kept", A{"name": r.Name, "finding": res.finding})
			c.Detail = scope
			b.add(c)
			continue
		}
		c.Status = Warn
		if r.Severity == "block" {
			c.Status = Block
		}
		c.Title = tx("rules.broken", A{"name": argLower(r.Name), "finding": res.finding})
		c.FlatTitle = c.Title
		if r.Message != "" {
			c.Detail = RawText(r.Message)
		}
		if c.Status == Warn {
			if r.Kind == RuleLinkParam && r.Operator == "on_every" {
				c.Ack = tx("rules.ackAsks", A{"finding": res.finding})
			} else {
				c.Ack = tx("rules.ack", A{"finding": res.finding, "name": r.Name})
			}
		}
		// The rule, what it checks and what broke it (technical details).
		by := cmp.Or(r.AddedBy, "none")
		c.Technical = []techRow{{Label: tx("tech.rule", nil), Value: tx("tech.ruleValue", A{"name": r.Name, "severity": r.Severity, "scope": r.AppliesTo, "by": by})},
			{Label: tx("tech.checks", nil), Value: sentences(tx("text", A{"text": argSentence(ruleDescription(r))}), RawText(r.Message))}}
		if res.count > 0 {
			c.Technical = append(c.Technical, techRow{Label: res.what, Value: tx("tech.ofLinks", A{"count": res.count, "total": res.total,
				"domain": cmp.Or(commonDomain(res.bad), "none")})})
			c.Raw = res.bad
		}
		switch {
		case r.Kind == RuleLinkParam && r.Operator == "on_every":
			value := slug(b.f.Campaign.Name)
			c.Fixes = append(c.Fixes, apply(FixRuleAddParam, tx("fix.addParam", A{"param": r.Value, "value": value}), "primary",
				map[string]string{"rule": r.ID, "param": r.Value, "value": value, "count": strconv.Itoa(res.count)}))
			c.Note = tx("rules.changesLinks", A{"count": res.count})
		case r.Kind == RuleSubjectLength || r.Operator == "subject_contains":
			c.Fixes = append(c.Fixes, link(tx("fix.editSubject", nil), b.editor(nil, "settings")))
		default:
			c.Fixes = append(c.Fixes, link(tx("fix.openInEditor", nil), b.editor(nil, "")))
		}
		b.add(c)
	}
}

// commonDomain is the organizational domain all the links share, "" when
// they don't share one.
func commonDomain(urls []string) string {
	common := ""
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		d := dnscheck.OrgDomain(strings.TrimPrefix(strings.ToLower(u.Hostname()), "www."))
		if common != "" && d != common {
			return ""
		}
		common = d
	}
	return common
}

var slugParts = regexp.MustCompile(`[^a-z0-9]+`)

// slug is a campaign's name as a query value: "Weekly Digest #142" →
// "weekly-digest-142".
func slug(name string) string {
	s := strings.Trim(slugParts.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if s == "" {
		return "campaign"
	}
	return s
}

// FixRuleAddParam adds a workspace rule's query parameter to the links
// missing it ("Add utm_campaign=weekly-digest-142").
const FixRuleAddParam = "rule_add_param"

var hrefAttr = regexp.MustCompile(`(?i)(href\s*=\s*)("([^"]*)"|'([^']*)')`)

// AddLinkParam adds param=value to every web link of src without param
// (merge tags, mailto: and anchors are left as they are) and says how many
// it changed.
func AddLinkParam(src, param, value string) (string, int) {
	n := 0
	out := hrefAttr.ReplaceAllStringFunc(src, func(m string) string {
		sub := hrefAttr.FindStringSubmatch(m)
		quote, raw := `"`, sub[3]
		if strings.HasPrefix(sub[2], "'") {
			quote, raw = "'", sub[4]
		}
		target := html.UnescapeString(raw)
		u, err := url.Parse(strings.TrimSpace(target))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || strings.Contains(target, "{{") || u.Query().Has(param) {
			return m
		}
		sep := "?"
		if strings.Contains(target, "?") {
			sep = "&"
		}
		frag := ""
		if i := strings.IndexByte(target, '#'); i >= 0 {
			target, frag = target[:i], target[i:]
		}
		next := target + sep + url.QueryEscape(param) + "=" + url.QueryEscape(value) + frag
		if strings.Contains(raw, "&amp;") {
			next = strings.ReplaceAll(next, "&", "&amp;")
		}
		n++
		return sub[1] + quote + next + quote
	})
	return out, n
}

// footerText is the text of the email's footer: the composer's footer
// block, or the end of the plain text of an email without one.
func footerText(mjmlSrc, plain string) string {
	if i := strings.Index(mjmlSrc, footerMarker); i >= 0 {
		rest := mjmlSrc[i+len(footerMarker):]
		if end := strings.Index(rest, "</mj-section>"); end >= 0 {
			rest = rest[:end]
		}
		return collapseSpace(html.UnescapeString(tagPattern.ReplaceAllString(rest, " ")))
	}
	plain = collapseSpace(plain)
	if n := len(plain); n > 600 {
		plain = plain[n-600:]
	}
	return plain
}

var tagPattern = regexp.MustCompile(`<[^>]*>`)

// sendDefaults is the workspace's audience defaults (the defaults when it
// can't be read).
func sendDefaults(ctx context.Context, q *store.Queries, wsID uuid.UUID) SendDefaults {
	d, err := q.SendDefaults(ctx, wsID)
	if err != nil {
		return SendDefaults{ExcludeUnengaged: true, TypedMultiple: 1.5}
	}
	return SendDefaults{ExcludeUnengaged: d.ExcludeUnengaged, TypedMultiple: d.TypedMultiple}
}
