package woocommerce_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/stores"
	"github.com/hostrepute/reputecast/internal/stores/woocommerce"
	"github.com/hostrepute/reputecast/internal/stores/woocommerce/wootest"
)

// Each code is a single-use coupon, made 100 a batch and confirmed at
// once; a taken code is reported; removal deletes the coupons; a
// read-only key is refused as missing access.
func TestDiscountCodes(t *testing.T) {
	ctx := context.Background()
	f := wootest.New()
	srv := f.Server()
	t.Cleanup(srv.Close)
	c, err := woocommerce.New(stores.Config{Domain: "store.example.com", BaseURL: srv.URL, Key: f.Key, Secret: f.Secret, HTTP: srv.Client(), RetryWait: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	var dc stores.DiscountCreator = c
	ends := time.Date(2026, 11, 28, 4, 59, 0, 0, time.UTC)
	spec := stores.DiscountSpec{Title: "Welcome 15%", Kind: stores.DiscountPercentage, Value: 15, Currency: "USD", MinimumCents: 5000, EndsAt: &ends, CollectionIDs: []string{"31"}}
	id, first, err := dc.CreateDiscount(ctx, spec, "WELCOME-SAMPLE1")
	if err != nil || id != "" || first == "" || dc.MaxCodesPerCall() != 100 {
		t.Fatalf("create: %q %q %v", id, first, err)
	}
	in := f.Coupons[0].Input
	if in["discount_type"] != "percent" || in["amount"] != "15" || in["usage_limit"] != float64(1) || in["usage_limit_per_user"] != float64(1) ||
		in["individual_use"] != true || in["minimum_amount"] != "50.00" || in["date_expires_gmt"] != "2026-11-28T04:59:00" || in["product_categories"].([]any)[0] != float64(31) {
		t.Fatalf("coupon %+v", in)
	}
	if _, err := dc.AddCodes(ctx, "", spec, make([]string, 101)); err == nil {
		t.Fatal("101 coupons in one batch")
	}
	got, err := dc.AddCodes(ctx, "", spec, []string{"WELCOME-A", "WELCOME-B", "WELCOME-SAMPLE1"})
	if err != nil || !got.Done || len(got.IDs) != 2 || len(got.Failed) != 1 || got.Failed[0] != "WELCOME-SAMPLE1" {
		t.Fatalf("add: %+v %v", got, err)
	}
	if b := f.Batches(); len(b) != 2 || len(b[1]) != 3 {
		t.Fatalf("batches %v", b)
	}
	if err := dc.RemoveCodes(ctx, "", []string{"WELCOME-A", "WELCOME-B"}, []string{got.IDs["WELCOME-A"], ""}); err != nil {
		t.Fatal(err)
	}
	if left := f.CouponCodes(); len(left) != 1 || left[0] != "welcome-sample1" {
		t.Fatalf("left %v", left)
	}
	free := stores.DiscountSpec{Title: "Free shipping", Kind: stores.DiscountFreeShipping}
	if _, _, err := dc.CreateDiscount(ctx, free, "SHIP-1"); err != nil || f.Coupons[1].Input["free_shipping"] != true || f.Coupons[1].Input["discount_type"] != "fixed_cart" {
		t.Fatalf("free shipping: %v %+v", err, f.Coupons[1].Input)
	}

	f.ReadOnly = true
	if _, _, err := dc.CreateDiscount(ctx, spec, "WELCOME-C"); !errors.Is(err, stores.ErrPermission) {
		t.Fatalf("read-only key: %v", err)
	}
	f.ReadOnly = false
	f.Throttle(1)
	if _, err := dc.AddCodes(ctx, "", spec, []string{"WELCOME-D"}); err == nil {
		t.Fatal("throttled batch succeeded")
	} else if _, ok := stores.IsRateLimited(err); !ok {
		t.Fatalf("throttled: %v", err)
	}
}
