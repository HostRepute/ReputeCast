package forms_test

// Custom domains and the hosts other workspaces use: a
// CNAME or matching addresses show that a host points at this server, not
// whose host it is, so a workspace may only have a host no other workspace
// used first, and the instance's own hosts need the TXT record.

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	rcapi "github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/store"
)

// instanceURL is the public URL of domainService's instance.
const instanceURL = "https://reputecast.example.com"

// domainService is a forms service on env's database that resolves names
// from zone and lives at instanceURL.
func domainService(env *apptest.Env, zone dnscheck.Resolver) *forms.Service {
	a := env.App
	return &forms.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Tokens: a.Links, URLs: publicurl.New(instanceURL), DNS: zone,
		Audience: &audience.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Mail: a.Mail, Links: a.Links, URLs: a.URLs},
	}
}

func workspaceBySlug(t *testing.T, env *apptest.Env, slug string) *store.Workspace {
	t.Helper()
	ws, err := env.App.Q.GetWorkspaceBySlug(context.Background(), slug)
	if err != nil {
		t.Fatal(err)
	}
	return &ws
}

func exec(t *testing.T, env *apptest.Env, sql string, args ...any) {
	t.Helper()
	if _, err := env.App.Pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// shortLinks gives ws a verified SMS short-link domain, verified at.
func shortLinks(t *testing.T, env *apptest.Env, ws uuid.UUID, host string, at time.Time) {
	t.Helper()
	exec(t, env, `INSERT INTO sms_settings (workspace_id, short_domain, short_domain_status, short_domain_verified_at, updated_at)
		VALUES ($1, $2, 'verified', $3, now())
		ON CONFLICT (workspace_id) DO UPDATE SET short_domain = excluded.short_domain, short_domain_status = 'verified',
			short_domain_verified_at = excluded.short_domain_verified_at`, ws, host, at)
}

func wantProblem(t *testing.T, err error, status int, detail string) {
	t.Helper()
	var p *rcapi.Problem
	if !errors.As(err, &p) || p.Status != status || p.Detail != detail {
		t.Fatalf("got %v, want %d %q", err, status, detail)
	}
}

// A workspace can't take a host another workspace uses: not for its pages,
// its short links or its link tracking, whichever came first.
func TestHostsOfAnotherWorkspace(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	eng := signedIn(env, "linh@acme.io")
	engAPI := "/workspaces/acme-engineering-blog"
	audited := audits(t, env, "public_pages.domain") + audits(t, env, "sms_short_domain.set") + audits(t, env, "domain.update")
	// Acme Newsletter's hosts: its pages (news.acme.io), its short links
	// (go.acme.io) and its link tracking (go.news.acme.io, not pointed yet).
	for _, host := range []string{"news.acme.io", "go.acme.io", "go.news.acme.io"} {
		p := eng.Do(http.MethodPut, engAPI+"/public-pages/domain", map[string]any{"domain": host}).MustStatus(t, 409).Problem(t)
		if p["detail"] != host+" is already used by another workspace on this ReputeCast, for its pages or its links. Remove it there first." {
			t.Errorf("%s as the pages' domain: %v", host, p["detail"])
		}
		p = eng.Do(http.MethodPut, engAPI+"/sms/short-domain", map[string]any{"domain": host}).MustStatus(t, 409).Problem(t)
		if p["detail"] != host+" is already used by another workspace on this ReputeCast, for its pages or its links. Remove it there first." {
			t.Errorf("%s as the short-link domain: %v", host, p["detail"])
		}
	}
	// A tracking domain sits under a sending domain of the workspace.
	sending := uuid.Must(uuid.NewV7())
	exec(t, env, `INSERT INTO sending_domains (id, workspace_id, domain) VALUES ($1, $2, 'acme.io')`, sending, workspaceBySlug(t, env, "acme-engineering-blog").ID)
	for _, host := range []string{"news.acme.io", "go.acme.io"} {
		p := eng.Do(http.MethodPatch, engAPI+"/domains/"+sending.String(), map[string]any{"trackingDomain": host}).MustStatus(t, 409).Problem(t)
		if p["detail"] != host+" is already used by another workspace on this ReputeCast, for its pages or its links. Remove it there first." {
			t.Errorf("%s as a tracking domain: %v", host, p["detail"])
		}
	}
	var tracking string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT tracking_domain FROM sending_domains WHERE id = $1`, sending).Scan(&tracking); err != nil || tracking != "" {
		t.Errorf("tracking domain %q (%v) after the refusals", tracking, err)
	}
	if n := audits(t, env, "public_pages.domain") + audits(t, env, "sms_short_domain.set") + audits(t, env, "domain.update"); n != audited {
		t.Errorf("%d changes audited, want %d: nothing changed", n, audited)
	}
}

// The M11 attack: a host that CNAMEs to this server for another workspace
// (its short links) doesn't verify as this workspace's pages, and a domain
// verified that way before the check existed stops at its next check,
// without the second check a DNS hiccup gets. The TXT record, whose value
// is this workspace's alone, still proves a host is its own.
func TestDomainOfAnotherWorkspaceDoesntVerify(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	acme, eng := workspaceBySlug(t, env, "acme-newsletter"), workspaceBySlug(t, env, "acme-engineering-blog")
	zone := dnscheck.NewZone()
	zone.Add("reputecast.example.com", "A", "203.0.113.10")
	zone.Add("go.blog.example", "CNAME", "reputecast.example.com.")
	svc := domainService(env, zone)
	shortLinks(t, env, eng.ID, "go.blog.example", time.Now().Add(-48*time.Hour))

	_, err := svc.SetDomain(ctx, acme, forms.PublicDomainInput{Domain: "go.blog.example"})
	wantProblem(t, err, 409, "go.blog.example is already used by another workspace on this ReputeCast, for its pages or its links. Remove it there first.")

	// Verified through its CNAME before the check existed.
	exec(t, env, `UPDATE public_page_settings SET custom_domain = 'go.blog.example', domain_status = 'verified', domain_verified_at = now() - interval '1 hour',
		domain_detail = 'CNAME points at reputecast.example.com.', texts = '{}' WHERE workspace_id = $1`, acme.ID)
	if got := svc.WorkspaceForHost(ctx, "go.blog.example"); got != "acme-newsletter" {
		t.Fatalf("before the check the host serves %q", got)
	}
	out, err := svc.CheckDomain(ctx, acme)
	if err != nil {
		t.Fatal(err)
	}
	if out.Domain.Status != "failed" || out.Domain.Detail != "Another workspace on this ReputeCast uses go.blog.example already, so its CNAME and addresses don't show it's yours." {
		t.Fatalf("a host another workspace used first: %+v", out.Domain)
	}
	if svc.WorkspaceForHost(ctx, "go.blog.example") != "" || svc.HostVerified(ctx, "go.blog.example") {
		t.Fatal("the pages still answer on the other workspace's host, and the TLS ask endpoint says yes")
	}
	// Matching addresses don't count either.
	zone2 := dnscheck.NewZone()
	zone2.Add("reputecast.example.com", "A", "203.0.113.10")
	zone2.Add("go.blog.example", "A", "203.0.113.10")
	if out, err = domainService(env, zone2).CheckDomain(ctx, acme); err != nil || out.Domain.Status != "failed" {
		t.Fatalf("matching addresses: %+v %v", out.Domain, err)
	}

	// The TXT record proves the host is Acme Newsletter's after all.
	zone.Add("_reputecast.go.blog.example", "TXT", out.Domain.TXTValue)
	if out, err = svc.CheckDomain(ctx, acme); err != nil || out.Domain.Status != "verified" || out.Domain.Detail != "TXT record found." {
		t.Fatalf("with the TXT record: %+v %v", out.Domain, err)
	}
}

// The first workspace to verify a host keeps it: another workspace using
// it later (through a path that didn't check, as before) takes nothing
// away. Its own uses don't count against it.
func TestDomainVerifiedFirstIsKept(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	acme, eng := workspaceBySlug(t, env, "acme-newsletter"), workspaceBySlug(t, env, "acme-engineering-blog")
	zone := dnscheck.NewZone()
	zone.Add("reputecast.example.com", "A", "203.0.113.10")
	zone.Add("first.example", "CNAME", "reputecast.example.com.")
	svc := domainService(env, zone)
	// Acme Newsletter's own short links on the host don't stand in its way.
	shortLinks(t, env, acme.ID, "first.example", time.Now().Add(-48*time.Hour))
	out, err := svc.SetDomain(ctx, acme, forms.PublicDomainInput{Domain: "first.example"})
	if err != nil || out.Domain.Status != "verified" || out.Domain.Detail != "CNAME points at reputecast.example.com." {
		t.Fatalf("set: %+v %v", out.Domain, err)
	}
	exec(t, env, `UPDATE public_page_settings SET domain_verified_at = now() - interval '1 hour' WHERE workspace_id = $1`, acme.ID)
	shortLinks(t, env, eng.ID, "first.example", time.Now())
	if out, err = svc.CheckDomain(ctx, acme); err != nil || out.Domain.Status != "verified" {
		t.Fatalf("a later use by another workspace took the domain: %+v %v", out.Domain, err)
	}
	// Another workspace's unproven use doesn't either.
	exec(t, env, `UPDATE sms_settings SET short_domain_status = 'pending', short_domain_verified_at = NULL WHERE workspace_id = $1`, eng.ID)
	if out, err = svc.CheckDomain(ctx, acme); err != nil || out.Domain.Status != "verified" {
		t.Fatalf("an unproven use took the domain: %+v %v", out.Domain, err)
	}
	// While a domain waits for its DNS, another workspace's use keeps it
	// waiting, and says why.
	exec(t, env, `UPDATE public_page_settings SET domain_status = 'pending', domain_verified_at = NULL WHERE workspace_id = $1`, acme.ID)
	if out, err = svc.CheckDomain(ctx, acme); err != nil || out.Domain.Status != "pending" ||
		out.Domain.Detail != "Another workspace on this ReputeCast uses first.example already, so its CNAME and addresses don't show it's yours." {
		t.Fatalf("pending: %+v %v", out.Domain, err)
	}
}

// The instance's own hosts: its address is never a workspace's, nor its
// SMTP relay's, and a host under its address needs the TXT record.
func TestDomainsOfTheInstance(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	acme := workspaceBySlug(t, env, "acme-newsletter")
	zone := dnscheck.NewZone()
	zone.Add("reputecast.example.com", "A", "203.0.113.10")
	zone.Add("acme.reputecast.example.com", "CNAME", "reputecast.example.com.")
	zone.Add("smtp.example.org", "A", "203.0.113.10")
	svc := domainService(env, zone)

	own := "That's this ReputeCast's own address. Use a domain of your newsletter, like news.acme.io."
	_, err := svc.SetDomain(ctx, acme, forms.PublicDomainInput{Domain: "reputecast.example.com"})
	wantProblem(t, err, 422, own)
	exec(t, env, `INSERT INTO smtp_relay (id, hostname) VALUES (true, 'smtp.example.org') ON CONFLICT (id) DO UPDATE SET hostname = 'smtp.example.org'`)
	_, err = svc.SetDomain(ctx, acme, forms.PublicDomainInput{Domain: "smtp.example.org"})
	wantProblem(t, err, 422, own)

	// A wildcard CNAME under the instance's address points every name here.
	out, err := svc.SetDomain(ctx, acme, forms.PublicDomainInput{Domain: "acme.reputecast.example.com"})
	if err != nil || out.Domain.Status != "pending" ||
		out.Domain.Detail != "acme.reputecast.example.com is under reputecast.example.com, this ReputeCast's own address, so its CNAME and addresses don't show it's yours." {
		t.Fatalf("under the instance's address: %+v %v", out.Domain, err)
	}
	zone.Add("_reputecast.acme.reputecast.example.com", "TXT", out.Domain.TXTValue)
	if out, err = svc.CheckDomain(ctx, acme); err != nil || out.Domain.Status != "verified" || out.Domain.Detail != "TXT record found." {
		t.Fatalf("with the TXT record: %+v %v", out.Domain, err)
	}
	if svc.WorkspaceForHost(ctx, "acme.reputecast.example.com") != "acme-newsletter" {
		t.Error("the TXT-verified host doesn't serve the pages")
	}
}
