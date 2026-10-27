package approvals_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness/readinesstest"
)

// A member who reads Vietnamese gets approvals' texts in Vietnamese:
// problems, an invalidation (stored while everyone read English) with its
// reason and changes, the inbox's lines and a request's rules. Requests
// stored before their texts were kept read in English, "after approval"
// said once.
func TestApprovalsInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	marcus, daniel, viMarcus, viDaniel := env.Client(), env.Client(), env.Client(), env.Client()
	marcus.SignInDemo("marcus@acme.io")
	daniel.SignInDemo("daniel@acme.io")
	viMarcus.SignInDemo("marcus@acme.io")
	viDaniel.SignInDemo("daniel@acme.io")
	viMarcus.Header = http.Header{"Accept-Language": {"vi"}}
	viDaniel.Header = http.Header{"Accept-Language": {"vi"}}

	// A problem detail.
	p := viDaniel.Do("GET", ws+"/approvals/"+uuid.NewString(), nil).MustStatus(t, 404).Problem(t)
	if p["detail"] != "Không tìm thấy yêu cầu phê duyệt." {
		t.Fatalf("not found: %v", p)
	}

	// Marcus approves the demo's request; Daniel then edits the subject, in
	// English: the approval is invalidated.
	nov := campaignID(t, daniel, "November Product Update")
	r := getReview(t, daniel, nov)
	marcus.Do("POST", ws+"/approvals/"+r.Approval.Request.ID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 200)
	var full map[string]any
	daniel.Do("GET", ws+"/campaigns/"+nov, nil).MustStatus(t, 200).JSON(t, &full)
	daniel.Do("PUT", ws+"/campaigns/"+nov+"/content", map[string]any{"baseVersion": full["version"], "subject": "November Product Update: faster exports and a new API",
		"preheader": full["preheader"], "mjml": full["mjml"], "html": full["html"], "label": "subject", "source": "manual",
		// The composer asked first: the editor agreed that saving cancels the approval.
		"cancelApproval": true}).MustStatus(t, 200)
	if r = getReview(t, daniel, nov); r.Approval.Request.Status != "invalidated" || r.Approval.Request.InvalidatedReason != "subject changed" {
		t.Fatalf("invalidated, in English: %+v", r.Approval.Request)
	}
	var vr struct {
		Approval struct {
			Request struct {
				InvalidatedReason, InvalidatedFields string
				Changes                              []struct{ Field, Label string }
			}
		}
	}
	viDaniel.Do("GET", ws+"/campaigns/"+nov+"/review", nil).MustStatus(t, 200).JSON(t, &vr)
	if q := vr.Approval.Request; q.InvalidatedReason != "đã thay đổi tiêu đề" || q.InvalidatedFields != "tiêu đề" || len(q.Changes) == 0 || q.Changes[0].Label != "Tiêu đề" {
		t.Fatalf("invalidated, in Vietnamese: %+v", q)
	}

	// The inbox says "after approval" once; the demo's older invalidated
	// request (its reason kept in English only) reads in English.
	notes := inboxNotes(t, viMarcus, "invalidated")
	if !slices.Contains(notes, "đã thay đổi tiêu đề sau khi được phê duyệt") || !slices.Contains(notes, "audience moved +14% after approval") {
		t.Fatalf("inbox notes: %q", notes)
	}
	if en := inboxNotes(t, marcus, "invalidated"); !slices.Contains(en, "subject changed after approval") || !slices.Contains(en, "audience moved +14% after approval") {
		t.Fatalf("inbox notes, English: %q", en)
	}

	// Daniel asks again (in English); Marcus reads the request's rules in Vietnamese.
	var asked struct{ ID string }
	// The checks run again first (the campaign changed since they ran).
	daniel.Do("POST", ws+"/campaigns/"+nov+"/approval", map[string]any{}).MustStatus(t, 409)
	readinesstest.Work(t, env)
	daniel.Do("POST", ws+"/campaigns/"+nov+"/approval", map[string]any{}).MustStatus(t, 201).JSON(t, &asked)
	var d struct {
		Rules             []struct{ Key, Label, Short, Badge string }
		InvalidatedReason string
	}
	viMarcus.Do("GET", ws+"/approvals/"+asked.ID, nil).MustStatus(t, 200).JSON(t, &d)
	badges := []string{}
	for _, rule := range d.Rules {
		badges = append(badges, rule.Badge)
		if rule.Key == "author_lacks_send" && !strings.HasPrefix(rule.Label, "Daniel Park chưa được gửi lần này") {
			t.Errorf("lacks send: %q", rule.Label)
		}
	}
	if !slices.Contains(badges, "người nhận ≥ 10.000") {
		t.Errorf("badges: %q", badges)
	}
	var list struct{ Items []struct{ ID, Rules string } }
	viMarcus.Do("GET", ws+"/approvals?status=pending", nil).MustStatus(t, 200).JSON(t, &list)
	for _, it := range list.Items {
		if it.ID == asked.ID && !strings.HasPrefix(it.Rules, "quy tắc: người nhận ≥ 10.000") {
			t.Errorf("inbox rules: %q", it.Rules)
		}
	}
}

func inboxNotes(t *testing.T, c *apptest.Client, status string) []string {
	t.Helper()
	var list struct{ Items []struct{ Note string } }
	c.Do("GET", ws+"/approvals?status="+status+"&days=0", nil).MustStatus(t, 200).JSON(t, &list)
	out := []string{}
	for _, it := range list.Items {
		out = append(out, it.Note)
	}
	return out
}
