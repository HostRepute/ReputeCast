package earn

// What offers, gifts and the cancel page (X17) need from Earn: the
// workspace's provider client, the pages' addresses, and hooks into the
// upgrade page, checkout and the payment events.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

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

// ErrLinkExpired means a personal link no longer verifies.
var ErrLinkExpired = errors.New("earn: the personal link expired")

// CancelTargetOf reads a cancel link (TokenCancel): ErrLinkExpired when it
// doesn't verify, ErrNoSubscription when the reader has no subscription to
// cancel (complimentary and gift access end on their own).
func (s *Service) CancelTargetOf(ctx context.Context, ws *store.Workspace, token string) (*CancelTarget, error) {
	if token == "" || s.Links == nil || len(token) > 200 {
		return nil, ErrLinkExpired
	}
	c, err := s.Links.Verify(token, TokenCancel)
	if err != nil || c.WorkspaceID != ws.ID || c.ContactID == uuid.Nil {
		return nil, ErrLinkExpired
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
