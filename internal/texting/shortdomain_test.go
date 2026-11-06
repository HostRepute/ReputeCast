package texting_test

// The short-link domain and the hosts other workspaces use: a CNAME shows
// that a host points at this server, not whose host it is, so every check
// (the manual one and the daily re-check) leaves a host with the workspace
// that verified it first, unless another has the TXT record that is its
// alone.

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/customhost"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/texting"
)

// shortInstance is the public URL of shortService's instance.
const shortInstance = "https://reputecast.example.com"

// shortService is the SMS domain on env's database at shortInstance,
// resolving names from zone.
func shortService(env *apptest.Env, zone dnscheck.Resolver) *texting.Service {
	s, _, _ := service(env, time.Now)
	s.URLs, s.DNS = publicurl.New(shortInstance), zone
	return s
}

type shortState struct {
	Status     string
	Detail     string
	VerifiedAt *time.Time
	CheckedAt  *time.Time
}

func shortOf(t *testing.T, env *apptest.Env, ws uuid.UUID) shortState {
	t.Helper()
	var s shortState
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT short_domain_status, short_domain_detail, short_domain_verified_at, short_domain_checked_at
		FROM sms_settings WHERE workspace_id = $1`, ws).Scan(&s.Status, &s.Detail, &s.VerifiedAt, &s.CheckedAt); err != nil {
		t.Fatal(err)
	}
	return s
}

func recheckShortDomains(t *testing.T, svc *texting.Service) {
	t.Helper()
	if err := (&texting.ShortDomainsWorker{S: svc}).Work(context.Background(), &river.Job[texting.ShortDomainsArgs]{}); err != nil {
		t.Fatal(err)
	}
}

// The M11 attack on short links: two workspaces stored the same host as
// their short-link domain before it had to be free. The re-check leaves
// the host with the workspace that verified it first and stops the other
// at once, without the TXT record that only it can publish; with the
// record, its links answer there again. The first holder keeps the host
// throughout.
func TestShortLinkDomainHeldFirstElsewhere(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	acme, eng := workspaceNamed(t, env, "acme-newsletter"), workspaceNamed(t, env, "acme-engineering-blog")
	zone := dnscheck.NewZone()
	zone.Add("reputecast.example.com", "A", "203.0.113.10")
	zone.Add("go.blog.example", "CNAME", "reputecast.example.com.")
	svc := shortService(env, zone)
	first := time.Now().Add(-48 * time.Hour).Truncate(time.Microsecond)
	shortDomain(t, env, eng.ID, "go.blog.example", first)
	shortDomain(t, env, acme.ID, "go.blog.example", time.Now().Add(-time.Hour))
	shortLink(t, env, acme.ID, "Acme02", "https://acme.example/sale")
	shortLink(t, env, eng.ID, "Engi02", "https://blog.example/post")

	// The daily re-check, due at once after the upgrade.
	recheckShortDomains(t, svc)
	got := shortOf(t, env, acme.ID)
	if got.Status != "failed" || got.VerifiedAt != nil || got.CheckedAt == nil ||
		got.Detail != "Another workspace on this ReputeCast uses go.blog.example already, so its CNAME doesn't show it's yours." {
		t.Fatalf("the second workspace after the re-check: %+v", got)
	}
	if held := shortOf(t, env, eng.ID); held.Status != "verified" || held.VerifiedAt == nil || !held.VerifiedAt.Equal(first) || held.CheckedAt == nil {
		t.Fatalf("the first holder after the re-check: %+v", held)
	}
	var actor, before, after string
	if err := env.App.Pool.QueryRow(ctx, `SELECT actor_type, before->>'status', after->>'status' FROM audit_events
		WHERE action = 'sms_short_domain.checked' AND workspace_id = $1 ORDER BY occurred_at DESC LIMIT 1`, acme.ID).Scan(&actor, &before, &after); err != nil ||
		actor != "system" || before != "verified" || after != "failed" {
		t.Fatalf("audit: %s %s → %s (%v)", actor, before, after, err)
	}
	if status, to := follow(t, env, "go.blog.example", "/Acme02"); status != http.StatusNotFound {
		t.Fatalf("the second workspace's link still answers on the host: %d to %q", status, to)
	}
	if status, to := follow(t, env, "go.blog.example", "/Engi02"); status != http.StatusFound || to != "https://blog.example/post" {
		t.Fatalf("the first holder's link: %d to %q", status, to)
	}

	// Routing & rules shows the TXT record that would prove the host is
	// Acme Newsletter's; the first holder needs none.
	rules, err := svc.Rules(ctx, acme, nil)
	if err != nil {
		t.Fatal(err)
	}
	if d := rules.ShortDomain; d.TXTName != "_reputecast.go.blog.example" || d.TXTValue != customhost.Token(acme.ID, "go.blog.example") {
		t.Fatalf("the TXT record to add: %+v", d)
	}
	if rules, err = svc.Rules(ctx, eng, nil); err != nil || rules.ShortDomain.TXTName != "" || rules.ShortDomain.TXTValue != "" {
		t.Fatalf("the first holder's record: %+v %v", rules.ShortDomain, err)
	}

	// Another workspace's TXT value doesn't count; this workspace's does.
	zone.Add("_reputecast.go.blog.example", "TXT", customhost.Token(eng.ID, "go.blog.example"))
	if rules, err = svc.CheckShortDomain(ctx, acme, nil); err != nil || rules.ShortDomain.Status != "failed" {
		t.Fatalf("with the other workspace's TXT value: %+v %v", rules.ShortDomain, err)
	}
	zone.Add("_reputecast.go.blog.example", "TXT", customhost.Token(acme.ID, "go.blog.example"))
	if rules, err = svc.CheckShortDomain(ctx, acme, nil); err != nil || rules.ShortDomain.Status != "verified" ||
		rules.ShortDomain.Detail != "TXT record found, and the CNAME points at reputecast.example.com." {
		t.Fatalf("with its TXT record: %+v %v", rules.ShortDomain, err)
	}
	if status, to := follow(t, env, "go.blog.example", "/Acme02"); status != http.StatusFound || to != "https://acme.example/sale" {
		t.Fatalf("the TXT record didn't bring the link back: %d to %q", status, to)
	}
	// Verified later, Acme Newsletter takes nothing from the first holder.
	run(t, env, `UPDATE sms_settings SET short_domain_checked_at = NULL WHERE workspace_id = $1`, eng.ID)
	recheckShortDomains(t, svc)
	if held := shortOf(t, env, eng.ID); held.Status != "verified" || held.Detail != "CNAME points at reputecast.example.com." {
		t.Fatalf("the first holder after the second proved its own: %+v", held)
	}
}

// The checks of a host nobody else uses work as before; the instance's
// own address is never a workspace's, and a host under it needs the TXT
// record.
func TestShortLinkDomainChecks(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	acme := workspaceNamed(t, env, "acme-newsletter")
	zone := dnscheck.NewZone()
	zone.Add("reputecast.example.com", "A", "203.0.113.10")
	zone.Add("go.acme.io", "CNAME", "reputecast.example.com.")
	zone.Add("links.reputecast.example.com", "CNAME", "reputecast.example.com.")
	svc := shortService(env, zone)

	rules, err := svc.CheckShortDomain(ctx, acme, nil)
	if err != nil || rules.ShortDomain.Status != "verified" || rules.ShortDomain.Detail != "CNAME points at reputecast.example.com." || rules.ShortDomain.TXTValue != "" {
		t.Fatalf("a host nobody else uses: %+v %v", rules.ShortDomain, err)
	}
	// Checked again a day later, it stays verified since the same moment.
	since := shortOf(t, env, acme.ID).VerifiedAt
	run(t, env, `UPDATE sms_settings SET short_domain_checked_at = now() - interval '25 hours' WHERE workspace_id = $1`, acme.ID)
	recheckShortDomains(t, svc)
	if got := shortOf(t, env, acme.ID); got.Status != "verified" || got.VerifiedAt == nil || !got.VerifiedAt.Equal(*since) || time.Since(*got.CheckedAt) > time.Minute {
		t.Fatalf("checked again: %+v", got)
	}

	rules, err = svc.SetShortDomain(ctx, acme, nil, texting.SmsShortDomainInput{Domain: "links.reputecast.example.com"})
	if err != nil || rules.ShortDomain.Status != "pending" || rules.ShortDomain.TXTName != "_reputecast.links.reputecast.example.com" ||
		rules.ShortDomain.Detail != "links.reputecast.example.com is under reputecast.example.com, this ReputeCast's own address, so its CNAME doesn't show it's yours." {
		t.Fatalf("under the instance's address: %+v %v", rules.ShortDomain, err)
	}
	zone.Add("_reputecast.links.reputecast.example.com", "TXT", rules.ShortDomain.TXTValue)
	if rules, err = svc.CheckShortDomain(ctx, acme, nil); err != nil || rules.ShortDomain.Status != "verified" {
		t.Fatalf("with the TXT record: %+v %v", rules.ShortDomain, err)
	}

	rules, err = svc.SetShortDomain(ctx, acme, nil, texting.SmsShortDomainInput{Domain: "reputecast.example.com"})
	if err != nil || rules.ShortDomain.Status != "pending" || rules.ShortDomain.TXTValue != "" ||
		rules.ShortDomain.Detail != "reputecast.example.com is one of this ReputeCast's own addresses. Use a domain of your own instead." {
		t.Fatalf("the instance's own address: %+v %v", rules.ShortDomain, err)
	}
}
