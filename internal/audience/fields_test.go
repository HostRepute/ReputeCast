package audience_test

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type use struct{ Kind, ID, Name string }

// Review 2026-11 AU-11: deleting a field first shows what uses it; the
// field waits 30 days with its values, can be restored as it was, keeps
// its key meanwhile, and its values go afterwards. A segment shows what
// uses it too.
func TestFieldsAndSegmentsShowWhatUsesThem(t *testing.T) {
	env, c, base := pagingWorkspace(t)
	ctx := context.Background()
	ws := workspaceID(t, env)
	var plan, size struct{ ID, Key string }
	c.Do(http.MethodPost, base+"/fields", map[string]any{"key": "plan", "label": "Plan", "type": "text"}).MustStatus(t, 201).JSON(t, &plan)
	c.Do(http.MethodPost, base+"/fields", map[string]any{"key": "size", "label": "Company size", "type": "number"}).MustStatus(t, 201).JSON(t, &size)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET attributes = '{"plan": "team"}' WHERE email LIKE 'reader1%'`); err != nil {
		t.Fatal(err)
	}
	// A segment on the field, a view showing it, a draft writing it in Liquid.
	var seg struct{ ID string }
	c.Do(http.MethodPost, base+"/segments", map[string]any{"name": "Team plan", "mode": "builder", "refresh": "dynamic",
		"definition": map[string]any{"groups": []any{map[string]any{"match": "all", "conditions": []any{map[string]any{"field": "custom:plan", "op": "equals", "value": "team"}}}}}}).
		MustStatus(t, 201).JSON(t, &seg)
	var view struct{ ID string }
	c.Do(http.MethodPost, base+"/contacts/views", map[string]any{"name": "Team accounts", "filters": map[string]any{"segment": seg.ID}, "columns": []string{"field:plan"}}).
		MustStatus(t, 201).JSON(t, &view)
	var draft struct{ ID string }
	c.Do(http.MethodPost, base+"/campaigns", map[string]any{"name": "Plan news"}).MustStatus(t, 201).JSON(t, &draft)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET html = '<p>Your {{ contact.plan }} plan</p>' WHERE id = $1`, draft.ID); err != nil {
		t.Fatal(err)
	}

	var overview struct {
		Usage []struct {
			ID     string
			Filled *int64
			Uses   []use
		}
		Deleted []struct {
			ID, Key      string
			RestoreUntil time.Time
		}
	}
	c.Do(http.MethodGet, base+"/fields/overview", nil).MustStatus(t, 200).JSON(t, &overview)
	kinds := func(uses []use) []string {
		out := []string{}
		for _, u := range uses {
			out = append(out, u.Kind)
		}
		slices.Sort(out)
		return out
	}
	for _, u := range overview.Usage {
		switch u.ID {
		case plan.ID:
			if u.Filled == nil || *u.Filled != 10 || !slices.Equal(kinds(u.Uses), []string{"campaign", "segment", "view"}) {
				t.Fatalf("plan: filled %v, uses %v", u.Filled, u.Uses)
			}
		case size.ID:
			if u.Filled == nil || *u.Filled != 0 || len(u.Uses) != 0 {
				t.Fatalf("size: filled %v, uses %v", u.Filled, u.Uses)
			}
		}
	}
	var impact struct{ Uses []use }
	c.Do(http.MethodGet, base+"/segments/"+seg.ID+"/impact", nil).MustStatus(t, 200).JSON(t, &impact)
	if !slices.Equal(kinds(impact.Uses), []string{"view"}) || impact.Uses[0].ID != view.ID {
		t.Fatalf("segment impact: %+v", impact.Uses)
	}

	// Deleted: gone from the fields, waiting with its values.
	c.Do(http.MethodGet, base+"/fields/"+plan.ID+"/impact", nil).MustStatus(t, 200).JSON(t, &impact)
	if len(impact.Uses) != 3 {
		t.Fatalf("field impact: %+v", impact.Uses)
	}
	c.Do(http.MethodDelete, base+"/fields/"+plan.ID, nil).MustStatus(t, 204)
	var fields struct{ Items []struct{ Key string } }
	c.Do(http.MethodGet, base+"/fields", nil).MustStatus(t, 200).JSON(t, &fields)
	if len(fields.Items) != 1 || fields.Items[0].Key != "size" {
		t.Fatalf("fields after delete: %+v", fields.Items)
	}
	c.Do(http.MethodGet, base+"/fields/overview", nil).MustStatus(t, 200).JSON(t, &overview)
	if len(overview.Deleted) != 1 || overview.Deleted[0].Key != "plan" || overview.Deleted[0].RestoreUntil.Before(time.Now().AddDate(0, 0, 29)) {
		t.Fatalf("deleted: %+v", overview.Deleted)
	}
	// Its key stays taken: a new "plan" would show the old values.
	c.Do(http.MethodPost, base+"/fields", map[string]any{"key": "plan", "label": "Plan", "type": "text"}).MustStatus(t, 409)
	c.Do(http.MethodPost, base+"/fields/"+plan.ID+"/restore", nil).MustStatus(t, 200)
	var kept int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM contacts WHERE workspace_id = $1 AND attributes ? 'plan'`, ws).Scan(&kept)
	if kept != 10 {
		t.Fatalf("values after restore: %d", kept)
	}

	// 31 days after a delete, the values go and so does the field.
	c.Do(http.MethodDelete, base+"/fields/"+plan.ID, nil).MustStatus(t, 204)
	svc := env.App.Audience()
	svc.Now = func() time.Time { return time.Now().AddDate(0, 0, 31) }
	if err := svc.PurgeDeletedFields(ctx); err != nil {
		t.Fatal(err)
	}
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM contacts WHERE workspace_id = $1 AND attributes ? 'plan'`, ws).Scan(&kept)
	var waiting int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM custom_fields_deleted WHERE id = $1`, uuid.MustParse(plan.ID)).Scan(&waiting)
	if kept != 0 || waiting != 0 {
		t.Fatalf("after 31 days: %d values, %d waiting", kept, waiting)
	}
	c.Do(http.MethodPost, base+"/fields/"+plan.ID+"/restore", nil).MustStatus(t, 404)
	var audits int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE target_id = $1 AND action IN ('field.delete', 'field.restore', 'field.values_removed')`, plan.ID).Scan(&audits)
	if audits != 4 {
		t.Fatalf("audited %d", audits)
	}
}

// Review 2026-11 AU-12: turning on "erase after N months in Sunset" shows
// a dry run first and takes a typed word; the policy can be switched off.
func TestSunsetErasingNeedsTheDryRun(t *testing.T) {
	env, c, base := pagingWorkspace(t)
	ctx := context.Background()
	if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET tier = 'sunset', sunset_at = now() - interval '14 months' WHERE email LIKE 'reader1%'`); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET tier = 'sunset', sunset_at = now() - interval '2 months' WHERE email LIKE 'reader2%'`); err != nil {
		t.Fatal(err)
	}
	var dry struct {
		Erase, Kept int64
		From        string
	}
	c.Do(http.MethodGet, base+"/audience/sunset-policy/dry-run?months=12", nil).MustStatus(t, 200).JSON(t, &dry)
	if dry.Erase != 10 || dry.Kept != 0 || dry.From == "" {
		t.Fatalf("dry run: %+v", dry)
	}
	policy := map[string]any{"enabled": true, "activeDays": 30, "atRiskDays": 91, "sunsetDays": 181, "countMachineOpens": false,
		"reengage": false, "stopBroadcasts": false, "stopAfterDays": 30, "deleteAfter": true, "deleteAfterMonths": 12}
	r := c.Do(http.MethodPut, base+"/audience/sunset-policy", policy).MustStatus(t, 409)
	if p := r.Problem(t); p["code"] != "confirm_erase" || p["meta"].(map[string]any)["erase"] != float64(10) {
		t.Fatalf("without the word: %v", p)
	}
	policy["confirmErase"] = "ERASE"
	var saved struct {
		DeleteAfter  bool
		ConfirmErase string
	}
	c.Do(http.MethodPut, base+"/audience/sunset-policy", policy).MustStatus(t, 200).JSON(t, &saved)
	if !saved.DeleteAfter || saved.ConfirmErase != "" {
		t.Fatalf("saved: %+v", saved)
	}
	// Already on: waiting longer needs no new word, erasing sooner does;
	// off is off.
	delete(policy, "confirmErase")
	policy["deleteAfterMonths"] = 18
	c.Do(http.MethodPut, base+"/audience/sunset-policy", policy).MustStatus(t, 200)
	policy["deleteAfterMonths"] = 1
	if p := c.Do(http.MethodPut, base+"/audience/sunset-policy", policy).MustStatus(t, 409).Problem(t); p["meta"].(map[string]any)["erase"] != float64(20) {
		t.Fatalf("sooner: %v", p)
	}
	policy["enabled"] = false
	var off struct{ Enabled bool }
	c.Do(http.MethodPut, base+"/audience/sunset-policy", policy).MustStatus(t, 200).JSON(t, &off)
	if off.Enabled {
		t.Fatal("the policy stayed on")
	}
}

// Review 2026-11 AU-14: a CSV upload is one group on the suppression list,
// and an accidental one comes off whole, with a reason, audited.
func TestRemoveASuppressionUpload(t *testing.T) {
	env, c, base := pagingWorkspace(t)
	ctx := context.Background()
	if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET status = 'suppressed' WHERE email = 'reader03@paging.example'`); err != nil {
		t.Fatal(err)
	}
	r := c.DoRaw(http.MethodPost, base+"/suppressions/upload?fileName=blocklist-oct.csv&reason=manual", "text/csv",
		strings.NewReader("email\nreader03@paging.example\nwrong1@example.com\nwrong2@example.com\n")).MustStatus(t, 200)
	var added struct{ Added int64 }
	r.JSON(t, &added)
	if added.Added != 3 {
		t.Fatalf("uploaded: %+v", added)
	}
	var list struct {
		Items []struct {
			ID, ImportID, Address, UploadedBy string
			Group, Upload                     bool
			Rows                              int64
		}
	}
	c.Do(http.MethodGet, base+"/suppressions", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || !list.Items[0].Group || !list.Items[0].Upload || list.Items[0].Rows != 3 || list.Items[0].Address != "blocklist-oct.csv" ||
		list.Items[0].UploadedBy == "" {
		t.Fatalf("the upload's row: %+v", list.Items)
	}
	upload := list.Items[0].ImportID
	// Opening the group lists its addresses.
	c.Do(http.MethodGet, base+"/suppressions?import="+upload, nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 3 {
		t.Fatalf("the group's rows: %d", len(list.Items))
	}
	c.Do(http.MethodPost, base+"/suppressions/uploads/"+upload+"/remove", map[string]any{"reason": ""}).MustStatus(t, 422)
	var res struct{ Removed int64 }
	c.Do(http.MethodPost, base+"/suppressions/uploads/"+upload+"/remove", map[string]any{"reason": "Wrong file uploaded"}).MustStatus(t, 200).JSON(t, &res)
	if res.Removed != 3 {
		t.Fatalf("removed %d", res.Removed)
	}
	var left, audits int
	var status string
	_ = env.App.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM suppressions WHERE workspace_id = c.workspace_id), c.status,
		(SELECT count(*) FROM audit_events WHERE action = 'suppression.upload_remove' AND reason = 'Wrong file uploaded')
		FROM contacts c WHERE c.email = 'reader03@paging.example'`).Scan(&left, &status, &audits)
	if left != 0 || status != "active" || audits != 1 {
		t.Fatalf("left %d, status %s, audits %d", left, status, audits)
	}
}

// Review 2026-11 AU-16: a static segment's refresh is a diff; members who
// stay keep when they entered, and each entry and leave is recorded.
func TestSegmentsKeepWhenMembersEnteredAndLeft(t *testing.T) {
	env, c, base := pagingWorkspace(t)
	ctx := context.Background()
	if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET tags = CASE
		WHEN email LIKE 'reader1%' AND email >= 'reader15' THEN '{a,b}'::text[]
		WHEN email LIKE 'reader1%' THEN '{a}'::text[]
		WHEN email LIKE 'reader2%' THEN '{b}'::text[]
		ELSE '{}' END`); err != nil {
		t.Fatal(err)
	}
	def := func(tag string) map[string]any {
		return map[string]any{"groups": []any{map[string]any{"match": "all", "conditions": []any{map[string]any{"field": "tag", "op": "has", "value": tag}}}}}
	}
	var seg struct{ ID string }
	c.Do(http.MethodPost, base+"/segments", map[string]any{"name": "Tagged a", "mode": "builder", "refresh": "static", "definition": def("a")}).
		MustStatus(t, 201).JSON(t, &seg)
	var firstEntered time.Time
	if err := env.App.Pool.QueryRow(ctx, `SELECT entered_at FROM segment_members sm JOIN contacts c ON c.id = sm.contact_id
		WHERE sm.segment_id = $1 AND c.email = 'reader15@paging.example'`, seg.ID).Scan(&firstEntered); err != nil {
		t.Fatal(err)
	}
	// Now tag b: reader10–14 leave, reader2x enter, reader15–19 stay.
	c.Do(http.MethodPut, base+"/segments/"+seg.ID, map[string]any{"name": "Tagged b", "mode": "builder", "refresh": "static", "definition": def("b")}).
		MustStatus(t, 200)
	var entered, left, members int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE kind = 'entered'), count(*) FILTER (WHERE kind = 'left'),
		(SELECT count(*) FROM segment_members WHERE segment_id = $1) FROM segment_member_events WHERE segment_id = $1`, seg.ID).Scan(&entered, &left, &members)
	if entered != 20 || left != 5 || members != 15 {
		t.Fatalf("entered %d (10 at first, 10 more), left %d, members %d", entered, left, members)
	}
	var stillEntered time.Time
	_ = env.App.Pool.QueryRow(ctx, `SELECT entered_at FROM segment_members sm JOIN contacts c ON c.id = sm.contact_id
		WHERE sm.segment_id = $1 AND c.email = 'reader15@paging.example'`, seg.ID).Scan(&stillEntered)
	if !stillEntered.Equal(firstEntered) {
		t.Fatalf("a member who stayed got a new entered_at: %v → %v", firstEntered, stillEntered)
	}
}
