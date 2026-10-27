package readiness_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/readiness/readinesstest"
)

// SR-10 (C10): Settings › Sending rules over the API. An Owner adds,
// changes and removes rules and the audience defaults (each audited); an
// Editor reads them and is told who can change them, never a permission
// key. A run records the send that broke a rule, the rule's fix tags the
// links, and the next run finds the rule kept.
func TestSendingRules(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	owner, editor := env.Client(), env.Client()
	owner.SignInDemo("linh@acme.io")
	editor.SignInDemo("sofia@acme.io")
	audits := func(action string) int {
		t.Helper()
		var n int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = $1`, action).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	list := func(c *apptest.Client) readiness.SendingRulesView {
		t.Helper()
		var v readiness.SendingRulesView
		c.Do(http.MethodGet, ws+"/sending-rules", nil).MustStatus(t, http.StatusOK).JSON(t, &v)
		return v
	}

	// The sample's four rules, read by an Editor.
	v := list(editor)
	if len(v.Rules) != 4 || v.CanManage || !v.Defaults.ExcludeUnengaged || v.Defaults.TypedMultiple != 1.5 || v.SunsetDays != 180 {
		t.Fatalf("the sample: %+v", v)
	}
	if strings.Contains(v.Reason, "workspace:manage") || !strings.Contains(v.Reason, "Linh Nguyen") {
		t.Errorf("why an Editor can't change them: %q", v.Reason)
	}
	if v.Rules[0].Description != "Every link has a utm_campaign query parameter" || v.Rules[0].Triggered == nil || v.Rules[0].Triggered.Campaign != "Weekly Digest #142" {
		t.Errorf("the UTM rule: %+v", v.Rules[0])
	}
	rule := map[string]any{"name": "Links need utm_term", "kind": "link_param", "operator": "on_every", "value": "utm_term", "severity": "warn",
		"appliesTo": "broadcast", "enabled": true, "message": "Our analytics read utm_term."}
	editor.Do(http.MethodPost, ws+"/sending-rules", rule).MustStatus(t, http.StatusForbidden)

	// Checked before it's kept.
	bad := map[string]any{"name": "Spaces", "kind": "link_param", "operator": "on_every", "value": "utm term", "severity": "warn", "appliesTo": "broadcast", "enabled": true}
	owner.Do(http.MethodPost, ws+"/sending-rules", bad).MustStatus(t, http.StatusUnprocessableEntity)
	bad = map[string]any{"name": "Subject", "kind": "subject_length", "operator": "on_every", "value": "70", "severity": "warn", "appliesTo": "broadcast", "enabled": true}
	owner.Do(http.MethodPost, ws+"/sending-rules", bad).MustStatus(t, http.StatusUnprocessableEntity)

	var created readiness.SendingRule
	owner.Do(http.MethodPost, ws+"/sending-rules", rule).MustStatus(t, http.StatusCreated).JSON(t, &created)
	if created.ID == "" || created.AddedBy != "Linh Nguyen" || created.Description != "Every link has a utm_term query parameter" || audits("sending_rule.created") != 5 {
		t.Fatalf("created: %+v, %d audited", created, audits("sending_rule.created"))
	}

	// A run of the digest breaks it, and records that it did.
	id := campaignID(t, owner, "Weekly Digest #142")
	run := runChecks(t, env, owner, id)
	ch, ok := checkOf(run, "rules."+created.ID)
	if !ok || ch.Status != readiness.Warn || ch.Group != readiness.GroupRules || !strings.HasPrefix(ch.Title, "Workspace rule: links need utm_term · ") || len(ch.Fixes) != 1 ||
		ch.Fixes[0].Action != readiness.FixRuleAddParam || ch.Fixes[0].Params["value"] != "weekly-digest-142" || len(ch.Raw) == 0 {
		t.Fatalf("the run: %+v", ch)
	}
	var found *readiness.SendingRule
	for _, r := range list(owner).Rules {
		if r.ID == created.ID {
			found = &r
		}
	}
	if found == nil || found.Triggered == nil || found.Triggered.Campaign != "Weekly Digest #142" || found.Triggered.Count != len(ch.Raw) || found.Triggered.Fixed {
		t.Fatalf("triggered: %+v", found)
	}

	// Its fix tags the links (a new version), and the next run finds it kept.
	fx := ch.Fixes[0]
	readinesstest.Fix(t, env, owner, ws+"/campaigns/"+id+"/readiness/fixes", map[string]any{"fixes": []map[string]any{{"action": fx.Action, "params": fx.Params}}}).
		MustStatus(t, http.StatusOK)
	after := readinesstest.Latest(t, owner, "acme-newsletter", id)
	if ch, _ := checkOf(after, "rules."+created.ID); ch.Status != readiness.Pass {
		t.Fatalf("after the fix: %+v", ch)
	}
	for _, r := range list(owner).Rules {
		if r.ID == created.ID && (r.Triggered == nil || !r.Triggered.Fixed) {
			t.Errorf("kept again: %+v", r.Triggered)
		}
	}

	// Turned off, then removed.
	rule["enabled"] = false
	var off readiness.SendingRule
	owner.Do(http.MethodPut, ws+"/sending-rules/"+created.ID, rule).MustStatus(t, http.StatusOK).JSON(t, &off)
	if off.Enabled || audits("sending_rule.updated") != 1 {
		t.Fatalf("off: %+v", off)
	}
	editor.Do(http.MethodDelete, ws+"/sending-rules/"+created.ID, nil).MustStatus(t, http.StatusForbidden)
	owner.Do(http.MethodDelete, ws+"/sending-rules/"+created.ID, nil).MustStatus(t, http.StatusNoContent)
	owner.Do(http.MethodDelete, ws+"/sending-rules/"+created.ID, nil).MustStatus(t, http.StatusNotFound)
	if len(list(owner).Rules) != 4 || audits("sending_rule.deleted") != 1 {
		t.Fatal("not removed")
	}

	// The audience defaults: the unengaged go in every send.
	owner.Do(http.MethodPut, ws+"/send-defaults", map[string]any{"excludeUnengaged": true, "typedMultiple": 25}).MustStatus(t, http.StatusUnprocessableEntity)
	editor.Do(http.MethodPut, ws+"/send-defaults", map[string]any{"excludeUnengaged": false, "typedMultiple": 2}).MustStatus(t, http.StatusForbidden)
	var saved readiness.SendingRulesView
	owner.Do(http.MethodPut, ws+"/send-defaults", map[string]any{"excludeUnengaged": false, "typedMultiple": 2}).MustStatus(t, http.StatusOK).JSON(t, &saved)
	if saved.Defaults.ExcludeUnengaged || saved.Defaults.TypedMultiple != 2 || audits("send_defaults.updated") != 1 {
		t.Fatalf("defaults: %+v", saved.Defaults)
	}
	in := runChecks(t, env, owner, id)
	if a := in.Estimates.Audience; a.UnengagedExcluded || a.Final != run.Estimates.Audience.Final+run.Estimates.Audience.ExcludedUnengaged {
		t.Fatalf("the unengaged go in: %+v (before %+v)", a, run.Estimates.Audience)
	}
}
