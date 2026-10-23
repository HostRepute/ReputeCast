package offers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// OfferAppliesInput is a tier and period an offer applies to.
type OfferAppliesInput struct {
	TierID string `json:"tierId" format:"uuid"`
	Period string `json:"period" enum:"month,year"`
}

// OfferInput creates or changes an offer.
type OfferInput struct {
	Name           string              `json:"name" maxLength:"80"`
	Kind           string              `json:"kind" enum:"signup,retention,referral"`
	DiscountType   string              `json:"discountType" enum:"percent,amount"`
	PercentOff     int                 `json:"percentOff,omitempty"`
	AmountOffCents int64               `json:"amountOffCents,omitempty"`
	Duration       string              `json:"duration" enum:"once,months,forever"`
	Months         int                 `json:"months,omitempty"`
	Applies        []OfferAppliesInput `json:"applies" maxItems:"40"`
	ByCode         bool                `json:"byCode,omitempty"`
	Code           string              `json:"code,omitempty" maxLength:"40"`
	ByLink         bool                `json:"byLink,omitempty"`
	Limit          *int                `json:"limit,omitempty"`
	// EndsOn is the last day of the offer, which ends at 23:59 in the
	// workspace's time zone.
	EndsOn        string `json:"endsOn,omitempty" doc:"YYYY-MM-DD"`
	NewOnly       bool   `json:"newOnly,omitempty"`
	MinPaidMonths int    `json:"minPaidMonths,omitempty"`
	CodeDays      int    `json:"codeDays,omitempty"`
}

var codeRe = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]{1,39}$`)

func invalid(ctx context.Context, location, key string, args i18n.Args) *api.Problem {
	return api.Invalid(location, text(ctx, key, args))
}

// notFound is a 404 in the reader's language.
func notFound(ctx context.Context) *api.Problem {
	return api.Err(http.StatusNotFound, api.CodeNotFound, text(ctx, "problem.notFound", nil))
}

// location is the workspace's time zone.
func location(ws *store.Workspace) *time.Location {
	if ws != nil && ws.Timezone != "" {
		if loc, err := time.LoadLocation(ws.Timezone); err == nil {
			return loc
		}
	}
	return time.UTC
}

// clean checks an input and fills what it implies.
type cleaned struct {
	in      OfferInput
	endsAt  *time.Time
	applies []appliesJSON
	// products are the provider's products of the tiers, by mode.
	products map[string][]string
}

func (s *Service) clean(ctx context.Context, ws *store.Workspace, id uuid.UUID, in OfferInput) (cleaned, error) {
	c := cleaned{in: in, products: map[string][]string{}}
	c.in.Name = strings.TrimSpace(in.Name)
	if c.in.Name == "" {
		return c, invalid(ctx, "body.name", "problem.name", nil)
	}
	switch in.DiscountType {
	case "percent":
		if in.PercentOff < 1 || in.PercentOff > 100 {
			return c, invalid(ctx, "body.percentOff", "problem.percent", nil)
		}
		c.in.AmountOffCents = 0
	case "amount":
		if in.AmountOffCents < 1 {
			return c, invalid(ctx, "body.amountOffCents", "problem.amount", nil)
		}
		c.in.PercentOff = 0
	}
	if in.Duration == "months" {
		if in.Months < 1 || in.Months > 36 {
			return c, invalid(ctx, "body.months", "problem.months", nil)
		}
	} else {
		c.in.Months = 0
	}
	_, prices, err := s.choices(ctx, ws)
	if err != nil {
		return c, err
	}
	if len(in.Applies) == 0 {
		return c, invalid(ctx, "body.applies", "problem.applies", nil)
	}
	seen := map[string]bool{}
	for i, a := range in.Applies {
		r, ok := prices[a.TierID+"|"+a.Period]
		if !ok {
			return c, invalid(ctx, fmt.Sprintf("body.applies[%d]", i), "problem.appliesPrice", nil)
		}
		if in.DiscountType == "amount" && in.AmountOffCents >= r.AmountCents {
			return c, invalid(ctx, "body.amountOffCents", "problem.amountTooBig", i18n.Args{"price": moneyOf(r.AmountCents, r.Currency)})
		}
		key := a.TierID + "|" + a.Period
		if seen[key] {
			continue
		}
		seen[key] = true
		c.applies = append(c.applies, appliesJSON{Tier: a.TierID, Period: a.Period})
		for m, prod := range map[string]string{"live": r.StripeProductLive, "test": r.StripeProductTest} {
			if prod != "" && !slices.Contains(c.products[m], prod) {
				c.products[m] = append(c.products[m], prod)
			}
		}
	}
	switch in.Kind {
	case KindSignup:
		c.in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
		if !in.ByCode && !in.ByLink {
			return c, invalid(ctx, "body.byCode", "problem.ways", nil)
		}
		if !codeRe.MatchString(c.in.Code) {
			return c, invalid(ctx, "body.code", "problem.code", nil)
		}
		taken, err := s.Q.OfferCodeTaken(ctx, store.OfferCodeTakenParams{WorkspaceID: ws.ID, Code: c.in.Code, ID: id})
		if err != nil {
			return c, err
		}
		if taken {
			return c, invalid(ctx, "body.code", "problem.codeTaken", i18n.Args{"code": c.in.Code})
		}
		c.in.MinPaidMonths = 0
	case KindRetention:
		c.in.ByCode, c.in.ByLink, c.in.Code, c.in.Limit, c.in.EndsOn, c.in.NewOnly = false, false, "", nil, "", false
		if in.MinPaidMonths < 0 || in.MinPaidMonths > 60 {
			return c, invalid(ctx, "body.minPaidMonths", "problem.minMonths", nil)
		}
	case KindReferral:
		c.in.ByCode, c.in.ByLink, c.in.Code, c.in.Limit, c.in.EndsOn, c.in.MinPaidMonths = false, false, "", nil, "", 0
		if c.in.CodeDays == 0 {
			c.in.CodeDays = 60
		}
		if c.in.CodeDays < 1 || c.in.CodeDays > 365 {
			return c, invalid(ctx, "body.codeDays", "problem.codeDays", nil)
		}
	}
	if c.in.CodeDays == 0 {
		c.in.CodeDays = 60
	}
	if c.in.Limit != nil && (*c.in.Limit < 1 || *c.in.Limit > 1_000_000) {
		return c, invalid(ctx, "body.limit", "problem.limit", nil)
	}
	if c.in.EndsOn != "" {
		d, err := time.ParseInLocation(time.DateOnly, c.in.EndsOn, location(ws))
		if err != nil {
			return c, invalid(ctx, "body.endsOn", "problem.endsOn", nil)
		}
		end := d.Add(23*time.Hour + 59*time.Minute)
		if !end.After(s.now()) {
			return c, invalid(ctx, "body.endsOn", "problem.endsPast", nil)
		}
		c.endsAt = &end
	}
	return c, nil
}

func int32p(n int) *int32 {
	v := int32(n) //nolint:gosec // bounded by the checks above
	return &v
}

func (c cleaned) row() (percent *int32, amount *int64, months *int32, limit *int32) {
	if c.in.DiscountType == "percent" {
		percent = int32p(c.in.PercentOff)
	} else {
		a := c.in.AmountOffCents
		amount = &a
	}
	if c.in.Duration == "months" {
		months = int32p(c.in.Months)
	}
	if c.in.Limit != nil {
		limit = int32p(*c.in.Limit)
	}
	return
}

// facts are an offer's audited fields.
func facts(in OfferInput, endsAt *time.Time) map[string]any {
	m := map[string]any{"name": in.Name, "kind": in.Kind, "discountType": in.DiscountType, "duration": in.Duration}
	if in.DiscountType == "percent" {
		m["percentOff"] = in.PercentOff
	} else {
		m["amountOffCents"] = in.AmountOffCents
	}
	if in.Months > 0 {
		m["months"] = in.Months
	}
	if in.Code != "" {
		m["code"] = in.Code
	}
	if in.Limit != nil {
		m["limit"] = *in.Limit
	}
	if endsAt != nil {
		m["endsAt"] = endsAt.UTC().Format(time.RFC3339)
	}
	if in.MinPaidMonths > 0 {
		m["minPaidMonths"] = in.MinPaidMonths
	}
	return m
}

// Create saves a new offer and creates its coupon and codes in the
// provider. A provider that's down leaves it a draft to save again.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, p *authz.Principal, in OfferInput) (OfferDetail, error) {
	id := newID()
	c, err := s.clean(ctx, ws, id, in)
	if err != nil {
		return OfferDetail{}, err
	}
	if c.in.Kind == KindRetention {
		if r, err := s.Q.OfferRetentionOf(ctx, ws.ID); err == nil {
			return OfferDetail{}, api.Err(http.StatusConflict, api.CodeConflict, text(ctx, "problem.oneRetention", i18n.Args{"name": r.Name}))
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return OfferDetail{}, err
		}
	}
	applies, _ := json.Marshal(c.applies)
	percent, amount, months, limit := c.row()
	now := s.now()
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.OfferInsert(ctx, store.OfferInsertParams{ID: id, WorkspaceID: ws.ID, Name: c.in.Name, Kind: c.in.Kind, DiscountType: c.in.DiscountType,
			PercentOff: percent, AmountOff: amount, Currency: currencyOf(ctx, s, ws), Duration: c.in.Duration, DurationMonths: months, Applies: applies,
			ByCode: c.in.ByCode, Code: c.in.Code, ByLink: c.in.ByLink, RedeemLimit: limit, EndsAt: c.endsAt, NewOnly: c.in.NewOnly,
			MinPaidMonths: int32(c.in.MinPaidMonths), CodeDays: int32(c.in.CodeDays), At: now, CreatedBy: userID(p)}); err != nil { //nolint:gosec // bounded
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.offer_created", TargetType: "earn_offer", TargetID: id.String(),
			TargetName: c.in.Name, After: facts(c.in, c.endsAt)})
	})
	if err != nil {
		return OfferDetail{}, err
	}
	s.sync(ctx, ws, id, c.products, true)
	return s.Detail(ctx, ws, p, id)
}

// Update changes an offer. A changed discount is a new coupon in the
// provider (codes already used keep theirs); a changed code, limit or date
// is a new promotion code. Raising the limit or moving the date of an
// offer that ended by them brings it back.
func (s *Service) Update(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in OfferInput) (OfferDetail, error) {
	before, err := s.Q.OfferGet(ctx, store.OfferGetParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return OfferDetail{}, notFound(ctx)
	}
	if err != nil {
		return OfferDetail{}, err
	}
	in.Kind = before.Kind // an offer keeps its kind
	c, err := s.clean(ctx, ws, id, in)
	if err != nil {
		return OfferDetail{}, err
	}
	applies, _ := json.Marshal(c.applies)
	percent, amount, months, limit := c.row()
	discountChanged := before.DiscountType != c.in.DiscountType || !eq32(before.PercentOff, percent) || !eq64(before.AmountOff, amount) ||
		before.Duration != c.in.Duration || !eq32(before.DurationMonths, months) || string(before.Applies) != string(applies)
	codeChanged := before.Code != c.in.Code || !eq32(before.RedeemLimit, limit) || !eqTime(before.EndsAt, c.endsAt) || before.NewOnly != c.in.NewOnly
	now := s.now()
	beforeIn := OfferInput{Name: before.Name, Kind: before.Kind, DiscountType: before.DiscountType, Duration: before.Duration, Code: before.Code, NewOnly: before.NewOnly,
		MinPaidMonths: int(before.MinPaidMonths)}
	if before.PercentOff != nil {
		beforeIn.PercentOff = int(*before.PercentOff)
	}
	if before.AmountOff != nil {
		beforeIn.AmountOffCents = *before.AmountOff
	}
	if before.DurationMonths != nil {
		beforeIn.Months = int(*before.DurationMonths)
	}
	if before.RedeemLimit != nil {
		n := int(*before.RedeemLimit)
		beforeIn.Limit = &n
	}
	reopen := before.Status == "ended" && (before.EndedReason == "date" || before.EndedReason == "limit") &&
		(c.endsAt == nil || c.endsAt.After(now)) && (codeChanged)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.OfferUpdate(ctx, store.OfferUpdateParams{ID: id, WorkspaceID: ws.ID, Name: c.in.Name, DiscountType: c.in.DiscountType, PercentOff: percent,
			AmountOff: amount, Currency: before.Currency, Duration: c.in.Duration, DurationMonths: months, Applies: applies, ByCode: c.in.ByCode, Code: c.in.Code,
			ByLink: c.in.ByLink, RedeemLimit: limit, EndsAt: c.endsAt, NewOnly: c.in.NewOnly, MinPaidMonths: int32(c.in.MinPaidMonths), //nolint:gosec // bounded
			CodeDays: int32(c.in.CodeDays), At: now}); err != nil { //nolint:gosec // bounded
			return err
		}
		if reopen {
			if err := q.OfferReopen(ctx, store.OfferReopenParams{ID: id, At: now}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.offer_updated", TargetType: "earn_offer", TargetID: id.String(),
			TargetName: c.in.Name, Before: facts(beforeIn, before.EndsAt), After: facts(c.in, c.endsAt)})
	})
	if err != nil {
		return OfferDetail{}, err
	}
	if discountChanged || codeChanged || before.SyncStatus != "synced" {
		s.sync(ctx, ws, id, c.products, discountChanged || before.SyncStatus != "synced")
	}
	return s.Detail(ctx, ws, p, id)
}

func eq32(a, b *int32) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
func eq64(a, b *int64) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
func eqTime(a, b *time.Time) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && a.Equal(*b))
}

// End ends an offer now: its code stops working; paid members who used it
// keep their discount.
func (s *Service) End(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (OfferDetail, error) {
	r, err := s.Q.OfferGet(ctx, store.OfferGetParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return OfferDetail{}, notFound(ctx)
	}
	if err != nil {
		return OfferDetail{}, err
	}
	if r.Status != "live" {
		return s.Detail(ctx, ws, p, id)
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.OfferEnd(ctx, store.OfferEndParams{ID: id, At: s.now(), Reason: "early", EndedBy: userID(p)}); err != nil {
			return err
		}
		before := map[string]any{"status": "live"}
		if r.EndsAt != nil {
			before["endsAt"] = r.EndsAt.UTC().Format(time.RFC3339)
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "earn.offer_ended", TargetType: "earn_offer", TargetID: id.String(),
			TargetName: r.Name, Before: before, After: map[string]any{"status": "ended", "reason": "early"}})
	})
	if err != nil {
		return OfferDetail{}, err
	}
	s.deactivate(ctx, ws, r)
	return s.Detail(ctx, ws, p, id)
}

// Recreate creates the offer's coupon again after it was deleted in the
// provider.
func (s *Service) Recreate(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (OfferDetail, error) {
	r, err := s.Q.OfferGet(ctx, store.OfferGetParams{ID: id, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return OfferDetail{}, notFound(ctx)
	}
	if err != nil {
		return OfferDetail{}, err
	}
	in := s.inputOf(r)
	c, err := s.clean(ctx, ws, id, in)
	if err != nil {
		return OfferDetail{}, err
	}
	if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &ws.ID, Action: "earn.offer_updated", TargetType: "earn_offer", TargetID: id.String(),
			TargetName: r.Name, Metadata: map[string]any{"coupon": "created again"}})
	}); err != nil {
		return OfferDetail{}, err
	}
	s.sync(ctx, ws, id, c.products, true)
	return s.Detail(ctx, ws, p, id)
}

// inputOf is an offer as an input, to sync it again.
func (s *Service) inputOf(r store.OfferGetRow) OfferInput {
	in := OfferInput{Name: r.Name, Kind: r.Kind, DiscountType: r.DiscountType, Duration: r.Duration, ByCode: r.ByCode, Code: r.Code, ByLink: r.ByLink,
		NewOnly: r.NewOnly, MinPaidMonths: int(r.MinPaidMonths), CodeDays: int(r.CodeDays)}
	if r.PercentOff != nil {
		in.PercentOff = int(*r.PercentOff)
	}
	if r.AmountOff != nil {
		in.AmountOffCents = *r.AmountOff
	}
	if r.DurationMonths != nil {
		in.Months = int(*r.DurationMonths)
	}
	if r.RedeemLimit != nil {
		n := int(*r.RedeemLimit)
		in.Limit = &n
	}
	for _, a := range parseApplies(r.Applies) {
		in.Applies = append(in.Applies, OfferAppliesInput{TierID: a.Tier, Period: a.Period})
	}
	return in
}

// currencyOf is the workspace's selling currency.
func currencyOf(ctx context.Context, s *Service, ws *store.Workspace) string {
	if st, err := s.Q.RevenueAttributionSettings(ctx, ws.ID); err == nil && st.Currency != "" {
		return st.Currency
	}
	return "usd"
}

// couponID is the readable id a signup offer's coupon gets: its code and
// discount ("FALL26-20").
func couponID(r store.OfferGetRow) string {
	if r.Kind != KindSignup || r.Code == "" {
		return ""
	}
	if r.PercentOff != nil {
		return r.Code + "-" + strconv.Itoa(int(*r.PercentOff))
	}
	if r.AmountOff != nil {
		return r.Code + "-" + strconv.FormatInt(*r.AmountOff, 10)
	}
	return ""
}

// sync creates the offer in the provider of the workspace's active mode:
// a coupon when the discount is new or changed, and a signup offer's
// promotion code. Failing leaves the offer to save again; nothing reaches
// the provider without a connection.
func (s *Service) sync(ctx context.Context, ws *store.Workspace, id uuid.UUID, products map[string][]string, newCoupon bool) {
	r, err := s.Q.OfferGet(ctx, store.OfferGetParams{ID: id, WorkspaceID: ws.ID})
	if err != nil {
		return
	}
	mode := s.Earn.ActiveMode(ctx, ws.ID)
	if mode == "" {
		_ = s.Q.OfferSetStripe(ctx, store.OfferSetStripeParams{ID: id, Stripe: r.Stripe, SyncStatus: "draft", SyncError: "", At: s.now()})
		return
	}
	st := parseStripe(r.Stripe)
	fail := func(err error) {
		s.log().WarnContext(ctx, "offers: saving in the provider", "offer", id, "err", err)
		b, _ := json.Marshal(st)
		_ = s.Q.OfferSetStripe(ctx, store.OfferSetStripeParams{ID: id, Stripe: b, SyncStatus: "failed", SyncError: syncError(err), At: s.now()})
	}
	client, _, err := s.Earn.ClientFor(ctx, ws.ID, mode)
	if err != nil {
		fail(err)
		return
	}
	off, ok := client.(payments.Offers)
	if !ok {
		_ = s.Q.OfferSetStripe(ctx, store.OfferSetStripeParams{ID: id, Stripe: r.Stripe, SyncStatus: "draft", SyncError: "", At: s.now()})
		return
	}
	sm := st[mode]
	if sm == nil {
		sm = &stripeMode{}
		st[mode] = sm
	}
	meta := map[string]string{"reputecast_offer": id.String(), "reputecast_workspace": ws.ID.String()}
	if newCoupon || sm.Coupon == "" || sm.MissingAt != "" {
		cp := payments.Coupon{Name: r.Name, Currency: r.Currency, Metadata: meta, Products: products[mode]}
		if r.PercentOff != nil {
			cp.PercentOff = float64(*r.PercentOff)
		}
		if r.AmountOff != nil {
			cp.AmountOff = *r.AmountOff
		}
		switch r.Duration {
		case "months":
			cp.Duration = "repeating"
			if r.DurationMonths != nil {
				cp.Months = int(*r.DurationMonths)
			}
		default:
			cp.Duration = r.Duration
		}
		if cid := couponID(r); cid != "" && !slices.Contains(sm.Coupons, cid) {
			cp.ID = cid
		}
		created, err := off.CreateCoupon(ctx, cp)
		var ae *payments.APIError
		if err != nil && cp.ID != "" && errors.As(err, &ae) && ae.Code == "resource_already_exists" {
			// Its readable id is taken (made by hand in Stripe): let Stripe name it.
			cp.ID = ""
			created, err = off.CreateCoupon(ctx, cp)
		}
		if err != nil {
			fail(err)
			return
		}
		sm.Coupon, sm.MissingAt = created.ID, ""
		if !slices.Contains(sm.Coupons, created.ID) {
			sm.Coupons = append(sm.Coupons, created.ID)
		}
	}
	if r.Kind == KindSignup && r.Status == "live" {
		// The code readers type, or the link applies; a link-only offer's
		// code isn't one to guess.
		code := r.Code
		if !r.ByCode {
			code = r.Code + "-" + strings.ToUpper(strings.ReplaceAll(newID().String(), "-", "")[24:30])
		}
		if sm.Promo != "" {
			_ = off.SetPromotionCodeActive(ctx, sm.Promo, false)
		}
		pc := payments.PromotionCode{Code: code, CouponID: sm.Coupon, ExpiresAt: r.EndsAt, FirstTimeOnly: r.NewOnly, Metadata: meta}
		if r.RedeemLimit != nil {
			// The limit counts redemptions so far; the new code allows the rest.
			used := s.results(ctx, ws, r, st).Redemptions
			pc.MaxRedemptions = max(int(*r.RedeemLimit)-int(used), 1)
		}
		created, err := off.CreatePromotionCode(ctx, pc)
		if err != nil {
			fail(err)
			return
		}
		sm.Promo = created.ID
	}
	b, _ := json.Marshal(st)
	_ = s.Q.OfferSetStripe(ctx, store.OfferSetStripeParams{ID: id, Stripe: b, SyncStatus: "synced", SyncError: "", At: s.now()})
}

// syncError is what saving in the provider answered, for the editor.
func syncError(err error) string {
	var ae *payments.APIError
	if errors.As(err, &ae) {
		return fmt.Sprintf("%d %s", ae.Status, ae.Message)
	}
	if errors.Is(err, payments.ErrUnavailable) {
		return syncUnreachable
	}
	return err.Error()
}

// syncUnreachable is the stored error of a provider that didn't answer.
const syncUnreachable = "unreachable"

// syncErrorIn is a stored sync error as a reader reads it: the provider's
// answer as it came, and an unreachable provider in the reader's language.
func syncErrorIn(ctx context.Context, stored string) string {
	if stored == syncUnreachable {
		return text(ctx, "sync.unreachable", nil)
	}
	return stored
}

// deactivate turns an ended offer's promotion code off in the provider.
func (s *Service) deactivate(ctx context.Context, ws *store.Workspace, r store.OfferGetRow) {
	st := parseStripe(r.Stripe)
	for mode, sm := range st {
		if sm == nil || sm.Promo == "" {
			continue
		}
		client, _, err := s.Earn.ClientFor(ctx, ws.ID, mode)
		if err != nil {
			continue
		}
		if off, ok := client.(payments.Offers); ok {
			if err := off.SetPromotionCodeActive(ctx, sm.Promo, false); err != nil {
				s.log().WarnContext(ctx, "offers: turning the code off", "offer", r.ID, "err", err)
			}
		}
	}
}

// checkCoupon notes a coupon deleted in the provider; it reports whether
// the offer changed.
func (s *Service) checkCoupon(ctx context.Context, ws *store.Workspace, r *store.OfferGetRow, mode string) bool {
	st := parseStripe(r.Stripe)
	sm := st[mode]
	if sm == nil || sm.Coupon == "" {
		return false
	}
	client, _, err := s.Earn.ClientFor(ctx, ws.ID, mode)
	if err != nil {
		return false
	}
	off, ok := client.(payments.Offers)
	if !ok {
		return false
	}
	_, err = off.GetCoupon(ctx, sm.Coupon)
	var ae *payments.APIError
	missing := err != nil && errors.As(err, &ae) && ae.Status == http.StatusNotFound
	if missing == (sm.MissingAt != "") {
		return false
	}
	sm.MissingAt = ""
	if missing {
		sm.MissingAt = s.now().UTC().Format(time.RFC3339)
	}
	b, _ := json.Marshal(st)
	_ = s.Q.OfferSetStripe(ctx, store.OfferSetStripeParams{ID: r.ID, Stripe: b, SyncStatus: r.SyncStatus, SyncError: r.SyncError, At: s.now()})
	return true
}
