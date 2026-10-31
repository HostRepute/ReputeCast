package manage_test

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// fakeIPs is a test adapter whose API lists the account's dedicated IPs
// (providers.IPReporter): it answers what setReported says.
type fakeIPs struct{ fake }

var reported struct {
	sync.Mutex
	ips   providers.SendingIPs
	err   error
	calls int
}

func setReported(ips providers.SendingIPs, err error) {
	reported.Lock()
	reported.ips, reported.err = ips, err
	reported.Unlock()
}

func reportedCalls() int {
	reported.Lock()
	defer reported.Unlock()
	return reported.calls
}

func (fakeIPs) Adapter() string { return "fakeips" }
func (fakeIPs) Capabilities() providers.Capabilities {
	c := fake{}.Capabilities()
	c.ReportsIPs = true
	return c
}
func (fakeIPs) SendingIPs(context.Context) (providers.SendingIPs, error) {
	reported.Lock()
	defer reported.Unlock()
	reported.calls++
	return reported.ips, reported.err
}

func init() {
	providers.Register(providers.Descriptor{Adapter: "fakeips", Name: "Fake IP ESP", Kind: "api", Capabilities: fakeIPs{}.Capabilities(),
		Fields: []providers.Field{{Key: "token", Label: "API token", Secret: true, Required: true}},
		New:    func(providers.Config) (providers.Provider, error) { return fakeIPs{}, nil }})
}

// addReporting connects the fake IP-reporting adapter to the workspace, on
// the shared pool as connecting leaves it.
func addReporting(t *testing.T, env *apptest.Env, w store.Workspace) store.Provider {
	t.Helper()
	ctx := context.Background()
	id := uuid.Must(uuid.NewV7())
	creds, err := env.App.Secrets.Seal([]byte(`{"token":"t0ken"}`), manage.CredentialsAD(store.Provider{ID: id}))
	if err != nil {
		t.Fatal(err)
	}
	token, _ := authn.NewToken()
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, status, status_detail,
		ip_mode, ips, capabilities, limits, health, pricing, tracking, connected_at)
		VALUES ($1,$2,'fakeips','Fake IP ESP','test','{}',$3,$4,'healthy','','shared','{}','{}','{}','{}','{}','{}', now())`, id, w.ID, creds, token); err != nil {
		t.Fatal(err)
	}
	p, err := env.App.Q.ProvGetAny(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func reload(t *testing.T, env *apptest.Env, id uuid.UUID) store.Provider {
	t.Helper()
	p, err := env.App.Q.ProvGetAny(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// A provider whose API lists its dedicated IPs keeps what it answers
// (review GS-10): each IP once and in address order, dedicated while it
// lists some, shared when it lists none. A change is audited by the
// system; the same answer again isn't a change; a check that fails keeps
// the IPs and says why.
func TestRefreshIPs(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := demoWorkspace(t, env)
	now := time.Now().UTC().Truncate(time.Second)
	svc := service(env, func() time.Time { return now })
	p := addReporting(t, env, w)
	updates := func() int {
		return count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'provider.ips_update' AND target_id = $1 AND actor_type = 'system'`, p.ID.String())
	}

	setReported(providers.SendingIPs{Dedicated: true, IPs: []string{"198.51.100.20", "198.51.100.9", "198.51.100.20"}}, nil)
	out, err := svc.RefreshIPs(ctx, p, manage.IPsOnConnect)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"198.51.100.9", "198.51.100.20"}
	for _, got := range []store.Provider{out, reload(t, env, p.ID)} {
		if got.IpMode != "dedicated" || !slices.Equal(got.Ips, want) {
			t.Fatalf("ip_mode %s, ips %q; want dedicated %q", got.IpMode, got.Ips, want)
		}
		if src := state.IPSourceOf(got); src.From != state.IPsFromProvider || src.CheckedAt == nil || !src.CheckedAt.Equal(now) || src.Error != "" {
			t.Fatalf("ip_source %+v", src)
		}
	}
	if n := updates(); n != 1 {
		t.Fatalf("%d provider.ips_update entries by the system, want 1", n)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'provider.ips_update' AND target_id = $1
		AND before->>'ip_mode' = 'shared' AND after->>'ip_mode' = 'dedicated' AND after->'ips' = '["198.51.100.9","198.51.100.20"]'::jsonb
		AND metadata->>'trigger' = 'connect'`, p.ID.String()); n != 1 {
		t.Fatal("the audit entry doesn't say what changed and what asked")
	}

	// The same answer: nothing changed, nothing to audit.
	if _, err := svc.RefreshIPs(ctx, p, manage.IPsDaily); err != nil {
		t.Fatal(err)
	}
	if n := updates(); n != 1 {
		t.Fatalf("%d entries after the same answer, want 1", n)
	}

	// An outage keeps the IPs and says why; the card reads it.
	checked := now
	now = now.Add(24 * time.Hour)
	setReported(providers.SendingIPs{}, &providers.CallError{Op: "GET /ips", Status: 503, Temporary: true, Message: "Service Unavailable"})
	if _, err = svc.RefreshIPs(ctx, p, manage.IPsDaily); err != nil {
		t.Fatal(err)
	}
	got := reload(t, env, p.ID)
	src := state.IPSourceOf(got)
	if got.IpMode != "dedicated" || !slices.Equal(got.Ips, want) || src.Error != "Service Unavailable" || src.ErrorAt == nil || !src.CheckedAt.Equal(checked) {
		t.Fatalf("after a failed check: %s %q %+v", got.IpMode, got.Ips, src)
	}
	if n := updates(); n != 1 {
		t.Fatalf("a failed check was audited as a change (%d entries)", n)
	}
	card, err := svc.Card(ctx, &w, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if s := card.IPSource; s.From != "provider" || s.Editable || s.Error != "Service Unavailable" || s.CheckedAt == nil || !slices.Equal(card.IPs, want) {
		t.Fatalf("card %+v %q", s, card.IPs)
	}

	// No dedicated IPs any more: the shared pool, the error gone.
	setReported(providers.SendingIPs{IPs: []string{}}, nil)
	if _, err := svc.RefreshIPs(ctx, p, manage.IPsDaily); err != nil {
		t.Fatal(err)
	}
	got = reload(t, env, p.ID)
	if src := state.IPSourceOf(got); got.IpMode != "shared" || len(got.Ips) != 0 || src.Error != "" || !src.CheckedAt.Equal(now) {
		t.Fatalf("shared pool: %s %q %+v", got.IpMode, got.Ips, src)
	}
	if n := updates(); n != 2 {
		t.Fatalf("%d entries, want 2", n)
	}
}

// The daily job asks every connected provider whose API reports its IPs,
// never a sample provider (demo SES) nor one whose IPs are set by hand.
func TestIPsWorker(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := demoWorkspace(t, env)
	svc := service(env, time.Now)
	p := addReporting(t, env, w)
	ses, postal := providerByAdapter(t, env, "ses"), providerByAdapter(t, env, "postal")
	setReported(providers.SendingIPs{Dedicated: true, IPs: []string{"198.51.100.30"}}, nil)
	before := reportedCalls()
	if err := (&manage.IPsWorker{S: svc}).Work(ctx, &river.Job[manage.IPsArgs]{}); err != nil {
		t.Fatal(err)
	}
	if n := reportedCalls() - before; n != 1 {
		t.Fatalf("%d calls to the provider, want 1", n)
	}
	if got := reload(t, env, p.ID); got.IpMode != "dedicated" || !slices.Equal(got.Ips, []string{"198.51.100.30"}) {
		t.Fatalf("the reporting provider: %s %q", got.IpMode, got.Ips)
	}
	for _, x := range []store.Provider{ses, postal} {
		if got := reload(t, env, x.ID); string(got.IpSource) != string(x.IpSource) || got.IpMode != x.IpMode || !slices.Equal(got.Ips, x.Ips) {
			t.Fatalf("%s changed: %s %q %s", x.Name, got.IpMode, got.Ips, got.IpSource)
		}
	}
}

// People who manage providers set the sending IPs of a provider whose API
// doesn't list them (Postal): validated, sorted, audited, and shown with
// who set them. A provider whose API reports them is refused, and Viewers
// can do neither.
func TestSetProviderIPs(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	postal, ses := providerByAdapter(t, env, "postal"), providerByAdapter(t, env, "ses")
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	path := ws + "/providers/" + postal.ID.String() + "/ips"
	body := map[string]any{"mode": "dedicated", "ips": []string{" 203.0.113.26", "2001:DB8::25", "203.0.113.25", "203.0.113.26", ""}}

	viewer.Do("PUT", path, body).MustStatus(t, 403)
	var card struct {
		IPMode   string
		IPs      []string
		IPSource struct {
			From     string
			Editable bool
			SetBy    string
			SetAt    *time.Time
		}
	}
	owner.Do("PUT", path, body).MustStatus(t, 200).JSON(t, &card)
	want := []string{"203.0.113.25", "203.0.113.26", "2001:db8::25"}
	if card.IPMode != "dedicated" || !slices.Equal(card.IPs, want) || card.IPSource.From != "manual" || !card.IPSource.Editable ||
		card.IPSource.SetBy != "Linh Nguyen" || card.IPSource.SetAt == nil {
		t.Fatalf("card %+v", card)
	}
	if got := reload(t, env, postal.ID); !slices.Equal(got.Ips, want) {
		t.Fatalf("stored %q", got.Ips)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'provider.ips_update' AND target_id = $1 AND actor_type = 'user'
		AND actor_name = 'Linh Nguyen' AND before->'ips' = '["203.0.113.25"]'::jsonb AND after->'ips' = '["203.0.113.25","203.0.113.26","2001:db8::25"]'::jsonb
		AND metadata->>'source' = 'manual'`, postal.ID.String()); n != 1 {
		t.Fatalf("%d audit entries for the change, want 1", n)
	}

	for _, c := range []struct {
		body map[string]any
		say  string
	}{
		{map[string]any{"mode": "dedicated", "ips": []string{"10.0.0.5"}}, "private or reserved"},
		{map[string]any{"mode": "dedicated", "ips": []string{"203.0.113.25", "mta.acme.io"}}, "isn't an IP address"},
		{map[string]any{"mode": "dedicated", "ips": []string{"203.0.113.0/24"}}, "isn't an IP address"},
		{map[string]any{"mode": "dedicated", "ips": []string{" "}}, "at least one IP address"},
		{map[string]any{"mode": "dedicated", "ips": manyIPs(providers.MaxSendingIPs + 1)}, "up to 64"},
		{map[string]any{"mode": "shared", "ips": []string{"203.0.113.25"}}, "shared pool has no IPs"},
		{map[string]any{"mode": "pool"}, ""},
	} {
		r := owner.Do("PUT", path, c.body).MustStatus(t, 422)
		if !strings.Contains(string(r.Body), c.say) {
			t.Fatalf("%v: %s, want it to say %q", c.body, r.Body, c.say)
		}
	}
	// Up to 64 is fine.
	owner.Do("PUT", path, map[string]any{"mode": "dedicated", "ips": manyIPs(providers.MaxSendingIPs)}).MustStatus(t, 200)

	// Back to the shared pool.
	owner.Do("PUT", path, map[string]any{"mode": "shared"}).MustStatus(t, 200).JSON(t, &card)
	if card.IPMode != "shared" || len(card.IPs) != 0 || card.IPSource.From != "manual" {
		t.Fatalf("shared: %+v", card)
	}

	// Amazon SES lists its IPs itself: they aren't set by hand.
	r := owner.Do("PUT", ws+"/providers/"+ses.ID.String()+"/ips", map[string]any{"mode": "dedicated", "ips": []string{"203.0.113.40"}}).MustStatus(t, 409)
	if !strings.Contains(string(r.Body), "Amazon SES reports its sending IPs itself") {
		t.Fatalf("409 without the reason: %s", r.Body)
	}
	if got := reload(t, env, ses.ID); got.IpMode != ses.IpMode || !slices.Equal(got.Ips, ses.Ips) {
		t.Fatal("a refused change changed the IPs")
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'provider.ips_update' AND target_id = $1`, ses.ID.String()); n != 0 {
		t.Fatalf("%d audit entries for a refused change", n)
	}
}

func manyIPs(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "198.51.100." + strconv.Itoa(i+1)
	}
	return out
}

// "Check again" asks a provider whose API reports its IPs now, audited; a
// provider whose IPs are set by hand has nothing to ask.
func TestCheckProviderIPs(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	w := demoWorkspace(t, env)
	p := addReporting(t, env, w)
	postal := providerByAdapter(t, env, "postal")
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	path := ws + "/providers/" + p.ID.String() + "/ips/check"

	setReported(providers.SendingIPs{Dedicated: true, IPs: []string{"198.51.100.41"}}, nil)
	viewer.Do("POST", path, nil).MustStatus(t, 403)
	var card struct {
		IPMode   string
		IPs      []string
		IPSource struct{ From, Error string }
	}
	owner.Do("POST", path, nil).MustStatus(t, 200).JSON(t, &card)
	if card.IPMode != "dedicated" || !slices.Equal(card.IPs, []string{"198.51.100.41"}) || card.IPSource.From != "provider" || card.IPSource.Error != "" {
		t.Fatalf("card %+v", card)
	}
	setReported(providers.SendingIPs{}, &providers.CallError{Op: "GET /ips", Status: 403, Auth: true, Message: "Key without access to IP addresses"})
	owner.Do("POST", path, nil).MustStatus(t, 200).JSON(t, &card)
	if !slices.Equal(card.IPs, []string{"198.51.100.41"}) || card.IPSource.Error != "Key without access to IP addresses" {
		t.Fatalf("a refused check: %+v", card)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'provider.ips_check' AND target_id = $1 AND actor_type = 'user'`, p.ID.String()); n != 2 {
		t.Fatalf("%d provider.ips_check entries, want 2", n)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'provider.ips_check' AND target_id = $1 AND metadata->>'result' = 'error'`, p.ID.String()); n != 1 {
		t.Fatal("the refused check isn't recorded as one")
	}
	owner.Do("POST", ws+"/providers/"+postal.ID.String()+"/ips/check", nil).MustStatus(t, 409)
}
