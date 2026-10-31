package app

import (
	"context"
	"errors"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/liquid"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/offers"
	"github.com/hostrepute/reputecast/internal/referrals"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/store"
)

// Grow › Referrals (M25): programs, counting, milestones and rewards, the
// review queue, the Referral and Reward blocks.

var (
	referralsMu       sync.Mutex
	referralsServices = map[*App]*referrals.Service{}
)

func (a *App) referrals() *referrals.Service {
	referralsMu.Lock()
	defer referralsMu.Unlock()
	if s, ok := referralsServices[a]; ok {
		return s
	}
	s := &referrals.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log, Links: a.Links,
		Pages: a.growPages,
		Mail:  func() mailer.Sender { return a.Mail },
		// Milestone emails go out on the Automations stream through the
		// sending engine, outside any flow.
		Emails: func() referrals.MilestoneSender { return milestoneMailer{a: a} },
		// Stripe coupons: Earn's Stripe connection and its offers of the
		// kind "Referral reward".
		Coupons:      referralCoupons(a),
		AllowPrivate: func() bool { return a.Cfg != nil && a.Cfg.Webhooks.AllowPrivateNetworks },
		Jobs: func() referrals.Inserter {
			if a.Jobs == nil {
				return nil
			}
			return a.Jobs
		},
		Base: a.URLs.Base,
	}
	referralsServices[a] = s
	return s
}

// referralCoupons is the Stripe coupon seam of rewards: Earn's "Referral
// reward" offers issue the codes. It's set in init, because Earn's service
// reaches the referrals service (an initializer would be a cycle). A nil
// seam shows "Connect Stripe".
var referralCoupons func(a *App) func() referrals.Coupons

func init() {
	referralCoupons = func(a *App) func() referrals.Coupons {
		return func() referrals.Coupons { return offerCoupons{earnCoupons: earnCoupons{a: a}, s: offerService(a)} }
	}
}

// earnCoupons is whether Earn's Stripe connection can issue coupons.
type earnCoupons struct{ a *App }

// Status implements referrals.Coupons: a connected Stripe account whose key
// still works.
func (c earnCoupons) Status(ctx context.Context, ws *store.Workspace) (referrals.CouponStatus, error) {
	var name string
	err := c.a.Pool.QueryRow(ctx, `SELECT account_name FROM payment_connections
		WHERE workspace_id = $1 AND status = 'connected' AND key_rejected_at IS NULL ORDER BY (mode = 'live') DESC LIMIT 1`, ws.ID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return referrals.CouponStatus{}, nil
	}
	if err != nil {
		return referrals.CouponStatus{}, err
	}
	return referrals.CouponStatus{Connected: true, Account: name}, nil
}

// offerCoupons issues a coupon reward's single-use code from one of
// Earn's live "Referral reward" offers.
type offerCoupons struct {
	earnCoupons
	s *offers.Service
}

// Offers implements referrals.Coupons.
func (c offerCoupons) Offers(ctx context.Context, ws *store.Workspace) ([]referrals.CouponOffer, error) {
	list, err := c.s.ReferralOffers(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := make([]referrals.CouponOffer, 0, len(list))
	for _, o := range list {
		out = append(out, referrals.CouponOffer{ID: o.ID.String(), Name: o.Name, Discount: o.Discount, AppliesTo: o.AppliesTo, Issued: o.Issued, Redeemed: o.Redeemed})
	}
	return out, nil
}

// Issue implements referrals.Coupons; an error leaves the reward waiting
// to try again.
func (c offerCoupons) Issue(ctx context.Context, ws *store.Workspace, offerID string, contact uuid.UUID, validDays int) (referrals.IssuedCoupon, error) {
	id, err := uuid.Parse(offerID)
	if err != nil {
		return referrals.IssuedCoupon{}, err
	}
	code, err := c.s.IssueReferralCode(ctx, ws, id, contact, validDays)
	if err != nil {
		return referrals.IssuedCoupon{}, err
	}
	return referrals.IssuedCoupon{Code: code.Code, ClaimURL: code.ClaimURL, ExpiresAt: code.ExpiresAt}, nil
}

// ClaimURL implements referrals.Coupons.
func (c offerCoupons) ClaimURL(ctx context.Context, ws *store.Workspace, contact uuid.UUID, code string) string {
	return c.s.ReferralClaimURL(ctx, ws, contact, code)
}

// growPages are a workspace's public page links (its custom domain's when
// verified).
func (a *App) growPages(ctx context.Context, ws *store.Workspace) (forms.PageLinks, error) {
	fs := a.formsService()
	set, err := fs.LoadSettings(ctx, ws)
	if err != nil {
		return forms.PageLinks{}, err
	}
	return fs.Links(ws, set), nil
}

// milestoneMailer queues milestone emails through the sending engine.
type milestoneMailer struct{ a *App }

// QueueMilestoneEmail implements referrals.MilestoneSender.
func (m milestoneMailer) QueueMilestoneEmail(ctx context.Context, tx pgx.Tx, ws uuid.UUID, emailID, contactID uuid.UUID, to string) (uuid.UUID, error) {
	q, err := engine.AutomationMailer{E: sendingEngine(m.a)}.QueueAutomationEmail(ctx, tx, automations.OutgoingEmail{
		WorkspaceID: ws, ContactID: contactID, To: to, EmailID: emailID, NodeID: "milestone",
	})
	return q.MessageID, err
}

// recipientData are the per-reader variables of blocks: Grow's referral.*
// and reward.*, and recommendations.* (wire_recommendations.go).
func recipientData(a *App) []liquid.RecipientData {
	out := []liquid.RecipientData{a.referrals()}
	for _, f := range recipientDataRegistrars {
		if d := f(a); d != nil {
			out = append(out, d)
		}
	}
	return out
}

var recipientDataRegistrars []func(*App) liquid.RecipientData

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		referrals.Register(h, a.referrals())
		referrals.RegisterContact(h, a.referrals(), referralPartnerEvents(a))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		referrals.RegisterJobs(a.JobReg, a.referrals())
	})
	signupHookRegistrars = append(signupHookRegistrars, func(a *App) forms.SignupHook {
		return referrals.Hook{S: a.referrals}
	})
}

// referralPartnerEvents adds partner requests to a contact's Grow timeline
// (wire_recommendations.go sets it).
var referralPartnerEvents = func(*App) func() referrals.PartnerEvents { return nil }
