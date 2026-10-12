package demo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hostrepute/reputecast/internal/authn"
)

func init() { seedProviders = seedProvidersImpl }

// seedProvidersImpl connects the three sending providers of the sample
// data: Amazon SES (Broadcasts 90%, Automations), Postmark (System) and a
// self-hosted Postal server warming up (Broadcasts 10%), with routing rules
// and failover policies per stream.
func seedProvidersImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	type prov struct {
		key, adapter, name, label, status, detail, ipMode string
		ips                                             []string
		config, capabilities, limits, health, pricing   string
		connected                                       time.Duration
		lastEvent                                       time.Duration
	}
	provs := []prov{
		{
			key: "ses", adapter: "ses", name: "Amazon SES", label: "us-east-1", status: "healthy", detail: "Production · no enforcement", ipMode: "shared",
			config:       `{"region":"us-east-1","configuration_set":"reputecast-acme-newsletter","access":"cloudformation","role_arn":"arn:aws:iam::123456789012:role/reputecast-ses"}`,
			capabilities: `{"bulk":"allowed","webhooks":"SNS","domain_api":true,"suppression_sync":true,"batch":50,"dedicated_ips":"optional, $24.95 per IP per month","sandbox":"detection","quota":"discovery","pricing":"$0.10 per 1,000"}`,
			limits:       `{"bounce":{"limit":5,"review":5,"pause":10,"label":"SES review"},"complaint":{"limit":0.1,"review":0.1,"pause":0.5}}`,
			health:       `{"bounce_rate":0.6,"complaint_rate":0.04,"quota":{"kind":"rolling_24h","max":50000,"used":2140},"rate":{"max":14,"now":0,"peak_today":14},"error_rate_7d":0.1,"account":"Production · no enforcement","webhook":"SNS"}`,
			pricing:      `{"model":"metered","per_1000":0.10,"source":"AWS pricing","cost_to_date":14.62}`,
			connected:    210 * 24 * time.Hour, lastEvent: 2 * time.Minute,
		},
		{
			key: "postmark", adapter: "postmark", name: "Postmark", label: "Transactional stream", status: "healthy", detail: "Approved · 10,000 per month plan", ipMode: "shared",
			config:       `{"server":"Acme Newsletter","message_stream":"outbound","broadcast_stream":null}`,
			capabilities: `{"bulk":"broadcast_streams_only","webhooks":"Registered","domain_api":true,"suppression_sync":true,"batch":500,"dedicated_ips":"from 300k per month","sandbox":"test token","quota":"discovery","pricing":"plan, $15 per month"}`,
			limits:       `{"bounce":{"limit":10,"label":"Postmark"},"complaint":{"limit":0.1,"label":"Postmark"}}`,
			health:       `{"bounce_rate":0.2,"complaint_rate":0.01,"quota":{"kind":"monthly_plan","max":10000,"used":3412},"rate":{"max":null,"now":0,"peak_today":3},"error_rate_7d":0.0,"account":"Approved · 10,000 per month plan","webhook":"Registered"}`,
			pricing:      `{"model":"plan","monthly":15,"per_1000":4.40,"source":"at current usage","cost_to_date":15.00}`,
			connected:    210 * 24 * time.Hour, lastEvent: 6 * time.Minute,
		},
		{
			key: "postal", adapter: "postal", name: "Postal", label: "self-hosted · mta.acme.io", status: "warming_up", detail: "Warming up · day 9 of 21", ipMode: "dedicated", ips: []string{"203.0.113.25"},
			config:       `{"server_url":"https://mta.acme.io","version":"3.3.4"}`,
			capabilities: `{"bulk":"your_policy","webhooks":"Registered","domain_api":"DKIM only","suppression_sync":true,"batch":true,"dedicated_ips":"Dedicated IP","sandbox":"mode","quota":"set manually","pricing":"self-hosted"}`,
			limits:       `{"bounce":{"limit":2,"label":"warm-up guard"},"complaint":{"limit":0.1}}`,
			health:       `{"bounce_rate":0.9,"complaint_rate":0.05,"quota":{"kind":"warmup_daily","max":6000,"used":0},"rate":{"max":2,"now":0,"peak_today":0,"label":"ramp limit"},"error_rate_7d":0.4,"account":"Postal 3.3.4 · reachable","webhook":"Registered"}`,
			pricing:      `{"model":"self_hosted","monthly":24,"per_1000":2.14,"source":"you entered $24 per month server","cost_to_date":0}`,
			connected:    9 * 24 * time.Hour, lastEvent: time.Minute,
		},
	}
	linh := s.Users["linh@acme.io"]
	for _, p := range provs {
		id := ID()
		token, _ := authn.NewToken()
		ips := p.ips
		if ips == nil {
			ips = []string{}
		}
		creds, err := s.Box.Seal([]byte(`{"demo":true}`), "provider:"+id.String()+":credentials")
		if err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, status, status_detail,
			ip_mode, ips, capabilities, limits, health, pricing, tracking, last_event_at, last_checked_at, connected_by, connected_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,'{"open_pixel":false,"link_rewrite":false}',$17,$18,$19,$20)`,
			id, ws.ID, p.adapter, p.name, p.label, json.RawMessage(p.config), creds, token, p.status, p.detail,
			p.ipMode, ips, json.RawMessage(p.capabilities), json.RawMessage(p.limits), json.RawMessage(p.health), json.RawMessage(p.pricing),
			s.Ago(p.lastEvent), s.Ago(time.Minute), linh.ID, s.Ago(p.connected)); err != nil {
			return err
		}
		s.Providers[p.key] = id
	}
	ses, postmark, postal := s.Providers["ses"], s.Providers["postmark"], s.Providers["postal"]
	beta := s.Lists["Beta Program"]
	rules := []struct {
		stream, name, conditions, splits string
		priority                          int
		fallback                          any
	}{
		{"broadcast", "Outlook and iCloud are strict with new IPs", `{"mailbox_providers":["outlook","icloud"]}`, `[{"provider_id":"` + ses.String() + `","weight":100}]`, 10, nil},
		{"broadcast", "small, high-value list", `{"lists":["` + beta.String() + `"]}`, `[{"provider_id":"` + ses.String() + `","weight":100}]`, 20, nil},
		{"broadcast", "", `{}`, `[{"provider_id":"` + ses.String() + `","weight":90},{"provider_id":"` + postal.String() + `","weight":10}]`, 100, nil},
		{"automation", "", `{}`, `[{"provider_id":"` + ses.String() + `","weight":100}]`, 100, postmark},
		{"system", "", `{}`, `[{"provider_id":"` + postmark.String() + `","weight":100}]`, 100, ses},
	}
	for _, r := range rules {
		if _, err := s.Tx.Exec(ctx, `INSERT INTO routing_rules (id, workspace_id, stream, priority, name, conditions, splits, fallback_provider_id, updated_by, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, ID(), ws.ID, r.stream, r.priority, r.name, json.RawMessage(r.conditions), json.RawMessage(r.splits), r.fallback, linh.ID, s.Ago(48*time.Hour)); err != nil {
			return err
		}
	}
	return nil
}
