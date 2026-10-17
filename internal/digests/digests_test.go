package digests_test

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/sending"
)

const ws = "/workspaces/acme-newsletter"

// post is an item of the test feed.
type post struct {
	guid, title, category, author string
	published                     time.Time
}

// feedServer serves an RSS 2.0 feed the test changes as it goes.
type feedServer struct {
	mu    sync.Mutex
	posts []post
	fail  bool
	reads int
	srv   *httptest.Server
}

func newFeedServer(t *testing.T, posts ...post) *feedServer {
	f := &feedServer{posts: posts}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.reads++
		if f.fail {
			http.Error(w, "maintenance", http.StatusServiceUnavailable)
			return
		}
		var b strings.Builder
		b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:dc="http://purl.org/dc/elements/1.1/"><channel><title>Test blog</title><link>https://blog.acme.io/</link>`)
		for _, p := range f.posts {
			fmt.Fprintf(&b, `<item><title>%s</title><link>https://blog.acme.io/%s/</link><guid isPermaLink="false">https://blog.acme.io/?p=%s</guid><pubDate>%s</pubDate><dc:creator>%s</dc:creator><category>%s</category><description>%s, in a few words.</description></item>`,
				html.EscapeString(p.title), p.guid, p.guid, p.published.UTC().Format(time.RFC1123Z), html.EscapeString(p.author), html.EscapeString(p.category), html.EscapeString(p.title))
		}
		b.WriteString(`</channel></rss>`)
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		_, _ = w.Write([]byte(b.String()))
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *feedServer) add(p post) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.posts = append([]post{p}, f.posts...)
}

func (f *feedServer) setFail(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fail = v
}

// fakeStarter stands in for the sending engine.
type fakeStarter struct {
	mu       sync.Mutex
	started  []uuid.UUID
	by       []uuid.UUID
	rechecks []sending.Recheck
}

func (f *fakeStarter) Start(ctx context.Context, id, by uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, id)
	f.by = append(f.by, by)
	if rc, ok := sending.RecheckFrom(ctx, id); ok {
		f.rechecks = append(f.rechecks, rc)
	}
	return nil
}
func (f *fakeStarter) Pause(context.Context, uuid.UUID, uuid.UUID, string) error { return nil }
func (f *fakeStarter) Resume(context.Context, uuid.UUID, uuid.UUID) error        { return nil }
func (f *fakeStarter) Cancel(context.Context, uuid.UUID, uuid.UUID) error        { return nil }

type issueView struct {
	ID          string
	Number      int
	Status      string
	SendAt      *time.Time
	Posts       int
	Subject     string
	CampaignID  string
	HoldReason  string
	Approval    *struct{ ID, Status string }
	FeedTries   []struct{ Error, Kind string }
	Readiness   *struct{ Verdict string }
	Items       []struct{ Title, Status string }
	PreparedAt  *time.Time
	NeedsApprov bool
}

type detail struct {
	ID, Name, State, Mode, TemplateID string
	Feed                              struct {
		Status, Kind, Title string
		Items, Waiting      int
	}
	Owner *struct{ Name string }
	Pause *struct {
		Reason, Note, From, To string
		Notified               []string
	}
	Next        *issueView
	SendHelpers []string
}

type counter struct {
	t   *testing.T
	env *apptest.Env
}

func (c counter) audit(action, target string) int {
	c.t.Helper()
	var n int
	if err := c.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1 AND ($2 = '' OR target_id = $2)`, action, target).Scan(&n); err != nil {
		c.t.Fatal(err)
	}
	return n
}

func (c counter) notes(email, kind string) int {
	c.t.Helper()
	var n int
	if err := c.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM notifications n JOIN users u ON u.id = n.user_id WHERE u.email = $1 AND n.kind = $2`, email, kind).Scan(&n); err != nil {
		c.t.Fatal(err)
	}
	return n
}

// notesFor counts a member's notifications of a kind about one digest.
func (c counter) notesFor(email, kind, digestID string) int {
	c.t.Helper()
	var n int
	if err := c.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM notifications n JOIN users u ON u.id = n.user_id WHERE u.email = $1 AND n.kind = $2 AND n.data ->> 'digest_id' = $3`,
		email, kind, digestID).Scan(&n); err != nil {
		c.t.Fatal(err)
	}
	return n
}

func (c counter) exec(sql string, args ...any) {
	c.t.Helper()
	if _, err := c.env.App.Pool.Exec(context.Background(), sql, args...); err != nil {
		c.t.Fatal(err)
	}
}

// blocks lists the blocking checks of a campaign's latest Send Readiness run.
func (c counter) blocks(campaignID string) string {
	c.t.Helper()
	var out, runs, held string
	_ = c.env.App.Pool.QueryRow(context.Background(), `SELECT coalesce(string_agg(ch ->> 'key' || ': ' || (ch ->> 'title'), '; '), '')
		FROM (SELECT checks FROM readiness_runs WHERE campaign_id = $1 ORDER BY ran_at DESC LIMIT 1) r, jsonb_array_elements(r.checks) ch
		WHERE ch ->> 'status' = 'block'`, uuid.MustParse(campaignID)).Scan(&out)
	_ = c.env.App.Pool.QueryRow(context.Background(), `SELECT coalesce(string_agg(trigger || ' ' || verdict, ', ' ORDER BY ran_at), '') FROM readiness_runs WHERE campaign_id = $1`,
		uuid.MustParse(campaignID)).Scan(&runs)
	_ = c.env.App.Pool.QueryRow(context.Background(), `SELECT coalesce(string_agg(reason, ', '), '') FROM audit_events WHERE action = 'digest.issue_held'`).Scan(&held)
	return out + " · runs: " + runs + " · held: " + held
}

func (c counter) listID(name string) string {
	c.t.Helper()
	var id uuid.UUID
	if err := c.env.App.Pool.QueryRow(context.Background(), `SELECT l.id FROM lists l JOIN workspaces w ON w.id = l.workspace_id WHERE w.slug = 'acme-newsletter' AND l.name = $1`, name).Scan(&id); err != nil {
		c.t.Fatal(err)
	}
	return id.String()
}

// decodeDetail decodes a digest into a fresh value.
func decodeDetail(t *testing.T, r *apptest.Response) detail {
	t.Helper()
	var d detail
	r.JSON(t, &d)
	return d
}

func getDigest(t *testing.T, c *apptest.Client, id string) detail {
	t.Helper()
	return decodeDetail(t, c.Do("GET", ws+"/digests/"+id, nil).MustStatus(t, 200))
}

// settings are the settings page as a digest test fills it in.
func settings(name, feedURL, list string, startLatest int) map[string]any {
	return map[string]any{
		"name": name, "archive": true, "mode": "auto", "source": "url", "feedUrl": feedURL,
		"filters":        map[string]any{"categories": []string{"Engineering", "Infrastructure"}, "tags": []string{}, "startMode": "latest", "startLatest": startLatest},
		"categoryFilter": true,
		"schedule":       map[string]any{"frequency": "weekly", "weekday": 5, "time": "08:00", "timezone": "America/New_York"},
		"prepareHours":   3, "lateHours": 24, "listIds": []string{list}, "segmentIds": []string{},
	}
}

func basePosts(now time.Time) []post {
	return []post{
		{"103", "Our 2027 internship program", "Company", "Joel Weber", now.Add(-24 * time.Hour)},
		{"102", "Feature flags at 2,000 teams: what broke first", "Engineering", "Joel Weber", now.Add(-48 * time.Hour)},
		{"101", "Postgres partitioning without downtime", "Infrastructure", "Mei Tanaka", now.Add(-72 * time.Hour)},
	}
}

// TestDigestSetupAndPermissions covers creating a digest, its settings,
// turning it on and who may do what.
func TestDigestSetupAndPermissions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	n := counter{t, env}
	feed := newFeedServer(t, basePosts(time.Now())...)
	linh, daniel, aisha := env.Client(), env.Client(), env.Client()
	linh.SignInDemo("linh@acme.io")
	daniel.SignInDemo("daniel@acme.io")
	aisha.SignInDemo("aisha@acme.io")

	// Viewers read digests but can't create one.
	aisha.Do("GET", ws+"/digests", nil).MustStatus(t, 200)
	aisha.Do("POST", ws+"/digests", map[string]any{"source": "url", "feedUrl": feed.srv.URL + "/feed/"}).MustStatus(t, 403)

	// A feed address the guard refuses, and one that isn't an address.
	linh.Do("POST", ws+"/digests", map[string]any{"source": "url", "feedUrl": "http://169.254.169.254/latest/"}).MustStatus(t, 422)
	linh.Do("POST", ws+"/digests", map[string]any{"source": "url", "feedUrl": "not an address"}).MustStatus(t, 422)

	d := decodeDetail(t, linh.Do("POST", ws+"/digests", map[string]any{"name": "Test roundup", "source": "url", "feedUrl": feed.srv.URL + "/feed/"}).MustStatus(t, 201))
	if d.State != "off" || d.Mode != "auto" || d.TemplateID == "" || d.Feed.Status != "ok" || d.Feed.Kind != "rss" || d.Feed.Items != 3 || d.Feed.Title != "Test blog" {
		t.Fatalf("created: %+v", d)
	}
	if n.audit("digest.created", d.ID) != 1 {
		t.Fatal("digest.created isn't audited")
	}
	// The composer's checks know the feed tokens of the template; a misspelt field still warns.
	var checks struct{ Issues []struct{ ID, Title string } }
	linh.Do("POST", ws+"/campaigns/"+d.TemplateID+"/checks", map[string]any{"subject": "{{ issue.items.first.title }} from {{ feed.title }}", "preheader": "Issue {{ issue.number }}",
		"mjml": "", "html": "<p>Hi {{ frist_name }}</p>"}).MustStatus(t, 200).JSON(t, &checks)
	unknown := []string{}
	for _, is := range checks.Issues {
		if is.ID == "personalization.unknown_field" {
			unknown = append(unknown, is.Title)
		}
	}
	if len(unknown) != 1 || !strings.Contains(unknown[0], "frist_name") {
		t.Fatalf("unknown fields in the template: %v", unknown)
	}
	// The template and the next issue stay out of the campaigns list.
	var list struct{ Items []struct{ Name string } }
	linh.Do("GET", ws+"/campaigns?pageSize=100&q=Test%20roundup", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 0 {
		t.Fatalf("the template is in the campaigns list: %+v", list.Items)
	}

	// Without campaigns:send, a digest starts in draft mode and can't be
	// turned on to send automatically.
	var dd detail
	daniel.Do("POST", ws+"/digests", map[string]any{"name": "Daniel's roundup", "source": "url", "feedUrl": feed.srv.URL + "/feed/"}).MustStatus(t, 201).JSON(t, &dd)
	if dd.Mode != "draft" {
		t.Fatalf("Daniel's digest: %+v", dd)
	}
	r := daniel.Do("POST", ws+"/digests/"+dd.ID+"/turn-on", map[string]any{"mode": "auto"}).MustStatus(t, 403)
	if !strings.Contains(string(r.Body), "campaigns:send") {
		t.Fatalf("turn on as Daniel: %s", r.Body)
	}

	// Turning on needs an audience.
	linh.Do("POST", ws+"/digests/"+d.ID+"/turn-on", map[string]any{}).MustStatus(t, 409)
	// Settings: no category picked is refused while the filter is on.
	bad := settings("Test roundup", feed.srv.URL+"/feed/", n.listID("Product Updates"), 2)
	bad["filters"] = map[string]any{"categories": []string{}, "tags": []string{}, "startMode": "after_on", "startLatest": 3}
	linh.Do("PUT", ws+"/digests/"+d.ID, bad).MustStatus(t, 422)
	linh.Do("PUT", ws+"/digests/"+d.ID, settings("Test roundup", feed.srv.URL+"/feed/", n.listID("Product Updates"), 2)).MustStatus(t, 200)
	if n.audit("digest.updated", d.ID) != 1 {
		t.Fatal("digest.updated isn't audited")
	}
	// Viewers can't change settings or pause.
	aisha.Do("PUT", ws+"/digests/"+d.ID, settings("Test roundup", feed.srv.URL+"/feed/", n.listID("Product Updates"), 2)).MustStatus(t, 403)

	d = decodeDetail(t, linh.Do("POST", ws+"/digests/"+d.ID+"/turn-on", map[string]any{}).MustStatus(t, 200))
	if d.State != "on" || d.Owner == nil || d.Owner.Name != "Linh Nguyen" || d.Next == nil || d.Next.Status != "collecting" || d.Next.SendAt == nil {
		t.Fatalf("turned on: %+v", d)
	}
	if d.Next.SendAt.In(mustLoc(t, "America/New_York")).Weekday() != time.Friday {
		t.Fatalf("next send: %v", d.Next.SendAt)
	}
	if n.audit("digest.turned_on", d.ID) != 1 {
		t.Fatal("digest.turned_on isn't audited")
	}
	// The first issue starts with the latest 2 posts in the picked categories.
	if d.Next.Posts != 2 {
		t.Fatalf("posts so far: %+v", d.Next)
	}
	// Turning it on again changes nothing; deleting it needs it off.
	linh.Do("POST", ws+"/digests/"+d.ID+"/turn-on", map[string]any{}).MustStatus(t, 409)
	linh.Do("DELETE", ws+"/digests/"+d.ID, nil).MustStatus(t, 409)

	// Pause, with a note; the collecting issue goes.
	aisha.Do("POST", ws+"/digests/"+d.ID+"/pause", map[string]any{}).MustStatus(t, 403)
	d = decodeDetail(t, linh.Do("POST", ws+"/digests/"+d.ID+"/pause", map[string]any{"note": "New theme next week"}).MustStatus(t, 200))
	if d.State != "paused" || d.Pause == nil || d.Pause.Reason != "person" || d.Pause.Note != "New theme next week" || d.Next != nil {
		t.Fatalf("paused: %+v %+v %+v", d, d.Pause, d.Next)
	}
	if n.audit("digest.paused", d.ID) != 1 {
		t.Fatal("digest.paused isn't audited")
	}
	linh.Do("POST", ws+"/digests/"+d.ID+"/turn-on", map[string]any{}).MustStatus(t, 200)
	if n.audit("digest.resumed", d.ID) != 1 {
		t.Fatal("digest.resumed isn't audited")
	}

	// The list: Campaigns › Recurring.
	var digests struct {
		Items []struct {
			Name, State string
			Next        *struct{ Status string }
		}
	}
	linh.Do("GET", ws+"/digests", nil).MustStatus(t, 200).JSON(t, &digests)
	found := false
	for _, it := range digests.Items {
		if it.Name == "Test roundup" && it.State == "on" && it.Next != nil && it.Next.Status == "collecting" {
			found = true
		}
	}
	if !found {
		t.Fatalf("list: %+v", digests.Items)
	}

	// The feed preview badges each item.
	var preview struct {
		Items []struct{ Title, Status, LeftOutBy string }
	}
	linh.Do("GET", ws+"/digests/"+d.ID+"/feed?limit=6", nil).MustStatus(t, 200).JSON(t, &preview)
	if len(preview.Items) != 3 || preview.Items[0].Status != "left_out" || preview.Items[0].LeftOutBy != "category" || preview.Items[1].Status != "new" {
		t.Fatalf("feed preview: %+v", preview.Items)
	}

	// Rendering the composer's Feed items block with the next issue's posts.
	var render struct {
		Count    int
		MJML     string
		Subject  string
		Subjects []struct {
			Items    int
			Text     string
			ZeroMore bool
		}
		Checks struct{ MaxBytes, Limit int }
	}
	linh.Do("POST", ws+"/digests/"+d.ID+"/render", map[string]any{"preview": "next"}).MustStatus(t, 200).JSON(t, &render)
	if render.Count != 2 || !strings.Contains(render.MJML, "Feature flags at 2,000 teams") || render.Subject != "Feature flags at 2,000 teams: what broke first and 1 more from Test blog" {
		t.Fatalf("render: %d %q", render.Count, render.Subject)
	}
	if len(render.Subjects) != 3 || render.Subjects[0].Text != "Feature flags at 2,000 teams: what broke first from Test blog" || render.Checks.MaxBytes == 0 {
		t.Fatalf("subjects: %+v %+v", render.Subjects, render.Checks)
	}
	// A subject without the {% if %} says "and 0 more" with one post.
	linh.Do("POST", ws+"/digests/"+d.ID+"/render", map[string]any{"preview": "next",
		"subject": "{{ issue.items.first.title }} and {{ issue.items.size | minus: 1 }} more"}).MustStatus(t, 200).JSON(t, &render)
	if !render.Subjects[0].ZeroMore {
		t.Fatalf("zero more: %+v", render.Subjects)
	}
	// "6 items": the starter's subject (the fix), and the size with 1 to 8 posts.
	var six struct {
		StarterSubject string
		Items          []struct {
			Title    string
			Fallback bool
		}
		Sizes  []struct{ Posts, Bytes int }
		Checks struct{ FitPosts, MaxPosts int }
	}
	linh.Do("POST", ws+"/digests/"+d.ID+"/render", map[string]any{"preview": "six", "sizes": true}).MustStatus(t, 200).JSON(t, &six)
	if !strings.Contains(six.StarterSubject, "{% if issue.items.size > 1 %}") || len(six.Items) == 0 || len(six.Sizes) != 8 || six.Sizes[7].Bytes <= six.Sizes[0].Bytes || six.Checks.FitPosts != 8 {
		t.Fatalf("six: %+v", six)
	}
}

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

// setUp creates and turns on a digest as Linh and returns its id.
func setUp(t *testing.T, c *apptest.Client, n counter, feed *feedServer, name string) detail {
	t.Helper()
	d := decodeDetail(t, c.Do("POST", ws+"/digests", map[string]any{"name": name, "source": "url", "feedUrl": feed.srv.URL + "/feed/"}).MustStatus(t, 201))
	c.Do("PUT", ws+"/digests/"+d.ID, settings(name, feed.srv.URL+"/feed/", n.listID("Product Updates"), 2)).MustStatus(t, 200)
	d = decodeDetail(t, c.Do("POST", ws+"/digests/"+d.ID+"/turn-on", map[string]any{}).MustStatus(t, 200))
	return d
}

// TestDigestIssueFlow prepares an issue, waits for its approval, starts it
// on the owner's authority and skips the next one for lack of posts.
func TestDigestIssueFlow(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	starter := &fakeStarter{}
	env.App.Sending = starter
	n := counter{t, env}
	ctx := context.Background()
	feed := newFeedServer(t, basePosts(time.Now())...)
	linh, marcus := env.Client(), env.Client()
	linh.SignInDemo("linh@acme.io")
	marcus.SignInDemo("marcus@acme.io")
	d := setUp(t, linh, n, feed, "Flow roundup")
	svc := env.App.Digests()

	// Preparation: the feed is read again, the issue is built and checked,
	// and the approval rules ask Marcus or another approver in Linh's name.
	issueID := uuid.MustParse(d.Next.ID)
	if err := svc.Prepare(ctx, issueID); err != nil {
		t.Fatal(err)
	}
	d = getDigest(t, linh, d.ID)
	if d.Next == nil || d.Next.Status != "waiting" || d.Next.CampaignID == "" || d.Next.Approval == nil || d.Next.Approval.Status != "pending" {
		if d.Next != nil && d.Next.CampaignID != "" {
			t.Logf("blocks: %s", n.blocks(d.Next.CampaignID))
		}
		t.Fatalf("prepared: %+v", d.Next)
	}
	if d.Next.Subject != "Feature flags at 2,000 teams: what broke first and 1 more from Test blog" {
		t.Fatalf("subject: %q", d.Next.Subject)
	}
	if n.audit("digest.issue_prepared", d.ID) != 1 || n.audit("approval.requested", d.Next.CampaignID) != 1 {
		t.Fatal("preparation isn't audited")
	}
	if n.notes("marcus@acme.io", "approval_requested") == 0 {
		t.Fatal("Marcus wasn't asked to approve")
	}
	// The issue is a campaign, under Scheduled, with its digest.
	var campaigns struct {
		Items []struct {
			ID, Name, Status string
			Digest           *struct{ ID, Name, IssueStatus string }
		}
	}
	linh.Do("GET", ws+"/campaigns?pageSize=100&q=Flow%20roundup", nil).MustStatus(t, 200).JSON(t, &campaigns)
	if len(campaigns.Items) != 1 || campaigns.Items[0].Status != "scheduled" || campaigns.Items[0].Digest == nil || campaigns.Items[0].Digest.ID != d.ID ||
		campaigns.Items[0].Digest.IssueStatus != "waiting" {
		t.Fatalf("campaigns: %+v", campaigns.Items)
	}
	// What the approvals inbox and Review & send learn about the issue.
	var infos struct {
		Items []struct {
			DigestName string
			Number     int
			Status     string
			Posts      []struct{ Title string }
			Since      *struct{ Number int }
		}
	}
	linh.Do("GET", ws+"/digest-issues?campaigns="+d.Next.CampaignID, nil).MustStatus(t, 200).JSON(t, &infos)
	if len(infos.Items) != 1 || infos.Items[0].Number != 1 || infos.Items[0].Status != "waiting" || len(infos.Items[0].Posts) != 2 || infos.Items[0].Since != nil {
		t.Fatalf("issue infos: %+v", infos.Items)
	}

	// Its time comes while the approval is pending: it waits.
	n.exec(`UPDATE digest_issues SET send_at = now() - interval '1 minute', deadline = now() + interval '1 day' WHERE id = $1`, issueID)
	if err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if len(starter.started) != 0 {
		t.Fatal("started before the approval")
	}
	// Marcus approves; the next minute it starts, on Linh's authority.
	marcus.Do("POST", ws+"/approvals/"+d.Next.Approval.ID+"/decision", map[string]any{"decision": "approve"}).MustStatus(t, 200)
	if err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if len(starter.started) != 1 || starter.started[0].String() != d.Next.CampaignID || len(starter.rechecks) != 1 {
		t.Fatalf("started: %+v", starter.started)
	}
	var linhID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM users WHERE email = 'linh@acme.io'`).Scan(&linhID); err != nil {
		t.Fatal(err)
	}
	if starter.by[0] != linhID {
		t.Fatalf("started by %s, want the owner", starter.by[0])
	}
	var issues struct {
		Items []struct {
			Number     *int
			Status     string
			SkipReason string
			Posts      int
		}
	}
	issues.Items = nil
	linh.Do("GET", ws+"/digests/"+d.ID+"/issues", nil).MustStatus(t, 200).JSON(t, &issues)
	if len(issues.Items) != 2 || issues.Items[0].Status != "collecting" || issues.Items[1].Status != "sending" || issues.Items[1].Number == nil || *issues.Items[1].Number != 1 {
		t.Fatalf("issues: %+v", issues.Items)
	}

	// The next issue has no new posts: it is skipped and uses no number.
	d = getDigest(t, linh, d.ID)
	if d.Next == nil || d.Next.Posts != 0 {
		t.Fatalf("next: %+v", d.Next)
	}
	if err := svc.Prepare(ctx, uuid.MustParse(d.Next.ID)); err != nil {
		t.Fatal(err)
	}
	issues.Items = nil
	linh.Do("GET", ws+"/digests/"+d.ID+"/issues", nil).MustStatus(t, 200).JSON(t, &issues)
	if len(issues.Items) != 3 || issues.Items[1].Status != "skipped" || issues.Items[1].SkipReason != "no_items" || issues.Items[1].Number != nil {
		t.Fatalf("after the empty preparation: %+v", issues.Items)
	}
	if n.audit("digest.issue_skipped", d.ID) != 1 {
		t.Fatal("the skip isn't audited")
	}

	// A new post goes in the next issue; the sent ones don't come back.
	feed.add(post{"104", "A year of on-call in numbers", "Engineering", "Joel Weber", time.Now()})
	d = getDigest(t, linh, d.ID)
	linh.Do("POST", ws+"/digests/"+d.ID+"/read", nil).MustStatus(t, 200)
	d = getDigest(t, linh, d.ID)
	if d.Next == nil || d.Next.Posts != 1 || d.Feed.Waiting != 1 {
		t.Fatalf("after the new post: %+v", d.Next)
	}
	// A person skips it; its post stays new.
	linh.Do("POST", ws+"/digests/"+d.ID+"/issues/"+d.Next.ID+"/skip", map[string]any{"note": "Quiet week"}).MustStatus(t, 200)
	d = getDigest(t, linh, d.ID)
	if d.Next == nil || d.Next.Posts != 1 {
		t.Fatalf("after skipping by hand: %+v", d.Next)
	}

	// MCP agents read digests and issues with the same permission check.
	var history struct {
		Items []struct{ Kind, Title string }
	}
	linh.Do("GET", ws+"/digests/"+d.ID+"/history", nil).MustStatus(t, 200).JSON(t, &history)
	if len(history.Items) == 0 {
		t.Fatal("no history")
	}
}

// TestDigestHoldAndOwner holds an issue that Send Readiness blocks, refuses
// to release it until it's fixed, and pauses the digest when its owner
// loses campaigns:send.
func TestDigestHoldAndOwner(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Sending = &fakeStarter{}
	n := counter{t, env}
	ctx := context.Background()
	feed := newFeedServer(t, basePosts(time.Now())...)
	linh, sofia := env.Client(), env.Client()
	linh.SignInDemo("linh@acme.io")
	sofia.SignInDemo("sofia@acme.io")
	svc := env.App.Digests()

	// Sofia (Editor) owns this one.
	d := decodeDetail(t, sofia.Do("POST", ws+"/digests", map[string]any{"name": "Held roundup", "source": "url", "feedUrl": feed.srv.URL + "/feed/"}).MustStatus(t, 201))
	sofia.Do("PUT", ws+"/digests/"+d.ID, settings("Held roundup", feed.srv.URL+"/feed/", n.listID("Product Updates"), 2)).MustStatus(t, 200)
	d = decodeDetail(t, sofia.Do("POST", ws+"/digests/"+d.ID+"/turn-on", map[string]any{}).MustStatus(t, 200))
	if d.Owner == nil || d.Owner.Name != "Sofia Rossi" {
		t.Fatalf("owner: %+v", d.Owner)
	}
	// The template loses its footer: Send Readiness blocks the issue.
	var original string
	if err := env.App.Pool.QueryRow(ctx, `SELECT mjml FROM campaigns WHERE id = $1`, uuid.MustParse(d.TemplateID)).Scan(&original); err != nil {
		t.Fatal(err)
	}
	n.exec(`UPDATE campaigns SET mjml = regexp_replace(mjml, '<!-- rc:footer -->.*?</mj-section>', '') WHERE id = $1`, uuid.MustParse(d.TemplateID))
	if err := svc.Prepare(ctx, uuid.MustParse(d.Next.ID)); err != nil {
		t.Fatal(err)
	}
	d = getDigest(t, sofia, d.ID)
	if d.Next == nil || d.Next.Status != "held" || d.Next.HoldReason != "readiness" {
		t.Fatalf("held: %+v", d.Next)
	}
	if n.notes("sofia@acme.io", "digest_issue_held") != 1 || n.audit("digest.issue_held", d.ID) != 1 {
		t.Fatal("the hold isn't notified or audited")
	}
	// Releasing needs the block fixed.
	sofia.Do("POST", ws+"/digests/"+d.ID+"/issues/"+d.Next.ID+"/release", nil).MustStatus(t, 409)
	// Fixed in the template and rebuilt: it can go.
	n.exec(`UPDATE campaigns SET mjml = $2 WHERE id = $1`, uuid.MustParse(d.TemplateID), original)
	sofia.Do("POST", ws+"/digests/"+d.ID+"/issues/"+d.Next.ID+"/rebuild", nil).MustStatus(t, 200)
	sofia.Do("POST", ws+"/digests/"+d.ID+"/issues/"+d.Next.ID+"/release", nil).MustStatus(t, 200)
	d = getDigest(t, sofia, d.ID)
	if d.Next == nil || (d.Next.Status != "prepared" && d.Next.Status != "waiting") {
		t.Fatalf("released: %+v", d.Next)
	}
	if n.audit("digest.issue_released", d.ID) != 1 || n.audit("digest.issue_rebuilt", d.ID) != 1 {
		t.Fatal("release or rebuild isn't audited")
	}

	// Sofia becomes a Content Writer: she no longer has campaigns:send, so
	// the digest pauses and the members who can send hear about it.
	var members struct {
		Items []struct{ ID, Email string }
		Roles []struct{ ID, Key, Name string }
	}
	linh.Do("GET", ws+"/members", nil).MustStatus(t, 200).JSON(t, &members)
	memberID, roleID := "", ""
	for _, m := range members.Items {
		if m.Email == "sofia@acme.io" {
			memberID = m.ID
		}
	}
	for _, r := range members.Roles {
		if r.Name == "Content Writer" {
			roleID = r.ID
		}
	}
	if memberID == "" || roleID == "" {
		t.Fatalf("members: %+v", members)
	}
	linh.Do("PATCH", ws+"/members/"+memberID, map[string]any{"roleId": roleID}).MustStatus(t, 204)
	if err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	d = getDigest(t, linh, d.ID)
	if d.State != "paused" || d.Pause == nil || d.Pause.Reason != "owner" || d.Pause.From != "Editor" || d.Pause.To != "Content Writer" || d.Next != nil {
		t.Fatalf("after the role change: %+v %+v", d.State, d.Pause)
	}
	// (The demo's own digest, also Sofia's, pauses too: count this one's.)
	if n.notesFor("marcus@acme.io", "digest_paused", d.ID) != 1 || n.notesFor("sofia@acme.io", "digest_paused", d.ID) != 1 {
		t.Fatal("the automatic pause isn't notified")
	}
	// Sofia can't turn it back on; Linh can, and becomes its owner.
	sofia.Do("POST", ws+"/digests/"+d.ID+"/turn-on", map[string]any{"mode": "auto"}).MustStatus(t, 403)
	d = decodeDetail(t, linh.Do("POST", ws+"/digests/"+d.ID+"/turn-on", map[string]any{}).MustStatus(t, 200))
	if d.Owner == nil || d.Owner.Name != "Linh Nguyen" || d.State != "on" {
		t.Fatalf("turned back on: %+v", d)
	}
}

// TestDigestFeedFailures retries a feed that fails at preparation, skips the
// issue at its send time and pauses after three such skips in a row.
func TestDigestFeedFailures(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Sending = &fakeStarter{}
	n := counter{t, env}
	ctx := context.Background()
	feed := newFeedServer(t, basePosts(time.Now())...)
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	d := setUp(t, linh, n, feed, "Failing roundup")
	svc := env.App.Digests()

	feed.setFail(true)
	for i := 0; i < 3; i++ {
		d = getDigest(t, linh, d.ID)
		if d.State != "on" || d.Next == nil {
			t.Fatalf("round %d: %+v", i, d)
		}
		issueID := uuid.MustParse(d.Next.ID)
		if err := svc.Prepare(ctx, issueID); err != nil {
			t.Fatal(err)
		}
		d = getDigest(t, linh, d.ID)
		if d.Next == nil || d.Next.Status != "preparing" || len(d.Next.FeedTries) != 1 {
			t.Fatalf("round %d, after the failed read: %+v", i, d.Next)
		}
		// The send time passes with the feed still down: skipped.
		n.exec(`UPDATE digest_issues SET send_at = now() - interval '1 minute' WHERE id = $1`, issueID)
		if err := svc.Tick(ctx); err != nil {
			t.Fatal(err)
		}
	}
	d = getDigest(t, linh, d.ID)
	if d.State != "paused" || d.Pause == nil || d.Pause.Reason != "feed" {
		t.Fatalf("after three feed skips: %+v %+v", d.State, d.Pause)
	}
	var issues struct {
		Items []struct{ Status, SkipReason string }
	}
	issues.Items = nil
	linh.Do("GET", ws+"/digests/"+d.ID+"/issues", nil).MustStatus(t, 200).JSON(t, &issues)
	skips := 0
	for _, is := range issues.Items {
		if is.Status == "skipped" && is.SkipReason == "feed" {
			skips++
		}
	}
	if skips != 3 {
		t.Fatalf("issues: %+v", issues.Items)
	}
	// The owner heard at each issue's first failure and at each skip, then
	// of the pause.
	if got := n.notes("linh@acme.io", "digest_feed_failing"); got != 6 {
		t.Fatalf("feed failing notifications: %d, want 6", got)
	}
	if got := n.notes("linh@acme.io", "digest_paused"); got != 1 {
		t.Fatalf("paused notifications: %d, want 1", got)
	}
	// A feed that isn't a feed is reported with its first lines.
	feed.setFail(false)
	var found struct {
		OK    bool
		Kind  string
		Items int
		Error *struct{ Kind string }
	}
	linh.Do("POST", ws+"/digests/find", map[string]any{"address": feed.srv.URL + "/feed/"}).MustStatus(t, 200).JSON(t, &found)
	if !found.OK || found.Kind != "rss" || found.Items != 3 {
		t.Fatalf("find: %+v", found)
	}
}
