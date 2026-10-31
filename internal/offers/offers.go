// Package offers is Earn's offers (design expansion phase 6, X17): signup
// offers readers get by code or link, the retention offer of the hosted
// cancel page, and referral rewards of single-use codes. Saving an offer
// creates its coupon and promotion codes in the payment provider; the
// offer's rules, its results and readers' decisions stay here.
package offers

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/revenue"
	"github.com/hostrepute/reputecast/internal/store"
)

//go:embed messages/*.json
var messagesFS embed.FS

// msgs holds the offers' words: problems, and the upgrade and cancel
// pages' lines. Members read the app's in their app language, readers the
// pages in the page's language and the Upgrade block in the email's.
var msgs = i18n.MustLoad(messagesFS, "offers")

// text is a problem or a line of the app, in the member's language.
func text(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// page is a public page's line in the reader's language: the page's own,
// which internal/publicpages puts in the request's context (English
// without one).
func page(ctx context.Context, key string, args i18n.Args) string {
	return msgs.M(key).Text(i18n.LangFrom(ctx), args)
}

// Kinds of offer.
const (
	KindSignup    = "signup"
	KindRetention = "retention"
	KindReferral  = "referral"
)

// RetentionGap is how long after taking a retention offer a paid member
// can't be offered one again (ROADMAP: one in 12 months).
const RetentionGap = 365 * 24 * time.Hour

// Service is the offers domain.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Earn  *earn.Service
	Log   *slog.Logger
	Now   func() time.Time

	// codeMisses limits the referral codes an address can try (referral.go).
	codeMisses misses
}

// now is the offers' clock: their own, else Earn's (tests set it).
func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	if s.Earn != nil && s.Earn.Now != nil {
		return s.Earn.Now()
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

func userID(p *authz.Principal) *uuid.UUID {
	if p == nil || p.Type != authz.ActorUser || p.UserID == uuid.Nil {
		return nil
	}
	id := p.UserID
	return &id
}

// OfferApplies is a tier and period an offer applies to, with its price.
type OfferApplies struct {
	TierID string `json:"tierId"`
	Tier   string `json:"tier"`
	// Short is the tier's name without its list's ("Insider").
	Short      string `json:"short"`
	Period     string `json:"period" enum:"month,year"`
	PriceCents int64  `json:"priceCents"`
	Currency   string `json:"currency"`
}

// OfferStripe is the offer in the payment provider.
type OfferStripe struct {
	Mode   string `json:"mode,omitempty"`
	Coupon string `json:"coupon,omitempty"`
	// PromotionCode is the code readers type, as the provider names it
	// (promo_1Q9bX2rT).
	PromotionCode string `json:"promotionCode,omitempty"`
	// Missing: the coupon no longer exists in the provider.
	Missing bool `json:"missing"`
	// CouponURL opens it in the provider's dashboard.
	CouponURL string `json:"couponUrl,omitempty"`
}

// OfferResults is what an offer did.
type OfferResults struct {
	// Signup: paid members on the offer, new ones, and their MRR after the
	// discount.
	Redemptions int64 `json:"redemptions"`
	NewMembers  int64 `json:"newMembers"`
	MRRCents    int64 `json:"mrrCents"`
	// Retention: shown on the cancel page, and kept with it.
	Shown      int64    `json:"shown"`
	Accepted   int64    `json:"accepted"`
	AcceptRate *float64 `json:"acceptRate,omitempty"`
	// Referral: codes issued and redeemed.
	CodesIssued   int64 `json:"codesIssued"`
	CodesRedeemed int64 `json:"codesRedeemed"`
}

// Offer is an offer as Plans & offers lists it.
type Offer struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Kind           string         `json:"kind" enum:"signup,retention,referral"`
	DiscountType   string         `json:"discountType" enum:"percent,amount"`
	PercentOff     int            `json:"percentOff,omitempty"`
	AmountOffCents int64          `json:"amountOffCents,omitempty"`
	Currency       string         `json:"currency"`
	Duration       string         `json:"duration" enum:"once,months,forever"`
	Months         int            `json:"months,omitempty"`
	Applies        []OfferApplies `json:"applies"`
	ByCode         bool           `json:"byCode"`
	Code           string         `json:"code"`
	ByLink         bool           `json:"byLink"`
	// Link is the offer link (https://news.acme.io/upgrade?offer=FALL26);
	// LinkHost the same without its scheme.
	Link          string     `json:"link,omitempty"`
	LinkHost      string     `json:"linkHost,omitempty"`
	Limit         *int       `json:"limit,omitempty"`
	EndsAt        *time.Time `json:"endsAt,omitempty"`
	NewOnly       bool       `json:"newOnly"`
	MinPaidMonths int        `json:"minPaidMonths"`
	CodeDays      int        `json:"codeDays"`
	// Status is live or ended; State is what Plans & offers shows: live,
	// at its limit, ended, its coupon missing in the provider, or a draft
	// without one.
	Status      string     `json:"status" enum:"live,ended"`
	State       string     `json:"state" enum:"live,limit,ended,missing,draft"`
	EndedAt     *time.Time `json:"endedAt,omitempty"`
	EndedReason string     `json:"endedReason,omitempty" enum:"date,limit,early"`
	EndedBy     string     `json:"endedBy,omitempty"`
	StartedAt   time.Time  `json:"startedAt"`
	StartedBy   string     `json:"startedBy,omitempty"`
	// Sync is how saving it in the provider went.
	Sync      string       `json:"sync" enum:"draft,synced,failed"`
	SyncError string       `json:"syncError,omitempty"`
	Stripe    OfferStripe  `json:"stripe"`
	Results   OfferResults `json:"results"`
}

// OfferCode is a referral reward's single-use code.
type OfferCode struct {
	Code       string     `json:"code"`
	IssuedTo   string     `json:"issuedTo"`
	IssuedAt   time.Time  `json:"issuedAt"`
	ExpiresAt  time.Time  `json:"expiresAt"`
	RedeemedAt *time.Time `json:"redeemedAt,omitempty"`
	Status     string     `json:"status" enum:"not_redeemed,redeemed,expired"`
}

// OfferList is Plans & offers' offers.
type OfferList struct {
	Items []Offer `json:"items"`
	// Account names the provider's account ("Acme Media"); Connected: a
	// provider takes payments.
	Account   string   `json:"account"`
	Provider  string   `json:"provider"`
	Connected bool     `json:"connected"`
	CanEdit   bool     `json:"canEdit"`
	Helpers   []string `json:"helpers"`
}

// OfferDetail is the offer editor.
type OfferDetail struct {
	Offer
	Codes []OfferCode `json:"codes"`
	// Choices are the tiers and periods an offer can apply to.
	Choices   []OfferApplies `json:"choices"`
	Account   string         `json:"account"`
	Provider  string         `json:"provider"`
	Connected bool           `json:"connected"`
	CanEdit   bool           `json:"canEdit"`
	Helpers   []string       `json:"helpers"`
	// UpgradeHost is the upgrade page's address without its scheme.
	UpgradeHost string `json:"upgradeHost"`
}

// stripeState is the earn_offers.stripe column: the offer per mode.
type stripeState map[string]*stripeMode

type stripeMode struct {
	Coupon  string   `json:"coupon,omitempty"`
	Coupons []string `json:"coupons,omitempty"`
	Promo   string   `json:"promo,omitempty"`
	// MissingAt is when the coupon was found deleted in the provider.
	MissingAt string `json:"missingAt,omitempty"`
}

func parseStripe(raw []byte) stripeState {
	out := stripeState{}
	_ = json.Unmarshal(raw, &out)
	return out
}

type appliesJSON struct {
	Tier   string `json:"tier"`
	Period string `json:"period"`
}

func parseApplies(raw []byte) []appliesJSON {
	var out []appliesJSON
	_ = json.Unmarshal(raw, &out)
	return out
}

// choices are the tiers and periods with a current price.
func (s *Service) choices(ctx context.Context, ws *store.Workspace) ([]OfferApplies, map[string]store.OfferTierPricesRow, error) {
	rows, err := s.Q.OfferTierPrices(ctx, ws.ID)
	if err != nil {
		return nil, nil, err
	}
	out := []OfferApplies{}
	by := map[string]store.OfferTierPricesRow{}
	for _, r := range rows {
		key := r.TierID.String() + "|" + r.Period
		by[key] = r
		out = append(out, OfferApplies{TierID: r.TierID.String(), Tier: r.Tier, Short: s.Earn.TierShortName(ctx, ws.ID, r.TierID), Period: r.Period,
			PriceCents: r.AmountCents, Currency: r.Currency})
	}
	return out, by, nil
}

// view turns a row into an offer with its results.
func (s *Service) view(ctx context.Context, ws *store.Workspace, r store.OfferGetRow, choices []OfferApplies, mode string) Offer {
	o := Offer{ID: r.ID.String(), Name: r.Name, Kind: r.Kind, DiscountType: r.DiscountType, Currency: r.Currency, Duration: r.Duration, Applies: []OfferApplies{},
		ByCode: r.ByCode, Code: r.Code, ByLink: r.ByLink, EndsAt: r.EndsAt, NewOnly: r.NewOnly, MinPaidMonths: int(r.MinPaidMonths), CodeDays: int(r.CodeDays),
		Status: r.Status, EndedAt: r.EndedAt, EndedReason: r.EndedReason, EndedBy: r.EndedByName, StartedAt: r.StartedAt, StartedBy: r.CreatedByName,
		Sync: r.SyncStatus, SyncError: syncErrorIn(ctx, r.SyncError)}
	if r.PercentOff != nil {
		o.PercentOff = int(*r.PercentOff)
	}
	if r.AmountOff != nil {
		o.AmountOffCents = *r.AmountOff
	}
	if r.DurationMonths != nil {
		o.Months = int(*r.DurationMonths)
	}
	if r.RedeemLimit != nil {
		n := int(*r.RedeemLimit)
		o.Limit = &n
	}
	for _, a := range parseApplies(r.Applies) {
		for _, c := range choices {
			if c.TierID == a.Tier && c.Period == a.Period {
				o.Applies = append(o.Applies, c)
			}
		}
	}
	if o.Kind == KindSignup && o.ByLink && o.Code != "" {
		o.Link = s.Earn.PageURL(ctx, ws, "upgrade?offer="+o.Code)
		o.LinkHost = s.Earn.PageHost(ctx, ws, "upgrade?offer="+o.Code)
	}
	st := parseStripe(r.Stripe)
	if mode == "" {
		for _, m := range []string{"live", "test"} {
			if st[m] != nil {
				mode = m
				break
			}
		}
	}
	if sm := st[mode]; sm != nil {
		o.Stripe = OfferStripe{Mode: mode, Coupon: sm.Coupon, PromotionCode: sm.Promo, Missing: sm.MissingAt != ""}
		if sm.Coupon != "" {
			o.Stripe.CouponURL = dashboard(mode, "coupons", sm.Coupon)
		}
	}
	o.Results = s.results(ctx, ws, r, st)
	o.State = state(o)
	return o
}

// dashboard links to an object in Stripe's dashboard.
func dashboard(mode, kind, id string) string {
	if d, ok := payments.Lookup("stripe"); ok && d.Dashboard != nil {
		return d.Dashboard(payments.Mode(mode), kind, id)
	}
	return ""
}

// state is what the offer's status cell shows.
func state(o Offer) string {
	switch {
	case o.Status == "ended" && o.EndedReason == "limit":
		return "limit"
	case o.Status == "ended":
		return "ended"
	case o.Stripe.Missing:
		return "missing"
	case o.Sync != "synced":
		return "draft"
	}
	return "live"
}

func (s *Service) results(ctx context.Context, ws *store.Workspace, r store.OfferGetRow, st stripeState) OfferResults {
	var out OfferResults
	switch r.Kind {
	case KindSignup:
		coupons := []string{}
		for _, m := range st {
			if m == nil {
				continue
			}
			coupons = append(coupons, m.Coupons...)
			if m.Coupon != "" {
				coupons = append(coupons, m.Coupon)
			}
		}
		if res, err := s.Q.OfferRedemptions(ctx, store.OfferRedemptionsParams{WorkspaceID: ws.ID, StartedAt: r.StartedAt, Coupons: coupons, Code: r.Code}); err == nil {
			out.Redemptions, out.NewMembers, out.MRRCents = res.Redemptions, res.NewMembers, mrr(res.Arr)
		}
	case KindRetention:
		if res, err := s.Q.OfferRetentionResults(ctx, store.OfferRetentionResultsParams{WorkspaceID: ws.ID, OfferID: &r.ID}); err == nil {
			out.Shown, out.Accepted = res.Shown, res.Accepted
			if res.Shown > 0 {
				rate := float64(res.Accepted) / float64(res.Shown) * 100
				out.AcceptRate = &rate
			}
		}
	case KindReferral:
		if codes, err := s.Q.OfferCodes(ctx, r.ID); err == nil {
			out.CodesIssued = int64(len(codes))
			for _, c := range codes {
				if c.RedeemedAt != nil {
					out.CodesRedeemed++
				}
			}
		}
	}
	return out
}

// mrr is the monthly value of yearly amounts, rounded to the cent.
func mrr(arr int64) int64 { return (arr + 6) / 12 }

// account names the workspace's payment account.
func (s *Service) account(ctx context.Context, ws *store.Workspace) (mode, account, provider string) {
	mode = s.Earn.ActiveMode(ctx, ws.ID)
	if mode == "" {
		return "", "", ""
	}
	_, conn, err := s.Earn.ClientFor(ctx, ws.ID, mode)
	if err != nil {
		return mode, "", ""
	}
	name := conn.Provider
	if d, ok := payments.Lookup(conn.Provider); ok {
		name = d.Name
	}
	return mode, conn.AccountName, name
}

// List is Plans & offers' offers with their results.
func (s *Service) List(ctx context.Context, ws *store.Workspace, p *authz.Principal) (OfferList, error) {
	out := OfferList{Items: []Offer{}, Helpers: []string{}, CanEdit: p.Can(authz.EarnEdit)}
	if !out.CanEdit {
		out.Helpers = revenue.Helpers(ctx, s.Q, ws.ID, p.UserID, 2, authz.EarnEdit)
	}
	mode, account, provider := s.account(ctx, ws)
	out.Account, out.Provider, out.Connected = account, provider, mode != ""
	choices, _, err := s.choices(ctx, ws)
	if err != nil {
		return out, err
	}
	rows, err := s.Q.OfferList(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Items = append(out.Items, s.view(ctx, ws, store.OfferGetRow(r), choices, mode))
	}
	return out, nil
}

// Detail is the offer editor: the offer, its codes and the tiers it can
// apply to. Opening it checks that the coupon still exists in the
// provider.
func (s *Service) Detail(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (OfferDetail, error) {
	r, err := s.Q.OfferGet(ctx, store.OfferGetParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return OfferDetail{}, notFound(ctx)
	}
	if err != nil {
		return OfferDetail{}, err
	}
	mode, account, provider := s.account(ctx, ws)
	if mode != "" && r.Status == "live" {
		if changed := s.checkCoupon(ctx, ws, &r, mode); changed {
			r, _ = s.Q.OfferGet(ctx, store.OfferGetParams{ID: id, WorkspaceID: ws.ID})
		}
	}
	return s.detail(ctx, ws, p, r, mode, account, provider)
}

// New is the editor of a new offer: its choices only.
func (s *Service) New(ctx context.Context, ws *store.Workspace, p *authz.Principal) (OfferDetail, error) {
	mode, account, provider := s.account(ctx, ws)
	choices, _, err := s.choices(ctx, ws)
	if err != nil {
		return OfferDetail{}, err
	}
	out := OfferDetail{Offer: Offer{Kind: KindSignup, DiscountType: "percent", Duration: "once", Applies: []OfferApplies{}, Status: "live", State: "draft", Sync: "draft",
		CodeDays: 60}, Codes: []OfferCode{}, Choices: choices, Account: account, Provider: provider, Connected: mode != "", CanEdit: p.Can(authz.EarnEdit),
		Helpers: []string{}, UpgradeHost: s.Earn.PageHost(ctx, ws, "upgrade")}
	if !out.CanEdit {
		out.Helpers = revenue.Helpers(ctx, s.Q, ws.ID, p.UserID, 2, authz.EarnEdit)
	}
	return out, nil
}

func (s *Service) detail(ctx context.Context, ws *store.Workspace, p *authz.Principal, r store.OfferGetRow, mode, account, provider string) (OfferDetail, error) {
	choices, _, err := s.choices(ctx, ws)
	if err != nil {
		return OfferDetail{}, err
	}
	out := OfferDetail{Offer: s.view(ctx, ws, r, choices, mode), Codes: []OfferCode{}, Choices: choices, Account: account, Provider: provider,
		Connected: mode != "", CanEdit: p.Can(authz.EarnEdit), Helpers: []string{}, UpgradeHost: s.Earn.PageHost(ctx, ws, "upgrade")}
	if !out.CanEdit {
		out.Helpers = revenue.Helpers(ctx, s.Q, ws.ID, p.UserID, 2, authz.EarnEdit)
	}
	if r.Kind == KindReferral {
		codes, err := s.Q.OfferCodes(ctx, r.ID)
		if err != nil {
			return out, err
		}
		now := s.now()
		for _, c := range codes {
			oc := OfferCode{Code: c.Code, IssuedTo: c.IssuedName, IssuedAt: c.IssuedAt, ExpiresAt: c.ExpiresAt, RedeemedAt: c.RedeemedAt, Status: "not_redeemed"}
			switch {
			case c.RedeemedAt != nil:
				oc.Status = "redeemed"
			case c.ExpiresAt.Before(now):
				oc.Status = "expired"
			}
			out.Codes = append(out.Codes, oc)
		}
	}
	return out, nil
}
