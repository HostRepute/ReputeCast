package visibility_test

import (
	"slices"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The preview contacts come from the count's samples: the first call
// counts once more to keep them, later calls read only those contacts, and
// a sample that left the audience is replaced from the audience.
func TestPreviewContactsFromTheCount(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	id := one[string](t, env, `SELECT id::text FROM campaigns WHERE name = 'Weekly Digest #143' AND workspace_id = (SELECT id FROM workspaces WHERE slug = 'acme-newsletter')`)

	type contact struct {
		ID      string
		Version int
	}
	list := func() []contact {
		t.Helper()
		var who struct{ Items []contact }
		linh.Do("GET", ws+"/campaigns/"+id+"/visibility/preview-contacts", nil).MustStatus(t, 200).JSON(t, &who)
		return who.Items
	}
	// One contact per version, in version order, then one per other
	// language (with the version they get).
	first := list()
	byVersion := map[int]string{}
	for i, c := range first {
		if _, seen := byVersion[c.Version]; !seen && c.Version == i+1 {
			byVersion[c.Version] = c.ID
		}
	}
	if byVersion[1] == "" || len(byVersion) < 2 {
		t.Fatalf("a contact for each version: %+v", first)
	}
	if n := one[int](t, env, `SELECT count(*) FROM visibility_counts, jsonb_object_keys(samples->'versions') WHERE campaign_id = $1`, id); n < len(byVersion) {
		t.Fatalf("the count kept %d version samples for %d versions", n, len(byVersion))
	}
	if again := list(); !slices.Equal(again, first) {
		t.Fatalf("the samples give the same contacts: %+v, then %+v", first, again)
	}

	// Version 1's contact unsubscribes from #143's list.
	if _, err := env.App.Pool.Exec(t.Context(), `UPDATE list_memberships SET status = 'unsubscribed' WHERE contact_id = $1`, byVersion[1]); err != nil {
		t.Fatal(err)
	}
	for _, c := range list() {
		if c.ID == byVersion[1] {
			t.Fatalf("a contact who left the audience is still previewed as")
		}
		if c.Version == 1 {
			return
		}
	}
	t.Fatalf("version 1 has no contact any more")
}
