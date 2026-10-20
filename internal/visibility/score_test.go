package visibility_test

// The score and its band as Visibility conditions, against the demo's
// scores (internal/scoring): the picker's catalog, a count of a real
// audience, preview as, and Scoring's Used by.

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/visibility"
)

func TestScoreConditionsOfAnAudience(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	// The picker: Score and Score band after the engagement tier, the bands
	// in their order and the workspace's thresholds.
	var cat struct {
		Groups []string
		Kinds  []struct {
			Key, Group string
			Ops        []struct{ Key string }
		}
		Bands []struct{ Value, Label string }
		Score struct {
			Configured        bool
			WarmFrom, HotFrom int
		}
	}
	c.Do("GET", ws+"/visibility/catalog", nil).MustStatus(t, 200).JSON(t, &cat)
	var kinds, bands []string
	for _, k := range cat.Kinds {
		kinds = append(kinds, k.Group+":"+k.Key)
		if k.Key == "score" && (len(k.Ops) != 3 || k.Ops[2].Key != "between") {
			t.Fatalf("score operators: %+v", k.Ops)
		}
	}
	for _, b := range cat.Bands {
		bands = append(bands, b.Value+"="+b.Label)
	}
	if got := strings.Join(kinds, ","); !strings.Contains(got, "engagement:tier,engagement:score,engagement:score_band") {
		t.Fatalf("kinds: %s", got)
	}
	if strings.Join(bands, ",") != "hot=Hot,warm=Warm,cold=Cold" || !cat.Score.Configured || cat.Score.WarmFrom != 20 || cat.Score.HotFrom != 50 {
		t.Fatalf("bands %v, score %+v", bands, cat.Score)
	}

	// Four readers of the boards on a list of the test's own: Aisha Rahman
	// (91) and Sam Rivera (52) are Hot, Hannah Berg (47) Warm, Kenji
	// Watanabe (18) Cold.
	var list struct{ ID string }
	c.Do("POST", ws+"/lists", map[string]any{"name": "Score test"}).MustStatus(t, 201).JSON(t, &list)
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO list_memberships (list_id, contact_id, status, subscribed_at)
		SELECT $1, c.id, 'subscribed', now() FROM contacts c JOIN workspaces w ON w.id = c.workspace_id
		WHERE w.slug = 'acme-newsletter' AND c.email IN ('aisha@rahmanlabs.io', 'sam.rivera@outlook.com', 'hannah@bergconsulting.de', 'kenji.w@icloud.com')`, list.ID); err != nil {
		t.Fatal(err)
	}
	scores := one[string](t, env, `SELECT string_agg(c.first_name || '=' || cs.score, ',' ORDER BY c.first_name) FROM list_memberships m
		JOIN contacts c ON c.id = m.contact_id JOIN contact_scores cs ON cs.contact_id = c.id WHERE m.list_id = $1`, list.ID)
	if scores != "Aisha=91,Hannah=47,Kenji=18,Sam=52" {
		t.Fatalf("the readers' scores: %s", scores)
	}

	var draft struct{ ID string }
	c.Do("POST", ws+"/campaigns", map[string]any{"name": "Score test draft"}).MustStatus(t, 201).JSON(t, &draft)
	c.Do("PATCH", ws+"/campaigns/"+draft.ID, map[string]any{"audience": map[string]any{"lists": []string{list.ID}}}).MustStatus(t, 200)
	hot := visibility.VisibilityRule{ID: "hot001", Show: visibility.ShowOnly, Conditions: []visibility.VisibilityCondition{{Kind: visibility.KindScoreBand, Op: "is", Value: "hot"}}}
	warm := visibility.VisibilityRule{ID: "warm01", Show: visibility.ShowOnly, Conditions: []visibility.VisibilityCondition{{Kind: visibility.KindScore, Op: "between", Value: "40", To: "49"}}}
	low := visibility.VisibilityRule{ID: "low001", Show: visibility.ShowExcept, Conditions: []visibility.VisibilityCondition{{Kind: visibility.KindScore, Op: "at_least", Value: "20"}}}
	src := `<mjml><mj-body>
<mj-section><mj-column><mj-text>Acme</mj-text></mj-column></mj-section>
<mj-section rc-name="Book a demo" rc-visible="` + attr(hot) + `"><mj-column><mj-text>Want a walkthrough for your team?</mj-text></mj-column></mj-section>
<mj-section rc-name="Almost there" rc-visible="` + attr(warm) + `"><mj-column><mj-text>Three posts you missed.</mj-text></mj-column></mj-section>
<mj-section rc-name="Start here" rc-visible="` + attr(low) + `"><mj-column><mj-text>New to the digest? Start here.</mj-text></mj-column></mj-section>
<!-- rc:footer -->
<mj-section><mj-column><mj-text><a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>
</mj-body></mjml>`
	res := mjml.Compile(src)
	if len(res.Errors) > 0 {
		t.Fatal(res.Errors)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET mjml = $2, html = $3 WHERE id = $1`, draft.ID, src, res.HTML); err != nil {
		t.Fatal(err)
	}

	// The saved content's versions, counted with the live scores.
	var r report
	c.Do("POST", ws+"/campaigns/"+draft.ID+"/visibility", map[string]any{}).MustStatus(t, 200).JSON(t, &r)
	shown, words := map[string]int64{}, map[string]string{}
	for _, b := range r.Blocks {
		shown[b.ID], words[b.ID] = b.Shown, b.Words
	}
	if !r.Counted || r.Total != 4 || len(r.Versions) != 3 || shown["hot001"] != 2 || shown["warm01"] != 1 || shown["low001"] != 1 {
		t.Fatalf("report: %+v", r)
	}
	if words["hot001"] != "Score band is Hot" || words["warm01"] != "Score is between 40 and 49" || words["low001"] != "Except Score is at least 20" {
		t.Fatalf("badges: %v", words)
	}

	// Kenji's version, and why he doesn't get Book a demo.
	kenji := one[string](t, env, `SELECT id::text FROM contacts WHERE email = 'kenji.w@icloud.com' AND workspace_id = (SELECT id FROM workspaces WHERE slug = 'acme-newsletter')`)
	var p struct {
		Version int
		Hidden  []struct{ ID, Why string }
		Shown   []string
		Traits  []string
	}
	c.Do("POST", ws+"/campaigns/"+draft.ID+"/visibility/preview", map[string]any{"contactId": kenji}).MustStatus(t, 200).JSON(t, &p)
	if len(p.Hidden) != 2 || p.Hidden[0].ID != "hot001" || p.Hidden[0].Why != "Kenji Watanabe’s score band is Cold, not Hot" || p.Hidden[1].Why != "Kenji Watanabe’s score is 18" ||
		strings.Join(p.Shown, ",") != "low001" || strings.Join(p.Traits, ",") != "score 18 · Cold" || p.Version == 0 {
		t.Fatalf("Kenji's version: %+v", p)
	}
	// Sam's version leaves out two blocks that aren't in alphabetical order
	// in the email: it is still found among the versions.
	sam := one[string](t, env, `SELECT id::text FROM contacts WHERE email = 'sam.rivera@outlook.com' AND workspace_id = (SELECT id FROM workspaces WHERE slug = 'acme-newsletter')`)
	c.Do("POST", ws+"/campaigns/"+draft.ID+"/visibility/preview", map[string]any{"contactId": sam}).MustStatus(t, 200).JSON(t, &p)
	if strings.Join(p.Shown, ",") != "hot001" || len(p.Hidden) != 2 || p.Version == 0 || strings.Join(p.Traits, ",") != "score 52 · Hot" {
		t.Fatalf("Sam's version: %+v", p)
	}
	// A reader to preview as for each version.
	var who struct {
		Items []struct {
			Name    string
			Trait   string
			Version int
		}
	}
	c.Do("GET", ws+"/campaigns/"+draft.ID+"/visibility/preview-contacts", nil).MustStatus(t, 200).JSON(t, &who)
	versions := map[int]string{}
	for _, x := range who.Items {
		versions[x.Version] = x.Trait
	}
	if _, none := versions[0]; len(versions) != 3 || none || !strings.HasPrefix(versions[1], "score ") || !strings.HasSuffix(versions[1], " · Hot") {
		t.Fatalf("preview contacts: %+v", who.Items)
	}

	// Scoring's Used by lists the three blocks with the last count, after
	// the demo's automation, segment and #143's Book a demo.
	var page struct {
		UsedBy []struct {
			Kind, Name, Where, Status, Href, Show string
			Contacts, Of                          *int64
			Conditions                            []struct{ Field, Op, Value, To string }
		}
	}
	c.Do("GET", ws+"/scoring", nil).MustStatus(t, 200).JSON(t, &page)
	var mine []string
	for _, u := range page.UsedBy {
		if u.Where != "Score test draft" {
			continue
		}
		if u.Kind != "block" || u.Status != "draft" || u.Contacts == nil || u.Of == nil || *u.Of != 4 || len(u.Conditions) != 1 || !strings.Contains(u.Href, "/campaigns/"+draft.ID+"/edit?select=") {
			t.Fatalf("a block: %+v", u)
		}
		cond := u.Conditions[0]
		value := cond.Value
		if cond.To != "" {
			value += "-" + cond.To
		}
		mine = append(mine, u.Name+":"+u.Show+":"+cond.Field+" "+cond.Op+" "+value+":"+strconv.FormatInt(*u.Contacts, 10))
	}
	if got := strings.Join(mine, ","); got != "Book a demo:only:score_band is hot:2,Almost there:only:score between 40-49:1,Start here:except:score at_least 20:1" {
		t.Fatalf("used by: %s", got)
	}
}
