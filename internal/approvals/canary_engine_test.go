package approvals_test

import (
	"fmt"
	"testing"

	"github.com/hostrepute/reputecast/internal/readiness/readinesstest"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// C1 end to end: the canary chosen on Review & send reaches the sending
// engine, which starts the send with its canary phase: the most engaged 5%
// first, everyone else held until the gates pass (review GS-10).
func TestReviewCanaryStartsTheCanaryPhase(t *testing.T) {
	s := sendtest.NewSuite(t)
	f := s.Fixture(t, "Canary from Review & send")
	// 300 engaged Gmail readers and 100 Yahoo readers who never opened,
	// without the seed mailboxes (their placement test is internal/placement's).
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.Exec(`INSERT INTO traffic_rules (workspace_id, canary) VALUES ($1, '{"seeds":false}')
		ON CONFLICT (workspace_id) DO UPDATE SET canary = traffic_rules.canary || '{"seeds":false}'`, f.WS)
	f.AddContacts(
		sendtest.ContactSpec{N: 300, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
		sendtest.ContactSpec{N: 100, Mailbox: "yahoo", Domain: "yahoo.com"},
	)
	linh := s.Env.Client()
	linh.SignInDemo("linh@acme.io")
	// No approval in the way.
	var policy struct{ Policy map[string]any }
	linh.Do("GET", ws+"/approval-policy", nil).MustStatus(t, 200).JSON(t, &policy)
	for _, k := range []string{"warmingProvider", "readyWithWarnings", "authorLacksSend", "approvalLists"} {
		policy.Policy[k] = false
	}
	policy.Policy["audienceMin"] = map[string]any{"enabled": false, "n": 10000}
	linh.Do("PUT", ws+"/approval-policy", policy.Policy).MustStatus(t, 200)

	id := f.Campaign.String()
	readinesstest.Post(t, s.Env, linh, ws+"/campaigns/"+id+"/readiness").MustStatus(t, 200)
	r := getReview(t, linh, id)
	linh.Do("POST", ws+"/campaigns/"+id+"/send", map[string]any{"runId": r.Run.ID, "acknowledged": warnings(r),
		"confirm": fmt.Sprint(r.Run.Estimates.Audience.Final), "canary": "on"}).MustStatus(t, 200)
	if err := s.E.DrivePrepare(s.Ctx, f.Campaign, s.Start); err != nil {
		t.Fatal(err)
	}
	if n := f.Count(`SELECT count(*) FROM campaigns WHERE id = $1 AND canary = 'on'`, f.Campaign); n != 1 {
		t.Fatal("the engine didn't keep the choice on the campaign")
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND canary`, f.Campaign); n != 20 {
		f.Dump()
		t.Fatalf("%d canary recipients, want 5%% of 400", n)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'held' AND hold_reason = 'canary'`, f.Campaign); n != 380 {
		t.Fatalf("%d held for the canary, want 380", n)
	}
}
