package approvals_test

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// SR-13: Review & send names people, never permission keys: the Owners who
// can use the emergency send (the More menu) and whether the policy allows
// it; and says whether the auto-pause thresholds are the workspace's
// defaults or this send's own.
func TestReviewSaysWhoCan(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	sofia := env.Client()
	sofia.SignInDemo("sofia@acme.io")
	digest := campaignID(t, sofia, "Weekly Digest #142")
	var r struct {
		Approval struct {
			EmergencyBypass bool
			Owners          []string
		}
		ThresholdsOwn bool
	}
	sofia.Do("GET", ws+"/campaigns/"+digest+"/review", nil).MustStatus(t, 200).JSON(t, &r)
	if len(r.Approval.Owners) != 1 || r.Approval.Owners[0] != "Linh Nguyen" || r.ThresholdsOwn {
		t.Fatalf("as Sofia: %+v", r)
	}
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var policy struct{ Policy map[string]any }
	linh.Do("GET", ws+"/approval-policy", nil).MustStatus(t, 200).JSON(t, &policy)
	allowed, _ := policy.Policy["emergencyBypass"].(bool)
	linh.Do("GET", ws+"/campaigns/"+digest+"/review", nil).MustStatus(t, 200).JSON(t, &r)
	if len(r.Approval.Owners) != 0 || r.Approval.EmergencyBypass != allowed {
		t.Fatalf("as Linh, the only Owner: %+v (policy allows: %v)", r.Approval, allowed)
	}
	linh.Do("PATCH", ws+"/campaigns/"+digest+"/send-settings", map[string]any{"thresholds": map[string]any{"bounce": 1.5, "complaint": 0.1, "mailboxComplaint": 0.1,
		"providerErrors": 5, "scope": "route"}}).MustStatus(t, 204)
	linh.Do("GET", ws+"/campaigns/"+digest+"/review", nil).MustStatus(t, 200).JSON(t, &r)
	if !r.ThresholdsOwn {
		t.Fatal("this send's own thresholds")
	}

	// A Viewer can't ask for approval: the reason names who can change the
	// campaign instead of a permission key (X-1).
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	var v struct {
		Access   struct{ EditHelpers, SendHelpers []string }
		Approval struct{ RequestReason string }
	}
	aisha.Do("GET", ws+"/campaigns/"+digest+"/review", nil).MustStatus(t, 200).JSON(t, &v)
	reason := v.Approval.RequestReason
	if len(v.Access.EditHelpers) == 0 || len(v.Access.SendHelpers) == 0 || !strings.HasPrefix(reason, "Your role can't change this campaign, so it can't ask for approval. Ask ") ||
		!strings.Contains(reason, v.Access.EditHelpers[0]) || strings.Contains(reason, "campaigns:") {
		t.Fatalf("as Aisha, a Viewer: %+v", v)
	}
}
