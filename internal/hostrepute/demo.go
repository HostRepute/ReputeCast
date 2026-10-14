package hostrepute

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

// The demo HostRepute serves the design's sample data (spec §6, add-on §4)
// without a network. `reputecast seed` stores a demo connection, and
// development instances with hostrepute.demo=true connect to it through an
// instant OAuth consent. Dates are relative to today, like every other demo
// date.

// DemoTeam is the sample HostRepute team.
const DemoTeam = "Acme Inc"

// DemoLimit is the sample plan's monitoring limit (3 assets).
const DemoLimit = 3

// demoLocation anchors demo days to the sample workspace's timezone.
func demoLocation() *time.Location {
	if loc, err := time.LoadLocation("America/New_York"); err == nil {
		return loc
	}
	return time.UTC
}

// demoDay returns daysAgo days before today at hh:mm in the demo timezone.
func demoDay(now time.Time, daysAgo, hh, mm int) time.Time {
	loc := demoLocation()
	n := now.In(loc)
	return time.Date(n.Year(), n.Month(), n.Day(), hh, mm, 0, 0, loc).AddDate(0, 0, -daysAgo)
}

// DemoAccount is the account behind the demo token.
func DemoAccount(monitored int) Account {
	return Account{
		ID: "acct_demo", Email: "linh@acme.io", Scopes: slices.Clone(Scopes), Sources: 84, CheckIntervalMinutes: 60,
		DMARCRua: "dmarc@reports.hostrepute.com",
		Team:     Team{ID: "team_acme", Name: DemoTeam, Plan: "Starter", MonitorLimit: DemoLimit, Monitored: monitored},
	}
}

// demoScores are HostRepute's scores for the sample assets.
var demoScores = map[string]struct {
	score    int
	verdict  string
	listings int
}{
	"news.acme.io": {96, "Good", 0},
	"acme.io":      {97, "Good", 0},
	"203.0.113.25": {81, "Fair", 1},
}

// DemoAssets returns the demo assets for the monitored names.
func DemoAssets(now time.Time, names []string) []Asset {
	checked := DemoLastCheck(now)
	out := make([]Asset, 0, len(names))
	for _, n := range names {
		a := Asset{ID: "asset_" + strings.NewReplacer(".", "_").Replace(n), Asset: n, Type: GuessType(n), Status: "synced", LastCheckedAt: &checked}
		if s, ok := demoScores[n]; ok {
			score := s.score
			a.Score, a.Verdict, a.Listings = &score, s.verdict, s.listings
		} else {
			a.Status, a.LastCheckedAt = "pending", nil
		}
		out = append(out, a)
	}
	return out
}

// DemoMonitored is what the sample team monitors.
var DemoMonitored = []string{"news.acme.io", "acme.io", "203.0.113.25"}

// DemoLastCheck is HostRepute's last full check (hourly, at :40).
func DemoLastCheck(now time.Time) time.Time {
	t := now.Truncate(time.Hour).Add(40 * time.Minute)
	if t.After(now) {
		t = t.Add(-time.Hour)
	}
	return t
}

// GuessType says whether an asset is an IP or a domain name.
func GuessType(asset string) string {
	if strings.Count(asset, ".") == 3 && strings.Trim(asset, "0123456789.") == "" || strings.Contains(asset, ":") {
		return AssetIP
	}
	return AssetDomain
}

// demoSource is one of the 84 blocklists the demo HostRepute checks.
type demoSource struct {
	name, kind, impact, url string
	rank                    int
}

func demoSources() []demoSource {
	src := func(name, kind, impact, url string) demoSource {
		return demoSource{name: name, kind: kind, impact: impact, url: url}
	}
	list := []demoSource{
		src("Spamhaus ZEN", "ip", "high", "https://check.spamhaus.org/"),
		src("Spamhaus DBL", "domain", "high", "https://check.spamhaus.org/"),
		src("SURBL", "domain", "high", "https://surbl.org/surbl-analysis"),
		src("URIBL", "domain", "high", "https://admin.uribl.com/"),
		src("Barracuda BRBL", "ip", "high", "https://www.barracudacentral.org/rbl/removal-request"),
		src("SpamCop", "ip", "medium", "https://www.spamcop.net/bl.shtml"),
		src("Invaluement", "ip", "medium", "https://www.invaluement.com/lookup/"),
	}
	high := []string{"Spamhaus SBL", "Spamhaus XBL", "Spamhaus CSS", "Spamhaus PBL", "Spamhaus ZRD", "Proofpoint IP Reputation", "Cisco Talos", "Microsoft Outlook.com", "Google Safe Browsing"}
	for _, n := range high {
		kind := "ip"
		if n == "Spamhaus ZRD" || n == "Google Safe Browsing" {
			kind = "domain"
		}
		list = append(list, src(n, kind, "high", "https://hostrepute.com/blocklists"))
	}
	medium := []string{"Mailspike", "Abusix Mail Intelligence", "Abusix DBL", "Invaluement ivmURI", "PSBL", "SpamRATS RATS-Spam", "Hostkarma", "NiX Spam", "JunkEmailFilter", "PhishTank", "OpenPhish", "Spamhaus AuthBL", "SEM Black", "SEM URI", "Lashback UBL", "Truncate GBUDB"}
	for _, n := range medium {
		kind := "ip"
		if strings.Contains(n, "URI") || strings.Contains(n, "DBL") || n == "PhishTank" || n == "OpenPhish" {
			kind = "domain"
		}
		list = append(list, src(n, kind, "medium", "https://hostrepute.com/blocklists"))
	}
	low := []string{
		"UCEPROTECT Level 1", "UCEPROTECT Level 2", "UCEPROTECT Level 3", "SpamRATS RATS-Dyna", "SpamRATS RATS-NoPtr", "DroneBL", "0spam", "Blocklist.de",
		"Backscatterer", "s5h.net", "SEM Backscatter", "SEM Fresh", "SEM URIRed", "URIBL Grey", "URIBL Red", "Nordspam BL", "Nordspam DBL",
		"Woodys SMTP Blacklist", "ZapBL", "Anonmails", "Calivent", "Cymru Bogons", "DRMX", "Fabel Sources", "Interserver", "Konstant",
		"RBL JP", "SpamSources", "Suomispam", "Swinog", "Triumf", "WPBL", "Abuse.ro", "Dan Tor exit", "Mailspike Z", "Spameatingmonkey",
		"Rbl.jp URIBL", "Hostkarma URIBL", "0spam URL", "Hijacked domains", "Squid blocklist", "Rspamd DBL",
		"SpamCannibal", "IPrange RBL", "JIPPG", "Kempt", "LNSG", "Mailblacklist", "MegaRBL", "Pedantic", "RATS-Auth", "TornevAll",
	}
	for _, n := range low {
		kind := "ip"
		if strings.Contains(n, "URI") || strings.Contains(n, "DBL") || strings.Contains(n, "domains") || strings.Contains(n, "URL") || n == "SEM Fresh" {
			kind = "domain"
		}
		u := "https://hostrepute.com/blocklists"
		if strings.HasPrefix(n, "UCEPROTECT") {
			u = "https://www.uceprotect.net/en/rblcheck.php"
		}
		list = append(list, src(n, kind, "low", u))
	}
	for i := range list {
		list[i].rank = i + 1
	}
	return list
}

// DemoBlocklists checks the monitored assets on the 84 demo sources. The
// Postal IP is listed on UCEPROTECT Level 3 (a network-range listing) since
// five days ago, with a delisting request pending since the next day.
func DemoBlocklists(now time.Time, monitored []string) Blocklists {
	checked := DemoLastCheck(now)
	listedSince := demoDay(now, 5, 7, 12)
	requested := demoDay(now, 4, 10, 5)
	var domains, ips []string
	for _, a := range monitored {
		switch GuessType(a) {
		case AssetIP:
			ips = append(ips, a)
		default:
			if !strings.HasPrefix(a, "mta.") { // hostnames are checked as IPs through their A record
				domains = append(domains, a)
			}
		}
	}
	out := Blocklists{Sources: 84, CheckedAt: checked, Results: []BlocklistResult{}}
	for _, s := range demoSources() {
		assets := ips
		if s.kind == "domain" {
			assets = domains
			if s.name == "Spamhaus DBL" && slices.Contains(domains, "news.acme.io") {
				assets = []string{"news.acme.io"} // the DBL is queried with the sending hostname
			}
		}
		for _, a := range assets {
			r := BlocklistResult{Asset: a, AssetType: s.kind, Source: s.name, Impact: s.impact, Rank: s.rank, DelistURL: s.url, CheckedAt: checked}
			if s.name == "UCEPROTECT Level 3" && a == "203.0.113.25" {
				r.Listed, r.ListedSince = true, &listedSince
				r.Detail = "Network-range listing (AS64500), low impact"
				r.Delisting = &Delisting{ID: "dl_demo", Status: "pending", RequestedAt: requested}
			}
			out.Results = append(out.Results, r)
		}
	}
	return out
}

// demoSpamRates are Gmail spam rates (percent) for the last 30 days, oldest
// first: steady on Brevo, a peak of 0.09% on the first send after the
// migration 24 days ago, back to 0.03% within a week.
var demoSpamRates = []float64{
	0.03, 0.03, 0.02, 0.03, 0.04, 0.03, // 30…25 days ago
	0.09, 0.08, 0.07, 0.06, 0.05, 0.04, 0.03, // first send after migration
	0.03, 0.04, 0.05, 0.06, 0.05, 0.04, 0.03, 0.05, 0.06, 0.07, 0.06, 0.05, 0.04, 0.05, 0.06, 0.05, 0.05,
}

// DemoPostmaster is 30 days of Postmaster Tools data for news.acme.io.
func DemoPostmaster(now time.Time, domain string) []PostmasterDay {
	if domain != "news.acme.io" {
		return []PostmasterDay{}
	}
	out := make([]PostmasterDay, 0, len(demoSpamRates))
	one := 1.0
	zero := 0
	for i, rate := range demoSpamRates {
		daysAgo := len(demoSpamRates) - i
		r := rate / 100
		rep := "HIGH"
		if daysAgo <= 24 && daysAgo >= 22 {
			rep = "MEDIUM"
		}
		errs := 0.001
		out = append(out, PostmasterDay{
			Day: demoDay(now, daysAgo, 0, 0).Format("2006-01-02"), SpamRate: &r, DomainReputation: rep, IPReputation: "HIGH",
			SPFRate: &one, DKIMRate: &one, DMARCRate: &one, TLSRate: &one, DeliveryErrors: &errs, FBLFlagged: &zero,
		})
	}
	return out
}

// demoSNDSMessages is the Postal IP's daily volume during its warm-up,
// oldest first (nine days; the last seven add up to 11,194).
var demoSNDSMessages = []int64{300, 600, 900, 1200, 1500, 1800, 1694, 2000, 2100}

// DemoSNDS is Microsoft SNDS data for the Postal IP since it started sending.
func DemoSNDS(now time.Time, ip string) []SNDSDay {
	if ip != "203.0.113.25" {
		return []SNDSDay{}
	}
	out := make([]SNDSDay, 0, len(demoSNDSMessages))
	jmrp := []int{0, 0, 0, 1, 0, 1, 0, 2, 0}
	for i, n := range demoSNDSMessages {
		daysAgo := len(demoSNDSMessages) - i
		cr := 0.0005
		out = append(out, SNDSDay{Day: demoDay(now, daysAgo, 0, 0).Format("2006-01-02"), FilterResult: "green", ComplaintRate: &cr, Messages: n, JMRPReports: jmrp[i]})
	}
	return out
}

// DemoDMARC is a week of aggregate reports for news.acme.io: every provider
// aligned, plus 612 messages from an unknown host that failed and were
// quarantined.
func DemoDMARC(now time.Time, domain string) []DMARCReport {
	if domain != "news.acme.io" {
		return []DMARCReport{}
	}
	type src struct {
		ip, ptr  string
		count    int64
		dkim     string
		spf      string
		dkimDom  string
		spfDom   string
		disp     string
		spfRes   string
		dkimRes  string
		reporter int // index into orgs
	}
	orgs := []string{"google.com", "Outlook.com", "Yahoo"}
	sources := []src{
		{"54.240.8.30", "a8-30.smtp-out.amazonses.com", 61204, "pass", "pass", "news.acme.io", "bounce.news.acme.io", "none", "pass", "pass", 0},
		{"54.240.8.41", "a8-41.smtp-out.amazonses.com", 38915, "pass", "pass", "news.acme.io", "bounce.news.acme.io", "none", "pass", "pass", 1},
		{"23.251.226.12", "e226-12.smtp-out.amazonses.com", 31485, "pass", "pass", "news.acme.io", "bounce.news.acme.io", "none", "pass", "pass", 2},
		{"203.0.113.25", "mta.acme.io", 11194, "pass", "pass", "news.acme.io", "rp.news.acme.io", "none", "pass", "pass", 0},
		{"50.31.156.104", "mta104.mtasv.net", 3412, "pass", "pass", "news.acme.io", "pm-bounces.news.acme.io", "none", "pass", "pass", 0},
		{"198.51.100.77", "", 612, "fail", "fail", "", "news.acme.io", "quarantine", "fail", "none", 0},
	}
	var out []DMARCReport
	// Reports cover whole UTC days, and the summary counts those that end in
	// the last 7 days: the days are counted in UTC too, so the week is whole
	// at any hour (New York's evening is already tomorrow in UTC).
	today := now.UTC().Truncate(24 * time.Hour)
	for day := 7; day >= 1; day-- {
		begin := today.AddDate(0, 0, -day)
		for oi, org := range orgs {
			rep := DMARCReport{
				Domain: domain, OrgName: org, ReportID: fmt.Sprintf("demo-%s-%d", strings.ToLower(strings.TrimSuffix(org, ".com")), day),
				DateBegin: begin, DateEnd: begin.Add(24*time.Hour - time.Second),
				Policy:  map[string]string{"domain": "acme.io", "p": "quarantine", "sp": "quarantine", "pct": "100", "adkim": "r", "aspf": "r"},
				Records: []DMARCRecord{},
			}
			for _, s := range sources {
				if s.reporter != oi {
					continue
				}
				// Spread each source's week evenly over the seven days.
				n := s.count / 7
				if day == 1 {
					n = s.count - n*6
				}
				rep.Records = append(rep.Records, DMARCRecord{
					SourceIP: s.ip, PTR: s.ptr, Count: n, Disposition: s.disp, DKIM: s.dkim, SPF: s.spf,
					DKIMResult: s.dkimRes, SPFResult: s.spfRes, HeaderFrom: domain, DKIMDomain: s.dkimDom, SPFDomain: s.spfDom,
				})
			}
			if len(rep.Records) > 0 {
				out = append(out, rep)
			}
		}
	}
	return out
}

// DemoAlerts are the sample team's HostRepute alerts.
func DemoAlerts(now time.Time) []Alert {
	return []Alert{
		{ID: "demo-alert-uceprotect", CreatedAt: demoDay(now, 5, 7, 12), Kind: "listed", Severity: "warn", Asset: "203.0.113.25", Source: "UCEPROTECT Level 3",
			Title: "203.0.113.25 listed on UCEPROTECT L3", Detail: "Network-range listing, low impact.", DelistURL: "https://www.uceprotect.net/en/rblcheck.php"},
		{ID: "demo-alert-migration", CreatedAt: demoDay(now, 24, 11, 30), Kind: "reputation_change", Severity: "info", Asset: "news.acme.io",
			Title: "First send from news.acme.io after the Brevo migration", Detail: "Gmail spam rate peaked at 0.09%, back to 0.03% within a week. 1,033 Brevo suppressions imported first."},
	}
}

// DemoClient is the offline HostRepute. Monitoring changes (Add to
// monitoring, swap) live in the client and are persisted by the caller in
// the connection's monitored assets.
type DemoClient struct {
	mu        sync.Mutex
	now       func() time.Time
	monitored []string
	checks    map[string]time.Time
}

// NewDemoClient returns the demo HostRepute for the monitored assets.
func NewDemoClient(now func() time.Time, monitored []string) *DemoClient {
	if now == nil {
		now = time.Now
	}
	return &DemoClient{now: now, monitored: slices.Clone(monitored), checks: map[string]time.Time{}}
}

// Account implements Client.
func (d *DemoClient) Account(context.Context) (Account, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return DemoAccount(len(d.monitored)), nil
}

// Assets implements Client.
func (d *DemoClient) Assets(context.Context) ([]Asset, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return DemoAssets(d.now(), d.monitored), nil
}

// AddAsset implements Client; the demo plan monitors three assets.
func (d *DemoClient) AddAsset(_ context.Context, asset, typ, _ string) (Asset, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if slices.Contains(d.monitored, asset) {
		return DemoAssets(d.now(), []string{asset})[0], nil
	}
	if len(d.monitored) >= DemoLimit {
		return Asset{}, &LimitError{Limit: DemoLimit, Used: len(d.monitored), Team: DemoTeam}
	}
	d.monitored = append(d.monitored, asset)
	a := DemoAssets(d.now(), []string{asset})[0]
	if typ != "" {
		a.Type = typ
	}
	return a, nil
}

// RemoveAsset implements Client.
func (d *DemoClient) RemoveAsset(_ context.Context, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, a := range DemoAssets(d.now(), d.monitored) {
		if a.ID == id || a.Asset == id {
			d.monitored = slices.Delete(d.monitored, i, i+1)
			return nil
		}
	}
	return ErrNotFound
}

// Monitored returns the assets the demo team monitors now.
func (d *DemoClient) Monitored() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.monitored)
}

// Blocklists implements Client.
func (d *DemoClient) Blocklists(context.Context) (Blocklists, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return DemoBlocklists(d.now(), d.monitored), nil
}

// Postmaster implements Client.
func (d *DemoClient) Postmaster(_ context.Context, domain string, _, _ time.Time) ([]PostmasterDay, error) {
	return DemoPostmaster(d.now(), domain), nil
}

// SNDS implements Client.
func (d *DemoClient) SNDS(_ context.Context, ip string, _, _ time.Time) ([]SNDSDay, error) {
	return DemoSNDS(d.now(), ip), nil
}

// DMARCReports implements Client.
func (d *DemoClient) DMARCReports(_ context.Context, domain string, _, _ time.Time) ([]DMARCReport, error) {
	return DemoDMARC(d.now(), domain), nil
}

// Alerts implements Client.
func (d *DemoClient) Alerts(_ context.Context, since time.Time) ([]Alert, error) {
	var out []Alert
	for _, a := range DemoAlerts(d.now()) {
		if a.CreatedAt.After(since) {
			out = append(out, a)
		}
	}
	return out, nil
}

// demoCheckDuration is how long a demo re-check of 84 sources takes.
const demoCheckDuration = 8 * time.Second

// StartCheck implements Client: the demo check answers 84 sources over
// about eight seconds.
func (d *DemoClient) StartCheck(context.Context, []string) (Check, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	id := fmt.Sprintf("chk_%d", d.now().UnixNano())
	d.checks[id] = d.now()
	return Check{ID: id, Status: "running", Total: 84, StartedAt: d.now()}, nil
}

// Check implements Client.
func (d *DemoClient) Check(_ context.Context, id string) (Check, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	started, ok := d.checks[id]
	if !ok {
		return Check{}, ErrNotFound
	}
	elapsed := d.now().Sub(started)
	c := Check{ID: id, Status: "running", Total: 84, StartedAt: started}
	c.Answered = min(84, int(float64(84)*elapsed.Seconds()/demoCheckDuration.Seconds()))
	if elapsed >= demoCheckDuration {
		done := started.Add(demoCheckDuration)
		c.Status, c.Answered, c.FinishedAt = "done", 84, &done
	}
	return c, nil
}

// RequestDelisting implements Client.
func (d *DemoClient) RequestDelisting(_ context.Context, asset, source string) (Delisting, error) {
	return Delisting{ID: "dl_" + strings.ReplaceAll(asset, ".", "_"), Status: "requested", RequestedAt: d.now(), URL: "https://www.uceprotect.net/en/rblcheck.php"}, nil
}

// Revoke implements Client.
func (d *DemoClient) Revoke(context.Context) error { return nil }
