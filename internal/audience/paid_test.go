package audience_test

import (
	"context"
	"net/http"
	"testing"
)

// Paid status in Contacts (Earn, EarnInAudience): the Paid status filter
// and its tiers, the Paid column, and the saved view "Paid members", for
// members with earn:view only.
func TestPaidStatusInContacts(t *testing.T) {
	env, c := demoEnv(t)
	ctx := context.Background()
	var access int64
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(DISTINCT s.contact_id) FROM paid_subscriptions s JOIN contacts c ON c.id = s.contact_id AND c.erased_at IS NULL
		WHERE s.workspace_id = (SELECT id FROM workspaces WHERE slug = 'acme-newsletter') AND s.mode = 'live' AND s.ended_at IS NULL
		AND (s.status IN ('trialing','active','complimentary','gift') OR (s.status = 'past_due' AND NOT s.access_paused))`).Scan(&access); err != nil {
		t.Fatal(err)
	}

	type row struct {
		Email string
		Paid  *struct{ Tier, Status, MemberID string }
	}
	type page struct {
		Total int64
		Items []row
	}
	var sum struct {
		PaidTiers []struct {
			ID, Name, Short string
			Top             bool
		}
	}
	c.Do(http.MethodGet, ws+"/audience", nil).MustStatus(t, 200).JSON(t, &sum)
	if len(sum.PaidTiers) != 2 || sum.PaidTiers[0].Short != "Insider" || sum.PaidTiers[1].Short != "Founding member" || !sum.PaidTiers[1].Top {
		t.Fatalf("paid tiers: %+v", sum.PaidTiers)
	}

	// Any paid tier with access: everyone with access, each with their tier and status.
	var p page
	c.Do(http.MethodGet, ws+"/contacts?paid=with_access&pageSize=100", nil).MustStatus(t, 200).JSON(t, &p)
	if p.Total != access || len(p.Items) == 0 {
		t.Fatalf("with access: %d, want %d", p.Total, access)
	}
	for _, r := range p.Items {
		if r.Paid == nil || r.Paid.Tier == "" || r.Paid.MemberID == "" || r.Paid.Status == "ended" {
			t.Fatalf("a paid member's row: %+v", r)
		}
	}
	// The top tier alone, the trials, the free subscribers and the ended members.
	var top, trials, free, ended page
	c.Do(http.MethodGet, ws+"/contacts?paid=with_access&paidTier="+sum.PaidTiers[1].ID, nil).MustStatus(t, 200).JSON(t, &top)
	c.Do(http.MethodGet, ws+"/contacts?paid=trialing&pageSize=100", nil).MustStatus(t, 200).JSON(t, &trials)
	c.Do(http.MethodGet, ws+"/contacts?paid=free", nil).MustStatus(t, 200).JSON(t, &free)
	c.Do(http.MethodGet, ws+"/contacts?paid=ended&pageSize=100", nil).MustStatus(t, 200).JSON(t, &ended)
	if top.Total == 0 || top.Total >= p.Total {
		t.Errorf("Founding member and above: %d of %d", top.Total, p.Total)
	}
	for _, r := range trials.Items {
		if r.Paid == nil || r.Paid.Status != "trialing" {
			t.Errorf("a trial's row: %+v", r)
		}
	}
	for _, r := range ended.Items {
		if r.Paid == nil || r.Paid.Status != "ended" {
			t.Errorf("an ended member's row: %+v", r)
		}
	}
	var all page
	c.Do(http.MethodGet, ws+"/contacts", nil).MustStatus(t, 200).JSON(t, &all)
	if trials.Total == 0 || ended.Total == 0 || free.Total+p.Total != all.Total {
		t.Errorf("trials %d, ended %d, free %d + with access %d of %d", trials.Total, ended.Total, free.Total, p.Total, all.Total)
	}

	// The saved view counts them.
	type views struct {
		Items []struct {
			Name  string
			Count int64
		}
	}
	var v views
	c.Do(http.MethodGet, ws+"/contacts/views", nil).MustStatus(t, 200).JSON(t, &v)
	found := false
	for _, it := range v.Items {
		if it.Name == "Paid members" {
			found = it.Count == access
		}
	}
	if !found {
		t.Errorf("the Paid members view: %+v", v.Items)
	}

	// Without earn:view (Daniel Park, Content Writer): no paid status anywhere.
	d := env.Client()
	d.SignInDemo("daniel@acme.io")
	d.Do(http.MethodGet, ws+"/contacts?paid=with_access", nil).MustStatus(t, http.StatusForbidden)
	var dv views
	d.Do(http.MethodGet, ws+"/contacts/views", nil).MustStatus(t, 200).JSON(t, &dv)
	for _, it := range dv.Items {
		if it.Name == "Paid members" {
			t.Error("the Paid members view shows without earn:view")
		}
	}
	var dp page
	d.Do(http.MethodGet, ws+"/contacts?pageSize=100", nil).MustStatus(t, 200).JSON(t, &dp)
	for _, r := range dp.Items {
		if r.Paid != nil {
			t.Fatalf("paid status without earn:view: %+v", r)
		}
	}
	d.Do(http.MethodGet, ws+"/audience", nil).MustStatus(t, 200).JSON(t, &sum)
	if len(sum.PaidTiers) != 0 {
		t.Errorf("tiers without earn:view: %+v", sum.PaidTiers)
	}
}
