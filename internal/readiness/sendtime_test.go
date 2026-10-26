package readiness_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/readiness"
)

// SR-3 (C3): the send-time recheck holds a scheduled send on what nobody
// confirmed. One case per reason.

func confirmedRun(t *testing.T) (readiness.ReadinessRun, *readiness.Confirmation, readiness.Digests) {
	t.Helper()
	res := stored(t, readiness.Assemble(digest()))
	run := readiness.ReadinessRun{ID: "run-1", RanAt: now, ReadinessResult: res}
	acks := []string{}
	for _, c := range run.Checks {
		if c.Status == readiness.Warn {
			acks = append(acks, c.Key)
		}
	}
	d := readiness.Digests{Content: "content-1", Sender: "sender-1", Route: "route-1"}
	conf := &readiness.Confirmation{RunID: run.ID, ConfirmedAt: now.Add(-23 * time.Hour), Recipients: run.Estimates.Audience.Final, Acknowledged: acks, Digests: d}
	return run, conf, d
}

func holdKinds(c readiness.SendTimeCheck) []string {
	out := []string{}
	for _, h := range c.Holds {
		out = append(out, h.Kind)
	}
	return out
}

// What was confirmed, unchanged: the send starts.
func TestSendTimeConfirmedStarts(t *testing.T) {
	run, conf, d := confirmedRun(t)
	if c := readiness.CheckSendTime(run, conf, d); c.Held() || c.Audience != 31604 || c.Confirmed != 31604 {
		t.Fatalf("confirmed: %+v", c)
	}
}

// A check that blocks holds the send, confirmed or not.
func TestSendTimeHoldsOnBlock(t *testing.T) {
	f := digest()
	f.Auth.DMARCStatus = "missing"
	run := readiness.ReadinessRun{ReadinessResult: stored(t, readiness.Assemble(f))}
	c := readiness.CheckSendTime(run, nil, readiness.Digests{})
	if !c.Held() || c.Holds[0].Kind != readiness.HoldBlocked || c.Holds[0].Keys[0] != "auth.dmarc" || !strings.Contains(c.Holds[0].English(), "DMARC") {
		t.Fatalf("block: %v %+v", holdKinds(c), c.Holds)
	}
}

// A warning nobody confirmed (it wasn't there at review) holds the send.
func TestSendTimeHoldsOnNewWarning(t *testing.T) {
	run, conf, d := confirmedRun(t)
	conf.Acknowledged = conf.Acknowledged[:1]
	c := readiness.CheckSendTime(run, conf, d)
	if len(c.Holds) != 1 || c.Holds[0].Kind != readiness.HoldNewWarning || !strings.HasPrefix(c.Holds[0].English(), "A warning nobody confirmed: ") {
		t.Fatalf("new warning: %+v", c.Holds)
	}
}

// An unknown check nobody chose to send without holds the send; one they
// accepted at review doesn't.
func TestSendTimeHoldsOnUnknown(t *testing.T) {
	f := digest()
	f.Reputation.Status, f.Reputation.Reason = readiness.Unknown, "HostRepute didn't answer"
	run := readiness.ReadinessRun{ReadinessResult: stored(t, readiness.Assemble(f))}
	_, conf, d := confirmedRun(t)
	conf.Acknowledged = []string{"audience.unengaged", "reputation.ip_blocklists.203.0.113.25"}
	c := readiness.CheckSendTime(run, conf, d)
	if kinds := holdKinds(c); len(kinds) == 0 || kinds[0] != readiness.HoldUnknown {
		t.Fatalf("unknown: %v", kinds)
	}
	for _, ch := range run.Checks {
		if ch.Status == readiness.Unknown {
			conf.Unknown = append(conf.Unknown, ch.Key)
		}
	}
	if c := readiness.CheckSendTime(run, conf, d); c.Held() {
		t.Fatalf("accepted at review: %v", holdKinds(c))
	}
}

// The audience changing by more than 10% after review holds the send
// (board: "the audience grew to 44,010 (+41% since you reviewed)").
func TestSendTimeHoldsOnAudienceChange(t *testing.T) {
	run, conf, d := confirmedRun(t)
	conf.Recipients = 31192
	run.Estimates.Audience.Final = 44010
	c := readiness.CheckSendTime(run, conf, d)
	if len(c.Holds) != 1 || c.Holds[0].Kind != readiness.HoldAudience || c.Holds[0].English() != "The audience grew to 44,010 (+41% since you reviewed)" {
		t.Fatalf("growth: %+v", c.Holds)
	}
	if vi := c.Holds[0].Text.In(i18n.For(i18n.Vietnamese)); !strings.Contains(vi, "44.010") {
		t.Errorf("growth in Vietnamese: %q", vi)
	}
	run.Estimates.Audience.Final = 27000
	if c := readiness.CheckSendTime(run, conf, d); len(c.Holds) != 1 || !strings.HasPrefix(c.Holds[0].English(), "The audience shrank to 27,000") {
		t.Fatalf("shrank: %+v", c.Holds)
	}
	run.Estimates.Audience.Final = 33000 // +5.8%
	if c := readiness.CheckSendTime(run, conf, d); c.Held() {
		t.Fatalf("within 10%%: %+v", c.Holds)
	}
}

// The content, sender or route edited after scheduling holds the send.
func TestSendTimeHoldsOnEdits(t *testing.T) {
	run, conf, d := confirmedRun(t)
	for _, tc := range []struct {
		edit func(*readiness.Digests)
		want string
	}{
		{func(x *readiness.Digests) { x.Content = "content-2" }, "The content changed after it was scheduled"},
		{func(x *readiness.Digests) { x.Sender = "sender-2" }, "The sender changed after it was scheduled"},
		{func(x *readiness.Digests) { x.Route = "route-2"; x.Content = "content-2" }, "The content and the route changed after it was scheduled"},
	} {
		now := d
		tc.edit(&now)
		c := readiness.CheckSendTime(run, conf, now)
		if len(c.Holds) != 1 || c.Holds[0].Kind != readiness.HoldEdited || c.Holds[0].English() != tc.want {
			t.Errorf("edit: %+v, want %q", c.Holds, tc.want)
		}
	}
}

// Without a confirmation (scheduled through the API before they were
// kept), only blocks hold the send.
func TestSendTimeWithoutConfirmation(t *testing.T) {
	run, _, _ := confirmedRun(t)
	if c := readiness.CheckSendTime(run, nil, readiness.Digests{Content: "x"}); c.Held() {
		t.Fatalf("no confirmation: %+v", c.Holds)
	}
}
