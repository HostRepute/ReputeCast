package approvals_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// fakeStarter stands in for the sending engine.
type fakeStarter struct {
	mu       sync.Mutex
	started  []uuid.UUID
	canceled []uuid.UUID
}

func (f *fakeStarter) Start(_ context.Context, id, _ uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, id)
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

	// The sample data: Weekly Digest approved and ready with two warnings,
	// the flash sale blocked, November waiting for Marcus or Linh.
	r := getReview(t, linh, digest)
	if r.Run == nil || r.Run.Verdict != "warnings" || r.Run.Counts.Warnings != 2 || r.Stale || !r.Approval.Valid {
		t.Fatalf("weekly digest: %+v", r)
	}
	if r.Run.Estimates.Audience.Final != 31604 {
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
	if inbox.Counts.Pending != 1 || inbox.Counts.Approved != 3 || inbox.Counts.ChangesRequested != 1 || inbox.Counts.Invalidated != 1 || inbox.Counts.All != 6 {
		t.Fatalf("inbox counts: %+v", inbox.Counts)
	}
	if inbox.Policy.MedianWaitMinutes == nil || *inbox.Policy.MedianWaitMinutes != 160 {
		t.Fatalf("median wait: %v", inbox.Policy.MedianWaitMinutes)
	}
	daniel.Do("GET", ws+"/approvals?status=all", nil).MustStatus(t, 200).JSON(t, &inbox)
	if inbox.Counts.All != 3 {
		t.Fatalf("Daniel sees his own requests: %+v", inbox.Items)
	}
	marcus.Do("GET", ws+"/approvals?status=pending", nil).MustStatus(t, 200).JSON(t, &inbox)
	if len(inbox.Items) != 1 || inbox.Items[0].Name != "November Product Update" {
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
	linh.Do("POST", ws+"/campaigns/"+digest+"/readiness", nil).MustStatus(t, 200).JSON(t, &run)
	if run.ID == "" || run.Counts.Total < 20 || len(run.Groups) != 7 {
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
	linh.Do("POST", ws+"/campaigns/"+flash+"/readiness/fixes", map[string]any{"fixes": []map[string]any{{"action": "insert_footer"}, {"action": "exclude_sunset"}}}).MustStatus(t, 200).JSON(t, &out)
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
	if rv.Thresholds.Bounce != 1.5 || rv.Thresholds.Scope != "campaign" || rv.Schedule.Mode != "at" || rv.Schedule.Time != "09:00" || !rv.Stale {
		t.Fatalf("settings: %+v", rv)
	}
	if n.audit("campaign.send_settings_updated", flash) != settings+1 {
		t.Fatal("settings are audited")
	}
}
