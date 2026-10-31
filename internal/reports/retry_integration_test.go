package reports_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/readiness/readinesstest"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

type blockedRetry struct {
	Blocked     int64
	Recipients  int64
	Unavailable string
	Campaign    *struct {
		ID, Name, Status string
		StartedAt        *time.Time
	}
	Of *struct{ ID, Name string }
}

// TestRetryBlocked: "Retry after fixing" (review AN-7, R5.L5). Addresses
// a mailbox provider blocked for the sender's reputation are valid, so once
// the block is fixed the issue goes to them again: a draft retry made from
// the report's Investigate dialog, counted and checked by Review & send,
// sent by the engine's usual path to exactly those still on the list, and
// to each of them once.
func TestRetryBlocked(t *testing.T) {
	s := sendtest.NewSuite(t)
	f := s.Fixture(t, "Retry blocked")
	ids := f.HandedOver(sendtest.ContactSpec{N: 6, Mailbox: "outlook", Domain: "outlook.com", Engaged: true})
	contact := func(id uuid.UUID) uuid.UUID {
		t.Helper()
		var c uuid.UUID
		if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT contact_id FROM messages WHERE id = $1`, id).Scan(&c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	event := func(id uuid.UUID, ev providers.Event) {
		t.Helper()
		var pid uuid.UUID
		if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT provider_id FROM messages WHERE id = $1`, id).Scan(&pid); err != nil {
			t.Fatal(err)
		}
		p, err := f.Env.App.Q.ProvGetAny(f.Ctx, pid)
		if err != nil {
			t.Fatal(err)
		}
		mid := id
		ev.MessageID, ev.OccurredAt = &mid, f.Start.Add(2*time.Minute)
		if err := sending.Tx(f.Ctx, f.Env.App.Pool, func(tx pgx.Tx) error {
			_, err := f.E.Normalizer.Apply(f.Ctx, tx, p, nil, "webhook", ev.OccurredAt, ev)
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	// Outlook blocks three for the sending IP's reputation (S3150), one
	// address doesn't exist, two are delivered.
	block := providers.Event{Type: providers.EventHardBounce, OriginalType: "Permanent", OriginalCode: "550 5.7.1", BounceClass: "reputation",
		Diagnostic: "550 5.7.1 Unfortunately, messages from [203.0.113.40] weren't sent. (S3150)"}
	event(ids[0], block)
	event(ids[1], providers.Event{Type: providers.EventSoftBounce, OriginalType: "Transient", OriginalCode: "451 4.7.650", BounceClass: "policy_block",
		Diagnostic: "451 4.7.650 The mail server [203.0.113.40] has been temporarily rate limited due to IP reputation."})
	event(ids[2], block)
	event(ids[3], providers.Event{Type: providers.EventHardBounce, OriginalType: "Permanent", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown"})
	event(ids[4], providers.Event{Type: providers.EventDelivered, OriginalCode: "250 2.0.0"})
	event(ids[5], providers.Event{Type: providers.EventDelivered, OriginalCode: "250 2.0.0"})
	if _, done := f.Drive(30 * time.Minute); !done {
		f.Dump()
		t.Fatal("the send didn't end")
	}
	// One of the blocked unsubscribes since: the retry leaves them out.
	f.Exec(`UPDATE list_memberships SET status = 'unsubscribed' WHERE list_id = $1 AND contact_id = $2`, f.List, contact(ids[2]))

	linh := f.Env.Client()
	linh.SignInDemo("linh@acme.io")
	aisha := f.Env.Client()
	aisha.SignInDemo("aisha@acme.io")
	review := func(campaign uuid.UUID) blockedRetry {
		t.Helper()
		var r struct{ Retry *blockedRetry }
		linh.Do("GET", fmt.Sprintf("/workspaces/acme-newsletter/campaigns/%s/report/bounces/blocked", campaign), nil).MustStatus(t, http.StatusOK).JSON(t, &r)
		if r.Retry == nil {
			t.Fatal("the blocked addresses' review has no retry")
		}
		return *r.Retry
	}
	if r := review(f.Campaign); r.Blocked != 3 || r.Recipients != 2 || r.Unavailable != "" || r.Campaign != nil {
		t.Fatalf("before a retry: %+v, want 3 blocked and 2 to send to", r)
	}
	// Other bounces have no retry.
	var other struct{ Retry *blockedRetry }
	linh.Do("GET", fmt.Sprintf("/workspaces/acme-newsletter/campaigns/%s/report/bounces/unknown_user", f.Campaign), nil).MustStatus(t, http.StatusOK).JSON(t, &other)
	if other.Retry != nil {
		t.Fatalf("unknown users are retried: %+v", other.Retry)
	}

	// A viewer can't make one; an editor can, once.
	path := fmt.Sprintf("/workspaces/acme-newsletter/campaigns/%s/retry", f.Campaign)
	aisha.Do("POST", path, nil).MustStatus(t, http.StatusForbidden)
	var retry struct {
		ID, Name, Status string
	}
	linh.Do("POST", path, nil).MustStatus(t, http.StatusCreated).JSON(t, &retry)
	if retry.Status != "draft" || retry.Name != "Retry blocked · retry" {
		t.Fatalf("the retry: %+v", retry)
	}
	if p := linh.Do("POST", path, nil).MustStatus(t, http.StatusConflict).Problem(t); p["detail"] == "" {
		t.Fatal("a second retry is refused without a reason")
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE action = 'campaign.create' AND target_id = $1 AND after ->> 'retryOf' = $2`,
		retry.ID, f.Campaign.String()); n != 1 {
		t.Fatalf("%d audit entries for the retry", n)
	}
	if r := review(f.Campaign); r.Campaign == nil || r.Campaign.ID != retry.ID || r.Campaign.StartedAt != nil {
		t.Fatalf("the review doesn't name the draft retry: %+v", r)
	}

	// Review & send counts and explains it.
	run := readinesstest.Run(t, f.Env, linh, "acme-newsletter", retry.ID)
	if run.Estimates.Audience.Final != 2 {
		t.Fatalf("Review & send counts %d recipients, want the 2 blocked still subscribed", run.Estimates.Audience.Final)
	}
	if !slices.ContainsFunc(run.Checks, func(c readiness.ReadinessCheck) bool { return c.Key == "audience.retry" && c.Status == "pass" }) {
		t.Fatal("Review & send has no line for the retry")
	}

	// The engine sends it to exactly those two.
	rid := uuid.MustParse(retry.ID)
	if err := f.E.StartWith(f.Ctx, rid, engine.StartOptions{By: f.Sofia, Now: true}); err != nil {
		t.Fatalf("start the retry: %v", err)
	}
	if err := f.E.DrivePrepare(f.Ctx, rid, f.Start); err != nil {
		t.Fatalf("prepare the retry: %v", err)
	}
	if _, done, err := f.E.Drive(f.Ctx, rid, f.Start, time.Hour); err != nil || !done {
		t.Fatalf("the retry's send: done %v, %v", done, err)
	}
	rows, err := f.Env.App.Pool.Query(f.Ctx, `SELECT contact_id FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, rid)
	if err != nil {
		t.Fatal(err)
	}
	var got []uuid.UUID
	for rows.Next() {
		var c uuid.UUID
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		got = append(got, c)
	}
	rows.Close()
	want := []uuid.UUID{contact(ids[0]), contact(ids[1])}
	slices.SortFunc(got, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	slices.SortFunc(want, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	if !slices.Equal(got, want) {
		t.Fatalf("the retry went to %v, want the blocked still subscribed %v", got, want)
	}
	if r := review(f.Campaign); r.Campaign == nil || r.Campaign.StartedAt == nil {
		t.Fatalf("the review doesn't show the retry went out: %+v", r)
	}
	// A retry isn't retried again.
	if r := review(rid); r.Unavailable != "is_retry" || r.Of == nil || r.Of.ID != f.Campaign.String() {
		t.Fatalf("a retry's own review: %+v", r)
	}
	linh.Do("POST", fmt.Sprintf("/workspaces/acme-newsletter/campaigns/%s/retry", rid), nil).MustStatus(t, http.StatusConflict)

	// Once each: with the retry gone, the two who had it aren't retried
	// again, and the one who unsubscribed isn't sent to.
	f.Exec(`UPDATE campaigns SET deleted_at = now() WHERE id = $1`, rid)
	if r := review(f.Campaign); r.Blocked != 1 || r.Recipients != 0 || r.Campaign != nil {
		t.Fatalf("after one retry: %+v, want 1 blocked left and nobody to send to", r)
	}
}
