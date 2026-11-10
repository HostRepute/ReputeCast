package automations_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/store"
)

// G-11: the paid-member triggers start on Earn's events, matching the
// tier, a tier change's direction and why access ended.
func TestPaidTriggers(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	svc := engine(env, &clock{t: time.Now()})
	var ws, founding, insider uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, f.id, i.id FROM workspaces w
		JOIN earn_tiers f ON f.workspace_id = w.id AND f.name = 'Founding member'
		JOIN earn_tiers i ON i.workspace_id = w.id AND i.name = 'Weekly Digest Insider'
		WHERE w.slug = 'acme-newsletter'`).Scan(&ws, &founding, &insider); err != nil {
		t.Fatal(err)
	}

	// The editor offers the tiers, and the demo's live Stripe connection.
	var opts struct {
		Tiers         []struct{ ID, Name string }
		PaidConnected bool `json:"paidConnected"`
	}
	c.Do("GET", wsPath+"/options", nil).MustStatus(t, 200).JSON(t, &opts)
	if !opts.PaidConnected || len(opts.Tiers) != 2 {
		t.Fatalf("options: %+v", opts)
	}

	upgraded := publishFlow(t, c, "Welcome, founding members", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerPaidTierChanged, TierID: founding.String(), Direction: "upgrade", Reentry: "always"})),
			node("tag", automations.NodeList, tagStep("founding-upgrade")),
		},
		Edges: []automations.AutomationEdge{link("t", "tag", "")},
	})
	failed := publishFlow(t, c, "Win back after a failed payment", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerPaidEnded, EndReasons: []string{"payment_failed"}, Reentry: "always"})),
			node("tag", automations.NodeList, tagStep("win-back")),
		},
		Edges: []automations.AutomationEdge{link("t", "tag", "")},
	})
	up := addContact(t, env, "upgrader@example.com", nil)
	down := addContact(t, env, "downgrader@example.com", nil)
	emit := func(kind string, contact uuid.UUID, payload map[string]any) {
		t.Helper()
		b, _ := json.Marshal(payload)
		if err := env.App.Q.InsertAutomationEvent(ctx, store.InsertAutomationEventParams{WorkspaceID: ws, Kind: kind, ContactID: contact,
			Ref: "sub:" + uuid.Must(uuid.NewV7()).String(), Payload: b, OccurredAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	emit(automations.TriggerPaidTierChanged, up, map[string]any{"tier_id": founding.String(), "direction": "upgrade", "from_tier_id": insider.String()})
	emit(automations.TriggerPaidTierChanged, down, map[string]any{"tier_id": founding.String(), "direction": "downgrade"})
	emit(automations.TriggerPaidTierChanged, down, map[string]any{"tier_id": insider.String(), "direction": "upgrade"})
	emit(automations.TriggerPaidEnded, up, map[string]any{"tier_id": founding.String(), "reason": "canceled"})
	emit(automations.TriggerPaidEnded, down, map[string]any{"tier_id": insider.String(), "reason": "payment_failed"})
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	runs := func(flow string, contact uuid.UUID) int64 {
		return countWhere(t, env, `SELECT count(*) FROM automation_runs WHERE automation_id = $1 AND contact_id = $2`, flow, contact)
	}
	if runs(upgraded, up) != 1 || runs(upgraded, down) != 0 {
		t.Fatalf("Changes tier runs: upgrade %d, downgrade or another tier %d", runs(upgraded, up), runs(upgraded, down))
	}
	if runs(failed, up) != 0 || runs(failed, down) != 1 {
		t.Fatalf("Stops paying runs: canceled %d, payment failed %d", runs(failed, up), runs(failed, down))
	}

	// A tier that isn't the workspace's can't be published.
	var created struct{ ID string }
	c.Do("POST", wsPath, map[string]any{"name": "Unknown tier"}).MustStatus(t, 201).JSON(t, &created)
	var saved struct {
		DraftVersion int
		Issues       []struct{ NodeID, Code, Severity string }
	}
	c.Do("PUT", wsPath+"/"+created.ID+"/draft", map[string]any{"graph": automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerPaidStarted, TierID: uuid.Must(uuid.NewV7()).String()})),
			node("tag", automations.NodeList, tagStep("new-member")),
		},
		Edges: []automations.AutomationEdge{link("t", "tag", "")},
	}, "revision": 0}).MustStatus(t, 200).JSON(t, &saved)
	if len(saved.Issues) != 1 || saved.Issues[0].NodeID != "t" || saved.Issues[0].Code != "missing_ref" || saved.Issues[0].Severity != "error" {
		t.Fatalf("issues: %+v", saved.Issues)
	}
	c.Do("POST", wsPath+"/"+created.ID+"/publish", map[string]any{"version": saved.DraftVersion, "migration": "finish"}).MustStatus(t, 422)
}
