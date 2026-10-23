package earn

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/store"
)

// TokenTestCheckout signs a team member's test checkout link (the editor's
// "Preview with test checkout"): the upgrade page in test mode, valid for a
// day. Its Link field holds the hour it expires.
const TokenTestCheckout linktoken.Kind = 163

// testTokenHours is how long a test checkout link works.
const testTokenHours = 24

// TestCheckoutURL is a test checkout link of the upgrade page, when a test
// key is connected ("" otherwise).
func (s *Service) TestCheckoutURL(ctx context.Context, ws *store.Workspace) string {
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return ""
	}
	if _, ok := connected(conns, "test"); !ok || s.Links == nil {
		return ""
	}
	until := uint32(s.now().Add(testTokenHours*time.Hour).Unix() / 3600)
	tok := s.Links.Sign(linktoken.Claims{Kind: TokenTestCheckout, WorkspaceID: ws.ID, Link: until})
	return s.pageURL(ctx, ws, "upgrade?test="+tok)
}

// TestMode reports whether a test checkout link is valid for the workspace.
func (s *Service) TestMode(ws *store.Workspace, token string) bool {
	if token == "" || s.Links == nil || len(token) > 200 {
		return false
	}
	c, err := s.Links.Verify(token, TokenTestCheckout)
	if err != nil || c.WorkspaceID != ws.ID {
		return false
	}
	return int64(c.Link) >= s.now().Unix()/3600
}

// PreferencesPaid is the "Your paid subscription" section of the
// preference center: a paid member's tier and billing, complimentary
// access or a gift, or, for a free reader while the upgrade page is
// public, what they could get.
type PreferencesPaid struct {
	// Kind: stripe, complimentary, gift or free.
	Kind    string
	Title   string
	Heading string
	Price   string
	Line    string
	// Manage opens the billing portal, Cancel the cancel step, Upgrade the
	// upgrade page (free readers, and gifts to keep going).
	Manage       string
	ManageLabel  string
	Cancel       string
	CancelLabel  string
	Upgrade      string
	UpgradeLabel string
	Test         bool
}

// PreferencesPaid builds the section for a contact (nil: none to show).
func (s *Service) PreferencesPaid(ctx context.Context, ws *store.Workspace, contact uuid.UUID, l i18n.Lang) (*PreferencesPaid, error) {
	acc, err := s.accessOf(ctx, ws, contact, "")
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return s.freeReaderOffer(ctx, ws, contact, l)
	}
	short := s.tierShortName(ctx, s.Q, ws.ID, acc.TierID)
	out := &PreferencesPaid{Kind: acc.Kind, Title: s.t(l, "prefs.title", nil), Test: acc.Mode == "test"}
	loc := location(ws)
	switch acc.Kind {
	case "complimentary":
		out.Heading = s.t(l, "prefs.compHeading", i18n.Args{"tier": short, "publisher": s.publisher(ctx, ws), "until": dateOrNone(l, acc.GrantUntil, loc)})
		out.Line = s.t(l, "prefs.compLine", i18n.Args{"tier": short})
	case "gift":
		var d struct {
			From   string `json:"from"`
			Months int    `json:"months"`
		}
		_ = jsonUnmarshal(acc.SourceDetail, &d)
		out.Heading = s.t(l, "prefs.giftHeading", i18n.Args{"from": firstNonEmpty(d.From, "none"), "until": dateOrNone(l, acc.CurrentPeriodEnd, loc)})
		out.Line = s.t(l, "prefs.giftLine", i18n.Args{"tier": acc.TierName, "months": d.Months})
		out.Upgrade, out.UpgradeLabel = s.UpgradeURL(ctx, ws, contact, uuid.Nil), s.t(l, "prefs.keepAfterGift", i18n.Args{"tier": short})
	default:
		out.Heading = acc.TierName
		if acc.Period != "" {
			out.Heading = s.t(l, "page.tierPeriod", i18n.Args{"tier": acc.TierName, "period": acc.Period})
		}
		out.Price = paidPriceLine(l, s, acc.PaidSubscription)
		switch {
		case acc.CancelAt != nil:
			out.Line = s.t(l, "prefs.endsOn", i18n.Args{"date": longDateIn(l, ws, *acc.CancelAt)})
		case acc.Status == StatusTrialing && acc.TrialEnd != nil:
			out.Line = s.t(l, "prefs.trialEnds", i18n.Args{"date": longDateIn(l, ws, *acc.TrialEnd)})
		case acc.Status == StatusPastDue:
			out.Line = s.t(l, "prefs.pastDue", nil)
		case acc.CurrentPeriodEnd != nil && acc.Period == "year":
			out.Line = s.t(l, "prefs.renewsYearly", i18n.Args{"date": longDateIn(l, ws, *acc.CurrentPeriodEnd)})
		case acc.CurrentPeriodEnd != nil:
			out.Line = s.t(l, "page.renewsMonthly", i18n.Args{"date": longDateIn(l, ws, *acc.CurrentPeriodEnd)})
		}
		out.Manage, out.ManageLabel = s.BillingURL(ctx, ws, contact), s.t(l, "prefs.manageBilling", nil)
		if acc.CancelAt == nil {
			out.Cancel, out.CancelLabel = s.CancelURL(ctx, ws, contact), s.t(l, "prefs.cancel", nil)
		}
	}
	return out, nil
}

// freeReaderOffer is "Get more with Insider · from $8 a month" for a free
// reader of a tier's list, while the upgrade page sells something.
func (s *Service) freeReaderOffer(ctx context.Context, ws *store.Workspace, contact uuid.UUID, l i18n.Lang) (*PreferencesPaid, error) {
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	conn, ok := active(conns)
	if !ok {
		return nil, nil
	}
	page, err := s.upgradePage(ctx, ws)
	if err != nil {
		return nil, err
	}
	tiers, err := s.sellable(ctx, ws, page, conn.Mode, "")
	if err != nil || len(tiers) == 0 {
		return nil, err
	}
	t := tiers[0]
	for _, x := range tiers {
		if x.t.ID.String() == page.Highlight {
			t = x
		}
	}
	from := t.monthly
	monthly := int64(0)
	cur := ""
	if from != nil {
		monthly, cur = from.AmountCents, from.Currency
	} else if t.yearly != nil {
		monthly, cur = t.yearly.AmountCents/12, t.yearly.Currency
	}
	return &PreferencesPaid{Kind: "free", Title: s.t(l, "prefs.title", nil),
		Heading: s.t(l, "prefs.freeHeading", i18n.Args{"tier": t.short, "price": publicPriceIn(l, monthly, cur, "month")}),
		Line:    t.t.Description, Upgrade: s.UpgradeURL(ctx, ws, contact, uuid.Nil), UpgradeLabel: s.t(l, "prefs.seeTier", i18n.Args{"tier": t.short})}, nil
}

func dateOrNone(l i18n.Lang, t *time.Time, loc *time.Location) string {
	if t == nil {
		return "none"
	}
	return dateIn(l, t.In(loc))
}

// UnsubscribePaid is the paid member's note on the unsubscribe page:
// unsubscribing doesn't cancel the subscription.
type UnsubscribePaid struct {
	Text        string
	Cancel      string
	CancelLabel string
}

// UnsubscribePaid builds the note for a contact with paid access (nil otherwise).
func (s *Service) UnsubscribePaid(ctx context.Context, ws *store.Workspace, contact uuid.UUID, l i18n.Lang) (*UnsubscribePaid, error) {
	acc, err := s.accessOf(ctx, ws, contact, "")
	if err != nil || acc == nil {
		return nil, err
	}
	short := s.tierShortName(ctx, s.Q, ws.ID, acc.TierID)
	out := &UnsubscribePaid{Text: s.t(l, "unsub.paid", i18n.Args{"tier": firstNonEmpty(short, "none"), "article": article(short)})}
	if acc.Kind == "stripe" && acc.CancelAt == nil {
		out.Cancel, out.CancelLabel = s.CancelURL(ctx, ws, contact), s.t(l, "prefs.cancel", nil)
	}
	return out, nil
}

// EarnChecklist is the launch checklist on Earn's Overview while a
// workspace sets up paid subscriptions (EarnStates).
type EarnChecklist struct {
	Steps []EarnChecklistStep `json:"steps"`
	Done  int                 `json:"done"`
	// Launched: every step is done, or readers pay through the live key
	// (the Overview shows revenue instead).
	Launched bool `json:"launched"`
	// TestCheckouts are the test paid members, left out of every number.
	TestCheckouts []EarnMemberRow `json:"testCheckouts"`
	TestCheckout  string          `json:"testCheckoutUrl,omitempty"`
	UpgradeHost   string          `json:"upgradeHost"`
	CanManage     bool            `json:"canManage"`
	CanEdit       bool            `json:"canEdit"`
}

// EarnChecklistStep is one step of the launch checklist.
type EarnChecklistStep struct {
	Key  string `json:"key" enum:"connect,tier,emails,page,test,live"`
	Done bool   `json:"done"`
	// Detail says what was done ("Test key · Acme Media · 17 Sep").
	Detail string `json:"detail,omitempty"`
	// Blocked says why the step can't be done yet (no Transactional route).
	Blocked string `json:"blocked,omitempty" enum:"no_route,"`
	// Facts for the detail line.
	Mode      string     `json:"mode,omitempty"`
	Account   string     `json:"account,omitempty"`
	At        *time.Time `json:"at,omitempty"`
	Count     int64      `json:"count,omitempty"`
	Total     int64      `json:"total,omitempty"`
	Name      string     `json:"name,omitempty"`
	PriceLine string     `json:"priceLine,omitempty"`
}

// Checklist is the launch checklist's state.
func (s *Service) Checklist(ctx context.Context, ws *store.Workspace, p *authz.Principal) (EarnChecklist, error) {
	out := EarnChecklist{Steps: []EarnChecklistStep{}, TestCheckouts: []EarnMemberRow{}, UpgradeHost: s.pageHost(ctx, ws, "upgrade"),
		CanManage: p.Can(authz.PaymentsManage), CanEdit: p.Can(authz.EarnEdit)}
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	live, hasLive := connected(conns, "live")
	test, hasTest := connected(conns, "test")
	step := EarnChecklistStep{Key: "connect"}
	switch {
	case hasLive:
		step.Done, step.Mode, step.Account, step.At = true, "live", live.AccountName, live.ConnectedAt
	case hasTest:
		step.Done, step.Mode, step.Account, step.At = true, "test", test.AccountName, test.ConnectedAt
	}
	out.Steps = append(out.Steps, step)
	// A tier with a price.
	tiers, err := s.Q.EarnTiers(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	prices, err := s.Q.EarnPrices(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	step = EarnChecklistStep{Key: "tier"}
	for _, t := range tiers {
		var parts []string
		for _, pr := range prices {
			if pr.TierID == t.ID && pr.Status == "current" {
				parts = append(parts, priceTextIn(lang(ctx), pr.AmountCents, pr.Currency, pr.Period))
			}
		}
		if len(parts) > 0 {
			step.Done, step.Name, step.PriceLine = true, t.Name, strings.Join(parts, ", ")
			break
		}
	}
	out.Steps = append(out.Steps, step)
	// The member emails, published.
	step = EarnChecklistStep{Key: "emails", Total: int64(len(MemberEmailOrder))}
	if s.Mailer != nil {
		if rows, err := s.Q.EarnMemberEmails(ctx, ws.ID); err == nil {
			var ids []uuid.UUID
			for _, r := range rows {
				if r.TemplateID != nil {
					ids = append(ids, *r.TemplateID)
				}
			}
			if states, err := s.Mailer.Templates(ctx, ws, ids); err == nil {
				for _, st := range states {
					if st.LiveVersion > 0 && !st.Deleted {
						step.Count++
					}
				}
			}
		}
		if r, err := s.Mailer.Route(ctx, ws); err == nil && r.Primary == "" {
			step.Blocked = "no_route"
		}
	}
	step.Done = step.Count == step.Total && step.Blocked == ""
	out.Steps = append(out.Steps, step)
	// The upgrade page, reviewed.
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	step = EarnChecklistStep{Key: "page"}
	if st.UpgradePublishedAt != nil {
		step.Done, step.At = true, st.UpgradePublishedAt
		if st.UpgradePublishedBy != nil {
			step.Name, _ = s.Q.EarnUserName(ctx, *st.UpgradePublishedBy)
		}
	}
	out.Steps = append(out.Steps, step)
	// A test checkout.
	step = EarnChecklistStep{Key: "test"}
	_ = s.Pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE paid_cents > 0) FROM paid_subscriptions WHERE workspace_id = $1 AND mode = 'test' AND kind = 'stripe'`, ws.ID).Scan(&step.Total, &step.Count)
	step.Done = step.Total > 0
	out.Steps = append(out.Steps, step)
	// The live key.
	step = EarnChecklistStep{Key: "live", Done: hasLive}
	if hasLive {
		step.Account, step.At = live.AccountName, live.ConnectedAt
	}
	out.Steps = append(out.Steps, step)
	for _, x := range out.Steps {
		if x.Done {
			out.Done++
		}
	}
	// Earn is live once every step is done, or once readers pay through the
	// live key (a step left undone later, such as a member email added since,
	// doesn't take the revenue overview away).
	out.Launched = out.Done == len(out.Steps)
	if !out.Launched && hasLive {
		_ = s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM paid_subscriptions WHERE workspace_id = $1 AND mode = 'live' AND kind = 'stripe' AND status <> 'incomplete')`, ws.ID).Scan(&out.Launched)
	}
	if hasTest {
		out.TestCheckout = s.TestCheckoutURL(ctx, ws)
		list, err := s.Members(ctx, ws, p, EarnMemberFilter{Status: FilterWithAccess, Test: true, PageSize: 20, Page: 1})
		if err == nil {
			for _, r := range list.Items {
				if r.Test {
					out.TestCheckouts = append(out.TestCheckouts, r)
				}
			}
		}
	}
	return out, nil
}

// EarnPreviewInput renders the upgrade page with unsaved settings.
type EarnPreviewInput struct {
	Page UpgradePage `json:"page"`
	// As is who the preview is for: an anonymous visitor, a free reader
	// from an email, or a paid member.
	As string `json:"as" enum:"visitor,free,paid,offer" default:"visitor"`
}

// EarnPreview is the rendered page.
type EarnPreview struct {
	HTML string `json:"html"`
}

// PreviewUpgrade renders the upgrade page as readers would see it with the
// editor's settings, without saving them. Its links and buttons work in
// no way (the page is inert).
func (s *Service) PreviewUpgrade(ctx context.Context, ws *store.Workspace, in EarnPreviewInput) (EarnPreview, error) {
	if s.Render == nil {
		return EarnPreview{}, nil
	}
	as := in.As
	if as == "visitor" || as == "offer" {
		as = ""
	}
	page := in.Page
	if page.DefaultPeriod == "" {
		page.DefaultPeriod = "year"
	}
	// The editor previews the page as its readers get it by default: in the
	// workspace's default language.
	v, err := s.Upgrade(ctx, ws, UpgradeRequest{Preview: true, As: as, Page: &page, Lang: workspaceLang(ctx, s.Q, ws.ID)})
	if err != nil {
		return EarnPreview{}, err
	}
	html, err := s.Render(ctx, ws, "upgrade", v)
	return EarnPreview{HTML: html}, err
}

// jsonUnmarshal ignores empty documents.
func jsonUnmarshal(b []byte, v any) error {
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, v)
}
