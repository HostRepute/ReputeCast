package audience_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The Audience screens' server text comes in the language the app asks
// for: problems, labels, country names, and the timeline's notes, which a
// consent event keeps for readers in another language. API clients that
// don't ask get English, and rows stored before a note was kept stay as
// they were written.
func TestAudienceInVietnamese(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Olive Owner"})
	w := env.CreateWorkspace("Audience", owner.ID)
	base := "/workspaces/" + w.Slug

	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignIn("owner@example.com", "")
	en := env.Client()
	en.SignIn("owner@example.com", "")

	// A problem.
	if p := vi.Do(http.MethodPost, base+"/lists", map[string]any{"name": " "}).MustStatus(t, 422).Problem(t); p["detail"] != "Hãy đặt tên cho danh sách." {
		t.Errorf("list name: %v", p["detail"])
	}
	if p := en.Do(http.MethodPost, base+"/lists", map[string]any{"name": " "}).MustStatus(t, 422).Problem(t); p["detail"] != "Give the list a name." {
		t.Errorf("list name in English: %v", p["detail"])
	}

	var weekly, beta struct{ ID string }
	en.Do(http.MethodPost, base+"/lists", map[string]any{"name": "Weekly"}).MustStatus(t, 201).JSON(t, &weekly)
	en.Do(http.MethodPost, base+"/lists", map[string]any{"name": "Beta"}).MustStatus(t, 201).JSON(t, &beta)
	var sam struct{ ID string }
	en.Do(http.MethodPost, base+"/contacts", map[string]any{"email": "sam@example.com", "lists": []string{weekly.ID}, "consent": "single"}).MustStatus(t, 201).JSON(t, &sam)
	if p := vi.Do(http.MethodPost, base+"/contacts", map[string]any{"email": "sam@example.com", "lists": []string{weekly.ID}, "consent": "single"}).MustStatus(t, 409).Problem(t); p["detail"] != "sam@example.com đã là một liên hệ. Hãy mở hồ sơ của họ để đổi danh sách." {
		t.Errorf("existing contact: %v", p["detail"])
	}
	en.Do(http.MethodPatch, base+"/contacts/"+sam.ID, map[string]any{"country": "US"}).MustStatus(t, 200)

	// Labels: the contact's source and country.
	var list struct {
		Items []struct{ SourceLabel string }
	}
	vi.Do(http.MethodGet, base+"/contacts", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].SourceLabel != "Do nhóm thêm vào" {
		t.Errorf("source in Vietnamese: %+v", list.Items)
	}
	en.Do(http.MethodGet, base+"/contacts", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].SourceLabel != "Added by the team" {
		t.Errorf("source in English: %+v", list.Items)
	}
	type profile struct {
		Fields []struct{ Key, Label, Value string }
	}
	field := func(p profile, key string) (string, string) {
		for _, f := range p.Fields {
			if f.Key == key {
				return f.Label, f.Value
			}
		}
		return "", ""
	}
	var pv, pe profile
	vi.Do(http.MethodGet, base+"/contacts/"+sam.ID, nil).MustStatus(t, 200).JSON(t, &pv)
	en.Do(http.MethodGet, base+"/contacts/"+sam.ID, nil).MustStatus(t, 200).JSON(t, &pe)
	if label, value := field(pv, "country_timezone"); label != "Quốc gia · múi giờ" || value != "Hoa Kỳ" {
		t.Errorf("country in Vietnamese: %q %q", label, value)
	}
	if label, value := field(pe, "country_timezone"); label != "Country · timezone" || value != "United States" {
		t.Errorf("country in English: %q %q", label, value)
	}

	// Stored for later readers: the team adds Sam to Beta in an English
	// request; the note reads in each reader's language.
	en.Do(http.MethodPost, base+"/contacts/bulk", map[string]any{"ids": []string{sam.ID}, "action": "add_to_list", "listId": beta.ID}).MustStatus(t, 200)
	// A note stored before notes were kept.
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO consent_events (id, contact_id, list_id, action, source, note, occurred_at)
		VALUES ($1, $2, $3, 'subscribed', 'team', 'added by the team', now() - interval '1 day')`, uuid.Must(uuid.NewV7()), uuid.MustParse(sam.ID), uuid.MustParse(weekly.ID)); err != nil {
		t.Fatal(err)
	}
	type activity struct {
		Items []struct{ Type, List, Note string }
	}
	notes := func(a activity) map[string]string {
		out := map[string]string{}
		for _, it := range a.Items {
			if it.Note != "" {
				out[it.List] = it.Note
			}
		}
		return out
	}
	var av, ae activity
	vi.Do(http.MethodGet, base+"/contacts/"+sam.ID+"/activity?filter=consent&limit=20", nil).MustStatus(t, 200).JSON(t, &av)
	en.Do(http.MethodGet, base+"/contacts/"+sam.ID+"/activity?filter=consent&limit=20", nil).MustStatus(t, 200).JSON(t, &ae)
	if n := notes(av); n["Beta"] != "do nhóm thêm vào" || n["Weekly"] != "added by the team" {
		t.Errorf("notes in Vietnamese: %v", n)
	}
	if n := notes(ae); n["Beta"] != "added by the team" || n["Weekly"] != "added by the team" {
		t.Errorf("notes in English: %v", n)
	}
}
