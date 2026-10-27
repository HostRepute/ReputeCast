package engine_test

import (
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// testSharedThrottles: two sends going out at once share a mailbox
// provider's throttle (review GS-8): together they keep to Outlook's 5 /s
// instead of 5 /s each. The live monitor says when each route and mailbox
// provider is done at that rate.
func testSharedThrottles(t *testing.T, s *sendtest.Suite) {
	// Each fixture's contacts are added before the next fixture exists:
	// their addresses carry the suite's fixture number.
	fixture := func(name string) *sendtest.Fixture {
		f := s.Fixture(t, name)
		f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
		f.AddContacts(
			sendtest.ContactSpec{N: 300, Mailbox: "outlook", Domain: "hotmail.com", Engaged: true},
			sendtest.ContactSpec{N: 100, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
		)
		return f
	}
	a := fixture("GS-8 a")
	b := fixture("GS-8 b")
	for _, f := range []*sendtest.Fixture{a, b} {
		f.StartSend(engine.StartOptions{})
	}
	at := a.Start
	for range 20 {
		for _, f := range []*sendtest.Fixture{a, b} {
			if _, _, err := f.E.Drive(f.Ctx, f.Campaign, at, time.Second); err != nil {
				t.Fatal(err)
			}
		}
		at = at.Add(time.Second)
	}
	n := a.Count(`SELECT count(*) FROM messages WHERE campaign_id IN ($1, $2) AND mailbox_provider = 'outlook' AND sent_at IS NOT NULL`, a.Campaign, b.Campaign)
	// Never over 5 /s together. Some of a slot can go unused: a send whose
	// sending provider is busy with the other send gives its part back
	// after the other one went.
	if n > 5*20+5 || n < 5*20*6/10 {
		a.Dump()
		t.Fatalf("%d sent to Outlook in 20 s by two sends, want up to 100: 5 /s together", n)
	}
	if na, nb := a.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND mailbox_provider = 'outlook' AND sent_at IS NOT NULL`, a.Campaign),
		b.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND mailbox_provider = 'outlook' AND sent_at IS NOT NULL`, b.Campaign); na == 0 || nb == 0 {
		t.Fatalf("Outlook sent %d and %d: both sends should get a share", na, nb)
	}
	setClock(t, a, at)
	snap, err := a.E.Snapshot(a.Ctx, a.WS, a.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	var outlook *time.Time
	for _, m := range snap.Mailboxes {
		if m.Key == "outlook" {
			outlook = m.DoneAt
		}
	}
	// What's left of this send's Outlook share at 5 /s: a few dozen
	// seconds at least.
	if outlook == nil || outlook.Sub(at) < 20*time.Second {
		t.Fatalf("Outlook done at %v (now %v), want its rest at 5 /s", outlook, at)
	}
	var last *time.Time
	for _, p := range snap.Providers {
		if p.DoneAt != nil && (last == nil || p.DoneAt.After(*last)) {
			last = p.DoneAt
		}
	}
	if last == nil || last.Before(outlook.Add(-time.Second)) {
		t.Fatalf("the routes are done at %v, before their Outlook mail (%v)", last, outlook)
	}
	if snap.FinishesAt == nil || snap.FinishesAt.Before(*outlook) {
		t.Fatalf("the send finishes at %v, before Outlook (%v)", snap.FinishesAt, outlook)
	}
}
