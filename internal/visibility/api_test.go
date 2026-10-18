package visibility_test

// Conditional content over the API, against the demo: counting an
// audience's versions with the facts read from Postgres, previewing one
// contact's version, and where a segment is used.

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/visibility"
)

const ws = "/workspaces/acme-newsletter"

func one[T any](t *testing.T, env *apptest.Env, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

func attr(r visibility.VisibilityRule) string {
	return strings.ReplaceAll(r.Encode(), `"`, "&quot;")
}

type report struct {
	Counted   bool
	CountedAt string
	Total     int64
	Missing   int
	Possible  int
	Blocks    []struct {
		ID       string
		Name     string
		Shown    int64
		Words    string
		Versions []int
	}
	Versions []struct {
		Number     int
		Shows      string
		Hidden     []string
		Recipients int64
		SizeBytes  int
		Empty      bool
	}
}

func TestVersionsOfAnAudience(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	// A draft of the test's own, to the Beta Program list.
	beta := one[string](t, env, `SELECT id::text FROM lists WHERE name = 'Beta Program' AND deleted_at IS NULL AND workspace_id = (SELECT id FROM workspaces WHERE slug = 'acme-newsletter')`)
	var draft struct{ ID string }
	c.Do("POST", ws+"/campaigns", map[string]any{"name": "Conditional content test"}).MustStatus(t, 201).JSON(t, &draft)
	c.Do("PATCH", ws+"/campaigns/"+draft.ID, map[string]any{"audience": map[string]any{"lists": []string{beta}}}).MustStatus(t, 200)

	atRisk := visibility.VisibilityRule{ID: "risk01", Show: visibility.ShowOnly, Conditions: []visibility.VisibilityCondition{{Kind: visibility.KindTier, Op: "is", Value: "at_risk"}}}
	german := visibility.VisibilityRule{ID: "lang01", Show: visibility.ShowExcept, Match: visibility.MatchAny, Conditions: []visibility.VisibilityCondition{
		{Kind: visibility.KindLanguage, Op: "is", Value: "de"}, {Kind: visibility.KindLanguage, Op: "is", Value: "vi"}}}
	src := `<mjml><mj-body>
<!-- rc:header -->
<mj-section><mj-column><mj-text>Acme</mj-text></mj-column></mj-section>
<mj-section rc-name="Win-back" rc-visible="` + attr(atRisk) + `"><mj-column><mj-text>We've missed you.</mj-text></mj-column></mj-section>
<mj-section><mj-column><mj-text>This week in the beta.</mj-text></mj-column></mj-section>
<mj-section rc-name="English and Spanish" rc-visible="` + attr(german) + `"><mj-column><mj-text>` + strings.Repeat("Read the release notes. ", 60) + `</mj-text></mj-column></mj-section>
<!-- rc:footer -->
<mj-section><mj-column><mj-text><a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>
</mj-body></mjml>`
	res := mjml.Compile(src)
	if len(res.Errors) > 0 {
		t.Fatal(res.Errors)
	}
	var r report
	c.Do("POST", ws+"/campaigns/"+draft.ID+"/visibility", map[string]any{"mjml": src, "html": res.HTML}).MustStatus(t, 200).JSON(t, &r)
	if !r.Counted || r.Total == 0 || len(r.Blocks) != 2 || r.Possible != 4 {
		t.Fatalf("report: %+v", r)
	}
	// Every recipient gets exactly one version, and each block's count is
	// the sum of the versions that show it.
	var sum int64
	shown := map[string]int64{}
	for _, v := range r.Versions {
		sum += v.Recipients
		if v.Empty {
			t.Fatalf("no version is empty: %+v", v)
		}
		for _, b := range r.Blocks {
			if !contains(v.Hidden, b.ID) {
				shown[b.ID] += v.Recipients
			}
		}
	}
	if sum != r.Total || r.Missing != r.Possible-len(r.Versions) {
		t.Fatalf("versions add up to %d of %d: %+v", sum, r.Total, r.Versions)
	}
	for _, b := range r.Blocks {
		if shown[b.ID] != b.Shown {
			t.Fatalf("block %s: shown %d, versions say %d", b.ID, b.Shown, shown[b.ID])
		}
	}
	// "Only at risk" and "everyone except at risk" split the audience.
	except := atRisk
	except.Show = visibility.ShowExcept
	other := strings.Replace(src, attr(atRisk), attr(except), 1)
	var r2 report
	c.Do("POST", ws+"/campaigns/"+draft.ID+"/visibility", map[string]any{"mjml": other, "html": mjml.Compile(other).HTML}).MustStatus(t, 200).JSON(t, &r2)
	if r.Blocks[0].ID != "risk01" || r.Blocks[0].Words != "At risk tier" || r2.Total != r.Total || r.Blocks[0].Shown+r2.Blocks[0].Shown != r.Total {
		t.Fatalf("only %+v and except %+v don't split the %d recipients", r.Blocks[0], r2.Blocks[0], r.Total)
	}
	if r2.Blocks[1].Shown != r.Blocks[1].Shown {
		t.Fatalf("the other block's count didn't change: %d then %d", r.Blocks[1].Shown, r2.Blocks[1].Shown)
	}
	// The versions with the long block are larger.
	var with, without int
	for _, v := range r.Versions {
		if contains(v.Hidden, "lang01") {
			without = max(without, v.SizeBytes)
		} else {
			with = max(with, v.SizeBytes)
		}
	}
	if with <= without || without == 0 {
		t.Fatalf("sizes: with the notes %d, without %d", with, without)
	}

	// The count is kept until the rules or the audience change.
	var again report
	c.Do("POST", ws+"/campaigns/"+draft.ID+"/visibility", map[string]any{"mjml": other, "html": mjml.Compile(other).HTML}).MustStatus(t, 200).JSON(t, &again)
	if again.CountedAt != r2.CountedAt {
		t.Fatalf("an unchanged email reuses its count: %s then %s", r2.CountedAt, again.CountedAt)
	}
	c.Do("POST", ws+"/campaigns/"+draft.ID+"/visibility", map[string]any{"mjml": other, "html": mjml.Compile(other).HTML, "recount": true}).MustStatus(t, 200).JSON(t, &again)
	if again.CountedAt == r2.CountedAt {
		t.Fatal("Recount counts again")
	}

	// One contact's version, and why blocks are left out.
	german1 := one[string](t, env, `SELECT c.id::text FROM contacts c JOIN list_memberships m ON m.contact_id = c.id
		WHERE m.list_id = $1 AND m.status = 'subscribed' AND c.status = 'active' AND c.tier IN ('active', 'cooling') AND c.language LIKE 'de%' ORDER BY c.email LIMIT 1`, beta)
	var p struct {
		Name    string
		Version int
		Hidden  []struct{ ID, Name, Why string }
		Shown   []string
	}
	// (The count is now the "except at risk" email's.)
	c.Do("POST", ws+"/campaigns/"+draft.ID+"/visibility/preview", map[string]any{"contactId": german1, "mjml": other, "html": mjml.Compile(other).HTML}).MustStatus(t, 200).JSON(t, &p)
	if len(p.Hidden) != 1 || p.Hidden[0].ID != "lang01" || len(p.Shown) != 1 || p.Shown[0] != "risk01" || p.Version == 0 {
		t.Fatalf("a German reader who isn't at risk gets the win-back block only: %+v", p)
	}
	if !strings.Contains(p.Hidden[0].Why, p.Name) || !strings.Contains(p.Hidden[0].Why, "German") {
		t.Fatalf("Show why names the contact and the reason: %+v", p.Hidden[0])
	}

	// A Viewer (subscribers:view) counts the versions too.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("POST", ws+"/campaigns/"+draft.ID+"/visibility", map[string]any{"mjml": src, "html": res.HTML}).MustStatus(t, 200)
}

func TestSegmentUsage(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var seg struct{ ID, Name string }
	def := map[string]any{"groups": []map[string]any{{"match": "all", "conditions": []map[string]any{{"field": "tier", "op": "is", "value": "active"}}}}}
	c.Do("POST", ws+"/segments", map[string]any{"name": "Visibility usage test", "refresh": "dynamic", "mode": "builder", "definition": def}).MustStatus(t, 201).JSON(t, &seg)
	var draft struct{ ID string }
	c.Do("POST", ws+"/campaigns", map[string]any{"name": "Usage test draft"}).MustStatus(t, 201).JSON(t, &draft)
	r := visibility.VisibilityRule{ID: "seg001", Show: visibility.ShowOnly, Conditions: []visibility.VisibilityCondition{{Kind: visibility.KindSegment, Op: "in", Value: seg.ID, Label: seg.Name}}}
	src := `<mjml><mj-body><mj-section><mj-column><mj-text>Hi</mj-text></mj-column></mj-section>
<mj-section rc-name="Annual billing offer" rc-visible="` + attr(r) + `"><mj-column><mj-text>Save two months</mj-text></mj-column></mj-section></mj-body></mjml>`
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE campaigns SET mjml = $2 WHERE id = $1`, draft.ID, src); err != nil {
		t.Fatal(err)
	}
	var u struct {
		Blocks int
		Emails []struct {
			Name   string
			Blocks []string
		}
	}
	c.Do("GET", ws+"/visibility/usage?kind=segment&ref="+seg.ID, nil).MustStatus(t, 200).JSON(t, &u)
	if u.Blocks != 1 || len(u.Emails) != 1 || u.Emails[0].Name != "Usage test draft" || len(u.Emails[0].Blocks) != 1 || u.Emails[0].Blocks[0] != "Annual billing offer" {
		t.Fatalf("usage: %+v", u)
	}
	c.Do("GET", ws+"/visibility/usage?kind=segment&ref=00000000-0000-7000-8000-000000000000", nil).MustStatus(t, 200).JSON(t, &u)
	if u.Blocks != 0 {
		t.Fatalf("an unused segment: %+v", u)
	}
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}
