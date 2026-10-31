package campaigns_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// CO-10: comments with @mentions, review links, presence and merges.

type thread struct {
	ID          string
	Anchor      string
	AnchorLabel string
	ResolvedAt  *time.Time
	ResolvedBy  string
	Comments    []struct {
		ID       string
		Body     string
		Author   string
		AuthorID string
		Reviewer bool
		Version  int
		Mentions []string
	}
}

type threadList struct {
	Threads  []thread
	Open     int
	Resolved int
}

func memberID(t *testing.T, c *apptest.Client, campaign, name string) string {
	t.Helper()
	var members struct {
		Items []struct{ ID, Name, Role string }
	}
	c.Do("GET", ws+"/campaigns/"+campaign+"/mentionable", nil).MustStatus(t, 200).JSON(t, &members)
	for _, m := range members.Items {
		if m.Name == name {
			return m.ID
		}
	}
	t.Fatalf("%s isn't mentionable: %+v", name, members.Items)
	return ""
}

func TestComposerCommentsAndMentions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var d detail
	linh.Do("POST", ws+"/campaigns", map[string]any{"name": "Comments"}).MustStatus(t, 201).JSON(t, &d)
	linh.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 1, "subject": "Hello", "mjml": fmt.Sprintf(bodyOf, "Hero")}).MustStatus(t, 200)
	marcus := memberID(t, linh, d.ID, "Marcus Chen")

	// A thread on a block, mentioning Marcus.
	var th thread
	linh.Do("POST", ws+"/campaigns/"+d.ID+"/comments", map[string]any{"body": "  @Marcus Chen can we link the changelog instead?  ", "anchor": "c7k2",
		"anchorLabel": "the button “Read the docs”", "mentions": []string{marcus}}).MustStatus(t, 201).JSON(t, &th)
	if th.Anchor != "c7k2" || th.AnchorLabel != "the button “Read the docs”" || len(th.Comments) != 1 || th.Comments[0].Author != "Linh Nguyen" ||
		th.Comments[0].Body != "@Marcus Chen can we link the changelog instead?" || th.Comments[0].Version != 2 || len(th.Comments[0].Mentions) != 1 {
		t.Fatalf("thread: %+v", th)
	}
	if auditCount(t, env, "campaign.comment", d.ID) != 1 {
		t.Fatal("comment not audited")
	}

	// Marcus is notified, with a link to the thread.
	m := env.Client()
	m.SignInDemo("marcus@acme.io")
	var notes struct {
		Items []struct{ Kind, Title, Body, Link, Category string }
	}
	m.Do("GET", ws+"/notifications", nil).MustStatus(t, 200).JSON(t, &notes)
	found := false
	for _, n := range notes.Items {
		if n.Kind == "comment_mention" {
			found = true
			if n.Title != "Linh Nguyen mentioned you" || n.Category != "team" || !strings.Contains(n.Body, "“Comments”") ||
				n.Link != "/w/acme-newsletter/campaigns/"+d.ID+"/edit?panel=comments&thread="+th.ID {
				t.Fatalf("notification: %+v", n)
			}
		}
	}
	if !found {
		t.Fatalf("no mention notification: %+v", notes.Items)
	}

	// He replies; the thread keeps its block.
	var reply thread
	m.Do("POST", ws+"/campaigns/"+d.ID+"/comments", map[string]any{"body": "Done.", "threadId": th.ID, "anchor": "zzzz"}).MustStatus(t, 201).JSON(t, &reply)
	if reply.ID != th.ID || reply.Anchor != "c7k2" || len(reply.Comments) != 2 || reply.Comments[1].Author != "Marcus Chen" {
		t.Fatalf("reply: %+v", reply)
	}
	// Only people in the workspace can be mentioned; blocks need a marker.
	linh.Do("POST", ws+"/campaigns/"+d.ID+"/comments", map[string]any{"body": "x", "mentions": []string{"0190a7c6-0000-7000-8000-000000000000"}}).MustStatus(t, 422)
	linh.Do("POST", ws+"/campaigns/"+d.ID+"/comments", map[string]any{"body": "x", "anchor": "Not An Id"}).MustStatus(t, 422)
	linh.Do("POST", ws+"/campaigns/"+d.ID+"/comments", map[string]any{"body": "   "}).MustStatus(t, 422)
	// A reply to a reply isn't a thread.
	linh.Do("POST", ws+"/campaigns/"+d.ID+"/comments", map[string]any{"body": "x", "threadId": reply.Comments[1].ID}).MustStatus(t, 404)

	// Viewers comment, but only people who edit campaigns resolve.
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	aisha.Do("POST", ws+"/campaigns/"+d.ID+"/comments", map[string]any{"body": "Looks good to me."}).MustStatus(t, 201)
	aisha.Do("PUT", ws+"/campaigns/"+d.ID+"/comments/"+th.ID+"/resolved", map[string]any{"resolved": true}).MustStatus(t, 403)
	linh.Do("PUT", ws+"/campaigns/"+d.ID+"/comments/"+th.ID+"/resolved", map[string]any{"resolved": true}).MustStatus(t, 204)
	var list threadList
	linh.Do("GET", ws+"/campaigns/"+d.ID+"/comments", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Open != 1 || list.Resolved != 1 || len(list.Threads) != 2 || list.Threads[1].ID != th.ID || list.Threads[1].ResolvedBy != "Linh Nguyen" || list.Threads[0].Anchor != "" {
		t.Fatalf("after resolve: %+v", list)
	}
	if auditCount(t, env, "campaign.comment.resolve", d.ID) != 1 {
		t.Fatal("resolve not audited")
	}
	linh.Do("PUT", ws+"/campaigns/"+d.ID+"/comments/"+th.ID+"/resolved", map[string]any{"resolved": false}).MustStatus(t, 204)
	linh.Do("GET", ws+"/campaigns/"+d.ID+"/comments", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Open != 2 || list.Resolved != 0 {
		t.Fatalf("after reopen: %+v", list)
	}

	// Comments carry their author and resolver: an API key or agent would
	// be recorded as the person who made it, so only people comment.
	var key struct{ Secret string }
	linh.Do("POST", ws+"/api-keys", map[string]any{"name": "Comment bot", "permissions": []string{"campaigns:edit"}}).MustStatus(t, 201).JSON(t, &key)
	bot := env.Client()
	bot.Bearer = key.Secret
	if p := bot.Do("POST", ws+"/campaigns/"+d.ID+"/comments", map[string]any{"body": "From a key"}).MustStatus(t, 403).Problem(t); p["code"] != "humans_only" {
		t.Fatalf("an API key commented: %v", p)
	}
	if p := bot.Do("PUT", ws+"/campaigns/"+d.ID+"/comments/"+th.ID+"/resolved", map[string]any{"resolved": true}).MustStatus(t, 403).Problem(t); p["code"] != "humans_only" {
		t.Fatalf("an API key resolved a thread: %v", p)
	}
	bot.Do("GET", ws+"/campaigns/"+d.ID+"/comments", nil).MustStatus(t, 200)

	// Other workspaces don't see them.
	linh.Do("GET", "/workspaces/acme-engineering-blog/campaigns/"+d.ID+"/comments", nil).MustStatus(t, 404)
	linh.Do("POST", "/workspaces/acme-engineering-blog/campaigns/"+d.ID+"/comments", map[string]any{"body": "x"}).MustStatus(t, 404)

	// Problems read in the reader's language.
	linh.Header = http.Header{"Accept-Language": []string{"vi"}}
	if p := linh.Do("POST", ws+"/campaigns/"+d.ID+"/comments", map[string]any{"body": "   "}).MustStatus(t, 422).Problem(t); !strings.Contains(fmt.Sprint(p), "Hãy viết nhận xét.") {
		t.Fatalf("vi problem: %v", p)
	}
}

type reviewLink struct {
	ID        string
	URL       string
	Version   int
	CreatedBy string
	ExpiresAt time.Time
}

func tokenOf(t *testing.T, l reviewLink) string {
	t.Helper()
	i := strings.LastIndex(l.URL, "/review/")
	if i < 0 {
		t.Fatalf("review URL: %s", l.URL)
	}
	return l.URL[i+len("/review/"):]
}

func TestComposerReviewLinks(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var d detail
	linh.Do("POST", ws+"/campaigns", map[string]any{"name": "For review"}).MustStatus(t, 201).JSON(t, &d)
	linh.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 1, "subject": "Hi {{ first_name | default: \"there\" }}",
		"mjml": fmt.Sprintf(bodyOf, `Hello {{ first_name | default: "there" }}`)}).MustStatus(t, 200)

	var link reviewLink
	linh.Do("POST", ws+"/campaigns/"+d.ID+"/review-links", nil).MustStatus(t, 201).JSON(t, &link)
	token := tokenOf(t, link)
	if link.Version != 2 || link.CreatedBy != "Linh Nguyen" || len(token) < 40 || time.Until(link.ExpiresAt) < 6*24*time.Hour {
		t.Fatalf("link: %+v", link)
	}
	if auditCount(t, env, "campaign.review_link.create", d.ID) != 1 {
		t.Fatal("link not audited")
	}
	// The version reviewers see is named, so it stays in the history.
	var v2 struct{ Name string }
	linh.Do("GET", ws+"/campaigns/"+d.ID+"/versions/2", nil).MustStatus(t, 200).JSON(t, &v2)
	if v2.Name != "Shared for review" {
		t.Fatalf("version name: %+v", v2)
	}
	// A review link records who shared or revoked it: only people do.
	var key struct{ Secret string }
	linh.Do("POST", ws+"/api-keys", map[string]any{"name": "Review bot", "permissions": []string{"campaigns:edit"}}).MustStatus(t, 201).JSON(t, &key)
	bot := env.Client()
	bot.Bearer = key.Secret
	if p := bot.Do("POST", ws+"/campaigns/"+d.ID+"/review-links", nil).MustStatus(t, 403).Problem(t); p["code"] != "humans_only" {
		t.Fatalf("an API key shared a review link: %v", p)
	}
	if p := bot.Do("DELETE", ws+"/campaigns/"+d.ID+"/review-links/"+link.ID, nil).MustStatus(t, 403).Problem(t); p["code"] != "humans_only" {
		t.Fatalf("an API key revoked a review link: %v", p)
	}

	// Members can copy it again; the token isn't stored in the clear.
	var links struct{ Items []reviewLink }
	linh.Do("GET", ws+"/campaigns/"+d.ID+"/review-links", nil).MustStatus(t, 200).JSON(t, &links)
	if len(links.Items) != 1 || links.Items[0].URL != link.URL {
		t.Fatalf("links: %+v", links)
	}
	var stored int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM campaign_review_links WHERE position($1::bytea IN token_sealed) > 0 OR position($1::bytea IN token_hash) > 0`, []byte(token)).Scan(&stored); err != nil || stored != 0 {
		t.Fatalf("token stored in the clear: %d %v", stored, err)
	}

	// Later edits don't change what reviewers see.
	linh.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 2, "subject": "Changed", "mjml": fmt.Sprintf(bodyOf, "Changed")}).MustStatus(t, 200)

	// A reviewer, without signing in.
	r := env.Client()
	resp := r.Do("GET", "/review/"+token, nil).MustStatus(t, 200)
	if resp.Header.Get("Cache-Control") != "no-store" || resp.Header.Get("X-Robots-Tag") != "noindex, nofollow" {
		t.Fatalf("headers: %v", resp.Header)
	}
	var page struct {
		Workspace, Campaign, Subject, HTML, Sample string
		Version                                    int
		Threads                                    []thread
	}
	resp.JSON(t, &page)
	if page.Workspace != "Acme Newsletter" || page.Campaign != "For review" || page.Version != 2 || page.Subject != "Hi Sam" || !strings.Contains(page.HTML, "Hello Sam") ||
		strings.Contains(page.HTML, "Changed") || strings.Contains(page.HTML, "first_name") || page.Sample != "Sam Rivera" {
		t.Fatalf("page: %+v", page)
	}
	for _, k := range []string{"audience", "sender", "route"} {
		if strings.Contains(strings.ToLower(string(resp.Body)), `"`+k) {
			t.Fatalf("the page shows the %s: %s", k, resp.Body)
		}
	}

	// The reviewer comments, signed with a name.
	var rt thread
	r.Do("POST", "/review/"+token+"/comments", map[string]any{"name": "Ana from Legal", "body": "Please add the cut-off time in UTC."}).MustStatus(t, 201).JSON(t, &rt)
	if len(rt.Comments) != 1 || rt.Comments[0].Author != "Ana from Legal" || !rt.Comments[0].Reviewer || rt.Comments[0].Version != 2 {
		t.Fatalf("reviewer thread: %+v", rt)
	}
	r.Do("POST", "/review/"+token+"/comments", map[string]any{"name": " ", "body": "x"}).MustStatus(t, 422)
	// Members see it in the composer, and reply.
	var list threadList
	linh.Do("GET", ws+"/campaigns/"+d.ID+"/comments", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Open != 1 || !list.Threads[0].Comments[0].Reviewer || list.Threads[0].Comments[0].Author != "Ana from Legal" {
		t.Fatalf("members' view: %+v", list)
	}
	linh.Do("POST", ws+"/campaigns/"+d.ID+"/comments", map[string]any{"body": "Added, thanks.", "threadId": rt.ID}).MustStatus(t, 201)
	// A thread of the team's own stays with the team.
	var own thread
	linh.Do("POST", ws+"/campaigns/"+d.ID+"/comments", map[string]any{"body": "Internal: check with sales first."}).MustStatus(t, 201).JSON(t, &own)
	r.Do("GET", "/review/"+token, nil).MustStatus(t, 200).JSON(t, &page)
	if len(page.Threads) != 1 || len(page.Threads[0].Comments) != 2 || page.Threads[0].Comments[1].Author != "Linh Nguyen" || page.Threads[0].Comments[1].AuthorID != "" ||
		strings.Contains(fmt.Sprint(page.Threads), "Internal") {
		t.Fatalf("reviewer's threads: %+v", page.Threads)
	}
	r.Do("POST", "/review/"+token+"/comments", map[string]any{"name": "Ana", "body": "Me too", "threadId": own.ID}).MustStatus(t, 404)

	// At most 30 comments an hour through one link.
	if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO campaign_comments (id, workspace_id, campaign_id, body, reviewer_name, review_link_id, version, created_at)
		SELECT gen_random_uuid(), workspace_id, campaign_id, 'flood', 'Bot', id, version, now() FROM campaign_review_links, generate_series(1, 29) WHERE id = $1`, link.ID); err != nil {
		t.Fatal(err)
	}
	r.Do("POST", "/review/"+token+"/comments", map[string]any{"name": "Ana", "body": "One more"}).MustStatus(t, 429)

	// Revoked: gone for the reviewer, and listed no more.
	linh.Do("DELETE", ws+"/campaigns/"+d.ID+"/review-links/"+link.ID, nil).MustStatus(t, 204)
	if p := r.Do("GET", "/review/"+token, nil).MustStatus(t, 404).Problem(t); !strings.Contains(fmt.Sprint(p["detail"]), "expired or was revoked") {
		t.Fatalf("revoked: %v", p)
	}
	linh.Do("GET", ws+"/campaigns/"+d.ID+"/review-links", nil).MustStatus(t, 200).JSON(t, &links)
	if len(links.Items) != 0 || auditCount(t, env, "campaign.review_link.revoke", d.ID) != 1 {
		t.Fatalf("after revoke: %+v", links)
	}
	linh.Do("DELETE", ws+"/campaigns/"+d.ID+"/review-links/"+link.ID, nil).MustStatus(t, 404)

	// Expired links stop working too.
	var second reviewLink
	linh.Do("POST", ws+"/campaigns/"+d.ID+"/review-links", nil).MustStatus(t, 201).JSON(t, &second)
	if second.Version != 3 {
		t.Fatalf("second link: %+v", second)
	}
	r.Do("GET", "/review/"+tokenOf(t, second), nil).MustStatus(t, 200)
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE campaign_review_links SET expires_at = now() - interval '1 minute' WHERE id = $1`, second.ID); err != nil {
		t.Fatal(err)
	}
	r.Do("GET", "/review/"+tokenOf(t, second), nil).MustStatus(t, 404)
	r.Do("GET", "/review/not-a-real-token-but-long-enough-to-look-like-one", nil).MustStatus(t, 404)

	// Only people who edit campaigns share them.
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	aisha.Do("POST", ws+"/campaigns/"+d.ID+"/review-links", nil).MustStatus(t, 403)
}

func TestComposerPresence(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var d detail
	linh.Do("POST", ws+"/campaigns", map[string]any{"name": "Presence"}).MustStatus(t, 201).JSON(t, &d)
	type presence struct {
		Others []struct {
			Name    string
			Editing bool
		}
	}
	var p presence
	linh.Do("PUT", ws+"/campaigns/"+d.ID+"/presence", map[string]any{"editing": true}).MustStatus(t, 200).JSON(t, &p)
	if len(p.Others) != 0 {
		t.Fatalf("alone: %+v", p)
	}
	marcus := env.Client()
	marcus.SignInDemo("marcus@acme.io")
	marcus.Do("PUT", ws+"/campaigns/"+d.ID+"/presence", map[string]any{"editing": false}).MustStatus(t, 200).JSON(t, &p)
	if len(p.Others) != 1 || p.Others[0].Name != "Linh Nguyen" || !p.Others[0].Editing {
		t.Fatalf("Marcus sees: %+v", p)
	}
	linh.Do("PUT", ws+"/campaigns/"+d.ID+"/presence", map[string]any{"editing": false}).MustStatus(t, 200).JSON(t, &p)
	if len(p.Others) != 1 || p.Others[0].Name != "Marcus Chen" || p.Others[0].Editing {
		t.Fatalf("Linh sees: %+v", p)
	}
	// Leaving, or a minute without a word, and you're gone.
	linh.Do("DELETE", ws+"/campaigns/"+d.ID+"/presence", nil).MustStatus(t, 204)
	marcus.Do("PUT", ws+"/campaigns/"+d.ID+"/presence", map[string]any{}).MustStatus(t, 200).JSON(t, &p)
	if len(p.Others) != 0 {
		t.Fatalf("after leaving: %+v", p)
	}
	linh.Do("PUT", ws+"/campaigns/"+d.ID+"/presence", map[string]any{}).MustStatus(t, 200)
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE campaign_presence SET seen_at = now() - interval '2 minutes' WHERE campaign_id = $1`, d.ID); err != nil {
		t.Fatal(err)
	}
	marcus.Do("PUT", ws+"/campaigns/"+d.ID+"/presence", map[string]any{}).MustStatus(t, 200).JSON(t, &p)
	if len(p.Others) != 0 {
		t.Fatalf("stale presence: %+v", p)
	}

	// An API key isn't someone with the campaign open: it sees who is, but
	// neither shows the person who made it there nor takes them away.
	var key struct{ Secret string }
	linh.Do("POST", ws+"/api-keys", map[string]any{"name": "Presence bot", "permissions": []string{"campaigns:view"}}).MustStatus(t, 201).JSON(t, &key)
	bot := env.Client()
	bot.Bearer = key.Secret
	bot.Do("PUT", ws+"/campaigns/"+d.ID+"/presence", map[string]any{"editing": true}).MustStatus(t, 200).JSON(t, &p)
	if len(p.Others) != 1 || p.Others[0].Name != "Marcus Chen" {
		t.Fatalf("the key sees: %+v", p)
	}
	marcus.Do("PUT", ws+"/campaigns/"+d.ID+"/presence", map[string]any{}).MustStatus(t, 200).JSON(t, &p)
	if len(p.Others) != 0 {
		t.Fatalf("an API key's call shows the person who made it: %+v", p)
	}
	linh.Do("PUT", ws+"/campaigns/"+d.ID+"/presence", map[string]any{}).MustStatus(t, 200)
	bot.Do("DELETE", ws+"/campaigns/"+d.ID+"/presence", nil).MustStatus(t, 204)
	marcus.Do("PUT", ws+"/campaigns/"+d.ID+"/presence", map[string]any{}).MustStatus(t, 200).JSON(t, &p)
	if len(p.Others) != 1 || p.Others[0].Name != "Linh Nguyen" {
		t.Fatalf("an API key's leave took the person who made it away: %+v", p)
	}
}

// Two people changed different sections of the same version: both kept.
func TestComposerMergesDifferentSections(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var d detail
	linh.Do("POST", ws+"/campaigns", map[string]any{"name": "Merge"}).MustStatus(t, 201).JSON(t, &d)
	base := fmt.Sprintf(bodyOf, "Hero")
	linh.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 1, "subject": "Hello", "preheader": "Pre", "mjml": base}).MustStatus(t, 200)

	// Marcus changes the footer and the subject (v3)...
	marcus := env.Client()
	marcus.SignInDemo("marcus@acme.io")
	theirs := strings.Replace(base, ">Footer<", ">New footer<", 1)
	marcus.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 2, "subject": "Their subject", "preheader": "Pre", "mjml": theirs}).MustStatus(t, 200)
	// ...while Linh, still on v2, changes the hero.
	mine := strings.Replace(base, ">Hero<", ">My hero<", 1)
	var saved struct {
		Version int
		HTML    string
		Merged  *struct {
			With     string
			Version  int
			Sections []struct {
				Index int
				Label string
				MJML  string
			}
			Subject   *string
			Preheader *string
		}
	}
	linh.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 2, "subject": "Hello", "preheader": "Pre", "mjml": mine, "html": "<p>stale</p>"}).MustStatus(t, 200).JSON(t, &saved)
	if saved.Version != 4 || saved.Merged == nil || saved.Merged.With != "Marcus Chen" || saved.Merged.Version != 3 || len(saved.Merged.Sections) != 1 ||
		saved.Merged.Sections[0].Index != 1 || !strings.Contains(saved.Merged.Sections[0].MJML, "New footer") || saved.Merged.Subject == nil || *saved.Merged.Subject != "Their subject" ||
		saved.Merged.Preheader != nil {
		t.Fatalf("merged save: %+v", saved.Merged)
	}
	linh.Do("GET", ws+"/campaigns/"+d.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Subject != "Their subject" || !strings.Contains(d.MJML, "My hero") || !strings.Contains(d.MJML, "New footer") || !strings.Contains(d.HTML, "My hero") ||
		!strings.Contains(d.HTML, "New footer") || strings.Contains(d.HTML, "stale") {
		t.Fatalf("stored: %+v", d)
	}
	var list versionList
	linh.Do("GET", ws+"/campaigns/"+d.ID+"/versions", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Items[0].Label != "merged with Marcus Chen's v3" || list.Items[0].Source != "merge" {
		t.Fatalf("history: %+v", list.Items[0])
	}

	// The same section on both sides still asks.
	clash := strings.Replace(base, ">Hero<", ">Another hero<", 1)
	if p := marcus.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 3, "subject": "Their subject", "preheader": "Pre", "mjml": clash}).MustStatus(t, 409).Problem(t); p["code"] != "version_conflict" {
		t.Fatalf("clash: %v", p)
	}
}
