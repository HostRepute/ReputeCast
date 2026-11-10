package engine_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/clicktoken"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// testUTMLinks (G-13): with the workspace's "Add UTM tags to links" on, a
// send's links (campaign_links) carry utm_source, utm_medium and
// utm_campaign where the author didn't set them, the click redirect goes to
// the tagged address and a provider-reported click on it finds its link; a
// campaign whose own choice is off stays untagged.
func testUTMLinks(t *testing.T, s *sendtest.Suite) {
	links := func(f *sendtest.Fixture) map[int]string {
		rows, err := f.Env.App.Pool.Query(f.Ctx, `SELECT position, url FROM campaign_links WHERE campaign_id = $1 ORDER BY position`, f.Campaign)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[int]string{}
		for rows.Next() {
			var p int
			var u string
			if err := rows.Scan(&p, &u); err != nil {
				t.Fatal(err)
			}
			out[p] = u
		}
		return out
	}
	start := func(f *sendtest.Fixture) {
		f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
		f.AddContacts(sendtest.ContactSpec{N: 2, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
		f.StartSend(engine.StartOptions{})
	}
	// One fixture at a time: a fixture's contact addresses follow the
	// suite's fixture count. The campaign's own choice wins over the
	// workspace's.
	off := s.Fixture(t, "Weekly Digest #142 untagged")
	off.Exec(`UPDATE workspaces SET sending_defaults = sending_defaults || '{"utm_links": true}' WHERE id = $1`, off.WS)
	defer off.Exec(`UPDATE workspaces SET sending_defaults = sending_defaults - 'utm_links' WHERE id = $1`, off.WS)
	off.Exec(`UPDATE campaigns SET content_settings = '{"minify": false, "utm": false}' WHERE id = $1`, off.Campaign)
	start(off)
	plain := links(off)
	for _, u := range plain {
		if strings.Contains(u, "utm_source=acme-newsletter") {
			t.Fatalf("a campaign with UTM tags off got them: %s", u)
		}
	}

	on := s.Fixture(t, "Weekly Digest #142")
	start(on)
	tagged := links(on)
	if len(tagged) == 0 || len(tagged) != len(plain) {
		t.Fatalf("%d tagged links, %d untagged", len(tagged), len(plain))
	}
	utm := clicktoken.UTM{Source: "acme-newsletter", Medium: "email", Campaign: "weekly-digest-142"}
	full := 0
	for p, u := range tagged {
		if want := clicktoken.AddUTM(plain[p], utm); u != want {
			t.Errorf("link %d: %q, want %q", p, u, want)
		}
		if !clicktoken.Taggable(plain[p]) {
			continue
		}
		q, _ := url.Parse(u)
		for _, k := range []string{"utm_source", "utm_medium", "utm_campaign"} {
			if !q.Query().Has(k) {
				t.Errorf("link %d has no %s: %s", p, k, u)
			}
		}
		if strings.HasSuffix(strings.SplitN(u, "#", 2)[0], "utm_source=acme-newsletter&utm_medium=email&utm_campaign=weekly-digest-142") {
			full++
		}
	}
	if full == 0 {
		t.Fatalf("no link got the three tags: %v", tagged)
	}

	// A provider that tracks the clicks reports the tagged address the
	// email showed: it is found as its link.
	for p, u := range tagged {
		if got := tracking.LinkPosition(tagged, u); got != p && tagged[got] != u {
			t.Errorf("a provider-reported click on %s found link %d, want %d", u, got, p)
		}
	}

	// ReputeCast's click redirect goes to the tagged address.
	if _, _, err := on.E.Drive(on.Ctx, on.Campaign, on.Start, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	var msg, contact uuid.UUID
	if err := on.Env.App.Pool.QueryRow(on.Ctx, `SELECT id, contact_id FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL ORDER BY id LIMIT 1`, on.Campaign).Scan(&msg, &contact); err != nil {
		t.Fatal(err)
	}
	click := on.Env.App.Links.Sign(linktoken.Claims{Kind: linktoken.Click, WorkspaceID: on.WS, MessageID: msg, ContactID: contact, Link: 0})
	noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Timeout: 10 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, on.Env.Server.URL+"/t/c/"+click, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36")
	resp, err := noFollow.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != tagged[0] {
		t.Fatalf("click %d to %q, want 302 to %q", resp.StatusCode, resp.Header.Get("Location"), tagged[0])
	}
}
