package commerce

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/stores"
	"github.com/hostrepute/reputecast/internal/stores/shopify"
)

// Discount codes made in a connected store (M59 G-22), for the coupons
// domain: which stores can make them and why the others can't, the store's
// DiscountCreator, and Shopify's "Allow discount codes", which asks the
// merchant to approve the app again with write_discounts and
// read_discounts.

// Why a store can't create discount codes (DiscountSource.Reason and a
// refusal's Meta["reason"]).
const (
	DiscountNotConnected = "not_connected"
	DiscountNotGranted   = "not_granted"
	DiscountUnsupported  = "unsupported"
	DiscountReadOnly     = "read_only"
)

// DiscountSource is a store as a source of discount codes.
type DiscountSource struct {
	StoreID                uuid.UUID
	Platform               string // shopify or woocommerce
	Name, Domain, Currency string
	Can                    bool
	// Reason is why not ("" when Can); Detail says it in the request's
	// language.
	Reason, Detail string
}

// discountsGranted reports whether the app may write discounts.
func discountsGranted(st store.Store) bool {
	return st.Platform == Shopify && slices.Contains(st.Scopes, shopify.DiscountScopes[0]) && slices.Contains(st.Scopes, shopify.DiscountScopes[1])
}

// discountReason is why a store can't create codes now ("" when it can),
// from what ReputeCast knows of it (no call to the store).
func discountReason(st store.Store) string {
	switch {
	case st.Status != StatusConnected && st.Status != StatusImporting:
		return DiscountNotConnected
	case st.Platform == Shopify && !discountsGranted(st):
		return DiscountNotGranted
	}
	d, ok := stores.Lookup(st.Platform)
	if !ok || d.Open == nil {
		return DiscountUnsupported
	}
	return ""
}

// discountText says why a store can't create discount codes.
func discountText(l i18n.Lang, reason string, st store.Store) string {
	args := i18n.Args{"name": st.Name, "platform": platformName(st.Platform)}
	switch reason {
	case DiscountNotConnected:
		return mStoreNotConnected.Text(l, args)
	case DiscountNotGranted:
		return mDiscountsNotGranted.Text(l, args)
	case DiscountReadOnly:
		return mDiscountsReadOnly.Text(l, args)
	}
	return mDiscountsUnsupported.Text(l, args)
}

// discountProblem refuses discount codes for a reason (409, Meta reason).
func discountProblem(ctx context.Context, reason string, st store.Store) *api.Problem {
	return api.Err(http.StatusConflict, api.CodeConflict, discountText(i18n.LangFrom(ctx), reason, st)).WithMeta("reason", reason, "storeId", st.ID.String())
}

// DiscountSources lists the workspace's stores with whether each can
// create discount codes now, and why not, without calling the stores.
func (s *Service) DiscountSources(ctx context.Context, ws *store.Workspace) ([]DiscountSource, error) {
	rows, err := s.Q.CommerceStores(ctx, ws.ID)
	if err != nil {
		return nil, err
	}
	l := i18n.LangFrom(ctx)
	out := make([]DiscountSource, 0, len(rows))
	for _, st := range rows {
		src := DiscountSource{StoreID: st.ID, Platform: st.Platform, Name: st.Name, Domain: st.Domain, Currency: st.Currency}
		if src.Reason = discountReason(st); src.Reason == "" {
			src.Can = true
		} else {
			src.Detail = discountText(l, src.Reason, st)
		}
		out = append(out, src)
	}
	return out, nil
}

// Discounts opens a connected store's discount codes. It refuses with a
// 409 problem whose detail says why and whose Meta["reason"] is one of the
// Discount* reasons; the creator it returns turns the store refusing a
// write into the same problem (a read-only WooCommerce key, a Shopify
// scope taken back). Rate limits stay *stores.RateLimited.
func (s *Service) Discounts(ctx context.Context, ws *store.Workspace, storeID uuid.UUID) (stores.DiscountCreator, error) {
	st, err := s.Q.CommerceStore(ctx, store.CommerceStoreParams{ID: storeID, WorkspaceID: ws.ID})
	if err != nil {
		return nil, s.notFound(ctx, err)
	}
	if reason := discountReason(st); reason != "" {
		return nil, discountProblem(ctx, reason, st)
	}
	c, err := s.client(st)
	if err != nil {
		return nil, err
	}
	dc, ok := c.(stores.DiscountCreator)
	if !ok {
		return nil, discountProblem(ctx, DiscountUnsupported, st)
	}
	return refusals{dc: dc, st: st}, nil
}

// refusals is a store's DiscountCreator whose refused writes are problems.
type refusals struct {
	dc stores.DiscountCreator
	st store.Store
}

func (r refusals) refused(ctx context.Context, err error) error {
	if !errors.Is(err, stores.ErrPermission) {
		return err
	}
	reason := DiscountReadOnly
	if r.st.Platform == Shopify {
		reason = DiscountNotGranted
	}
	return discountProblem(ctx, reason, r.st)
}

func (r refusals) MaxCodesPerCall() int { return r.dc.MaxCodesPerCall() }

func (r refusals) CreateDiscount(ctx context.Context, spec stores.DiscountSpec, firstCode string) (string, string, error) {
	id, code, err := r.dc.CreateDiscount(ctx, spec, firstCode)
	return id, code, r.refused(ctx, err)
}

func (r refusals) AddCodes(ctx context.Context, discountID string, spec stores.DiscountSpec, codes []string) (stores.DiscountCodes, error) {
	out, err := r.dc.AddCodes(ctx, discountID, spec, codes)
	return out, r.refused(ctx, err)
}

func (r refusals) CodesStatus(ctx context.Context, discountID, jobID string, codes []string) (stores.DiscountCodes, error) {
	out, err := r.dc.CodesStatus(ctx, discountID, jobID, codes)
	return out, r.refused(ctx, err)
}

func (r refusals) RemoveCodes(ctx context.Context, discountID string, codes []string, ids []string) error {
	return r.refused(ctx, r.dc.RemoveCodes(ctx, discountID, codes, ids))
}

func (r refusals) DisableDiscount(ctx context.Context, discountID string) error {
	return r.refused(ctx, r.dc.DisableDiscount(ctx, discountID))
}

// AllowDiscounts is "Allow discount codes" on a connected Shopify store:
// Shopify's approval page asking for the discount scopes too. The store
// keeps syncing meanwhile; the callback stores the scopes granted.
func (s *Service) AllowDiscounts(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (CommerceInstall, error) {
	st, err := s.load(ctx, ws, id)
	if err != nil {
		return CommerceInstall{}, err
	}
	if st.Platform != Shopify || st.Status != StatusConnected || st.ClientID == "" || len(st.CredentialsSealed) == 0 {
		return CommerceInstall{}, s.wrongStep(ctx, st)
	}
	return s.approval(ctx, st, p, installScopes(st, true))
}

// discountsRow is the store page's "Discount codes" row: Shopify asks for
// the permission (who allowed it is the first approval that granted it)
// and the pools that create codes in the store.
func (s *Service) discountsRow(ctx context.Context, st store.Store) (CommerceStoreDiscounts, error) {
	out := CommerceStoreDiscounts{Supported: st.Platform == Shopify, Granted: discountsGranted(st), Pools: []string{}}
	if out.Granted {
		if by, err := s.Q.CouponStoreAllowed(ctx, st.ID.String()); err == nil {
			out.AllowedBy, out.AllowedAt = by.ActorName, &by.OccurredAt
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return out, err
		}
	}
	pools, err := s.Q.CouponStorePools(ctx, &st.ID)
	if err != nil {
		return out, err
	}
	out.Pools = append(out.Pools, pools...)
	return out, nil
}
