package approvals_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness/readinesstest"
	"github.com/hostrepute/reputecast/internal/sending"
)

// fakeStarter stands in for the sending engine.
type fakeStarter struct {
	mu       sync.Mutex
	started  []uuid.UUID
	canceled []uuid.UUID
	// rechecks are the send-time rechecks handed over with each start.
	rechecks []sending.Recheck
}

func (f *fakeStarter) Start(ctx context.Context, id, _ uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, id)
	if rc, ok := sending.RecheckFrom(ctx, id); ok {
		f.rechecks = append(f.rechecks, rc)
	}
	return nil
}
func (f *fakeStarter) Pause(context.Context, uuid.UUID, uuid.UUID, string) error { return nil }
func (f *fakeStarter) Resume(context.Context, uuid.UUID, uuid.UUID) error        { return nil }
func (f *fakeStarter) Cancel(_ context.Context, id, _ uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.canceled = append(f.canceled, id)
	return nil
}

const ws = "/workspaces/acme-newsletter"

type review struct {
	Campaign struct {
		ID, Name, Status string
		ScheduledAt      *string
	}
	Scheduled *struct{ By string }
	Stale     bool
	Run       *struct {
		ID      string
		Verdict string
		Counts  struct{ Passed, Warnings, Blocks int }
		Checks  []struct {
			Key, Status, Title string
			Fixes              []struct{ Kind, Action, Label string }
		}
		Estimates struct {
			Audience struct{ Final int64 }
		}
	}
	Approval struct {
		Required, Valid, CanRequest, CanBypass bool
		Rules                                  []struct{ Key, Label string }
		Request                                *struct {
			ID, Status        string
			Given, Needed     int
			InvalidatedReason string
			Changes           []struct{ Field, Before, After string }
		}
	}
	Access struct{ CanSend, CanApprove bool }
}

func campaignID(t *testing.T, c *apptest.Client, name string) string {
	t.Helper()
	var list struct {
		Items []struct{ ID, Name string }
	}
	c.Do("GET", ws+"/campaigns?pageSize=100&q="+strings.ReplaceAll(name, " ", "%20"), nil).MustStatus(t, 200).JSON(t, &list)
	for _, it := range list.Items {
		if it.Name == name {
			return it.ID
		}
	}
	t.Fatalf("campaign %q not found", name)
	return ""
}

func getReview(t *testing.T, c *apptest.Client, id string) review {
	t.Helper()
	var r review
	c.Do("GET", ws+"/campaigns/"+id+"/review", nil).MustStatus(t, 200).JSON(t, &r)
	return r
}

func warnings(r review) []string {
	out := []string{}
	if r.Run != nil {
		for _, ch := range r.Run.Checks {
			if ch.Status == "warn" {
				out = append(out, ch.Key)
			}
		}
	}
	return out
}

// counter counts audit entries and notifications, so assertions are about
// what an action added (the demo data has its own).
type counter struct {
	t   *testing.T
	env *apptest.Env
}

func (c counter) audit(action, target string) int {
	c.t.Helper()
	var n int
	if err := c.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1 AND ($2 = '' OR target_id = $2)`, action, target).Scan(&n); err != nil {
		c.t.Fatal(err)
	}
	return n
}

func (c counter) notes(email, kind string) int {
	c.t.Helper()
	var n int
	if err := c.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM notifications n JOIN users u ON u.id = n.user_id WHERE u.email = $1 AND n.kind = $2`, email, kind).Scan(&n); err != nil {
		c.t.Fatal(err)
	}
	return n
}

func TestReviewAndApprovals(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	starter := &fakeStarter{}
	env.App.Sending = starter
	n := counter{t, env}

	linh, marcus, daniel, aisha := env.Client(), env.Client(), env.Client(), env.Client()
	linh.SignInDemo("linh@acme.io")
	marcus.SignInDemo("marcus@acme.io")
	daniel.SignInDemo("daniel@acme.io")
	aisha.SignInDemo("aisha@acme.io")
	nov := campaignID(t, linh, "November Product Update")
	digest := campaignID(t, linh, "Weekly Digest #142")
	flash := campaignID(t, linh, "Flash sale: 48 hours only")

	// The sample data: Weekly Digest approved and ready with a warning,
	// the flash sale blocked, November waiting for Marcus or Linh.
	r := getReview(t, linh, digest)
	if r.Run == nil || r.Run.Verdict != "warnings" || r.Run.Counts.Warnings != 1 || r.Stale || !r.Approval.Valid {
		t.Fatalf("weekly digest: %+v", r)
	}
	if r.Run.Estimates.Audience.Final != 31192 {
		t.Fatalf("final recipients: %d", r.Run.Estimates.Audience.Final)
	}
	r = getReview(t, linh, flash)
	if r.Run == nil || r.Run.Verdict != "blocked" || r.Run.Counts.Blocks != 3 {
		t.Fatalf("flash sale: %+v", r.Run)
	}
	r = getReview(t, daniel, nov)
	if r.Approval.Request == nil || r.Approval.Request.Status != "pending" || len(r.Approval.Rules) != 3 || r.Access.CanSend || r.Access.CanApprove {
		t.Fatalf("november as Daniel: %+v", r.Approval)
	}
	reqID := r.Approval.Request.ID

	// The inbox: approvers see every request, authors their own.
	var inbox struct {
		Items  []struct{ Name, Status string }
		Counts struct{ Pending, Approved, ChangesRequested, Rejected, Invalidated, All int }
		Policy struct{ MedianWaitMinutes *int }
	}
	marcus.Do("GET", ws+"/approvals?status=all", nil).MustStatus(t, 200).JSON(t, &inbox)
	// Approved: the board's three and the recurring digest's issues #1 and #2.
	// Pending: November and Daniel's post "Q4 roadmap deep dive" (WebsiteStates).
	if inbox.Counts.Pending != 2 || inbox.Counts.Approved != 5 || inbox.Counts.ChangesRequested != 1 || inbox.Counts.Invalidated != 1 || inbox.Counts.All != 9 {
		t.Fatalf("inbox counts: %+v", inbox.Counts)
	}
	if inbox.Policy.MedianWaitMinutes == nil || *inbox.Policy.MedianWaitMinutes != 160 {
		t.Fatalf("median wait: %v", inbox.Policy.MedianWaitMinutes)
	}
	daniel.Do("GET", ws+"/approvals?status=all", nil).MustStatus(t, 200).JSON(t, &inbox)
	if inbox.Counts.All != 4 {
		t.Fatalf("Daniel sees his own requests: %+v", inbox.Items)
	}
	marcus.Do("GET", ws+"/approvals?status=pending", nil).MustStatus(t, 200).JSON(t, &inbox)
	pending := []string{}
	for _, it := range inbox.Items {
		pending = append(pending, it.Name)
	}
	if len(pending) != 2 || !slices.Contains(pending, "November Product Update") || !slices.Contains(pending, "Q4 roadmap deep dive") {
		t.Fatalf("pending tab: %+v", inbox.Items)
	}

	// Permissions: Viewers read, never decide or run checks; authors never
	// approve their own campaign.
	aisha.Do("GET", ws+"/campaigns/"+nov+"/review", nil).MustStatus(t, 200)
	aisha.Do("POST", ws+"/campaigns/"+nov+"/readiness", nil).MustStatus(t, 403)
	aisha.Do("POST", ws+"/approvals/"+reqID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 403)
	aisha.Do("GET", ws+"/approvals/"+reqID, nil).MustStatus(t, 404)
	daniel.Do("POST", ws+"/approvals/"+reqID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 403)
	daniel.Do("POST", ws+"/campaigns/"+nov+"/send", map[string]any{"runId": r.Run.ID}).MustStatus(t, 403)

	// A comment, then Marcus approves: audited, and Daniel hears about it.
	daniel.Do("POST", ws+"/approvals/"+reqID+"/comments", map[string]any{"body": "Thanks both."}).MustStatus(t, 204)
	approved, decided := n.audit("campaign.approved", nov), n.notes("daniel@acme.io", "approval_decided")
	marcus.Do("POST", ws+"/approvals/"+reqID+"/decision", map[string]any{"decision": "approve", "comment": "swap the bit.ly link before it goes out"}).MustStatus(t, 200)
	r = getReview(t, daniel, nov)
	if !r.Approval.Valid || r.Approval.Request.Status != "approved" || r.Campaign.Status != "approved" {
		t.Fatalf("after approval: %+v %+v", r.Campaign, r.Approval.Request)
	}
	if n.audit("campaign.approved", nov) != approved+1 || n.notes("daniel@acme.io", "approval_decided") != decided+1 {
		t.Fatal("approval is audited and the author notified")
	}
	marcus.Do("POST", ws+"/approvals/"+reqID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 409)

	// Daniel edits the subject: the approval is invalidated with the diff.
	var full map[string]any
	daniel.Do("GET", ws+"/campaigns/"+nov, nil).MustStatus(t, 200).JSON(t, &full)
	invalidated := n.audit("approval.invalidated", reqID)
	toMarcus, toLinh := n.notes("marcus@acme.io", "approval_invalidated"), n.notes("linh@acme.io", "approval_invalidated")
	newSubject := "November Product Update: faster exports, a new API and self-hosted sending"
	daniel.Do("PUT", ws+"/campaigns/"+nov+"/content", map[string]any{"baseVersion": full["version"], "subject": newSubject, "preheader": full["preheader"],
		"mjml": full["mjml"], "html": full["html"], "label": "subject", "source": "manual"}).MustStatus(t, 200)
	r = getReview(t, daniel, nov)
	if r.Approval.Valid || r.Approval.Request == nil || r.Approval.Request.Status != "invalidated" || r.Approval.Request.InvalidatedReason != "subject changed" {
		t.Fatalf("invalidated: %+v", r.Approval.Request)
	}
	if len(r.Approval.Request.Changes) != 1 || r.Approval.Request.Changes[0].Field != "subject" || r.Approval.Request.Changes[0].After != newSubject || r.Campaign.Status != "draft" {
		t.Fatalf("diff: %+v %s", r.Approval.Request.Changes, r.Campaign.Status)
	}
	if n.notes("marcus@acme.io", "approval_invalidated") != toMarcus+1 || n.notes("linh@acme.io", "approval_invalidated") != toLinh+1 {
		t.Fatal("approvers hear about the invalidation")
	}
	if n.audit("approval.invalidated", reqID) != invalidated+1 {
		t.Fatal("invalidation is audited")
	}

	// Marcus approves the campaign as it stands: a new request with the diff.
	var dec struct {
		ID, Status, FollowsID string
		ChangesSince          []struct{ Field string }
	}
	// The campaign changed since the checks ran: they run again first (in
	// the background), and Marcus approves once they finish.
	if p := marcus.Do("POST", ws+"/approvals/"+reqID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 409).Problem(t); p["code"] != "readiness_running" {
		t.Fatalf("approve before the checks ran again: %v", p)
	}
	readinesstest.Work(t, env)
	marcus.Do("POST", ws+"/approvals/"+reqID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 200).JSON(t, &dec)
	if dec.Status != "approved" || dec.ID == reqID || dec.FollowsID != reqID || len(dec.ChangesSince) != 1 || dec.ChangesSince[0].Field != "subject" {
		t.Fatalf("approve the new version: %+v", dec)
	}

	// Daniel schedules it as approved: warnings confirmed, the count typed.
	r = getReview(t, daniel, nov)
	if r.Stale || r.Run == nil || !r.Approval.Valid {
		t.Fatalf("approved and fresh after the edit: %+v", r)
	}
	final := fmt.Sprint(r.Run.Estimates.Audience.Final)
	daniel.Do("POST", ws+"/approvals/"+dec.ID+"/schedule", map[string]any{"runId": r.Run.ID, "acknowledged": warnings(r)}).MustStatus(t, 422) // the count isn't typed
	if len(warnings(r)) > 0 {
		daniel.Do("POST", ws+"/approvals/"+dec.ID+"/schedule", map[string]any{"runId": r.Run.ID, "confirm": final}).MustStatus(t, 422) // warnings not confirmed
	}
	scheduled := n.audit("campaign.scheduled", nov)
	var sent struct{ Status string }
	daniel.Do("POST", ws+"/approvals/"+dec.ID+"/schedule", map[string]any{"runId": r.Run.ID, "acknowledged": warnings(r), "confirm": final}).MustStatus(t, 200).JSON(t, &sent)
	if sent.Status != "scheduled" || len(starter.started) != 1 || n.audit("campaign.scheduled", nov) != scheduled+1 {
		t.Fatalf("scheduled: %+v %v", sent, starter.started)
	}
	if r = getReview(t, daniel, nov); r.Campaign.Status != "scheduled" || r.Scheduled == nil || r.Stale {
		t.Fatalf("the review shows the scheduled send: %+v", r.Campaign)
	}
	daniel.Do("POST", ws+"/campaigns/"+nov+"/unschedule", nil).MustStatus(t, 204)
	if r = getReview(t, daniel, nov); r.Campaign.Status != "approved" || len(starter.canceled) != 1 {
		t.Fatalf("unscheduled keeps the approval: %s", r.Campaign.Status)
	}

	// Without the sending engine nothing is scheduled and the error says so.
	env.App.Sending = nil
	linh.Do("POST", ws+"/campaigns/"+nov+"/send", map[string]any{"runId": r.Run.ID, "acknowledged": warnings(r), "confirm": final}).MustStatus(t, 503)
	env.App.Sending = starter
	if r = getReview(t, linh, nov); r.Campaign.Status != "approved" || r.Campaign.ScheduledAt != nil {
		t.Fatalf("nothing was scheduled: %+v", r.Campaign)
	}

	// Emergency bypass: Owners only, with a reason and their password.
	marcus.Do("POST", ws+"/campaigns/"+nov+"/bypass", map[string]any{"runId": r.Run.ID, "reason": "Customer-facing outage notice."}).MustStatus(t, 403)
	bypass := map[string]any{"runId": r.Run.ID, "acknowledged": warnings(r), "confirm": final, "reason": "short", "password": "reputecast-demo"}
	linh.Do("POST", ws+"/campaigns/"+nov+"/bypass", bypass).MustStatus(t, 422)
	bypass["reason"], bypass["password"] = "Customer-facing outage notice, approved verbally by Marcus at 09:12.", "not the password"
	linh.Do("POST", ws+"/campaigns/"+nov+"/bypass", bypass).MustStatus(t, 422)
	bypassed, toApprover := n.audit("campaign.approval_bypassed", nov), n.notes("marcus@acme.io", "approval_decided")
	bypass["password"] = "reputecast-demo"
	linh.Do("POST", ws+"/campaigns/"+nov+"/bypass", bypass).MustStatus(t, 200)
	if n.audit("campaign.approval_bypassed", nov) != bypassed+1 || n.notes("marcus@acme.io", "approval_decided") != toApprover+1 {
		t.Fatal("the bypass is audited and approvers are told")
	}

	// Blocked sends never start.
	r = getReview(t, linh, flash)
	linh.Do("POST", ws+"/campaigns/"+flash+"/send", map[string]any{"runId": r.Run.ID, "acknowledged": warnings(r), "confirm": fmt.Sprint(r.Run.Estimates.Audience.Final)}).MustStatus(t, 409)

	// The policy: anyone reads it; workspace:manage changes it (audited).
	var policy struct {
		Policy    map[string]any
		Approvers []struct{ Name string }
		Matches   struct{ Warming string }
	}
	daniel.Do("GET", ws+"/approval-policy", nil).MustStatus(t, 200).JSON(t, &policy)
	if len(policy.Approvers) != 2 || !strings.Contains(policy.Matches.Warming, "Postal") {
		t.Fatalf("policy view: %+v", policy)
	}
	p := policy.Policy
	p["approvalsRequired"] = 2
	daniel.Do("PUT", ws+"/approval-policy", p).MustStatus(t, 403)
	updated := n.audit("approval_policy.updated", "")
	linh.Do("PUT", ws+"/approval-policy", p).MustStatus(t, 200).JSON(t, &policy)
	if policy.Policy["approvalsRequired"].(float64) != 2 || n.audit("approval_policy.updated", "") != updated+1 {
		t.Fatalf("policy saved: %+v", policy.Policy)
	}
	p["approvalsRequired"] = 3
	linh.Do("PUT", ws+"/approval-policy", p).MustStatus(t, 422)
}

func TestReadinessRunAndFixes(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	n := counter{t, env}
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	digest := campaignID(t, linh, "Weekly Digest #142")
	flash := campaignID(t, linh, "Flash sale: 48 hours only")

	// A run on the live data replaces the sample run.
	var run struct {
		ID, Verdict string
		Counts      struct{ Total int }
		Groups      []struct{ Key string }
	}
	runs := n.audit("readiness.run", digest)
	readinesstest.Post(t, env, linh, ws+"/campaigns/"+digest+"/readiness").MustStatus(t, 200).JSON(t, &run)
	// Seven groups of 1.0, Accessibility, and Languages when an email goes
	// out in several languages (the sample's #142 does).
	if run.ID == "" || run.Counts.Total < 20 || len(run.Groups) < 8 {
		t.Fatalf("run: %+v", run)
	}
	if n.audit("readiness.run", digest) != runs+1 {
		t.Fatal("a manual run is audited")
	}
	var progress struct{ Running *struct{ RunID string } }
	linh.Do("GET", ws+"/campaigns/"+digest+"/readiness/progress", nil).MustStatus(t, 200).JSON(t, &progress)
	if progress.Running != nil {
		t.Fatalf("nothing runs after the run: %+v", progress.Running)
	}

	// One-click fixes: a link that isn't in the email can't be replaced; the
	// footer goes back in and the unengaged are left out, both audited.
	var out struct {
		Applied []string
		Version int
		Run     *struct {
			Verdict string
			Checks  []struct{ Key, Status string }
		}
	}
	linh.Do("POST", ws+"/campaigns/"+flash+"/readiness/fixes", map[string]any{"fixes": []map[string]any{
		{"action": "replace_link", "params": map[string]string{"from": "https://nowhere.example/x", "to": "https://acme.io/sale"}},
	}}).MustStatus(t, 422)
	saves, updates := n.audit("campaign.content.save", flash), n.audit("campaign.update", flash)
	readinesstest.Fix(t, env, linh, ws+"/campaigns/"+flash+"/readiness/fixes", map[string]any{"fixes": []map[string]any{{"action": "insert_footer"}, {"action": "exclude_sunset"}}}).MustStatus(t, 200).JSON(t, &out)
	if len(out.Applied) != 2 || out.Run == nil {
		t.Fatalf("fixes: %+v", out)
	}
	for _, ch := range out.Run.Checks {
		if ch.Key == "compliance.footer" && ch.Status != "pass" {
			t.Fatalf("footer fixed: %+v", ch)
		}
	}
	if n.audit("campaign.content.save", flash) != saves+1 || n.audit("campaign.update", flash) != updates+1 {
		t.Fatal("fixes are audited")
	}

	// Send settings: thresholds and schedule are validated, saved and audited.
	thresholds := map[string]any{"bounce": 1.5, "complaint": 0.08, "mailboxComplaint": 0.1, "providerErrors": 5, "scope": "campaign"}
	linh.Do("PATCH", ws+"/campaigns/"+flash+"/send-settings", map[string]any{"thresholds": map[string]any{"bounce": 50, "complaint": 0.1, "mailboxComplaint": 0.1, "providerErrors": 5, "scope": "route"}}).MustStatus(t, 422)
	linh.Do("PATCH", ws+"/campaigns/"+flash+"/send-settings", map[string]any{"thresholds": thresholds,
		"schedule": map[string]any{"mode": "at", "date": "2030-01-02", "time": "09:00", "timezone": "Mars/Base"}}).MustStatus(t, 422)
	settings := n.audit("campaign.send_settings_updated", flash)
	linh.Do("PATCH", ws+"/campaigns/"+flash+"/send-settings", map[string]any{"thresholds": thresholds,
		"schedule": map[string]any{"mode": "at", "date": "2030-01-02", "time": "09:00", "timezone": "America/New_York"}}).MustStatus(t, 204)
	var rv struct {
		Thresholds struct {
			Bounce float64
			Scope  string
		}
		Schedule struct{ Mode, Date, Time string }
		Stale    bool
	}
	linh.Do("GET", ws+"/campaigns/"+flash+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
	// A new time keeps the checks: they still describe the campaign.
	if rv.Thresholds.Bounce != 1.5 || rv.Thresholds.Scope != "campaign" || rv.Schedule.Mode != "at" || rv.Schedule.Time != "09:00" || rv.Stale {
		t.Fatalf("settings: %+v", rv)
	}
	if n.audit("campaign.send_settings_updated", flash) != settings+1 {
		t.Fatal("settings are audited")
	}
}

// The send-time recheck the sending engine asks for: it holds nothing and
// tells nobody, names what blocks, blocks a send whose approval is still
// pending, and hands over Send Readiness's mailbox throttles; a hold the
// engine records shows its checks from that run on Review & send.
func TestSendRecheck(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	n := counter{t, env}
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	recheck := app.SendRecheck(env.App)
	flash := uuid.MustParse(campaignID(t, linh, "Flash sale: 48 hours only"))
	digest := uuid.MustParse(campaignID(t, linh, "Weekly Digest #142"))
	nov := uuid.MustParse(campaignID(t, linh, "November Product Update"))
	holds := func(id uuid.UUID) int {
		t.Helper()
		var c int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'recheck_blocked'`, id).Scan(&c); err != nil {
			t.Fatal(err)
		}
		return c
	}

	before, told := holds(flash), n.notes("linh@acme.io", "send_blocked")
	r, err := recheck(ctx, flash)
	if err != nil {
		t.Fatal(err)
	}
	if r.Verdict != "blocked" || len(r.Blocking) == 0 || !slices.Contains(r.Blocking, "Compliance: the email body has no unsubscribe link or postal address") {
		t.Fatalf("flash sale: %+v", r)
	}
	if holds(flash) != before || n.notes("linh@acme.io", "send_blocked") != told {
		t.Fatal("the recheck records no hold and tells nobody")
	}

	r, err = recheck(ctx, digest)
	if err != nil {
		t.Fatal(err)
	}
	if r.Verdict == "blocked" || r.Throttles["outlook"] != 5 || r.Throttles["yahoo"] != 4 || r.Throttles["icloud"] != 3 || r.Throttles["gmail"] != 0 {
		t.Fatalf("weekly digest: %+v", r)
	}
	// A hold the engine records from it counts these recipients.
	if r.Audience == 0 || r.RunID == "" {
		t.Fatalf("the recheck should carry its run and recipients: %+v", r)
	}

	// November still waits for Marcus or Linh.
	r, err = recheck(ctx, nov)
	if err != nil {
		t.Fatal(err)
	}
	if r.Verdict != "blocked" || !slices.Contains(r.Blocking, "Approval is still pending") {
		t.Fatalf("november: %+v", r)
	}

	// A hold the engine recorded (only the titles of what blocked): the
	// review shows the checks of the send-time run and the recipients held.
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO campaign_incidents (id, campaign_id, kind, scope, metric, detail, tripped_at)
		VALUES ($1, $2, 'recheck_blocked', 'campaign', 'readiness', '{"blocking": ["Links: go.acme-promo.link is listed on SURBL and Spamhaus DBL"]}', now())`,
		uuid.Must(uuid.NewV7()), flash); err != nil {
		t.Fatal(err)
	}
	var rv struct {
		Held *struct {
			Reason string
			Held   int64
			Blocks []struct{ Key, Status string }
		}
	}
	linh.Do("GET", ws+"/campaigns/"+flash.String()+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
	if rv.Held == nil || rv.Held.Reason != "Links: go.acme-promo.link is listed on SURBL and Spamhaus DBL" || rv.Held.Held == 0 || len(rv.Held.Blocks) == 0 || rv.Held.Blocks[0].Status != "block" {
		t.Fatalf("held by the engine: %+v", rv.Held)
	}
}

// Sending now starts on the run the person just confirmed (it must be
// recent): it is handed to the sending engine, which doesn't run the checks
// again, and nothing runs inside the request (SR-8).
func TestSendNowRunsReadinessOnce(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	starter := &fakeStarter{}
	env.App.Sending = starter
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	digest := campaignID(t, linh, "Weekly Digest #142")
	linh.Do("POST", ws+"/campaigns/"+digest+"/unschedule", nil).MustStatus(t, 204)
	r := getReview(t, linh, digest)
	if r.Run == nil || r.Run.Verdict == "blocked" || r.Stale {
		t.Fatalf("weekly digest after unscheduling: %+v", r)
	}
	sendTimeRuns := func() (n int, latest string) {
		t.Helper()
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*), coalesce((SELECT id::text FROM readiness_runs WHERE campaign_id = $1 AND trigger = 'send_time' ORDER BY ran_at DESC LIMIT 1), '')
			FROM readiness_runs WHERE campaign_id = $1 AND trigger = 'send_time'`, digest).Scan(&n, &latest); err != nil {
			t.Fatal(err)
		}
		return n, latest
	}
	send := func(run string, want int) *apptest.Response {
		return linh.Do("POST", ws+"/campaigns/"+digest+"/bypass", map[string]any{"runId": run, "acknowledged": warnings(r),
			"confirm": fmt.Sprint(r.Run.Estimates.Audience.Final), "schedule": map[string]any{"mode": "now"},
			"reason": "Customer-facing outage notice, approved verbally by Marcus.", "password": "reputecast-demo"}).MustStatus(t, want)
	}
	// The seeded run is hours old: for a send that starts now the checks
	// run again first, in the background (SR-8), and nothing is sent.
	if p := send(r.Run.ID, 409).Problem(t); p["code"] != "readiness_old" {
		t.Fatalf("an old run: %v", p)
	}
	if len(starter.rechecks) != 0 {
		t.Fatal("the send started on an old run")
	}
	readinesstest.Work(t, env)
	r = getReview(t, linh, digest)
	before, _ := sendTimeRuns()
	var runs int
	countRuns := func() int {
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM readiness_runs WHERE campaign_id = $1`, digest).Scan(&runs); err != nil {
			t.Fatal(err)
		}
		return runs
	}
	stored := countRuns()
	var sent struct {
		Status     string
		Recipients int64
	}
	send(r.Run.ID, 200).JSON(t, &sent)
	if sent.Status != "sending" {
		t.Fatalf("sent: %+v", sent)
	}
	// The run the person confirmed is the send's check: nothing runs
	// inside the request, and the engine starts on that run.
	if after, _ := sendTimeRuns(); after != before || countRuns() != stored {
		t.Fatalf("a run inside the request: %d runs, then %d", stored, runs)
	}
	if len(starter.rechecks) != 1 || starter.rechecks[0].RunID != r.Run.ID || starter.rechecks[0].Verdict == "blocked" || starter.rechecks[0].Audience == 0 {
		t.Fatalf("the engine should start on the confirmed run %s: %+v", r.Run.ID, starter.rechecks)
	}
}

// A scheduled send keeps its time when an edit invalidates its approval,
// but it doesn't start until approvers approve the new version: the
// author can ask again while the time stays reserved, and a request
// approvers send back holds the send too.
func TestScheduledSendLosesItsApproval(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Sending = &fakeStarter{}
	ctx := context.Background()
	recheck := app.SendRecheck(env.App)
	marcus, daniel := env.Client(), env.Client()
	marcus.SignInDemo("marcus@acme.io")
	daniel.SignInDemo("daniel@acme.io")
	nov := campaignID(t, daniel, "November Product Update")
	novID := uuid.MustParse(nov)
	blocking := func(want string) {
		t.Helper()
		r, err := recheck(ctx, novID)
		if err != nil {
			t.Fatal(err)
		}
		if want == "" && r.Verdict == "blocked" || want != "" && (r.Verdict != "blocked" || !slices.Contains(r.Blocking, want)) {
			t.Fatalf("recheck: want %q, got %+v", want, r)
		}
	}

	// Approved, then scheduled by Daniel on checks that ran just now (what
	// he confirms is what the send-time recheck compares with, SR-3).
	r := getReview(t, daniel, nov)
	marcus.Do("POST", ws+"/approvals/"+r.Approval.Request.ID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 200)
	readinesstest.Post(t, env, daniel, ws+"/campaigns/"+nov+"/readiness").MustStatus(t, 200)
	r = getReview(t, daniel, nov)
	daniel.Do("POST", ws+"/approvals/"+r.Approval.Request.ID+"/schedule", map[string]any{"runId": r.Run.ID, "acknowledged": warnings(r),
		"confirm": fmt.Sprint(r.Run.Estimates.Audience.Final)}).MustStatus(t, 200)
	blocking("")

	// The subject changes: the approval is invalidated, the time stays.
	var full map[string]any
	daniel.Do("GET", ws+"/campaigns/"+nov, nil).MustStatus(t, 200).JSON(t, &full)
	daniel.Do("PUT", ws+"/campaigns/"+nov+"/content", map[string]any{"baseVersion": full["version"], "subject": "November Product Update: faster exports, a new API and self-hosted sending",
		"preheader": full["preheader"], "mjml": full["mjml"], "html": full["html"], "label": "subject", "source": "manual"}).MustStatus(t, 200)
	r = getReview(t, daniel, nov)
	if r.Campaign.Status != "scheduled" || r.Approval.Request == nil || r.Approval.Request.Status != "invalidated" {
		t.Fatalf("invalidated, still scheduled: %s %+v", r.Campaign.Status, r.Approval.Request)
	}
	blocking("The approval no longer covers the campaign")

	// Daniel asks again; the time stays reserved while Marcus decides.
	var asked struct{ ID string }
	daniel.Do("POST", ws+"/campaigns/"+nov+"/approval", map[string]any{"note": "Only the subject changed."}).MustStatus(t, 201).JSON(t, &asked)
	if r = getReview(t, daniel, nov); r.Campaign.Status != "scheduled" || r.Approval.Request.Status != "pending" {
		t.Fatalf("asked again: %s %+v", r.Campaign.Status, r.Approval.Request)
	}
	blocking("Approval is still pending")
	marcus.Do("POST", ws+"/approvals/"+asked.ID+"/decision", map[string]any{"decision": "request_changes", "comment": "Keep the Postal milestone in the subject."}).MustStatus(t, 200)
	blocking("An approver asked for changes")

	// Asked once more and approved: the send can start at its time.
	daniel.Do("POST", ws+"/campaigns/"+nov+"/approval", map[string]any{}).MustStatus(t, 201).JSON(t, &asked)
	marcus.Do("POST", ws+"/approvals/"+asked.ID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 200)
	if r = getReview(t, daniel, nov); r.Campaign.Status != "scheduled" || !r.Approval.Valid {
		t.Fatalf("approved again: %s %+v", r.Campaign.Status, r.Approval)
	}
	blocking("")
}

// Publishing an automation can need approval too (A5 §3.8): the publish
// waits for an approver, and their approval puts the version live.
func TestAutomationPublishApproval(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	n := counter{t, env}
	linh, marcus := env.Client(), env.Client()
	linh.SignInDemo("linh@acme.io")
	marcus.SignInDemo("marcus@acme.io")

	var policy struct{ Policy map[string]any }
	linh.Do("GET", ws+"/approval-policy", nil).MustStatus(t, 200).JSON(t, &policy)
	policy.Policy["automationPublish"] = true
	linh.Do("PUT", ws+"/approval-policy", policy.Policy).MustStatus(t, 200)

	var autos struct {
		Items []struct{ ID, Name string }
	}
	linh.Do("GET", ws+"/automations", nil).MustStatus(t, 200).JSON(t, &autos)
	welcome := ""
	for _, a := range autos.Items {
		if a.Name == "Welcome series" {
			welcome = a.ID
		}
	}
	if welcome == "" {
		t.Fatal("no Welcome series")
	}
	requested := n.notes("marcus@acme.io", "approval_requested")
	var res struct{ Status, ApprovalID string }
	linh.Do("POST", ws+"/automations/"+welcome+"/publish", map[string]any{"version": 4, "migration": "finish"}).MustStatus(t, 200).JSON(t, &res)
	if res.Status != "approval_required" || res.ApprovalID == "" || n.notes("marcus@acme.io", "approval_requested") != requested+1 {
		t.Fatalf("publish waits for approval: %+v", res)
	}
	var live struct{ LiveVersion *int }
	linh.Do("GET", ws+"/automations/"+welcome, nil).MustStatus(t, 200).JSON(t, &live)
	if live.LiveVersion == nil || *live.LiveVersion != 3 {
		t.Fatalf("still v3 live: %v", live.LiveVersion)
	}
	// The publisher can't approve their own version; Marcus can.
	linh.Do("POST", ws+"/approvals/"+res.ApprovalID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 403)
	approved := n.audit("automation.publish_approved", welcome)
	marcus.Do("POST", ws+"/approvals/"+res.ApprovalID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 200)
	linh.Do("GET", ws+"/automations/"+welcome, nil).MustStatus(t, 200).JSON(t, &live)
	if live.LiveVersion == nil || *live.LiveVersion != 4 || n.audit("automation.publish_approved", welcome) != approved+1 {
		t.Fatalf("approved and live: %v", live.LiveVersion)
	}
}

// Review & send and approvals queue their webhook events (approval.requested,
// approval.decided, campaign.scheduled, and campaign.paused when the
// send-time checks hold a send) for the endpoints that subscribe.
func TestApprovalWebhookEvents(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Sending = &fakeStarter{}
	ctx := context.Background()
	marcus, daniel, linh := env.Client(), env.Client(), env.Client()
	marcus.SignInDemo("marcus@acme.io")
	daniel.SignInDemo("daniel@acme.io")
	linh.SignInDemo("linh@acme.io")
	nov := campaignID(t, daniel, "November Product Update")

	// An endpoint for every approval and campaign event, marked as demo data
	// so nothing is ever sent to it.
	hook := uuid.Must(uuid.NewV7())
	var wsID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO webhooks (id, workspace_id, name, url, events, secret_enc, demo) VALUES ($1, $2, 'Approvals', 'https://hooks.example.com/approvals', ARRAY['approval.*', 'campaign.*'], '\x00', true)`,
		hook, wsID); err != nil {
		t.Fatal(err)
	}
	queued := func(event string) int {
		t.Helper()
		var c int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM webhook_deliveries WHERE webhook_id = $1 AND event = $2`, hook, event).Scan(&c); err != nil {
			t.Fatal(err)
		}
		return c
	}

	r := getReview(t, daniel, nov)
	marcus.Do("POST", ws+"/approvals/"+r.Approval.Request.ID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 200)
	if queued("approval.decided") != 1 {
		t.Fatal("approval.decided is queued")
	}
	r = getReview(t, daniel, nov)
	daniel.Do("POST", ws+"/approvals/"+r.Approval.Request.ID+"/schedule", map[string]any{"runId": r.Run.ID, "acknowledged": warnings(r),
		"confirm": fmt.Sprint(r.Run.Estimates.Audience.Final)}).MustStatus(t, 200)
	if queued("campaign.scheduled") != 1 {
		t.Fatal("campaign.scheduled is queued")
	}

	// The post's domain appears on a URI blocklist before the send: the
	// send-time check holds the send, tells people and queues campaign.paused.
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO link_domain_listings (domain, lists, source, checked_at) VALUES ('acme.io', ARRAY['SURBL'], 'test', now())
		ON CONFLICT (domain) DO UPDATE SET lists = EXCLUDED.lists`); err != nil {
		t.Fatal(err)
	}
	err := app.HeldSendCheck(env.App)(ctx, wsID, uuid.MustParse(nov))
	if !errors.Is(err, sending.ErrNotReady) {
		t.Fatalf("held: %v", err)
	}
	var rv struct {
		Held *struct {
			Reason string
			Blocks []struct{ Key string }
		}
	}
	linh.Do("GET", ws+"/campaigns/"+nov+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
	if rv.Held == nil || len(rv.Held.Blocks) == 0 || queued("campaign.paused") != 1 {
		t.Fatalf("held send: %+v, %d campaign.paused", rv.Held, queued("campaign.paused"))
	}

	// Asking again queues approval.requested.
	var full map[string]any
	daniel.Do("GET", ws+"/campaigns/"+nov, nil).MustStatus(t, 200).JSON(t, &full)
	daniel.Do("PUT", ws+"/campaigns/"+nov+"/content", map[string]any{"baseVersion": full["version"], "subject": "November Product Update: faster exports and a new API",
		"preheader": full["preheader"], "mjml": full["mjml"], "html": full["html"], "label": "subject", "source": "manual"}).MustStatus(t, 200)
	if _, err := env.App.Pool.Exec(ctx, `DELETE FROM link_domain_listings WHERE domain = 'acme.io'`); err != nil {
		t.Fatal(err)
	}
	readinesstest.Post(t, env, daniel, ws+"/campaigns/"+nov+"/readiness").MustStatus(t, 200)
	daniel.Do("POST", ws+"/campaigns/"+nov+"/approval", map[string]any{}).MustStatus(t, 201)
	if queued("approval.requested") != 1 {
		t.Fatal("approval.requested is queued")
	}
	var attempted int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM webhook_deliveries WHERE webhook_id = $1 AND status_code IS NOT NULL`, hook).Scan(&attempted); err != nil {
		t.Fatal(err)
	}
	if attempted != 0 {
		t.Fatalf("a demo endpoint is never called: %d attempts", attempted)
	}
}
