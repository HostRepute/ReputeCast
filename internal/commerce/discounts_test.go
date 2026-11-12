package commerce_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/commerce"
	"github.com/hostrepute/reputecast/internal/stores"
)

// The demo's Acme Merch allowed discount codes: it is a source, its
// creator makes codes; without the grant it says why; the connect dialog
// shows the grant and "Allow discount codes" asks for it again.
func TestDiscountSources(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	ws, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	svc := env.App.Commerce()
	srcs, err := svc.DiscountSources(ctx, &ws)
	if err != nil || len(srcs) == 0 {
		t.Fatalf("sources %v %v", srcs, err)
	}
	var merch commerce.DiscountSource
	for _, s := range srcs {
		if s.Name == "Acme Merch" {
			merch = s
		}
	}
	if !merch.Can || merch.Platform != commerce.Shopify || merch.Reason != "" {
		t.Fatalf("Acme Merch %+v", merch)
	}
	dc, err := svc.Discounts(ctx, &ws, merch.StoreID)
	if err != nil {
		t.Fatal(err)
	}
	id, first, err := dc.CreateDiscount(ctx, stores.DiscountSpec{Title: "Welcome 15%", Kind: stores.DiscountPercentage, Value: 15, Currency: "USD"}, "WELCOME-T"+uuid.NewString()[:6])
	if err != nil || id == "" || first == "" || dc.MaxCodesPerCall() != 250 {
		t.Fatalf("create %q %q %v", id, first, err)
	}
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var st commerce.CommerceConnectState
	c.Do("GET", "/workspaces/acme-newsletter/stores/"+merch.StoreID.String()+"/connect", nil).MustStatus(t, 200).JSON(t, &st)
	if !st.DiscountsGranted {
		t.Fatalf("connect state %+v", st)
	}
	var inst commerce.CommerceInstall
	c.Do("POST", "/workspaces/acme-newsletter/stores/"+merch.StoreID.String()+"/allow-discounts", nil).MustStatus(t, 200).JSON(t, &inst)
	if inst.URL == "" {
		t.Fatal("no approval page")
	}

	// Without the grant.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE stores SET scopes = array_remove(array_remove(scopes, 'write_discounts'), 'read_discounts') WHERE id = $1`, merch.StoreID); err != nil {
		t.Fatal(err)
	}
	srcs, _ = svc.DiscountSources(ctx, &ws)
	for _, s := range srcs {
		if s.StoreID == merch.StoreID && (s.Can || s.Reason != commerce.DiscountNotGranted || s.Detail == "") {
			t.Fatalf("not granted %+v", s)
		}
	}
	_, err = svc.Discounts(ctx, &ws, merch.StoreID)
	var p *api.Problem
	if !errors.As(err, &p) || p.Status != 409 || p.Meta["reason"] != commerce.DiscountNotGranted {
		t.Fatalf("not granted: %v", err)
	}
	if _, err := svc.Discounts(ctx, &ws, uuid.Must(uuid.NewV7())); !errors.As(err, &p) || p.Status != 404 {
		t.Fatalf("unknown store: %v", err)
	}
}
