package languages_test

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
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
