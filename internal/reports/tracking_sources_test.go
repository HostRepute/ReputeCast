package reports_test

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// TestTrackingSources: a sending provider whose own tracking is kept, and
// that reported the send's opens, is named as where its opens and clicks
// came from; otherwise they are ReputeCast's own (ReportStates' tracking
// source notes, review AN-4).
func TestTrackingSources(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	id := campaignID(t, c, "October Product Update")
	type column struct {
		Short, TrackedBy, Events string
		DeliveryInferred         bool
	}
	get := func() map[string]column {
		t.Helper()
		var r struct {
			Providers struct{ Columns []column }
		}
		c.Do("GET", "/workspaces/acme-newsletter/campaigns/"+id+"/report", nil).MustStatus(t, 200).JSON(t, &r)
		out := map[string]column{}
		for _, col := range r.Providers.Columns {
			out[col.Short] = col
		}
		return out
	}
	cols := get()
	if cols["SES"].TrackedBy != "reputecast" || cols["Postal"].TrackedBy != "reputecast" || cols["SES"].Events != "sns" || cols["SES"].DeliveryInferred {
		t.Fatalf("the demo's own tracking: %+v", cols)
	}
	// Postal keeps its own tracking now, but it reported nothing of this send.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE providers SET tracking = jsonb_set(coalesce(tracking, '{}'::jsonb), '{mode}', '"provider"')
		WHERE adapter = 'postal' AND workspace_id = (SELECT workspace_id FROM campaigns WHERE id = $1)`, id); err != nil {
		t.Fatal(err)
	}
	if cols = get(); cols["Postal"].TrackedBy != "reputecast" {
		t.Fatalf("a provider that reported nothing counted the opens: %+v", cols["Postal"])
	}
	// Once Postal reported an open of the send, its tracking counted them.
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO delivery_events (id, workspace_id, campaign_id, provider_id, type, occurred_at, original_type)
		SELECT gen_random_uuid(), c.workspace_id, c.id, p.id, 'open', now(), 'Open'
		FROM campaigns c JOIN providers p ON p.workspace_id = c.workspace_id AND p.adapter = 'postal' WHERE c.id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if cols = get(); cols["Postal"].TrackedBy != "provider" || cols["SES"].TrackedBy != "reputecast" {
		t.Fatalf("Postal's tracking: %+v", cols)
	}
}
