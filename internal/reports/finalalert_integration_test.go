package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/reports"
)

// TestFinalAlert: someone reading a report whose results are still coming
// in asks to be emailed when they are final ("Email me when it's final",
// ReportStates); once the 72 hours are over the job tells them once and
// forgets the request.
func TestFinalAlert(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	id := campaignID(t, c, "Weekly Digest #141")
	path := "/workspaces/acme-newsletter/campaigns/" + id + "/report/final-alert"
	audits := func(action string) int {
		t.Helper()
		var n int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = $1 AND target_id = $2`, action, id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// Results that are already final aren't waited for.
	c.Do("PUT", path, nil).MustStatus(t, 409)

	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET started_at = now() - interval '3 hours', completed_at = now() - interval '2 hours' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	var a struct {
		On      bool
		FinalAt *time.Time
	}
	c.Do("PUT", path, nil).MustStatus(t, 200).JSON(t, &a)
	if !a.On || a.FinalAt == nil || time.Until(*a.FinalAt) < 69*time.Hour {
		t.Fatalf("asked: %+v", a)
	}
	c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &a)
	if !a.On || audits("report_final_alert.set") != 1 {
		t.Fatalf("read back: %+v, %d audit entries", a, audits("report_final_alert.set"))
	}
	// Asking twice keeps one request.
	c.Do("PUT", path, nil).MustStatus(t, 200)
	// A Viewer reads reports, so they can ask too.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("PUT", path, nil).MustStatus(t, 200)

	svc := &reports.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}
	if n, err := svc.SendFinalAlerts(ctx); err != nil || n != 0 {
		t.Fatalf("told before the results are final: %d %v", n, err)
	}
	svc.Now = func() time.Time { return time.Now().Add(71 * time.Hour) }
	if n, err := svc.SendFinalAlerts(ctx); err != nil || n != 2 {
		t.Fatalf("told %d people: %v", n, err)
	}
	var title, link string
	if err := env.App.Pool.QueryRow(ctx, `SELECT n.title, n.link FROM notifications n JOIN users u ON u.id = n.user_id
		WHERE u.email = 'linh@acme.io' AND n.kind = 'results_final'`).Scan(&title, &link); err != nil {
		t.Fatal(err)
	}
	if title != "Results are final: Weekly Digest #141" || link != "/w/acme-newsletter/campaigns/"+id+"/report" {
		t.Fatalf("notification: %q %q", title, link)
	}
	// Told once.
	if n, err := svc.SendFinalAlerts(ctx); err != nil || n != 0 {
		t.Fatalf("told again: %d %v", n, err)
	}

	// Stopping a request: gone, and audited once.
	c.Do("PUT", path, nil).MustStatus(t, 200)
	c.Do("DELETE", path, nil).MustStatus(t, 204)
	c.Do("DELETE", path, nil).MustStatus(t, 204)
	c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &a)
	if a.On || audits("report_final_alert.remove") != 1 {
		t.Fatalf("stopped: %+v, %d audit entries", a, audits("report_final_alert.remove"))
	}
}
