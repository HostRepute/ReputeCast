package referrals_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/referrals"
	"github.com/hostrepute/reputecast/internal/store"
)

const testOffer = "01a1aaaa-0000-7000-8000-000000000001"

// testCoupons is the payment side of coupon rewards: one offer, one code,
// and claim links like a referrer's own upgrade link.
type testCoupons struct{}

func (testCoupons) Status(context.Context, *store.Workspace) (referrals.CouponStatus, error) {
	return referrals.CouponStatus{Connected: true, Account: "Grow Test"}, nil
}

func (testCoupons) Offers(context.Context, *store.Workspace) ([]referrals.CouponOffer, error) {
	return []referrals.CouponOffer{{ID: testOffer, Name: "Referral reward"}}, nil
}

func (c testCoupons) Issue(ctx context.Context, ws *store.Workspace, _ string, contact uuid.UUID, days int) (referrals.IssuedCoupon, error) {
	code := "INS-QM8XRT4PH7K2"
	return referrals.IssuedCoupon{Code: code, ClaimURL: c.ClaimURL(ctx, ws, contact, code), ExpiresAt: time.Now().AddDate(0, 0, days)}, nil
}

func (testCoupons) ClaimURL(_ context.Context, _ *store.Workspace, contact uuid.UUID, code string) string {
	return "https://news.grow.test/upgrade?r=" + contact.String() + "&offer=" + code
}

// A coupon reward's claim link is its referrer's own upgrade link, made
// when it is shown: on the hub and when copied. A reward issued before
// codes worked only through that link kept upgrade?offer=…, which shows
// anyone the normal prices; it gets the link that works.
func TestCouponClaimLinkIsTheReferrersOwn(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	f.s.Coupons = func() referrals.Coupons { return testCoupons{} }
	admin := api.WithPrincipal(ctx, &authz.Principal{Type: authz.ActorUser, ID: f.owner.ID, UserID: f.owner.ID, WorkspaceID: f.ws.ID, Permissions: authz.All()})
	p, err := f.s.Create(ctx, &f.ws, referrals.ReferralCreateInput{ListID: f.list.String()})
	if err != nil {
		t.Fatal(err)
	}
	pid := uuid.MustParse(p.ID)
	if _, err := f.s.CreateMilestone(admin, &f.ws, pid, referrals.ReferralMilestoneInput{Count: 1, RewardName: "A free month", Type: referrals.RewardCoupon,
		CouponOfferID: testOffer, CouponValidDays: 30}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.TurnOn(ctx, &f.ws, pid); err != nil {
		t.Fatal(err)
	}
	f.confirm(f.signup(f.code(p.ID, f.sam), "one@contoso.example", ""))
	f.advance(7*24*time.Hour + time.Minute)
	if _, err := f.s.ProcessHolds(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.DeliverDue(ctx); err != nil {
		t.Fatal(err)
	}
	var rid uuid.UUID
	var status, stored string
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT id, status, claim_url FROM referral_rewards WHERE contact_id = $1`, f.sam).Scan(&rid, &status, &stored); err != nil {
		t.Fatal(err)
	}
	own := testCoupons{}.ClaimURL(ctx, &f.ws, f.sam, "INS-QM8XRT4PH7K2")
	if status != referrals.RewardSent || stored != own {
		t.Fatalf("issued: %s %q", status, stored)
	}
	claimOnHub := func() string {
		t.Helper()
		hub, err := f.s.Hub(ctx, &f.ws, f.sam, f.s.HubToken(f.ws.ID, f.sam, f.clock()), nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range hub.Rewards {
			if r.Code == "INS-QM8XRT4PH7K2" {
				return r.ClaimURL
			}
		}
		t.Fatalf("no coupon on the hub: %+v", hub.Rewards)
		return ""
	}

	// Issued before codes needed the referrer's link.
	old := "https://news.grow.test/upgrade?offer=INS-QM8XRT4PH7K2"
	f.exec(`UPDATE referral_rewards SET claim_url = $2 WHERE id = $1`, rid, old)
	if got := claimOnHub(); got != own {
		t.Fatalf("the hub's claim link: %q", got)
	}
	if link, err := f.s.ClaimLink(admin, &f.ws, rid); err != nil || link.URL != own {
		t.Fatalf("the copied claim link: %+v %v", link, err)
	}
	// Without the payment side the stored link is all there is.
	f.s.Coupons = nil
	if got := claimOnHub(); got != old {
		t.Fatalf("without the payment side: %q", got)
	}
}
