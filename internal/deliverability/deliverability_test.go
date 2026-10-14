package deliverability_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/deliverability"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/hostrepute"
	"github.com/hostrepute/reputecast/internal/store"
)

// rateView is a bounce or complaint rate against a provider's thresholds.
type rateView struct {
	Label, Thresholds, Tone string
	Value                   *float64
	Usage                   float64
}

// toneOf is the tone a rate gets at a share of its first threshold.
func toneOf(usage float64) string {
	switch {
	case usage >= 100:
		return "bad"
	case usage >= 50:
		return "warn"
	}
	return "ok"
}

type overview struct {
	HostRepute struct {
		Status      string
		TeamName    string
		Sources     int
		Unreachable *struct{ Message string }
	}
	Checks struct {
		Via, Interval string
		LastAt        *string
		Running       *struct{ ID, Status string }
	}
	Assets []struct {
		Asset, Type, Provider, Verdict, Tone string
		UsedBy                               []string
		Score                                *int
		Warmup                               *struct{ Day, Days int }
		Blocklists                           struct {
			Via             string
			Sources, Listed int
			Listings        []string
		}
		Fallback        *struct{ Via string }
		Stale           *string
		Auth            *struct{ SPF, DKIM, DMARC string }
		DMARCAligned    *float64 `json:"dmarcAligned"`
		GmailReputation string
		ReverseDNS      *struct {
			PTR     string
			Matches bool
		} `json:"reverseDns"`
		SNDS *struct {
			FilterResult string
			TrapHits     int
		}
	}
	Pools []struct {
		Short, AccountStatus, OnlyStream string
		DedicatedIPOption                string `json:"dedicatedIpOption"`
		VolumeThisMonth                  *int64
	}
	Providers []struct {
		Name, Badge, Tone string
		Bounce, Complaint rateView
		Note              *struct{ Label, Value string }
	}
	Blocklists struct {
		Via                    string
		Sources, Clean, Listed int
		Assets                 []string
		Results                []struct {
			Source    string
			Assets    []string
			Listed    bool
			DelistURL string
			Delisting string
		}
	}
	Postmaster *struct {
		DomainReputation string
		SpamRate         float64
		Series           []struct{ SpamRate *float64 }
		Peak             *struct {
			SpamRate float64
			Note     string
		}
	}
	SNDS *struct {
		FilterResult string
		Messages7d   int64
		JMRPReports  int
	}
	DMARC struct {
		Via      string
		PassRate float64
		Messages int64
		Sources  []struct {
			Label, SPF, DKIM, Result string
			Known                    bool
			Messages                 int64
		}
		Unknown *struct{ IP string }
	}
	Incidents struct {
		Items []struct {
			Title, Detail, Kind, Severity, Source string
			Links                                 []struct{ Kind, Label, ID string }
		}
		FromHostRepute bool
	}
}

func TestOverviewDemo(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var o overview
	c.Do("GET", "/workspaces/acme-newsletter/deliverability", nil).MustStatus(t, 200).JSON(t, &o)

	if o.HostRepute.Status != "connected" || o.HostRepute.TeamName != "Acme Inc" || o.Checks.Via != "hostrepute" || o.Checks.Interval != "hourly" {
		t.Fatalf("header: %+v %+v", o.HostRepute, o.Checks)
	}
	if len(o.Assets) != 2 {
		t.Fatalf("assets: %+v", o.Assets)
	}
	d, ip := o.Assets[0], o.Assets[1]
	if d.Asset != "news.acme.io" || *d.Score != 96 || d.Verdict != "Good" || d.Blocklists.Sources != 84 || d.Blocklists.Listed != 0 ||
		strings.Join(d.UsedBy, ",") != "SES,Postmark,Postal" || d.Auth.SPF != "pass" || d.Auth.DKIM != "pass" || d.Auth.DMARC != "quarantine" ||
		*d.DMARCAligned != 99.6 || d.GmailReputation != "high" {
		t.Fatalf("domain card: %+v", d)
	}
	if ip.Asset != "203.0.113.25" || *ip.Score != 81 || ip.Tone != "warn" || ip.Provider != "Postal" || ip.Warmup.Day != 9 || ip.Warmup.Days != 21 ||
		ip.Blocklists.Listings[0] != "UCEPROTECT Level 3" || ip.ReverseDNS.PTR != "mta.acme.io" || !ip.ReverseDNS.Matches || ip.SNDS.FilterResult != "green" {
		t.Fatalf("IP card: %+v", ip)
	}
	if len(o.Pools) != 2 || o.Pools[0].Short != "SES" || o.Pools[0].DedicatedIPOption != "$24.95 per IP per month" ||
		o.Pools[1].AccountStatus != "Approved" || o.Pools[1].OnlyStream != "system" || *o.Pools[1].VolumeThisMonth != 3412 {
		t.Fatalf("pools: %+v", o.Pools)
	}
	// Rates are the providers workstream's 7-day figures (manage.UsageOf),
	// so they match Sending providers; SES's are exactly the board's.
	ses, pm, postal := o.Providers[0], o.Providers[1], o.Providers[2]
	if ses.Badge != "Healthy · no enforcement" || ses.Bounce.Thresholds != "review 5%, pause 10%" || *ses.Bounce.Value != 0.6 ||
		*ses.Complaint.Value != 0.04 || ses.Complaint.Usage != 40 || ses.Note.Value != "Bounce and complaint metrics only; no VDM" ||
		pm.Badge != "Approved" || pm.Complaint.Label != "Spam complaint rate" || pm.Bounce.Thresholds != "limit 10%" || pm.Note.Value != "Transactional · System mail only" ||
		postal.Name != "Postal (self-hosted)" || postal.Badge != "Warming up" || postal.Bounce.Thresholds != "warm-up guard 2%" ||
		postal.Complaint.Thresholds != "guard 0.1%" || postal.Note.Value != "Set by ReputeCast; Postal has no account limits" {
		t.Fatalf("providers: %+v", o.Providers)
	}
	for _, p := range o.Providers {
		for _, r := range []rateView{p.Bounce, p.Complaint} {
			if r.Value == nil || r.Tone != toneOf(r.Usage) {
				t.Fatalf("%s %s: %+v", p.Name, r.Label, r)
			}
		}
	}
	b := o.Blocklists
	if b.Via != "hostrepute" || b.Sources != 84 || b.Clean != 83 || b.Listed != 1 || strings.Join(b.Assets, ", ") != "news.acme.io, acme.io, 203.0.113.25" {
		t.Fatalf("blocklists: %+v", b)
	}
	first := []string{"UCEPROTECT Level 3", "Spamhaus ZEN", "Spamhaus DBL", "SURBL", "URIBL", "Barracuda BRBL", "SpamCop", "Invaluement"}
	for i, want := range first {
		if b.Results[i].Source != want {
			t.Fatalf("row %d: %s, want %s", i, b.Results[i].Source, want)
		}
	}
	if !b.Results[0].Listed || b.Results[0].Delisting != "pending" || b.Results[0].DelistURL == "" || b.Results[1].DelistURL != "" {
		t.Fatalf("listing row: %+v", b.Results[:2])
	}
	if o.Postmaster.DomainReputation != "high" || o.Postmaster.SpamRate != 0.05 || len(o.Postmaster.Series) != 15 || o.Postmaster.Peak.SpamRate != 0.09 ||
		o.Postmaster.Peak.Note != "first send after migration" {
		t.Fatalf("postmaster: %+v", o.Postmaster)
	}
	if o.SNDS.Messages7d != 11194 || o.SNDS.JMRPReports != 4 {
		t.Fatalf("snds: %+v", o.SNDS)
	}
	if o.DMARC.PassRate != 99.6 || len(o.DMARC.Sources) != 4 || o.DMARC.Sources[0].Label != "Amazon SES" || o.DMARC.Sources[0].Messages != 131604 ||
		o.DMARC.Sources[1].Label != "Postal · 203.0.113.25" || o.DMARC.Unknown.IP != "198.51.100.77" || o.DMARC.Sources[3].Result != "quarantined" ||
		o.DMARC.Sources[3].SPF != "fail" || o.DMARC.Sources[3].DKIM != "none" {
		t.Fatalf("dmarc: %+v", o.DMARC)
	}
	items := o.Incidents.Items
	if len(items) != 5 || items[0].Kind != "warmup_slowdown" || len(items[0].Links) != 2 || items[0].Links[1].Label != "October Product Update" ||
		!delistingRequested.MatchString(items[1].Detail) ||
		items[1].Links[0].Label != "Postal" || items[3].Links[0].Kind != "campaign_report" || items[4].Links[0].Kind != "import" {
		t.Fatalf("incidents: %+v", items)
	}

	// Viewers and the Content Writer can read Deliverability.
	for _, u := range []string{"aisha@acme.io", "daniel@acme.io"} {
		v := env.Client()
		v.SignInDemo(u)
		v.Do("GET", "/workspaces/acme-newsletter/deliverability", nil).MustStatus(t, 200)
	}
}

func TestRunChecksAndPermissions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("aisha@acme.io") // a Viewer can refresh the checks
	var run struct{ ID, Via, Status string }
	c.Do("POST", "/workspaces/acme-newsletter/deliverability/checks", nil).MustStatus(t, http.StatusAccepted).JSON(t, &run)
	if run.Via != "hostrepute" || run.Status != "running" {
		t.Fatalf("run: %+v", run)
	}
	var again struct{ ID string }
	c.Do("POST", "/workspaces/acme-newsletter/deliverability/checks", nil).MustStatus(t, http.StatusAccepted).JSON(t, &again)
	if again.ID != run.ID {
		t.Fatal("a running check is returned instead of starting another")
	}
	c.Do("GET", "/workspaces/acme-newsletter/deliverability/checks/"+run.ID, nil).MustStatus(t, 200)
	var audits int
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = 'deliverability.check_run'`).Scan(&audits)
	if audits != 1 {
		t.Fatalf("audits: %d", audits)
	}
	var jobs []string
	rows, _ := env.App.Pool.Query(context.Background(), `SELECT kind FROM river_job ORDER BY id`)
	for rows.Next() {
		var k string
		_ = rows.Scan(&k)
		jobs = append(jobs, k)
	}
	if !strings.Contains(strings.Join(jobs, ","), "hostrepute.sync") || !strings.Contains(strings.Join(jobs, ","), "deliverability.check") {
		t.Fatalf("jobs: %v", jobs)
	}
	// Viewers can't act on HostRepute or upload reports.
	c.Do("POST", "/workspaces/acme-newsletter/deliverability/delistings", map[string]any{"asset": "203.0.113.25"}).MustStatus(t, 403)
}

func TestDelistingRequest(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var out struct {
		Requests []struct {
			Source, Status string
			Already        bool
		}
	}
	c.Do("POST", "/workspaces/acme-newsletter/deliverability/delistings", map[string]any{"asset": "203.0.113.25"}).MustStatus(t, 200).JSON(t, &out)
	if len(out.Requests) != 1 || !out.Requests[0].Already || out.Requests[0].Status != "pending" {
		t.Fatalf("pending request is reported, not repeated: %+v", out)
	}
	_, _ = env.App.Pool.Exec(context.Background(), `UPDATE blocklist_results SET delisting = '', delisting_requested_at = NULL WHERE listed`)
	c.Do("POST", "/workspaces/acme-newsletter/deliverability/delistings", map[string]any{"asset": "203.0.113.25"}).MustStatus(t, 200).JSON(t, &out)
	if out.Requests[0].Already || out.Requests[0].Status != "requested" {
		t.Fatalf("new request: %+v", out)
	}
	var audits int
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = 'deliverability.delisting_request'`).Scan(&audits)
	if audits != 1 {
		t.Fatalf("audits: %d", audits)
	}
	c.Do("POST", "/workspaces/acme-newsletter/deliverability/delistings", map[string]any{"asset": "news.acme.io"}).MustStatus(t, 422)
}

const sampleReport = `<?xml version="1.0" encoding="UTF-8"?>
<feedback>
  <report_metadata><org_name>fastmail.com</org_name><email>dmarc@fastmail.com</email><report_id>fm-42</report_id>
    <date_range><begin>%d</begin><end>%d</end></date_range></report_metadata>
  <policy_published><domain>acme.io</domain><adkim>r</adkim><aspf>r</aspf><p>quarantine</p><sp>quarantine</sp><pct>100</pct></policy_published>
  <record><row><source_ip>192.0.2.10</source_ip><count>40</count><policy_evaluated><disposition>reject</disposition><dkim>fail</dkim><spf>fail</spf></policy_evaluated></row>
    <identifiers><header_from>news.acme.io</header_from></identifiers>
    <auth_results><spf><domain>news.acme.io</domain><result>softfail</result></spf></auth_results></record>
</feedback>`

func TestUploadReportAndInvestigate(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	end := time.Now().Unix()
	report := fmt.Sprintf(sampleReport, end-86400, end)
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(report))
	_ = zw.Close()

	st, body := rawPost(t, env, "linh@acme.io", "/workspaces/acme-newsletter/deliverability/dmarc/reports", buf.Bytes())
	if st != 201 {
		t.Fatalf("upload: %d %s", st, body)
	}
	var audits int
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = 'dmarc_report.upload'`).Scan(&audits)
	if audits != 1 {
		t.Fatalf("upload audits: %d", audits)
	}
	var o overview
	c.Do("GET", "/workspaces/acme-newsletter/deliverability", nil).MustStatus(t, 200).JSON(t, &o)
	found := false
	for _, s := range o.DMARC.Sources {
		if s.Label == "Unknown · 192.0.2.10" && s.Result == "rejected" && s.SPF == "softfail" && s.DKIM == "none" {
			found = true
		}
	}
	if !found {
		t.Fatalf("uploaded source: %+v", o.DMARC.Sources)
	}
	var src struct {
		IP, SPF, DKIM, Disposition string
		Known                      bool
		Messages                   int64
		Reporters                  []string
	}
	c.Do("GET", "/workspaces/acme-newsletter/deliverability/dmarc/sources/198.51.100.77", nil).MustStatus(t, 200).JSON(t, &src)
	if src.Known || src.Messages != 612 || src.SPF != "fail" || src.Disposition != "quarantine" || len(src.Reporters) == 0 {
		t.Fatalf("investigate: %+v", src)
	}
	c.Do("GET", "/workspaces/acme-newsletter/deliverability/dmarc/sources/192.0.2.99", nil).MustStatus(t, 404)

	if st, _ := rawPost(t, env, "linh@acme.io", "/workspaces/acme-newsletter/deliverability/dmarc/reports", []byte("<feedback><oops")); st != 422 {
		t.Fatalf("garbage: %d", st)
	}
	other := strings.ReplaceAll(report, "news.acme.io", "news.example.org")
	other = strings.ReplaceAll(other, "<domain>acme.io</domain>", "<domain>example.org</domain>")
	if st, body := rawPost(t, env, "linh@acme.io", "/workspaces/acme-newsletter/deliverability/dmarc/reports", []byte(other)); st != 422 || !strings.Contains(body, "isn't a sending domain") {
		t.Fatalf("foreign domain: %d %s", st, body)
	}
	// Uploading needs domains:manage: the Viewer is refused.
	if st, _ := rawPost(t, env, "aisha@acme.io", "/workspaces/acme-newsletter/deliverability/dmarc/reports", []byte(report)); st != 403 {
		t.Fatalf("viewer upload: %d", st)
	}
}

// rawPost sends a non-JSON body with a fresh session for the demo user.
func rawPost(t *testing.T, env *apptest.Env, email, path string, body []byte) (int, string) {
	t.Helper()
	ctx := context.Background()
	u, err := env.App.Q.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	token, hash := authn.NewToken()
	if _, err := env.App.Q.CreateSession(ctx, store.CreateSessionParams{ID: uuid.Must(uuid.NewV7()), UserID: u.ID, TokenHash: hash, AuthMethod: "test", MfaVerified: true, ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodPost, env.Server.URL+"/api/v1"+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(&http.Cookie{Name: env.App.Cfg.Auth.CookieName, Value: token})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestExportAndReputation(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	r := c.Do("GET", "/workspaces/acme-newsletter/deliverability/export", nil).MustStatus(t, 200)
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "text/csv") || !strings.Contains(r.Header.Get("Content-Disposition"), "deliverability-acme-newsletter-") ||
		!strings.Contains(string(r.Body), "UCEPROTECT Level 3") {
		t.Fatalf("export: %v %s", r.Header, r.Body[:200])
	}
	var rep struct {
		Status, Via, Reason string
		Listings            []struct{ Source, Impact string }
	}
	c.Do("GET", "/workspaces/acme-newsletter/deliverability/reputation", nil).MustStatus(t, 200).JSON(t, &rep)
	if rep.Status != "warn" || rep.Via != "hostrepute" || len(rep.Listings) != 1 || rep.Listings[0].Impact != "low" {
		t.Fatalf("reputation: %+v", rep)
	}
	_, _ = env.App.Pool.Exec(context.Background(), `UPDATE hostrepute_connections SET status = 'revoked', token_rejected_at = now()`)
	c.Do("GET", "/workspaces/acme-newsletter/deliverability/reputation", nil).MustStatus(t, 200).JSON(t, &rep)
	if rep.Status != "unknown" || !strings.Contains(rep.Reason, "rejected the token") {
		t.Fatalf("unknown, never a block, while the token is rejected: %+v", rep)
	}
	var o overview
	c.Do("GET", "/workspaces/acme-newsletter/deliverability", nil).MustStatus(t, 200).JSON(t, &o)
	if o.HostRepute.Status != "revoked" || o.Assets[0].Stale == nil || o.Assets[0].Fallback == nil || o.Assets[0].Tone != "bad" {
		t.Fatalf("stale cards with the built-in fallback: %+v", o.Assets[0])
	}
}

// The built-in check finds a listing with nothing but DNS: an injected zone.
func TestBuiltinCheck(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	c.Do("DELETE", "/workspaces/acme-newsletter/integrations/hostrepute", nil).MustStatus(t, 204)

	zone := dnscheck.NewZone()
	zone.Add("25.113.0.203.zen.spamhaus.org", "A", "127.0.0.3")
	zone.Add("news.acme.io", "TXT", "v=spf1 -all")
	s := &deliverability.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), DNS: zone,
		HostRepute: &hostrepute.Service{}}
	var wsID, runID uuid.UUID
	_ = env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID)
	_ = env.App.Pool.QueryRow(ctx, `SELECT id FROM deliverability_check_runs WHERE trigger = 'fallback'`).Scan(&runID)
	if err := s.RunChecks(ctx, deliverability.CheckArgs{WorkspaceID: wsID, RunID: &runID, Blocklists: true}); err != nil {
		t.Fatal(err)
	}
	var o overview
	c.Do("GET", "/workspaces/acme-newsletter/deliverability", nil).MustStatus(t, 200).JSON(t, &o)
	if o.HostRepute.Status != "not_connected" || o.Checks.Via != "builtin" || o.Checks.LastAt == nil || o.Blocklists.Via != "builtin" ||
		o.Blocklists.Sources != 8 || o.Blocklists.Listed != 1 || o.Blocklists.Results[0].Source != "Spamhaus ZEN" {
		t.Fatalf("built-in: %+v %+v", o.Checks, o.Blocklists)
	}
	ip := o.Assets[1]
	if ip.Blocklists.Via != "builtin" || ip.Blocklists.Listed != 1 || ip.Tone != "bad" || ip.Score != nil {
		t.Fatalf("IP card from the built-in check: %+v", ip)
	}
	if d := o.Assets[0]; d.Auth == nil || d.Auth.DMARC != "missing" || d.Auth.DKIM != "missing" {
		t.Fatalf("the DNS checks ran on the injected zone: %+v", d.Auth)
	}
	var status string
	var listed int
	_ = env.App.Pool.QueryRow(ctx, `SELECT status, listed FROM deliverability_check_runs WHERE id = $1`, runID).Scan(&status, &listed)
	if status != "done" || listed != 1 {
		t.Fatalf("run: %s %d", status, listed)
	}
	var inc, notes int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM incidents WHERE source = 'builtin' AND title = '203.0.113.25 listed on Spamhaus ZEN'`).Scan(&inc)
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE title = '203.0.113.25 listed on Spamhaus ZEN'`).Scan(&notes)
	if inc != 1 || notes == 0 {
		t.Fatalf("new listing recorded (%d) and notified (%d)", inc, notes)
	}
	// Running again keeps the first listing date and doesn't notify twice.
	if err := s.RunChecks(ctx, deliverability.CheckArgs{WorkspaceID: wsID, Blocklists: true}); err != nil {
		t.Fatal(err)
	}
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM incidents WHERE source = 'builtin'`).Scan(&inc)
	if inc != 1 {
		t.Fatalf("incidents after a second run: %d", inc)
	}
}

// delistingRequested is the blocklist incident's detail, whatever day the
// demo's delisting request falls on.
var delistingRequested = regexp.MustCompile(`Delisting requested \d{1,2} [A-Z][a-z]{2}, pending\.$`)
