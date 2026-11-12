package forms_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/tracker"
)

// Form targeting by segment (M62 G-32): a popup or embedded form shows to
// (or hides from) the contacts of a segment, known through Website
// tracking on the same site. The session reads the tracker's id of the
// browser (rcv), finds its contact and checks the segment's materialised
// members; a visitor nobody knows is in no segment. Live forms' dynamic
// segments are watched, as flows' are.
func TestFormTargetingBySegment(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := signedIn(env, "linh@acme.io")
	signup := formByName(t, overview(t, c), "Newsletter signup")
	f := getForm(t, c, signup)
	in := inputOf(f)
	content := in["content"].(map[string]any)

	var ws, seg, inside, outside uuid.UUID
	q := func(sql string, args []any, dest ...any) {
		t.Helper()
		if err := env.App.Pool.QueryRow(ctx, sql, args...).Scan(dest...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	q(`SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`, nil, &ws)
	q(`SELECT id FROM segments WHERE workspace_id = $1 AND refresh <> 'static' ORDER BY created_at, id LIMIT 1`, []any{ws}, &seg)
	q(`SELECT id FROM contacts WHERE workspace_id = $1 AND status = 'active' AND erased_at IS NULL ORDER BY id LIMIT 1`, []any{ws}, &inside)
	q(`SELECT id FROM contacts WHERE workspace_id = $1 AND status = 'active' AND erased_at IS NULL AND id <> $2 ORDER BY id LIMIT 1`, []any{ws, inside}, &outside)
	exec(`INSERT INTO segment_members (segment_id, contact_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, seg, inside)
	exec(`DELETE FROM segment_members WHERE segment_id = $1 AND contact_id = $2`, seg, outside)
	// Two browsers the tracker linked to contacts, and one it doesn't know.
	const knownIn, knownOut, stranger = "KKKKKKKKKKKKKKKKKKKKKK", "OOOOOOOOOOOOOOOOOOOOOO", "UUUUUUUUUUUUUUUUUUUUUU"
	for vid, contact := range map[string]uuid.UUID{knownIn: inside, knownOut: outside} {
		h, ok := tracker.VisitorHash(ws, vid)
		if !ok {
			t.Fatalf("%s isn't a tracker id", vid)
		}
		exec(`INSERT INTO tracker_visitors (id, workspace_id, vid_hash, contact_id, identified_by, first_seen, last_seen, identified_at) VALUES ($1, $2, $3, $4, 'click', now(), now(), now())`,
			uuid.Must(uuid.NewV7()), ws, h, contact)
	}

	target := func(visitors, segment string) map[string]any {
		return map[string]any{"pages": map[string]any{"mode": "all", "rules": []any{}}, "sources": map[string]any{"utmSource": []string{}, "utmMedium": []string{}, "utmCampaign": []string{}, "referrer": "any"},
			"devices": []string{}, "countries": map[string]any{"mode": "all", "codes": []string{}}, "visitors": visitors, "segmentId": segment, "schedule": map[string]any{}}
	}
	// A segment rule needs one of the workspace's segments.
	content["targeting"] = target("in_segment", "")
	if r := c.Do("PUT", api+"/forms/"+signup, in); r.Status != 422 || !strings.Contains(string(r.Body), "Pick the segment") {
		t.Fatalf("no segment: %d %s", r.Status, r.Body)
	}
	content["targeting"] = target("in_segment", uuid.NewString())
	if r := c.Do("PUT", api+"/forms/"+signup, in); r.Status != 422 || !strings.Contains(string(r.Body), "no longer exists") {
		t.Fatalf("another workspace's segment: %d %s", r.Status, r.Body)
	}

	session := "/p/acme-newsletter/forms/frm_8c2e1a/session"
	shows := func(rcv string) bool {
		t.Helper()
		path := session
		if rcv != "" {
			path += "?rcv=" + rcv
		}
		m := pubJSON(t, pub(t, env, "GET", path, "", "", "https://acme.io"))
		return m["show"] == true
	}

	// In the segment: only a known contact who is in it.
	content["targeting"] = target("in_segment", seg.String())
	c.Do("PUT", api+"/forms/"+signup, in).MustStatus(t, 200)
	if g, _ := getForm(t, c, signup).Content["targeting"].(map[string]any); g["visitors"] != "in_segment" || g["segmentId"] != seg.String() {
		t.Fatalf("saved targeting: %+v", g)
	}
	for _, tc := range []struct {
		rcv  string
		want bool
	}{{knownIn, true}, {knownOut, false}, {stranger, false}, {"", false}, {"not-a-tracker-id", false}} {
		if got := shows(tc.rcv); got != tc.want {
			t.Errorf("in_segment, visitor %q: show %v, want %v", tc.rcv, got, tc.want)
		}
	}
	// The script waits for the session, and carries the rule it can't check.
	js := pub(t, env, "GET", "/p/acme-newsletter/forms/frm_8c2e1a.js", "", "", "")
	if !strings.Contains(js.Body, `"visitors":"in_segment"`) || !strings.Contains(js.Body, `"wait":true`) || strings.Contains(js.Body, seg.String()) {
		t.Errorf("script: %s", js.Body[:min(len(js.Body), 600)])
	}

	// Not in the segment: everyone else, visitors nobody knows included.
	content["targeting"] = target("not_in_segment", seg.String())
	c.Do("PUT", api+"/forms/"+signup, in).MustStatus(t, 200)
	for _, tc := range []struct {
		rcv  string
		want bool
	}{{knownIn, false}, {knownOut, true}, {stranger, true}, {"", true}} {
		if got := shows(tc.rcv); got != tc.want {
			t.Errorf("not_in_segment, visitor %q: show %v, want %v", tc.rcv, got, tc.want)
		}
	}

	// The live form's dynamic segment is watched, so its members stay
	// current; once the form is paused, only a flow would keep it watched.
	watched := func() bool {
		t.Helper()
		rows, err := env.App.Q.AutomationWatchedSegments(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return slices.ContainsFunc(rows, func(r store.AutomationWatchedSegmentsRow) bool { return r.ID == seg })
	}
	if !watched() {
		t.Error("the form's segment isn't watched")
	}
	var byFlow bool
	q(`SELECT EXISTS (SELECT 1 FROM automations a CROSS JOIN LATERAL jsonb_array_elements(a.triggers) t
		WHERE a.status = 'live' AND a.deleted_at IS NULL AND t.value ->> 'segmentId' = $1)`, []any{seg.String()}, &byFlow)
	exec(`UPDATE forms SET status = 'paused' WHERE id = $1`, uuid.MustParse(signup))
	if watched() != byFlow {
		t.Errorf("a paused form's segment: watched %v, a flow watches it %v", watched(), byFlow)
	}
}
