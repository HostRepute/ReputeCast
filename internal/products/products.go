// Package products implements digital products (design expansion phase 7,
// M29): a file and a price, sold through Stripe Checkout on the
// publisher's own Stripe account; orders with their refunds and disputes,
// which happen in Stripe and arrive through its webhooks; delivery through
// the Purchases category's "Product delivery" transactional template, with
// a personal download link that expires after a number of days and of
// downloads; and the lead magnets signup forms offer after confirmation.
//
// Everything is free: ReputeCast takes no fee and gates nothing. No card
// details are ever stored, and the download link isn't either: it is
// signed from its id when it's needed.
package products

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "products")

// Link token kinds (design expansion X19: 190–199).
const (
	// TokenDownload is a buyer's download link: MessageID is the link's
	// id, ContactID the order's.
	TokenDownload linktoken.Kind = 190
	// TokenLeadMagnet is a lead magnet's download after confirmation:
	// MessageID is the lead_magnet_links row.
	TokenLeadMagnet linktoken.Kind = 191
	// TokenThanks opens an order's thank-you page from the checkout's
	// return: MessageID is the order.
	TokenThanks linktoken.Kind = 192
)

// Statuses.
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"

	PriceOneTime = "one_time"
	PricePWYW    = "pwyw"

	OrderPaid              = "paid"
	OrderRefunded          = "refunded"
	OrderPartiallyRefunded = "partially_refunded"
	OrderDisputed          = "disputed"

	SourcePage        = "page"
	SourceShop        = "shop"
	SourceProductPage = "product_page"

	// MinimumCents is the least a price, or a pay-what-you-want minimum,
	// may be: Stripe can't charge less than about $1 in most currencies.
	MinimumCents int64 = 100
)

// Service holds the products domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Log   *slog.Logger
	Now   func() time.Time
	URLs  publicurl.Builder
	Links *linktoken.Signer
	Files Store
	// Forms gives the public site's links (the custom domain once it's
	// verified).
	Forms *forms.Service
	// Earn is the payment connection products sell through.
	Earn func() *earn.Service
	// Transactional sends the delivery email.
	Transactional func() *transactional.Service
	// Audience subscribes a buyer who ticked the list offer.
	Audience func() *audience.Service
	// Jobs queues the delivery emails, in the transaction that makes them.
	Jobs func() *jobs.Client
	// Flows makes contacts leave "Bought a product" automations after a
	// refund (nil: none).
	Flows Flows
	// Media keeps cover images in the image library.
	Media *media.Service
	// SiteOn says whether the publication website is on: buyers can sign in
	// to its account page for a new link (nil: off).
	SiteOn func(ctx context.Context, ws *store.Workspace) bool
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// Live is a published product as the shop, its pages and checkout see it.
type Live struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	ShortDesc        string `json:"shortDesc"`
	LongDesc         string `json:"longDesc"`
	CoverURL         string `json:"coverUrl"`
	CoverAlt         string `json:"coverAlt"`
	PriceKind        string `json:"priceKind"`
	PriceCents       int64  `json:"priceCents"`
	MinCents         int64  `json:"minCents"`
	SuggestedCents   int64  `json:"suggestedCents"`
	SalesPageID      string `json:"salesPageId,omitempty"`
	ShowInShop       bool   `json:"showInShop"`
	OfferOn          bool   `json:"offerOn"`
	OfferListID      string `json:"offerListId,omitempty"`
	OfferText        string `json:"offerText"`
	LinkDays         int32  `json:"linkDays"`
	LinkDownloads    int32  `json:"linkDownloads"`
	DeliveryTemplate string `json:"deliveryTemplate,omitempty"`
	RevokeOnRefund   bool   `json:"revokeOnRefund"`
}

// draftOf is a product's draft as a Live snapshot (to publish or compare).
func draftOf(p store.Product) Live {
	l := Live{Name: p.Name, Slug: p.Slug, ShortDesc: p.ShortDesc, LongDesc: p.LongDesc, CoverURL: p.CoverUrl, CoverAlt: p.CoverAlt,
		PriceKind: p.PriceKind, PriceCents: p.PriceCents, MinCents: p.MinCents, SuggestedCents: p.SuggestedCents, ShowInShop: p.ShowInShop,
		OfferOn: p.OfferOn, OfferText: p.OfferText, LinkDays: p.LinkDays, LinkDownloads: p.LinkDownloads, RevokeOnRefund: p.RevokeOnRefund}
	if p.SalesPageID != nil {
		l.SalesPageID = p.SalesPageID.String()
	}
	if p.OfferListID != nil {
		l.OfferListID = p.OfferListID.String()
	}
	if p.DeliveryTemplateID != nil {
		l.DeliveryTemplate = p.DeliveryTemplateID.String()
	}
	return l
}

// liveOf reads a product's published snapshot (ok false while a draft).
func liveOf(p store.Product) (Live, bool) {
	if len(p.Live) == 0 || string(p.Live) == "null" {
		return Live{}, false
	}
	var l Live
	if err := json.Unmarshal(p.Live, &l); err != nil {
		return Live{}, false
	}
	return l, true
}

// stripeIDs are a product's objects at the payment provider, per mode.
type stripeIDs map[string]stripeMode

type stripeMode struct {
	Product   string   `json:"product"`
	Price     string   `json:"price"`
	Kind      string   `json:"kind"`
	Amount    int64    `json:"amount"`
	Min       int64    `json:"min"`
	Suggested int64    `json:"suggested"`
	Retired   []string `json:"retired,omitempty"`
}

func readStripe(raw []byte) stripeIDs {
	out := stripeIDs{}
	_ = json.Unmarshal(raw, &out)
	return out
}

// ShortName is how tables name a product: "Field Guide" for "The
// Deliverability Field Guide", "Launch Kit" for "Newsletter Launch Kit".
// The full name stays in the panel and in tooltips.
func ShortName(name string) string {
	words := strings.Fields(name)
	if len(words) > 0 && (strings.EqualFold(words[0], "the") || strings.EqualFold(words[0], "a")) {
		words = words[1:]
	}
	if len(words) >= 3 {
		words = words[len(words)-2:]
	}
	out := strings.Join(words, " ")
	if out == "" {
		return name
	}
	return out
}

// kindLabel is a file's kind in capitals, as the product's summary shows
// it ("PDF", "ZIP", "XLSX").
func kindLabel(kind, name string) string {
	switch kind {
	case "", "text":
		if e := extOf(name); e != "" {
			return strings.ToUpper(e)
		}
		return "File"
	case "jpeg":
		return "JPEG"
	}
	return strings.ToUpper(kind)
}

// Slugify makes an address from a name.
func Slugify(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			// Letters outside a–z (é, ü) are dropped rather than guessed.
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 80 {
		out = strings.Trim(out[:80], "-")
	}
	if out == "" {
		out = "product"
	}
	return out
}

// actor is the principal's user, for the audit and "by" columns.
func actor(p *authz.Principal) *uuid.UUID {
	if p == nil || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

// lang is the request's language for server words.
func lang(ctx context.Context) i18n.Lang { return i18n.LangFrom(ctx) }

// word is one of the package's server words in a language.
func word(l i18n.Lang, key string, args i18n.Args) string { return msgs.M(key).Text(l, args) }

// notFound is a 404 whose sentence is in the reader's language: what names
// a notFound.* message ("product": "Product not found.").
func notFound(ctx context.Context, what string) *api.Problem {
	return api.Err(http.StatusNotFound, api.CodeNotFound, word(lang(ctx), "notFound."+what, nil))
}

// mapErr is api.MapError with its not-found sentence in the reader's
// language.
func mapErr(ctx context.Context, err error, what string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound(ctx, what)
	}
	return api.MapError(err, msgs.M("notFound."+what).Source())
}

// standard reports whether a language writes ReputeCast's own formats.
func standard(l i18n.Lang) bool { return l.Region == "" || l.Region == i18n.Standard }

// slugRe is an address's shape: lowercase letters, digits and single hyphens.
var slugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
