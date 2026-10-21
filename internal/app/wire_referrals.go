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
		// Stripe coupons: Earn's Stripe connection (M26); offers of the
		// kind "Referral reward" come with Earn's offers (M27).
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

// referralCoupons is the Stripe coupon seam of rewards. By default it reads
// Earn's Stripe connection (M26) and has no offers to issue yet; Earn's
// offers (M27) replace it from their wire file's init. A nil seam shows
// "Connect Stripe".
var referralCoupons = func(a *App) func() referrals.Coupons {
	return func() referrals.Coupons { return earnCoupons{a: a} }
}

// earnCoupons is the coupon reward's view of Earn before its offers exist:
// whether Stripe is connected, no "Referral reward" offers, nothing to
// issue (rewards wait and retry).
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

// Offers implements referrals.Coupons: none until Earn's offers.
func (earnCoupons) Offers(context.Context, *store.Workspace) ([]referrals.CouponOffer, error) {
	return nil, nil
}

// errNoCouponOffers: a coupon can't be issued without Earn's offers.
var errNoCouponOffers = errors.New("no Referral reward offer can issue a coupon yet: offers come with Earn's offers")

// Issue implements referrals.Coupons: the reward waits and tries again.
func (earnCoupons) Issue(context.Context, *store.Workspace, string, uuid.UUID, int) (referrals.IssuedCoupon, error) {
	return referrals.IssuedCoupon{}, errNoCouponOffers
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
