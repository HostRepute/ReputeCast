package placement_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/placement"
	"github.com/hostrepute/reputecast/internal/placement/seedbox/seedtest"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/systemmail"
)

// clock is the tests' controllable time.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

// fakeDeliver hands copies straight to the fake IMAP servers: where says
// which folder a copy lands in ("inbox", "spam", "promotions", "drop").
type fakeDeliver struct {
	mu    sync.Mutex
	q     *store.Queries
	clock *clock
	srv   map[string]*seedtest.Server // seed address → its server
	spam  map[string]string           // seed address → spam folder
	where func(provider, to string) string
	sent  []providers.Message
	held  map[string][]byte // dropped copies, by seed and provider
}

func (f *fakeDeliver) Send(_ context.Context, p store.Provider, msg providers.Message, kind string) (systemmail.Delivery, error) {
	if kind != placement.Kind {
		return systemmail.Delivery{}, fmt.Errorf("kind %q", kind)
	}
	now := f.clock.now()
	f.mu.Lock()
	f.sent = append(f.sent, msg)
	f.mu.Unlock()
	to := msg.To.Address
	// The messages row the real deliverer writes (kind placement).
	if err := f.q.SysmailInsertMessage(context.Background(), store.SysmailInsertMessageParams{ID: msg.ID, WorkspaceID: p.WorkspaceID, Stream: string(msg.Stream),
		Email: to, ProviderID: &p.ID, MailboxProvider: "other", QueuedAt: now, Kind: kind}); err != nil {
		return systemmail.Delivery{}, err
	}
	raw := rawCopy(p, msg, now)
	srv := f.srv[to]
	switch f.where(p.Adapter, to) {
	case "drop":
		f.mu.Lock()
		f.held[to+"/"+p.Adapter] = raw
		f.mu.Unlock()
	case "spam":
		_ = srv.Deliver(to, f.spam[to], raw, now.Add(9*time.Second), "")
	case "promotions":
		_ = srv.Deliver(to, seedtest.Inbox, raw, now.Add(11*time.Second), "promotions")
	default:
		_ = srv.Deliver(to, seedtest.Inbox, raw, now.Add(14*time.Second), "")
	}
	return systemmail.Delivery{ProviderMessageID: "pm-" + msg.ID.String()[:8], QueuedAt: now, SentAt: now}, nil
}

// late delivers a dropped copy now.
func (f *fakeDeliver) late(to, adapter string, at time.Time) error {
	f.mu.Lock()
	raw := f.held[to+"/"+adapter]
	f.mu.Unlock()
	if raw == nil {
		return fmt.Errorf("no held copy for %s through %s", to, adapter)
	}
	return f.srv[to].Deliver(to, seedtest.Inbox, raw, at, "")
}

// rawCopy is a copy as the receiving server stores it, with its
// Authentication-Results and the Received line of the hand-over.
func rawCopy(p store.Provider, m providers.Message, at time.Time) []byte {
	domain := m.From.Address[strings.IndexByte(m.From.Address, '@')+1:]
	lines := []string{
		"Authentication-Results: mx.seeds.test; dkim=pass header.d=" + domain + " header.s=pt2026; spf=pass smtp.mailfrom=bounce." + domain + "; dmarc=pass header.from=" + domain,
		"Received: from mta." + p.Adapter + ".test (mta." + p.Adapter + ".test [198.51.100.7]) by mx.seeds.test with ESMTPS; " + at.UTC().Format(time.RFC1123Z),
		"From: " + m.From.String(),
		"To: " + m.To.Address,
		"Subject: " + m.Subject,
		"Message-ID: <" + m.ID.String() + "@" + domain + ">",
	}
	for k, v := range m.Headers {
		lines = append(lines, k+": "+v)
	}
	lines = append(lines, "Content-Type: text/plain; charset=utf-8", "", m.Text, "")
	return []byte(strings.Join(lines, "\r\n"))
}

// notes records the notifications.
type notes struct {
	mu       sync.Mutex
	finished []string
	failed   []string
}

func (n *notes) TestFinished(_ context.Context, _ *store.Workspace, t store.PlacementTest, summary string) {
	n.mu.Lock()
	n.finished = append(n.finished, t.Name+" | "+summary)
	n.mu.Unlock()
}

func (n *notes) SeedFailed(_ context.Context, _ *store.Workspace, s store.PlacementSeed) {
	n.mu.Lock()
	n.failed = append(n.failed, s.Email+" "+s.Status)
	n.mu.Unlock()
}

type fixture struct {
	env     *apptest.Env
	ws      store.Workspace
	ctx     context.Context
	svc     *placement.Service
	clock   *clock
	deliver *fakeDeliver
	notes   *notes
	prov    map[string]uuid.UUID
}

func setup(t *testing.T) *fixture {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	ws, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	linh, err := env.App.Q.GetUserByEmail(ctx, "linh@acme.io")
	if err != nil {
		t.Fatal(err)
	}
	p := &authz.Principal{Type: authz.ActorUser, ID: linh.ID, UserID: linh.ID, Name: linh.Name, WorkspaceID: ws.ID, RoleKey: authz.RoleOwner,
		Permissions: authz.NewSet(authz.PlacementRun, authz.PlacementManage, authz.DeliverabilityView)}
	c := &clock{t: time.Now().Truncate(time.Second)}
	f := &fixture{env: env, ws: ws, ctx: api.WithPrincipal(ctx, p), clock: c, notes: &notes{}, prov: map[string]uuid.UUID{}}
	f.deliver = &fakeDeliver{q: env.App.Q, clock: c, srv: map[string]*seedtest.Server{}, spam: map[string]string{}, held: map[string][]byte{},
		where: func(string, string) string { return "inbox" }}
	f.svc = &placement.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Deliver: f.deliver,
		Links: env.App.Links, URLs: env.App.URLs, Notify: f.notes, Inline: true, Guard: netguard.Policy{AllowPrivate: true},
		HTTP: &http.Client{Timeout: 10 * time.Second}, InsecureIMAP: true, Now: c.now}
	rows, err := env.App.Pool.Query(ctx, `SELECT adapter, id FROM providers WHERE workspace_id = $1 AND disconnected_at IS NULL`, ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var a string
		var id uuid.UUID
		if err := rows.Scan(&a, &id); err != nil {
			t.Fatal(err)
		}
		f.prov[a] = id
	}
	return f
}

// seed adds a seed mailbox on its own fake server.
func (f *fixture) seed(t *testing.T, provider, email, spam string, gmail bool) placement.PlacementSeedCheck {
	t.Helper()
	srv := seedtest.New(t, seedtest.Options{Gmail: gmail})
	// App passwords are shown in groups and typed with spaces; GMX and
	// Web.de take the account's own password as it is.
	pw := "abcdefghijklmnop"
	if provider == "gmx" || provider == "webde" {
		pw = "abcd efgh ijkl mnop"
	}
	srv.AddUser(email, pw, spam)
	f.deliver.srv[email], f.deliver.spam[email] = srv, spam
	out, err := f.svc.AddSeed(f.ctx, &f.ws, placement.PlacementSeedInput{Provider: provider, Email: email, Host: srv.Host, Port: srv.Port,
		Username: email, Password: "abcd efgh ijkl mnop"})
	if err != nil {
		t.Fatalf("add seed %s: %v", email, err)
	}
	return out
}

func (f *fixture) campaign(t *testing.T, name string) string {
	t.Helper()
	var id uuid.UUID
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT id FROM campaigns WHERE workspace_id = $1 AND name = $2 AND deleted_at IS NULL`, f.ws.ID, name).Scan(&id); err != nil {
		t.Fatalf("campaign %s: %v", name, err)
	}
	return id.String()
}

func (f *fixture) audits(t *testing.T, action string) int {
	t.Helper()
	var n int
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = $2`, f.ws.ID, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func cell(d placement.PlacementTestDetail, email, provider string) placement.PlacementCopyView {
	for _, c := range d.Copies {
		if c.SeedEmail == email && c.ProviderID == provider {
			return c
		}
	}
	return placement.PlacementCopyView{}
}

func TestCompareFindsCopiesLateAndDeletesThem(t *testing.T) {
	f := setup(t)
	yahoo := f.seed(t, "yahoo", "pt-yahoo@seeds.test", "Bulk", false)
	gmail := f.seed(t, "gmail", "pt-gmail@seeds.test", "[Gmail]/Spam", true)
	gmx := f.seed(t, "gmx", "pt-gmx@seeds.test", "Spamverdacht", false)

	// Folder detection: the spam folder by \Junk, Gmail's tabs over X-GM-RAW.
	if got := folderRoles(yahoo.Folders); got != "inbox:Inbox spam:Bulk" {
		t.Fatalf("yahoo folders = %s", got)
	}
	if got := folderRoles(gmail.Folders); !strings.Contains(got, "tab:Promotions") || !strings.Contains(got, "spam:Spam") {
		t.Fatalf("gmail folders = %s", got)
	}
	if f.audits(t, "placement.seed_add") != 3 {
		t.Fatal("adding seeds isn't audited")
	}
	list, err := f.svc.Seeds(f.ctx, &f.ws)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := json.Marshal(list); strings.Contains(string(b), "ijkl") {
		t.Fatal("the seed list returns a password")
	}

	ses, postal := f.prov["ses"].String(), f.prov["postal"].String()
	f.deliver.where = func(adapter, to string) string {
		switch {
		case strings.HasPrefix(to, "pt-gmail"):
			return "promotions"
		case adapter == "postal" && strings.HasPrefix(to, "pt-yahoo"):
			return "spam"
		case adapter == "postal" && strings.HasPrefix(to, "pt-gmx"):
			return "drop"
		}
		return "inbox"
	}
	in := placement.PlacementRunInput{Content: placement.PlacementContentRef{Kind: "campaign", ID: f.campaign(t, "Weekly Digest #142")},
		Through: "compare", Providers: []string{ses, postal}, Seeds: []string{yahoo.Seed.ID, gmail.Seed.ID, gmx.Seed.ID}, Origin: "review"}
	plan, err := f.svc.Plan(f.ctx, &f.ws, in)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.CanStart || plan.Copies != 6 || len(plan.Costs) != 2 {
		t.Fatalf("plan = canStart %v, %d copies, problems %v, sender problem %q", plan.CanStart, plan.Copies, plan.Problems, plan.SenderProblem)
	}

	d, err := f.svc.Start(f.ctx, &f.ws, in)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != "sending" || d.Rounds != 2 || d.Counts.Copies != 6 || len(f.deliver.sent) != 3 {
		t.Fatalf("after start: status %s, rounds %d, copies %d, sent %d", d.Status, d.Rounds, d.Counts.Copies, len(f.deliver.sent))
	}
	if f.audits(t, "placement.test_start") != 1 {
		t.Fatal("starting isn't audited")
	}
	// Every copy carries its token, the bulk headers and tracked links.
	for _, m := range f.deliver.sent {
		if !strings.HasPrefix(m.Headers["X-ReputeCast-Test"], "pt-") || m.Headers["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" {
			t.Fatalf("copy headers = %v", m.Headers)
		}
	}
	id := uuid.MustParse(d.ID)

	// Two minutes later the second round goes out; then the test waits.
	f.clock.add(placement.CompareGap)
	if _, err := f.svc.SendDue(f.ctx, id); err != nil {
		t.Fatal(err)
	}
	if len(f.deliver.sent) != 6 {
		t.Fatalf("sent %d copies, want 6", len(f.deliver.sent))
	}
	d, _ = f.svc.Test(f.ctx, &f.ws, id)
	if d.Status != "waiting" || d.Counts.Found != 5 || d.Counts.Waiting != 1 {
		t.Fatalf("waiting: status %s, found %d, waiting %d", d.Status, d.Counts.Found, d.Counts.Waiting)
	}
	if c := cell(d, "pt-gmail@seeds.test", ses); c.Folder != "tab" || c.Tab != "Promotions" || c.ArriveMs == nil || *c.ArriveMs != 11000 || c.SPF != "pass" {
		t.Fatalf("gmail cell = %+v", c)
	}
	if c := cell(d, "pt-yahoo@seeds.test", postal); c.Folder != "spam" || c.FolderName != "Bulk" {
		t.Fatalf("yahoo through Postal = %+v", c)
	}

	// 15 minutes after the last hand-off the missing copy is Not arrived.
	f.clock.add(placement.WaitFor + time.Second)
	if err := f.svc.Check(f.ctx, id); err != nil {
		t.Fatal(err)
	}
	d, _ = f.svc.Test(f.ctx, &f.ws, id)
	if d.Status != "finished" || d.FinishReason != "timeout" || d.Counts.NotArrived != 1 {
		t.Fatalf("finished: status %s (%s), not arrived %d", d.Status, d.FinishReason, d.Counts.NotArrived)
	}
	if len(f.notes.finished) != 1 || !strings.Contains(f.notes.finished[0], "Inbox") {
		t.Fatalf("finished notifications = %v", f.notes.finished)
	}

	// It turns up 30 minutes after its hand-off: the result updates, late.
	f.clock.add(14 * time.Minute)
	if err := f.deliver.late("pt-gmx@seeds.test", "postal", f.clock.now()); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Check(f.ctx, id); err != nil {
		t.Fatal(err)
	}
	d, _ = f.svc.Test(f.ctx, &f.ws, id)
	if c := cell(d, "pt-gmx@seeds.test", postal); c.Status != "found" || !c.Late || c.Folder != "inbox" {
		t.Fatalf("late copy = %+v", c)
	}
	if d.Counts.NotArrived != 0 || d.Counts.Found != 6 {
		t.Fatalf("after the late copy: found %d, not arrived %d", d.Counts.Found, d.Counts.NotArrived)
	}

	// The copy detail shows the headers that matter.
	cd, err := f.svc.CopyDetail(f.ctx, &f.ws, id, uuid.MustParse(cell(d, "pt-yahoo@seeds.test", postal).ID))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cd.KeyHeaders, "Authentication-Results") || !strings.Contains(cd.KeyHeaders, "X-ReputeCast-Test") || !strings.Contains(cd.KeyHeaders, "198.51.100.7") ||
		cd.SPF == nil || cd.SPF.IP != "198.51.100.7" || cd.Path != "pt-yahoo@seeds.test › Bulk" {
		t.Fatalf("copy detail = %+v", cd)
	}

	// Deleting finds the copies again by token and removes only them.
	other := []byte("From: friend@example.com\r\nTo: pt-yahoo@seeds.test\r\nSubject: Hello\r\nMessage-ID: <hello@example.com>\r\n\r\nhi\r\n")
	_ = f.deliver.srv["pt-yahoo@seeds.test"].Deliver("pt-yahoo@seeds.test", seedtest.Inbox, other, f.clock.now(), "")
	d, err = f.svc.DeleteCopiesNow(f.ctx, &f.ws, id)
	if err != nil {
		t.Fatal(err)
	}
	if d.CopiesDeletedAt == nil || f.audits(t, "placement.copies_delete") != 1 {
		t.Fatal("deleting copies isn't recorded and audited")
	}
	y := f.deliver.srv["pt-yahoo@seeds.test"]
	if n := y.Count("pt-yahoo@seeds.test", seedtest.Inbox) + y.Count("pt-yahoo@seeds.test", "Bulk"); n != 1 {
		t.Fatalf("yahoo holds %d messages after deleting, want only the other one", n)
	}
	if _, err := f.svc.DeleteCopiesNow(f.ctx, &f.ws, id); err == nil {
		t.Fatal("deleting twice should be refused")
	}
}

func folderRoles(fs []placement.PlacementFolder) string {
	parts := []string{}
	for _, f := range fs {
		parts = append(parts, f.Role+":"+f.Label)
	}
	return strings.Join(parts, " ")
}

func TestSignInFailsDuringTestAndStop(t *testing.T) {
	f := setup(t)
	a := f.seed(t, "aol", "pt-aol@seeds.test", "Bulk", false)
	b := f.seed(t, "icloud", "pt-icloud@seeds.test", "Junk", false)
	f.deliver.where = func(_, to string) string {
		if strings.HasPrefix(to, "pt-aol") {
			return "drop"
		}
		return "inbox"
	}
	// The iCloud password stops working before the copies are looked for.
	f.deliver.srv["pt-icloud@seeds.test"].Refuse("pt-icloud@seeds.test")
	d, err := f.svc.Start(f.ctx, &f.ws, placement.PlacementRunInput{Content: placement.PlacementContentRef{Kind: "campaign", ID: f.campaign(t, "Weekly Digest #142")},
		Through: "provider", Providers: []string{f.prov["ses"].String()}, Seeds: []string{a.Seed.ID, b.Seed.ID}})
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.MustParse(d.ID)
	d, _ = f.svc.Test(f.ctx, &f.ws, id)
	if c := cell(d, "pt-icloud@seeds.test", f.prov["ses"].String()); c.Status != "not_checked" || c.Reason != placement.ReasonSignInFailed || c.ReasonAt == nil {
		t.Fatalf("icloud cell = %+v", c)
	}
	if len(f.notes.failed) != 1 || f.notes.failed[0] != "pt-icloud@seeds.test failed" {
		t.Fatalf("seed failure notifications = %v", f.notes.failed)
	}
	// A second round of checks doesn't notify again.
	_ = f.svc.Check(f.ctx, id)
	if len(f.notes.failed) != 1 {
		t.Fatalf("notified %d times", len(f.notes.failed))
	}

	// Stopping early: the AOL copy is Not arrived and not looked for any more.
	f.clock.add(6 * time.Minute)
	d, err = f.svc.Stop(f.ctx, &f.ws, id)
	if err != nil {
		t.Fatal(err)
	}
	if c := cell(d, "pt-aol@seeds.test", f.prov["ses"].String()); d.Status != "stopped" || d.FinishReason != "stopped" || c.Status != "not_arrived" || c.Reason != placement.ReasonStopped {
		t.Fatalf("stopped: %s (%s), aol %+v", d.Status, d.FinishReason, c)
	}
	if f.audits(t, "placement.test_stop") != 1 {
		t.Fatal("stopping isn't audited")
	}
	if _, err := f.svc.Stop(f.ctx, &f.ws, id); err == nil {
		t.Fatal("stopping twice should be refused")
	}
	// A failed seed is left out of the next test and shown as Not checked.
	plan, err := f.svc.Plan(f.ctx, &f.ws, placement.PlacementRunInput{Content: placement.PlacementContentRef{Kind: "campaign", ID: f.campaign(t, "Weekly Digest #142")},
		Through: "provider", Providers: []string{f.prov["ses"].String()}})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range plan.Seeds {
		if s.Email == "pt-icloud@seeds.test" && (s.Selected || s.Status != "failed") {
			t.Fatalf("failed seed in the plan = %+v", s)
		}
	}
}

// tokenServer is Google's token endpoint for one seed: it trades the code
// and refreshes, counting the refreshes.
type tokenServer struct {
	mu        sync.Mutex
	refreshes int
	email     string
}

func (s *tokenServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	w.Header().Set("Content-Type", "application/json")
	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		if r.PostForm.Get("code") != "code-1" || r.PostForm.Get("code_verifier") == "" || r.PostForm.Get("client_secret") != "seed-app-secret" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		payload := base64.RawURLEncoding.EncodeToString([]byte(`{"email":"` + s.email + `"}`))
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok-1", "refresh_token": "ref-1", "token_type": "Bearer", "expires_in": 1,
			"id_token": "eyJhbGciOiJub25lIn0." + payload + "."})
	case "refresh_token":
		s.mu.Lock()
		s.refreshes++
		s.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok-2", "token_type": "Bearer", "expires_in": 3600})
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

type apps struct{}

func (apps) MailApp(_ context.Context, kind string) (placement.MailApp, bool, error) {
	if kind != "google" {
		return placement.MailApp{}, false, nil
	}
	return placement.MailApp{Kind: kind, ClientID: "seed-app", ClientSecret: "seed-app-secret"}, true, nil
}

func TestOAuthSeedAndTokenRefresh(t *testing.T) {
	f := setup(t)
	const email = "pt-workspace@seeds.test"
	srv := seedtest.New(t, seedtest.Options{Gmail: true})
	srv.AddUser(email, "unused", "[Gmail]/Spam")
	srv.AllowToken("tok-2", email)
	ts := &tokenServer{email: email}
	tok := httptest.NewServer(ts)
	defer tok.Close()
	f.svc.Apps = apps{}
	f.svc.OAuthEndpoints = &placement.Endpoints{GoogleAuth: tok.URL + "/auth", GoogleToken: tok.URL + "/token"}
	f.svc.IMAPHost, f.svc.IMAPPort = srv.Host, srv.Port

	start, err := f.svc.StartOAuth(f.ctx, &f.ws, placement.PlacementOAuthInput{Provider: "google_workspace", Email: email})
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(start.URL)
	q := u.Query()
	if q.Get("code_challenge_method") != "S256" || q.Get("access_type") != "offline" || q.Get("login_hint") != email || !strings.Contains(q.Get("scope"), "https://mail.google.com/") {
		t.Fatalf("consent URL = %s", start.URL)
	}
	linh, _ := f.env.App.Q.GetUserByEmail(context.Background(), "linh@acme.io")
	marcus, _ := f.env.App.Q.GetUserByEmail(context.Background(), "marcus@acme.io")
	// Someone else can't finish it; the state is single use.
	if loc := f.svc.OAuthCallback(context.Background(), &marcus, "", q.Get("state"), "code-1", "", ""); !strings.Contains(loc, "expired") {
		t.Fatalf("someone else's callback went to %s", loc)
	}
	start, _ = f.svc.StartOAuth(f.ctx, &f.ws, placement.PlacementOAuthInput{Provider: "google_workspace", Email: email})
	u, _ = url.Parse(start.URL)
	loc := f.svc.OAuthCallback(context.Background(), &linh, "", u.Query().Get("state"), "code-1", "", "")
	if !strings.Contains(loc, "/deliverability/placement/seeds?") || !strings.Contains(loc, "added=") {
		t.Fatalf("callback went to %s", loc)
	}
	// The first access token had expired: it was refreshed once, sealed,
	// and the next sign-in uses the refreshed one.
	if ts.refreshes != 1 {
		t.Fatalf("refreshes = %d, want 1", ts.refreshes)
	}
	list, _ := f.svc.Seeds(f.ctx, &f.ws)
	var seedID string
	for _, s := range list.Seeds {
		if s.Email == email {
			seedID = s.ID
			if s.Auth != "oauth" || s.Status != "connected" || !strings.Contains(folderRoles(s.Folders), "tab:Promotions") {
				t.Fatalf("oauth seed = %+v", s)
			}
		}
	}
	if seedID == "" {
		t.Fatal("the OAuth seed wasn't added")
	}
	f.deliver.srv[email], f.deliver.spam[email] = srv, "[Gmail]/Spam"
	if _, err := f.svc.Start(f.ctx, &f.ws, placement.PlacementRunInput{Content: placement.PlacementContentRef{Kind: "campaign", ID: f.campaign(t, "Weekly Digest #142")},
		Through: "provider", Providers: []string{f.prov["ses"].String()}, Seeds: []string{seedID}}); err != nil {
		t.Fatal(err)
	}
	if ts.refreshes != 1 {
		t.Fatalf("refreshes = %d after a test, want the sealed token reused", ts.refreshes)
	}
	if f.audits(t, "placement.seed_add") != 1 {
		t.Fatal("the OAuth seed isn't audited")
	}
}

func TestPermissionsOverHTTP(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	base := "/workspaces/acme-newsletter/placement"
	daniel := env.Client() // Content Writer: sees tests, can't run them
	daniel.SignInDemo("daniel@acme.io")
	daniel.Do("GET", base+"/tests", nil).MustStatus(t, 200)
	daniel.Do("GET", base+"/seeds", nil).MustStatus(t, 200)
	daniel.Do("POST", base+"/tests", map[string]any{"content": map[string]any{"kind": "campaign", "id": uuid.NewString()}, "through": "route"}).MustStatus(t, 403)
	daniel.Do("POST", base+"/seeds", map[string]any{"provider": "yahoo", "email": "x@seeds.test", "password": "secret"}).MustStatus(t, 403)
	sofia := env.Client() // Editor: runs tests, can't manage seeds
	sofia.SignInDemo("sofia@acme.io")
	sofia.Do("POST", base+"/seeds", map[string]any{"provider": "yahoo", "email": "x@seeds.test", "password": "secret"}).MustStatus(t, 403)
	sofia.Do("PUT", base+"/settings", map[string]any{"deleteAfter": "never"}).MustStatus(t, 403)
	var tests placement.PlacementTestList
	sofia.Do("GET", base+"/tests", nil).MustStatus(t, 200).JSON(t, &tests)
	if !tests.CanRun || tests.CanManage || len(tests.ManageHelpers) == 0 {
		t.Fatalf("sofia's list: canRun %v canManage %v helpers %v", tests.CanRun, tests.CanManage, tests.ManageHelpers)
	}
	// The OAuth callback needs the person who started it.
	anon := env.Client()
	r := anon.Do("GET", "/placement/oauth/callback?state=nope&code=x", nil)
	if r.Status != http.StatusFound || !strings.HasPrefix(r.Header.Get("Location"), "/sign-in?next=") {
		t.Fatalf("anonymous callback = %d %s", r.Status, r.Header.Get("Location"))
	}
}

var hrefPattern = regexp.MustCompile(`href="([^"]+/t/c/[^"]+)"`)

// A copy's links work like a real send's, but nobody is unsubscribed and
// its clicks are never counted.
func TestCopyLinksNeverCount(t *testing.T) {
	f := setup(t)
	s := f.seed(t, "yahoo", "pt-links@seeds.test", "Bulk", false)
	if _, err := f.svc.Start(f.ctx, &f.ws, placement.PlacementRunInput{Content: placement.PlacementContentRef{Kind: "campaign", ID: f.campaign(t, "Weekly Digest #142")},
		Through: "provider", Providers: []string{f.prov["ses"].String()}, Seeds: []string{s.Seed.ID}}); err != nil {
		t.Fatal(err)
	}
	m := f.deliver.sent[0]
	web := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	unsub := strings.TrimPrefix(strings.SplitN(m.Headers["List-Unsubscribe"], ">", 2)[0], "<")
	// call makes a request and returns its status, Location and body.
	call := func(method, u, form string) (int, string, string) {
		t.Helper()
		var body io.Reader
		if form != "" {
			body = strings.NewReader(form)
		}
		req, _ := http.NewRequestWithContext(context.Background(), method, u, body)
		if form != "" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		res, err := web.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = res.Body.Close() }()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, res.Header.Get("Location"), string(b)
	}
	if code, _, body := call(http.MethodGet, unsub, ""); code != http.StatusOK || !strings.Contains(body, "This is a placement test copy. Nobody was unsubscribed.") {
		t.Fatalf("unsubscribe page = %d", code)
	}
	if code, _, body := call(http.MethodPost, unsub, "List-Unsubscribe=One-Click"); code != http.StatusOK || !strings.Contains(body, "Nobody was unsubscribed") {
		t.Fatalf("one-click unsubscribe = %d %s", code, body)
	}
	var unsubscribed int
	_ = f.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM delivery_events WHERE message_id = $1 AND type = 'unsubscribe'`, m.ID).Scan(&unsubscribed)
	if unsubscribed != 0 {
		t.Fatalf("%d unsubscribes recorded", unsubscribed)
	}
	// The click redirects to the email's own link and isn't recorded.
	links := hrefPattern.FindStringSubmatch(m.HTML)
	if links == nil {
		t.Fatal("the copy has no tracked link")
	}
	code, loc, _ := call(http.MethodGet, strings.ReplaceAll(links[1], "&amp;", "&"), "")
	if code != http.StatusFound || !strings.HasPrefix(loc, "http") || strings.Contains(loc, "/t/c/") {
		t.Fatalf("click = %d %s", code, loc)
	}
	time.Sleep(300 * time.Millisecond) // engagement is recorded in the background
	var clicked *time.Time
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT clicked_at FROM messages WHERE id = $1`, m.ID).Scan(&clicked); err != nil {
		t.Fatal(err)
	}
	var events int
	_ = f.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM delivery_events WHERE message_id = $1 AND type IN ('open', 'click')`, m.ID).Scan(&events)
	if clicked != nil || events != 0 {
		t.Fatalf("the click was counted: clicked_at %v, %d events", clicked, events)
	}
}
