package seedbox_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/hostrepute/reputecast/internal/placement/seedbox"
	"github.com/hostrepute/reputecast/internal/placement/seedbox/seedtest"
)

// copyMessage is a placement copy as a mailbox stores it, with the headers
// the receiving server added.
func copyMessage(token, authResults, received string) []byte {
	return []byte(strings.Join([]string{
		"Authentication-Results: " + authResults,
		"Received: by 2002:a05:6214:2d4b:b0:6f5:1c2a with SMTP id kd11csp912011qvb; Sun, 12 Oct 2026 13:20:09 -0700 (PDT)",
		"Received: " + received,
		"DKIM-Signature: v=1; a=rsa-sha256; c=relaxed/relaxed; d=news.acme.io; s=7kq2n3x; h=from:to:subject; bh=abc=; b=def=",
		"From: Acme Newsletter <digest@news.acme.io>",
		"To: acme.seed01@gmail.com",
		"Subject: Three launches, one deprecation, and the roadmap for Q4",
		"Message-ID: <" + token + ".01.ses@news.acme.io>",
		"X-ReputeCast-Test: " + token,
		"Content-Type: text/plain; charset=utf-8",
		"",
		"Hi there,",
		"",
	}, "\r\n"))
}

func other(subject string) []byte {
	return []byte("From: someone@example.com\r\nTo: acme.seed@yahoo.com\r\nSubject: " + subject + "\r\nMessage-ID: <x1@example.com>\r\n\r\nhello\r\n")
}

const gmailAuth = `mx.google.com; dkim=pass header.i=@news.acme.io header.s=7kq2n3x header.b=AbCdEf12; spf=pass (google.com: domain of bounce@bounce.news.acme.io designates 54.240.8.25 as permitted sender) smtp.mailfrom=bounce@bounce.news.acme.io; dmarc=pass (p=NONE sp=QUARANTINE dis=NONE) header.from=news.acme.io`
const gmailReceived = `from a8-25.smtp-out.amazonses.com (a8-25.smtp-out.amazonses.com. [54.240.8.25]) by mx.google.com with ESMTPS id 6a1803df08f44-6f5d3e2b9b1si1144906d6.112; Sun, 12 Oct 2026 13:20:09 -0700 (PDT)`

func open(t *testing.T, srv *seedtest.Server, provider, user, password string) seedbox.Box {
	t.Helper()
	b, err := seedbox.Open(context.Background(), seedbox.Config{Provider: provider, Email: user, Auth: seedbox.AuthPassword,
		Host: srv.Host, Port: srv.Port, Username: user, Password: password, InsecureIMAP: true, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("open %s: %v", user, err)
	}
	t.Cleanup(func() { _ = b.Close() })
	return b
}

func TestFoldersAndFind(t *testing.T) {
	srv := seedtest.New(t, seedtest.Options{})
	srv.AddUser("acme.seed@yahoo.com", "app-password", "Bulk")
	now := time.Now().Add(-time.Minute).Truncate(time.Second)
	if err := srv.Deliver("acme.seed@yahoo.com", "INBOX", copyMessage("pt-00aa11bb", gmailAuth, gmailReceived), now, ""); err != nil {
		t.Fatal(err)
	}
	if err := srv.Deliver("acme.seed@yahoo.com", "Bulk", copyMessage("pt-22cc33dd", gmailAuth, gmailReceived), now.Add(10*time.Second), ""); err != nil {
		t.Fatal(err)
	}
	_ = srv.Deliver("acme.seed@yahoo.com", "INBOX", other("Your order"), now, "")

	b := open(t, srv, seedbox.Yahoo, "acme.seed@yahoo.com", "app-password")
	folders, err := b.Folders(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 2 || folders[0].Role != seedbox.RoleInbox || folders[1].Role != seedbox.RoleSpam || folders[1].Name != "Bulk" || folders[1].Label != "Bulk" {
		t.Fatalf("folders = %+v, want Inbox and Bulk (\\Junk)", folders)
	}
	found, err := b.Find(context.Background(), []string{"pt-00aa11bb", "pt-22cc33dd", "pt-99999999"}, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 2 {
		t.Fatalf("found %d copies, want 2: %+v", len(found), found)
	}
	by := map[string]seedbox.Found{}
	for _, f := range found {
		by[f.Token] = f
	}
	in, spam := by["pt-00aa11bb"], by["pt-22cc33dd"]
	if in.Role != seedbox.RoleInbox || spam.Role != seedbox.RoleSpam || spam.Folder != "Bulk" {
		t.Errorf("roles: inbox %+v, spam %+v", in, spam)
	}
	if !in.ReceivedAt.Equal(now) {
		t.Errorf("received at %v, want the mailbox's own time %v", in.ReceivedAt, now)
	}
	a := in.Header.AuthResults()
	if a.SPF == nil || a.SPF.Result != "pass" || a.SPF.Domain != "bounce.news.acme.io" {
		t.Errorf("spf = %+v", a.SPF)
	}
	if a.DKIM == nil || a.DKIM.Result != "pass" || a.DKIM.Domain != "news.acme.io" || a.DKIM.Selector != "7kq2n3x" {
		t.Errorf("dkim = %+v", a.DKIM)
	}
	if a.DMARC == nil || a.DMARC.Result != "pass" || a.DMARC.Domain != "news.acme.io" {
		t.Errorf("dmarc = %+v", a.DMARC)
	}
	hop := in.Header.HandOver()
	if hop == nil || hop.IP != "54.240.8.25" || hop.Host != "a8-25.smtp-out.amazonses.com" || hop.By != "mx.google.com" {
		t.Errorf("hand-over hop = %+v", hop)
	}
}

func TestGmailTabs(t *testing.T) {
	srv := seedtest.New(t, seedtest.Options{Gmail: true})
	srv.AddUser("acme.seed01@gmail.com", "app-password", "[Gmail]/Spam")
	now := time.Now().Truncate(time.Second)
	_ = srv.Deliver("acme.seed01@gmail.com", "INBOX", copyMessage("pt-0000aaaa", gmailAuth, gmailReceived), now, "promotions")
	_ = srv.Deliver("acme.seed01@gmail.com", "INBOX", copyMessage("pt-0000bbbb", gmailAuth, gmailReceived), now, "")

	b := open(t, srv, seedbox.Gmail, "acme.seed01@gmail.com", "app-password")
	folders, _ := b.Folders(context.Background())
	labels := []string{}
	for _, f := range folders {
		labels = append(labels, f.Label)
	}
	if got := strings.Join(labels, ","); got != "Inbox,Promotions,Updates,Social,Forums,Spam" {
		t.Errorf("gmail folders = %s", got)
	}
	found, err := b.Find(context.Background(), []string{"pt-0000aaaa", "pt-0000bbbb"}, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]seedbox.Found{}
	for _, f := range found {
		by[f.Token] = f
	}
	if f := by["pt-0000aaaa"]; f.Role != seedbox.RoleTab || f.Tab != "Promotions" {
		t.Errorf("promotions copy = %+v", f)
	}
	if f := by["pt-0000bbbb"]; f.Role != seedbox.RoleInbox || f.Tab != "Primary" {
		t.Errorf("primary copy = %+v", f)
	}
}

func TestDeleteByToken(t *testing.T) {
	srv := seedtest.New(t, seedtest.Options{})
	srv.AddUser("acme.seed@gmx.de", "pw", "Spamverdacht")
	now := time.Now().Truncate(time.Second)
	_ = srv.Deliver("acme.seed@gmx.de", "INBOX", copyMessage("pt-11112222", gmailAuth, gmailReceived), now, "")
	_ = srv.Deliver("acme.seed@gmx.de", "Spamverdacht", copyMessage("pt-33334444", gmailAuth, gmailReceived), now, "")
	_ = srv.Deliver("acme.seed@gmx.de", "INBOX", other("Not a test copy"), now, "")

	b := open(t, srv, seedbox.GMX, "acme.seed@gmx.de", "pw")
	n, err := b.Delete(context.Background(), []string{"pt-11112222", "pt-33334444"}, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("deleted %d, want 2", n)
	}
	if got := srv.Count("acme.seed@gmx.de", "INBOX"); got != 1 {
		t.Errorf("INBOX holds %d messages, want only the one without a token", got)
	}
	if got := srv.Count("acme.seed@gmx.de", "Spamverdacht"); got != 0 {
		t.Errorf("Spamverdacht holds %d messages, want 0", got)
	}
}

func TestGmailDeleteGoesThroughTrash(t *testing.T) {
	srv := seedtest.New(t, seedtest.Options{Gmail: true})
	srv.AddUser("acme.seed02@gmail.com", "pw", "[Gmail]/Spam")
	now := time.Now().Truncate(time.Second)
	_ = srv.Deliver("acme.seed02@gmail.com", "INBOX", copyMessage("pt-5555aaaa", gmailAuth, gmailReceived), now, "promotions")
	b := open(t, srv, seedbox.Gmail, "acme.seed02@gmail.com", "pw")
	if n, err := b.Delete(context.Background(), []string{"pt-5555aaaa"}, now.Add(-time.Hour)); err != nil || n != 1 {
		t.Fatalf("delete = %d, %v", n, err)
	}
	if srv.Count("acme.seed02@gmail.com", "INBOX") != 0 || srv.Count("acme.seed02@gmail.com", "[Gmail]/Trash") != 0 {
		t.Errorf("the copy should be gone from the inbox and expunged from the trash")
	}
}

func TestPasswordRefused(t *testing.T) {
	srv := seedtest.New(t, seedtest.Options{})
	srv.AddUser("acme.seed@icloud.com", "old-app-password", "Junk")
	_, err := seedbox.Open(context.Background(), seedbox.Config{Provider: seedbox.ICloud, Email: "acme.seed@icloud.com", Auth: seedbox.AuthPassword,
		Host: srv.Host, Port: srv.Port, Password: "revoked", InsecureIMAP: true, Timeout: 5 * time.Second})
	if !errors.Is(err, seedbox.ErrAuthFailed) {
		t.Fatalf("a rejected password = %v, want ErrAuthFailed", err)
	}
}

func TestUnreachable(t *testing.T) {
	_, err := seedbox.Open(context.Background(), seedbox.Config{Provider: seedbox.Yahoo, Email: "a@yahoo.com", Auth: seedbox.AuthPassword,
		Host: "127.0.0.1", Port: 1, Password: "x", InsecureIMAP: true, Timeout: 2 * time.Second})
	if !errors.Is(err, seedbox.ErrUnreachable) {
		t.Fatalf("a closed port = %v, want ErrUnreachable", err)
	}
}

// tokenEndpoint is a fake OAuth token endpoint: it refreshes "good" refresh
// tokens and refuses the rest with invalid_grant, as Google and Microsoft do
// when a grant ends.
func tokenEndpoint(t *testing.T, access string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		w.Header().Set("Content-Type", "application/json")
		if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "good-refresh" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"Token has been expired or revoked."}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": access, "token_type": "Bearer", "expires_in": 3599})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func tokenSource(endpoint, refresh string) oauth2.TokenSource {
	cfg := &oauth2.Config{ClientID: "id", ClientSecret: "secret", Endpoint: oauth2.Endpoint{TokenURL: endpoint, AuthStyle: oauth2.AuthStyleInParams}}
	// An expired access token forces a refresh.
	return cfg.TokenSource(context.Background(), &oauth2.Token{AccessToken: "stale", RefreshToken: refresh, Expiry: time.Now().Add(-time.Minute)})
}

func TestXOAUTH2AndRefresh(t *testing.T) {
	srv := seedtest.New(t, seedtest.Options{Gmail: true})
	srv.AddUser("seed@gws.acmeseed.io", "unused", "[Gmail]/Spam")
	srv.AllowToken("ya29.fresh-access", "seed@gws.acmeseed.io")
	te := tokenEndpoint(t, "ya29.fresh-access")
	cfg := seedbox.Config{Provider: seedbox.GoogleWorkspace, Email: "seed@gws.acmeseed.io", Auth: seedbox.AuthOAuth,
		Host: srv.Host, Port: srv.Port, InsecureIMAP: true, Timeout: 5 * time.Second, Token: tokenSource(te.URL, "good-refresh")}
	b, err := seedbox.Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("XOAUTH2 with a refreshed token: %v", err)
	}
	_ = b.Close()

	cfg.Token = tokenSource(te.URL, "revoked-refresh")
	if _, err := seedbox.Open(context.Background(), cfg); !errors.Is(err, seedbox.ErrAuthExpired) {
		t.Fatalf("a refresh the provider refuses = %v, want ErrAuthExpired", err)
	}
}

// graphFake answers Microsoft Graph's mail endpoints for one seed.
type graphFake struct {
	token   string
	deleted []string
}

func (g *graphFake) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+g.token {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"InvalidAuthenticationToken","message":"Lifetime validation failed, the token is expired."}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		msg := func(id, token, class string, at time.Time) map[string]any {
			return map[string]any{
				"id": id, "internetMessageId": "<0100019a.ses@email.amazonses.com>", "receivedDateTime": at.UTC().Format(time.RFC3339),
				"inferenceClassification": class,
				"internetMessageHeaders": []map[string]string{
					{"name": "Authentication-Results", "value": "spf=pass (sender IP is 203.0.113.25) smtp.mailfrom=rp.news.acme.io; dkim=pass (signature was verified) header.d=news.acme.io;dmarc=pass action=none header.from=news.acme.io;compauth=pass reason=100"},
					{"name": "Received", "value": "from mta.acme.io (203.0.113.25) by AM4PEPF00027A6C.mail.protection.outlook.com (10.167.16.90) with Microsoft SMTP Server (version=TLS1_3); Sun, 12 Oct 2026 20:20:44 +0000"},
					{"name": "DKIM-Signature", "value": "v=1; a=rsa-sha256; d=news.acme.io; s=postal-2026; h=from:to; bh=x; b=y"},
					{"name": "X-ReputeCast-Test", "value": token},
				},
			}
		}
		now := time.Now()
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/me/mailFolders/inbox/messages"):
			_ = json.NewEncoder(w).Encode(map[string]any{"value": []any{msg("AAMk1", "pt-aaaa0001", "focused", now), msg("AAMk2", "pt-aaaa0002", "other", now)}})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/me/mailFolders/junkemail/messages"):
			_ = json.NewEncoder(w).Encode(map[string]any{"value": []any{msg("AAMk3", "pt-aaaa0003", "focused", now)}})
		case r.Method == http.MethodGet && r.URL.Path == "/me/mailFolders/inbox":
			_, _ = w.Write([]byte(`{"id":"AQMkInbox","displayName":"Inbox"}`))
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/me/messages/"):
			g.deleted = append(g.deleted, strings.TrimPrefix(r.URL.Path, "/me/messages/"))
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected Graph call %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func TestGraphFocusedOtherAndJunk(t *testing.T) {
	g := &graphFake{token: "eyJ0eXAi.graph"}
	srv := httptest.NewServer(g.handler(t))
	defer srv.Close()
	te := tokenEndpoint(t, "eyJ0eXAi.graph")
	b, err := seedbox.Open(context.Background(), seedbox.Config{Provider: seedbox.Outlook, Email: "acme.seed@outlook.com", Auth: seedbox.AuthOAuth,
		GraphURL: srv.URL, Token: tokenSource(te.URL, "good-refresh"), Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	found, err := b.Find(context.Background(), []string{"pt-aaaa0001", "pt-aaaa0002", "pt-aaaa0003"}, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range found {
		got[f.Token] = fmt.Sprintf("%s/%s/%s", f.Role, f.Folder, f.Tab)
	}
	want := map[string]string{"pt-aaaa0001": "inbox/Inbox/Focused", "pt-aaaa0002": "inbox/Inbox/Other", "pt-aaaa0003": "spam/Junk Email/"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	a := found[0].Header.AuthResults()
	if a.SPF == nil || a.SPF.Domain != "rp.news.acme.io" || a.SPF.Detail != "sender IP is 203.0.113.25" {
		t.Errorf("Microsoft's Authentication-Results without an authserv-id: spf = %+v", a.SPF)
	}
	if a.DKIM == nil || a.DKIM.Selector != "postal-2026" {
		t.Errorf("selector from the DKIM-Signature: %+v", a.DKIM)
	}
	if hop := found[0].Header.HandOver(); hop == nil || hop.IP != "203.0.113.25" || hop.Host != "mta.acme.io" {
		t.Errorf("hop = %+v", hop)
	}
	if n, err := b.Delete(context.Background(), []string{"pt-aaaa0003"}, time.Now().Add(-time.Hour)); err != nil || n != 1 || len(g.deleted) != 1 || g.deleted[0] != "AAMk3" {
		t.Errorf("delete = %d, %v, %v", n, err, g.deleted)
	}

	// A grant Microsoft ended reads as expired.
	if _, err := seedbox.Open(context.Background(), seedbox.Config{Provider: seedbox.Microsoft365, Email: "seed@m365.acmeseed.io", Auth: seedbox.AuthOAuth,
		GraphURL: srv.URL, Token: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "ended"}), Timeout: 5 * time.Second}); !errors.Is(err, seedbox.ErrAuthExpired) {
		t.Errorf("a token Graph refuses = %v, want ErrAuthExpired", err)
	}
}

func TestAuthResultsForms(t *testing.T) {
	h := seedbox.ParseHeader([]byte("Authentication-Results: mx.aol.com;\r\n  spf=fail smtp.mailfrom=bounce.example.com;\r\n  dkim=fail (bad signature) header.i=@example.com header.s=sel1;\r\n  dmarc=fail header.from=example.com\r\n"))
	a := h.AuthResults()
	if a.SPF.Result != "fail" || a.DKIM.Result != "fail" || a.DKIM.Selector != "sel1" || a.DKIM.Domain != "example.com" || a.DMARC.Result != "fail" {
		t.Errorf("folded RFC 8601 header: %+v %+v %+v", a.SPF, a.DKIM, a.DMARC)
	}
	if got := seedbox.ParseHeader([]byte("Message-ID: <pt-7f3a91c2.04.postal@news.acme.io>\r\n")).Tokens(); len(got) != 1 || got[0] != "pt-7f3a91c2" {
		t.Errorf("token from the Message-ID = %v", got)
	}
}
