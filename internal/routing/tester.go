package routing

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// RouteTestInput asks which provider would send an email.
type RouteTestInput struct {
	Stream string `json:"stream" enum:"broadcast,automation,system,transactional"`
	ListID string `json:"listId,omitempty"`
	// Template is a transactional template's ID ("password-reset"): its
	// category decides category conditions.
	Template     string `json:"template,omitempty" maxLength:"64"`
	SenderDomain string `json:"senderDomain,omitempty" maxLength:"253"`
	Email        string `json:"email" minLength:"3" maxLength:"254"`
	// Draft tests unsaved routing instead of the saved one.
	Draft *RoutingSaveInput `json:"draft,omitempty"`
}

// RouteTestSplit is one provider of a weighted split, for the step text.
type RouteTestSplit struct {
	Short  string `json:"short"`
	Weight int    `json:"weight"`
}

// RouteTestStep is one step of the decision, in order.
type RouteTestStep struct {
	// Kind: stream, condition, split, primary, suppression, held.
	Kind     string           `json:"kind" enum:"stream,condition,split,primary,suppression,held"`
	Stream   string           `json:"stream,omitempty"`
	Field    string           `json:"field,omitempty"`
	Value    string           `json:"value,omitempty"`
	Provider string           `json:"provider,omitempty"`
	Bucket   int              `json:"bucket"`
	Splits   []RouteTestSplit `json:"splits,omitempty"`
	// Conditions is how many conditions the stream has (stream step).
	Conditions int `json:"conditions"`
	// Warming names a provider of the default split that is warming up
	// ("Postal is still warming up").
	Warming    string `json:"warming,omitempty"`
	Suppressed bool   `json:"suppressed"`
	Email      string `json:"email,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// RouteTestCost is what one message costs on the chosen provider.
type RouteTestCost struct {
	Basis      string   `json:"basis" enum:"metered,plan,self_hosted"`
	PerMessage *float64 `json:"perMessage,omitempty"`
	Per1000    *float64 `json:"per1000,omitempty"`
}

// RouteTestFallback says what happens if the provider fails.
type RouteTestFallback struct {
	Policy   string `json:"policy" enum:"automatic,pause_and_ask,warm_only"`
	Provider string `json:"provider,omitempty"`
	Short    string `json:"short,omitempty"`
}

// RouteTestResult answers "Which provider would send this email?".
type RouteTestResult struct {
	ProviderID      string `json:"providerId"`
	Provider        string `json:"provider"`
	Short           string `json:"short"`
	Color           string `json:"color" enum:"ses,postal,postmark,other"`
	MailboxProvider string `json:"mailboxProvider"`
	// MailboxDetail names the hosting behind a company domain ("Google
	// Workspace"), found from its MX records; empty for consumer domains.
	MailboxDetail string            `json:"mailboxDetail,omitempty"`
	DKIMSelector  string            `json:"dkimSelector"`
	Cost          RouteTestCost     `json:"cost"`
	Steps         []RouteTestStep   `json:"steps"`
	Fallback      RouteTestFallback `json:"fallback"`
	Suppressed    bool              `json:"suppressed"`
	// Held is true when the provider can't take mail right now.
	Held bool `json:"held"`
}

// Test runs the route tester for one address.
func (s *Service) Test(ctx context.Context, ws *store.Workspace, in RouteTestInput) (RouteTestResult, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if i := strings.LastIndexByte(email, '@'); i < 1 || i == len(email)-1 {
		return RouteTestResult{}, api.Invalid("body.email", say(ctx, "problem.email", nil))
	}
	set, err := Load(ctx, s.Q, ws.ID)
	if err != nil {
		return RouteTestResult{}, err
	}
	if in.Draft != nil {
		if set, err = toSet(ctx, *in.Draft, set); err != nil {
			return RouteTestResult{}, err
		}
	}
	stream := providers.Stream(in.Stream)
	rcpt := Recipient{Stream: stream, Email: email, SenderDomain: in.SenderDomain}
	if in.ListID != "" {
		rcpt.Lists = []string{in.ListID}
	}
	if stream == StreamTransactional && in.Template != "" {
		if cat, err := s.Q.TxnRouteTemplateCategory(ctx, store.TxnRouteTemplateCategoryParams{WorkspaceID: ws.ID, Key: in.Template}); err == nil && cat != nil {
			rcpt.Category = cat.String()
		}
	}
	// A known contact carries its mailbox provider (from MX) and lists.
	detail := ""
	if c, err := s.Q.RoutingContact(ctx, store.RoutingContactParams{WorkspaceID: ws.ID, Email: email}); err == nil {
		if c.MailboxProvider != "" && c.MailboxProvider != Other {
			rcpt.MailboxProvider = c.MailboxProvider
			detail = c.MailboxDetail
		}
		if in.ListID == "" {
			for _, l := range c.ListIds {
				rcpt.Lists = append(rcpt.Lists, l.String())
			}
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return RouteTestResult{}, err
	}
	if rcpt.MailboxProvider == "" {
		if s.Mailbox != nil {
			cl := s.Mailbox.ForEmail(ctx, email)
			rcpt.MailboxProvider, detail = cl.Provider, cl.Detail
		} else {
			rcpt.MailboxProvider = MailboxProvider(email)
		}
	}
	d, ok := set.Resolve(rcpt)
	if !ok {
		return RouteTestResult{}, api.Err(422, "no_route", say(ctx, "problem.noRoute", i18n.Args{"stream": streamArg(stream)}))
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return RouteTestResult{}, err
	}
	byID := map[uuid.UUID]store.Provider{}
	for _, p := range provs {
		byID[p.ID] = p
	}
	p, found := byID[d.ProviderID]
	if !found {
		return RouteTestResult{}, api.Err(422, "no_route", say(ctx, "problem.goneProvider", nil))
	}
	facts, _, err := s.facts(ctx, ws)
	if err != nil {
		return RouteTestResult{}, err
	}
	out := RouteTestResult{ProviderID: p.ID.String(), Provider: p.Name, Short: state.Short(p.Name, p.Adapter), Color: state.Color(p.Adapter),
		MailboxProvider: rcpt.MailboxProvider, MailboxDetail: detail, Steps: []RouteTestStep{}}
	conditions := 0
	for _, r := range set.StreamRules(stream) {
		if !r.Conditions.Empty() {
			conditions++
		}
	}
	def, _ := set.Default(stream)
	primary := ""
	if len(def.Splits) > 0 {
		if pp, ok := byID[def.Splits[0].ProviderID]; ok {
			primary = pp.Name
		}
	}
	out.Steps = append(out.Steps, RouteTestStep{Kind: "stream", Stream: string(stream), Conditions: conditions, Provider: primary})
	switch {
	case d.Matched != "":
		step := RouteTestStep{Kind: "condition", Field: d.Matched, Value: d.MatchedValue, Provider: p.Name}
		switch d.Matched {
		case "mailbox_provider":
			step.Value = MailboxNameIn(i18n.LangFrom(ctx), d.MatchedValue)
		case "category":
			if cats, err := s.Q.TxnRouteCategories(ctx, ws.ID); err == nil {
				for _, c := range cats {
					if c.ID.String() == d.MatchedValue {
						step.Value = c.Name
					}
				}
			}
		case "list":
			if lists, err := s.Q.RoutingLists(ctx, ws.ID); err == nil {
				for _, l := range lists {
					if l.ID.String() == d.MatchedValue {
						step.Value = l.Name
					}
				}
			}
		}
		for _, sp := range def.Splits {
			if f, ok := facts[sp.ProviderID]; ok && f.WarmingUp && sp.ProviderID != d.ProviderID {
				step.Warming = f.Short
			}
		}
		out.Steps = append(out.Steps, step)
	case d.Split:
		step := RouteTestStep{Kind: "split", Bucket: d.Bucket, Provider: p.Name}
		for _, sp := range d.Rule.Splits {
			if pp, ok := byID[sp.ProviderID]; ok {
				step.Splits = append(step.Splits, RouteTestSplit{Short: state.Short(pp.Name, pp.Adapter), Weight: sp.Weight})
			}
		}
		out.Steps = append(out.Steps, step)
	}
	suppressed, err := s.Q.RoutingIsSuppressed(ctx, store.RoutingIsSuppressedParams{WorkspaceID: ws.ID, Email: email})
	if err != nil {
		return RouteTestResult{}, err
	}
	if stream == StreamTransactional && suppressed {
		// Unsubscribes and role addresses are marketing-only: transactional
		// mail still reaches them.
		if suppressed, err = s.Q.TxnRouteSuppressedAllMail(ctx, store.TxnRouteSuppressedAllMailParams{WorkspaceID: ws.ID, Email: email}); err != nil {
			return RouteTestResult{}, err
		}
	}
	out.Suppressed = suppressed
	out.Steps = append(out.Steps, RouteTestStep{Kind: "suppression", Suppressed: suppressed, Email: email})
	if !state.Available(p) {
		out.Held = true
		reason := "down"
		if p.Status == state.Degraded {
			reason = "webhooks_silent"
		}
		out.Steps = append(out.Steps, RouteTestStep{Kind: "held", Provider: p.Name, Reason: reason})
	}
	out.Fallback = RouteTestFallback{Policy: string(d.Policy)}
	if d.Fallback != nil {
		if fp, ok := byID[*d.Fallback]; ok {
			out.Fallback.Provider, out.Fallback.Short = fp.Name, state.Short(fp.Name, fp.Adapter)
		}
	}
	out.DKIMSelector = s.dkim(ctx, ws.ID, p.ID, in.SenderDomain)
	out.Cost = s.cost(ctx, p)
	return out, nil
}

// dkim returns the DKIM record name the provider signs with for a domain.
func (s *Service) dkim(ctx context.Context, wsID, providerID uuid.UUID, domain string) string {
	rows, err := s.Q.RoutingSenderDomains(ctx, wsID)
	if err != nil {
		return ""
	}
	for _, d := range rows {
		if domain != "" && !strings.EqualFold(d.Domain, domain) {
			continue
		}
		var recs []struct {
			ProviderID string `json:"provider_id"`
			Purpose    string `json:"purpose"`
			Name       string `json:"name"`
		}
		_ = json.Unmarshal(d.DnsRecords, &recs)
		for _, r := range recs {
			if r.Purpose == "dkim" && r.ProviderID == providerID.String() {
				return r.Name
			}
		}
		if domain == "" {
			break
		}
	}
	return ""
}

// cost is the price of one message: metered price, or a plan or server
// price spread over this month's volume.
func (s *Service) cost(ctx context.Context, p store.Provider) RouteTestCost {
	pr := state.PricingOf(p)
	switch pr.Model {
	case "plan", "self_hosted":
		c := RouteTestCost{Basis: pr.Model}
		if pr.Model == "self_hosted" {
			zero := 0.0
			c.PerMessage = &zero
		}
		now := s.now()
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		rows, err := s.Q.ProvDailyTotals(ctx, store.ProvDailyTotalsParams{ProviderIds: []uuid.UUID{p.ID}, Since: start, Until: now})
		if err == nil && len(rows) == 1 && rows[0].Sent > 0 && pr.Monthly > 0 {
			per := pr.Monthly / float64(rows[0].Sent)
			per1000 := per * 1000
			if pr.Model == "plan" {
				c.PerMessage = &per
			}
			c.Per1000 = &per1000
		}
		return c
	}
	per1000 := pr.Per1000
	per := per1000 / 1000
	return RouteTestCost{Basis: "metered", PerMessage: &per, Per1000: &per1000}
}
