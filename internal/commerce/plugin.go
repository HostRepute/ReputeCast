package commerce

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
)

// WooCommerce has no abandoned checkouts and no marketing consent of its
// own: the ReputeCast WordPress plugin adds the consent checkbox at the
// checkout and reports the checkouts left without an order, with the
// recovery link it answers. The site's key (the plugin's) calls this; the
// checkouts count only for the WooCommerce store connected at that
// address.

// OpWooCheckouts is the plugin's checkout report (every WordPress site's key
// may call it, like its report).
const OpWooCheckouts = "report-woocommerce-checkouts"

// CommerceWooCheckoutsInput is a batch of checkouts from the plugin.
type CommerceWooCheckoutsInput struct {
	SiteURL   string                `json:"siteUrl" maxLength:"300" example:"https://store.example.com"`
	Checkouts []CommerceWooCheckout `json:"checkouts" maxItems:"200"`
}

// CommerceWooCheckout is one checkout: who (when they gave their email), what
// they ticked, the cart and the plugin's recovery link.
type CommerceWooCheckout struct {
	Token     string `json:"token" minLength:"8" maxLength:"64" doc:"The plugin's checkout token"`
	Email     string `json:"email,omitempty" maxLength:"254"`
	FirstName string `json:"firstName,omitempty" maxLength:"100"`
	LastName  string `json:"lastName,omitempty" maxLength:"100"`
	// Accepted: the consent checkbox was ticked; ConsentText is its exact
	// text and ConsentAt when it was ticked.
	Accepted    bool                      `json:"accepted"`
	ConsentText string                    `json:"consentText,omitempty" maxLength:"500"`
	ConsentAt   *time.Time                `json:"consentAt,omitempty"`
	Items       []CommerceWooCheckoutItem `json:"items" maxItems:"100"`
	// Amounts are decimal strings in the store's currency ("76.00").
	Subtotal    string     `json:"subtotal" maxLength:"20"`
	Total       string     `json:"total" maxLength:"20"`
	Currency    string     `json:"currency" maxLength:"3"`
	RecoveryURL string     `json:"recoveryUrl" maxLength:"1000"`
	ClickToken  string     `json:"clickToken,omitempty" maxLength:"200"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty" doc:"When it became an order"`
}

// CommerceWooCheckoutItem is a cart line.
type CommerceWooCheckoutItem struct {
	ProductID string `json:"productId" maxLength:"40"`
	VariantID string `json:"variantId,omitempty" maxLength:"40"`
	Title     string `json:"title" maxLength:"300"`
	Variant   string `json:"variant,omitempty" maxLength:"200"`
	Quantity  int    `json:"quantity" minimum:"0" maximum:"100000"`
	Price     string `json:"price" maxLength:"20" doc:"The unit price, a decimal string"`
	ImageURL  string `json:"imageUrl,omitempty" maxLength:"1000"`
}

// CommerceWooCheckoutsResult says what the batch did.
type CommerceWooCheckoutsResult struct {
	Store    string `json:"store" doc:"The store's name"`
	Accepted int    `json:"accepted"`
	Ignored  int    `json:"ignored"`
}

// WooCheckouts applies the plugin's checkouts to the WooCommerce store at
// the site's address.
func (s *Service) WooCheckouts(ctx context.Context, ws *store.Workspace, in CommerceWooCheckoutsInput) (CommerceWooCheckoutsResult, error) {
	l := i18n.LangFrom(ctx)
	u, err := url.Parse(strings.TrimSpace(in.SiteURL))
	if err != nil || u.Hostname() == "" {
		return CommerceWooCheckoutsResult{}, api.Invalid("body.siteUrl", mAddressMissing.Text(l, nil))
	}
	host := stores.NormalizeHost(u.Hostname())
	st, err := s.Q.CommerceStoreByAddress(ctx, store.CommerceStoreByAddressParams{WorkspaceID: ws.ID, Platform: WooCommerce, Address: host})
	if errors.Is(err, pgx.ErrNoRows) || err == nil && st.Status != StatusConnected && st.Status != StatusImporting {
		return CommerceWooCheckoutsResult{}, api.Err(http.StatusConflict, "store_not_connected", mWooNotConnected.Text(l, i18n.Args{"address": host}))
	}
	if err != nil {
		return CommerceWooCheckoutsResult{}, err
	}
	out := CommerceWooCheckoutsResult{Store: st.Name}
	if !st.SyncCheckouts {
		out.Ignored = len(in.Checkouts)
		return out, nil
	}
	if !st.PluginFound {
		// The plugin reports: it's there.
		_ = s.Q.CommerceStoreSetPlugin(ctx, store.CommerceStoreSetPluginParams{ID: st.ID, PluginFound: true, At: s.now()})
		s.forget(ws.ID)
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, c := range in.Checkouts {
			ch, ok := wooCheckout(c, st)
			if !ok {
				out.Ignored++
				continue
			}
			res, err := s.applyCheckout(ctx, q, st, ch, sourcePlugin)
			if err != nil {
				return err
			}
			if res.status == "applied" {
				out.Accepted++
			} else {
				out.Ignored++
			}
		}
		return nil
	})
	if err != nil {
		return CommerceWooCheckoutsResult{}, err
	}
	_ = s.Q.CommerceStoreTouchEvent(ctx, store.CommerceStoreTouchEventParams{ID: st.ID, At: s.now()})
	return out, nil
}

// wooCheckout reads a plugin checkout, its amounts in the store's currency.
func wooCheckout(c CommerceWooCheckout, st store.Store) (stores.Checkout, bool) {
	if strings.TrimSpace(c.Token) == "" {
		return stores.Checkout{}, false
	}
	cur := strings.ToUpper(strings.TrimSpace(c.Currency))
	if cur == "" {
		cur = st.Currency
	}
	sub, total := stores.ParseAmount(c.Subtotal, cur), stores.ParseAmount(c.Total, cur)
	ch := stores.Checkout{ID: "plugin:" + strings.TrimSpace(c.Token), Email: strings.ToLower(strings.TrimSpace(c.Email)), AcceptsMarketing: c.Accepted,
		Subtotal: sub, Total: total, Currency: cur, RecoveryURL: c.RecoveryURL, ClickToken: c.ClickToken, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
		CompletedAt: c.CompletedAt}
	for _, it := range c.Items {
		price := stores.ParseAmount(it.Price, cur)
		ch.Items = append(ch.Items, stores.LineItem{ProductID: it.ProductID, VariantID: it.VariantID, Title: it.Title, Variant: it.Variant,
			Quantity: it.Quantity, Price: price, ImageURL: it.ImageURL})
	}
	if ch.Email != "" {
		cust := stores.Customer{Email: ch.Email, FirstName: c.FirstName, LastName: c.LastName, UpdatedAt: c.UpdatedAt}
		if c.Accepted {
			cust.Consent = stores.Consent{State: stores.ConsentSubscribed, Level: "single_opt_in", UpdatedAt: c.ConsentAt, Text: c.ConsentText}
		} else {
			cust.Consent = stores.Consent{State: stores.ConsentNotSubscribed}
		}
		ch.Customer = &cust
	}
	return ch, true
}
