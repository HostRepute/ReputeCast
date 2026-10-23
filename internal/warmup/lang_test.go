package warmup_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

type warmupLog struct {
	Plans []struct {
		ID, ProviderID, StatusDetail string
		Log                          []struct{ Kind, Title, Body string }
	}
}

// The Warm-up tab reads in the request's language: its problems, and what a
// plan stored for later readers. A log entry written while a member read
// English keeps its English in the row and reads in Vietnamese for a
// Vietnamese reader; entries written before (the demo's) stay English.
func TestWarmupInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	var o warmupLog
	en.Do("GET", ws+"/warmup", nil).MustStatus(t, 200).JSON(t, &o)
	plan := o.Plans[0]
	path := ws + "/warmup/" + plan.ID

	// Postal already has a plan.
	body := map[string]any{"providerId": plan.ProviderID, "days": 21, "engagedFirst": true, "guardBounce": 2, "guardComplaint": 0.1, "guardAction": "hold", "spreadMailbox": true}
	if pr := vi.Do("POST", ws+"/warmup", body).MustStatus(t, 409).Problem(t); pr["detail"] != "Postal đã có kế hoạch khởi động. Hãy đổi cài đặt của kế hoạch đó." {
		t.Fatalf("Vietnamese problem: %v", pr)
	}
	if pr := en.Do("POST", ws+"/warmup", body).MustStatus(t, 409).Problem(t); pr["detail"] != "Postal already has a warm-up plan. Change its settings instead." {
		t.Fatalf("English problem: %v", pr)
	}

	// Paused while reading English.
	en.Do("POST", path+"/pause", nil).MustStatus(t, 200)
	var stored, detail string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT e.body, p.status_detail FROM warmup_events e JOIN warmup_plans p ON p.id = e.plan_id
		WHERE e.plan_id = $1 AND e.kind = 'paused'`, plan.ID).Scan(&stored, &detail); err != nil {
		t.Fatal(err)
	}
	if stored != "Warm-up paused by Linh Nguyen. No mail goes to Postal until it resumes." || detail != "Paused by Linh Nguyen" {
		t.Fatalf("stored: %q, %q", stored, detail)
	}
	en.Do("GET", ws+"/warmup", nil).MustStatus(t, 200).JSON(t, &o)
	if p := o.Plans[0]; p.StatusDetail != "Paused by Linh Nguyen" || p.Log[0].Body != stored {
		t.Fatalf("English: %q, %+v", p.StatusDetail, p.Log[0])
	}
	vi.Do("GET", ws+"/warmup", nil).MustStatus(t, 200).JSON(t, &o)
	p := o.Plans[0]
	if p.StatusDetail != "Linh Nguyen đã tạm dừng" || p.Log[0].Body != "Linh Nguyen đã tạm dừng khởi động. Không có thư nào gửi qua Postal cho đến khi tiếp tục." {
		t.Fatalf("Vietnamese: %q, %+v", p.StatusDetail, p.Log[0])
	}
	// The demo's entries were written before: they read in English.
	last := p.Log[len(p.Log)-1]
	if last.Kind != "started" || last.Body == "" || last.Body[0] != 'P' {
		t.Fatalf("an entry from before: %+v", last)
	}
	vi.Do("POST", path+"/resume", nil).MustStatus(t, 200)
	vi.Do("GET", ws+"/warmup", nil).MustStatus(t, 200).JSON(t, &o)
	if p := o.Plans[0]; p.StatusDetail != "" || p.Log[0].Body != "Linh Nguyen đã tiếp tục khởi động." {
		t.Fatalf("resumed: %q, %+v", p.StatusDetail, p.Log[0])
	}
}
