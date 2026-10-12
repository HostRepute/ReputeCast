package demo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
)

func init() { seedIntegrations = seedIntegrationsImpl }

// demoKey builds a token whose masked form matches the prototype
// ("rc_live_8f2c…a91d"); the secret itself is random and never shown.
func demoKey(prefix, head, tail string) (string, []byte) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	body := hex.EncodeToString(b)
	body = head + body[len(head):len(body)-len(tail)] + tail
	key := prefix + body
	return key, authn.HashToken(key)
}

func pickedFirst(perms []authz.Permission) []string {
	out := []string{}
	for _, p := range perms {
		out = append(out, string(p))
	}
	for _, k := range authz.NewSet(perms...).Strings() {
		found := false
		for _, o := range out {
			if o == k {
				found = true
			}
		}
		if !found {
			out = append(out, k)
		}
	}
	return out
}

func addr(s string) *netip.Addr {
	a := netip.MustParseAddr(s)
	return &a
}

// seedIntegrationsImpl adds the API keys, MCP agents and outgoing webhooks
// of the prototype, with delivery logs and the audit stream.
func seedIntegrationsImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh, marcus := s.Users["linh@acme.io"], s.Users["marcus@acme.io"]
	digest := s.Lists["Weekly Digest"]
	type key struct {
		kind, name, desc, prefix, head, tail string
		perms                                []authz.Permission
		lists                                []uuid.UUID
		creator                              uuid.UUID
		created                              time.Time
		expires                              *time.Time
		lastUsed                             *time.Time
		ip, client, lastCall                 string
	}
	t := func(v time.Time) *time.Time { return &v }
	keys := []key{
		{kind: "api_key", name: "Website signup form", prefix: "rc_live_", head: "8f2c", tail: "a91d", perms: []authz.Permission{authz.SubscribersCreate},
			lists: []uuid.UUID{digest}, creator: linh.ID, created: s.Ago(5*time.Minute + 58*time.Second), lastUsed: t(s.Ago(3 * time.Minute)), ip: "198.51.100.12"},
		{kind: "api_key", name: "Data warehouse sync", prefix: "rc_live_", head: "3b17", tail: "c04e", perms: []authz.Permission{authz.ReportsView, authz.SubscribersView},
			creator: marcus.ID, created: s.designDay(20, 14, 10), expires: t(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)), lastUsed: t(s.designDay(1, 3, 5)),
			ip: "10.0.4.22", client: "nightly job"},
		{kind: "api_key", name: "Old CI key", prefix: "rc_live_", head: "11aa", tail: "ff02", perms: []authz.Permission{authz.CampaignsView},
			creator: linh.ID, created: s.designDay(223, 11, 0), expires: t(s.designDay(41, 0, 0)), lastUsed: t(s.designDay(43, 22, 14)), ip: "10.0.9.14", client: "GitHub Actions runner"},
		{kind: "mcp_agent", name: "Claude — analytics", desc: "Claude Desktop", prefix: "rc_agent_", head: "7a", tail: "c2", perms: []authz.Permission{authz.ReportsView, authz.DeliverabilityView},
			creator: linh.ID, created: s.designDay(12, 15, 30), expires: t(time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)), lastUsed: t(s.Ago(10*time.Minute + 12*time.Second)),
			ip: "203.0.113.9", client: "Claude Desktop", lastCall: "get_campaign_report"},
		{kind: "mcp_agent", name: "Ops agent", desc: "on-call runbook", prefix: "rc_agent_", head: "e0", tail: "44", perms: []authz.Permission{authz.CampaignsPause, authz.ProvidersView},
			creator: marcus.ID, created: s.designDay(10, 18, 0), lastUsed: t(s.Ago(25*time.Minute + 46*time.Second)), ip: "10.0.7.3", lastCall: "pause_campaign"},
	}
	for _, k := range keys {
		id := ID()
		_, hash := demoKey(k.prefix, k.head, k.tail)
		masked := k.prefix + k.head + "…" + k.tail
		scope := json.RawMessage(`{}`)
		if len(k.lists) > 0 {
			b, _ := json.Marshal(authz.Scope{Lists: k.lists})
			scope = b
		}
		creator := k.creator
		var ip *netip.Addr
		if k.ip != "" {
			ip = addr(k.ip)
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO api_keys (id, workspace_id, kind, name, description, prefix, token_hash, permissions, scope, created_by, created_at, expires_at,
			last_used_at, last_used_ip, last_used_client, last_call) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
			id, ws.ID, k.kind, k.name, k.desc, masked, hash, pickedFirst(k.perms), scope, creator, k.created, k.expires,
			k.lastUsed, ip, k.client, k.lastCall); err != nil {
			return err
		}
		s.Extra["key:"+k.name] = id
	}
	// The old CI key still gets called: 212 rejected calls in the last 7
	// days from a GitHub Actions runner, the first on the day it expired.
	old := s.Extra["key:Old CI key"].(uuid.UUID)
	if _, err := s.Tx.Exec(ctx, `INSERT INTO api_key_rejections (api_key_id, day, ip, client, reason, calls, first_at, last_at)
		VALUES ($1, $2::date, '10.0.9.14', 'GitHub Actions runner', 'expired', 24, $3, $3)`, old, s.designDay(41, 0, 0), s.designDay(41, 2, 0)); err != nil {
		return err
	}
	perDay := Split(212, []float64{1, 1, 1, 1, 1, 1, 1})
	for i, n := range perDay {
		day := s.designDay(i, 2, 0)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO api_key_rejections (api_key_id, day, ip, client, reason, calls, first_at, last_at)
			VALUES ($1, $2::date, '10.0.9.14', 'GitHub Actions runner', 'expired', $3, $4, $4)
			ON CONFLICT (api_key_id, day, ip) DO UPDATE SET calls = api_key_rejections.calls + excluded.calls`, old, day, n, day); err != nil {
			return err
		}
	}
	if err := seedWebhooks(ctx, s); err != nil {
		return err
	}
	// The audit log streams to the warehouse; entries seeded so far count as delivered.
	secret, err := s.Box.SealString("whsec_demo_audit_stream", "workspace:"+ws.ID.String()+":audit_stream_secret")
	if err != nil {
		return err
	}
	_, err = s.Tx.Exec(ctx, `UPDATE workspaces SET audit_stream_enabled = true, audit_stream_url = 'https://warehouse.acme.io/ingest/audit',
		audit_stream_secret_enc = $2, audit_stream_cursor_at = $3, audit_stream_status = $4 WHERE id = $1`,
		ws.ID, secret, s.Now.Add(time.Second), json.RawMessage(fmt.Sprintf(`{"last_ok_at":%q,"delivered":41196}`, s.Ago(40*time.Second).UTC().Format(time.RFC3339))))
	return err
}

// seedWebhooks adds the three endpoints of the prototype with a day of
// delivery history: the product backend and Slack healthy, the warehouse
// failing for 55 minutes with events queued behind it.
func seedWebhooks(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh := s.Users["linh@acme.io"]
	type hook struct {
		name, desc, url string
		events          []string
		rotated         *time.Time
		total, ok       int
		failingSince    *time.Duration
	}
	d := func(v time.Duration) *time.Duration { return &v }
	rot := s.designDay(11, 10, 0)
	hooks := []hook{
		{name: "Product backend", url: "https://api.acme.io/hooks/reputecast", events: []string{"contact.subscribed", "contact.unsubscribed", "email.bounced"}, rotated: &rot, total: 2114, ok: 2110},
		{name: "Slack #email-ops", url: "https://hooks.slack.com/services/T0…/B0…", events: []string{"campaign.paused", "provider.down", "campaign.completed"}, total: 5, ok: 5},
		{name: "Data warehouse", desc: "all events", url: "https://warehouse.acme.io/ingest/events", events: []string{"email.*"}, total: 18402, ok: 11225, failingSince: d(55 * time.Minute)},
	}
	ids := map[string]uuid.UUID{}
	for _, h := range hooks {
		id := ID()
		ids[h.name] = id
		secret, err := s.Box.SealString("whsec_demo_"+id.String()[:8], "webhook:"+id.String()+":secret")
		if err != nil {
			return err
		}
		created := s.Ago(120 * 24 * time.Hour)
		rotated := &created
		if h.rotated != nil {
			rotated = h.rotated
		}
		var failing *time.Time
		lastStatus := 200
		if h.failingSince != nil {
			f := s.Ago(*h.failingSince)
			failing = &f
			lastStatus = 502
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO webhooks (id, workspace_id, name, description, url, events, secret_enc, enabled, created_by, created_at, secret_rotated_at,
			last_status, last_delivery_at, failing_since) VALUES ($1,$2,$3,$4,$5,$6,$7,true,$8,$9,$10,$11,$12,$13)`,
			id, ws.ID, h.name, h.desc, h.url, h.events, secret, linh.ID, created, rotated, lastStatus, s.Ago(time.Minute), failing); err != nil {
			return err
		}
		// A day of history, older than the rows shown at the top of the log.
		events := h.events
		if h.name == "Data warehouse" {
			events = []string{"email.delivered", "email.opened", "email.clicked", "email.bounced"}
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO webhook_deliveries (id, webhook_id, event, event_id, payload, attempt, max_attempts, state, status_code, duration_ms,
				created_at, last_attempt_at, delivered_at)
			SELECT gen_random_uuid(), $1, ($2::text[])[1 + (g % cardinality($2::text[]))], gen_random_uuid(),
			       jsonb_build_object('type', ($2::text[])[1 + (g % cardinality($2::text[]))], 'workspace', $3::text),
			       1, 8, 'delivered', 200, 60 + (g % 90),
			       $4::timestamptz - (g * ($5::double precision / $6)) * interval '1 second',
			       $4::timestamptz - (g * ($5::double precision / $6)) * interval '1 second' + interval '1 second',
			       $4::timestamptz - (g * ($5::double precision / $6)) * interval '1 second' + interval '1 second'
			FROM generate_series(1, $6::int) g`,
			id, events, ws.Slug, s.Ago(57*time.Minute), float64(23*3600), h.ok); err != nil {
			return err
		}
		failed := h.total - h.ok
		if failed > 0 && h.name == "Product backend" {
			// A few failures that later went through on retry count as failed attempts.
			if _, err := s.Tx.Exec(ctx, `INSERT INTO webhook_deliveries (id, webhook_id, event, event_id, payload, attempt, max_attempts, state, status_code, duration_ms, error,
					created_at, last_attempt_at)
				SELECT gen_random_uuid(), $1, 'contact.subscribed', gen_random_uuid(), jsonb_build_object('type', 'contact.subscribed'), 8, 8, 'dead', 503, 30000, 'HTTP 503 Service Unavailable',
				       $2::timestamptz - g * interval '30 minutes', $2::timestamptz - g * interval '30 minutes' + interval '2 hours'
				FROM generate_series(1, $3::int) g`, id, s.Ago(20*time.Hour), failed); err != nil {
				return err
			}
		}
		if failed > 0 && h.name == "Data warehouse" {
			// Everything since the endpoint started failing is queued behind the probe.
			if _, err := s.Tx.Exec(ctx, `INSERT INTO webhook_deliveries (id, webhook_id, event, event_id, payload, attempt, max_attempts, state, status_code, duration_ms, error,
					created_at, last_attempt_at, next_attempt_at)
				SELECT gen_random_uuid(), $1, (ARRAY['email.delivered','email.opened','email.clicked'])[1 + (g % 3)], gen_random_uuid(),
				       jsonb_build_object('type', (ARRAY['email.delivered','email.opened','email.clicked'])[1 + (g % 3)]),
				       0, 8, 'pending', NULL, NULL, '', $2::timestamptz + g * ($3::double precision / $4) * interval '1 second', NULL, now()
				FROM generate_series(1, $4::int) g`, id, s.Ago(54*time.Minute), float64(52*60), failed-2); err != nil {
				return err
			}
		}
	}
	// The five rows at the top of the delivery log.
	type row struct {
		hook, event, subtype, state string
		at                          time.Duration
		code, ms, attempt           int
		next                        *time.Duration
	}
	rows := []row{
		{hook: "Product backend", event: "email.delivered", state: "delivered", at: 1*time.Minute + 48*time.Second, code: 200, ms: 84, attempt: 1},
		{hook: "Data warehouse", event: "email.delivered", state: "retrying", at: 1*time.Minute + 49*time.Second, code: 502, ms: 1240, attempt: 4, next: d(-(8*time.Minute - 1*time.Minute - 49*time.Second))},
		{hook: "Product backend", event: "email.bounced", subtype: "hard", state: "delivered", at: 2*time.Minute + 2*time.Second, code: 200, ms: 91, attempt: 1},
		{hook: "Slack #email-ops", event: "campaign.paused", state: "delivered", at: 25*time.Minute + 46*time.Second, code: 200, ms: 212, attempt: 1},
		{hook: "Data warehouse", event: "email.opened", state: "dead", at: 56*time.Minute + 57*time.Second, code: 502, ms: 1310, attempt: 8},
	}
	for _, r := range rows {
		at := s.Ago(r.at)
		data := map[string]any{"campaign": "October Product Update"}
		if r.subtype != "" {
			data = map[string]any{"subtype": r.subtype, "provider": "ses", "provider_code": "550 5.1.1", "email": "j.doe@example.com"}
		}
		payload, _ := json.Marshal(map[string]any{"id": ID().String(), "type": r.event, "created_at": at.UTC(), "workspace": ws.Slug, "data": data})
		var next *time.Time
		if r.next != nil {
			n := s.Ago(*r.next)
			next = &n
		}
		var delivered *time.Time
		errText, resp := "", "ok"
		if r.state == "delivered" {
			delivered = &at
		} else {
			errText, resp = "HTTP 502 Bad Gateway", "<html><head><title>502 Bad Gateway</title></head><body>nginx</body></html>"
		}
		created := at
		if r.state == "retrying" {
			// The probe: the oldest open delivery of the failing endpoint.
			created = s.Ago(55 * time.Minute)
		} else if r.attempt > 1 {
			created = at.Add(-time.Duration(r.attempt) * 7 * time.Minute)
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO webhook_deliveries (id, webhook_id, event, event_id, payload, attempt, max_attempts, state, status_code, duration_ms, response, error,
			created_at, last_attempt_at, next_attempt_at, delivered_at, enqueued_at) VALUES ($1,$2,$3,$4,$5,$6,8,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
			ID(), ids[r.hook], r.event, ID(), payload, r.attempt, r.state, r.code, r.ms, resp, errText, created, at, next, delivered, at); err != nil {
			return err
		}
	}
	return nil
}
