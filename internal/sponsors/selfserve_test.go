package sponsors_test

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/sponsors"
)

// The plan's self-serve test (M65): a request doesn't hold its issue; an
// approval holds it until the pay-by date, so approving another request for
// it is refused; Stripe's payment books it; an approval left unpaid
// expires and the issue opens again.
func TestSelfServeBooking(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("sofia@acme.io") // Editor: sponsors:manage
	w, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	svc := env.App.Sponsors()
	page, err := svc.Page(ctx, &w)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Slots) == 0 || len(page.Slots[0].Issues) < 2 {
		t.Fatalf("the media kit offers nothing: %+v", page.Slots)
	}
	slot := page.Slots[0]
	first, second := slot.Issues[0].Number, slot.Issues[1].Number
	request := func(company, email string, issue int) string {
		t.Helper()
		out, err := svc.Request(ctx, &w, sponsors.RequestInput{Company: company, Name: "Nia Park", Email: email, Slot: slot.ID.String(), Issue: strconv.Itoa(issue),
			Headline: "Notes that write themselves", Text: "Northwind turns meetings into notes.", Button: "Try it free", Link: "https://northwind.example", Lang: "en"})
		if err != nil || len(out.Errors) != 0 || out.Token == "" {
			t.Fatalf("request from %s: %+v %v", company, out, err)
		}
		var id string
		if err := env.App.Pool.QueryRow(ctx, `SELECT id::text FROM sponsor_bookings WHERE workspace_id = $1 AND request->>'company' = $2`, w.ID, company).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	status := func(id string) string {
		t.Helper()
		var s string
		if err := env.App.Pool.QueryRow(ctx, `SELECT status FROM sponsor_bookings WHERE id = $1`, id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	issueBooking := func(n int) string {
		t.Helper()
		var d struct{ BookingID string }
		c.Do(http.MethodGet, ws+"/sponsor-slots/"+slot.ID.String()+"/issues/"+strconv.Itoa(n), nil).MustStatus(t, 200).JSON(t, &d)
		return d.BookingID
	}

	// Two sponsors ask for the same issue: requests, and the issue is still open.
	one := request("Northwind Labs", "nia@northwind.example", first)
	two := request("Contoso Labs", "nia@contoso.example", first)
	if status(one) != "requested" || status(two) != "requested" || issueBooking(first) != "" {
		t.Fatalf("requests: %s, %s; the issue's booking %q", status(one), status(two), issueBooking(first))
	}
	var list sponsors.SponsorRequestList
	c.Do(http.MethodGet, ws+"/sponsor-requests", nil).MustStatus(t, 200).JSON(t, &list)
	listed := 0
	for _, r := range list.Items {
		if (r.ID == one || r.ID == two) && r.Status == "requested" && r.IssueState == "open" {
			listed++
		}
	}
	if listed != 2 {
		t.Fatalf("the Requests tab: %+v", list.Items)
	}

	// Approving holds the issue until the pay-by date.
	var d sponsors.SponsorRequestDetail
	c.Do(http.MethodPost, ws+"/sponsor-requests/"+one+"/approve", map[string]any{}).MustStatus(t, 200).JSON(t, &d)
	if d.Status != "awaiting_payment" || d.PaymentDueAt == nil || issueBooking(first) != one {
		t.Fatalf("approved: %+v; the issue's booking %q", d.SponsorRequestItem, issueBooking(first))
	}
	if want := time.Now().AddDate(0, 0, slot.PayDays); d.PaymentDueAt.After(want.Add(time.Hour)) {
		t.Fatalf("pay by %v, after the slot's %d days", d.PaymentDueAt, slot.PayDays)
	}
	if auditCount(t, env, "sponsor_booking.approved", one) != 1 {
		t.Fatal("the approval isn't in the audit log")
	}
	// The other request can't have the held issue.
	c.Do(http.MethodPost, ws+"/sponsor-requests/"+two+"/approve", map[string]any{}).MustStatus(t, 409)
	if status(two) != "requested" {
		t.Fatalf("the refused approval changed the request: %s", status(two))
	}

	// Stripe's payment books it: paid by Stripe, the creative ready.
	tx, err := env.App.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	out, handled, err := svc.Consume(ctx, tx, earn.PaymentEvent{Workspace: &w, Mode: payments.Live, Type: "checkout.session.completed",
		Payment: payments.Payment{Kind: payments.PaymentCompleted, Paid: true, AmountCents: d.PriceCents, Currency: d.Currency, SessionID: "cs_selfserve_1",
			Created: time.Now(), Metadata: map[string]string{sponsors.MetaBooking: one}}})
	if err != nil || !handled || out.Outcome != "sponsor_paid" {
		_ = tx.Rollback(ctx)
		t.Fatalf("payment: %+v %v %v", out, handled, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var b sponsors.SponsorBookingDetail
	c.Do(http.MethodGet, ws+"/sponsor-bookings/"+one, nil).MustStatus(t, 200).JSON(t, &b)
	if b.Status != "booked" || b.Payment.Status != "paid" || b.Payment.Source != "stripe" || b.CreativeSt != "ready" || b.Request == nil || b.Request.ApprovedBy == "" {
		t.Fatalf("booked: %s, payment %+v, creative %s, request %+v", b.Status, b.Payment, b.CreativeSt, b.Request)
	}

	// The other sponsor is approved for the next issue and never pays: it expires.
	c.Do(http.MethodPost, ws+"/sponsor-requests/"+two+"/approve", map[string]any{"issue": second}).MustStatus(t, 200).JSON(t, &d)
	if d.Status != "awaiting_payment" || d.Issue != second || issueBooking(second) != two {
		t.Fatalf("approved for #%d: %+v", second, d.SponsorRequestItem)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE sponsor_bookings SET payment_due_at = now() - interval '1 minute' WHERE id = $1`, two); err != nil {
		t.Fatal(err)
	}
	if err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if status(two) != "expired" || issueBooking(second) != "" {
		t.Fatalf("past its pay-by date: %s; the issue's booking %q", status(two), issueBooking(second))
	}
	if auditCount(t, env, "sponsor_booking.expired", two) != 1 {
		t.Fatal("the expiry isn't in the audit log")
	}
}
