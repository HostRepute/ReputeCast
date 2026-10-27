package campaigns_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

type versionList struct {
	Items []struct {
		Version int
		Label   string
		Source  string
		Name    string
		Current bool
	}
	Total      int
	Named      int
	NextBefore *int
}

type savedContent struct {
	Version        int
	Unchanged      bool
	StashedVersion int
}

const bodyOf = `<mjml><mj-body><mj-section><mj-column><mj-text>%s</mj-text></mj-column></mj-section><mj-section><mj-column><mj-text>Footer</mj-text></mj-column></mj-section></mj-body></mjml>`

// CO-2: "Load theirs" keeps your edits in the history as a version of
// their own, then puts theirs back on top; the content doesn't change.
func TestComposerLoadTheirsKeepsYours(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var d detail
	c.Do("POST", ws+"/campaigns", map[string]any{"name": "Conflict"}).MustStatus(t, 201).JSON(t, &d)
	var s savedContent
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 1, "subject": "Theirs", "mjml": fmt.Sprintf(bodyOf, "Their hero")}).MustStatus(t, 200).JSON(t, &s)
	if s.Version != 2 {
		t.Fatalf("their save: %+v", s)
	}
	// Mine started from v1: a conflict.
	mine := map[string]any{"baseVersion": 1, "subject": "Mine", "mjml": fmt.Sprintf(bodyOf, "My hero")}
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", mine).MustStatus(t, 409)
	// Load theirs: mine becomes v3 in the history, theirs v4 on top.
	mine["stash"] = true
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", mine).MustStatus(t, 200).JSON(t, &s)
	if s.Version != 4 || s.StashedVersion != 3 {
		t.Fatalf("stash: %+v", s)
	}
	c.Do("GET", ws+"/campaigns/"+d.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Subject != "Theirs" || !strings.Contains(d.MJML, "Their hero") || d.Version != 4 {
		t.Fatalf("after load theirs: %+v", d)
	}
	var v3 struct {
		Subject string
		MJML    string
		Source  string
		Label   string
	}
	c.Do("GET", ws+"/campaigns/"+d.ID+"/versions/3", nil).MustStatus(t, 200).JSON(t, &v3)
	if v3.Subject != "Mine" || !strings.Contains(v3.MJML, "My hero") || v3.Source != "conflict" || v3.Label != "your edits, kept when Linh Nguyen's v2 was loaded" {
		t.Fatalf("kept version: %+v", v3)
	}
	var list versionList
	c.Do("GET", ws+"/campaigns/"+d.ID+"/versions", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Total != 4 || !list.Items[0].Current || list.Items[0].Label != "loaded Linh Nguyen's v2" {
		t.Fatalf("history: %+v", list)
	}
	if auditCount(t, env, "campaign.content.stash", d.ID) != 1 {
		t.Fatal("stash not audited")
	}
	// The history reads in the reader's language.
	c.Header = http.Header{"Accept-Language": []string{"vi"}}
	c.Do("GET", ws+"/campaigns/"+d.ID+"/versions", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Items[0].Label != "đã mở v2 của Linh Nguyen" {
		t.Fatalf("vietnamese label: %q", list.Items[0].Label)
	}
}

// CO-2: saving what is already saved stores nothing and never conflicts,
// so a slow save and a retry of the same content don't fight.
func TestComposerSaveUnchanged(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var d detail
	c.Do("POST", ws+"/campaigns", map[string]any{"name": "Same"}).MustStatus(t, 201).JSON(t, &d)
	body := map[string]any{"baseVersion": 1, "subject": "Hello", "mjml": fmt.Sprintf(bodyOf, "Hi")}
	var s savedContent
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", body).MustStatus(t, 200).JSON(t, &s)
	if s.Version != 2 || s.Unchanged {
		t.Fatalf("first save: %+v", s)
	}
	// Same content again, still based on v1: no conflict, no new version.
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", body).MustStatus(t, 200).JSON(t, &s)
	if s.Version != 2 || !s.Unchanged {
		t.Fatalf("same content: %+v", s)
	}
	if n := auditCount(t, env, "campaign.content.save", d.ID); n != 1 {
		t.Fatalf("an unchanged save must not be audited: %d", n)
	}
}

// CO-2: versions can be named, listed on their own and paged; named
// versions survive pruning.
func TestComposerNamedVersionsAndPages(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var d detail
	c.Do("POST", ws+"/campaigns", map[string]any{"name": "History"}).MustStatus(t, 201).JSON(t, &d)
	for i := 1; i <= 5; i++ {
		c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": i, "subject": fmt.Sprintf("S%d", i), "mjml": fmt.Sprintf(bodyOf, fmt.Sprintf("Text %d", i))}).MustStatus(t, 200)
	}
	var named struct {
		Version int
		Name    string
	}
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/versions/3/name", map[string]any{"name": "  Sent to   legal "}).MustStatus(t, 200).JSON(t, &named)
	if named.Version != 3 || named.Name != "Sent to legal" {
		t.Fatalf("named: %+v", named)
	}
	if auditCount(t, env, "campaign.version.name", d.ID) != 1 {
		t.Fatal("naming not audited")
	}
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/versions/99/name", map[string]any{"name": "x"}).MustStatus(t, 404)

	// Pages of two, newest first.
	var page versionList
	c.Do("GET", ws+"/campaigns/"+d.ID+"/versions?limit=2", nil).MustStatus(t, 200).JSON(t, &page)
	if len(page.Items) != 2 || page.Items[0].Version != 6 || page.NextBefore == nil || *page.NextBefore != 5 || page.Total != 6 || page.Named != 1 {
		t.Fatalf("first page: %+v", page)
	}
	c.Do("GET", fmt.Sprintf("%s/campaigns/%s/versions?limit=2&before=%d", ws, d.ID, *page.NextBefore), nil).MustStatus(t, 200).JSON(t, &page)
	if len(page.Items) != 2 || page.Items[0].Version != 4 || page.Items[1].Version != 3 || page.Items[1].Name != "Sent to legal" {
		t.Fatalf("second page: %+v", page)
	}
	before := *page.NextBefore
	page = versionList{}
	c.Do("GET", fmt.Sprintf("%s/campaigns/%s/versions?limit=2&before=%d", ws, d.ID, before), nil).MustStatus(t, 200).JSON(t, &page)
	if len(page.Items) != 2 || page.NextBefore != nil {
		t.Fatalf("last page: %+v", page)
	}
	c.Do("GET", ws+"/campaigns/"+d.ID+"/versions?named=true", nil).MustStatus(t, 200).JSON(t, &page)
	if len(page.Items) != 1 || page.Items[0].Version != 3 {
		t.Fatalf("named only: %+v", page)
	}

	// Pruning keeps the first version and named ones.
	if _, err := env.App.Pool.Exec(context.Background(), `DELETE FROM campaign_versions v WHERE v.campaign_id = $1 AND v.source = 'autosave' AND v.version <> 1 AND v.name = '' AND v.version <= 6 - 1`, d.ID); err != nil {
		t.Fatal(err)
	}
	c.Do("GET", ws+"/campaigns/"+d.ID+"/versions", nil).MustStatus(t, 200).JSON(t, &page)
	got := []int{}
	for _, it := range page.Items {
		got = append(got, it.Version)
	}
	if fmt.Sprint(got) != "[6 3 1]" {
		t.Fatalf("after pruning: %v", got)
	}

	// A Viewer reads the history but can't name versions.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", ws+"/campaigns/"+d.ID+"/versions?named=true", nil).MustStatus(t, 200)
	r := v.Do("PUT", ws+"/campaigns/"+d.ID+"/versions/3/name", map[string]any{"name": "x"})
	r.MustStatus(t, 403)
	if r.Problem(t)["permission"] != "campaigns:edit" {
		t.Fatalf("viewer: %s", r.Body)
	}
}

// CO-2: a restore made right after saving unsaved edits is based on the
// version that save made, so it never fails with a conflict.
func TestComposerRestoreAfterSavingEdits(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var d detail
	c.Do("POST", ws+"/campaigns", map[string]any{"name": "Restore"}).MustStatus(t, 201).JSON(t, &d)
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 1, "subject": "One", "mjml": fmt.Sprintf(bodyOf, "One")}).MustStatus(t, 200)
	// The unsaved edits are saved first (v3), then v2 is restored on top of v3.
	var s savedContent
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 2, "subject": "Two", "mjml": fmt.Sprintf(bodyOf, "Two"), "source": "manual", "label": "edits before restoring v2"}).MustStatus(t, 200).JSON(t, &s)
	c.Do("POST", fmt.Sprintf("%s/campaigns/%s/versions/2/restore", ws, d.ID), map[string]any{"baseVersion": s.Version}).MustStatus(t, 200).JSON(t, &s)
	c.Do("GET", ws+"/campaigns/"+d.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Subject != "One" || s.Version != 4 {
		t.Fatalf("restored: %+v %+v", d, s)
	}
	// Restoring what is already current stores nothing.
	c.Do("POST", fmt.Sprintf("%s/campaigns/%s/versions/2/restore", ws, d.ID), map[string]any{"baseVersion": 4}).MustStatus(t, 200).JSON(t, &s)
	if !s.Unchanged || s.Version != 4 {
		t.Fatalf("restore of the current content: %+v", s)
	}
}
