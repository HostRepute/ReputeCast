package approvals_test

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

type txnApproval struct {
	Required, CoversDraft, CanRequest bool
	Status, RequestID, RequestReason  string
	Approvers                         []string
}

// The optional rule "Publishing a transactional template needs approval":
// off by default; on, a publish waits for an approver other than the
// publisher, the approval covers that draft only, and rolling back needs
// none.
func TestTransactionalPublishApproval(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	n := counter{t, env}
	linh, marcus, daniel, aisha := env.Client(), env.Client(), env.Client(), env.Client()
	linh.SignInDemo("linh@acme.io")
	marcus.SignInDemo("marcus@acme.io")
	daniel.SignInDemo("daniel@acme.io")
	aisha.SignInDemo("aisha@acme.io")

	var tpl struct {
		ID            string
		DraftRevision int
	}
	linh.Do("POST", ws+"/transactional/templates", map[string]any{"name": "Approval test", "key": "approval-test", "starter": "password-reset"}).MustStatus(t, 201).JSON(t, &tpl)
	base := ws + "/transactional/templates/" + tpl.ID

	// Off by default: publishing goes ahead.
	var st txnApproval
	linh.Do("GET", base+"/approval", nil).MustStatus(t, 200).JSON(t, &st)
	if st.Required || st.CanRequest || st.Status != "none" {
		t.Fatalf("rule off: %+v", st)
	}
	linh.Do("POST", base+"/publish", map[string]any{"baseRevision": tpl.DraftRevision}).MustStatus(t, 200)
	linh.Do("POST", base+"/approval", map[string]any{}).MustStatus(t, 409)

	var policy struct{ Policy map[string]any }
	linh.Do("GET", ws+"/approval-policy", nil).MustStatus(t, 200).JSON(t, &policy)
	if policy.Policy["transactionalPublish"] != false {
		t.Fatalf("the rule is off by default: %+v", policy.Policy)
	}
	policy.Policy["transactionalPublish"] = true
	linh.Do("PUT", ws+"/approval-policy", policy.Policy).MustStatus(t, 200)

	// A new draft: publishing it needs Marcus (or another approver).
	save := func(subject string) int {
		t.Helper()
		var cur struct{ DraftRevision int }
		linh.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &cur)
		var saved struct{ DraftRevision int }
		linh.Do("PUT", base+"/draft", map[string]any{"baseRevision": cur.DraftRevision, "subject": subject}).MustStatus(t, 200).JSON(t, &saved)
		return saved.DraftRevision
	}
	rev := save("Reset your password, {{ first_name | default: \"there\" }}")
	var preview struct {
		CanPublish bool
		Approval   struct {
			Required  bool
			Approvers []string
			Reason    string
		}
	}
	linh.Do("POST", base+"/publish-preview", map[string]any{}).MustStatus(t, 200).JSON(t, &preview)
	if preview.CanPublish || !preview.Approval.Required || len(preview.Approval.Approvers) == 0 {
		t.Fatalf("the dialog asks for approval: %+v", preview)
	}
	for _, a := range preview.Approval.Approvers {
		if a == "Linh Nguyen" {
			t.Fatal("the publisher can't approve their own change")
		}
	}
	linh.Do("POST", base+"/publish", map[string]any{"baseRevision": rev}).MustStatus(t, 409)

	// Daniel can't publish, so he can't ask; Aisha only reads.
	daniel.Do("POST", base+"/approval", map[string]any{}).MustStatus(t, 403)
	aisha.Do("GET", base+"/approval", nil).MustStatus(t, 200).JSON(t, &st)
	if st.CanRequest || st.RequestReason == "" {
		t.Fatalf("viewer: %+v", st)
	}

	requested := n.notes("marcus@acme.io", "approval_requested")
	linh.Do("POST", base+"/approval", map[string]any{"note": "New subject"}).MustStatus(t, 201).JSON(t, &st)
	if st.Status != "pending" || !st.CoversDraft || st.RequestID == "" || n.notes("marcus@acme.io", "approval_requested") != requested+1 {
		t.Fatalf("requested: %+v", st)
	}
	if n.audit("transactional.publish_approval_requested", tpl.ID) != 1 {
		t.Fatal("the request is audited")
	}
	linh.Do("POST", base+"/publish", map[string]any{"baseRevision": rev}).MustStatus(t, 409)

	// The approval inbox lists it with the template.
	var inbox struct {
		Items []struct{ ID, SubjectType, TransactionalTemplateID, Name string }
	}
	marcus.Do("GET", ws+"/approvals", nil).MustStatus(t, 200).JSON(t, &inbox)
	found := false
	for _, it := range inbox.Items {
		if it.ID == st.RequestID {
			found = it.SubjectType == "transactional_template" && it.TransactionalTemplateID == tpl.ID && it.Name == "Approval test"
		}
	}
	if !found {
		t.Fatalf("inbox: %+v", inbox.Items)
	}
	var detail struct {
		Transactional *struct {
			Key           string
			Version       int
			SubjectBefore string
			SubjectAfter  string
		}
	}
	marcus.Do("GET", ws+"/approvals/"+st.RequestID, nil).MustStatus(t, 200).JSON(t, &detail)
	if detail.Transactional == nil || detail.Transactional.Key != "approval-test" || detail.Transactional.Version != 2 || detail.Transactional.SubjectAfter == "" {
		t.Fatalf("review: %+v", detail.Transactional)
	}

	// Linh can't approve her own change; Marcus approves; publishing goes ahead.
	linh.Do("POST", ws+"/approvals/"+st.RequestID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 403)
	decided := n.notes("linh@acme.io", "approval_decided")
	marcus.Do("POST", ws+"/approvals/"+st.RequestID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 200)
	if n.audit("transactional.publish_approved", tpl.ID) != 1 || n.notes("linh@acme.io", "approval_decided") != decided+1 {
		t.Fatal("the decision is audited and the publisher told")
	}
	linh.Do("GET", base+"/approval", nil).MustStatus(t, 200).JSON(t, &st)
	if st.Status != "approved" || !st.CoversDraft {
		t.Fatalf("approved: %+v", st)
	}
	var published struct{ Version int }
	linh.Do("POST", base+"/publish", map[string]any{"baseRevision": rev}).MustStatus(t, 200).JSON(t, &published)
	if published.Version != 2 {
		t.Fatalf("published v%d", published.Version)
	}

	// A changed draft needs a new approval; rolling back to v1 doesn't.
	rev = save("Reset your Acme Cloud password now")
	linh.Do("GET", base+"/approval", nil).MustStatus(t, 200).JSON(t, &st)
	if st.CoversDraft {
		t.Fatalf("the approval covered the earlier draft: %+v", st)
	}
	linh.Do("POST", base+"/publish", map[string]any{"baseRevision": rev}).MustStatus(t, 409)
	linh.Do("POST", base+"/rollback", map[string]any{"version": 1}).MustStatus(t, 200)
}
