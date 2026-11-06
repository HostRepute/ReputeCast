package domains_test

// Link tracking domains and the hosts other workspaces use: a CNAME shows
// that a host points at this server, not whose host it is, so every check
// (the domain's own and the periodic re-check) leaves a host with the
// workspace that verified it first, unless another has the TXT record that
// is its alone. A refused domain stops carrying links and the open pixel.

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/customhost"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/domains"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/store"
)

// trackingInstance is the public URL of trackingService's instance.
const trackingInstance = "https://reputecast.example.com"

// trackingService is the domains service on env's database at
// trackingInstance, resolving names from zone.
func trackingService(env *apptest.Env, zone dnscheck.Resolver) *domains.Service {
	a := env.App
	return &domains.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, DNS: zone, URLs: publicurl.New(trackingInstance),
		Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func workspaceOf(t *testing.T, env *apptest.Env, slug string) *store.Workspace {
	t.Helper()
	w, err := env.App.Q.GetWorkspaceBySlug(context.Background(), slug)
	if err != nil {
		t.Fatal(err)
	}
	return &w
}

// trackingDomain gives ws a sending domain whose tracking domain is
// tracking, verified at (never checked when at is nil, as before the
// upgrade recorded it).
func trackingDomain(t *testing.T, env *apptest.Env, ws uuid.UUID, domain, tracking string, at *time.Time) store.SendingDomain {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO sending_domains (id, workspace_id, domain, tracking_domain, tracking_status, tracking_verified_at)
		VALUES ($1, $2, $3, $4, 'verified', $5)`, id, ws, domain, tracking, at); err != nil {
		t.Fatal(err)
	}
	d, err := env.App.Q.DomGetAny(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func check(t *testing.T, svc *domains.Service, d store.SendingDomain) store.SendingDomain {
	t.Helper()
	d, err := svc.Check(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// carries reports whether ws's transactional mail uses host for its links
// and open pixel.
func carries(t *testing.T, env *apptest.Env, ws uuid.UUID, host string) bool {
	t.Helper()
	ok, err := env.App.Q.TxnTrackingDomainVerified(context.Background(), store.TxnTrackingDomainVerifiedParams{WorkspaceID: ws, Domain: host})
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

// The M11 attack on link tracking: another workspace's tracking domain,
// verified there first, doesn't verify as this workspace's by its CNAME.
// The re-check stops it at once, without the TXT record that only this
// workspace can publish; with the record, it carries links again. The
// first holder keeps the host throughout.
func TestTrackingDomainHeldFirstElsewhere(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	acme, eng := workspaceOf(t, env, "acme-newsletter"), workspaceOf(t, env, "acme-engineering-blog")
	zone := dnscheck.NewZone()
	zone.Add("reputecast.example.com", "A", "203.0.113.10")
	zone.Add("t.blog.example", "CNAME", "reputecast.example.com.")
	svc := trackingService(env, zone)
	first, later := time.Now().Add(-48*time.Hour).Truncate(time.Microsecond), time.Now().Add(-time.Hour)
	held := trackingDomain(t, env, eng.ID, "blog.example", "t.blog.example", &first)
	taken := trackingDomain(t, env, acme.ID, "blog.example", "t.blog.example", &later)

	// The periodic re-check.
	if err := (&domains.CheckWorker{S: svc}).Work(ctx, &river.Job[domains.CheckArgs]{}); err != nil {
		t.Fatal(err)
	}
	got, err := env.App.Q.DomGetAny(ctx, taken.ID)
	if err != nil || got.TrackingStatus != domains.StatusRefused || got.TrackingVerifiedAt != nil {
		t.Fatalf("the second workspace's tracking domain after the re-check: %q since %v (%v)", got.TrackingStatus, got.TrackingVerifiedAt, err)
	}
	if carries(t, env, acme.ID, "t.blog.example") {
		t.Fatal("the second workspace's links still use the host")
	}
	kept, err := env.App.Q.DomGetAny(ctx, held.ID)
	if err != nil || kept.TrackingStatus != domains.StatusVerified || kept.TrackingVerifiedAt == nil || !kept.TrackingVerifiedAt.Equal(first) || !carries(t, env, eng.ID, "t.blog.example") {
		t.Fatalf("the first holder after the re-check: %q since %v (%v)", kept.TrackingStatus, kept.TrackingVerifiedAt, err)
	}
	var actor, before, after string
	if err := env.App.Pool.QueryRow(ctx, `SELECT actor_type, before->>'status', after->>'status' FROM audit_events
		WHERE action = 'domain.tracking_check' AND target_id = $1 ORDER BY occurred_at DESC LIMIT 1`, taken.ID.String()).Scan(&actor, &before, &after); err != nil ||
		actor != "system" || before != "verified" || after != "refused" {
		t.Fatalf("audit: %s %s → %s (%v)", actor, before, after, err)
	}

	// The records page says why, and which TXT record would prove it.
	det, err := svc.Detail(ctx, acme, taken.ID.String(), "", false)
	if err != nil {
		t.Fatal(err)
	}
	if tr := det.Tracking; tr.Status != "refused" || tr.Detail != "Another workspace on this ReputeCast uses t.blog.example already, so its CNAME doesn't show it's yours." ||
		tr.TXTName != "_reputecast.t.blog.example" || tr.TXTValue != customhost.Token(acme.ID, "t.blog.example") {
		t.Fatalf("the records page: %+v", tr)
	}
	if det, err = svc.Detail(ctx, eng, held.ID.String(), "", false); err != nil || det.Tracking.Status != "verified" || det.Tracking.TXTValue != "" || det.Tracking.Detail != "" {
		t.Fatalf("the first holder's records page: %+v %v", det.Tracking, err)
	}
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var ov overviewT
	c.Do("GET", ws+"/senders", nil).MustStatus(t, 200).JSON(t, &ov)
	found := false
	for _, d := range ov.Domains {
		if d.ID == taken.ID.String() {
			found = d.Tracking.Text == "t.blog.example · not used" && d.Tracking.Tone == "warn"
		}
	}
	if !found {
		t.Fatalf("the domains table: %+v", ov.Domains)
	}

	// Another workspace's TXT value doesn't count; this workspace's does.
	zone.Add("_reputecast.t.blog.example", "TXT", customhost.Token(eng.ID, "t.blog.example"))
	if got = check(t, svc, got); got.TrackingStatus != domains.StatusRefused {
		t.Fatalf("with the other workspace's TXT value: %q", got.TrackingStatus)
	}
	zone.Add("_reputecast.t.blog.example", "TXT", customhost.Token(acme.ID, "t.blog.example"))
	if got = check(t, svc, got); got.TrackingStatus != domains.StatusVerified || got.TrackingVerifiedAt == nil || !carries(t, env, acme.ID, "t.blog.example") {
		t.Fatalf("with its TXT record: %q since %v", got.TrackingStatus, got.TrackingVerifiedAt)
	}
	// Verified later, Acme Newsletter takes nothing from the first holder.
	if kept = check(t, svc, kept); kept.TrackingStatus != domains.StatusVerified || !kept.TrackingVerifiedAt.Equal(first) {
		t.Fatalf("the first holder after the second proved its own: %q since %v", kept.TrackingStatus, kept.TrackingVerifiedAt)
	}
}

// Nobody recorded when a tracking domain was verified before the upgrade,
// so a host two workspaces verified then is neither's by its CNAME: each
// needs its own TXT record. A host nobody else uses verifies as before, and
// from then on its time is on record.
func TestTrackingDomainsFromBeforeTheUpgrade(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	acme, eng := workspaceOf(t, env, "acme-newsletter"), workspaceOf(t, env, "acme-engineering-blog")
	zone := dnscheck.NewZone()
	zone.Add("reputecast.example.com", "A", "203.0.113.10")
	zone.Add("t.blog.example", "CNAME", "reputecast.example.com.")
	zone.Add("t.acme.example", "CNAME", "reputecast.example.com.")
	svc := trackingService(env, zone)
	one := trackingDomain(t, env, eng.ID, "blog.example", "t.blog.example", nil)
	two := trackingDomain(t, env, acme.ID, "blog.example", "t.blog.example", nil)
	alone := trackingDomain(t, env, acme.ID, "acme.example", "t.acme.example", nil)

	if one, two = check(t, svc, one), check(t, svc, two); one.TrackingStatus != domains.StatusRefused || two.TrackingStatus != domains.StatusRefused {
		t.Fatalf("a host two workspaces verified before the upgrade: %q and %q", one.TrackingStatus, two.TrackingStatus)
	}
	zone.Add("_reputecast.t.blog.example", "TXT", customhost.Token(eng.ID, "t.blog.example"))
	if one, two = check(t, svc, one), check(t, svc, two); one.TrackingStatus != domains.StatusVerified || two.TrackingStatus != domains.StatusRefused {
		t.Fatalf("after the TXT record of one: %q and %q", one.TrackingStatus, two.TrackingStatus)
	}
	if alone = check(t, svc, alone); alone.TrackingStatus != domains.StatusVerified || alone.TrackingVerifiedAt == nil || !carries(t, env, acme.ID, "t.acme.example") {
		t.Fatalf("a host nobody else uses: %q since %v", alone.TrackingStatus, alone.TrackingVerifiedAt)
	}
}

// The tracking domain's checks of a host nobody else uses work as before;
// the instance's own address is never a workspace's, and a host under it
// needs the TXT record.
func TestTrackingDomainChecks(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	acme := workspaceOf(t, env, "acme-newsletter")
	zone := dnscheck.NewZone()
	zone.Add("reputecast.example.com", "A", "203.0.113.10")
	zone.Add("t.acme.example", "CNAME", "reputecast.example.com.")
	zone.Add("t.other.example", "CNAME", "elsewhere.example.net.")
	zone.Add("t.reputecast.example.com", "CNAME", "reputecast.example.com.")
	svc := trackingService(env, zone)

	for _, c := range []struct{ domain, tracking, want string }{
		{"acme.example", "t.acme.example", domains.StatusVerified},
		{"other.example", "t.other.example", domains.StatusWrong},
		{"missing.example", "t.missing.example", domains.StatusMissing},
		{"reputecast.example.com", "t.reputecast.example.com", domains.StatusRefused},
		{"example.com", "reputecast.example.com", domains.StatusRefused},
	} {
		d := trackingDomain(t, env, acme.ID, c.domain, c.tracking, nil)
		run := func() {
			if d = check(t, svc, d); d.TrackingStatus != c.want {
				t.Errorf("%s: %q, want %q", c.tracking, d.TrackingStatus, c.want)
			}
		}
		run()
		det, err := svc.Detail(ctx, acme, d.ID.String(), "", false)
		if err != nil {
			t.Fatal(err)
		}
		switch c.tracking {
		case "t.reputecast.example.com":
			if det.Tracking.Detail != "t.reputecast.example.com is under reputecast.example.com, this ReputeCast's own address, so its CNAME doesn't show it's yours." ||
				det.Tracking.TXTName != "_reputecast.t.reputecast.example.com" {
				t.Errorf("under the instance's address: %+v", det.Tracking)
			}
			zone.Add("_reputecast.t.reputecast.example.com", "TXT", det.Tracking.TXTValue)
			c.want = domains.StatusVerified
			run()
		case "reputecast.example.com":
			if det.Tracking.Detail != "reputecast.example.com is one of this ReputeCast's own addresses. Use a subdomain of your sending domain instead." || det.Tracking.TXTValue != "" {
				t.Errorf("the instance's own address: %+v", det.Tracking)
			}
		default:
			if det.Tracking.Detail != "" || det.Tracking.TXTValue != "" {
				t.Errorf("%s: %+v", c.tracking, det.Tracking)
			}
		}
	}
}
