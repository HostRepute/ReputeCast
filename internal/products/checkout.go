package products

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Buying: a Buy button opens a Stripe Checkout session for the product
// (a one-time payment, or the buyer's amount above the minimum). The list
// offer's checkbox sits on the publisher's page, next to the Buy button,
// because Stripe's page can't carry it: its state, the page, the time and
// the IP address travel with the session. Back from Stripe, the thank-you
// page has the download.

// CheckoutStart is a Buy button pressed.
type CheckoutStart struct {
	ProductID uuid.UUID
	// Ticked is the list offer's checkbox.
	Ticked bool
	// Source is where the Buy button was: a page (PageID), the shop or the
	// default product page.
	Source string
	PageID *uuid.UUID
	// PageURL is the page's address (the cancel return); PageLabel how
	// consent names it ("news.acme.io/field-guide").
	PageURL   string
	PageLabel string
	IP        string
	// Email and ContactID are a signed-in member's: the email is filled in.
	Email     string
	ContactID *uuid.UUID
	// ThanksURL is the thank-you page, with {CHECKOUT_SESSION_ID} where the
	// session's id goes.
	ThanksURL string
}

// CheckoutHandoff is what "Taking you to Stripe…" shows before it goes.
type CheckoutHandoff struct {
	URL         string
	ProductName string
	Price       Live
	Currency    string
	Account     string
	Tax         bool
	Test        bool
}

// ErrNotForSale is a product that can't be bought (a draft, archived, or
// Stripe isn't connected).
var ErrNotForSale = errors.New("products: not for sale")

// StartCheckout opens a Checkout session for a published product.
func (s *Service) StartCheckout(ctx context.Context, ws *store.Workspace, in CheckoutStart) (CheckoutHandoff, error) {
	l := lang(ctx)
	p, err := s.Q.ProductGet(ctx, store.ProductGetParams{WorkspaceID: ws.ID, ID: in.ProductID})
	if err != nil {
		return CheckoutHandoff{}, ErrNotForSale
	}
	live, ok := liveOf(p)
	if !ok || p.Status != StatusPublished {
		return CheckoutHandoff{}, ErrNotForSale
	}
	sel, ok, err := s.Earn().Selling(ctx, ws)
	if err != nil {
		return CheckoutHandoff{}, err
	}
	if !ok {
		return CheckoutHandoff{}, ErrNotForSale
	}
	client, err := s.Earn().SellingClient(ctx, ws, sel.ID)
	if err != nil {
		return CheckoutHandoff{}, err
	}
	one, ok := client.(payments.OneTimePayments)
	if !ok {
		return CheckoutHandoff{}, ErrNotForSale
	}
	currency := p.Currency
	if currency == "" {
		currency = sel.Currency
	}
	// The price in this mode: published in live, first bought in test, or
	// the other way round.
	ids := readStripe(p.Stripe)
	m, err := ensurePrice(ctx, client, ids[string(sel.Mode)], p.ID, live, currency, sel.PricesIncludeTax)
	if err != nil {
		return CheckoutHandoff{}, api.Err(503, "stripe_unavailable", word(l, "checkout.failed", nil))
	}
	if prev := ids[string(sel.Mode)]; m.Price != prev.Price || m.Product != prev.Product {
		ids[string(sel.Mode)] = m
		b, _ := json.Marshal(ids)
		if err := s.Q.ProductSetStripe(ctx, store.ProductSetStripeParams{ID: p.ID, Stripe: b}); err != nil {
			return CheckoutHandoff{}, err
		}
	}
	now := s.now()
	consent := checkoutConsent{Offered: live.OfferOn && live.OfferListID != "", Ticked: in.Ticked && live.OfferOn && live.OfferListID != "",
		Text: live.OfferText, ListID: live.OfferListID, Page: in.PageLabel, IP: in.IP, At: &now}
	cb, _ := json.Marshal(consent)
	cid := newID()
	source := in.Source
	if source == "" {
		source = SourcePage
	}
	if err := s.Q.ProductCheckoutInsert(ctx, store.ProductCheckoutInsertParams{ID: cid, WorkspaceID: ws.ID, ProductID: p.ID, Mode: string(sel.Mode),
		ConnectionID: &sel.ID, Email: strings.ToLower(in.Email), ContactID: in.ContactID, Consent: cb, Source: source, SourcePageID: in.PageID,
		SourceUrl: in.PageURL, At: now}); err != nil {
		return CheckoutHandoff{}, err
	}
	cancel := in.PageURL
	if cancel == "" {
		cancel = s.siteLinks(ctx, ws).product(live.Slug)
	}
	cs, err := one.CreatePaymentCheckout(ctx, payments.PaymentCheckoutRequest{PriceID: m.Price, CustomerEmail: in.Email, Tax: sel.TaxAtCheckout,
		SuccessURL: in.ThanksURL, CancelURL: withQuery(cancel, "checkout", "cancelled"), ClientRef: cid.String(),
		Metadata: map[string]string{"reputecast_checkout": cid.String(), "reputecast_product": p.ID.String()}})
	if err != nil {
		_ = s.Q.ProductCheckoutFailed(ctx, cid)
		s.log().WarnContext(ctx, "products: checkout", "err", err)
		return CheckoutHandoff{}, api.Err(503, "stripe_unavailable", word(l, "checkout.failed", nil))
	}
	if err := s.Q.ProductCheckoutSetSession(ctx, store.ProductCheckoutSetSessionParams{ID: cid, SessionID: cs.ID}); err != nil {
		return CheckoutHandoff{}, err
	}
	return CheckoutHandoff{URL: cs.URL, ProductName: live.Name, Price: live, Currency: currency, Account: sel.Account, Tax: sel.TaxAtCheckout,
		Test: sel.Mode == payments.Test}, nil
}

// withQuery adds a query parameter to an address.
func withQuery(raw, key, val string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	q.Set(key, val)
	u.RawQuery = q.Encode()
	return u.String()
}

// ThanksView is the thank-you page after Stripe.
type ThanksView struct {
	Order       store.ProductOrder
	ProductName string
	Files       ProductFilesSummary
	DownloadURL string
	EmailMasked string
	ExpiresAt   time.Time
	Downloads   int32
	// OfferPending: the buyer ticked the list offer and its double opt-in
	// waits for the confirmation click.
	OfferPending bool
	OfferList    string
	Test         bool
}

// Thanks reads the order a returning buyer paid for. Back from Stripe
// before its webhook arrives, the payment is read from Stripe and applied
// now (the webhook then finds the order made).
func (s *Service) Thanks(ctx context.Context, ws *store.Workspace, productID uuid.UUID, sessionID string) (ThanksView, bool, error) {
	if sessionID == "" || len(sessionID) > 200 {
		return ThanksView{}, false, nil
	}
	o, err := s.Q.ProductOrderBySession(ctx, sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		co, cerr := s.Q.ProductCheckoutBySession(ctx, sessionID)
		if cerr != nil || co.WorkspaceID != ws.ID || co.ProductID != productID || co.ConnectionID == nil {
			return ThanksView{}, false, nil
		}
		client, cerr := s.Earn().SellingClient(ctx, ws, *co.ConnectionID)
		if cerr != nil {
			return ThanksView{}, false, cerr
		}
		one, ok := client.(payments.OneTimePayments)
		if !ok {
			return ThanksView{}, false, nil
		}
		pay, cerr := one.GetPayment(ctx, sessionID)
		if cerr != nil || !pay.Paid {
			return ThanksView{}, false, cerr
		}
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			var e error
			o, _, e = s.applyPaid(ctx, tx, ws, co, pay, "checkout.session.completed")
			return e
		})
	}
	if err != nil {
		return ThanksView{}, false, err
	}
	if o.WorkspaceID != ws.ID || o.ProductID != productID {
		return ThanksView{}, false, nil
	}
	return s.thanksView(ctx, ws, o)
}

func (s *Service) thanksView(ctx context.Context, ws *store.Workspace, o store.ProductOrder) (ThanksView, bool, error) {
	links, err := s.Q.ProductOrderLinks(ctx, o.ID)
	if err != nil || len(links) == 0 {
		return ThanksView{}, false, err
	}
	// The purchase's link: the thank-you page never mints a new one.
	link := links[len(links)-1].ProductOrderLink
	for _, l := range links {
		if l.ProductOrderLink.Reason == "purchase" {
			link = l.ProductOrderLink
		}
	}
	files, err := s.Q.DownloadFilesOf(ctx, store.DownloadFilesOfParams{OwnerKind: OwnerProduct, OwnerID: o.ProductID})
	if err != nil {
		return ThanksView{}, false, err
	}
	fs := make([]store.DownloadFile, len(files))
	for i, f := range files {
		fs[i] = f.DownloadFile
	}
	v := ThanksView{Order: o, ProductName: o.ProductName, Files: summary(fs), DownloadURL: s.DownloadURL(ctx, ws, o.ID, link.ID),
		EmailMasked: MaskEmail(o.BuyerEmail), ExpiresAt: link.ExpiresAt, Downloads: link.DownloadsAllowed, Test: o.Mode == string(payments.Test)}
	var consent checkoutConsent
	_ = json.Unmarshal(o.Consent, &consent)
	if consent.Ticked && o.ContactID != nil {
		if lid, err := uuid.Parse(consent.ListID); err == nil {
			if lst, err := s.Q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: lid}); err == nil {
				v.OfferList = lst.Name
				if m, err := s.Q.ProductBuyerMembership(ctx, store.ProductBuyerMembershipParams{ListID: lid, ContactID: *o.ContactID}); err == nil && m.Status == "pending" {
					v.OfferPending = true
				}
			}
		}
	}
	return v, true, nil
}

// MaskEmail shows an address without giving it away ("a•••@example.com").
func MaskEmail(email string) string {
	at := strings.LastIndexByte(email, '@')
	if at <= 0 {
		return "•••"
	}
	first := []rune(email[:at])[0]
	return string(first) + "•••" + email[at:]
}
