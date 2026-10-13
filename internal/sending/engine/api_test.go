package engine_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/store"
)

func testAPI(t *testing.T, f *fixture) {
	f.exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.campaign)
	f.addContacts(contactSpec{n: 200, mailbox: "gmail", domain: "gmail.com", engaged: true})
	f.startSend(engine.StartOptions{})
	if _, _, err := f.e.Drive(f.ctx, f.campaign, f.start, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	base := "/workspaces/acme-newsletter/campaigns/" + f.campaign.String() + "/live"

	viewer := f.env.Client()
	viewer.SignInDemo("aisha@acme.io")
	var snap engine.LiveSnapshot
	viewer.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &snap)
	if snap.State != "sending" || snap.Sent == 0 || snap.Recipients != 200 {
		t.Fatalf("viewer snapshot %s sent=%d of %d", snap.State, snap.Sent, snap.Recipients)
	}
	// Viewers watch but can't pause; Content Writers can't resume or cancel.
	viewer.Do("POST", base+"/pause", map[string]any{}).MustStatus(t, 403)
	writer := f.env.Client()
	writer.SignInDemo("daniel@acme.io")
	writer.Do("POST", base+"/cancel", map[string]any{}).MustStatus(t, 403)

	owner := f.env.Client()
	owner.SignInDemo("linh@acme.io")
	var res engine.LiveActionResult
	owner.Do("POST", base+"/pause", map[string]any{"reason": "Wrong link in the footer"}).MustStatus(t, 200).JSON(t, &res)
	if res.Snapshot.State != "paused" || res.Snapshot.PausedBy != "Linh Nguyen" {
		t.Fatalf("after pause: %s by %q", res.Snapshot.State, res.Snapshot.PausedBy)
	}
	writer.Do("POST", base+"/resume", map[string]any{}).MustStatus(t, 403)
	if n := f.count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.send_pause' AND actor_name = 'Linh Nguyen'`, f.campaign.String()); n != 1 {
		t.Fatalf("%d pause audit entries by Linh", n)
	}

	// An MCP agent with campaigns:pause (the demo's "Ops agent") can stop a
	// send but never start or resume one.
	token, hash := authn.NewToken()
	token = "rc_agent_" + token
	hash = authn.HashToken(token)
	var linh uuid.UUID
	_ = f.env.App.Pool.QueryRow(f.ctx, `SELECT id FROM users WHERE email = 'linh@acme.io'`).Scan(&linh)
	f.exec(`INSERT INTO api_keys (id, workspace_id, kind, name, prefix, token_hash, permissions, scope, created_by)
		VALUES ($1, $2, 'mcp_agent', 'Ops test agent', 'rc_agent_test', $3, ARRAY['campaigns:pause','providers:view'], '{}', $4)`, uuid.Must(uuid.NewV7()), f.ws, hash, linh)
	agent := f.env.Client()
	agent.Bearer = token
	agent.Do("POST", base+"/resume", map[string]any{}).MustStatus(t, 403)
	owner.Do("POST", base+"/resume", map[string]any{}).MustStatus(t, 200)
	agent.Do("POST", base+"/pause", map[string]any{"reason": "Spike in complaints"}).MustStatus(t, 200)
	if n := f.count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.send_pause' AND actor_type = 'mcp_agent'`, f.campaign.String()); n != 1 {
		t.Fatalf("%d pause audit entries by the agent", n)
	}

	// Other workspaces' campaigns are invisible.
	var other uuid.UUID
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT c.id FROM campaigns c JOIN workspaces w ON w.id = c.workspace_id WHERE w.slug <> 'acme-newsletter' LIMIT 1`).Scan(&other); err == nil {
		owner.Do("GET", "/workspaces/acme-newsletter/campaigns/"+other.String()+"/live", nil).MustStatus(t, 404)
		owner.Do("POST", "/workspaces/acme-newsletter/campaigns/"+other.String()+"/live/pause", map[string]any{}).MustStatus(t, 404)
	}

	// The event stream sends a snapshot at once.
	ctx, cancel := context.WithTimeout(f.ctx, 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, f.env.Server.URL+"/api/v1"+base+"/stream", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.AddCookie(sessionCookie(t, f, "linh@acme.io"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("stream %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 1<<20), 4<<20)
	var gotID, gotEvent bool
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "id: "):
			gotID = true
		case line == "event: snapshot":
			gotEvent = true
		case strings.HasPrefix(line, "data: ") && gotEvent:
			var s engine.LiveSnapshot
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &s); err != nil {
				t.Fatal(err)
			}
			if !gotID || s.CampaignID != f.campaign.String() {
				t.Fatalf("stream snapshot id=%v campaign %s", gotID, s.CampaignID)
			}
			return
		}
	}
	t.Fatalf("no snapshot event on the stream: %v", sc.Err())
}

func testTracking(t *testing.T, f *fixture) {
	f.exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.campaign)
	f.addContacts(contactSpec{n: 3, mailbox: "gmail", domain: "gmail.com", engaged: true})
	f.startSend(engine.StartOptions{})
	if _, _, err := f.e.Drive(f.ctx, f.campaign, f.start, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	var msg, contact uuid.UUID
	if err := f.env.App.Pool.QueryRow(f.ctx, `SELECT id, contact_id FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL ORDER BY id LIMIT 1`, f.campaign).Scan(&msg, &contact); err != nil {
		t.Fatal(err)
	}
	// Back-date the send so the open isn't "too fast to be a person".
	f.exec(`UPDATE messages SET sent_at = now() - interval '20 minutes' WHERE id = $1`, msg)
	links := f.env.App.Links
	open := links.Sign(linktoken.Claims{Kind: linktoken.Open, WorkspaceID: f.ws, MessageID: msg, ContactID: contact})
	click := links.Sign(linktoken.Claims{Kind: linktoken.Click, WorkspaceID: f.ws, MessageID: msg, ContactID: contact, Link: 0})
	noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Timeout: 10 * time.Second}
	get := func(path, ua string) *http.Response {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, f.env.Server.URL+path, nil)
		if ua != "" {
			req.Header.Set("User-Agent", ua)
		}
		resp, err := noFollow.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp
	}
	// Apple Mail Privacy Protection prefetch first: a machine open.
	if r := get("/t/o/"+open, "Mozilla/5.0"); r.StatusCode != 200 || r.Header.Get("Content-Type") != "image/gif" || !strings.Contains(r.Header.Get("Cache-Control"), "no-store") {
		t.Fatalf("pixel %d %s %s", r.StatusCode, r.Header.Get("Content-Type"), r.Header.Get("Cache-Control"))
	}
	waitFor(t, func() bool {
		return f.count(`SELECT coalesce(sum(opened_machine), 0) FROM campaign_counters WHERE campaign_id = $1`, f.campaign) == 1
	})
	// Then the person opens: the open moves from machine to human.
	get("/t/o/"+open, "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36")
	waitFor(t, func() bool {
		return f.count(`SELECT coalesce(sum(opened_human), 0) FROM campaign_counters WHERE campaign_id = $1`, f.campaign) == 1 &&
			f.count(`SELECT coalesce(sum(opened_machine), 0) FROM campaign_counters WHERE campaign_id = $1`, f.campaign) == 0
	})
	// The engagement chart moves it too: it adds up to the counters.
	if h, m := f.count(`SELECT coalesce(sum(opened_human), 0) FROM campaign_hourly WHERE campaign_id = $1`, f.campaign),
		f.count(`SELECT coalesce(sum(opened_machine), 0) FROM campaign_hourly WHERE campaign_id = $1`, f.campaign); h != 1 || m != 0 {
		t.Fatalf("hourly opens: %d human, %d machine; want 1 and 0", h, m)
	}
	var want string
	_ = f.env.App.Pool.QueryRow(f.ctx, `SELECT url FROM campaign_links WHERE campaign_id = $1 AND position = 0`, f.campaign).Scan(&want)
	r := get("/t/c/"+click, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36")
	if r.StatusCode != http.StatusFound || r.Header.Get("Location") != want {
		t.Fatalf("click %d to %q, want 302 to %q", r.StatusCode, r.Header.Get("Location"), want)
	}
	waitFor(t, func() bool {
		return f.count(`SELECT coalesce(sum(clicks), 0) FROM campaign_links WHERE campaign_id = $1`, f.campaign) == 1 &&
			f.count(`SELECT coalesce(sum(clicked), 0) FROM campaign_counters WHERE campaign_id = $1`, f.campaign) == 1
	})
	// A scanner clicking is redirected but never counted.
	get("/t/c/"+click, "Mozilla/5.0 Barracuda Sentinel")
	waitFor(t, func() bool {
		return f.count(`SELECT count(*) FROM delivery_events WHERE message_id = $1 AND type = 'click' AND machine`, msg) == 1
	})
	if n := f.count(`SELECT coalesce(sum(clicks), 0) FROM campaign_links WHERE campaign_id = $1`, f.campaign); n != 1 {
		t.Fatalf("scanner click counted: %d clicks", n)
	}
	// Never an open redirect: forged tokens, other kinds and unknown links fail.
	if r := get("/t/c/"+open, ""); r.StatusCode != 404 {
		t.Fatalf("an open token used as a click: %d", r.StatusCode)
	}
	if r := get("/t/c/not-a-token", ""); r.StatusCode != 404 {
		t.Fatalf("forged click token: %d", r.StatusCode)
	}
	unknown := links.Sign(linktoken.Claims{Kind: linktoken.Click, WorkspaceID: f.ws, MessageID: msg, ContactID: contact, Link: 9999})
	if r := get("/t/c/"+unknown, ""); r.StatusCode != 404 || r.Header.Get("Location") != "" {
		t.Fatalf("unknown link: %d %q", r.StatusCode, r.Header.Get("Location"))
	}
	otherWS := links.Sign(linktoken.Claims{Kind: linktoken.Click, WorkspaceID: uuid.New(), MessageID: msg, ContactID: contact, Link: 0})
	if r := get("/t/c/"+otherWS, ""); r.StatusCode != 404 {
		t.Fatalf("token of another workspace: %d", r.StatusCode)
	}
	if r := get("/t/o/garbage", ""); r.StatusCode != 200 || r.Header.Get("Content-Type") != "image/gif" {
		t.Fatalf("a bad pixel token still gets the GIF: %d", r.StatusCode)
	}
	if n := f.count(`SELECT count(*) FROM delivery_events WHERE message_id = $1 AND type = 'open' AND NOT machine AND diagnostic = 'Chrome on macOS'`, msg); n != 1 {
		t.Fatalf("%d human opens described as Chrome on macOS", n)
	}
}

// sessionCookie signs a demo user in for a raw HTTP request (the stream).
func sessionCookie(t *testing.T, f *fixture, email string) *http.Cookie {
	t.Helper()
	u, err := f.env.App.Q.GetUserByEmail(f.ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	token, hash := authn.NewToken()
	if _, err := f.env.App.Q.CreateSession(f.ctx, store.CreateSessionParams{ID: uuid.Must(uuid.NewV7()), UserID: u.ID, TokenHash: hash, AuthMethod: "test",
		MfaVerified: true, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: f.env.App.Cfg.Auth.CookieName, Value: token}
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("condition not met within 5 s")
}
