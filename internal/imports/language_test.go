package imports_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// A column mapped to the contact's language is normalized ("Spanish",
// "es_MX" and "ES" become es and es-MX), values that aren't languages are
// listed on Map fields and left empty, and contacts remember the import as
// where their language came from.
func TestCSVImportLanguage(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Olive Owner"})
	ws := env.CreateWorkspace("Importers", owner.ID)
	c := env.Client()
	c.SignIn("owner@example.com", "")
	base := "/workspaces/" + ws.Slug

	var list struct{ ID string }
	c.Do(http.MethodPost, base+"/lists", map[string]any{"name": "Newsletter"}).MustStatus(t, 201).JSON(t, &list)
	c.Do(http.MethodPost, base+"/contacts", map[string]any{"email": "keeps@gmail.com", "lists": []string{list.ID}, "consent": "single", "language": "de"}).MustStatus(t, 201)
	var d detail
	c.Do(http.MethodPost, base+"/imports", map[string]any{"source": "csv"}).MustStatus(t, 201).JSON(t, &d)
	id := d.ID
	file := strings.Join([]string{
		"email,first_name,lang",
		"ana@gmail.com,Ana,Spanish",
		"ben@gmail.com,Ben,es_MX",
		"cara@gmail.com,Cara,ES",
		"dan@gmail.com,Dan,Klingon",
		"eve@gmail.com,Eve,",
		"fay@gmail.com,Fay,pt-br",
		"gus@gmail.com,Gus,Klingon",
		"keeps@gmail.com,Kim,fr",
	}, "\n")
	svc := service(env, nil)
	if _, err := svc.Upload(ctx, &ws, uuid.MustParse(id), "people.csv", []byte(file)); err != nil {
		t.Fatalf("upload: %v", err)
	}
	opts := d.Options
	opts["lists"] = []map[string]string{{"sourceId": "file", "target": list.ID}}
	opts["pullSuppressions"] = false
	c.Do(http.MethodPatch, base+"/imports/"+id, map[string]any{"options": opts}).MustStatus(t, 200)
	c.Do(http.MethodPost, base+"/imports/"+id+"/stage", nil).MustStatus(t, 200)
	if err := svc.Stage(ctx, uuid.MustParse(id)); err != nil {
		t.Fatalf("stage: %v", err)
	}

	var fields struct {
		Fields []struct{ Source, Target, Type string }
	}
	c.Do(http.MethodGet, base+"/imports/"+id+"/fields", nil).MustStatus(t, 200).JSON(t, &fields)
	choices := []map[string]string{}
	for _, f := range fields.Fields {
		if f.Source == "lang" && (f.Target != "language" || f.Type != "language") {
			t.Fatalf("lang is guessed as the language column: %+v", f)
		}
		choices = append(choices, map[string]string{"source": f.Source, "target": f.Target})
	}
	var col struct {
		Normalized []struct {
			Value, Code string
			Rows        int64
		}
		Unknown []struct {
			Value string
			Rows  int64
		}
		UnknownValues int
		UnknownRows   int64
		Languages     []struct {
			Code, Name string
			Rows       int64
		}
	}
	c.Do(http.MethodGet, base+"/imports/"+id+"/fields/lang/languages", nil).MustStatus(t, 200).JSON(t, &col)
	if col.UnknownValues != 1 || col.UnknownRows != 2 || len(col.Unknown) != 1 || col.Unknown[0].Value != "Klingon" {
		t.Fatalf("unknown values: %+v", col)
	}
	norm := map[string]string{}
	for _, n := range col.Normalized {
		norm[n.Value] = n.Code
	}
	if norm["Spanish"] != "es" || norm["es_MX"] != "es-MX" || norm["ES"] != "es" || norm["pt-br"] != "pt-BR" {
		t.Fatalf("normalized: %+v", col.Normalized)
	}
	if len(col.Languages) == 0 || col.Languages[0].Code != "es" || col.Languages[0].Rows != 3 {
		t.Fatalf("languages: %+v", col.Languages)
	}
	c.Do(http.MethodGet, base+"/imports/"+id+"/fields/nope/languages", nil).MustStatus(t, 404)

	c.Do(http.MethodPut, base+"/imports/"+id+"/fields", map[string]any{"fields": choices}).MustStatus(t, 200)
	c.Do(http.MethodPut, base+"/imports/"+id+"/consent", map[string]any{
		"map": map[string]string{"subscribed": "confirmed", "pending": "pending", "transactional": "skip"}, "evidence": "trust", "pullSuppressions": false, "recordHistory": true,
	}).MustStatus(t, 200)
	c.Do(http.MethodPost, base+"/imports/"+id+"/preview", nil).MustStatus(t, 200)
	c.Do(http.MethodPost, base+"/imports/"+id+"/start", nil).MustStatus(t, 200)
	if err := svc.Run(ctx, uuid.MustParse(id)); err != nil {
		t.Fatalf("run: %v", err)
	}
	for email, want := range map[string]string{"ana@gmail.com": "es", "ben@gmail.com": "es-MX", "cara@gmail.com": "es", "dan@gmail.com": "", "eve@gmail.com": "", "fay@gmail.com": "pt-BR", "keeps@gmail.com": "de"} {
		var lang, source, name string
		if err := env.App.Pool.QueryRow(ctx, `SELECT language, language_source, language_source_name FROM contacts WHERE workspace_id = $1 AND email = $2`, ws.ID, email).Scan(&lang, &source, &name); err != nil {
			t.Fatalf("%s: %v", email, err)
		}
		if lang != want {
			t.Errorf("%s: language %q, want %q", email, lang, want)
		}
		switch {
		case email == "keeps@gmail.com" && source != "team":
			t.Errorf("an import fills an empty language only, and keeps where a known one came from: %q", source)
		case want != "" && email != "keeps@gmail.com" && (source != "import" || name != "CSV"):
			t.Errorf("%s: source %q %q, want the CSV import", email, source, name)
		case want == "" && source != "":
			t.Errorf("%s: no language, but a source %q", email, source)
		}
	}
}
