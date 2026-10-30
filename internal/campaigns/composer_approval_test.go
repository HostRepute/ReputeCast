package campaigns_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// CO-7: saving content into an approved campaign cancels its approval, so
// it takes the person's say-so: without cancelApproval the save is a 409
// that names the approver; with it the save goes through, the approval is
// invalidated and the save's audit entry names the request.
func TestComposerEditingAnApprovedCampaignAsksFirst(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var d detail
	c.Do("POST", ws+"/campaigns", map[string]any{"name": "Approved one"}).MustStatus(t, 201).JSON(t, &d)
	doc := `<mjml><mj-body><mj-section><mj-column><mj-text>Approved words</mj-text></mj-column></mj-section></mj-body></mjml>`
	var saved struct{ Version int }
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 1, "subject": "Approved", "mjml": doc}).MustStatus(t, 200).JSON(t, &saved)

	var wsID, marcus uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, u.id FROM workspaces w, users u WHERE w.slug = 'acme-newsletter' AND u.email = 'marcus@acme.io'`).Scan(&wsID, &marcus); err != nil {
		t.Fatal(err)
	}
	req := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO approval_requests (id, workspace_id, campaign_id, requested_by, status, snapshot, campaign_version, decided_by, decided_at)
		SELECT $1, $2, $3, u.id, 'approved', '{}', $4, $5, now() FROM users u WHERE u.email = 'linh@acme.io'`, req, wsID, d.ID, saved.Version, marcus); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO approval_decisions (id, request_id, approver_id, decision) VALUES ($1, $2, $3, 'approve')`, uuid.Must(uuid.NewV7()), req, marcus); err != nil {
		t.Fatal(err)
	}

	type approval struct {
		ID, Status, RequestedBy string
		ApprovedBy              []string
		Version                 int
	}
	var got struct{ Approval *approval }
	c.Do("GET", ws+"/campaigns/"+d.ID, nil).MustStatus(t, 200).JSON(t, &got)
	if a := got.Approval; a == nil || a.Status != "approved" || a.Version != saved.Version || len(a.ApprovedBy) != 1 || a.ApprovedBy[0] != "Marcus Chen" || a.RequestedBy != "Linh Nguyen" {
		t.Fatalf("approval: %+v", got.Approval)
	}

	edited := map[string]any{"baseVersion": saved.Version, "subject": "Approved, then edited", "mjml": doc}
	var problem struct {
		Code   string
		Detail string
		Meta   map[string]any
	}
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", edited).MustStatus(t, 409).JSON(t, &problem)
	if problem.Code != "approval_would_cancel" || problem.Meta["status"] != "approved" || problem.Detail == "" {
		t.Fatalf("409: %+v", problem)
	}
	// Saving the content it already has asks nothing: nothing changes.
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": saved.Version, "subject": "Approved", "mjml": doc}).MustStatus(t, 200)

	edited["cancelApproval"] = true
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", edited).MustStatus(t, 200)
	var status string
	if err := env.App.Pool.QueryRow(ctx, `SELECT status FROM approval_requests WHERE id = $1`, req).Scan(&status); err != nil || status != "invalidated" {
		t.Fatalf("the approval should be invalidated: %q %v", status, err)
	}
	var meta json.RawMessage
	if err := env.App.Pool.QueryRow(ctx, `SELECT metadata FROM audit_events WHERE action = 'campaign.content.save' AND target_id = $1 ORDER BY occurred_at DESC LIMIT 1`, d.ID).Scan(&meta); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(meta, &m)
	if m["cancelsApproval"] != req.String() {
		t.Fatalf("audit metadata: %s", meta)
	}
	got.Approval = nil
	c.Do("GET", ws+"/campaigns/"+d.ID, nil).MustStatus(t, 200).JSON(t, &got)
	if got.Approval != nil {
		t.Fatalf("no open approval after the edit: %+v", got.Approval)
	}
	// Editing goes on without asking again.
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": saved.Version + 1, "subject": "Edited twice", "mjml": doc}).MustStatus(t, 200)
}
