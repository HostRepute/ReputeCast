package audience_test

import (
	"context"
	"net/http"
	"testing"
)

type contactLanguage struct {
	Code       string
	Name       string
	Offered    bool
	Gets       string
	Source     string
	SourceName string
	Browser    bool
	SetAt      *string
	Options    []struct {
		Code    string
		Name    string
		Offered bool
	}
}

func TestContactLanguage(t *testing.T) {
	env, c := demoEnv(t)
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	find := func(email string) string {
		t.Helper()
		var page struct {
			Items []struct{ ID, Email string }
		}
		c.Do(http.MethodGet, ws+"/contacts?q="+email, nil).MustStatus(t, 200).JSON(t, &page)
		if len(page.Items) != 1 {
			t.Fatalf("%s: %d contacts", email, len(page.Items))
		}
		return page.Items[0].ID
	}
	profile := func(id string) contactLanguage {
		t.Helper()
		var p struct{ Language contactLanguage }
		c.Do(http.MethodGet, ws+"/contacts/"+id, nil).MustStatus(t, 200).JSON(t, &p)
		return p.Language
	}

	t.Run("the profile says where the language came from", func(t *testing.T) {
		sam := profile(find("sam.rivera@outlook.com"))
		if sam.Code != "en" || sam.Source != "import" || sam.SourceName != "Brevo" || !sam.Offered || sam.SetAt == nil {
			t.Fatalf("Sam: %+v", sam)
		}
		if len(sam.Options) != 4 || sam.Options[0].Code != "en" || sam.Options[1].Name != "Spanish" {
			t.Fatalf("the select lists the workspace's languages: %+v", sam.Options)
		}
		lucia := profile(find("lucia.fernandez@example.com"))
		if lucia.Code != "es-MX" || lucia.Source != "form" || lucia.SourceName != "Newsletter signup" || !lucia.Browser || lucia.Gets != "es" {
			t.Fatalf("Lucía: %+v", lucia)
		}
		if lucia.Options[1].Code != "es-MX" {
			t.Fatalf("Spanish keeps Lucía's regional code: %+v", lucia.Options)
		}
		jonas := profile(find("jonas.weber@example.de"))
		if jonas.Code != "de-AT" || jonas.Source != "preference_center" {
			t.Fatalf("Jonas: %+v", jonas)
		}
		rafael := profile(find("rafael.souza@example.com.br"))
		if rafael.Code != "pt-BR" || rafael.Offered || rafael.Gets != "en" || rafael.Source != "api" {
			t.Fatalf("Rafael: %+v", rafael)
		}
		if len(rafael.Options) != 5 || rafael.Options[0].Code != "pt-BR" || rafael.Options[0].Offered || rafael.Options[0].Name != "Portuguese" {
			t.Fatalf("a language that isn't offered comes first, marked: %+v", rafael.Options)
		}
	})

	t.Run("changing it normalizes, records the member and audits", func(t *testing.T) {
		id := find("sam.rivera@outlook.com")
		before := audits(t, env, "contact.update")
		c.Do(http.MethodPatch, ws+"/contacts/"+id, map[string]any{"language": "Spanish"}).MustStatus(t, 200)
		got := profile(id)
		if got.Code != "es" || got.Source != "team" || got.SourceName != "Linh Nguyen" {
			t.Fatalf("after the change: %+v", got)
		}
		if audits(t, env, "contact.update") != before+1 {
			t.Fatal("no audit entry")
		}
		var lang string
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT after->>'language' FROM audit_events WHERE action = 'contact.update' ORDER BY occurred_at DESC LIMIT 1`).Scan(&lang); err != nil || lang != "es" {
			t.Fatalf("the audit entry keeps the new language: %q, %v", lang, err)
		}
		c.Do(http.MethodPatch, ws+"/contacts/"+id, map[string]any{"language": "es_mx"}).MustStatus(t, 200)
		if got := profile(id); got.Code != "es-MX" {
			t.Fatalf("es_mx is stored as es-MX: %+v", got)
		}
		c.Do(http.MethodPatch, ws+"/contacts/"+id, map[string]any{"language": "en"}).MustStatus(t, 200)
	})

	t.Run("an invalid value answers 422 with the closest match", func(t *testing.T) {
		id := find("sam.rivera@outlook.com")
		var p struct {
			Detail string
			Meta   map[string]string
			Errors []struct{ Location string }
		}
		c.Do(http.MethodPatch, ws+"/contacts/"+id, map[string]any{"language": "Spanosh"}).MustStatus(t, 422).JSON(t, &p)
		if p.Meta["closest"] != "es" || p.Detail != `"Spanosh" isn't a language. Did you mean Spanish (es)?` || len(p.Errors) != 1 || p.Errors[0].Location != "body.language" {
			t.Fatalf("problem: %+v", p)
		}
		c.Do(http.MethodPost, ws+"/contacts", map[string]any{"email": "new.reader@example.com", "lists": []string{}, "consent": "single", "language": "Klingon"}).MustStatus(t, 422)
		viewer.Do(http.MethodPatch, ws+"/contacts/"+id, map[string]any{"language": "de"}).MustStatus(t, 403)
	})

	t.Run("the Contacts filter", func(t *testing.T) {
		total := func(q string) int64 {
			t.Helper()
			var page struct{ Total int64 }
			c.Do(http.MethodGet, ws+"/contacts?"+q, nil).MustStatus(t, 200).JSON(t, &page)
			return page.Total
		}
		// SettingsLanguages: 4,180 Spanish (es, es-MX, es-ES…), 2,240
		// German, 773 Vietnamese, 5,508 with no language and 12 with a
		// language Acme doesn't offer.
		for q, want := range map[string]int64{"language=es": 4180, "language=de": 2240, "language=vi": 773, "language=none": 5508, "language=not_offered": 12} {
			if got := total(q); got != want {
				t.Errorf("%s: %d, want %d", q, got, want)
			}
		}
		// The Summit's 338 new registrants read English (X21, brief §8.12).
		if got := total("language=en"); got != 48213+338-4180-2240-773-5508-12 {
			t.Errorf("English: %d", got)
		}
		c.Do(http.MethodGet, ws+"/contacts?language=klingon", nil).MustStatus(t, 422)
	})
}
