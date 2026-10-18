package website_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/website"
)

const ws = "/workspaces/acme-newsletter"

// demoSite is the demo with its website off (turned off when the demo
// seeds it on), and Linh signed in.
func demoSite(t *testing.T) (*apptest.Env, *apptest.Client) {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var ov website.WebsiteOverview
	c.Do("GET", ws+"/website", nil).MustStatus(t, 200).JSON(t, &ov)
	if ov.Status == website.StatusOn {
		c.Do("POST", ws+"/website/turn-off", map[string]any{"confirm": ov.Address.Host}).MustStatus(t, 200)
	}
	return env, c
}

// turnOn turns the website on and runs the background job inline.
func turnOn(t *testing.T, env *apptest.Env, c *apptest.Client) website.WebsiteOverview {
	t.Helper()
	var ov website.WebsiteOverview
	c.Do("POST", ws+"/website/turn-on", map[string]any{"layout": "featured", "signinOn": true}).MustStatus(t, 200).JSON(t, &ov)
	if ov.Status != website.StatusTurningOn && ov.Status != website.StatusOn {
		t.Fatalf("turning on: %s", ov.Status)
	}
	if err := env.App.Website().RunTurnOn(context.Background(), workspaceID(t, env)); err != nil {
		t.Fatalf("run turn on: %v", err)
	}
	c.Do("GET", ws+"/website", nil).MustStatus(t, 200).JSON(t, &ov)
	if ov.Status != website.StatusOn {
		t.Fatalf("after the job: %s", ov.Status)
	}
	return ov
}

func workspaceID(t *testing.T, env *apptest.Env) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func campaignID(t *testing.T, env *apptest.Env, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT c.id FROM campaigns c JOIN workspaces w ON w.id = c.workspace_id
		WHERE w.slug = 'acme-newsletter' AND c.name = $1 AND c.deleted_at IS NULL`, name).Scan(&id); err != nil {
		t.Fatalf("campaign %q: %v", name, err)
	}
	return id
}

// Turning the website on makes every archived campaign a post at the
// archive's address, each archived list a tag, and redirects the archive's
// addresses; turning it off needs the address typed and keeps everything.
func TestWebsiteTurnOnOff(t *testing.T) {
	env, c := demoSite(t)
	var off website.WebsiteOverview
	c.Do("GET", ws+"/website", nil).MustStatus(t, 200).JSON(t, &off)
	if off.Status != website.StatusOff || off.Archive.Issues == 0 || len(off.Archive.Tags) == 0 {
		t.Fatalf("off: %+v", off.Archive)
	}
	// Only website:manage turns it on.
	sofia := env.Client()
	sofia.SignInDemo("daniel@acme.io")
	sofia.Do("POST", ws+"/website/turn-on", map[string]any{"layout": "list", "signinOn": true}).MustStatus(t, 403)

	ov := turnOn(t, env, c)
	if ov.Posts.Published < off.Archive.Issues || ov.Progress == nil || ov.Progress.Posts.Done != ov.Progress.Posts.Total {
		t.Fatalf("on: %+v progress %+v (archive had %d)", ov.Posts, ov.Progress, off.Archive.Issues)
	}
	var redirects int64
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM website_redirects WHERE path LIKE 'archive/%'`).Scan(&redirects); err != nil {
		t.Fatal(err)
	}
	if redirects < off.Archive.Issues {
		t.Fatalf("%d archive redirects for %d issues", redirects, off.Archive.Issues)
	}
	var tags []website.WebsiteTag
	c.Do("GET", ws+"/website/tags", nil).MustStatus(t, 200).JSON(t, &tags)
	if len(tags) < len(off.Archive.Tags) {
		t.Fatalf("tags %+v, archive's %v", tags, off.Archive.Tags)
	}
	// Turning on twice is a conflict; running the job again changes nothing.
	c.Do("POST", ws+"/website/turn-on", map[string]any{"layout": "list", "signinOn": true}).MustStatus(t, 409)
	if err := env.App.Website().RunTurnOn(context.Background(), workspaceID(t, env)); err != nil {
		t.Fatal(err)
	}
	// Off: the address must be typed.
	r := c.Do("POST", ws+"/website/turn-off", map[string]any{"confirm": "wrong.example"})
	if r.Status != 422 || !strings.Contains(string(r.Body), ov.Address.Host) {
		t.Fatalf("wrong confirmation: %d %s", r.Status, r.Body)
	}
	c.Do("POST", ws+"/website/turn-off", map[string]any{"confirm": strings.ToUpper(ov.Address.Host)}).MustStatus(t, 200).JSON(t, &off)
	if off.Status != website.StatusOff || off.Posts.Published != ov.Posts.Published {
		t.Fatalf("off again: %s, %d posts kept of %d", off.Status, off.Posts.Published, ov.Posts.Published)
	}
	var n int
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action IN ('website.turned_on','website.live','website.turned_off')`).Scan(&n)
	if n < 3 {
		t.Fatalf("audit entries: %d", n)
	}
}

// A web-only post: addresses (free, taken, renamed with a redirect),
// content saves with conflicts, publishing now with undo, scheduling,
// unpublishing to a redirect or 410.
func TestWebsitePostLifecycle(t *testing.T) {
	env, c := demoSite(t)
	var p website.WebsitePost
	c.Do("POST", ws+"/website/posts", map[string]any{"title": "Inside our Postgres migration: the runbook"}).MustStatus(t, 201).JSON(t, &p)
	if p.Kind != website.KindWeb || p.Status != website.PostDraft || p.Slug != "inside-our-postgres-migration-the-runbook" {
		t.Fatalf("new post: %+v", p.WebsitePostRow)
	}
	id := p.ID
	// Publishing while the website is off is a conflict.
	c.Do("POST", ws+"/website/posts/"+id+"/publish", map[string]any{"when": "now"}).MustStatus(t, 409)
	turnOn(t, env, c)

	settings := func(slug string) map[string]any {
		return map[string]any{"title": p.Title, "slug": slug, "excerpt": "How we moved 2.1 TB.", "coverUrl": "", "coverAlt": "", "tags": []string{},
			"visibility": "members", "tiers": []string{}, "neverFree": false, "featured": false, "webOn": true, "publishMode": "now",
			"seo": map[string]any{"title": "", "description": "", "socialImage": "", "canonicalUrl": "", "hideFromSearch": false}}
	}
	c.Do("PUT", ws+"/website/posts/"+id, settings("Not A Slug")).MustStatus(t, 422)
	// An address another post holds is taken.
	var taken string
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT slug FROM website_posts WHERE status = 'published' LIMIT 1`).Scan(&taken)
	if taken == "" {
		t.Fatal("no published post to collide with")
	}
	c.Do("PUT", ws+"/website/posts/"+id, settings(taken)).MustStatus(t, 409)
	c.Do("PUT", ws+"/website/posts/"+id, settings("postgres-runbook")).MustStatus(t, 200).JSON(t, &p)
	if p.Slug != "postgres-runbook" || p.Visibility != "members" {
		t.Fatalf("saved: %+v", p.WebsitePostRow)
	}
	// Content: a save from an older version is a conflict unless forced.
	mjml := `<mjml><mj-body><mj-section><mj-column><mj-text><p>One.</p></mj-text><mj-text><p>Two.</p></mj-text></mj-column></mj-section></mj-body></mjml>`
	var saved website.WebsitePostSaved
	c.Do("PUT", ws+"/website/posts/"+id+"/content", map[string]any{"mjml": mjml, "baseVersion": p.Version, "source": "manual"}).MustStatus(t, 200).JSON(t, &saved)
	c.Do("PUT", ws+"/website/posts/"+id+"/content", map[string]any{"mjml": mjml, "baseVersion": p.Version, "source": "manual"}).MustStatus(t, 409)
	c.Do("PUT", ws+"/website/posts/"+id+"/content", map[string]any{"mjml": mjml, "baseVersion": p.Version, "source": "manual", "force": true}).MustStatus(t, 200)

	// Publish now, then Undo within seconds.
	c.Do("POST", ws+"/website/posts/"+id+"/publish", map[string]any{"when": "now"}).MustStatus(t, 200).JSON(t, &p)
	if p.Status != website.PostPublished || p.PublishedAt == nil {
		t.Fatalf("published: %+v", p.WebsitePostRow)
	}
	c.Do("POST", ws+"/website/posts/"+id+"/publish", map[string]any{"when": "now"}).MustStatus(t, 409)
	c.Do("POST", ws+"/website/posts/"+id+"/undo-publish", nil).MustStatus(t, 200).JSON(t, &p)
	if p.Status != website.PostDraft {
		t.Fatalf("undone: %s", p.Status)
	}
	// Schedule: a time in the past is refused; the minute job publishes it.
	c.Do("POST", ws+"/website/posts/"+id+"/publish", map[string]any{"when": "at", "at": time.Now().Add(-time.Hour)}).MustStatus(t, 422)
	at := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second)
	c.Do("POST", ws+"/website/posts/"+id+"/publish", map[string]any{"when": "at", "at": at, "tz": "America/New_York"}).MustStatus(t, 200).JSON(t, &p)
	if p.Status != website.PostScheduled || p.PublishAt == nil || !p.PublishAt.Equal(at) {
		t.Fatalf("scheduled: %+v", p.WebsitePostRow)
	}
	svc := env.App.Website()
	svc.Now = func() time.Time { return at.Add(time.Minute) }
	defer func() { svc.Now = nil }()
	n, err := svc.PublishDue(context.Background())
	if err != nil || n < 1 {
		t.Fatalf("publish due: %d, %v", n, err)
	}
	svc.Now = nil
	c.Do("GET", ws+"/website/posts/"+id, nil).MustStatus(t, 200).JSON(t, &p)
	if p.Status != website.PostPublished {
		t.Fatalf("after its time: %s", p.Status)
	}
	// Renaming a published post keeps the old address as a redirect.
	c.Do("PUT", ws+"/website/posts/"+id, settings("the-postgres-runbook")).MustStatus(t, 200)
	var redirect string
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT post_id::text FROM website_redirects WHERE path = 'posts/postgres-runbook'`).Scan(&redirect)
	if redirect != id {
		t.Fatalf("old address leads to %q, want %s", redirect, id)
	}
	// The old address stays taken.
	var other website.WebsitePost
	c.Do("POST", ws+"/website/posts", map[string]any{"title": "Another"}).MustStatus(t, 201).JSON(t, &other)
	c.Do("PUT", ws+"/website/posts/"+other.ID, map[string]any{"title": "Another", "slug": "postgres-runbook", "excerpt": "", "coverUrl": "", "coverAlt": "",
		"tags": []string{}, "visibility": "public", "tiers": []string{}, "neverFree": false, "featured": false, "webOn": true, "publishMode": "now",
		"seo": map[string]any{"title": "", "description": "", "socialImage": "", "canonicalUrl": "", "hideFromSearch": false}}).MustStatus(t, 409)
	// Unpublish: a redirect must be a path on the site, not itself.
	c.Do("POST", ws+"/website/posts/"+id+"/unpublish", map[string]any{"reason": "Out of date", "redirectTo": "https://evil.example"}).MustStatus(t, 422)
	c.Do("POST", ws+"/website/posts/"+id+"/unpublish", map[string]any{"reason": "Out of date", "redirectTo": "/posts/the-postgres-runbook"}).MustStatus(t, 422)
	c.Do("POST", ws+"/website/posts/"+id+"/unpublish", map[string]any{"reason": "Out of date"}).MustStatus(t, 200).JSON(t, &p)
	if p.Status != website.PostUnpublished || p.UnpublishReason != "Out of date" {
		t.Fatalf("unpublished: %+v", p.WebsitePostRow)
	}
	c.Do("PUT", ws+"/website/posts/"+id+"/redirect", map[string]any{"redirectTo": "/archive"}).MustStatus(t, 200).JSON(t, &p)
	if p.RedirectTo != "/archive" {
		t.Fatalf("redirect: %q", p.RedirectTo)
	}
	// The list counts it, and the audit log has every step.
	var list website.WebsitePostList
	c.Do("GET", ws+"/website/posts?status=unpublished", nil).MustStatus(t, 200).JSON(t, &list)
	found := false
	for _, r := range list.Posts {
		found = found || r.ID == id
	}
	if !found || list.Counts.Unpublished < 1 {
		t.Fatalf("unpublished list: %+v", list.Counts)
	}
	for _, action := range []string{"post.created", "post.slug_changed", "post.published", "post.publish_undone", "post.scheduled", "post.unpublished", "post.redirect_changed"} {
		var k int
		_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1 AND target_id = $2`, action, id).Scan(&k)
		if k == 0 {
			t.Errorf("no %s in the audit log", action)
		}
	}
	// Deleting it.
	c.Do("DELETE", ws+"/website/posts/"+other.ID, nil).MustStatus(t, 204)
	c.Do("GET", ws+"/website/posts/"+other.ID, nil).MustStatus(t, 404)
}

// A Content Writer can't publish: Publish becomes Request approval under
// "The author lacks the send permission"; once an approver approves, the
// post goes live by itself. A change after the request invalidates it.
func TestWebsitePostApproval(t *testing.T) {
	env, c := demoSite(t)
	turnOn(t, env, c)
	daniel := env.Client()
	daniel.SignInDemo("daniel@acme.io")
	var p website.WebsitePost
	daniel.Do("POST", ws+"/website/posts", map[string]any{"title": "Q4 roadmap deep dive"}).MustStatus(t, 201).JSON(t, &p)
	if p.Publish.CanPublish || !p.Publish.NeedsApproval || p.Publish.Rule != "author_lacks_send" || len(p.Publish.Approvers) == 0 {
		t.Fatalf("publish state: %+v", p.Publish)
	}
	daniel.Do("POST", ws+"/website/posts/"+p.ID+"/publish", map[string]any{"when": "now"}).MustStatus(t, 403)
	daniel.Do("POST", ws+"/website/posts/"+p.ID+"/approval", map[string]any{"note": "Ready for Thursday."}).MustStatus(t, 201).JSON(t, &p)
	if p.Publish.Pending == nil {
		t.Fatalf("pending request: %+v", p.Publish)
	}
	// The inbox lists the post.
	var inbox struct {
		Items []struct {
			ID, SubjectType, WebsitePostID, Name, Status string
		}
	}
	c.Do("GET", ws+"/approvals", nil).MustStatus(t, 200).JSON(t, &inbox)
	reqID := ""
	for _, it := range inbox.Items {
		if it.WebsitePostID == p.ID && it.SubjectType == "website_post" && it.Status == "pending" {
			reqID = it.ID
		}
	}
	if reqID == "" {
		t.Fatalf("inbox: %+v", inbox.Items)
	}
	// Daniel can't approve his own post; Linh approves it and it goes live.
	daniel.Do("POST", ws+"/approvals/"+reqID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 403)
	c.Do("POST", ws+"/approvals/"+reqID+"/decision", map[string]any{"decision": "approve", "comment": "Looks good"}).MustStatus(t, 200)
	daniel.Do("GET", ws+"/website/posts/"+p.ID, nil).MustStatus(t, 200).JSON(t, &p)
	if p.Status != website.PostPublished {
		t.Fatalf("after approval: %s", p.Status)
	}

	// Another post: a change after the request invalidates it.
	var q website.WebsitePost
	daniel.Do("POST", ws+"/website/posts", map[string]any{"title": "Changing after the request"}).MustStatus(t, 201).JSON(t, &q)
	daniel.Do("POST", ws+"/website/posts/"+q.ID+"/approval", map[string]any{}).MustStatus(t, 201)
	daniel.Do("PUT", ws+"/website/posts/"+q.ID+"/content", map[string]any{"mjml": `<mjml><mj-body><mj-section><mj-column><mj-text>Changed</mj-text></mj-column></mj-section></mj-body></mjml>`,
		"baseVersion": 0, "source": "manual"}).MustStatus(t, 200)
	var status string
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT status FROM approval_requests WHERE website_post_id = $1 ORDER BY requested_at DESC LIMIT 1`, q.ID).Scan(&status)
	if status != "invalidated" {
		t.Fatalf("request after a change: %s", status)
	}

	// With the rule "Publishing a post or page needs approval" on, Sofia
	// (who holds website:publish) needs approval too.
	var policy map[string]any
	c.Do("GET", ws+"/approval-policy", nil).MustStatus(t, 200).JSON(t, &policy)
	pol := policy["policy"].(map[string]any)
	pol["postPublish"] = true
	c.Do("PUT", ws+"/approval-policy", pol).MustStatus(t, 200)
	sofia := env.Client()
	sofia.SignInDemo("sofia@acme.io")
	var r website.WebsitePost
	sofia.Do("POST", ws+"/website/posts", map[string]any{"title": "Sofia's post"}).MustStatus(t, 201).JSON(t, &r)
	if !r.Publish.NeedsApproval || r.Publish.Rule != "post_publish" {
		t.Fatalf("with the rule on: %+v", r.Publish)
	}
	resp := sofia.Do("POST", ws+"/website/posts/"+r.ID+"/publish", map[string]any{"when": "now"})
	if resp.Status != 409 || !strings.Contains(string(resp.Body), "approval_required") {
		t.Fatalf("publish with the rule on: %d %s", resp.Status, resp.Body)
	}
}

// The composer's Web tab: defaults from the campaign (subject, preheader,
// first image, the lists' tags, on the web when its lists are archived),
// the first save makes the post; the Campaigns list's Web column.
func TestCampaignWebTab(t *testing.T) {
	env, c := demoSite(t)
	turnOn(t, env, c)
	draft := campaignID(t, env, "Flash sale: 48 hours only")
	var web website.WebsiteCampaignWeb
	c.Do("GET", ws+"/campaigns/"+draft.String()+"/web", nil).MustStatus(t, 200).JSON(t, &web)
	if !web.SiteOn || web.Post.ID != "" || web.Post.Title == "" || web.Post.Slug == "" {
		t.Fatalf("defaults: %+v", web.Post.WebsitePostRow)
	}
	in := map[string]any{"title": "", "slug": web.Post.Slug, "excerpt": "", "coverUrl": web.Post.CoverURL, "coverAlt": "A sale banner", "tags": []string{},
		"visibility": "public", "tiers": []string{}, "neverFree": false, "featured": false, "webOn": true, "publishMode": "with_send",
		"seo": map[string]any{"title": "", "description": "", "socialImage": "", "canonicalUrl": "", "hideFromSearch": false}}
	c.Do("PUT", ws+"/campaigns/"+draft.String()+"/web", in).MustStatus(t, 200).JSON(t, &web)
	if web.Post.ID == "" || !web.Post.WebOn || web.Chip != "with_send" {
		t.Fatalf("saved: chip %s %+v", web.Chip, web.Post.WebsitePostRow)
	}
	// The Web column.
	sent := campaignID(t, env, "October Product Update")
	var states website.WebsiteCampaignStates
	c.Do("GET", ws+"/campaigns-web?ids="+draft.String()+","+sent.String(), nil).MustStatus(t, 200).JSON(t, &states)
	got := map[string]string{}
	for _, s := range states.States {
		got[s.CampaignID] = s.Chip
	}
	if got[draft.String()] != "with_send" {
		t.Fatalf("web column: %+v", got)
	}
	// Send as a campaign: web-only posts only.
	c.Do("POST", ws+"/website/posts/"+web.Post.ID+"/send-as-campaign", nil).MustStatus(t, 409)
	var p website.WebsitePost
	c.Do("POST", ws+"/website/posts", map[string]any{"title": "A web-only post"}).MustStatus(t, 201).JSON(t, &p)
	var made website.WebsiteSentAsCampaign
	c.Do("POST", ws+"/website/posts/"+p.ID+"/send-as-campaign", nil).MustStatus(t, 201).JSON(t, &made)
	var name string
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT name FROM campaigns WHERE id = $1`, made.CampaignID).Scan(&name)
	if name != "A web-only post" {
		t.Fatalf("campaign made from the post: %q", name)
	}
	c.Do("POST", ws+"/website/posts/"+p.ID+"/send-as-campaign", nil).MustStatus(t, 409)
}

// The contact profile's Website card and "Sign out everywhere".
func TestContactWebsiteCard(t *testing.T) {
	env, c := demoSite(t)
	turnOn(t, env, c)
	ctx := context.Background()
	var contact uuid.UUID
	wsID := workspaceID(t, env)
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND erased_at IS NULL ORDER BY created_at LIMIT 1`, wsID).Scan(&contact); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO website_members (contact_id, workspace_id, first_signed_in, last_signed_in, last_method, sign_ins)
		VALUES ($1, $2, $3, $3, 'link', 1) ON CONFLICT (contact_id) DO NOTHING`, contact, wsID, now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for i := range 2 {
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO website_sessions (id, workspace_id, contact_id, token_hash, method, created_at, last_seen_at, expires_at)
			VALUES ($1, $2, $3, $4, 'link', $5, $5, $6)`, uuid.Must(uuid.NewV7()), wsID, contact, []byte(fmt.Sprintf("token-%d-%s", i, contact)), now, now.Add(90*24*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	var card website.WebsiteContactCard
	c.Do("GET", ws+"/contacts/"+contact.String()+"/website", nil).MustStatus(t, 200).JSON(t, &card)
	if !card.Member || card.Sessions < 2 || !card.CanSignOut {
		t.Fatalf("card: %+v", card)
	}
	// The contacts filter.
	var list struct{ Total int64 }
	c.Do("GET", ws+"/contacts?signedIn=true", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Total < 1 {
		t.Fatalf("signed in contacts: %d", list.Total)
	}
	// Viewers can't sign members out.
	viewer := env.Client()
	viewer.SignInDemo("daniel@acme.io")
	viewer.Do("POST", ws+"/contacts/"+contact.String()+"/website/sign-out", nil).MustStatus(t, 403)
	var out website.WebsiteSignedOut
	c.Do("POST", ws+"/contacts/"+contact.String()+"/website/sign-out", nil).MustStatus(t, 200).JSON(t, &out)
	if out.Sessions < 2 {
		t.Fatalf("ended %d sessions", out.Sessions)
	}
	c.Do("GET", ws+"/contacts/"+contact.String()+"/website", nil).MustStatus(t, 200).JSON(t, &card)
	if card.Sessions != 0 || len(card.Events) == 0 || card.Events[0].Kind != "signed_out_everywhere" || card.Events[0].Actor != "Linh Nguyen" {
		t.Fatalf("after signing out: %+v", card)
	}
	var k int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'website.member_signed_out' AND target_id = $1`, contact.String()).Scan(&k)
	if k != 1 {
		t.Fatalf("audit: %d", k)
	}
}
