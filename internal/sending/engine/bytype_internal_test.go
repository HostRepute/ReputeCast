package engine

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// The thresholds by bounce type come from the workspace's defaults
// (Traffic rules) and the campaign's own; unknown actions and windows keep
// the defaults (review GS-11).
func TestTypedThresholds(t *testing.T) {
	th := ParseThresholds(json.RawMessage(`{"block_bounce":0.3,"block_action":"warn"}`),
		json.RawMessage(`{"deferral_slow":10,"deferral_slow_window":5,"deferral_action":"explode","unknown_pause":4}`))
	if th.BlockBounce != 0.3 || th.BlockAction != actWarn || th.DeferralSlow != 10 || th.DeferralSlowWindow != 5 || th.DeferralAction != actPauseMailbox ||
		th.UnknownPause != 4 || th.UnknownPauseAction != actPauseSend || th.Deferral != 40 || th.DeferralWindow != 10 {
		t.Fatalf("%+v", th)
	}
}

// The levels: block/policy bounces per mailbox provider, deferrals over
// the window (the pausing level first), unknown users over the send; a
// level that acted doesn't act again.
func TestCheckTyped(t *testing.T) {
	pid := uuid.New()
	rs := &runState{thresholds: DefaultThresholds, holds: map[string]hold{}, queue: map[queueKey]queueCell{}}
	ts := tallyRows([]store.SendingCountersRow{
		{ProviderID: pid, MailboxProvider: "outlook", Sent: 1000, Delivered: 990},
		{ProviderID: pid, MailboxProvider: "yahoo", Sent: 1000, Delivered: 990},
	})
	e := &Engine{}
	rs.typed = typed{block: map[string]int64{"outlook": 12}, unknown: map[string]int64{},
		deferred: map[int]map[string]int64{10: {"yahoo": 200}}, sent: map[int]map[string]int64{10: {"yahoo": 1000, "outlook": 1000}}}
	tt := e.checkTyped(rs, ts, rs.typed.at)
	if tt == nil || tt.level != "block" || tt.mailbox != "outlook" || tt.action != actPauseMailbox {
		t.Fatalf("got %+v, want Outlook's block/policy bounces", tt)
	}
	rs.guard.markActed("block", "outlook", rs.typed.at)
	if tt = e.checkTyped(rs, ts, rs.typed.at); tt == nil || tt.level != "deferral1" || tt.mailbox != "yahoo" || tt.action != actSlow {
		t.Fatalf("got %+v, want Yahoo slowed for its deferrals", tt)
	}
	rs.guard.markActed("deferral1", "yahoo", rs.typed.at)
	rs.typed.deferred[10]["yahoo"] = 470
	if tt = e.checkTyped(rs, ts, rs.typed.at); tt == nil || tt.level != "deferral2" || tt.action != actPauseMailbox {
		t.Fatalf("got %+v, want Yahoo paused at the second level", tt)
	}
	rs.guard.markActed("deferral2", "yahoo", rs.typed.at)
	rs.typed.unknown = map[string]int64{"outlook": 70, "yahoo": 60}
	if tt = e.checkTyped(rs, ts, rs.typed.at); tt == nil || tt.level != "unknown2" || tt.scope != "campaign" || tt.action != actPauseSend {
		t.Fatalf("got %+v, want the send paused for its unknown users", tt)
	}
	rs.guard.markActed("unknown2", "", rs.typed.at)
	if tt = e.checkTyped(rs, ts, rs.typed.at); tt == nil || tt.level != "unknown1" || tt.action != actWarn {
		t.Fatalf("got %+v, want the first level's warning", tt)
	}
	rs.guard.markActed("unknown1", "", rs.typed.at)
	if tt = e.checkTyped(rs, ts, rs.typed.at); tt != nil {
		t.Fatalf("got %+v, want nothing left to act", tt)
	}
	// A person resuming Yahoo re-arms its levels.
	rs.guard.clearActed("yahoo")
	if !rs.guard.acted("block", "outlook") || rs.guard.acted("deferral1", "yahoo") {
		t.Fatalf("acted %v", rs.guard.Acted)
	}
}
