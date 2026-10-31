package languages_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/store"
)

// "Preview as" on the composer's canvas (ComposerLanguages): one contact
// per workspace language of Weekly Digest #142's audience, then one whose
// language Acme doesn't offer, each with the version they get.
func TestCampaignPreviewContactsAPI(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	id := campaignByName(t, env, "Weekly Digest #142")

	var list struct {
		Items []struct {
			ID, Name, Email, Language, Gets string
			Offered                         bool
		}
	}
	linh.Do("GET", ws+"/campaigns/"+id+"/languages/preview-contacts", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 5 {
		t.Fatalf("one contact per language and one not offered: %+v", list.Items)
	}
	base := func(code string) string {
		return strings.ToLower(strings.SplitN(strings.ReplaceAll(code, "_", "-"), "-", 2)[0])
	}
	// Spanish and German have versions; Vietnamese has none and gets the fallback.
	for i, want := range []struct{ lang, gets string }{{"en", "en"}, {"es", "es"}, {"de", "de"}, {"vi", "en"}} {
		it := list.Items[i]
		if base(it.Language) != want.lang || it.Gets != want.gets || !it.Offered || it.Email == "" || it.ID == "" {
			t.Fatalf("item %d: want %s getting %s, got %+v", i, want.lang, want.gets, it)
		}
	}
	if other := list.Items[4]; other.Offered || other.Gets != "en" || other.Language == "" {
		t.Fatalf("a language Acme doesn't offer gets English: %+v", other)
	}

	linh.Do("GET", ws+"/campaigns/01890000-0000-7000-8000-000000000000/languages/preview-contacts", nil).MustStatus(t, 404)
}

// Each language's contact to preview as: someone with a first name if the
// language has anyone, then the one who opened last, then the first email;
// the contacts sorting the whole audience picks.
func TestPreviewContactPerLanguage(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	var list uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM lists WHERE workspace_id = $1 AND name = 'Weekly Digest'`, w.ID).Scan(&list); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	add := func(email, first, lang string, opened *time.Time) {
		id := uuid.Must(uuid.NewV7())
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email, first_name, language, last_open_at) VALUES ($1, $2, $3, $4, $5, $6)`,
			id, w.ID, email, first, lang, opened); err != nil {
			t.Fatal(err)
		}
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO list_memberships (list_id, contact_id, status, subscribed_at) VALUES ($1, $2, 'subscribed', now())`, list, id); err != nil {
			t.Fatal(err)
		}
	}
	at := func(d time.Duration) *time.Time { v := now.Add(-d); return &v }
	// Dutch (the demo has none): two people with a first name who never
	// opened come before one without who did, and the first email wins
	// between them.
	add("zoe.nl@example.com", "", "nl", at(time.Hour))
	add("bea.nl@example.com", "Bea", "nl-BE", nil)
	add("ana.nl@example.com", "Ana", "nl", nil)
	// Swedish: nobody has a first name: the one who opened last.
	add("a.sv@example.com", "", "sv", at(48*time.Hour))
	add("b.sv@example.com", "", "sv_SE", at(time.Hour))
	add("c.sv@example.com", "", "sv", nil)

	params := store.LanguagesPreviewContactsParams{WorkspaceID: w.ID, Lists: []uuid.UUID{list}, ScopeLists: []uuid.UUID{}}
	rows, err := env.App.Q.LanguagesPreviewContacts(ctx, params)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	bases := []string{}
	for _, r := range rows {
		b := strings.ToLower(strings.SplitN(strings.ReplaceAll(r.Language, "_", "-"), "-", 2)[0])
		got[b] = r.Email
		bases = append(bases, b)
	}
	if got["nl"] != "ana.nl@example.com" || got["sv"] != "b.sv@example.com" {
		t.Fatalf("Dutch and Swedish: %v", got)
	}

	// Every language: the same contacts, in the same order, as sorting the
	// audience by language, first name, last open and email.
	sorted, err := env.App.Pool.Query(ctx, `
		SELECT DISTINCT ON (lower(split_part(replace(c.language, '_', '-'), '-', 1))) lower(split_part(replace(c.language, '_', '-'), '-', 1)), c.email
		FROM contacts c
		WHERE c.workspace_id = $1 AND c.erased_at IS NULL AND c.status = 'active' AND c.language <> ''
		  AND EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = c.id AND m.status = 'subscribed' AND m.list_id = $2)
		ORDER BY 1, (c.first_name = '') ASC, c.last_open_at DESC NULLS LAST, c.email`, w.ID, list)
	if err != nil {
		t.Fatal(err)
	}
	defer sorted.Close()
	i := 0
	for sorted.Next() {
		var base, email string
		if err := sorted.Scan(&base, &email); err != nil {
			t.Fatal(err)
		}
		if i >= len(bases) || bases[i] != base || got[base] != email {
			t.Fatalf("language %d: sorting picks %s %s, got %v %v", i, base, email, bases, got)
		}
		i++
	}
	if sorted.Err() != nil || i != len(bases) || i < 9 {
		t.Fatalf("%d languages sorted, %d picked (%v)", i, len(bases), sorted.Err())
	}
}
