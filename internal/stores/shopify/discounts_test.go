package shopify_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/stores"
	"github.com/hostrepute/reputecast/internal/stores/shopify"
	"github.com/hostrepute/reputecast/internal/stores/shopify/shopifytest"
)

func discountClient(t *testing.T, f *shopifytest.Fake) *shopify.Client {
	t.Helper()
	f.IssueToken(token)
	srv := f.Server()
	t.Cleanup(srv.Close)
	c, err := shopify.New(stores.Config{Domain: f.Domain, BaseURL: srv.URL, Token: token, HTTP: srv.Client(), RetryWait: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// A discount is created with its first code; codes are added 250 a call
// in Shopify's background job, followed until it's done; refused codes
// are reported; codes are removed and the discount deactivated.
func TestDiscountCodes(t *testing.T) {
	ctx := context.Background()
	f := shopifytest.New()
	f.Scopes = append(f.Scopes, shopify.DiscountScopes...)
	f.BulkPolls = 2
	c := discountClient(t, f)
	var dc stores.DiscountCreator = c
	if dc.MaxCodesPerCall() != 250 {
		t.Fatalf("max %d", dc.MaxCodesPerCall())
	}
	ends := time.Date(2026, 11, 28, 4, 59, 0, 0, time.UTC)
	spec := stores.DiscountSpec{Title: "Welcome 15%", Kind: stores.DiscountPercentage, Value: 15, Currency: "USD", MinimumCents: 5000, EndsAt: &ends, CollectionIDs: []string{"501"}}
	id, first, err := dc.CreateDiscount(ctx, spec, "WELCOME-SAMPLE1")
	if err != nil || id == "" || first == "" {
		t.Fatalf("create: %q %q %v", id, first, err)
	}
	d := f.Discounts[0]
	in := d.Input
	gets := in["customerGets"].(map[string]any)
	if d.Kind != "DiscountCodeBasic" || in["usageLimit"] != float64(1) || in["appliesOncePerCustomer"] != true || in["endsAt"] != "2026-11-28T04:59:00Z" ||
		gets["value"].(map[string]any)["percentage"] != 0.15 ||
		gets["items"].(map[string]any)["collections"].(map[string]any)["add"].([]any)[0] != "gid://shopify/Collection/501" ||
		in["minimumRequirement"].(map[string]any)["subtotal"].(map[string]any)["greaterThanOrEqualToSubtotal"] != "50.00" {
		t.Fatalf("input %+v", in)
	}
	if _, _, err := dc.CreateDiscount(ctx, spec, "WELCOME-SAMPLE1"); err == nil {
		t.Fatal("a code already taken was accepted")
	}

	codes := make([]string, 251)
	for i := range codes {
		codes[i] = "WELCOME-" + string(rune('A'+i%26)) + string(rune('A'+i/26))
	}
	if _, err := dc.AddCodes(ctx, id, spec, codes); err == nil {
		t.Fatal("251 codes in one call")
	}
	batch := append(codes[:3:3], "WELCOME-SAMPLE1")
	job, err := dc.AddCodes(ctx, id, spec, batch)
	if err != nil || job.JobID == "" || job.Done {
		t.Fatalf("add: %+v %v", job, err)
	}
	for range 2 {
		st, err := dc.CodesStatus(ctx, id, job.JobID, batch)
		if err != nil || st.Done {
			t.Fatalf("still running: %+v %v", st, err)
		}
	}
	st, err := dc.CodesStatus(ctx, id, job.JobID, batch)
	if err != nil || !st.Done || len(st.IDs) != 3 || len(st.Failed) != 1 || st.Failed[0] != "WELCOME-SAMPLE1" {
		t.Fatalf("done: %+v %v", st, err)
	}
	if adds := f.BulkAdds(); len(adds) != 1 || len(adds[0]) != 4 {
		t.Fatalf("bulk adds %v", adds)
	}

	// Removed by id, and by code alone (looked up).
	if err := dc.RemoveCodes(ctx, id, []string{batch[0], batch[1], "WELCOME-NOPE"}, []string{st.IDs[batch[0]], "", ""}); err != nil {
		t.Fatal(err)
	}
	if got := f.Codes(id); len(got) != 2 || got[0] != "WELCOME-SAMPLE1" || got[1] != batch[2] {
		t.Fatalf("left %v", got)
	}
	if err := dc.DisableDiscount(ctx, id); err != nil || f.Discounts[0].Active {
		t.Fatalf("deactivate: %v", err)
	}

	// Free shipping has no value or items.
	if _, _, err := dc.CreateDiscount(ctx, stores.DiscountSpec{Title: "Free shipping", Kind: stores.DiscountFreeShipping}, "SHIP-FREE1"); err != nil {
		t.Fatal(err)
	}
	if fs := f.Discounts[1]; fs.Kind != "DiscountCodeFreeShipping" || fs.Input["destination"] == nil || fs.Input["customerGets"] != nil {
		t.Fatalf("free shipping %+v", fs.Input)
	}
}

// Without write_discounts every call is refused as a missing scope;
// throttling is the contract's RateLimited.
func TestDiscountCodesRefused(t *testing.T) {
	ctx := context.Background()
	f := shopifytest.New()
	c := discountClient(t, f)
	spec := stores.DiscountSpec{Title: "Welcome 15%", Kind: stores.DiscountPercentage, Value: 15, Currency: "USD"}
	if _, _, err := c.CreateDiscount(ctx, spec, "WELCOME-1"); !errors.Is(err, stores.ErrPermission) {
		t.Fatalf("without the scope: %v", err)
	}
	f.Scopes = append(f.Scopes, shopify.DiscountScopes...)
	f.Throttle(1)
	if _, _, err := c.CreateDiscount(ctx, spec, "WELCOME-1"); err == nil {
		t.Fatal("throttled call succeeded")
	} else if _, ok := stores.IsRateLimited(err); !ok {
		t.Fatalf("throttled: %v", err)
	}
}
