package demo

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/commerce"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/testdb"
)

// The Shopify sample (brief 4, "Shopify: Acme Merch"): the numbers
// IntegrationsShopify, CommerceStates and CommerceEmails show, read back
// through the commerce service.
func TestStoresSample(t *testing.T) {
	if testing.Short() {
		t.Skip("seeds the whole demo")
	}
	pool := testdb.New(t)
	cfg := config.Defaults()
	box, err := secrets.New(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := Seed(ctx, pool, &cfg, box, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("seed: %v", err)
	}
	q := store.New(pool)
	ws, err := q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	var id string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM stores WHERE name = 'Acme Merch'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	svc := &commerce.Service{Pool: pool, Q: q, Box: box}
	linh, err := q.GetUserByEmail(ctx, "linh@acme.io")
	if err != nil {
		t.Fatal(err)
	}
	p := &authz.Principal{Type: authz.ActorUser, ID: linh.ID, UserID: linh.ID, WorkspaceID: ws.ID, Permissions: authz.NewSet(authz.IntegrationsManage)}
	page, err := svc.Page(ctx, &ws, p, mustUUID(t, id))
	if err != nil {
		t.Fatal(err)
	}
	n := page.Numbers
	if n.Orders != 412 || n.Revenue != 2183600 || n.Credited != 96 || n.CreditedRevenue != 518400 || n.Share != 23.7 || n.ListMembers != 2318 {
		t.Errorf("numbers: %+v", n)
	}
	f := page.Sync
	if f.Products != 38 || f.Accepted != 2318 || f.Known != 1947 || f.New != 371 || f.NotAdded != 3922 || f.OrdersImported != 7880 || f.Months != 12 ||
		f.Webhooks != 11 || f.WebhooksTotal != 11 || f.ImportMinutes != 38 {
		t.Errorf("sync: %+v", f)
	}
	if strings.Join(f.Collections, ", ") != "Apparel, Desk, Stickers, New arrivals" {
		t.Errorf("collections: %v", f.Collections)
	}
	a := page.Attribution
	if a.Days != 5 || a.ByToken != 89 || a.ByEmail != 7 || a.SetBy != "Marcus Chen" || a.TagHost != "shop.acme.io" {
		t.Errorf("attribution: %+v", a)
	}
	want := []struct {
		name            string
		orders, revenue int64
	}{{"October Product Update", 31, 161200}, {"Weekly Digest #141", 18, 90200}, {"Welcome series", 11, 59800}, {"Abandoned cart", 8, 47200}, {"Weekly Digest #140", 6, 31800}}
	if len(a.Rows) != len(want) {
		t.Fatalf("rows: %+v", a.Rows)
	}
	for i, w := range want {
		if r := a.Rows[i]; r.Name != w.name || r.Orders != w.orders || r.Revenue != w.revenue {
			t.Errorf("row %d: %+v, want %+v", i, r, w)
		}
	}
	if a.Others == nil || a.Others.Count != 9 || a.Others.Orders != 22 || a.Others.Revenue != 128200 {
		t.Errorf("others: %+v", a.Others)
	}
	if page.Recovered.Orders != 8 || page.Recovered.Revenue != 47200 {
		t.Errorf("recovered: %+v", page.Recovered)
	}
	// The activity card: the board's seven entries, newest first.
	kinds := []string{}
	for _, it := range page.Activity {
		kinds = append(kinds, it.Kind+" "+it.Order)
	}
	if strings.Join(kinds, ", ") != "order #4421, order #4417, reconciliation , checkout_abandoned , webhook_registered , webhook_missing , order #4409" {
		t.Errorf("activity: %v", kinds)
	}
	// Abandoned cart's trigger: 104 abandoned, 78 with an email, 44 entered.
	var flow string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM automations WHERE name = 'Abandoned cart'`).Scan(&flow); err != nil {
		t.Fatal(err)
	}
	stats, err := svc.TriggerStats(ctx, &ws, mustUUID(t, flow))
	if err != nil {
		t.Fatal(err)
	}
	if stats.Abandoned != 104 || stats.WithEmail != 78 || stats.Entered != 44 || stats.NoConsent != 34 || stats.Orders != 8 || stats.Revenue != 47200 {
		t.Errorf("trigger: %+v", stats)
	}
	// Earn › Overview's card.
	rev, err := svc.RevenueSummary(ctx, &ws)
	if err != nil {
		t.Fatal(err)
	}
	if rev.Revenue != 2183600 || rev.CreditedRevenue != 518400 || rev.Share != 23.7 {
		t.Errorf("revenue: %+v", rev)
	}
	// The Audience boards' 48,213 contacts: the store's customers are sample
	// contacts, Grace Liu among them.
	var contacts, grace int64
	if err := pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE first_name = 'Grace' AND last_name = 'Liu') FROM contacts
		WHERE workspace_id = $1 AND erased_at IS NULL`, ws.ID).Scan(&contacts, &grace); err != nil {
		t.Fatal(err)
	}
	if contacts != 48213 || grace != 1 {
		t.Errorf("contacts: %d, Grace Liu %d", contacts, grace)
	}
	// The October Product Update carries its grid as sent; a grid that
	// names the sticker pack deleted on 13 Oct is flagged.
	var opu string
	if err := pool.QueryRow(ctx, `SELECT mjml FROM campaigns WHERE name = 'October Product Update'`).Scan(&opu); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(opu, `rc-name="Product grid"`) || !strings.Contains(opu, "Feature Flag sticker pack") {
		t.Errorf("October Product Update's grid is missing")
	}
	block, err := svc.RenderBlock(ctx, &ws, commerce.CommerceBlock{Store: id, Mode: "chosen", Products: []string{"8841203", "8841215"}, Button: "Shop now"})
	if err != nil {
		t.Fatal(err)
	}
	issues := svc.RemovedProducts(ctx, contentcheck.Input{WorkspaceID: ws.ID, HTML: block.HTML})
	if len(issues) != 1 || issues[0].Title != "Feature Flag sticker pack is no longer in Acme Merch" {
		t.Errorf("removed product: %+v", issues)
	}
	// Sam Rivera's order is on his profile, credited by email.
	var sam string
	if err := pool.QueryRow(ctx, `SELECT o.matched || ' ' || c.name FROM store_orders o JOIN campaigns c ON c.id = o.campaign_id WHERE o.name = '#4409'`).Scan(&sam); err != nil {
		t.Fatal(err)
	}
	if sam != "email October Product Update" {
		t.Errorf("#4409: %s", sam)
	}
}

func mustUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
