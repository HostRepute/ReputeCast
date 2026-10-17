package earn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// The public Earn pages' view models, rendered by internal/publicpages
// with the publisher's branding: the upgrade page, its welcome page after
// checkout, and the paid parts of the preference center and the
// unsubscribe page. Their built-in words come from this package's
// catalog.

// UpgradeRequest is a visit of the upgrade page.
type UpgradeRequest struct {
	// Token is a personal upgrade link's token (?r=).
	Token string
	// Tier is a hidden tier's slug (/upgrade/{slug}).
	Tier   string
	Period string
	// Canceled is set when Checkout sent the reader back (?canceled=1).
	Canceled bool
	// Preview renders for the editor: As is anonymous, free, paid.
	Preview bool
	As      string
	Page    *UpgradePage
	Lang    i18n.Lang
}

// UpgradeView is the upgrade page.
type UpgradeView struct {
	Headline string
	Intro    string
	TestMode bool
	// Known is the reader of a personal link.
	Known    *KnownReader
	Notice   *PageNotice
	Period   string
	Toggle   bool
	Monthly  string
	Yearly   string
	Saving   string
	Free     *FreeCardView
	Cards    []TierCardView
	TaxNote  string
	Trust    string
	FAQ      []UpgradeFAQ
	FAQTitle string
	// Paid replaces the cards for a reader with paid access.
	Paid *PaidView
	// Unavailable: no payment provider is connected, or nothing to buy.
	Unavailable bool
	TermsURL    string
	Support     string
	Action      string
	Token       string
	Words       map[string]string
}

// KnownReader is the reader of a personal link.
type KnownReader struct {
	Email   string
	NotYou  string
	Reading bool
}

// PageNotice is a notice at the top of the page.
type PageNotice struct {
	Tone  string // info, bad
	Title string
	Text  string
	Retry bool
}

// FreeCardView is the free list's card.
type FreeCardView struct {
	Name      string
	Price     string
	Line      string
	Text      string
	Reading   bool
	Subscribe string
	Button    string
	ReadingIt string
}

// TierCardView is a tier's card.
type TierCardView struct {
	ID        string
	Name      string
	Highlight string
	Amount    string
	Unit      string
	Sub       string
	Trial     string
	Benefits  []string
	Button    string
	Primary   bool
	Note      string
	Period    string
}

// PaidView is the "already a paid member" card.
type PaidView struct {
	Heading string
	Email   string
	NotYou  string
	Tier    string
	Price   string
	Renews  string
	Manage  string
	Button  string
	Note    string
}

func (s *Service) t(l i18n.Lang, key string, args i18n.Args) string { return msgs.M(key).Text(l, args) }

// pageTier is a tier the page can sell: its prices in the page's mode.
type pageTier struct {
	t        store.EarnTier
	short    string
	monthly  *store.EarnPrice
	yearly   *store.EarnPrice
	stripeID map[string]string // period → provider price id
}

// sellable lists the tiers the page shows, in its order, with the prices
// the page's mode can sell.
func (s *Service) sellable(ctx context.Context, ws *store.Workspace, page UpgradePage, mode string, hidden string) ([]pageTier, error) {
	tiers, err := s.Q.EarnTiers(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	prices, err := s.Q.EarnPrices(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	out := []pageTier{}
	for _, t := range orderTiers(tiers, page) {
		show := t.Visibility == VisibilityPublic && !slices.Contains(page.Off, t.ID.String())
		if hidden != "" {
			show = t.Slug == hidden && t.Visibility != VisibilityArchived
		}
		if !show {
			continue
		}
		names, _ := s.Q.EarnListNames(ctx, t.ListIds)
		pt := pageTier{t: t, short: shortTier(t.Name, names), stripeID: map[string]string{}}
		for i := range prices {
			p := prices[i]
			if p.TierID != t.ID || p.Status != "current" {
				continue
			}
			sid := p.StripeLiveID
			if mode == "test" {
				sid = p.StripeTestID
			}
			if mode != "" && sid == "" {
				continue
			}
			pt.stripeID[p.Period] = sid
			if p.Period == "month" {
				pt.monthly = &prices[i]
			} else {
				pt.yearly = &prices[i]
			}
		}
		if pt.monthly != nil || pt.yearly != nil {
			out = append(out, pt)
		}
	}
	return out, nil
}

// readerOf verifies a personal upgrade link.
func (s *Service) readerOf(ctx context.Context, ws *store.Workspace, token string) (store.EarnContactBriefRow, linktoken.Claims, bool) {
	if token == "" || s.Links == nil || len(token) > 200 {
		return store.EarnContactBriefRow{}, linktoken.Claims{}, false
	}
	c, err := s.Links.Verify(token, TokenUpgrade)
	if err != nil || c.WorkspaceID != ws.ID || c.ContactID == uuid.Nil {
		return store.EarnContactBriefRow{}, linktoken.Claims{}, false
	}
	ct, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: c.ContactID, WorkspaceID: ws.ID})
	if err != nil {
		return store.EarnContactBriefRow{}, linktoken.Claims{}, false
	}
	return ct, c, true
}

// access is a contact's paid access with its tier.
type access struct {
	store.PaidSubscription
	TierName string
}

// accessOf is a contact's paid access, the highest tier first.
func (s *Service) accessOf(ctx context.Context, ws *store.Workspace, contact uuid.UUID, mode string) (*access, error) {
	rows, err := s.Q.EarnSubscriptionsOfContact(ctx, store.EarnSubscriptionsOfContactParams{ContactID: contact, WorkspaceID: ws.ID})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		p := r.PaidSubscription
		if p.EndedAt == nil && hasAccess(p.Status, p.AccessPaused) && (mode == "" || p.Mode == mode) {
			return &access{PaidSubscription: p, TierName: r.TierName}, nil
		}
	}
	return nil, nil
}

func hasAccess(status string, paused bool) bool {
	switch status {
	case StatusTrialing, StatusActive, StatusComplimentary, StatusGift:
		return true
	case StatusPastDue:
		return !paused
	}
	return false
}

// Upgrade builds the upgrade page.
func (s *Service) Upgrade(ctx context.Context, ws *store.Workspace, r UpgradeRequest) (UpgradeView, error) {
	l := r.Lang
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return UpgradeView{}, err
	}
	conn, connectedNow := active(conns)
	mode := ""
	if connectedNow {
		mode = conn.Mode
	}
	page := r.Page
	if page == nil {
		p, err := s.upgradePage(ctx, ws)
		if err != nil {
			return UpgradeView{}, err
		}
		page = &p
	}
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return UpgradeView{}, err
	}
	v := UpgradeView{Headline: page.Headline, Intro: page.Intro, TestMode: mode == "test", FAQ: page.FAQ, FAQTitle: s.t(l, "page.questions", nil),
		Trust: s.t(l, "page.trust", i18n.Args{"publisher": ws.Name}), TermsURL: st.TermsUrl, Support: st.SupportEmail,
		Monthly: s.t(l, "page.monthly", nil), Yearly: s.t(l, "page.yearly", nil), Token: r.Token,
		Words: map[string]string{"testRibbon": s.t(l, "page.testRibbon", nil), "testNote": s.t(l, "page.testNote", nil), "terms": s.t(l, "page.terms", nil),
			"back": s.t(l, "page.backToArchive", nil), "notYou": s.t(l, "page.notYou", nil), "tryAgain": s.t(l, "page.tryAgain", nil),
			"mostPopular": s.t(l, "page.mostPopular", nil)}}
	if !connectedNow && !r.Preview {
		v.Unavailable = true
	}
	tiers, err := s.sellable(ctx, ws, *page, mode, r.Tier)
	if err != nil {
		return UpgradeView{}, err
	}
	if r.Preview && !connectedNow {
		if tiers, err = s.sellable(ctx, ws, *page, "", r.Tier); err != nil {
			return UpgradeView{}, err
		}
	}
	cur := st.Currency
	// The reader of a personal link.
	var reader *store.EarnContactBriefRow
	if ct, _, ok := s.readerOf(ctx, ws, r.Token); ok {
		reader = &ct
	}
	if r.Preview && (r.As == "free" || r.As == "paid") {
		reader = s.previewReader(ctx, ws, r.As)
	}
	if reader != nil {
		v.Known = &KnownReader{Email: reader.Email, NotYou: s.pageURL(ctx, ws, "upgrade")}
		if r.As != "paid" {
			acc, err := s.accessOf(ctx, ws, reader.ID, "")
			if err != nil {
				return UpgradeView{}, err
			}
			if acc != nil || (r.Preview && r.As == "paid") {
				v.Paid = s.paidView(ctx, ws, l, *reader, acc)
				return v, nil
			}
		} else if acc, err := s.accessOf(ctx, ws, reader.ID, ""); err == nil {
			v.Paid = s.paidView(ctx, ws, l, *reader, acc)
			return v, nil
		}
	}
	// Monthly · Yearly: shown when a tier has both; the saving reads "2 months free".
	hasMonth, hasYear, saving := false, false, 0
	for _, t := range tiers {
		hasMonth = hasMonth || t.monthly != nil
		hasYear = hasYear || t.yearly != nil
		if t.monthly != nil && t.yearly != nil {
			if n := yearlySaving(t.monthly.AmountCents, t.yearly.AmountCents); n > saving {
				saving = n
			}
		}
	}
	v.Toggle = hasMonth && hasYear
	v.Period = page.DefaultPeriod
	if r.Period == "month" || r.Period == "year" {
		v.Period = r.Period
	}
	if !hasYear {
		v.Period = "month"
	} else if !hasMonth {
		v.Period = "year"
	}
	if saving > 0 {
		v.Saving = s.t(l, "page.saving", i18n.Args{"count": saving})
	}
	if r.Canceled {
		v.Notice = &PageNotice{Tone: "info", Title: s.t(l, "page.canceledTitle", nil), Text: s.t(l, "page.canceledText", nil)}
	}
	if id, err := uuid.Parse(page.FreeList); err == nil {
		names, _ := s.Q.EarnListNames(ctx, []uuid.UUID{id})
		if len(names) > 0 {
			fc := &FreeCardView{Name: s.t(l, "page.free", nil), Price: money(0, cur, true), Line: page.FreeLine, Text: page.FreeText,
				Subscribe: s.pageURL(ctx, ws, "subscribe"), Button: s.t(l, "page.subscribeFree", nil), ReadingIt: s.t(l, "page.readingIt", nil)}
			if reader != nil {
				var on bool
				_ = s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM list_memberships WHERE list_id = $1 AND contact_id = $2 AND status = 'subscribed')`, id, reader.ID).Scan(&on)
				fc.Reading = on
				if v.Known != nil {
					v.Known.Reading = on
				}
			}
			v.Free = fc
		}
	}
	hadTrial := false
	if reader != nil {
		hadTrial, _ = s.Q.EarnContactHasHadTrial(ctx, reader.ID)
	}
	now := s.now()
	for _, t := range tiers {
		v.Cards = append(v.Cards, s.card(ctx, ws, l, t, *page, v.Period, hadTrial, now))
	}
	if len(v.Cards) > 0 {
		cn := cur
		if cn == "" {
			cn = "usd"
		}
		key := "page.taxExclusive"
		if st.PricesIncludeTax {
			key = "page.taxInclusive"
		}
		v.TaxNote = s.t(l, key, i18n.Args{"currency": currencyName(cn)})
	}
	return v, nil
}

func currencyName(code string) string {
	switch strings.ToLower(code) {
	case "usd":
		return "US dollars"
	case "eur":
		return "euros"
	case "gbp":
		return "pounds sterling"
	case "cad":
		return "Canadian dollars"
	case "aud":
		return "Australian dollars"
	}
	return strings.ToUpper(code)
}

// card words one tier's card for a period: the price before checkout
// opens, the trial, when the first charge and the renewal come.
func (s *Service) card(ctx context.Context, ws *store.Workspace, l i18n.Lang, t pageTier, page UpgradePage, period string, hadTrial bool, now time.Time) TierCardView {
	p := t.yearly
	if period == "month" && t.monthly != nil || p == nil {
		p = t.monthly
		period = "month"
	}
	if p == nil {
		p = t.yearly
		period = "year"
	}
	c := TierCardView{ID: t.t.ID.String(), Name: t.t.Name, Benefits: t.t.Benefits, Period: period, Amount: money(p.AmountCents, p.Currency, true)}
	if c.Benefits == nil {
		c.Benefits = []string{}
	}
	if page.Highlight == t.t.ID.String() {
		c.Highlight = page.HighlightLabel
		c.Primary = true
	}
	if period == "year" {
		c.Unit = s.t(l, "page.aYear", nil)
		c.Sub = s.t(l, "page.perMonthYearly", i18n.Args{"amount": perMonth(p.AmountCents, p.Currency)})
	} else {
		c.Unit = s.t(l, "page.aMonth", nil)
		c.Sub = s.t(l, "page.billedMonthly", nil)
	}
	price := publicPrice(p.AmountCents, p.Currency, period)
	loc := location(ws)
	trial := int(t.t.TrialDays)
	if hadTrial {
		trial = 0
	}
	if trial > 0 {
		ends := now.In(loc).AddDate(0, 0, trial)
		c.Trial = s.t(l, "page.trialDays", i18n.Args{"count": trial})
		c.Button = s.t(l, "page.startTrial", nil)
		c.Note = s.t(l, "page.trialNote", i18n.Args{"date": dayMonth(ends, now, loc), "price": price})
	} else {
		c.Button = page.Buttons[t.t.ID.String()]
		if c.Button == "" {
			c.Button = s.t(l, "page.become", i18n.Args{"tier": strings.ToLower(t.short[:1]) + t.short[1:], "article": article(t.short)})
		}
		renews := now.In(loc).AddDate(1, 0, 0)
		if period == "month" {
			renews = now.In(loc).AddDate(0, 1, 0)
			c.Note = s.t(l, "page.chargedMonthly", i18n.Args{"amount": money(p.AmountCents, p.Currency, true), "date": dayMonth(renews, now, loc)})
		} else {
			c.Note = s.t(l, "page.chargedYearly", i18n.Args{"amount": money(p.AmountCents, p.Currency, true), "date": renews.Format("2 Jan 2006")})
		}
	}
	return c
}

func dayMonth(t, now time.Time, loc *time.Location) string {
	if t.In(loc).Year() == now.In(loc).Year() {
		return t.In(loc).Format("2 Jan")
	}
	return t.In(loc).Format("2 Jan 2006")
}

// previewReader is the sample reader the editor previews as: a free
// reader of the free list, or a paid member.
func (s *Service) previewReader(ctx context.Context, ws *store.Workspace, as string) *store.EarnContactBriefRow {
	var id uuid.UUID
	var err error
	if as == "paid" {
		err = s.Pool.QueryRow(ctx, `SELECT s.contact_id FROM paid_subscriptions s WHERE s.workspace_id = $1 AND s.mode = 'live' AND s.kind = 'stripe'
			AND s.ended_at IS NULL AND s.status = 'active' ORDER BY s.started_at DESC LIMIT 1`, ws.ID).Scan(&id)
	} else {
		err = s.Pool.QueryRow(ctx, `SELECT c.id FROM contacts c WHERE c.workspace_id = $1 AND c.erased_at IS NULL AND c.status = 'active'
			AND NOT EXISTS (SELECT 1 FROM paid_subscriptions s WHERE s.contact_id = c.id AND s.ended_at IS NULL)
			ORDER BY c.created_at LIMIT 1`, ws.ID).Scan(&id)
	}
	if err != nil {
		return nil
	}
	ct, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return nil
	}
	return &ct
}

// paidView is the card a reader with paid access gets instead of the tiers.
func (s *Service) paidView(ctx context.Context, ws *store.Workspace, l i18n.Lang, reader store.EarnContactBriefRow, acc *access) *PaidView {
	v := &PaidView{Email: reader.Email, NotYou: s.pageURL(ctx, ws, "upgrade"), Manage: s.BillingURL(ctx, ws, reader.ID),
		Button: s.t(l, "page.manage", nil), Note: s.t(l, "page.manageNote", nil)}
	if acc == nil {
		v.Heading = s.t(l, "page.youreMember", i18n.Args{"tier": "none"})
		return v
	}
	short := s.tierShortName(ctx, s.Q, ws.ID, acc.TierID)
	v.Heading = s.t(l, "page.youreMember", i18n.Args{"tier": firstNonEmpty(short, "none"), "article": article(short)})
	v.Tier = acc.TierName
	if acc.Period != "" {
		v.Tier = s.t(l, "page.tierPeriod", i18n.Args{"tier": acc.TierName, "period": acc.Period})
	}
	v.Price = paidPriceLine(l, s, acc.PaidSubscription)
	if acc.CurrentPeriodEnd != nil && acc.CancelAt == nil {
		key := "page.renewsMonthly"
		if acc.Period == "year" {
			key = "page.renewsYearly"
		}
		v.Renews = s.t(l, key, i18n.Args{"date": longDate(ws, *acc.CurrentPeriodEnd)})
	} else if end := firstTime(acc.CancelAt, acc.GrantUntil); end != nil {
		v.Renews = s.t(l, "page.endsOn", i18n.Args{"date": longDate(ws, *end)})
	}
	if acc.Kind != "stripe" {
		v.Button, v.Manage = "", ""
	}
	return v
}

// paidPriceLine is "$64 for your first year with FALL26, then $80 a year".
func paidPriceLine(l i18n.Lang, s *Service, sub store.PaidSubscription) string {
	if sub.Kind != "stripe" || sub.AmountCents == 0 {
		return ""
	}
	full := publicPrice(sub.AmountCents, sub.Currency, sub.Period)
	if len(sub.Discount) > 0 {
		var d struct {
			Label    string `json:"label"`
			Amount   int64  `json:"amountCents"`
			Duration string `json:"duration"`
		}
		_ = json.Unmarshal(sub.Discount, &d)
		if d.Amount > 0 && d.Amount < sub.AmountCents {
			key := "page.discountFirst"
			if d.Duration == "forever" {
				key = "page.discountForever"
			}
			return s.t(l, key, i18n.Args{"amount": money(d.Amount, sub.Currency, true), "period": sub.Period, "offer": d.Label, "price": full})
		}
	}
	return full
}

// Visit counts a visit of an Earn page (not the editor's previews).
func (s *Service) Visit(ctx context.Context, ws *store.Workspace, page string) {
	loc := location(ws)
	day := s.now().In(loc)
	d := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	_ = s.Q.EarnPageVisit(ctx, store.EarnPageVisitParams{WorkspaceID: ws.ID, Day: d, Page: page})
}

// CheckoutRequest is a reader's click on a card's button.
type CheckoutRequest struct {
	Tier   string
	Period string
	Token  string
	// Test is a test checkout from the editor, by a team member.
	Test      bool
	StartedBy *authz.Principal
	// ReturnTo is the page Checkout comes back to (the hidden tier's).
	ReturnTo string
}

// ErrUnavailable means checkout can't open: the provider is down or not
// connected.
var ErrUnavailable = errors.New("earn: payments aren't available")

// StartCheckout opens the provider's checkout for a tier and returns its
// address. The reader's email is filled in when known; the trial is offered
// to readers who haven't had one; tax is added when Stripe Tax is on;
// offers apply as promotion codes and a reader can type one.
func (s *Service) StartCheckout(ctx context.Context, ws *store.Workspace, r CheckoutRequest) (string, error) {
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return "", err
	}
	conn, ok := active(conns)
	if r.Test {
		conn, ok = connected(conns, "test")
	}
	if !ok {
		return "", ErrUnavailable
	}
	page, err := s.upgradePage(ctx, ws)
	if err != nil {
		return "", err
	}
	tid, err := uuid.Parse(r.Tier)
	if err != nil {
		return "", ErrUnavailable
	}
	t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: tid, WorkspaceID: ws.ID})
	if err != nil || t.Visibility == VisibilityArchived {
		return "", ErrUnavailable
	}
	tiers, err := s.sellable(ctx, ws, page, conn.Mode, func() string {
		if t.Visibility == VisibilityHidden {
			return t.Slug
		}
		return ""
	}())
	if err != nil {
		return "", err
	}
	var pt *pageTier
	for i := range tiers {
		if tiers[i].t.ID == tid {
			pt = &tiers[i]
		}
	}
	if pt == nil && r.Test {
		// The editor's test checkout may try a hidden tier too.
		if tiers, err = s.sellable(ctx, ws, page, conn.Mode, t.Slug); err == nil && len(tiers) > 0 {
			pt = &tiers[0]
		}
	}
	if pt == nil {
		return "", ErrUnavailable
	}
	period := r.Period
	price := pt.yearly
	if period == "month" && pt.monthly != nil || price == nil {
		price, period = pt.monthly, "month"
	}
	if price == nil {
		price, period = pt.yearly, "year"
	}
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return "", err
	}
	req := payments.CheckoutRequest{PriceID: pt.stripeID[period], AllowCodes: true, Tax: st.TaxAtCheckout && conn.TaxEnabled, TrialEndCancel: true,
		SuccessURL: s.pageURL(ctx, ws, "upgrade/welcome?session={CHECKOUT_SESSION_ID}"),
		CancelURL:  firstNonEmpty(r.ReturnTo, s.pageURL(ctx, ws, "upgrade")) + "?canceled=1"}
	attribution := map[string]any{"kind": "page"}
	var contact *uuid.UUID
	email := ""
	trial := int(t.TrialDays)
	if ct, claims, ok := s.readerOf(ctx, ws, r.Token); ok {
		id := ct.ID
		contact, email = &id, ct.Email
		req.CustomerEmail = ct.Email
		if had, _ := s.Q.EarnContactHasHadTrial(ctx, ct.ID); had {
			trial = 0
		}
		if rows, err := s.Q.EarnSubscriptionsOfContact(ctx, store.EarnSubscriptionsOfContactParams{ContactID: ct.ID, WorkspaceID: ws.ID}); err == nil {
			for _, x := range rows {
				if x.PaidSubscription.Mode == conn.Mode && x.PaidSubscription.StripeCustomerID != "" {
					req.CustomerID, req.CustomerEmail = x.PaidSubscription.StripeCustomerID, ""
					break
				}
			}
		}
		if claims.MessageID != uuid.Nil {
			if a := s.emailAttribution(ctx, claims.MessageID); a != nil {
				attribution = a
			}
		}
	}
	if r.Test {
		attribution = map[string]any{"kind": "test"}
		if r.StartedBy != nil && req.CustomerEmail == "" && req.CustomerID == "" {
			req.CustomerEmail = r.StartedBy.Email
		}
	}
	req.TrialDays = trial
	chkID := newID()
	req.ClientRef = chkID.String()
	req.Metadata = map[string]string{"reputecast_checkout": chkID.String(), "reputecast_tier": t.ID.String()}
	attr, _ := json.Marshal(attribution)
	now := s.now()
	var by *uuid.UUID
	if r.StartedBy != nil {
		by = userID(r.StartedBy)
	}
	pid := price.ID
	if err := s.Q.EarnCheckoutInsert(ctx, store.EarnCheckoutInsertParams{ID: chkID, WorkspaceID: ws.ID, Mode: conn.Mode, TierID: &t.ID, PriceID: &pid, Period: period,
		TrialDays: int32(trial), ContactID: contact, Email: email, StripeSessionID: "", Status: "open", Attribution: attr, StartedBy: by, CreatedAt: now}); err != nil {
		return "", err
	}
	client, err := s.open(conn.PaymentConnection)
	if err != nil {
		return "", ErrUnavailable
	}
	cs, err := client.CreateCheckout(ctx, req)
	if err != nil {
		_, _ = s.Pool.Exec(ctx, `UPDATE earn_checkouts SET status = 'failed' WHERE id = $1`, chkID)
		if errors.Is(err, payments.ErrKeyRejected) {
			_ = s.Q.PayConnectionKeyRejected(ctx, store.PayConnectionKeyRejectedParams{ID: conn.ID, At: now})
		}
		return "", fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if err := s.Q.EarnCheckoutSetSession(ctx, store.EarnCheckoutSetSessionParams{ID: chkID, StripeSessionID: cs.ID, Status: "open"}); err != nil {
		return "", err
	}
	return cs.URL, nil
}

// emailAttribution credits a checkout to the email whose link the reader
// followed: the campaign (or automation) and when.
func (s *Service) emailAttribution(ctx context.Context, messageID uuid.UUID) map[string]any {
	var campaign *uuid.UUID
	var automation *uuid.UUID
	var name string
	err := s.Pool.QueryRow(ctx, `SELECT m.campaign_id, m.automation_id, coalesce(c.name, a.name, '') FROM messages m
		LEFT JOIN campaigns c ON c.id = m.campaign_id LEFT JOIN automations a ON a.id = m.automation_id WHERE m.id = $1`, messageID).Scan(&campaign, &automation, &name)
	if err != nil {
		return nil
	}
	out := map[string]any{"message_id": messageID.String(), "clicked_at": s.now().UTC().Format(time.RFC3339), "label": name}
	switch {
	case campaign != nil:
		out["kind"], out["campaign_id"] = "campaign", campaign.String()
	case automation != nil:
		out["kind"], out["automation_id"] = "automation", automation.String()
	default:
		return nil
	}
	return out
}

// WelcomeView is the welcome page after checkout.
type WelcomeView struct {
	Heading  string
	Line     string
	Included string
	Benefits []string
	Manage   string
	Button   string
	Archive  string
	Past     string
	Note     string
	Pending  bool
}

// Welcome is the welcome page of a completed checkout. When the webhook
// hasn't arrived yet, the checkout is read from the provider and applied
// now, so the page never waits on it.
func (s *Service) Welcome(ctx context.Context, ws *store.Workspace, session string, l i18n.Lang) (WelcomeView, error) {
	chk, err := s.Q.EarnCheckoutBySession(ctx, session)
	if err != nil || chk.WorkspaceID != ws.ID {
		return WelcomeView{}, ErrUnavailable
	}
	if chk.Status != "completed" {
		if err := s.completeFromReturn(ctx, ws, chk); err != nil {
			s.log().WarnContext(ctx, "welcome page: checkout not applied yet", "session", session, "err", err)
		}
		chk, _ = s.Q.EarnCheckoutBySession(ctx, session)
	}
	page, err := s.upgradePage(ctx, ws)
	if err != nil {
		return WelcomeView{}, err
	}
	v := WelcomeView{Included: s.t(l, "welcome.included", nil), Button: s.t(l, "page.manage", nil), Past: s.t(l, "welcome.past", nil),
		Note: s.t(l, "welcome.note", i18n.Args{"tier": "none"}), Archive: s.pageURL(ctx, ws, "archive")}
	first := ""
	if chk.ContactID != nil {
		if ct, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: *chk.ContactID, WorkspaceID: ws.ID}); err == nil {
			first = ct.FirstName
			v.Manage = s.BillingURL(ctx, ws, ct.ID)
		}
	}
	short := ""
	if chk.TierID != nil {
		if t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: *chk.TierID, WorkspaceID: ws.ID}); err == nil {
			names, _ := s.Q.EarnListNames(ctx, t.ListIds)
			short = shortTier(t.Name, names)
			v.Benefits = t.Benefits
			v.Note = s.t(l, "welcome.note", i18n.Args{"tier": firstNonEmpty(short, "none")})
			v.Past = s.t(l, "welcome.past", nil)
		}
	}
	next := s.nextIssue(ctx, ws, chk.TierID, l)
	text, _ := liquid.Render(page.Welcome, map[string]any{"first_name": firstNonEmpty(first, s.t(l, "welcome.there", nil)), "next_issue": next, "tier": short})
	text = strings.TrimSpace(text)
	head, rest, found := strings.Cut(text, ". ")
	if found {
		v.Heading, v.Line = head+".", strings.TrimSpace(rest)
	} else {
		v.Heading = text
	}
	if first == "" {
		v.Heading = strings.Replace(v.Heading, ", "+s.t(l, "welcome.there", nil), "", 1)
	}
	v.Pending = chk.Status != "completed"
	if v.Benefits == nil {
		v.Benefits = []string{}
	}
	return v, nil
}

// nextIssue is when the next paid issue goes out: the next scheduled
// campaign to one of the tier's lists ("Tue 21 Oct").
func (s *Service) nextIssue(ctx context.Context, ws *store.Workspace, tier *uuid.UUID, l i18n.Lang) string {
	if tier == nil {
		return s.t(l, "welcome.soon", nil)
	}
	var at *time.Time
	_ = s.Pool.QueryRow(ctx, `SELECT min((c.schedule->>'at')::timestamptz) FROM campaigns c, earn_tiers t
		WHERE t.id = $1 AND c.workspace_id = t.workspace_id AND c.status IN ('scheduled','approved','draft')
		AND (c.schedule->>'at') IS NOT NULL AND (c.schedule->>'at')::timestamptz > now()
		AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(coalesce(c.audience->'lists', '[]'::jsonb)) x(id) WHERE x.id::uuid = ANY(t.list_ids))`, *tier).Scan(&at)
	if at == nil {
		return s.t(l, "welcome.soon", nil)
	}
	return at.In(location(ws)).Format("Mon 2 Jan")
}

// completeFromReturn applies a checkout the reader came back from before
// its webhook did: the session and its subscription, read from the
// provider. The events that follow find everything in place.
func (s *Service) completeFromReturn(ctx context.Context, ws *store.Workspace, chk store.EarnCheckout) error {
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return err
	}
	conn, ok := connected(conns, chk.Mode)
	if !ok {
		return ErrUnavailable
	}
	client, err := s.open(conn.PaymentConnection)
	if err != nil {
		return err
	}
	cs, err := client.GetCheckout(ctx, chk.StripeSessionID)
	if err != nil {
		return err
	}
	if cs.Status != "complete" || cs.SubscriptionID == "" {
		return nil
	}
	body, _ := json.Marshal(map[string]any{"id": "return_" + cs.ID, "object": "event", "type": "checkout.session.completed", "created": s.now().Unix(),
		"livemode": conn.Mode == "live", "data": map[string]any{"object": map[string]any{"id": cs.ID, "object": "checkout.session", "status": cs.Status,
			"customer": cs.CustomerID, "customer_details": map[string]any{"email": cs.CustomerEmail}, "subscription": cs.SubscriptionID,
			"client_reference_id": cs.ClientRef, "livemode": cs.Livemode}}})
	v, _ := payments.ParserFor(conn.Provider)
	ev, err := v.ParseEvent(body)
	if err != nil {
		return err
	}
	pe := store.PaymentEvent{ID: newID(), WorkspaceID: ws.ID, Mode: conn.Mode, Type: ev.Type}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		out, err := s.applyCheckout(ctx, tx, ws, conn.PaymentConnection, pe, ev)
		if err != nil {
			return err
		}
		_ = out
		return nil
	})
}

// Portal returns the billing portal of a personal link's paid member;
// cancel opens it on the cancel step of their subscription.
func (s *Service) Portal(ctx context.Context, ws *store.Workspace, token string, kind linktoken.Kind) (string, error) {
	c, err := s.Links.Verify(token, kind)
	if err != nil || c.WorkspaceID != ws.ID || c.ContactID == uuid.Nil {
		return "", ErrUnavailable
	}
	acc, err := s.accessOf(ctx, ws, c.ContactID, "")
	if err != nil {
		return "", err
	}
	if acc == nil || acc.Kind != "stripe" || acc.StripeCustomerID == "" {
		return "", ErrNoSubscription
	}
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return "", err
	}
	conn, ok := connected(conns, acc.Mode)
	if !ok {
		return "", ErrUnavailable
	}
	client, err := s.open(conn.PaymentConnection)
	if err != nil {
		return "", ErrUnavailable
	}
	flow := payments.PortalFlow{ReturnURL: s.pageURL(ctx, ws, "preferences")}
	if kind == TokenCancel {
		flow.Cancel, flow.SubscriptionID = true, acc.StripeSubscriptionID
	}
	u, err := client.PortalSession(ctx, acc.StripeCustomerID, flow)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	return u, nil
}

// ErrNoSubscription means the link's reader has no paid subscription to manage.
var ErrNoSubscription = errors.New("earn: no paid subscription")

// CancelURL is a paid member's link to cancel (the cancel page, X17;
// until then the portal's cancel step).
func (s *Service) CancelURL(ctx context.Context, ws *store.Workspace, contact uuid.UUID) string {
	tok := s.Links.Sign(linktoken.Claims{Kind: TokenCancel, WorkspaceID: ws.ID, ContactID: contact})
	return s.pageURL(ctx, ws, "cancel/"+tok)
}

var _ = url.QueryEscape
