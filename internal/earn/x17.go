package earn

// What offers, gifts and the cancel page (X17) need from Earn: the
// workspace's provider client, the pages' addresses, and hooks into the
// upgrade page, checkout and the payment events.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// OfferCodeRedeemer, when the Offers hook has it, marks a single-use offer
// code (a referral reward's) used once a subscription takes it.
type OfferCodeRedeemer interface {
	CodeRedeemed(ctx context.Context, tx pgx.Tx, ws *store.Workspace, code string, at time.Time) error
}

// OfferHook applies offers on the upgrade page and at checkout.
type OfferHook interface {
	// UpgradeOffer shows an offer link's offer (?offer=FALL26) on the
	// upgrade page: its line and the price of the tier and period it
	// applies to, or that it has ended and the normal prices.
	UpgradeOffer(ctx context.Context, ws *store.Workspace, mode, code string, v *UpgradeView) error
	// CheckoutCode is the provider's promotion code to apply for an offer
	// link, when the offer is live and applies to the tier and period; ""
	// otherwise.
	CheckoutCode(ctx context.Context, ws *store.Workspace, mode, code string, tier uuid.UUID, period string) string
	// PeriodFor is the period an offer link opens the page on (the one
	// the offer applies to); "" for the page's default.
	PeriodFor(ctx context.Context, ws *store.Workspace, code string) string
}

// OneTimeHook takes the payment events of one-time payments: gifts.
type OneTimeHook interface {
	// CheckoutCompleted handles a paid one-time checkout; handled is false
	// when it isn't one of the hook's.
	CheckoutCompleted(ctx context.Context, tx pgx.Tx, ws *store.Workspace, mode string, cs payments.CheckoutSession, at time.Time) (handled bool, detail map[string]any, err error)
	// Refunded handles the refund of a one-time payment.
	Refunded(ctx context.Context, tx pgx.Tx, ws *store.Workspace, mode string, ch payments.Charge, at time.Time) (handled bool, detail map[string]any, err error)
}

// OfferBanner is an offer link's line on the upgrade page.
type OfferBanner struct {
	// Label is the offer's name in bold ("FALL26:").
	Label string
	Text  string
}

// ClientFor opens the provider client of the workspace's connection in
// mode (live or test).
func (s *Service) ClientFor(ctx context.Context, wsID uuid.UUID, mode string) (payments.Client, store.PaymentConnection, error) {
	conns, err := s.connections(ctx, wsID)
	if err != nil {
		return nil, store.PaymentConnection{}, err
	}
	c, ok := connected(conns, mode)
	if !ok {
		return nil, store.PaymentConnection{}, ErrUnavailable
	}
	client, err := s.open(c.PaymentConnection)
	if err != nil {
		return nil, store.PaymentConnection{}, err
	}
	return client, c.PaymentConnection, nil
}

// ActiveMode is the mode the workspace sells in: live once the live key is
// connected, else test; "" without a connection.
func (s *Service) ActiveMode(ctx context.Context, wsID uuid.UUID) string {
	conns, err := s.connections(ctx, wsID)
	if err != nil {
		return ""
	}
	if c, ok := active(conns); ok {
		return c.Mode
	}
	return ""
}

// PageURL is an Earn page's address: on the workspace's own domain when it
// has one ("https://news.acme.io/gift").
func (s *Service) PageURL(ctx context.Context, ws *store.Workspace, path string) string {
	return s.pageURL(ctx, ws, path)
}

// PageHost is PageURL without its scheme ("news.acme.io/gift").
func (s *Service) PageHost(ctx context.Context, ws *store.Workspace, path string) string {
	return s.pageHost(ctx, ws, path)
}

// Money is an amount as the Earn pages write it: "$64", "$4.50".
func Money(cents int64, currency string) string { return money(cents, currency, true) }

// MoneyCents is an amount with its cents: "$4.00".
func MoneyCents(cents int64, currency string) string { return money(cents, currency, false) }

// PageSettings is what every Earn page shows: the terms of sale's
// address, the support email, the currency, and whether tax is added at
// checkout.
type PageSettings struct {
	TermsURL, Support, Currency string
	Tax                         bool
}

// PageSettingsOf reads the Earn pages' settings.
func (s *Service) PageSettingsOf(ctx context.Context, wsID uuid.UUID) PageSettings {
	st, err := s.settingsRow(ctx, s.Q, wsID)
	if err != nil {
		return PageSettings{Currency: "usd"}
	}
	cur := st.Currency
	if cur == "" {
		cur = "usd"
	}
	return PageSettings{TermsURL: st.TermsUrl, Support: st.SupportEmail, Currency: cur, Tax: st.TaxAtCheckout}
}

// TierShortName is a tier's name without its list's ("Insider" for
// Weekly Digest Insider on Weekly Digest).
func (s *Service) TierShortName(ctx context.Context, wsID uuid.UUID, tier uuid.UUID) string {
	return s.tierShortName(ctx, s.Q, wsID, &tier)
}

// ShortTierName shortens a tier's name with its lists' names in hand.
func ShortTierName(name string, lists []string) string { return shortTier(name, lists) }

// CancelTarget is whom a cancel link is for: the reader and the paid
// subscription the cancel page is about, with its tier's names and the
// tier's lists (what they keep getting for free).
type CancelTarget struct {
	Contact store.EarnContactBriefRow
	Sub     store.PaidSubscription
	Tier    string
	Short   string
	Lists   []string
}

// ErrLinkExpired means a personal link no longer verifies: its signature
// doesn't match (an earlier instance key) or its reader is gone.
var ErrLinkExpired = errors.New("earn: the personal link expired")

// ErrLinkInvalid means a personal link isn't one: malformed, another kind of
// link, or another workspace's.
var ErrLinkInvalid = errors.New("earn: not a personal link")

// CancelTargetOf reads a cancel link (TokenCancel): ErrLinkInvalid when it
// isn't one, ErrLinkExpired when it no longer verifies, ErrNoSubscription
// when the reader has no subscription to cancel (complimentary and gift
// access end on their own).
func (s *Service) CancelTargetOf(ctx context.Context, ws *store.Workspace, token string) (*CancelTarget, error) {
	if token == "" || s.Links == nil || len(token) > 200 {
		return nil, ErrLinkInvalid
	}
	c, err := s.Links.Verify(token, TokenCancel)
	switch {
	case errors.Is(err, linktoken.ErrSignature):
		return nil, ErrLinkExpired
	case err != nil || c.WorkspaceID != ws.ID || c.ContactID == uuid.Nil:
		return nil, ErrLinkInvalid
	}
	ct, err := s.Q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: c.ContactID, WorkspaceID: ws.ID})
	if err != nil {
		return nil, ErrLinkExpired
	}
	acc, err := s.accessOf(ctx, ws, c.ContactID, "")
	if err != nil {
		return nil, err
	}
	if acc == nil || acc.Kind != "stripe" || acc.StripeSubscriptionID == "" {
		return nil, ErrNoSubscription
	}
	out := &CancelTarget{Contact: ct, Sub: acc.PaidSubscription, Tier: acc.TierName, Short: acc.TierName}
	if acc.TierID != nil {
		if t, err := s.Q.EarnTier(ctx, store.EarnTierParams{ID: *acc.TierID, WorkspaceID: ws.ID}); err == nil {
			out.Lists, _ = s.Q.EarnListNames(ctx, t.ListIds)
			out.Short = shortTier(t.Name, out.Lists)
		}
	}
	return out, nil
}

// CancelPortal opens the provider portal on the subscription's cancel
// step; the reader comes back to returnURL.
func (s *Service) CancelPortal(ctx context.Context, ws *store.Workspace, sub store.PaidSubscription, returnURL string) (string, error) {
	client, _, err := s.ClientFor(ctx, ws.ID, sub.Mode)
	if err != nil {
		return "", ErrUnavailable
	}
	u, err := client.PortalSession(ctx, sub.StripeCustomerID, payments.PortalFlow{Cancel: true, SubscriptionID: sub.StripeSubscriptionID, ReturnURL: returnURL})
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	return u, nil
}

// PaidAccessOf is the paid access a contact has now, in mode ("" for
// either): the subscription and its tier's name, or nil without any.
func (s *Service) PaidAccessOf(ctx context.Context, ws *store.Workspace, contact uuid.UUID, mode string) (*store.PaidSubscription, string, error) {
	acc, err := s.accessOf(ctx, ws, contact, mode)
	if err != nil || acc == nil {
		return nil, "", err
	}
	p := acc.PaidSubscription
	return &p, acc.TierName, nil
}

// Timeline adds an entry to a paid member's timeline in the caller's
// transaction (with its webhook event, when the kind has one).
func (s *Service) Timeline(ctx context.Context, q *store.Queries, wsID, contact uuid.UUID, sub *uuid.UUID, mode, kind string, at time.Time, actor *uuid.UUID,
	data map[string]any) error {
	return s.timeline(ctx, q, wsID, contact, sub, mode, kind, at, nil, "", nil, "", actor, data)
}

// MemberEmailResult is what sending a member email did: sent, or not_sent
// and why (not_published, no_route, suppressed, failed, no_mailer,
// no_template).
type MemberEmailResult struct {
	Status    string
	Reason    string
	MessageID *uuid.UUID
	Template  string
	Version   int
}

// Sent reports whether the email went out.
func (r MemberEmailResult) Sent() bool { return r.Status == "sent" }

// SendMemberEmailTo sends one of the member emails (EmailGiftPurchased…)
// to an address that needn't be a contact: a gift's buyer, or its
// recipient before redeeming. key makes a retry send nothing twice.
func (s *Service) SendMemberEmailTo(ctx context.Context, ws *store.Workspace, email, to, toName string, data map[string]any, key string) MemberEmailResult {
	return s.SendMemberEmailIn(ctx, ws, email, to, toName, data, key, nil)
}

// SendMemberEmailIn is SendMemberEmailTo with values written in the
// email's language: texts gives them in the language of the version the
// send picks (data has them in English).
func (s *Service) SendMemberEmailIn(ctx context.Context, ws *store.Workspace, email, to, toName string, data map[string]any, key string,
	texts func(l i18n.Lang) map[string]any) MemberEmailResult {
	if s.Mailer == nil {
		return MemberEmailResult{Status: "not_sent", Reason: "no_mailer"}
	}
	tpl, err := s.memberTemplate(ctx, ws, email)
	if err != nil {
		return MemberEmailResult{Status: "not_sent", Reason: "no_template"}
	}
	out := MemberEmailResult{Status: "sent", Template: tpl.Key, Version: tpl.LiveVersion}
	res, err := s.Mailer.Send(ctx, ws, MemberMessage{Template: tpl.Key, To: to, ToName: toName, Data: data, Key: key, Localize: s.localizer(ctx, ws, texts)})
	switch {
	case errors.Is(err, ErrNotPublished):
		out.Status, out.Reason = "not_sent", "not_published"
	case errors.Is(err, ErrNoRoute):
		out.Status, out.Reason = "not_sent", "no_route"
	case errors.Is(err, ErrSuppressed):
		out.Status, out.Reason = "not_sent", "suppressed"
	case err != nil:
		out.Status, out.Reason = "not_sent", "failed"
		s.log().WarnContext(ctx, "member email", "email", email, "err", err)
	case res.MessageID != uuid.Nil:
		id := res.MessageID
		out.MessageID = &id
	}
	return out
}

// DateText is a date as the member emails write it ("24 Dec", with the
// year when it isn't this year's), in the workspace's time zone.
func (s *Service) DateText(ws *store.Workspace, t time.Time) string { return s.dateText(ws, t) }

// DateTextIn is DateText in a language ("24 thg 12").
func (s *Service) DateTextIn(l i18n.Lang, ws *store.Workspace, t time.Time) string {
	return s.dateTextIn(l, ws, t)
}

// DateIn is a day with its year in a language: "7 Oct 2027", "7 thg 10, 2027".
func DateIn(l i18n.Lang, t time.Time) string { return dateIn(l, t) }

// TimeIn is a time of day in a language's region format: "08:00".
func TimeIn(l i18n.Lang, t time.Time) string {
	if standard(l) {
		return t.Format("15:04")
	}
	return l.Region.Time(t, "")
}

// DayMonthIn is a day without its year in a language: "24 Dec", "24 thg 12".
func DayMonthIn(l i18n.Lang, t time.Time) string { return dayMonthIn(l, t) }

// WorkspaceLang is the language a workspace writes its emails and pages
// in by default, as their readers read it.
func WorkspaceLang(ctx context.Context, q *store.Queries, wsID uuid.UUID) i18n.Lang {
	return workspaceLang(ctx, q, wsID)
}

// EnglishText reports whether one of the Earn catalog's messages is
// written in English for a language (it isn't translated yet).
func EnglishText(l i18n.Lang, key string) bool { return englishText(l, key) }

// FailureShort is a payment failure in a few words in a language.
func FailureShort(l i18n.Lang, reason string) string { return failureShort(l, reason) }

// WelcomeLine is the upgrade page's welcome text after its first sentence
// ("Your first members-only deep dive arrives Tue 21 Oct."), for a new
// paid member of the tier: the gift's redeem page says it too, in the
// page's language.
func (s *Service) WelcomeLine(ctx context.Context, ws *store.Workspace, tier *uuid.UUID, first string, l i18n.Lang) string {
	page, err := s.upgradePageIn(ctx, ws, l)
	if err != nil {
		return ""
	}
	short := ""
	if tier != nil {
		short = s.tierShortName(ctx, s.Q, ws.ID, tier)
	}
	text, err := liquid.Render(page.Welcome, map[string]any{"first_name": firstNonEmpty(first, s.t(l, "welcome.there", nil)),
		"next_issue": s.nextIssue(ctx, ws, tier, l), "tier": short})
	if err != nil {
		return ""
	}
	_, rest, found := strings.Cut(strings.TrimSpace(text), ". ")
	if !found {
		return ""
	}
	return strings.TrimSpace(rest)
}
