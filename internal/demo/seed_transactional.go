package demo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/domains"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional"
)

// Transactional email (design expansion phase 1, brief §4): Acme Cloud,
// Acme's SaaS product at app.acme.io, sends its product mail through the
// Acme Newsletter workspace, from no-reply@notify.acme.io, over Postmark
// with Amazon SES as the fallback. Six templates with their versions, a
// week of messages whose numbers match the Transactional board, the latest
// messages of the log, the keys, the SMTP relay, the 9 Oct outage and the
// 3 Oct bounce alert.
//
// The brief's calendar reads Thursday 15 Oct 2026; days are counted back
// from the demo's "today" (designDay) like the other seeders.
func init() { seedTransactional = seedTransactionalImpl }

// SkipTransactional leaves the demo's transactional mail out: Acme Cloud's,
// and the paid tiers' Memberships emails (seed_earn.go).
// The transactional and SMTP relay tests set it in TestMain: they start
// from an empty Transactional tab.
var SkipTransactional bool

// Acme Cloud's sending domain and the relay's host.
const (
	txnDomain    = "notify.acme.io"
	txnRelayHost = "reputecast.acme.io"
	txnRelayIP   = "203.0.113.40"
)

// txnSESDKIM are notify.acme.io's SES Easy DKIM tokens.
var txnSESDKIM = []string{"n7f2k9x", "c4j8w1q", "u6h3d5m"}

// transactionalDemoZone adds notify.acme.io's records and the relay's host
// to the demo DNS zone (DeliverabilityDNSZone).
func transactionalDemoZone(z *dnscheck.Zone) {
	z.Add(txnDomain, "TXT", "v=spf1 include:amazonses.com include:spf.mtasv.net -all")
	z.Add(deliverabilityPostmarkSel+"._domainkey."+txnDomain, "TXT", "k=rsa; p="+deliverabilityDemoKey)
	z.Add("pm-bounces."+txnDomain, "CNAME", "pm.mtasv.net")
	for _, t := range txnSESDKIM {
		z.Add(t+"._domainkey."+txnDomain, "CNAME", t+".dkim.amazonses.com")
		z.Add(t+".dkim.amazonses.com", "TXT", "p="+deliverabilityDemoKey)
	}
	z.Add("bounce."+txnDomain, "MX", "10 feedback-smtp.us-east-1.amazonses.com")
	z.Add("bounce."+txnDomain, "TXT", "v=spf1 include:amazonses.com ~all")
	z.Add(txnRelayHost, "A", txnRelayIP)
}

// txnPlan is one template's week: what providers accepted, and what came
// of it. The numbers are the Transactional board's (last 7 days).
type txnPlan struct {
	key, name, category string
	sent                int // accepted by a provider, the samples and the outage included
	hard, soft          int // bounces
	deferred            int // still deferred
	complained          int
	last24              int // of sent, accepted in the last 24 hours
	bounced24           int
	deferred24          int
	quantiles           [][2]float64 // delivery seconds by quantile
	tags                []string
	from                string
	opens               bool
}

var txnPlans = []txnPlan{
	{key: "password-reset", name: "Password reset", category: "Account", sent: 3212, hard: 2, soft: 1, last24: 520,
		quantiles: [][2]float64{{0, 0.9}, {0.25, 1.25}, {0.35, 1.3}, {0.5, 1.4}, {0.9, 2.0}, {1, 4.0}}, tags: []string{"auth"}, from: "no-reply@notify.acme.io"},
	{key: "magic-link", name: "Magic sign-in link", category: "Account", sent: 6881, hard: 9, soft: 3, deferred: 2, last24: 1067, bounced24: 2, deferred24: 2,
		quantiles: [][2]float64{{0, 0.7}, {0.5, 1.2}, {0.9, 1.3}, {1, 3.0}}, tags: []string{"auth"}, from: "no-reply@notify.acme.io"},
	{key: "welcome", name: "Welcome to Acme Cloud", category: "Account", sent: 1904, hard: 5, soft: 2, deferred: 1, last24: 297, bounced24: 1,
		quantiles: [][2]float64{{0, 1.0}, {0.05, 1.3}, {0.5, 1.9}, {1, 4.5}}, tags: []string{"onboarding"}, from: "no-reply@notify.acme.io"},
	{key: "invoice-paid", name: "Invoice paid", category: "Billing", sent: 412, last24: 62,
		quantiles: [][2]float64{{0, 1.4}, {0.5, 2.3}, {1, 4.0}}, tags: []string{"billing"}, from: "billing@notify.acme.io"},
	{key: "trial-ending", name: "Trial ends in 3 days", category: "Reminders", sent: 71, soft: 1, last24: 9, bounced24: 1,
		quantiles: [][2]float64{{0, 1.2}, {0.5, 2.0}, {1, 3.5}}, tags: []string{"trial"}, from: "no-reply@notify.acme.io"},
	{key: "usage-summary", name: "Weekly usage summary", category: "Product updates", sent: 1240, hard: 3, soft: 1, deferred: 3, complained: 1, last24: 1, opens: true,
		quantiles: [][2]float64{{0, 0.9}, {0.1, 1.25}, {0.15, 1.3}, {0.5, 1.6}, {1, 3.5}}, tags: []string{"weekly"}, from: "no-reply@notify.acme.io"},
}

var txnCategories = []struct{ name, kind, desc string }{
	{"Account", transactional.KindEssential, "Sign-in, password and account emails"},
	{"Billing", transactional.KindEssential, "Receipts and invoices"},
	{"Reminders", transactional.KindOptional, "Trial and renewal reminders"},
	{"Product updates", transactional.KindOptional, "Weekly usage summaries"},
}

// txnSeed carries what the steps share.
type txnSeed struct {
	s                 *Seeder
	ws                store.Workspace
	rng               *rand.Rand
	postmark, ses     uuid.UUID
	domain            uuid.UUID
	senders           map[string]uuid.UUID
	categories        map[string]uuid.UUID
	templates         map[string]uuid.UUID
	content           map[string]transactional.DemoTemplate
	live              map[string]int
	keys              map[string]uuid.UUID
	t0                time.Time // the newest message of the log
	messages, txnRows [][]any
	events            [][]any
	contents          [][]any
}

func seedTransactionalImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	if SkipTransactional || s.Providers["postmark"] == uuid.Nil || s.Providers["ses"] == uuid.Nil {
		return nil
	}
	t := &txnSeed{s: s, ws: ws, rng: rand.New(rand.NewPCG(15, 1026)), //nolint:gosec // deterministic sample data
		postmark: s.Providers["postmark"], ses: s.Providers["ses"], senders: map[string]uuid.UUID{}, categories: map[string]uuid.UUID{},
		templates: map[string]uuid.UUID{}, content: map[string]transactional.DemoTemplate{}, live: map[string]int{}, keys: map[string]uuid.UUID{},
		t0: s.Ago(60 * time.Second)}
	for _, step := range []func(context.Context) error{t.domainAndSenders, t.settings, t.route, t.categoriesAndOptOuts, t.templatesAndVersions, t.keysAndRelay,
		t.week, t.samples, t.history, t.flush, t.incidents, t.webhooks, t.audit} {
		if err := step(ctx); err != nil {
			return err
		}
	}
	// The suppressed addresses take the place of entries the audience
	// seeder writes once every seeder has run.
	s.After(func(ctx context.Context, _ *Seeder) error { return t.suppressions(ctx) })
	return nil
}

func (t *txnSeed) user(email string) *uuid.UUID {
	u, ok := t.s.Users[email]
	if !ok {
		return nil
	}
	id := u.ID
	return &id
}

// domainAndSenders adds notify.acme.io, verified on Postmark and SES, and
// its two senders.
func (t *txnSeed) domainAndSenders(ctx context.Context) error {
	s, ws := t.s, t.ws
	q := store.New(s.Tx)
	marcus := t.user("marcus@acme.io")
	created := s.designDay(43, 15, 40)
	t.domain = ID()
	if _, err := q.DomCreate(ctx, store.DomCreateParams{ID: t.domain, WorkspaceID: ws.ID, Domain: txnDomain, Note: "transactional mail only", DnsRecords: json.RawMessage(`[]`),
		CreatedBy: marcus, CreatedAt: created}); err != nil {
		return err
	}
	if err := q.TxnSetDomainTransactional(ctx, store.TxnSetDomainTransactionalParams{ID: t.domain, Transactional: true}); err != nil {
		return err
	}
	s.Domains[txnDomain] = t.domain
	postal := s.Providers["postal"]
	for _, sn := range []struct{ name, email, reply string }{
		{"Acme Cloud", "no-reply@notify.acme.io", "support@acme.io"},
		{"Acme Billing", "billing@notify.acme.io", "billing@acme.io"},
	} {
		id := ID()
		ver := json.RawMessage(`{}`)
		if postal != uuid.Nil {
			ver, _ = json.Marshal(map[string]string{postal.String(): "not_used"})
		}
		if _, err := q.SenderCreate(ctx, store.SenderCreateParams{ID: id, WorkspaceID: ws.ID, DomainID: t.domain, Name: sn.name, Email: sn.email, ReplyTo: sn.reply,
			Verification: ver, CreatedBy: marcus, CreatedAt: created.Add(12 * time.Minute)}); err != nil {
			return err
		}
		s.Senders[sn.email] = id
		t.senders[sn.email] = id
	}
	// The records Postmark and SES need, checked against the demo zone.
	s.After(func(ctx context.Context, s *Seeder) error {
		q := store.New(s.Tx)
		pm, ses := t.postmark.String(), t.ses.String()
		recs := []domains.Record{
			{ProviderID: pm, Purpose: domains.PurposeDKIM, Type: "TXT", Name: deliverabilityPostmarkSel + "._domainkey." + txnDomain, Value: "k=rsa; p=" + deliverabilityDemoKey, Source: "provider"},
			{ProviderID: pm, Purpose: domains.PurposeReturnPath, Type: "CNAME", Name: "pm-bounces." + txnDomain, Value: "pm.mtasv.net", Source: "provider"},
		}
		for _, tok := range txnSESDKIM {
			recs = append(recs, domains.Record{ProviderID: ses, Purpose: domains.PurposeDKIM, Type: "CNAME", Name: tok + "._domainkey." + txnDomain, Value: tok + ".dkim.amazonses.com", Source: "provider"})
		}
		recs = append(recs,
			domains.Record{ProviderID: ses, Purpose: domains.PurposeReturnPath, Type: "MX", Name: "bounce." + txnDomain, Value: "10 feedback-smtp.us-east-1.amazonses.com", Source: "provider"},
			domains.Record{ProviderID: ses, Purpose: domains.PurposeSPF, Type: "TXT", Name: "bounce." + txnDomain, Value: "v=spf1 include:amazonses.com ~all", Source: "provider"},
			domains.Record{ProviderID: ses, Purpose: domains.PurposeSPF, Type: "TXT", Name: txnDomain, Mechanisms: []string{"include:amazonses.com", "include:spf.mtasv.net"}, Source: "reputecast"},
		)
		if err := q.DomUpdateRecords(ctx, store.DomUpdateRecordsParams{ID: t.domain, DnsRecords: domains.EncodeRecords(recs), UpdatedAt: s.Ago(time.Hour)}); err != nil {
			return err
		}
		svc := SendersDomainService(s.Tx, s.Cfg.HTTP.PublicURL)
		svc.Now = func() time.Time { return s.Ago(4 * time.Minute) }
		d, err := q.DomGetAny(ctx, t.domain)
		if err != nil {
			return err
		}
		_, err = svc.Check(ctx, d)
		return err
	})
	return nil
}

// settings: contact linking on, content kept 30 days, the default limits,
// Acme Cloud as the default sender.
func (t *txnSeed) settings(ctx context.Context) error {
	_, err := t.s.Tx.Exec(ctx, `INSERT INTO transactional_settings (workspace_id, default_sender_id, default_reply_to, guide, updated_by, updated_at)
		VALUES ($1, $2, 'support@acme.io', '{"step":"done","completed":true}', $3, $4)`,
		t.ws.ID, t.senders["no-reply@notify.acme.io"], t.user("marcus@acme.io"), t.s.designDay(43, 15, 58))
	return err
}

// route: Postmark, then Amazon SES with automatic failover; first in line
// on shared providers. SES gets its configuration set for the stream.
func (t *txnSeed) route(ctx context.Context) error {
	s := t.s
	splits, _ := json.Marshal([]map[string]any{{"provider_id": t.postmark, "weight": 100}})
	if _, err := s.Tx.Exec(ctx, `INSERT INTO routing_rules (id, workspace_id, stream, priority, name, conditions, splits, fallback_provider_id, updated_by, updated_at)
		VALUES ($1, $2, 'transactional', 100, '', '{}', $3, $4, $5, $6)`, ID(), t.ws.ID, splits, t.ses, t.user("marcus@acme.io"), s.designDay(43, 16, 0)); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO stream_policies (workspace_id, stream, failover, priority) VALUES ($1, 'transactional', 'automatic', true)
		ON CONFLICT (workspace_id, stream) DO UPDATE SET failover = EXCLUDED.failover, priority = EXCLUDED.priority`, t.ws.ID); err != nil {
		return err
	}
	_, err := s.Tx.Exec(ctx, `UPDATE providers SET config = jsonb_set(config, '{configuration_sets}', coalesce(config->'configuration_sets', '{}'::jsonb) || '{"transactional":"rc-transactional"}'::jsonb)
		WHERE id = $1`, t.ses)
	return err
}

// categoriesAndOptOuts: Account and Billing (Essential), Reminders (2
// opted out) and Product updates (31 opted out, Kim Tran among them).
func (t *txnSeed) categoriesAndOptOuts(ctx context.Context) error {
	s := t.s
	q := store.New(s.Tx)
	for i, c := range txnCategories {
		id := ID()
		if err := q.TxnCategoryInsert(ctx, store.TxnCategoryInsertParams{ID: id, WorkspaceID: t.ws.ID, Name: c.name, Kind: c.kind, Description: c.desc,
			CreatedBy: t.user("marcus@acme.io"), CreatedAt: s.designDay(43, 16, 10).Add(time.Duration(i) * time.Minute)}); err != nil {
			return err
		}
		t.categories[c.name] = id
	}
	optOut := func(email, category, source string, at time.Time) error {
		_, err := s.Tx.Exec(ctx, `INSERT INTO transactional_opt_outs (workspace_id, email, category_id, opted_out_at, source) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT DO NOTHING`, t.ws.ID, email, t.categories[category], at, source)
		return err
	}
	if err := optOut("kim.tran@example.com", "Product updates", "preference_center", s.designDay(17, 19, 20)); err != nil {
		return err
	}
	if err := optOut("m.ortiz@example.com", "Product updates", "unsubscribe_link", s.designDay(8, 8, 41)); err != nil {
		return err
	}
	sources := []string{"preference_center", "unsubscribe_link", "one_click"}
	for i := 0; i < 29; i++ {
		if err := optOut(fmt.Sprintf("member%03d@example.com", i+1), "Product updates", sources[i%3], s.designDay(35-i, 9+i%9, (i*7)%60)); err != nil {
			return err
		}
	}
	for i, e := range []string{"jo.lee@example.com", "dev+trial@contoso.example"} {
		if err := optOut(e, "Reminders", sources[i], s.designDay(12+i*5, 18, 5)); err != nil {
			return err
		}
	}
	return nil
}

// templatesAndVersions writes the six templates (seed_transactional_templates.go)
// with their published versions and drafts: only trial-ending's differs
// from its live version (Daniel Park's v2).
func (t *txnSeed) templatesAndVersions(ctx context.Context) error {
	s := t.s
	q := store.New(s.Tx)
	for n, tpl := range txSeedTemplates() {
		p := t.plan(tpl.key)
		if p == nil {
			return fmt.Errorf("no week planned for %s", tpl.key)
		}
		var sender *uuid.UUID
		if tpl.sender != "" {
			id := t.senders[tpl.sender]
			sender = &id
		}
		last := tpl.versions[len(tpl.versions)-1]
		draft := last.content
		savedBy, savedAt := last.by, s.Ago(last.ago)
		if tpl.draft != nil {
			draft = *tpl.draft
			savedBy, savedAt = tpl.draftBy, s.Ago(tpl.draftAgo)
		}
		dc, err := draft.demo()
		if err != nil {
			return fmt.Errorf("%s: %w", tpl.key, err)
		}
		id := ID()
		t.templates[tpl.key] = id
		cat := t.categories[tpl.category]
		first := tpl.versions[0]
		// The list shows templates in the order they were created: the
		// board's, password-reset first, all before the oldest v1.
		created := s.Ago(201*24*time.Hour - time.Duration(n)*time.Minute)
		if err := q.TxnTemplateInsert(ctx, store.TxnTemplateInsertParams{ID: id, WorkspaceID: t.ws.ID, Key: tpl.key, Name: tpl.name, Description: tpl.description, CategoryID: &cat,
			Subject: dc.Subject, Preheader: dc.Preheader, EditorMode: "visual", Mjml: dc.MJML, Html: dc.HTML, PlainText: "", Variables: mustJSON(dc.Variables),
			TestData: mustJSON(tpl.testData), SenderID: sender, ReplyTo: "", TrackOpens: tpl.opens, TrackClicks: false,
			DraftHash: transactional.DemoHash(dc, sender, "", tpl.opens, false), Now: created, CreatedBy: t.user(first.by), Origin: json.RawMessage(`{}`)}); err != nil {
			return err
		}
		for i, v := range tpl.versions {
			vc, err := v.content.demo()
			if err != nil {
				return fmt.Errorf("%s v%d: %w", tpl.key, i+1, err)
			}
			at := s.Ago(v.ago)
			version := int32(i + 1)
			if err := q.TxnVersionInsert(ctx, store.TxnVersionInsertParams{TemplateID: id, Version: version, Subject: vc.Subject, Preheader: vc.Preheader, EditorMode: "visual",
				Mjml: vc.MJML, Html: vc.HTML, PlainText: "", Variables: mustJSON(vc.Variables), TestData: mustJSON(tpl.testData), SenderID: sender, ReplyTo: "",
				TrackOpens: tpl.opens, TrackClicks: false, ContentHash: transactional.DemoHash(vc, sender, "", tpl.opens, false), SizeBytes: int32(len(vc.HTML)),
				Change: v.change, PublishedBy: t.user(v.by), PublishedAt: at}); err != nil {
				return err
			}
			if err := q.TxnTemplatePublish(ctx, store.TxnTemplatePublishParams{WorkspaceID: t.ws.ID, ID: id, Version: &version, Now: at, LiveBy: t.user(v.by), Variables: mustJSON(vc.Variables)}); err != nil {
				return err
			}
			t.live[tpl.key] = i + 1
			t.content[tpl.key] = vc
		}
		// The draft: its variables (publishing sets the live ones), who
		// saved it and when.
		revision := int32(len(tpl.versions))
		if tpl.draft != nil {
			revision++
		}
		if _, err := s.Tx.Exec(ctx, `UPDATE transactional_templates SET variables = $2, draft_revision = $3, draft_saved_by = $4, draft_saved_at = $5 WHERE id = $1`,
			id, mustJSON(dc.Variables), revision, t.user(savedBy), savedAt); err != nil {
			return err
		}
		if err := q.TxnTemplateInsertDraftRevision(ctx, store.TxnTemplateInsertDraftRevisionParams{TemplateID: id, Revision: revision, Subject: dc.Subject, Preheader: dc.Preheader,
			Mjml: dc.MJML, Html: dc.HTML, Variables: mustJSON(dc.Variables), SavedBy: t.user(savedBy), SavedAt: savedAt, Label: "", Source: "manual"}); err != nil {
			return err
		}
	}
	return nil
}

// keysAndRelay: the production and staging send-only keys, the Billing
// service's SMTP credential, and the relay that Linh Nguyen turned on.
func (t *txnSeed) keysAndRelay(ctx context.Context) error {
	s := t.s
	type key struct {
		name, prefix, head, tail, by, ip, mode, preset string
		created, lastUsed                              time.Time
		expires                                        *time.Time
		categories                                     []uuid.UUID
		tests                                          []string
	}
	exp := time.Date(2027, 10, 15, 12, 0, 0, 0, time.UTC)
	for _, k := range []key{
		{name: "Acme Cloud · production", prefix: "rc_live_", head: "4c1e", tail: "d07b", by: "marcus@acme.io", ip: "198.51.100.24", mode: "live", preset: "send_only",
			created: s.designDay(43, 16, 5), lastUsed: s.Ago(2 * time.Minute), expires: &exp},
		{name: "Acme Cloud · staging", prefix: "rc_test_", head: "9a3f", tail: "12ce", by: "marcus@acme.io", ip: "10.0.8.15", mode: "test", preset: "send_only",
			created: s.designDay(43, 16, 7), lastUsed: s.Ago(time.Hour), tests: []string{"qa+*@acme.io"}},
		{name: "Billing service (SMTP)", prefix: "rc_smtp_", head: "77b0", tail: "a9e4", by: "linh@acme.io", ip: "198.51.100.40", mode: "live", preset: "smtp",
			created: s.designDay(30, 11, 20), lastUsed: s.Ago(38 * time.Minute), categories: []uuid.UUID{t.categories["Billing"]}},
	} {
		id := ID()
		_, hash := demoKey(k.prefix, k.head, k.tail)
		cats := k.categories
		if cats == nil {
			cats = []uuid.UUID{}
		}
		tests := k.tests
		if tests == nil {
			tests = []string{}
		}
		client := "acme-cloud/2.14 (node)"
		if k.preset == "smtp" {
			client = "SMTP relay"
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO api_keys (id, workspace_id, kind, name, prefix, token_hash, permissions, scope, created_by, created_at, expires_at,
			last_used_at, last_used_ip, last_used_client, last_call, preset, mode, templates, categories, test_addresses, rate_per_second)
			VALUES ($1, $2, 'api_key', $3, $4, $5, $6, '{}', $7, $8, $9, $10, $11, $12, $13, $14, $15, '{}', $16, $17, 50)`,
			id, t.ws.ID, k.name, k.prefix+k.head+"…"+k.tail, hash, []string{string(authz.TransactionalSend)}, t.user(k.by), k.created, k.expires,
			k.lastUsed, addr(k.ip), client, "POST /transactional/messages", k.preset, k.mode, cats, tests); err != nil {
			return err
		}
		t.keys[k.name] = id
		s.Extra["key:"+k.name] = id
	}
	// The relay: on since 15 Sep, Let's Encrypt's certificate valid until 2
	// Dec, every check passing. It's the demo's, so no listener starts and
	// the periodic check leaves it be until an instance admin changes it.
	now := s.Ago(6 * time.Minute)
	status, _ := json.Marshal(map[string]any{"demo": true, "checkedAt": now, "from": "this server", "checks": []map[string]any{
		{"key": "dns", "ok": true, "detail": "DNS · " + txnRelayHost + " points to " + txnRelayIP},
		{"key": "starttls", "ok": true, "port": 587, "detail": "Port 587 reachable from the internet"},
		{"key": "tls", "ok": true, "port": 465, "detail": "Port 465 reachable from the internet"},
		{"key": "certificate", "ok": true, "detail": "Certificate valid until 2 Dec 2026"},
	}})
	notAfter := s.designDay(-48, 12, 0)
	_, err := s.Tx.Exec(ctx, `UPDATE smtp_relay SET enabled = true, hostname = $1, starttls_port = 587, tls_port = 465, cert_source = 'acme', status = $2,
		cert_not_after = $3, cert_issuer = 'Let''s Encrypt', enabled_by = $4, enabled_at = $5, updated_by = $4, updated_at = $5`,
		txnRelayHost, status, notAfter, t.user("linh@acme.io"), s.designDay(30, 11, 12))
	return err
}

// suppressions: jordan@example.org hard-bounced on 2 Oct (Postmark) and
// nine other addresses stopped this week's sends; billing@northwind is a
// role address, marketing only. The ten bounces and complaints are among
// the Suppressions board's 1,874: each takes the place of an older entry
// Postmark reported for the same reason, so the board's numbers hold (the
// role entry is the one more).
func (t *txnSeed) suppressions(ctx context.Context) error {
	s := t.s
	pm := t.postmark
	take := func(email, reason, code, detail string, at time.Time) error {
		tag, err := s.Tx.Exec(ctx, `UPDATE suppressions SET email = $3, original_code = $4, detail = $5, created_at = $6, context = 'Transactional'
			WHERE id = (SELECT id FROM suppressions WHERE workspace_id = $1 AND reason = $2 AND source_provider_id = $7 AND source = 'provider_sync' AND context = ''
			            ORDER BY created_at, id LIMIT 1)`, t.ws.ID, reason, email, code, detail, at, pm)
		if err != nil || tag.RowsAffected() == 1 {
			return err
		}
		_, err = s.Tx.Exec(ctx, `INSERT INTO suppressions (id, workspace_id, email, reason, source, source_provider_id, original_code, detail, created_at, scope, context)
			VALUES ($1, $2, $3, $4, 'webhook', $5, $6, $7, $8, 'all', 'Transactional') ON CONFLICT (workspace_id, email) DO NOTHING`,
			ID(), t.ws.ID, email, reason, pm, code, detail, at)
		return err
	}
	if err := take("jordan@example.org", "hard_bounce", "550 5.1.1", "mailbox does not exist", s.designDay(13, 8, 14)); err != nil {
		return err
	}
	for i, e := range txnSuppressed {
		reason, code, detail := txnSuppressedReason(i)
		if err := take(e, reason, code, detail, txnSuppressedAt(s, i)); err != nil {
			return err
		}
	}
	_, err := s.Tx.Exec(ctx, `INSERT INTO suppressions (id, workspace_id, email, reason, source, original_code, detail, created_at, scope, context)
		VALUES ($1, $2, 'billing@northwind.example', 'role', 'import', '', 'role address', $3, 'marketing', '') ON CONFLICT (workspace_id, email) DO NOTHING`,
		ID(), t.ws.ID, s.designDay(60, 10, 0))
	return err
}

// txnSuppressedReason: every third address complained, the others bounced.
func txnSuppressedReason(i int) (reason, code, detail string) {
	if i%3 == 2 {
		return "complaint", "abuse", "Feedback loop report"
	}
	return "hard_bounce", "550 5.1.1", "user unknown"
}

// txnSuppressedAt is when the address was suppressed: before the Brevo
// import, so the Suppressions board's first page stays as drawn.
func txnSuppressedAt(s *Seeder, i int) time.Time { return s.designDay(26+i, 9+i%6, 10+i) }

// txnSuppressed are the other suppressed addresses this week's calls named.
var txnSuppressed = []string{"old.user@hotmial.com", "nina.k@gmial.com", "ops-alerts@contoso.example", "ted@yaho.com", "r.patel@example.net",
	"former.employee@northwind.example", "test1234@example.com", "lucas.b@outlook.com", "hello@initech.example"}

// txnMsg is one message being built.
type txnMsg struct {
	id                 uuid.UUID
	plan               *txnPlan
	key                string // template key; "" for Raw (SMTP)
	email, name        string
	contact            *uuid.UUID
	mailbox            string
	status             string
	accepted           time.Time
	deliverAfter       time.Duration
	provider           *uuid.UUID
	pmid               string
	data               map[string]any
	subject            string
	keyName            string
	ip                 string
	idem               string
	tags               []string
	metadata           map[string]string
	test, testDeliver  bool
	reject             string
	rejectDetail       map[string]any
	errorText          string
	bounce             string // hard, soft
	bounceCode         string
	bounceDiag         string
	deferrals          []time.Duration
	complaint          bool
	opened             *time.Time
	failover           bool
	attempts           []map[string]any
	source             string
	attachments        []map[string]any
	keepContent        bool
	contentRemoved     *time.Time
	fromEmail          string
	delivDetail        string
	delivCode          string
	failedOverAt       *time.Time
	routeFailedAtStart bool
	// rawCategory is the category Raw (SMTP) mail named (X-ReputeCast-Category,
	// or the credential's only one).
	rawCategory string
}

var txnFirstNames = []string{"Sam", "Priya", "Alex", "Jordan", "Kim", "Lena", "Mateo", "Aisha", "Noah", "Mia", "Lucas", "Zoe", "Omar", "Ivy", "Hugo", "Nora", ""}

var txnMailboxes = []struct {
	key, domain string
	weight      float64
}{
	{"gmail", "gmail.com", 45}, {"outlook", "outlook.com", 22}, {"yahoo", "yahoo.com", 10}, {"icloud", "icloud.com", 8}, {"other", "contoso.example", 8}, {"other", "northwind.example", 7},
}

func (t *txnSeed) mailbox() (string, string) {
	r := t.rng.Float64() * 100
	for _, m := range txnMailboxes {
		if r < m.weight {
			return m.key, m.domain
		}
		r -= m.weight
	}
	return "other", "example.com"
}

// quantile reads a piecewise-linear quantile function.
func quantile(q [][2]float64, p float64) float64 {
	for i := 1; i < len(q); i++ {
		if p <= q[i][0] {
			a, b := q[i-1], q[i]
			if b[0] == a[0] {
				return b[1]
			}
			return a[1] + (b[1]-a[1])*(p-a[0])/(b[0]-a[0])
		}
	}
	return q[len(q)-1][1]
}

func (t *txnSeed) hex(n int) string {
	const digits = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = digits[t.rng.IntN(16)]
	}
	return string(b)
}

// dataFor is a call's variables (sensitive values masked, as the log keeps them).
func (t *txnSeed) dataFor(key, first string, at time.Time) map[string]any {
	d := map[string]any{}
	if first != "" {
		d["first_name"] = first
	}
	switch key {
	case "password-reset":
		d["reset_url"], d["expires_in"] = transactional.Mask, "30 minutes"
	case "magic-link":
		delete(d, "first_name")
		d["sign_in_url"], d["expires_in"] = transactional.Mask, "15 minutes"
	case "welcome":
		d["dashboard_url"] = "https://app.acme.io/"
	case "invoice-paid":
		delete(d, "first_name")
		n := 300 + t.rng.IntN(150)
		d["invoice_number"], d["amount"], d["paid_on"], d["plan_name"] = fmt.Sprintf("INV-2026-%04d", n), []string{"$49.00", "$19.00", "$199.00"}[n%3], at.Format("2006-01-02"), []string{"Pro", "Starter", "Team"}[n%3]
		d["invoice_url"] = "https://app.acme.io/billing/" + d["invoice_number"].(string)
	case "trial-ending":
		d["days_left"], d["upgrade_url"] = 3, "https://app.acme.io/billing/plans"
	case "usage-summary":
		d["projects"], d["deploys"], d["usage_url"] = 1+t.rng.IntN(9), t.rng.IntN(120), "https://app.acme.io/usage"
	}
	return d
}

func subjectFor(key string, d map[string]any) string {
	switch key {
	case "password-reset":
		return "Reset your Acme Cloud password"
	case "magic-link":
		return "Your sign-in link for Acme Cloud"
	case "welcome":
		n, _ := d["first_name"].(string)
		if n == "" {
			n = "there"
		}
		return "Welcome to Acme Cloud, " + n
	case "invoice-paid":
		return fmt.Sprintf("Invoice %s paid · %s", d["invoice_number"], d["amount"])
	case "trial-ending":
		return "Your Acme Cloud trial ends in 3 days"
	case "usage-summary":
		return "Your week in Acme Cloud"
	}
	return ""
}

func (t *txnSeed) plan(key string) *txnPlan {
	for i := range txnPlans {
		if txnPlans[i].key == key {
			return &txnPlans[i]
		}
	}
	return nil
}

// week builds the bulk of the last seven days per template: the numbers of
// the board minus the samples, history and outage messages added later.
func (t *txnSeed) week(ctx context.Context) error {
	s := t.s
	type reserved struct{ sent, hard, soft, deferred, complained, last24, bounced24, deferred24 int }
	// What the samples, the outage and this week's history already take.
	res := map[string]reserved{
		"password-reset": {sent: 1 + 1 + 13 + 1, last24: 2},           // Sam and Lucía today; 13 outage messages and Sam's on 9 Oct
		"magic-link":     {sent: 16},                                  // outage messages
		"welcome":        {sent: 1 + 7, last24: 1},                    // Priya today; outage messages
		"invoice-paid":   {sent: 1, last24: 1},                        // billing@northwind today
		"trial-ending":   {sent: 1, soft: 1, last24: 1, bounced24: 1}, // ops@contoso today
		"usage-summary":  {sent: 1, last24: 1},                        // Alex today
	}
	for i := range txnPlans {
		p := &txnPlans[i]
		r := res[p.key]
		n := p.sent - r.sent
		older := n - (p.last24 - r.last24)
		hard, soft := p.hard-r.hard, p.soft-r.soft
		deferred := p.deferred - r.deferred
		complained := p.complained - r.complained
		// The last 24 hours' share of the bounces (hard first) and deferrals;
		// the rest goes to the older days.
		bounced24, deferred24 := p.bounced24-r.bounced24, p.deferred24-r.deferred24
		hard24 := min(hard, bounced24)
		soft24 := bounced24 - hard24
		// Delivery times by quantile, shuffled.
		delivered := n - hard - soft - deferred
		hard, soft, deferred = hard-hard24, soft-soft24, deferred-deferred24
		times := make([]float64, delivered)
		for j := range times {
			times[j] = quantile(p.quantiles, (float64(j)+0.5)/float64(delivered))
		}
		t.rng.Shuffle(len(times), func(a, b int) { times[a], times[b] = times[b], times[a] })
		di := 0
		for j := 0; j < n; j++ {
			m := &txnMsg{id: ID(), plan: p, key: p.key, keyName: "Acme Cloud · production", ip: "198.51.100.24", tags: p.tags, source: "api", fromEmail: p.from}
			mbp, dom := t.mailbox()
			first := txnFirstNames[t.rng.IntN(len(txnFirstNames))]
			local := strings.ToLower(first)
			if local == "" {
				local = "user"
			}
			m.mailbox, m.email = mbp, fmt.Sprintf("%s.%05d@%s", local, t.rng.IntN(100000), dom)
			m.metadata = map[string]string{"user_id": fmt.Sprintf("u_%05d", 10000+t.rng.IntN(89999))}
			// When: the older ones over days 1 to 7, the rest in the last 24
			// hours (but before the log's sample rows).
			if j < older {
				if p.key == "usage-summary" {
					// The weekly batch: Monday, 08:00 to 08:20.
					m.accepted = s.designDay(3, 8, 0).Add(time.Duration(t.rng.IntN(20*60)) * time.Second)
				} else {
					m.accepted = s.Ago(24*time.Hour + time.Duration(t.rng.Int64N(int64(6*24*time.Hour-30*time.Minute))))
				}
			} else if hard24 > 0 || soft24 > 0 {
				// The day's bounces are two hours old or more: the last hour
				// stays under the alert's limit (no alert is active).
				m.accepted = s.Ago(2*time.Hour + time.Duration(t.rng.Int64N(int64(21*time.Hour+30*time.Minute))))
			} else {
				m.accepted = s.Ago(15*time.Minute + time.Duration(t.rng.Int64N(int64(23*time.Hour+15*time.Minute))))
			}
			// Both stay half an hour inside the last 24 hours, so the day's
			// counts hold while the data ages (tests share one seed for
			// minutes).
			m.data = t.dataFor(p.key, first, m.accepted)
			m.subject = subjectFor(p.key, m.data)
			m.name = first
			in24 := j >= older
			switch {
			case in24 && hard24 > 0:
				hard24--
				m.status, m.bounce = "bounced", "hard"
			case in24 && soft24 > 0:
				soft24--
				m.status, m.bounce = "bounced", "soft"
			case in24 && deferred24 > 0:
				deferred24--
				m.status = "deferred"
				m.deferrals = []time.Duration{time.Duration(2+t.rng.IntN(8)) * time.Second}
			case !in24 && hard > 0 && (j%37 == 5 || older-j <= hard+soft+deferred):
				hard--
				m.status, m.bounce = "bounced", "hard"
			case !in24 && soft > 0 && (j%41 == 7 || older-j <= soft+deferred):
				soft--
				m.status, m.bounce = "bounced", "soft"
			case !in24 && deferred > 0 && (j%53 == 3 || older-j <= deferred):
				deferred--
				m.status = "deferred"
				m.deferrals = []time.Duration{time.Duration(3+t.rng.IntN(6)) * time.Second}
			default:
				m.status = "delivered"
				if di < len(times) {
					m.deliverAfter = time.Duration(times[di] * float64(time.Second))
				} else {
					m.deliverAfter = 1500 * time.Millisecond
				}
				di++
				if complained > 0 && !in24 && j >= older/2 {
					m.complaint = true
					m.status = "complained"
					complained--
				}
				if p.opens && t.rng.Float64() < 0.38 {
					o := m.accepted.Add(m.deliverAfter + time.Duration(1+t.rng.IntN(240))*time.Minute)
					if o.Before(s.Now) {
						m.opened = &o
					}
				}
			}
			switch m.bounce {
			case "hard":
				m.bounceCode, m.bounceDiag = "550 5.1.1", "user unknown"
				if strings.HasSuffix(m.email, "outlook.com") {
					m.bounceDiag = "Requested action not taken: mailbox unavailable"
				}
			case "soft":
				m.bounceCode, m.bounceDiag = "452 4.2.2", "mailbox full"
			}
			if err := t.add(m); err != nil {
				return err
			}
		}
	}
	return t.weekExtras(ctx)
}

// weekExtras: the rejected calls of the week (8 suppressed and 1 opted
// out besides today's) and the staging key's test messages.
func (t *txnSeed) weekExtras(context.Context) error {
	s := t.s
	// Nine suppressed addresses besides Jordan's: one in the last 24 hours
	// (the Messages tab's third rejection), eight over the days before.
	keys := []string{"magic-link", "password-reset", "password-reset", "password-reset", "magic-link", "magic-link", "magic-link", "welcome", "welcome"}
	for i, key := range keys {
		e := txnSuppressed[i]
		at := s.Ago(3*time.Hour + 47*time.Minute)
		if i > 0 {
			at = s.Ago(26*time.Hour + time.Duration(i-1)*17*time.Hour)
		}
		reason, code, detail := txnSuppressedReason(i)
		m := &txnMsg{id: ID(), plan: t.plan(key), key: key, email: e, mailbox: "other", status: "rejected", accepted: at, keyName: "Acme Cloud · production", ip: "198.51.100.24",
			reject: "suppressed", rejectDetail: map[string]any{"reason": reason, "since": txnSuppressedAt(s, i), "source": "Postmark", "code": code, "detail": detail},
			source: "api", fromEmail: "no-reply@notify.acme.io", tags: t.plan(key).tags, metadata: map[string]string{"user_id": fmt.Sprintf("u_%05d", 20000+i)}}
		m.data = t.dataFor(key, "", at)
		m.subject = subjectFor(key, m.data)
		if err := t.add(m); err != nil {
			return err
		}
	}
	// Monday's summary to m.ortiz, who had turned Product updates off.
	mo := &txnMsg{id: ID(), plan: t.plan("usage-summary"), key: "usage-summary", email: "m.ortiz@example.com", mailbox: "other", status: "rejected",
		accepted: s.designDay(3, 8, 7), keyName: "Acme Cloud · production", ip: "198.51.100.24", reject: "opted_out",
		rejectDetail: map[string]any{"category": "Product updates", "since": s.designDay(8, 8, 41), "source": "unsubscribe_link"}, source: "api",
		fromEmail: "no-reply@notify.acme.io", tags: []string{"weekly"}, metadata: map[string]string{"user_id": "u_31877"}}
	mo.data = t.dataFor("usage-summary", "Mateo", mo.accepted)
	mo.subject = subjectFor("usage-summary", mo.data)
	if err := t.add(mo); err != nil {
		return err
	}
	// The staging key's tests: 24 today besides the two sample rows, and
	// a few a day before.
	for i := 0; i < 24+90; i++ {
		key := []string{"welcome", "password-reset", "magic-link", "invoice-paid"}[i%4]
		at := s.Ago(16*time.Minute + time.Duration(t.rng.Int64N(int64(23*time.Hour+14*time.Minute))))
		if i >= 24 {
			at = s.Ago(25*time.Hour + time.Duration(t.rng.Int64N(int64(5*24*time.Hour))))
		}
		allowed := i%5 != 0
		email := fmt.Sprintf("qa+%s%d@acme.io", strings.Split(key, "-")[0], i)
		if !allowed {
			email = fmt.Sprintf("tester%d@example.com", i)
		}
		m := &txnMsg{id: ID(), plan: t.plan(key), key: key, email: email, mailbox: "other", accepted: at, keyName: "Acme Cloud · staging", ip: "10.0.8.15",
			test: true, testDeliver: allowed, source: "api", fromEmail: t.plan(key).from, tags: append([]string{"staging"}, t.plan(key).tags...),
			metadata: map[string]string{"build": fmt.Sprintf("staging-%d", 400+i%20)}}
		if allowed {
			m.status, m.deliverAfter = "delivered", time.Duration(900+t.rng.IntN(1500))*time.Millisecond
		} else {
			m.status, m.reject = "cancelled", "test_not_allowed"
			m.rejectDetail = map[string]any{"allowed": []string{"qa+*@acme.io"}}
			m.errorText = "Test mode: not a member or an allowed test address, so not delivered."
		}
		m.data = t.dataFor(key, "QA", at)
		m.subject = subjectFor(key, m.data)
		if err := t.add(m); err != nil {
			return err
		}
	}
	return nil
}

// add turns a message into its rows: messages, transactional_messages,
// delivery events (and content when it's kept).
func (t *txnSeed) add(m *txnMsg) error {
	ws := t.ws
	var provider *uuid.UUID
	if m.provider != nil {
		provider = m.provider
	} else if m.reject == "" && m.status != "cancelled" {
		pm := t.postmark
		provider = &pm
	}
	if m.test && m.testDeliver {
		pm := t.postmark
		provider = &pm
	}
	if provider != nil && m.pmid == "" {
		m.pmid = uuid.NewString()
	}
	sent := m.accepted.Add(time.Duration(230+t.rng.IntN(200)) * time.Millisecond)
	var sentAt, deliveredAt *time.Time
	if provider != nil && m.status != "failed" {
		sentAt = &sent
	}
	if m.status == "delivered" || m.status == "complained" {
		d := m.accepted.Add(m.deliverAfter)
		deliveredAt = &d
	}
	routeNote := ""
	if m.failover {
		routeNote = "failover"
	}
	var pmid *string
	if provider != nil && sentAt != nil {
		pmid = &m.pmid
	}
	t.messages = append(t.messages, []any{m.id, ws.ID, "transactional", m.contact, m.email, provider, pmid, m.mailbox, m.status, m.accepted, sentAt, deliveredAt, m.opened,
		m.errorText, kindOf(m.source), routeNote, m.failover})
	// The log row.
	var tplID *uuid.UUID
	var version *int32
	tplKey, tplName := "", ""
	var catID *uuid.UUID
	catName, catKind := "", ""
	var sensitive, dataKeys []string
	if m.key != "" {
		id := t.templates[m.key]
		tplID = &id
		v := int32(t.live[m.key])
		version = &v
		tplKey, tplName = m.key, m.plan.name
		c := t.categories[m.plan.category]
		catID, catName = &c, m.plan.category
		for _, cc := range txnCategories {
			if cc.name == m.plan.category {
				catKind = cc.kind
			}
		}
		for _, v := range t.content[m.key].Variables {
			if v.Sensitive {
				sensitive = append(sensitive, v.Name)
			}
		}
		for k := range m.data {
			dataKeys = append(dataKeys, k)
		}
	} else if m.rawCategory != "" {
		c := t.categories[m.rawCategory]
		catID, catName = &c, m.rawCategory
		for _, cc := range txnCategories {
			if cc.name == m.rawCategory {
				catKind = cc.kind
			}
		}
	}
	if sensitive == nil {
		sensitive = []string{}
	}
	if dataKeys == nil {
		dataKeys = []string{}
	}
	keyID := t.keys[m.keyName]
	var keyRef *uuid.UUID
	if keyID != uuid.Nil {
		keyRef = &keyID
	}
	var ip *netip.Addr
	if m.ip != "" {
		ip = addr(m.ip)
	}
	if m.idem == "" && m.source == "api" {
		m.idem = strings.Split(m.key, "-")[0] + "-" + t.hex(6)
	}
	fromName := "Acme Cloud"
	replyTo := "Acme Cloud <support@acme.io>"
	if m.fromEmail == "billing@notify.acme.io" {
		fromName, replyTo = "Acme Billing", "Acme Billing <billing@acme.io>"
	}
	route := map[string]any{"provider": t.postmark, "provider_name": "Postmark", "adapter": "postmark", "primary": t.postmark, "primary_name": "Postmark",
		"fallback": t.ses, "fallback_name": "Amazon SES", "policy": "automatic", "rule": "Default"}
	if m.routeFailedAtStart {
		route["provider"], route["provider_name"], route["adapter"], route["failed_over"] = t.ses, "Amazon SES", "ses", true
	}
	attempts := m.attempts
	if attempts == nil && sentAt != nil {
		pname, pid := "Postmark", t.postmark
		if provider != nil && *provider == t.ses {
			pname, pid = "Amazon SES", t.ses
		}
		attempts = []map[string]any{{"provider_id": pid, "provider": pname, "at": sent, "ok": true, "provider_message_id": m.pmid}}
	}
	if attempts == nil {
		attempts = []map[string]any{}
	}
	files := m.attachments
	if files == nil {
		files = []map[string]any{}
	}
	size := 18421 + t.rng.IntN(900)
	if m.key == "" {
		size = 9200
	}
	for _, f := range files {
		if n, ok := f["size"].(int); ok {
			size += (n + 2) / 3 * 4
		}
	}
	rendered := m.accepted.Add(44 * time.Millisecond)
	expires := m.accepted.Add(30 * 24 * time.Hour)
	var expiresAt *time.Time
	if m.reject != "suppressed" && m.reject != "opted_out" && m.contentRemoved == nil {
		expiresAt = &expires
	}
	var providerSince *time.Time
	if provider != nil {
		ps := m.accepted
		if m.failedOverAt != nil {
			ps = *m.failedOverAt
		}
		providerSince = &ps
	}
	var failedFrom *uuid.UUID
	if m.failedOverAt != nil {
		pm := t.postmark
		failedFrom = &pm
	}
	detail := m.rejectDetail
	if detail == nil {
		detail = map[string]any{}
	}
	t.txnRows = append(t.txnRows, []any{m.id, ws.ID, m.source, tplID, tplKey, tplName, version, catID, catName, catKind, keyRef, m.keyName, ip, m.idem,
		m.subject, m.fromEmail, fromName, replyTo, m.name, m.tags, mustJSON(m.metadata), mustJSON(m.data), dataKeys, sensitive, mustJSON(files), int32(size),
		m.test, m.test && m.testDeliver, m.reject, mustJSON(detail), m.plan != nil && m.plan.opens, false, mustJSON(route), mustJSON(attempts), failedFrom, m.failedOverAt,
		providerSince, m.accepted, &rendered, expiresAt, m.contentRemoved})
	// Provider events.
	if provider == nil || sentAt == nil {
		return nil
	}
	ev := func(typ string, at time.Time, origType, code, diag, class string) {
		t.events = append(t.events, []any{ID(), ws.ID, m.id, *provider, typ, at, m.mailbox, origType, code, diag, class, m.email, m.pmid, ingest.SourceLabel("", "", nil, kindOf(m.source)), at.Add(400 * time.Millisecond),
			"demo-txn-" + m.id.String() + "-" + typ + "-" + fmt.Sprint(len(t.events))})
	}
	for _, d := range m.deferrals {
		code, diag := "451 4.7.1", "greylisted, try again later"
		if m.status == "deferred" {
			code, diag = "421 4.7.0", "Try again later, closing connection"
		}
		ev("deferred", m.accepted.Add(d), "Transient", code, diag, "temporary")
	}
	switch {
	case deliveredAt != nil:
		detail, code := m.delivDetail, m.delivCode
		if detail == "" {
			detail, code = "Accepted by "+mxFor(m.mailbox), "250 2.0.0"
		}
		ev("delivered", *deliveredAt, "Delivery", code, detail, "")
		if m.complaint {
			ev("complaint", deliveredAt.Add(47*time.Minute), "SpamComplaint", "abuse", "Feedback loop report", "")
		}
	case m.status == "bounced":
		typ, class, orig := "hard_bounce", "mailbox_unknown", "HardBounce"
		if m.bounce == "soft" {
			typ, class, orig = "soft_bounce", "mailbox_full", "SoftBounce"
		}
		ev(typ, m.accepted.Add(time.Duration(2+t.rng.IntN(9))*time.Second), orig, m.bounceCode, m.bounceDiag, class)
	}
	return nil
}

func kindOf(source string) string {
	if source == "smtp" {
		return "smtp"
	}
	return "api"
}

func mxFor(mailbox string) string {
	switch mailbox {
	case "gmail":
		return "gmail-smtp-in.l.google.com"
	case "outlook":
		return "outlook-com.olc.protection.outlook.com"
	case "yahoo":
		return "mta5.am0.yahoodns.net"
	case "icloud":
		return "mx01.mail.icloud.com"
	}
	return "mx.example.net"
}

// samples are the log's latest rows (the TransactionalMessages board),
// newest first.
func (t *txnSeed) samples(ctx context.Context) error {
	s := t.s
	t0 := t.t0
	var sam uuid.UUID
	if err := s.Tx.QueryRow(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND email = 'sam.rivera@outlook.com'`, t.ws.ID).Scan(&sam); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	var samRef *uuid.UUID
	if sam != uuid.Nil {
		samRef = &sam
	}
	pm := t.postmark
	samID := uuid.MustParse("0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44")
	samData := map[string]any{"first_name": "Sam", "reset_url": transactional.Mask, "expires_in": "30 minutes"}
	samMsg := &txnMsg{id: samID, plan: t.plan("password-reset"), key: "password-reset", email: "sam.rivera@outlook.com", name: "Sam Rivera", contact: samRef, mailbox: "outlook",
		status: "delivered", accepted: t0, deliverAfter: 1310 * time.Millisecond, provider: &pm, pmid: "7e1f3c2a-91d4-4b70-a6f2-0c8e55d1b9a3", data: samData,
		subject: "Reset your Acme Cloud password", keyName: "Acme Cloud · production", ip: "198.51.100.24", idem: "pwreset-7f3a91", tags: []string{"auth"},
		metadata: map[string]string{"user_id": "u_48213"}, source: "api", fromEmail: "no-reply@notify.acme.io", keepContent: true,
		delivDetail: "Accepted by outlook-com.olc.protection.outlook.com · Queued mail for delivery", delivCode: "250 2.6.0",
		attempts: []map[string]any{{"provider_id": t.postmark, "provider": "Postmark", "at": t0.Add(282 * time.Millisecond), "ok": true, "provider_message_id": "7e1f3c2a-91d4-4b70-a6f2-0c8e55d1b9a3"}}}
	rows := []*txnMsg{samMsg}
	jordanAt := t0.Add(-23 * time.Second)
	rows = append(rows, &txnMsg{id: ID(), plan: t.plan("magic-link"), key: "magic-link", email: "jordan@example.org", mailbox: "other", status: "rejected", accepted: jordanAt,
		data: map[string]any{"sign_in_url": transactional.Mask, "expires_in": "15 minutes"}, subject: "Your sign-in link for Acme Cloud", keyName: "Acme Cloud · production",
		ip: "198.51.100.24", tags: []string{"auth"}, metadata: map[string]string{"user_id": "u_40218"}, source: "api", fromEmail: "no-reply@notify.acme.io", reject: "suppressed",
		rejectDetail: map[string]any{"reason": "hard_bounce", "since": s.designDay(13, 8, 14), "source": "Postmark", "code": "550 5.1.1", "detail": "mailbox does not exist"}})
	invoice := map[string]any{"invoice_number": "INV-2026-0418", "amount": "$49.00", "paid_on": t0.Format("2006-01-02"), "plan_name": "Pro", "invoice_url": "https://app.acme.io/billing/INV-2026-0418"}
	rows = append(rows, &txnMsg{id: ID(), plan: t.plan("invoice-paid"), key: "invoice-paid", email: "billing@northwind.example", mailbox: "other", status: "delivered",
		accepted: t0.Add(-58 * time.Second), deliverAfter: 2410 * time.Millisecond, data: invoice, subject: "Invoice INV-2026-0418 paid · $49.00", keyName: "Acme Cloud · production",
		ip: "198.51.100.24", tags: []string{"billing"}, metadata: map[string]string{"user_id": "u_13377", "invoice": "INV-2026-0418"}, source: "api", fromEmail: "billing@notify.acme.io",
		attachments: []map[string]any{{"filename": "INV-2026-0418.pdf", "content_type": "application/pdf", "size": 48211, "content_id": ""}}, keepContent: true})
	priyaAt := t0.Add(-131 * time.Second)
	rows = append(rows, &txnMsg{id: ID(), plan: t.plan("welcome"), key: "welcome", email: "priya.shah@example.net", name: "Priya Shah", mailbox: "other", status: "delivered",
		accepted: priyaAt, deliverAfter: 3*time.Minute + 5*time.Second, deferrals: []time.Duration{3 * time.Second}, data: map[string]any{"first_name": "Priya", "dashboard_url": "https://app.acme.io/"},
		subject: "Welcome to Acme Cloud, Priya", keyName: "Acme Cloud · production", ip: "198.51.100.24", tags: []string{"onboarding"}, metadata: map[string]string{"user_id": "u_51120"},
		source: "api", fromEmail: "no-reply@notify.acme.io", keepContent: true, delivDetail: "Accepted by mx.example.net on the second try"})
	rows = append(rows, &txnMsg{id: ID(), plan: t.plan("trial-ending"), key: "trial-ending", email: "ops@contoso.example", mailbox: "other", status: "bounced", bounce: "soft",
		bounceCode: "452 4.2.2", bounceDiag: "mailbox full", accepted: t0.Add(-167 * time.Second), data: map[string]any{"first_name": "Ops", "days_left": 3, "upgrade_url": "https://app.acme.io/billing/plans"},
		subject: "Your Acme Cloud trial ends in 3 days", keyName: "Acme Cloud · production", ip: "198.51.100.24", tags: []string{"trial"}, metadata: map[string]string{"user_id": "u_27731"},
		source: "api", fromEmail: "no-reply@notify.acme.io", errorText: "452 4.2.2 mailbox full", keepContent: true})
	rows = append(rows, &txnMsg{id: ID(), plan: t.plan("usage-summary"), key: "usage-summary", email: "kim.tran@example.com", name: "Kim Tran", mailbox: "other", status: "rejected",
		accepted: t0.Add(-256 * time.Second), data: map[string]any{"first_name": "Kim", "projects": 2, "deploys": 14, "usage_url": "https://app.acme.io/usage"},
		subject: "Your week in Acme Cloud", keyName: "Acme Cloud · production", ip: "198.51.100.24", tags: []string{"weekly"}, metadata: map[string]string{"user_id": "u_38210"},
		source: "api", fromEmail: "no-reply@notify.acme.io", reject: "opted_out",
		rejectDetail: map[string]any{"category": "Product updates", "since": s.designDay(17, 19, 20), "source": "preference_center"}})
	rows = append(rows, &txnMsg{id: ID(), plan: t.plan("welcome"), key: "welcome", email: "qa+signup@acme.io", name: "QA", mailbox: "other", status: "delivered",
		accepted: t0.Add(-274 * time.Second), deliverAfter: 1100 * time.Millisecond, data: map[string]any{"first_name": "QA", "dashboard_url": "https://staging.acme.io/"},
		subject: "Welcome to Acme Cloud, QA", keyName: "Acme Cloud · staging", ip: "10.0.8.15", tags: []string{"staging", "onboarding"}, metadata: map[string]string{"build": "staging-412"},
		source: "api", fromEmail: "no-reply@notify.acme.io", test: true, testDeliver: true})
	rows = append(rows, &txnMsg{id: ID(), plan: t.plan("welcome"), key: "welcome", email: "lena@example.com", name: "Lena", mailbox: "other", status: "cancelled",
		accepted: t0.Add(-278 * time.Second), data: map[string]any{"first_name": "Lena", "dashboard_url": "https://staging.acme.io/"}, subject: "Welcome to Acme Cloud, Lena",
		keyName: "Acme Cloud · staging", ip: "10.0.8.15", tags: []string{"staging", "onboarding"}, metadata: map[string]string{"build": "staging-412"}, source: "api",
		fromEmail: "no-reply@notify.acme.io", test: true, reject: "test_not_allowed", rejectDetail: map[string]any{"allowed": []string{"qa+*@acme.io"}},
		errorText: "Test mode: not a member or an allowed test address, so not delivered."})
	rows = append(rows, &txnMsg{id: ID(), email: "finance@northwind.example", mailbox: "other", status: "delivered", accepted: t0.Add(-366 * time.Second), deliverAfter: 1820 * time.Millisecond,
		data: map[string]any{}, subject: "Your receipt #4821", keyName: "Billing service (SMTP)", ip: "198.51.100.40", tags: []string{"receipt"}, metadata: map[string]string{},
		source: "smtp", fromEmail: "billing@notify.acme.io", keepContent: true, rawCategory: "Billing"})
	alexAt := t0.Add(-734 * time.Second)
	opened := alexAt.Add(3*time.Minute + 2*time.Second)
	rows = append(rows, &txnMsg{id: ID(), plan: t.plan("usage-summary"), key: "usage-summary", email: "alex.morgan@example.com", name: "Alex Morgan", mailbox: "other", status: "delivered",
		accepted: alexAt, deliverAfter: 1520 * time.Millisecond, opened: &opened,
		data:    map[string]any{"first_name": "Alex", "projects": 4, "deploys": 37, "usage_url": "https://app.acme.io/usage"},
		subject: "Your week in Acme Cloud", keyName: "Acme Cloud · production", ip: "198.51.100.24", tags: []string{"weekly"}, metadata: map[string]string{"user_id": "u_29034"},
		source: "api", fromEmail: "no-reply@notify.acme.io", keepContent: true})
	// Lucía Fernández's reset in Spanish, her contact's language (the
	// languages brief: "At 14:02 it went to lucia.fernandez@example.com:
	// Spanish, from the contact"), 18 minutes ago on that page's clock.
	lucia := &txnMsg{id: ID(), plan: t.plan("password-reset"), key: "password-reset", email: "lucia.fernandez@example.com", name: "Lucía Fernández", mailbox: "other",
		status: "delivered", accepted: s.Ago(18 * time.Minute), deliverAfter: 1220 * time.Millisecond, provider: &pm,
		data: map[string]any{"first_name": "Lucía", "reset_url": transactional.Mask, "expires_in": "30 minutes"}, subject: "Restablece tu contraseña de Acme Cloud",
		keyName: "Acme Cloud · production", ip: "198.51.100.24", tags: []string{"auth"}, metadata: map[string]string{"user_id": "u_51337"}, source: "api",
		fromEmail: "no-reply@notify.acme.io"}
	rows = append(rows, lucia)
	// X05's seeder names the contact and translates the template after
	// this one: link and label the message once every seeder has run.
	s.After(func(ctx context.Context, s *Seeder) error {
		_, err := s.Tx.Exec(ctx, `UPDATE messages SET contact_id = (SELECT id FROM contacts WHERE workspace_id = $1 AND email = $3 AND erased_at IS NULL LIMIT 1),
			language = 'es', contact_language = 'es', language_reason = 'match' WHERE id = $2`, t.ws.ID, lucia.id, lucia.email)
		return err
	})
	for _, m := range rows {
		if err := t.add(m); err != nil {
			return err
		}
		if m.keepContent {
			if err := t.keepContent(m); err != nil {
				return err
			}
		}
	}
	return nil
}

// keepContent stores a message's rendered content (sensitive values masked).
func (t *txnSeed) keepContent(m *txnMsg) error {
	to := m.email
	var subject, html, text string
	if m.key == "" {
		subject = m.subject
		html = `<!doctype html><html><body style="font-family:Arial,sans-serif;color:#111827"><h1 style="font-size:20px">Thanks for your payment</h1>` +
			`<p>We received $49.00 for invoice #4821.</p><p>Acme Billing · billing@acme.io</p></body></html>`
		text = "Thanks for your payment\n\nWe received $49.00 for invoice #4821.\n\nAcme Billing · billing@acme.io"
	} else {
		var err error
		subject, html, text, err = transactional.DemoRender(t.content[m.key], m.data, to, m.name, map[string]string{"unsubscribe_url": "#unsubscribe", "preferences_url": "#preferences"})
		if err != nil {
			return err
		}
	}
	from := "Acme Cloud <no-reply@notify.acme.io>"
	if m.fromEmail == "billing@notify.acme.io" {
		from = "Acme Billing <billing@notify.acme.io>"
	}
	hdr := map[string]string{"From": from, "To": to, "Subject": subject, "Date": m.accepted.UTC().Format(time.RFC1123Z), "Message-ID": "<" + m.id.String() + "@notify.acme.io>",
		"Feedback-ID": strings.TrimSuffix(m.key, "") + ":acme-newsletter:transactional:reputecast"}
	if m.key == "" {
		hdr["Feedback-ID"] = "raw:acme-newsletter:transactional:reputecast"
	}
	t.contents = append(t.contents, []any{m.id, html, text, mustJSON(hdr), m.accepted.Add(30 * 24 * time.Hour)})
	return nil
}

// history: Jordan's hard bounce on 2 Oct, Sam's reset on 12 Sep whose
// content was removed after 30 days, and the 9 Oct Postmark outage: 37
// messages through Amazon SES, and one invoice that every provider refused.
func (t *txnSeed) history(ctx context.Context) error {
	s := t.s
	pm, ses := t.postmark, t.ses
	var sam uuid.UUID
	_ = s.Tx.QueryRow(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND email = 'sam.rivera@outlook.com'`, t.ws.ID).Scan(&sam)
	var samRef *uuid.UUID
	if sam != uuid.Nil {
		samRef = &sam
	}
	bounceAt := s.designDay(13, 8, 14)
	if err := t.add(&txnMsg{id: ID(), plan: t.plan("magic-link"), key: "magic-link", email: "jordan@example.org", mailbox: "other", status: "bounced", bounce: "hard",
		bounceCode: "550 5.1.1", bounceDiag: "mailbox does not exist", accepted: bounceAt.Add(-6 * time.Second), provider: &pm,
		data: map[string]any{"sign_in_url": transactional.Mask, "expires_in": "15 minutes"}, subject: "Your sign-in link for Acme Cloud", keyName: "Acme Cloud · production",
		ip: "198.51.100.24", tags: []string{"auth"}, metadata: map[string]string{"user_id": "u_40218"}, source: "api", fromEmail: "no-reply@notify.acme.io",
		errorText: "550 5.1.1 mailbox does not exist"}); err != nil {
		return err
	}
	removed := s.designDay(3, 8, 1)
	if err := t.add(&txnMsg{id: ID(), plan: t.plan("password-reset"), key: "password-reset", email: "sam.rivera@outlook.com", name: "Sam Rivera", contact: samRef, mailbox: "outlook",
		status: "delivered", accepted: s.designDay(33, 8, 1), deliverAfter: 1280 * time.Millisecond, provider: &pm,
		data: map[string]any{"first_name": "Sam", "reset_url": transactional.Mask, "expires_in": "30 minutes"}, subject: "Reset your Acme Cloud password",
		keyName: "Acme Cloud · production", ip: "198.51.100.24", tags: []string{"auth"}, metadata: map[string]string{"user_id": "u_48213"}, source: "api",
		fromEmail: "no-reply@notify.acme.io", contentRemoved: &removed}); err != nil {
		return err
	}
	// 9 Oct, 14:02 to 14:06: Postmark answered 503.
	start := s.designDay(6, 14, 2)
	outage := []string{}
	for i := 0; i < 13; i++ {
		outage = append(outage, "password-reset")
	}
	for i := 0; i < 16; i++ {
		outage = append(outage, "magic-link")
	}
	for i := 0; i < 7; i++ {
		outage = append(outage, "welcome")
	}
	for i, key := range outage {
		at := start.Add(time.Duration(5+i*6) * time.Second)
		down := at.Add(150 * time.Millisecond)
		mbp, dom := t.mailbox()
		m := &txnMsg{id: ID(), plan: t.plan(key), key: key, email: fmt.Sprintf("user.%05d@%s", 30000+i, dom), mailbox: mbp, status: "delivered", accepted: at,
			deliverAfter: time.Duration(2700+t.rng.IntN(800)) * time.Millisecond, provider: &ses, failover: true, failedOverAt: &down, keyName: "Acme Cloud · production",
			ip: "198.51.100.24", tags: t.plan(key).tags, metadata: map[string]string{"user_id": fmt.Sprintf("u_%05d", 60000+i)}, source: "api", fromEmail: "no-reply@notify.acme.io",
			attempts: []map[string]any{
				{"provider_id": pm, "provider": "Postmark", "at": down, "ok": false, "error": "503 Service Unavailable", "gave_up": true, "down": true},
				{"provider_id": ses, "provider": "Amazon SES", "at": down.Add(120 * time.Millisecond), "ok": true},
			}}
		m.data = t.dataFor(key, "", at)
		m.subject = subjectFor(key, m.data)
		if err := t.add(m); err != nil {
			return err
		}
	}
	// Sam's reset during the outage: 14:03:02, delivered through SES in 3.1 s.
	samAt := s.designDay(6, 14, 3).Add(2 * time.Second)
	samDown := samAt.Add(90 * time.Millisecond)
	if err := t.add(&txnMsg{id: ID(), plan: t.plan("password-reset"), key: "password-reset", email: "sam.rivera@outlook.com", name: "Sam Rivera", contact: samRef, mailbox: "outlook",
		status: "delivered", accepted: samAt, deliverAfter: 3100 * time.Millisecond, provider: &ses, failover: true, failedOverAt: &samDown,
		data: map[string]any{"first_name": "Sam", "reset_url": transactional.Mask, "expires_in": "30 minutes"}, subject: "Reset your Acme Cloud password",
		keyName: "Acme Cloud · production", ip: "198.51.100.24", tags: []string{"auth"}, metadata: map[string]string{"user_id": "u_48213"}, source: "api",
		fromEmail: "no-reply@notify.acme.io", delivDetail: "Accepted by outlook-com.olc.protection.outlook.com", delivCode: "250 2.6.0",
		attempts: []map[string]any{
			{"provider_id": pm, "provider": "Postmark", "at": samDown, "ok": false, "error": "503 Service Unavailable", "gave_up": true, "down": true},
			{"provider_id": ses, "provider": "Amazon SES", "at": samDown.Add(80 * time.Millisecond), "ok": true},
		}}); err != nil {
		return err
	}
	// The invoice to billing@initech: Postmark 503 three times, then SES
	// refused the address (its account-level suppression list).
	inv := s.designDay(6, 14, 2).Add(10 * time.Second)
	t1, t2, t3 := inv.Add(200*time.Millisecond), inv.Add(90*time.Second), inv.Add(170*time.Second)
	t4 := inv.Add(185 * time.Second)
	failed := &txnMsg{id: ID(), plan: t.plan("invoice-paid"), key: "invoice-paid", email: "billing@initech.example", mailbox: "other", status: "failed", accepted: inv, provider: &ses,
		keyName: "Acme Cloud · production", ip: "198.51.100.24", tags: []string{"billing"}, metadata: map[string]string{"user_id": "u_17002", "invoice": "INV-2026-0402"},
		source: "api", fromEmail: "billing@notify.acme.io", failedOverAt: &t4,
		errorText:   "Every provider in the route failed: MessageRejected: address is on the account-level suppression list",
		attachments: []map[string]any{{"filename": "INV-2026-0402.pdf", "content_type": "application/pdf", "size": 47012, "content_id": ""}},
		attempts: []map[string]any{
			{"provider_id": pm, "provider": "Postmark", "at": t1, "ok": false, "error": "503 Service Unavailable", "temporary": true},
			{"provider_id": pm, "provider": "Postmark", "at": t2, "ok": false, "error": "503 Service Unavailable", "temporary": true},
			{"provider_id": pm, "provider": "Postmark", "at": t3, "ok": false, "error": "503 Service Unavailable", "gave_up": true, "down": true},
			{"provider_id": ses, "provider": "Amazon SES", "at": t4, "ok": false, "error": "MessageRejected: address is on the account-level suppression list", "gave_up": true},
		}}
	failed.data = map[string]any{"invoice_number": "INV-2026-0402", "amount": "$199.00", "paid_on": inv.Format("2006-01-02"), "plan_name": "Team", "invoice_url": "https://app.acme.io/billing/INV-2026-0402"}
	failed.subject = "Invoice INV-2026-0402 paid · $199.00"
	t.s.Extra["txn:failed"] = failed.id
	return t.add(failed)
}

// flush writes the rows built so far.
func (t *txnSeed) flush(ctx context.Context) error {
	s := t.s
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"messages"}, []string{"id", "workspace_id", "stream", "contact_id", "email", "provider_id", "provider_message_id",
		"mailbox_provider", "status", "queued_at", "sent_at", "delivered_at", "opened_at", "error", "kind", "route_note", "failover"}, pgx.CopyFromRows(t.messages)); err != nil {
		return fmt.Errorf("messages: %w", err)
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"transactional_messages"}, []string{"message_id", "workspace_id", "source", "template_id", "template_key", "template_name",
		"template_version", "category_id", "category_name", "category_kind", "key_id", "key_name", "ip", "idempotency_key", "subject", "from_email", "from_name", "reply_to",
		"to_name", "tags", "metadata", "data", "data_keys", "sensitive", "attachments", "size_bytes", "test", "test_delivered", "reject_reason", "reject_detail",
		"track_opens", "track_clicks", "route", "attempts", "failed_over_from", "failed_over_at", "provider_since", "accepted_at", "rendered_at", "content_expires_at",
		"content_removed_at"}, pgx.CopyFromRows(t.txnRows)); err != nil {
		return fmt.Errorf("transactional_messages: %w", err)
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"delivery_events"}, []string{"id", "workspace_id", "message_id", "provider_id", "type", "occurred_at", "mailbox_provider",
		"original_type", "original_code", "diagnostic", "bounce_class", "recipient", "provider_message_id", "source", "received_at", "dedupe_key"}, pgx.CopyFromRows(t.events)); err != nil {
		return fmt.Errorf("delivery_events: %w", err)
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"transactional_message_content"}, []string{"message_id", "html", "text", "headers", "expires_at"}, pgx.CopyFromRows(t.contents)); err != nil {
		return fmt.Errorf("content: %w", err)
	}
	// Alex opened the summary at 10:33: the open event.
	_, err := s.Tx.Exec(ctx, `INSERT INTO delivery_events (id, workspace_id, message_id, provider_id, type, occurred_at, mailbox_provider, original_type, recipient, provider_message_id, source, received_at, dedupe_key)
		SELECT gen_random_uuid(), m.workspace_id, m.id, m.provider_id, 'open', m.opened_at, m.mailbox_provider, 'Open', m.email, m.provider_message_id, 'tracking', m.opened_at, 'demo-txn-open-' || m.id
		FROM messages m WHERE m.workspace_id = $1 AND m.stream = 'transactional' AND m.email = 'alex.morgan@example.com' AND m.opened_at IS NOT NULL`, t.ws.ID)
	return err
}

// incidents: the 9 Oct failover and the 3 Oct bounce alert, with the
// alert's notification.
func (t *txnSeed) incidents(ctx context.Context) error {
	s := t.s
	pm := t.postmark
	if _, err := s.Tx.Exec(ctx, `INSERT INTO transactional_alerts (id, workspace_id, kind, provider_id, value, threshold, detail, started_at, ended_at)
		VALUES ($1, $2, 'failover', $3, 37, 0, $4, $5, $6)`, ID(), t.ws.ID, pm, mustJSON(map[string]any{"messages": 37, "lost": 0, "fallback": "Amazon SES", "median_ms": 3100}),
		s.designDay(6, 14, 2), s.designDay(6, 14, 6)); err != nil {
		return err
	}
	alertAt := s.designDay(12, 9, 15)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO transactional_alerts (id, workspace_id, kind, template_id, value, threshold, detail, started_at, ended_at, seen_at, seen_by)
		VALUES ($1, $2, 'bounce_rate', $3, 2.4, 2, $4, $5, $6, $7, $8)`, ID(), t.ws.ID, t.templates["magic-link"],
		mustJSON(map[string]any{"sent": 620, "bounced": 15, "hard": 15, "domains": []map[string]any{{"domain": "gmial.com", "count": 9}, {"domain": "yaho.com", "count": 4}, {"domain": "hotmial.com", "count": 2}}}),
		alertAt, alertAt.Add(time.Hour), alertAt.Add(25*time.Minute), t.user("marcus@acme.io")); err != nil {
		return err
	}
	body := "Magic sign-in link: bounce rate 2.4% in one hour (limit 2%). Most bounces went to gmial.com and yaho.com."
	rows, err := s.Tx.Query(ctx, `SELECT m.user_id, r.key, r.builtin, r.permissions FROM workspace_members m JOIN roles r ON r.id = m.role_id WHERE m.workspace_id = $1 AND m.status = 'active'`, t.ws.ID)
	if err != nil {
		return err
	}
	var users []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		var key string
		var builtin bool
		var perms []string
		if err := rows.Scan(&id, &key, &builtin, &perms); err != nil {
			rows.Close()
			return err
		}
		if authz.Effective(key, builtin, perms).Has(authz.TransactionalView) {
			users = append(users, id)
		}
	}
	rows.Close()
	wsID := t.ws.ID
	for _, u := range users {
		read := alertAt.Add(40 * time.Minute)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO notifications (id, user_id, workspace_id, kind, severity, title, summary, body, link, created_at, read_at, email_status, emailed_at)
			VALUES ($1, $2, $3, 'transactional_alert', 'bad', 'Transactional delivery alert', '', $4, $5, $6, $7, 'sent', $6)`,
			ID(), u, &wsID, body, "/w/acme-newsletter/transactional/messages?template=magic-link&status=bounced&range=1h", alertAt, read); err != nil {
			return err
		}
	}
	return nil
}

// webhooks: Product backend also listens to message.bounced and
// message.failed; the failed invoice's event reached it on 9 Oct.
func (t *txnSeed) webhooks(ctx context.Context) error {
	s := t.s
	var hook uuid.UUID
	err := s.Tx.QueryRow(ctx, `UPDATE webhooks SET events = events || ARRAY['message.bounced', 'message.failed'] WHERE workspace_id = $1 AND name = 'Product backend' RETURNING id`, t.ws.ID).Scan(&hook)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	failedID, _ := s.Extra["txn:failed"].(uuid.UUID)
	at := s.designDay(6, 14, 5).Add(12 * time.Second)
	payload := mustJSON(map[string]any{"id": ID(), "type": "message.failed", "created_at": at.UTC(), "workspace": "acme-newsletter", "data": map[string]any{
		"message_id": failedID, "template": "invoice-paid", "template_version": t.live["invoice-paid"], "category": "Billing", "to": "billing@initech.example", "status": "failed",
		"reason": "MessageRejected: address is on the account-level suppression list", "provider": "Amazon SES", "tags": []string{"billing"},
		"metadata": map[string]string{"user_id": "u_17002", "invoice": "INV-2026-0402"}, "at": at.UTC()}})
	_, err = s.Tx.Exec(ctx, `INSERT INTO webhook_deliveries (id, webhook_id, event, event_id, payload, attempt, status_code, duration_ms, state, created_at, delivered_at, last_attempt_at)
		VALUES ($1, $2, 'message.failed', $3, $4, 1, 200, 88, 'delivered', $5, $5, $5)`, ID(), hook, ID(), payload, at)
	return err
}

// audit: the transactional history of the workspace (§3.7) and the
// relay's instance entry.
func (t *txnSeed) audit(ctx context.Context) error {
	s := t.s
	m := func(kv ...any) map[string]any {
		out := map[string]any{}
		for i := 0; i+1 < len(kv); i += 2 {
			out[kv[i].(string)] = kv[i+1]
		}
		return out
	}
	tpl := func(key string) string { return t.templates[key].String() }
	events := []auditSeed{
		{at: s.designDay(43, 15, 58), actorType: "user", actor: "marcus@acme.io", action: "transactional.settings_updated", targetType: "transactional_settings", targetID: t.ws.ID.String(),
			target: "Transactional settings", ip: "203.0.113.9", ua: uaChromeMac, meta: m("summary", "Default sender Acme Cloud <no-reply@notify.acme.io> · reply-to support@acme.io")},
		{at: s.designDay(43, 16, 5), actorType: "user", actor: "marcus@acme.io", action: "apikey.created", targetType: "api_key", targetID: t.keys["Acme Cloud · production"].String(),
			target: "Acme Cloud · production", ip: "203.0.113.9", ua: uaChromeMac, meta: m("summary", "transactional:send · all templates · expires Oct 15, 2027", "permission", "apikeys:manage")},
		{at: s.designDay(43, 16, 7), actorType: "user", actor: "marcus@acme.io", action: "apikey.created", targetType: "api_key", targetID: t.keys["Acme Cloud · staging"].String(),
			target: "Acme Cloud · staging", ip: "203.0.113.9", ua: uaChromeMac, meta: m("summary", "transactional:send · all templates · test mode", "permission", "apikeys:manage")},
		{at: s.designDay(30, 11, 20), actorType: "user", actor: "linh@acme.io", action: "apikey.created", targetType: "api_key", targetID: t.keys["Billing service (SMTP)"].String(),
			target: "Billing service (SMTP)", ip: "203.0.113.9", ua: uaChromeMac, meta: m("summary", "transactional:send · category Billing · SMTP credential", "permission", "apikeys:manage")},
		{at: s.Ago(9 * 24 * time.Hour), actorType: "user", actor: "linh@acme.io", action: "transactional.template_published", targetType: "transactional_template", targetID: tpl("password-reset"),
			target: "password-reset", ip: "203.0.113.9", ua: uaChromeMac, meta: m("summary", "v3 · Footer address and the 30 minute expiry", "version", 3)},
		{at: s.Ago(8 * 24 * time.Hour), actorType: "user", actor: "sofia@acme.io", action: "transactional.template_published", targetType: "transactional_template", targetID: tpl("invoice-paid"),
			target: "invoice-paid", ip: "203.0.113.14", ua: uaChromeMac, meta: m("summary", "v4 · The PDF goes along", "version", 4)},
		{at: s.Ago(25 * time.Minute), actorType: "user", actor: "daniel@acme.io", action: "transactional.template_saved", targetType: "transactional_template", targetID: tpl("trial-ending"),
			target: "trial-ending", ip: "203.0.113.9", ua: uaChromeMac, meta: m("summary", "Draft v2 · trial_end_date replaces days_left")},
		{at: s.Ago(3 * time.Hour), actorType: "user", actor: "linh@acme.io", action: "transactional.content_viewed", targetType: "transactional_message",
			targetID: "0192f1c4-7b2e-7c51-9d0a-3f5e8b1a2c44", target: "Password reset to sam.rivera@outlook.com", ip: "203.0.113.9", ua: uaChromeMac,
			meta: m("summary", "Viewed the content of Password reset to sam.rivera@outlook.com", "template", "password-reset")},
		{at: s.designDay(2, 11, 40), actorType: "user", actor: "marcus@acme.io", action: "transactional.log_exported", targetType: "transactional_log", targetID: t.ws.ID.String(),
			target: t.ws.Name, ip: "203.0.113.9", ua: uaChromeMac, meta: m("summary", "Exported 1,204 messages of the message log", "rows", 1204, "status", "bounced", "range", "7d")},
	}
	for _, e := range events {
		if err := s.insertAudit(ctx, t.ws.ID, e); err != nil {
			return err
		}
	}
	// The relay is an instance setting: its entry has no workspace.
	linh := t.user("linh@acme.io")
	_, err := s.Tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name,
		result, reason, ip, user_agent, request_id, before, after, metadata)
		VALUES ($1, NULL, $2, 'user', $3, 'Linh Nguyen', 'smtp_relay.enabled', 'smtp_relay', '', $4, 'allowed', '', $5, $6, $7, $8, $9, $10)`,
		ID(), s.designDay(30, 11, 12), linh, txnRelayHost, addr("203.0.113.9"), uaChromeMac, "req_"+ID().String()[24:32],
		mustJSON(map[string]any{"enabled": false}), mustJSON(map[string]any{"enabled": true, "hostname": txnRelayHost, "starttls_port": 587, "tls_port": 465, "cert_source": "acme"}),
		mustJSON(map[string]any{"summary": "Turned the SMTP relay on at " + txnRelayHost + " (ports 587 and 465)"}))
	return err
}
