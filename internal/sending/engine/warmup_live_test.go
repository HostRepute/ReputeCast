package engine_test

import (
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// A share today's warm-up cap holds shows on the live monitor like the
// other holds (review GS-14 with PR-8): which route, how many recipients
// wait, until when, where they could move, and an ETA that counts the wait
// for tomorrow's cap. It isn't a decision the send stops for, and resuming
// the send leaves it waiting for the cap.
func TestWarmupHoldOnTheLiveMonitor(t *testing.T) {
	s := sendtest.NewSuite(t)
	f := s.Fixture(t, "Warm-up hold live")
	postal, ses, plan, _ := warmupFixture(t, f, false, false)
	f.AddContacts(sendtest.ContactSpec{N: 90, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	f.StartSend(engine.StartOptions{})
	at, done := f.Drive(10 * time.Minute)
	if done {
		f.Dump()
		t.Fatal("the send finished past the cap")
	}
	setClock(t, f, at)
	postalOf := func(snap engine.LiveSnapshot) engine.LiveProvider {
		t.Helper()
		for _, p := range snap.Providers {
			if p.ID == postal.String() {
				return p
			}
		}
		t.Fatalf("no Postal row in %+v", snap.Providers)
		return engine.LiveProvider{}
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	p := postalOf(snap)
	h := p.WarmupHold
	if p.RouteStatus != "held" || h == nil {
		t.Fatalf("Postal %s, hold %+v", p.RouteStatus, h)
	}
	if h.PlanID != plan || h.Held != 30 || h.Paused || h.Decision != "" || h.Until == nil || h.Since == nil || h.Cap == 0 {
		t.Fatalf("hold %+v", h)
	}
	if !h.Until.After(at) || h.Until.Sub(at) > 24*time.Hour {
		t.Fatalf("waits until %v at %v: tomorrow's cap starts within a day", h.Until, at)
	}
	movable := false
	for _, x := range h.Targets {
		movable = movable || (x.ProviderID == ses.String() && x.Allowed)
		if x.ProviderID == postal.String() {
			t.Fatal("the held provider is offered as its own target")
		}
	}
	if !movable {
		t.Fatalf("SES can take the 30, targets %+v", h.Targets)
	}
	// The ETA: the 30 go out with tomorrow's cap, not now.
	if p.DoneAt == nil || p.DoneAt.Before(*h.Until) {
		t.Fatalf("Postal done at %v, before tomorrow's cap at %v", p.DoneAt, h.Until)
	}
	if snap.FinishesAt == nil || snap.FinishesAt.Before(*h.Until) {
		t.Fatalf("send finishes at %v, before tomorrow's cap at %v", snap.FinishesAt, h.Until)
	}
	for _, m := range snap.Mailboxes {
		if m.Key == "gmail" && (m.DoneAt == nil || m.DoneAt.Before(*h.Until)) {
			t.Fatalf("Gmail done at %v, before tomorrow's cap", m.DoneAt)
		}
	}
	// The outage path leaves it alone: no outage decision for a cap.
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'provider_outage'`, f.Campaign); n != 0 {
		t.Fatalf("%d outage incidents for a share the warm-up cap holds", n)
	}
	// The hold is shown, not a decision the send waits for.
	if snap.State != "partial" || snap.Incident != nil || snap.Options.CanResume || f.Status() != "sending" {
		t.Fatalf("state %s, decision %+v, can resume %v, campaign %s", snap.State, snap.Incident, snap.Options.CanResume, f.Status())
	}

	owner := f.Env.Client()
	owner.SignInDemo("linh@acme.io")
	live := "/workspaces/acme-newsletter/campaigns/" + f.Campaign.String() + "/live"
	// Someone chooses to wait: the monitor says so.
	owner.Do("POST", "/workspaces/acme-newsletter/warmup/"+plan+"/held/"+f.Campaign.String(), map[string]any{"action": "wait"}).MustStatus(t, 200)
	var got engine.LiveSnapshot
	owner.Do("GET", live, nil).MustStatus(t, 200).JSON(t, &got)
	if h := postalOf(got).WarmupHold; h == nil || h.Decision != "wait" || h.Held != 30 {
		t.Fatalf("after waiting: %+v", h)
	}
	// Resuming the share on its own is refused: it has its own choices.
	owner.Do("POST", live+"/resume", map[string]any{"providerId": postal.String()}).MustStatus(t, 409)
	// Pausing and resuming the send leaves it waiting for the cap.
	owner.Do("POST", live+"/pause", map[string]any{}).MustStatus(t, 200)
	owner.Do("POST", live+"/resume", map[string]any{}).MustStatus(t, 200)
	if n := f.Count(`SELECT count(*) FROM campaign_routes WHERE campaign_id = $1 AND provider_id = $2 AND status = 'held' AND status_detail LIKE '%"reason":"warmup_cap"%'`, f.Campaign, postal); n != 1 {
		f.Dump()
		t.Fatal("resuming the send released the share the warm-up cap holds")
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND metric = 'warmup_cap' AND resolved_at IS NULL`, f.Campaign); n != 1 {
		t.Fatal("resuming the send resolved the warm-up hold")
	}
}
