package workspaces_test

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// DP-4, DP-6: Settings › General › Data retention starts at 30 days for
// unconfirmed signups and 12 months for open and click details; only the
// offered periods are taken, only by people who manage the workspace, and
// every change is audited.
func TestRetentionSettings(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	const ws = "/workspaces/acme-newsletter"
	var got struct {
		Retention struct{ UnconfirmedEraseDays, EventDetailDays int }
	}
	c.Do("GET", ws+"/settings", nil).MustStatus(t, 200).JSON(t, &got)
	if got.Retention.UnconfirmedEraseDays != 30 || got.Retention.EventDetailDays != 365 {
		t.Fatalf("defaults: %+v", got.Retention)
	}
	c.Do("PUT", ws+"/settings/retention", map[string]any{"unconfirmedEraseDays": 45, "eventDetailDays": 365}).MustStatus(t, 422)
	c.Do("PUT", ws+"/settings/retention", map[string]any{"unconfirmedEraseDays": 0, "eventDetailDays": 90}).MustStatus(t, 204)
	c.Do("GET", ws+"/settings", nil).MustStatus(t, 200).JSON(t, &got)
	if got.Retention.UnconfirmedEraseDays != 0 || got.Retention.EventDetailDays != 90 {
		t.Fatalf("saved: %+v", got.Retention)
	}
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = 'workspace.updated' AND after::text LIKE '%retention%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("%d audit entries (%v)", n, err)
	}
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("PUT", ws+"/settings/retention", map[string]any{"unconfirmedEraseDays": 30, "eventDetailDays": 365}).MustStatus(t, 403)
}
