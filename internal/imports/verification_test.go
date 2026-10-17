package imports_test

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/imports"
	"github.com/hostrepute/reputecast/internal/store"
)

// fakeChecks is a verification service's answers for one import.
type fakeChecks struct{ answers map[string]imports.ImportAnswer }

func (f fakeChecks) ImportAnswers(context.Context, *store.Workspace, uuid.UUID) (*imports.ImportChecks, error) {
	at := time.Date(2026, 10, 15, 13, 12, 0, 0, time.UTC)
	return &imports.ImportChecks{JobID: uuid.Must(uuid.NewV7()), Service: "zerobounce", ServiceName: "ZeroBounce", Status: "finished", CheckedAt: &at,
		Checked: int64(len(f.answers)), Cost: 0.04, Answers: f.answers}, nil
}

// TestImportWithAddressChecks: a verification service's answers join the
// preview's groups (its invalid rows are skipped, its risky and unknown ones
// imported and tagged), and the import follows them (X14).
func TestImportWithAddressChecks(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Olive Owner"})
	ws := env.CreateWorkspace("Checked imports", owner.ID)
	c := env.Client()
	c.SignIn("owner@example.com", "")
	base := "/workspaces/" + ws.Slug

	var list struct{ ID string }
	c.Do(http.MethodPost, base+"/lists", map[string]any{"name": "Newsletter"}).MustStatus(t, 201).JSON(t, &list)
	var d detail
	c.Do(http.MethodPost, base+"/imports", map[string]any{"source": "csv"}).MustStatus(t, 201).JSON(t, &d)
	id := d.ID
	file := strings.Join([]string{
		"email,first_name,status",
		"ana@gmail.com,Ana,subscribed",
		"bad@corpmail.io,Bad,subscribed",
		"cat@catchall.io,Cat,subscribed",
		"unk@slowmx.io,Unk,subscribed",
		"info@gmail.com,Info,subscribed",
		"temp@mailinator.com,Temp,subscribed",
	}, "\n")
	svc := service(env, nil)
	svc.Checks = fakeChecks{answers: map[string]imports.ImportAnswer{
		"ana@gmail.com":   {Result: "valid", Code: "valid"},
		"bad@corpmail.io": {Result: "invalid", Reason: "invalid_address", Code: "invalid / mailbox_not_found"},
		"cat@catchall.io": {Result: "risky", Reason: "catch_all", Code: "catch-all"},
		"unk@slowmx.io":   {Result: "unknown", Code: "unknown"},
		"info@gmail.com":  {Result: "risky", Reason: "role", Code: "do_not_mail / role_based"},
	}}
	if _, err := svc.Upload(ctx, &ws, uuid.MustParse(id), "people.csv", []byte(file)); err != nil {
		t.Fatalf("upload: %v", err)
	}
	c.Do(http.MethodGet, base+"/imports/"+id, nil).MustStatus(t, 200).JSON(t, &d)
	opts := d.Options
	opts["lists"] = []map[string]string{{"sourceId": "file", "target": list.ID}}
	opts["pullSuppressions"] = false
	c.Do(http.MethodPatch, base+"/imports/"+id, map[string]any{"options": opts}).MustStatus(t, 200)
	c.Do(http.MethodPost, base+"/imports/"+id+"/stage", nil).MustStatus(t, 200)
	if err := svc.Stage(ctx, uuid.MustParse(id)); err != nil {
		t.Fatalf("stage: %v", err)
	}
	var fields struct {
		Fields []struct{ Source, Target string }
	}
	c.Do(http.MethodGet, base+"/imports/"+id+"/fields", nil).MustStatus(t, 200).JSON(t, &fields)
	choices := []map[string]string{}
	for _, f := range fields.Fields {
		choices = append(choices, map[string]string{"source": f.Source, "target": f.Target})
	}
	c.Do(http.MethodPut, base+"/imports/"+id+"/fields", map[string]any{"fields": choices}).MustStatus(t, 200)
	c.Do(http.MethodPut, base+"/imports/"+id+"/consent", map[string]any{
		"map": map[string]string{"subscribed": "confirmed", "pending": "pending", "transactional": "skip"}, "evidence": "trust", "pullSuppressions": false, "recordHistory": true,
	}).MustStatus(t, 200)

	// The preview runs through the API's own importer, which has no service
	// wired here; the inline one with the answers computes the same thing.
	p, err := svc.Preview(ctx, &ws, uuid.MustParse(id))
	if err != nil {
		t.Fatal(err)
	}
	if p.WillImport != 1 || p.Invalid != 1 || p.Risky != 4 {
		t.Fatalf("preview: import %d, invalid %d, risky %d", p.WillImport, p.Invalid, p.Risky)
	}
	v := p.Verification
	if v == nil || v.ServiceName != "ZeroBounce" || v.Invalid != 1 || v.Risky != 2 || v.Fewer != 3 {
		t.Fatalf("verification line: %+v", v)
	}
	groups := map[string]imports.ImportRiskyGroup{}
	for _, g := range p.RiskyGroups {
		groups[g.Key] = g
	}
	if g := groups["role"]; g.Rows != 1 || g.Agree != 1 || g.More != 0 {
		t.Errorf("role: %+v", g)
	}
	if g := groups["disposable"]; g.Rows != 1 || g.Own != 1 || g.More != 0 || g.Agree != 0 {
		t.Errorf("disposable: %+v", g)
	}
	if g := groups["catch_all"]; g.Rows != 1 || g.More != 1 || g.Action != "import" {
		t.Errorf("catch-all: %+v", g)
	}
	if g := groups["unknown"]; g.Rows != 1 || g.More != 1 {
		t.Errorf("unknown: %+v", g)
	}
	if i := slices.IndexFunc(p.InvalidGroups, func(g imports.ImportInvalidGroup) bool { return g.Service == "ZeroBounce" }); i < 0 || p.InvalidGroups[i].Rows != 1 {
		t.Errorf("invalid groups: %+v", p.InvalidGroups)
	}

	c.Do(http.MethodPost, base+"/imports/"+id+"/start", nil).MustStatus(t, 200)
	if err := svc.Run(ctx, uuid.MustParse(id)); err != nil {
		t.Fatalf("run: %v", err)
	}
	c.Do(http.MethodGet, base+"/imports/"+id, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Stats == nil || d.Stats.Reasons["invalid_service"] != 1 || d.Stats.Imported != 4 {
		t.Fatalf("stats: %+v", d.Stats)
	}
	tags := func(email string) []string {
		var out []string
		if err := env.App.Pool.QueryRow(ctx, `SELECT tags FROM contacts WHERE workspace_id = $1 AND email = $2`, ws.ID, email).Scan(&out); err != nil {
			t.Fatalf("%s: %v", email, err)
		}
		return out
	}
	if got := tags("cat@catchall.io"); !slices.Contains(got, "verify:catch-all") {
		t.Errorf("catch-all tags %v", got)
	}
	if got := tags("unk@slowmx.io"); !slices.Contains(got, "verify:unknown") {
		t.Errorf("unknown tags %v", got)
	}
	var gone int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM contacts WHERE workspace_id = $1 AND email IN ('bad@corpmail.io', 'temp@mailinator.com')`, ws.ID).Scan(&gone)
	if gone != 0 {
		t.Errorf("%d skipped rows became contacts", gone)
	}
}
