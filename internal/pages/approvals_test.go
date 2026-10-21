package pages_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/hostrepute/reputecast/internal/pages"
)

// Approvals for publishing pages: with "Publishing a post or page needs
// approval" on, Publish asks for one (409 approval_required); another
// approver's approval puts the page live as it was asked; a change after
// it needs a new one. Under "The author lacks the send permission" a
// Content Writer asks for approval, and a change invalidates the request.
func TestPagePublishApprovals(t *testing.T) {
	f := newFixture(t)
	f.exec(`UPDATE workspaces SET approval_policy = jsonb_set(approval_policy, '{post_publish}', 'true') WHERE id = $1`, f.ws.ID)
	d := f.page("Partner program", plain("Partner program", "Earn a fifth of every referral."))
	path := pagesAPI + "/" + d.ID

	p := mustProblem(t, f.owner.Do(http.MethodPost, path+"/publish", map[string]any{"when": "now"}), http.StatusConflict, "approval_required")
	var approvers []string
	for _, a := range p.Meta["approvers"].([]any) {
		approvers = append(approvers, a.(string))
	}
	if p.Meta["rule"] != "post_publish" || !slices.Contains(approvers, "Marcus Chen") || slices.Contains(approvers, "Linh Nguyen") {
		t.Fatalf("approval required: %+v", p)
	}
	if got := f.detail(f.owner, d.ID); !got.Publish.NeedsApproval || got.Publish.CanPublish || got.Publish.Rule != "post_publish" || got.Status != pages.StatusDraft {
		t.Fatalf("the page's publish state: %+v", got.Publish)
	}

	var asked pages.PageDetail
	f.owner.Do(http.MethodPost, path+"/approval", map[string]any{"note": "Ready for review"}).MustStatus(t, http.StatusCreated).JSON(t, &asked)
	if asked.Publish.Pending == nil || asked.Publish.Pending.RequestedBy != "Linh Nguyen" {
		t.Fatalf("pending: %+v", asked.Publish)
	}
	request := asked.Publish.Pending.ID
	if n := f.audits("page.publish_approval_requested", d.ID); n != 1 {
		t.Errorf("%d approval request audit entries", n)
	}
	decision := "/workspaces/acme-newsletter/approvals/" + request + "/decision"
	if r := f.owner.Do(http.MethodPost, decision, map[string]any{"decision": "approve"}); r.Status < 400 {
		t.Fatalf("the author approved her own page: %d", r.Status)
	}
	marcus := f.as("marcus@acme.io")
	marcus.Do(http.MethodPost, decision, map[string]any{"decision": "approve"}).MustStatus(t, http.StatusOK)
	got := f.detail(f.owner, d.ID)
	if got.Status != pages.StatusPublished || got.Live == nil || got.Publish.Pending != nil {
		t.Fatalf("after the approval: %s %+v", got.Status, got.Live)
	}
	if res := f.resolve("partner-program"); res.Page == nil {
		t.Fatal("the approved page isn't live")
	}
	if name := scan[string](f, `SELECT actor_name FROM audit_events WHERE action = 'page.published' AND target_id = $1`, d.ID); name != "Approved by Marcus Chen" {
		t.Errorf("published by %q", name)
	}
	if n := f.audits("page.publish_approved", d.ID); n != 1 {
		t.Errorf("%d page.publish_approved entries", n)
	}
	// A change needs a new approval.
	f.save(f.owner, d.ID, got.Draft.Version, plain("Partner program", "Earn a quarter of every referral."))
	mustProblem(t, f.owner.Do(http.MethodPost, path+"/publish", map[string]any{"when": "now"}), http.StatusConflict, "approval_required")
	if res := f.resolve("partner-program"); res.Page == nil || res.Page.Version != got.Live.Version {
		t.Fatal("the change went live without an approval")
	}

	// A Content Writer under "The author lacks the send permission".
	f.exec(`UPDATE workspaces SET approval_policy = jsonb_set(approval_policy, '{post_publish}', 'false') WHERE id = $1`, f.ws.ID)
	daniel := f.as("daniel@acme.io")
	dd := f.create(daniel, pages.TemplateBlank, "Sponsor FAQ", "")
	if st := f.detail(daniel, dd.ID).Publish; !st.NeedsApproval || st.CanPublish || st.Rule != "author_lacks_send" {
		t.Fatalf("Daniel's publish state: %+v", st)
	}
	daniel.Do(http.MethodPost, pagesAPI+"/"+dd.ID+"/approval", map[string]any{}).MustStatus(t, http.StatusCreated).JSON(t, &asked)
	if asked.Publish.Pending == nil || asked.Publish.Pending.RequestedBy != "Daniel Park" {
		t.Fatalf("Daniel's request: %+v", asked.Publish)
	}
	f.save(daniel, dd.ID, asked.Draft.Version, plain("Sponsor FAQ", "Changed after the request."))
	if st := f.detail(daniel, dd.ID).Publish; st.Pending != nil {
		t.Errorf("a change kept the request pending: %+v", st.Pending)
	}
	if s := scan[string](f, `SELECT status FROM approval_requests WHERE id = $1`, asked.Publish.Pending.ID); s != "invalidated" {
		t.Errorf("the request after a change: %s", s)
	}
}
