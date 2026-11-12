package publicpages_test

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/mailer"
)

var muteLink = regexp.MustCompile(`/comments/mute/([A-Za-z0-9_-]{20,})`)

// Comments under a post (M61): a visitor reads them and is asked to sign
// in, with no form; Alex's first comment waits for review, seen only by
// him on a page never cached; a moderator publishes it and visitors see
// it; a post from another site is refused; Sam's reply emails Alex once and
// "Stop these emails" stops the next; the 11th comment in 10 minutes is
// refused; on a paid-only site Alex sees what it takes; erasing Alex
// anonymises his comment.
func TestPostComments(t *testing.T) {
	env, ids := demo(t)
	ctx := context.Background()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	const slug = "the-case-for-boring-infrastructure"
	path := "/posts/" + slug
	post := one[uuid.UUID](t, env, `SELECT id FROM website_posts WHERE workspace_id = $1 AND slug = $2`, ids.ws, slug)
	exec(`UPDATE website_settings SET comments = 'members', comments_review = 'first' WHERE workspace_id = $1`, ids.ws)
	// The demo's Alex has a display name from his comment on another post:
	// this test is his first time.
	exec(`UPDATE website_members SET display_name = '' WHERE contact_id = (SELECT id FROM contacts WHERE workspace_id = $1 AND email = 'a.murphy@icloud.com')`, ids.ws)
	mail := &fakeMail{}
	env.App.Website().Mail = func() mailer.Sender { return mail }
	token := formToken(env, "website_comment:"+post.String(), 5*time.Second)
	member := func(key string, origin bool) hdr {
		h := hdr{"Host": "news.acme.io", "Cookie": "rc_member=demo-session-" + key + "-" + ids.ws.String()}
		if origin {
			h["Origin"] = "http://news.acme.io"
		}
		return h
	}

	// A visitor: the comments and a way in, no form and no token.
	r := do(t, env, http.MethodGet, path, nil, newsHost)
	must(t, r, 200, "Comments (0)", "Sign in to join the conversation", `id="comments" aria-labelledby="comments-h"`)
	lacks(t, r, `id="cm-body"`, `name="_t"`)

	// Alex, a free member: the form asks for a display name, never cached.
	r = do(t, env, http.MethodGet, path, nil, member("alex", false))
	must(t, r, 200, `id="cm-body"`, `maxlength="5000"`, "Display name", "Add a comment", `name="website"`)
	if cc := r.Header.Get("Cache-Control"); cc != "private, no-store" {
		t.Errorf("Alex's page is cached: %q", cc)
	}
	form := url.Values{"body": {"Is there a v1 to v2 migration guide for the Python SDK?"}, "name": {"Alex Murphy"}, "_t": {token}}
	foreign := member("alex", false)
	foreign["Origin"] = "https://evil.example"
	must(t, do(t, env, http.MethodPost, path+"/comments", form, foreign), 403)
	r = do(t, env, http.MethodPost, path+"/comments", form, member("alex", true))
	if r.Status != http.StatusSeeOther || !strings.Contains(r.Header.Get("Location"), "#comment-") {
		t.Fatalf("Alex's comment: %d %s\n%s", r.Status, r.Header.Get("Location"), clip(r.Body))
	}
	id := one[uuid.UUID](t, env, `SELECT id FROM website_comments WHERE post_id = $1`, post)
	if st := one[string](t, env, `SELECT status || '/' || held_reason FROM website_comments WHERE id = $1`, id); st != "pending/first_comment" {
		t.Fatalf("Alex's first comment is %s", st)
	}
	r = do(t, env, http.MethodGet, path, nil, member("alex", false))
	must(t, r, 200, "Waiting for review", "Is there a v1 to v2 migration guide")
	r = do(t, env, http.MethodGet, path, nil, newsHost)
	lacks(t, r, "Is there a v1 to v2 migration guide")

	// A Viewer can't moderate; the Owner publishes, audited, and visitors see it.
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	decide := map[string]any{"ids": []string{id.String()}, "decision": "publish"}
	viewer.Do(http.MethodPost, wsAPI+"/website/comments/decide", decide).MustStatus(t, http.StatusForbidden)
	owner(env).Do(http.MethodPost, wsAPI+"/website/comments/decide", decide).MustStatus(t, http.StatusOK)
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'comment.published' AND target_id = $2`, ids.ws, id.String()); n != 1 {
		t.Errorf("%d comment.published entries", n)
	}
	r = do(t, env, http.MethodGet, path, nil, newsHost)
	must(t, r, 200, "Comments (1)", "Alex Murphy", "Is there a v1 to v2 migration guide", `"commentCount":1`)
	if cc := r.Header.Get("Cache-Control"); !strings.HasPrefix(cc, "public") {
		t.Errorf("a visitor's page isn't cacheable: %q", cc)
	}

	// Sam replies: Alex gets one email; Stop these emails stops the next.
	exec(`UPDATE website_settings SET comments_review = 'none' WHERE workspace_id = $1`, ids.ws)
	reply := url.Values{"body": {"The migration guide is in the docs, under SDKs."}, "parent": {id.String()}, "name": {"Sam Rivera"}, "_t": {token}}
	must(t, do(t, env, http.MethodPost, path+"/comments", reply, member("sam", true)), http.StatusSeeOther)
	if n, err := env.App.Website().SendCommentNotices(ctx); err != nil || n != 1 {
		t.Fatalf("reply emails: %d (%v)", n, err)
	}
	got := mail.to("a.murphy@icloud.com")
	if len(got) != 1 || got[0].Kind != "website_comment_reply" || !strings.Contains(got[0].Subject, "replied to your comment") {
		t.Fatalf("Alex's email: %+v", got)
	}
	m := muteLink.FindStringSubmatch(got[0].Text)
	if m == nil {
		t.Fatalf("no Stop these emails link in %q", got[0].Text)
	}
	must(t, do(t, env, http.MethodGet, "/comments/mute/"+m[1], nil, newsHost), 200, "Stop reply emails?")
	must(t, do(t, env, http.MethodPost, "/comments/mute/"+m[1], url.Values{}, newsHost), 200, "Reply emails are off")
	must(t, do(t, env, http.MethodPost, path+"/comments", reply, member("sam", true)), http.StatusSeeOther)
	if n, _ := env.App.Website().SendCommentNotices(ctx); n != 0 || len(mail.to("a.murphy@icloud.com")) != 1 {
		t.Errorf("a muted member got another email (%d sent)", n)
	}

	// The 11th comment in 10 minutes is refused.
	more := url.Values{"body": {"Another thought."}, "_t": {token}}
	for range 8 {
		must(t, do(t, env, http.MethodPost, path+"/comments", more, member("sam", true)), http.StatusSeeOther)
	}
	must(t, do(t, env, http.MethodPost, path+"/comments", more, member("sam", true)), http.StatusTooManyRequests, "commented a lot")

	// Paid members only: Alex reads and sees what it takes.
	exec(`UPDATE website_settings SET comments = 'paid' WHERE workspace_id = $1`, ids.ws)
	r = do(t, env, http.MethodGet, path, nil, member("alex", false))
	must(t, r, 200, "Comments are for paid members", "Upgrade to comment")
	lacks(t, r, `id="cm-body"`)

	// Erasing Alex anonymises his comment; Sam's reply under it stays.
	alex := one[uuid.UUID](t, env, `SELECT id FROM contacts WHERE workspace_id = $1 AND email = 'a.murphy@icloud.com'`, ids.ws)
	owner(env).Do(http.MethodPost, wsAPI+"/contacts/erase", map[string]any{"ids": []string{alex.String()}, "confirm": 1}).MustStatus(t, 200)
	if st := one[string](t, env, `SELECT status || '/' || body || '/' || coalesce(host(ip), '') FROM website_comments WHERE id = $1`, id); st != "deleted//" {
		t.Errorf("Alex's comment after erasure: %q", st)
	}
	exec(`UPDATE website_settings SET comments = 'members' WHERE workspace_id = $1`, ids.ws)
	must(t, do(t, env, http.MethodGet, path, nil, newsHost), 200, "This comment was removed.", "The migration guide is in the docs")
}
