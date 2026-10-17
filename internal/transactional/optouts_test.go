package transactional_test

import (
	"context"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/linktoken"
)

// page sends a request to a public page and returns its status and text.
func (s *sendEnv) page(method, path string, form url.Values) (int, string) {
	s.t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(context.Background(), method, s.env.Server.URL+path, body)
	if err != nil {
		s.t.Fatal(err)
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		s.t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, html.UnescapeString(string(b))
}

var listUnsubscribeRe = regexp.MustCompile(`List-Unsubscribe: <([^>]+)>`)

func TestCategoryOptOuts(t *testing.T) {
	s := newSendEnv(t)
	s.publish("trial-ending", "Trial ends in 3 days", "trial-ending")
	_, secret := s.key("Production", map[string]any{"preset": "send_only"})
	trial := func(to string) map[string]any {
		return map[string]any{"template": "trial-ending", "to": map[string]any{"email": to},
			"data": map[string]any{"first_name": "Ops", "days_left": 3, "upgrade_url": "https://app.acme.io/billing"}}
	}
	var r sendResult
	s.call(secret, "/messages", "", trial("ops@contoso.example")).MustStatus(t, 202).JSON(t, &r)
	s.step()
	msgs := s.sink.Messages()
	m := listUnsubscribeRe.FindStringSubmatch(unfold(msgs[len(msgs)-1].Data))
	if m == nil {
		t.Fatal("no List-Unsubscribe header")
	}
	u, err := url.Parse(m[1])
	if err != nil {
		t.Fatal(err)
	}
	var cat uuid.UUID
	s.one(`SELECT category_id FROM transactional_templates WHERE key = 'trial-ending' AND workspace_id = $1`, &cat, s.ws)
	optedOut := func() (n int, source, reason string) {
		_ = s.env.App.Pool.QueryRow(s.ctx, `SELECT count(*) OVER (), source, reason FROM transactional_opt_outs WHERE email = 'ops@contoso.example' AND category_id = $1`, cat).Scan(&n, &source, &reason)
		return n, source, reason
	}

	// Link scanners check with HEAD: nothing changes.
	if code, _ := s.page(http.MethodHead, u.Path, nil); code != 200 {
		t.Fatalf("HEAD %d", code)
	}
	if n, _, _ := optedOut(); n != 0 {
		t.Fatal("HEAD turned the category off")
	}
	// The link turns off just that category, and says so.
	code, body := s.page(http.MethodGet, u.Path, nil)
	if code != 200 || !strings.Contains(body, "You won't get Product updates anymore") || !strings.Contains(body, "ops@contoso.example is off product updates emails from Acme Cloud.") ||
		!strings.Contains(body, "Account and billing emails still arrive") || !strings.Contains(body, "Turn Product updates back on") {
		t.Fatalf("category page %d:\n%s", code, body)
	}
	if n, source, _ := optedOut(); n != 1 || source != "unsubscribe_link" {
		t.Fatalf("opt-out %d %s", n, source)
	}
	// Sending to the address again is refused.
	p := s.call(secret, "/messages", "", trial("ops@contoso.example")).MustStatus(t, 422).Problem(t)
	if p["meta"].(map[string]any)["reason"] != "opted_out" {
		t.Fatalf("after opt-out: %v", p)
	}
	// Turned off by mistake: back on.
	_, body = s.page(http.MethodPost, u.Path, url.Values{"action": {"on"}})
	if !strings.Contains(body, "Product updates is on again") {
		t.Fatalf("back on:\n%s", body)
	}
	if n, _, _ := optedOut(); n != 0 {
		t.Fatal("still off")
	}
	// The mail app's one-click unsubscribe (RFC 8058).
	code, body = s.page(http.MethodPost, u.Path, url.Values{"List-Unsubscribe": {"One-Click"}})
	if code != 200 || !strings.Contains(body, "Unsubscribed") {
		t.Fatalf("one-click %d %s", code, body)
	}
	if n, source, _ := optedOut(); n != 1 || source != "one_click" {
		t.Fatalf("one-click opt-out %d %s", n, source)
	}
	_, body = s.page(http.MethodPost, u.Path, url.Values{"action": {"reason"}, "reason": {"in_app"}})
	if !strings.Contains(body, "Thanks for telling us.") {
		t.Fatalf("reason:\n%s", body)
	}
	if _, _, reason := optedOut(); reason != "in_app" {
		t.Fatalf("reason %q", reason)
	}

	// The preference center for someone who isn't a contact: just the
	// notifications section.
	prefs := "/p/acme-newsletter/preferences/" + s.env.App.Links.Sign(linktoken.Claims{Kind: linktoken.Preferences, WorkspaceID: s.ws, MessageID: uuid.MustParse(r.ID)})
	code, body = s.page(http.MethodGet, prefs, nil)
	if code != 200 || !strings.Contains(body, "Notifications from Acme Cloud") || !strings.Contains(body, "Product updates") || strings.Contains(body, "Weekly Digest") ||
		strings.Contains(body, "Unsubscribe from everything") {
		t.Fatalf("preferences %d:\n%s", code, body)
	}
	if code, _ = s.page(http.MethodPost, prefs, url.Values{"category": {cat.String()}, "categories_shown": {cat.String()}}); code != 303 {
		t.Fatalf("save %d", code)
	}
	if n, _, _ := optedOut(); n != 0 {
		t.Fatal("preference center didn't turn it on")
	}
	s.page(http.MethodPost, prefs, url.Values{"categories_shown": {cat.String()}})
	if n, source, _ := optedOut(); n != 1 || source != "preference_center" {
		t.Fatalf("preference center opt-out %d %s", n, source)
	}

	// A contact's preference center has the section next to the lists, and
	// saves both.
	var sam uuid.UUID
	s.one(`SELECT id FROM contacts WHERE workspace_id = $1 AND email = 'sam.rivera@outlook.com'`, &sam, s.ws)
	samPrefs := "/p/acme-newsletter/preferences/" + s.env.App.Links.Sign(linktoken.Claims{Kind: linktoken.Preferences, WorkspaceID: s.ws, ContactID: sam})
	_, body = s.page(http.MethodGet, samPrefs, nil)
	if !strings.Contains(body, "Notifications from Acme Cloud") || !strings.Contains(body, "Weekly Digest") {
		t.Fatalf("contact preferences:\n%s", body)
	}
	var lists []string
	rows, err := s.env.App.Pool.Query(s.ctx, `SELECT l.id::text FROM lists l JOIN list_memberships m ON m.list_id = l.id JOIN contacts c ON c.id = m.contact_id
		WHERE c.id = $1 AND m.status = 'subscribed'`, sam)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		lists = append(lists, id)
	}
	rows.Close()
	form := url.Values{"list": lists, "shown": {strings.Join(lists, ",")}, "categories_shown": {cat.String()}}
	if code, _ = s.page(http.MethodPost, samPrefs, form); code != 303 {
		t.Fatalf("contact save %d", code)
	}
	var n int
	s.one(`SELECT count(*) FROM transactional_opt_outs WHERE email = 'sam.rivera@outlook.com' AND category_id = $1`, &n, cat)
	if n != 1 {
		t.Fatal("contact's category not off")
	}
	s.one(`SELECT count(*) FROM list_memberships WHERE contact_id = $1 AND status = 'subscribed'`, &n, sam)
	if n != len(lists) {
		t.Fatalf("lists changed: %d of %d", n, len(lists))
	}
}
