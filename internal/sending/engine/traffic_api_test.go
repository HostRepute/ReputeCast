package engine_test

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// testTrafficRulesAPI: the Traffic rules screen reads a workspace's rules
// for people who can view providers and saves them for people who can
// manage them (review GS-11, GS-12; C9): a new reply rule, thresholds by
// bounce type, checked and audited.
func testTrafficRulesAPI(t *testing.T, f *sendtest.Fixture) {
	path := "/workspaces/acme-newsletter/traffic-rules"
	viewer := f.Env.Client()
	viewer.SignInDemo("aisha@acme.io")
	var got map[string]any
	viewer.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &got)
	th, _ := got["thresholds"].(map[string]any)
	if th["blockBounce"] != 0.5 || th["deferralSlowWindow"] != 10.0 || th["complaintAction"] != "pause_mailbox" {
		t.Fatalf("thresholds %v", th)
	}
	// What the rules did, by mailbox provider name and with percentages
	// (the demo's October Product Update auto-pause).
	recent, _ := got["recent"].([]any)
	found := false
	for _, r := range recent {
		if r.(map[string]any)["text"] == "Outlook complaints reached 0.18% (limit 0.10%) · paused Outlook" {
			found = true
		}
	}
	if !found {
		t.Fatalf("recent actions %v, want Outlook's auto-pause in words", recent)
	}
	body := map[string]any{
		"thresholds": map[string]any{"blockBounce": 0.4, "blockAction": "pause_mailbox", "unknownWarn": 3, "unknownWarnAction": "warn", "unknownPause": 5,
			"unknownPauseAction": "pause_send", "deferralSlow": 15, "deferralSlowWindow": 10, "deferralSlowAction": "slow", "deferral": 40, "deferralWindow": 10,
			"deferralAction": "pause_mailbox", "complaint": 0.1, "complaintAction": "pause_mailbox"},
		"replyRules": []any{map[string]any{"mailbox": "outlook", "contains": "S3150", "count": 1, "windowMinutes": 10, "action": "pause_route", "durationMinutes": 0, "enabled": true}},
		"canary":     got["canary"],
		"watch":      got["watch"],
	}
	viewer.Do("PUT", path, body).MustStatus(t, 403)
	owner := f.Env.Client()
	owner.SignInDemo("linh@acme.io")
	bad := map[string]any{"thresholds": body["thresholds"], "replyRules": []any{map[string]any{"mailbox": "outlook", "contains": " ", "count": 1, "windowMinutes": 10,
		"action": "pause", "durationMinutes": 0, "enabled": true}}, "canary": got["canary"], "watch": got["watch"]}
	owner.Do("PUT", path, bad).MustStatus(t, 422)
	var saved map[string]any
	owner.Do("PUT", path, body).MustStatus(t, 200).JSON(t, &saved)
	rules, _ := saved["replyRules"].([]any)
	if len(rules) != 1 {
		t.Fatalf("reply rules %v", saved["replyRules"])
	}
	r := rules[0].(map[string]any)
	if r["contains"] != "S3150" || r["action"] != "pause_route" || r["means"] == "" || r["id"] == "" {
		t.Fatalf("rule %v", r)
	}
	if th, _ := saved["thresholds"].(map[string]any); th["blockBounce"] != 0.4 {
		t.Fatalf("thresholds %v", th)
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'traffic_rules.updated'`, f.WS); n != 1 {
		t.Fatalf("%d audit entries, want 1", n)
	}
	// Saving it again keeps the rule (by its id) instead of adding one.
	body["replyRules"] = rules
	owner.Do("PUT", path, body).MustStatus(t, 200).JSON(t, &saved)
	if again, _ := saved["replyRules"].([]any); len(again) != 1 || again[0].(map[string]any)["id"] != r["id"] {
		t.Fatalf("reply rules after saving again %v", saved["replyRules"])
	}
	f.Exec(`DELETE FROM traffic_reply_rules WHERE workspace_id = $1`, f.WS)
	f.Exec(`UPDATE workspaces SET send_thresholds = send_thresholds - 'block_bounce' WHERE id = $1`, f.WS)
}
