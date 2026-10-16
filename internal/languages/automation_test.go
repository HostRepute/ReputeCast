package languages_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

func itoa(n int) string { return strconv.Itoa(n) }

func campaignByName(t *testing.T, env *apptest.Env, name string) string {
	t.Helper()
	var id string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT id::text FROM campaigns WHERE name = $1`, name).Scan(&id); err != nil {
		t.Fatalf("campaign %q: %v", name, err)
	}
	return id
}

// The sample's language versions: Weekly Digest #142 goes out in English,
// Spanish and German, all up to date, and its approval and Review & send
// run cover them; #143 (when the accessibility sample is there) has German
// outdated by two blocks.
func TestDemoLanguageVersions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	id := campaignByName(t, env, "Weekly Digest #142")
	var langs emailLanguages
	linh.Do("GET", ws+"/campaigns/"+id+"/languages", nil).MustStatus(t, 200).JSON(t, &langs)
	if langs.Source != "en" || langs.version(t, "es").Status != "up_to_date" || langs.version(t, "de").Status != "up_to_date" ||
		langs.version(t, "vi").Status != "missing" || langs.version(t, "vi").Gets != "en" {
		t.Fatalf("#142: %+v", langs)
	}
	var review struct {
		Stale bool
		Run   struct {
			Checks []struct {
				Key, Status string
				Rows        []struct {
					Label, Language, Gets string
					Count                 int64
				}
			}
		}
		Approval struct{ Valid bool }
	}
	linh.Do("GET", ws+"/campaigns/"+id+"/review", nil).MustStatus(t, 200).JSON(t, &review)
	if review.Stale || !review.Approval.Valid {
		t.Fatalf("#142's run and approval cover its versions: stale %v, approval valid %v", review.Stale, review.Approval.Valid)
	}
	rows := 0
	for _, c := range review.Run.Checks {
		if c.Key == "languages.recipients" {
			rows = len(c.Rows)
			if c.Rows[3].Language != "vi" || c.Rows[3].Gets != "en" || c.Rows[3].Count != 506 {
				t.Fatalf("Vietnamese readers: %+v", c.Rows[3])
			}
		}
	}
	if rows != 4 {
		t.Fatalf("Recipients per language: %d rows", rows)
	}
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM campaigns WHERE name = 'Weekly Digest #143'`).Scan(&n); err != nil || n == 0 {
		return // the accessibility sample isn't merged in
	}
	linh.Do("GET", ws+"/campaigns/"+campaignByName(t, env, "Weekly Digest #143")+"/languages", nil).MustStatus(t, 200).JSON(t, &langs)
	if de := langs.version(t, "de"); de.Status != "outdated" || de.Attention != 2 || langs.version(t, "es").Status != "up_to_date" {
		t.Fatalf("#143: %+v", langs)
	}
}

type saved struct {
	Version   int
	Languages emailLanguages
}

// An automation email's language versions over HTTP: writing one makes the
// email's next version, editing the email carries them along (Outdated
// where it changed), and the publish dialog names them.
func TestAutomationLanguagesAPI(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh, aisha := env.Client(), env.Client()
	linh.SignInDemo("linh@acme.io")
	aisha.SignInDemo("aisha@acme.io")
	saves, removes := audited(t, env, "automation_email.language.save"), audited(t, env, "automation_email.language.remove")

	var email struct {
		ID            string
		LatestVersion int
	}
	linh.Do("POST", ws+"/automation-emails", map[string]any{"name": `Step 1 "Welcome"`, "subject": "Welcome"}).MustStatus(t, 201).JSON(t, &email)
	base := ws + "/automation-emails/" + email.ID + "/languages"
	html := `<html><body><p>Welcome aboard.</p><p><a href="https://acme.io/start">Get started</a></p>` +
		`<p><a href="{{ unsubscribe_url }}">Unsubscribe</a> · Acme Inc · 500 Market St</p></body></html>`
	var content struct{ Version int }
	linh.Do("PUT", ws+"/automation-emails/"+email.ID, map[string]any{"subject": "Welcome", "preheader": "", "html": html}).MustStatus(t, 200).JSON(t, &content)
	v := content.Version

	var langs emailLanguages
	aisha.Do("GET", base+"?version="+itoa(v), nil).MustStatus(t, 200).JSON(t, &langs)
	if langs.Source != "en" || langs.CanEdit || langs.version(t, "es").Status != "missing" {
		t.Fatalf("before Spanish: %+v", langs)
	}
	es := strings.NewReplacer("Welcome aboard.", "Bienvenido a bordo.", "Get started", "Empezar", "Unsubscribe", "Darse de baja").Replace(html)
	aisha.Do("PUT", base+"/es", map[string]any{"version": v, "subject": "Bienvenido", "html": es, "translated": true}).MustStatus(t, 403)
	linh.Do("PUT", base+"/en", map[string]any{"version": v, "subject": "Welcome", "html": html}).MustStatus(t, 422)
	var out saved
	linh.Do("PUT", base+"/es", map[string]any{"version": v, "subject": "Bienvenido", "html": es, "translated": true}).MustStatus(t, 200).JSON(t, &out)
	if out.Version != v+1 || out.Languages.version(t, "es").Status != "up_to_date" {
		t.Fatalf("Spanish written as the next version: %+v", out)
	}
	withSpanish := out.Version
	var one struct{ Language, Status, Subject, HTML string }
	linh.Do("GET", base+"/es?version="+itoa(withSpanish), nil).MustStatus(t, 200).JSON(t, &one)
	if one.Status != "up_to_date" || one.Subject != "Bienvenido" || !strings.Contains(one.HTML, "Bienvenido a bordo") {
		t.Fatalf("Spanish content: %+v", one)
	}
	// The version the step sent before has no Spanish: it never changes.
	linh.Do("GET", base+"?version="+itoa(v), nil).MustStatus(t, 200).JSON(t, &langs)
	if langs.version(t, "es").Status != "missing" {
		t.Fatalf("the earlier version stays as it was: %+v", langs.version(t, "es"))
	}

	// Editing the email carries Spanish to its next version, outdated.
	linh.Do("PUT", ws+"/automation-emails/"+email.ID, map[string]any{"subject": "Welcome", "preheader": "", "html": strings.Replace(html, "Welcome aboard.", "Welcome aboard, and thanks.", 1)}).
		MustStatus(t, 200).JSON(t, &content)
	linh.Do("GET", base+"?version="+itoa(content.Version), nil).MustStatus(t, 200).JSON(t, &langs)
	if s := langs.version(t, "es"); s.Status != "outdated" {
		t.Fatalf("Spanish after the edit: %+v", s)
	}
	linh.Do("POST", base+"/es/mark-up-to-date", map[string]any{"version": content.Version}).MustStatus(t, 200).JSON(t, &out)
	if out.Version != content.Version+1 || out.Languages.version(t, "es").Status != "up_to_date" {
		t.Fatalf("marked up to date: %+v", out)
	}
	linh.Do("DELETE", base+"/es?version="+itoa(out.Version), nil).MustStatus(t, 200).JSON(t, &out)
	if out.Languages.version(t, "es").Status != "missing" || audited(t, env, "automation_email.language.save") != saves+1 || audited(t, env, "automation_email.language.remove") != removes+1 {
		t.Fatalf("removed: %+v", out)
	}

	// A flow sends v; its draft switches the step to the version with
	// Spanish: the publish dialog names it instead of a changed step.
	flows := ws + "/automations"
	var flow struct{ ID string }
	linh.Do("POST", flows, map[string]any{"name": "Languages flow"}).MustStatus(t, 201).JSON(t, &flow)
	graph := func(version int) map[string]any {
		return map[string]any{
			"nodes": []map[string]any{
				{"id": "t", "type": "trigger", "position": map[string]any{"x": 0, "y": 0}, "data": map[string]any{"trigger": map[string]any{"kind": "api_event", "event": "signed_up"}}},
				{"id": "m", "type": "email", "position": map[string]any{"x": 0, "y": 120},
					"data": map[string]any{"email": map[string]any{"emailId": email.ID, "emailVersion": version, "subject": "Welcome"}}},
			},
			"edges": []map[string]any{{"id": "t-m", "source": "t", "target": "m", "data": map[string]any{}}},
		}
	}
	var draft struct {
		DraftVersion int
		Revision     int64
	}
	linh.Do("PUT", flows+"/"+flow.ID+"/draft", map[string]any{"graph": graph(v), "revision": 0}).MustStatus(t, 200).JSON(t, &draft)
	linh.Do("POST", flows+"/"+flow.ID+"/publish", map[string]any{"version": draft.DraftVersion, "migration": "finish"}).MustStatus(t, 200)
	var detail struct{ Revision int64 }
	linh.Do("GET", flows+"/"+flow.ID, nil).MustStatus(t, 200).JSON(t, &detail)
	linh.Do("PUT", flows+"/"+flow.ID+"/draft", map[string]any{"graph": graph(withSpanish), "revision": detail.Revision}).MustStatus(t, 200)
	var preview struct {
		Items []struct {
			Kind, Title string
			NodeIDs     []string
		}
		Checks []struct{ Status, Label string }
	}
	linh.Do("GET", flows+"/"+flow.ID+"/publish-preview", nil).MustStatus(t, 200).JSON(t, &preview)
	if len(preview.Items) == 0 || preview.Items[0].Kind != "added" || preview.Items[0].Title != "Spanish version of step 1" {
		t.Fatalf("publish diff: %+v", preview.Items)
	}
	for _, it := range preview.Items {
		if it.Kind == "changed" {
			t.Fatalf("a step whose email changed only in its translations is listed as changed: %+v", preview.Items)
		}
	}
	// The step sends the edited email, whose Spanish is outdated: a warning
	// that doesn't stop publishing.
	linh.Do("GET", flows+"/"+flow.ID, nil).MustStatus(t, 200).JSON(t, &detail)
	linh.Do("PUT", flows+"/"+flow.ID+"/draft", map[string]any{"graph": graph(content.Version), "revision": detail.Revision}).MustStatus(t, 200)
	linh.Do("GET", flows+"/"+flow.ID+"/publish-preview", nil).MustStatus(t, 200).JSON(t, &preview)
	warned := false
	for _, c := range preview.Checks {
		warned = warned || (c.Status == "warn" && c.Label == "Step 1’s Spanish version is outdated")
	}
	if !warned {
		t.Fatalf("publish checks: %+v", preview.Checks)
	}
}
