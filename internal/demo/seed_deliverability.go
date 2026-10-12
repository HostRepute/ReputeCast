package demo

import (
	"context"
	"encoding/json"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/hostrepute"
	"github.com/hostrepute/reputecast/internal/store"
)

func init() { seedDeliverability = seedDeliverabilityImpl }

// The sample sending setup's DNS (spec §6): SPF, DKIM and DMARC
// p=quarantine aligned on every provider, custom return paths, and the
// Postal IP 203.0.113.25 with forward-confirmed reverse DNS mta.acme.io.
var (
	sesDKIMTokens = []string{"4fpa3xq7utjl2kdrmw5a3zvyxnoqbe2h", "w6c2fzl3ryq7h4dk5xg2qkzv7m3jp4xa", "kd7a2hx9pq3mzl4w5vbn8rt6yc2ej7fs"}
	demoKey       = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA" + strings.Repeat("u3Zq8xN2pL7vR4kT9wY6bH1mC5", 13) + "IDAQAB"
	postmarkSel   = "20260807pm"
	postalSel     = "postal-x7kq2"
)

// demoRecords are the records each provider requires for news.acme.io.
func demoRecords(s *Seeder) []map[string]any {
	ses, postmark, postal := s.Providers["ses"].String(), s.Providers["postmark"].String(), s.Providers["postal"].String()
	var out []map[string]any
	rec := func(provider, purpose, typ, name, value string) {
		out = append(out, map[string]any{"provider_id": provider, "purpose": purpose, "type": typ, "name": name, "value": value, "status": "verified"})
	}
	for _, t := range sesDKIMTokens {
		rec(ses, "dkim", "CNAME", t+"._domainkey.news.acme.io", t+".dkim.amazonses.com")
	}
	rec(ses, "return_path", "MX", "bounce.news.acme.io", "10 feedback-smtp.us-east-1.amazonses.com")
	rec(ses, "spf", "TXT", "bounce.news.acme.io", "v=spf1 include:amazonses.com ~all")
	rec(postmark, "dkim", "TXT", postmarkSel+"._domainkey.news.acme.io", "k=rsa; p="+demoKey)
	rec(postmark, "return_path", "CNAME", "pm-bounces.news.acme.io", "pm.mtasv.net")
	rec(postal, "dkim", "TXT", postalSel+"._domainkey.news.acme.io", "v=DKIM1; t=s; h=sha256; p="+demoKey)
	rec(postal, "return_path", "CNAME", "rp.news.acme.io", "rp.mta.acme.io")
	rec(postal, "spf", "TXT", "news.acme.io", "ip4:203.0.113.25")
	return out
}

// DNSZone answers the demo's DNS: the acme.io names, the provider names
// they point to, and every built-in blocklist lookup of the demo assets
// (not listed).
func DNSZone() (*dnscheck.Zone, []string) {
	z := dnscheck.NewZone()
	z.Add("news.acme.io", "TXT", "v=spf1 include:amazonses.com include:spf.mtasv.net ip4:203.0.113.25 -all")
	z.Add("news.acme.io", "MX", "10 mx.acme.io")
	z.Add("acme.io", "TXT", "v=spf1 include:_spf.google.com -all")
	z.Add("acme.io", "MX", "1 aspmx.l.google.com")
	z.Add("_spf.google.com", "TXT", "v=spf1 ip4:35.190.247.0/24 ip4:64.233.160.0/19 -all")
	z.Add("_dmarc.acme.io", "TXT", "v=DMARC1; p=quarantine; sp=quarantine; adkim=r; aspf=r; rua=mailto:dmarc@reports.hostrepute.com")
	z.Add("amazonses.com", "TXT", "v=spf1 ip4:199.255.192.0/22 ip4:199.127.232.0/22 ip4:54.240.0.0/18 ip4:69.169.224.0/20 ip4:23.249.208.0/20 ip4:23.251.224.0/19 ip4:76.223.176.0/20 -all")
	z.Add("spf.mtasv.net", "TXT", "v=spf1 ip4:50.31.156.96/27 ip4:104.245.209.192/26 -all")
	z.Add("bounce.news.acme.io", "MX", "10 feedback-smtp.us-east-1.amazonses.com")
	z.Add("bounce.news.acme.io", "TXT", "v=spf1 include:amazonses.com ~all")
	z.Add("pm-bounces.news.acme.io", "CNAME", "pm.mtasv.net")
	z.Add("pm.mtasv.net", "TXT", "v=spf1 include:spf.mtasv.net -all")
	z.Add("rp.news.acme.io", "CNAME", "rp.mta.acme.io")
	z.Add("rp.mta.acme.io", "MX", "10 mta.acme.io")
	z.Add("rp.mta.acme.io", "TXT", "v=spf1 a:mta.acme.io -all")
	z.Add("mta.acme.io", "A", "203.0.113.25")
	z.Add("203.0.113.25", "PTR", "mta.acme.io.")
	for _, t := range sesDKIMTokens {
		z.Add(t+"._domainkey.news.acme.io", "CNAME", t+".dkim.amazonses.com")
		z.Add(t+".dkim.amazonses.com", "TXT", "p="+demoKey)
	}
	z.Add(postmarkSel+"._domainkey.news.acme.io", "TXT", "k=rsa; p="+demoKey)
	z.Add(postalSel+"._domainkey.news.acme.io", "TXT", "v=DKIM1; t=s; h=sha256; p="+demoKey)
	owned := []string{"acme.io"}
	for _, l := range dnscheck.Builtin {
		for _, a := range []string{"news.acme.io", "203.0.113.25"} {
			if n, err := l.QueryName(a); err == nil {
				owned = append(owned, n)
			}
		}
	}
	return z, owned
}

// DNSOverlay answers the demo names from DNSZone and everything else from r.
func DNSOverlay(r dnscheck.Resolver) dnscheck.Resolver {
	z, owned := DNSZone()
	return dnscheck.Overlay(z, owned, r)
}

// seedDeliverabilityImpl seeds Deliverability and HostRepute (spec §6,
// add-on §4): news.acme.io clean on 84 of 84 blocklists, the Postal IP
// listed on UCEPROTECT L3 with a delisting request pending, authentication
// aligned on every provider, 30 days of Postmaster and SNDS data, a week of
// DMARC reports, the incident timeline, and HostRepute connected to team
// "Acme Inc" monitoring news.acme.io, acme.io and 203.0.113.25.
func seedDeliverabilityImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]

	// The sending domain (the senders seeder may have created it already).
	domainID, ok := s.Domains["news.acme.io"]
	records, _ := json.Marshal(demoRecords(s))
	if !ok {
		auth := json.RawMessage(`{"spf":"pass","dkim":"pass","dmarc":"quarantine","bimi":"missing","return_path":"aligned"}`)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO sending_domains (id, workspace_id, domain, dns_records, auth_summary, status, last_checked_at, created_at)
			VALUES ($1,$2,'news.acme.io',$3,$4,'verified',$5,$6) ON CONFLICT (workspace_id, domain) DO NOTHING`,
			ID(), ws.ID, records, auth, s.Ago(65*time.Minute), s.Ago(400*24*time.Hour)); err != nil {
			return err
		}
		if err := s.Tx.QueryRow(ctx, `SELECT id FROM sending_domains WHERE workspace_id = $1 AND domain = 'news.acme.io'`, ws.ID).Scan(&domainID); err != nil {
			return err
		}
		s.Domains["news.acme.io"] = domainID
	}

	// Local DNS checks, run for real against the demo zone.
	z, owned := DNSZone()
	r := dnscheck.Overlay(z, owned, dnscheck.NewZone())
	checked := s.Ago(65 * time.Minute).Truncate(time.Minute)
	var dnsRecords json.RawMessage
	if err := s.Tx.QueryRow(ctx, `SELECT dns_records FROM sending_domains WHERE id = $1`, domainID).Scan(&dnsRecords); err != nil {
		return err
	}
	var reqs []dnscheck.Requirement
	var raw []struct {
		ProviderID string `json:"provider_id"`
		Purpose    string `json:"purpose"`
		Type       string `json:"type"`
		Name       string `json:"name"`
		Value      string `json:"value"`
	}
	_ = json.Unmarshal(dnsRecords, &raw)
	for _, x := range raw {
		reqs = append(reqs, dnscheck.Requirement{ProviderID: x.ProviderID, Purpose: x.Purpose, Type: x.Type, Name: x.Name, Value: x.Value})
	}
	if len(reqs) == 0 {
		_ = json.Unmarshal(records, &raw)
		for _, x := range raw {
			reqs = append(reqs, dnscheck.Requirement{ProviderID: x.ProviderID, Purpose: x.Purpose, Type: x.Type, Name: x.Name, Value: x.Value})
		}
	}
	q := store.New(s.Tx)
	save := func(asset, typ string, v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return q.UpsertAssetCheck(ctx, store.UpsertAssetCheckParams{WorkspaceID: ws.ID, Asset: asset, AssetType: typ, Result: b, CheckedAt: checked})
	}
	if err := save("news.acme.io", "domain", map[string]any{"auth": dnscheck.CheckDomain(ctx, r, "news.acme.io", reqs)}); err != nil {
		return err
	}
	if err := save("acme.io", "domain", map[string]any{"dmarc": dnscheck.CheckDMARC(ctx, r, "acme.io")}); err != nil {
		return err
	}
	if err := save("203.0.113.25", "ip", map[string]any{"rdns": dnscheck.CheckReverseDNS(ctx, r, netip.MustParseAddr("203.0.113.25"))}); err != nil {
		return err
	}

	// The built-in daily check (clean on its 8 lists), shown while
	// HostRepute is disconnected or unreachable.
	assets := []dnscheck.Asset{{Name: "news.acme.io", Kind: dnscheck.KindDomain}, {Name: "203.0.113.25", Kind: dnscheck.KindIP}}
	for _, l := range dnscheck.Builtin {
		for _, a := range assets {
			if !l.Applies(a) {
				continue
			}
			if err := q.InsertBlocklistResult(ctx, store.InsertBlocklistResultParams{
				ID: ID(), WorkspaceID: ws.ID, Via: "builtin", AssetType: a.Kind, Asset: a.Name, Source: l.Name, Impact: l.Impact,
				Rank: int32(l.Rank), DelistUrl: l.DelistURL, CheckedAt: checked.Add(time.Minute),
			}); err != nil {
				return err
			}
		}
	}
	builtinAt := checked.Add(time.Minute)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO deliverability_check_runs (id, workspace_id, via, trigger, status, assets, total, answered, started_at, finished_at)
		VALUES ($1,$2,'builtin','schedule','done',$3,8,8,$4,$5)`, ID(), ws.ID, []string{"news.acme.io", "203.0.113.25"}, builtinAt.Add(-8*time.Second), builtinAt); err != nil {
		return err
	}

	// HostRepute: connected by Linh on Aug 20 (53 days before the design's
	// today) to team "Acme Inc", synced 4 minutes ago from the demo HostRepute.
	access, err := s.Box.SealString("hr_demo_access", "hostrepute:"+ws.ID.String()+":token")
	if err != nil {
		return err
	}
	refresh, err := s.Box.SealString("hr_demo_refresh", "hostrepute:"+ws.ID.String()+":refresh")
	if err != nil {
		return err
	}
	connectedAt := s.designDay(53, 14, 12)
	acct := hostrepute.DemoAccount(len(hostrepute.DemoMonitored))
	limit := int32(acct.Team.MonitorLimit)
	if _, err := q.UpsertHostReputeConnection(ctx, store.UpsertHostReputeConnectionParams{
		WorkspaceID: ws.ID, Method: "oauth", AccountEmail: acct.Email, TeamName: acct.Team.Name, Scopes: acct.Scopes, TokenEnc: access, RefreshEnc: refresh,
		TokenExpiresAt: ptr(s.Now.Add(50 * time.Minute)), ApiUrl: hostrepute.DemoAPIURL, AccountID: acct.ID, TeamID: acct.Team.ID, Plan: acct.Team.Plan,
		MonitorLimit: &limit, Sources: int32(acct.Sources), CheckIntervalMinutes: int32(acct.CheckIntervalMinutes), DmarcRua: acct.DMARCRua,
		ConnectedBy: &linh.ID, ConnectedAt: connectedAt,
	}); err != nil {
		return err
	}
	syncedAt := s.Ago(4 * time.Minute)
	c := hostrepute.NewDemoClient(func() time.Time { return s.Now }, hostrepute.DemoMonitored)
	snap, err := hostrepute.Fetch(ctx, c, s.Now, s.Now.AddDate(0, 0, -hostrepute.PostmasterDays))
	if err != nil {
		return err
	}
	if _, err := hostrepute.Apply(ctx, s.Tx, ws.ID, snap, syncedAt); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO deliverability_check_runs (id, workspace_id, via, trigger, status, assets, total, answered, listed, started_at, finished_at)
		VALUES ($1,$2,'hostrepute','schedule','done',$3,84,84,1,$4,$5)`, ID(), ws.ID, hostrepute.DemoMonitored, syncedAt.Add(-20*time.Second), syncedAt); err != nil {
		return err
	}

	// Incidents ReputeCast recorded itself (HostRepute's alerts came with
	// the sync above). Campaign links are set once the campaigns exist.
	postal, ses := s.Providers["postal"], s.Providers["ses"]
	type inc struct {
		key, kind, severity, title, detail string
		at                                 time.Time
		resolved                           *time.Time
		provider                           uuid.UUID
		campaign                           string
	}
	incidents := []inc{
		{key: "warmup", kind: "warmup_slowdown", severity: "warn", title: "Postal warm-up slowdown, bounces 2.3%", detail: "Day 6 cap held at 2,600. Mostly Yahoo deferrals.",
			at: s.designDay(3, 10, 20), provider: postal, campaign: "October Product Update"},
		{key: "webhooks", kind: "webhooks_silent", severity: "bad", title: "Postal webhooks silent for 6 h", detail: "Worker restart on the Postal host; 1,212 events backfilled from the API.",
			at: s.designDay(10, 1, 30), resolved: ptr(s.designDay(10, 7, 30)), provider: postal},
		{key: "autopause", kind: "auto_pause", severity: "bad", title: "Weekly Digest #139 auto-paused on Yahoo", detail: "Yahoo deferral rate 14% for 3 min (421 TSS04). Resumed at 2 /s after 11 min.",
			at: s.designDay(19, 9, 20), resolved: ptr(s.designDay(19, 9, 31)), provider: ses, campaign: "Weekly Digest #139"},
	}
	ids := map[string]uuid.UUID{}
	for _, in := range incidents {
		id := ID()
		ids[in.key] = id
		provider := in.provider
		if _, err := s.Tx.Exec(ctx, `INSERT INTO incidents (id, workspace_id, occurred_at, kind, severity, title, detail, provider_id, resolved_at, source)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'reputecast')`, id, ws.ID, in.at, in.kind, in.severity, in.title, in.detail, &provider, in.resolved); err != nil {
			return err
		}
	}
	s.After(func(ctx context.Context, s *Seeder) error {
		for _, in := range incidents {
			if in.campaign == "" {
				continue
			}
			cid, ok := s.Campaigns[in.campaign]
			if !ok {
				continue
			}
			if _, err := s.Tx.Exec(ctx, `UPDATE incidents SET campaign_id = $2 WHERE id = $1`, ids[in.key], cid); err != nil {
				return err
			}
		}
		return nil
	})
	return nil
}
