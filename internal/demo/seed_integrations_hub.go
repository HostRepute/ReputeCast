package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/mjml"
)

func init() { seedIntegrationsHub = seedIntegrationsHubImpl }

// hubTime is a moment of the expansion's sample day relative to its "now",
// Wed 15 Oct 2026 14:20: hubTime(14, 18) is two minutes ago.
func (s *Seeder) hubTime(hour, minute int) time.Time {
	return s.Now.Add(-(time.Duration(14*60+20-(hour*60+minute)) * time.Minute))
}

// hubDay is a day before the sample day at a clock time, as hubTime
// counts: hubDay(1, 22, 10) is 13 Oct 22:10 when the sample day is the 15th.
func (s *Seeder) hubDay(daysAgo, hour, minute int) time.Time {
	return s.hubTime(hour, minute).Add(-time.Duration(daysAgo) * 24 * time.Hour)
}

// hubIntegration is a connection of the sample with its key.
type hubIntegration struct {
	kind, label         string
	items               []string
	by                  string
	at                  time.Time
	head, tail          string
	lastUsed            time.Time
	ip, client          string
	settings            map[string]any
	firstCall, lastCall time.Time
}

// seedIntegrationsHubImpl adds the integrations hub of the expansion's
// sample (brief 4): Zapier and n8n connected with their calls and
// workflows, n8n's managed webhook, Make not connected, and the WordPress
// sites blog.acme.io and staging.acme.io with their post-draft rule and the
// draft it made.
func seedIntegrationsHubImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	marcus, linh, sofia := s.Users["marcus@acme.io"], s.Users["linh@acme.io"], s.Users["sofia@acme.io"]
	platform := []string{"contact.subscribed", "contact.unsubscribed", "contact.updated", "form.submitted", "campaign.completed", "message.bounced",
		"upsert_contact", "add_to_list", "add_tag", "record_event", "send_transactional"}
	conns := []hubIntegration{
		{kind: "zapier", label: "Acme ops", items: platform, by: "marcus@acme.io", at: s.hubDay(21, 10, 12), head: "5d1e", tail: "7b20",
			lastUsed: s.hubTime(14, 18), ip: "3.214.87.19", client: "Zapier, ReputeCast app 1.2.0", firstCall: s.hubDay(21, 10, 19), lastCall: s.hubTime(14, 18)},
		{kind: "n8n", label: "Acme automations", items: []string{"contact.subscribed", "contact.unsubscribed", "contact.updated", "form.submitted",
			"upsert_contact", "add_to_list", "add_tag", "record_event"}, by: "linh@acme.io", at: s.hubDay(16, 11, 40), head: "9a40", tail: "e1c3",
			lastUsed: s.hubTime(14, 14), ip: "198.51.100.61", client: "n8n, ReputeCast node 1.1.0", settings: map[string]any{"address": "https://n8n.acme.io"},
			firstCall: s.hubDay(16, 11, 46), lastCall: s.hubTime(14, 14)},
		{kind: "wordpress", label: "blog.acme.io", items: []string{"list_forms", "create_drafts"}, by: "marcus@acme.io", at: s.hubDay(20, 15, 30),
			head: "2b7d", tail: "a51f", lastUsed: s.hubTime(14, 14), ip: "203.0.113.80", client: "WordPress, ReputeCast plugin 1.2.0",
			firstCall: s.hubDay(20, 15, 33), lastCall: s.hubTime(14, 14)},
		{kind: "wordpress", label: "staging.acme.io", items: []string{"list_forms", "create_drafts"}, by: "marcus@acme.io", at: s.hubDay(13, 9, 20),
			head: "c03e", tail: "6d92", lastUsed: s.hubDay(1, 22, 10), ip: "203.0.113.81", client: "WordPress, ReputeCast plugin 1.3.0",
			firstCall: s.hubDay(13, 9, 24), lastCall: s.hubDay(1, 22, 10)},
	}
	ids := map[string]uuid.UUID{}
	keys := map[string]uuid.UUID{}
	for _, c := range conns {
		id := ID()
		ids[c.label] = id
		settings, _ := json.Marshal(c.settings)
		if c.settings == nil {
			settings = []byte(`{}`)
		}
		off := []string{}
		for _, k := range append(append([]string{}, platform...), "list_forms", "create_drafts") {
			if c.kind == "wordpress" && k != "list_forms" && k != "create_drafts" || c.kind != "wordpress" && (k == "list_forms" || k == "create_drafts") {
				continue
			}
			found := false
			for _, it := range c.items {
				if it == k {
					found = true
				}
			}
			if !found {
				off = append(off, k)
			}
		}
		by := s.Users[c.by].ID
		if _, err := s.Tx.Exec(ctx, `INSERT INTO integrations (id, workspace_id, kind, label, items, off_items, settings, connected_by, connected_at, first_call_at, last_call_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, ws.ID, c.kind, c.label, c.items, off, settings, by, c.at, c.firstCall, c.lastCall); err != nil {
			return fmt.Errorf("integration %s: %w", c.label, err)
		}
		keyID := ID()
		keys[c.label] = keyID
		_, hash := demoKey("rc_live_", c.head, c.tail)
		name := hubName(c.kind) + " · " + c.label
		perms := hubPermissions(c.kind, c.items)
		ip := netip.MustParseAddr(c.ip)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO api_keys (id, workspace_id, kind, name, prefix, token_hash, permissions, scope, created_by, created_at,
			last_used_at, last_used_ip, last_used_client, integration_id) VALUES ($1,$2,'api_key',$3,$4,$5,$6,'{}',$7,$8,$9,$10,$11,$12)`,
			keyID, ws.ID, name, "rc_live_"+c.head+"…"+c.tail, hash, perms, by, c.at, c.lastUsed, ip, c.client, id); err != nil {
			return fmt.Errorf("integration key %s: %w", name, err)
		}
		s.Extra["integration:"+c.label] = id
		if err := hubAudit(ctx, s, s.Users[c.by].ID, s.Users[c.by].Name, "user", "integration.connected", id, fmt.Sprintf("%s · key “%s”", hubName(c.kind), name),
			map[string]any{"summary": hubConnectSummary(c.kind, c.items), "permission": string(authz.IntegrationsManage)}, c.at); err != nil {
			return err
		}
	}
	if err := seedHubZapierCalls(ctx, s, ids["Acme ops"], keys["Acme ops"]); err != nil {
		return err
	}
	if err := seedHubN8n(ctx, s, ids["Acme automations"], keys["Acme automations"], linh.ID); err != nil {
		return err
	}
	if err := seedHubWordPress(ctx, s, ids, keys, marcus.ID, sofia.ID); err != nil {
		return err
	}
	return nil
}

func hubName(kind string) string {
	switch kind {
	case "zapier":
		return "Zapier"
	case "make":
		return "Make"
	case "n8n":
		return "n8n"
	}
	return "WordPress"
}

// hubPermissions are the permissions the sample's keys hold, as the hub
// stores them: what the items need, then what that includes.
func hubPermissions(kind string, items []string) []string {
	var out []authz.Permission
	has := func(k string) bool {
		for _, i := range items {
			if i == k {
				return true
			}
		}
		return false
	}
	switch kind {
	case "wordpress":
		out = []authz.Permission{authz.SubscribersView, authz.CampaignsEdit}
	case "zapier":
		out = []authz.Permission{authz.SubscribersCreate, authz.SubscribersEdit, authz.CampaignsView, authz.TransactionalView, authz.TransactionalSend, authz.WebhooksManage}
	default:
		out = []authz.Permission{authz.SubscribersCreate, authz.SubscribersEdit}
		if has("contact.subscribed") {
			out = append(out, authz.WebhooksManage)
		}
	}
	return pickedFirst(out)
}

func hubConnectSummary(kind string, items []string) string {
	if kind == "wordpress" {
		return "list signup forms, create drafts from posts · all lists · no expiry"
	}
	return fmt.Sprintf("%d triggers and actions · all lists · no expiry", len(items))
}

// hubAudit writes one audit entry of the sample.
func hubAudit(ctx context.Context, s *Seeder, actorID uuid.UUID, actorName, actorType, action string, target uuid.UUID, targetName string, meta map[string]any, at time.Time) error {
	ws := s.WS()
	m, _ := json.Marshal(meta)
	_, err := s.Tx.Exec(ctx, `INSERT INTO audit_events (id, workspace_id, occurred_at, actor_type, actor_id, actor_name, action, target_type, target_id, target_name,
		result, ip, user_agent, request_id, metadata) VALUES ($1,$2,$3,$4,$5,$6,$7,'integration',$8,$9,'allowed',$10,'',$11,$12)`,
		ID(), ws.ID, at, actorType, actorID, actorName, action, target.String(), targetName, netip.MustParseAddr("203.0.113.9"), "req_"+ID().String()[:8], m)
	return err
}

// hubCall is one call of a sample call log.
type hubCall struct {
	at                time.Time
	wfKey, wfName     string
	op, item, method  string
	path              string
	status, ms        int
	code, detail      string
	target            map[string]any
	note              string
	request, response any
}

func insertHubCall(ctx context.Context, s *Seeder, integration, key uuid.UUID, ip, client string, c hubCall) error {
	ws := s.WS()
	target, _ := json.Marshal(c.target)
	var req, resp []byte
	if c.request != nil {
		req, _ = json.Marshal(c.request)
	}
	if c.response != nil {
		resp, _ = json.Marshal(c.response)
	}
	_, err := s.Tx.Exec(ctx, `INSERT INTO integration_calls (id, integration_id, workspace_id, api_key_id, occurred_at, workflow_key, workflow_name, operation, item,
		method, path, status, code, detail, duration_ms, target, note, request, response, ip, client)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
		ID(), integration, ws.ID, key, c.at, c.wfKey, c.wfName, c.op, c.item, c.method, c.path, c.status, c.code, c.detail, c.ms, target, c.note,
		nullJSON(req), nullJSON(resp), netip.MustParseAddr(ip), client)
	return err
}

func nullJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

// spread places n moments evenly between from and to, newest first.
func spread(n int, from, to time.Time) []time.Time {
	out := make([]time.Time, n)
	if n == 1 {
		out[0] = to
		return out
	}
	step := to.Sub(from) / time.Duration(n-1)
	for i := range n {
		out[i] = to.Add(-time.Duration(i) * step)
	}
	return out
}

// seedHubZapierCalls adds Zapier's 169 calls of the last 7 days from its
// three Zaps, with the two errors fixed in the Zap and the one where
// consent was kept.
func seedHubZapierCalls(ctx context.Context, s *Seeder, id, key uuid.UUID) error {
	ws := s.WS()
	const ip, client = "3.214.87.19", "Zapier, ReputeCast app 1.2.0"
	beta := struct{ key, name string }{"283551240", "Beta application (Typeform) → ReputeCast contact"}
	hub := struct{ key, name string }{"283551917", "HubSpot deal won → tag customer"}
	stripe := struct{ key, name string }{"284113907", "Stripe payment failed → payment-failed email"}
	contacts := "/api/v1/workspaces/" + ws.Slug + "/contacts/upsert"
	tags := "/api/v1/workspaces/" + ws.Slug + "/contacts/tags"
	messages := "/api/v1/workspaces/" + ws.Slug + "/transactional/messages"
	betaList := s.Lists["Beta Program"]
	applicants := []string{"priya.n@example.com", "j.okafor@example.org", "m.lindqvist@example.se", "carla.mendes@example.com.br", "t.nakamura@example.jp",
		"a.haddad@example.com", "sven.berg@example.no", "leah.cohen@example.com"}
	// Beta applications: 20 in the last 24 hours, 108 before, one kept out.
	times := append(spread(20, s.Ago(23*time.Hour+40*time.Minute), s.hubTime(14, 18)), spread(108, s.hubDay(6, 15, 0), s.hubDay(1, 13, 50))...)
	for i, at := range times {
		email := applicants[i%len(applicants)]
		c := hubCall{at: at, wfKey: beta.key, wfName: beta.name, op: "upsert-contact", item: "upsert_contact", method: "POST", path: contacts, status: 200, ms: 44 + (i*7)%27,
			target: map[string]any{"kind": "upsert_contact", "email": email, "list": "Beta Program"},
			request: map[string]any{"headers": map[string]string{"X-ReputeCast-Workflow": beta.name + "; id=" + beta.key},
				"body": map[string]any{"email": email, "firstName": "", "tags": []string{"beta-applicant"}, "lists": []string{betaList.String()}, "consent": "double"}},
			response: map[string]any{"contactId": ID().String(), "email": email, "created": true, "confirmationSent": true,
				"lists": []map[string]any{{"id": betaList.String(), "name": "Beta Program", "status": "pending"}}, "tags": []string{"beta-applicant"}}}
		if i == 0 {
			c.ms = 64
		}
		if err := insertHubCall(ctx, s, id, key, ip, client, c); err != nil {
			return err
		}
	}
	left := s.hubDay(13, 9, 41).In(time.UTC)
	if err := insertHubCall(ctx, s, id, key, ip, client, hubCall{at: s.hubDay(1, 16, 12), wfKey: beta.key, wfName: beta.name, op: "upsert-contact", item: "upsert_contact",
		method: "POST", path: contacts, status: 409, ms: 52, code: "unsubscribed",
		detail: "rob@example.net left Beta Program on 2 Oct. ReputeCast doesn't add people back to a list they left.",
		target: map[string]any{"kind": "upsert_contact", "email": "rob@example.net", "list": "Beta Program"},
		request: map[string]any{"headers": map[string]string{"X-ReputeCast-Workflow": beta.name + "; id=" + beta.key},
			"body": map[string]any{"email": "rob@example.net", "tags": []string{"beta-applicant"}, "lists": []string{betaList.String()}, "consent": "double"}},
		response: map[string]any{"status": 409, "title": "Conflict", "code": "unsubscribed",
			"detail": "rob@example.net left Beta Program on 2 Oct. ReputeCast doesn't add people back to a list they left.",
			"meta":   map[string]any{"list": "Beta Program", "listId": betaList.String(), "leftAt": left}}}); err != nil {
		return err
	}
	// HubSpot deals: 7 in the last 24 hours, 30 before.
	buyers := []string{"dmitri@fabrikam.example", "ops@tailspin.example", "kim@wingtip.example", "finance@adatum.example"}
	times = append(spread(7, s.Ago(23*time.Hour+10*time.Minute), s.hubTime(12, 51)), spread(30, s.hubDay(6, 16, 30), s.hubDay(1, 12, 0))...)
	for i, at := range times {
		email := buyers[i%len(buyers)]
		if err := insertHubCall(ctx, s, id, key, ip, client, hubCall{at: at, wfKey: hub.key, wfName: hub.name, op: "tag-contact", item: "add_tag", method: "POST",
			path: tags, status: 200, ms: 33 + (i*5)%19, target: map[string]any{"kind": "add_tag", "email": email, "tag": "customer"},
			request:  map[string]any{"headers": map[string]string{"X-ReputeCast-Workflow": hub.name + "; id=" + hub.key}, "body": map[string]any{"email": email, "tags": []string{"customer"}}},
			response: map[string]any{"contactId": ID().String(), "email": email, "created": false, "lists": []any{}, "tags": []string{"customer"}, "confirmationSent": false}}); err != nil {
			return err
		}
	}
	// Stripe: two runs without amount_due, fixed in the Zap at 12:15.
	fixed, _ := json.Marshal(map[string]any{"fixedBy": "Marcus Chen", "fixedAt": s.hubTime(12, 15)})
	send := func(at time.Time, email string, status, ms int, idem string) hubCall {
		c := hubCall{at: at, wfKey: stripe.key, wfName: stripe.name, op: "send-transactional-message", item: "send_transactional", method: "POST", path: messages,
			status: status, ms: ms, target: map[string]any{"kind": "send_transactional", "template": "payment-failed", "email": email},
			request: map[string]any{"headers": map[string]string{"X-ReputeCast-Workflow": stripe.name + "; id=" + stripe.key, "Idempotency-Key": "zap-284113907-" + idem},
				"body": map[string]any{"template": "payment-failed", "to": map[string]any{"email": email}, "data": map[string]any{"first_name": "Priya", "update_url": "••••••"}}},
			response: map[string]any{"id": "msg_" + ID().String()[:12], "status": "queued"}}
		if status == 422 {
			c.code = "missing_variables"
			c.detail = "payment-failed v1 needs amount_due."
			c.note = string(fixed)
			c.response = map[string]any{"type": "https://reputecast.dev/problems/missing_variables", "title": "Missing variables", "status": 422,
				"code": "missing_variables", "detail": "payment-failed v1 needs amount_due.", "missing": []string{"amount_due"}}
		}
		return c
	}
	for _, c := range []hubCall{
		send(s.hubTime(11, 52), "billing@northwind.example", 422, 38, "evt_1QKw9F"),
		send(s.hubTime(12, 6), "finance@initech.example", 422, 41, "evt_1QKx2L"),
		send(s.hubTime(12, 40), "finance@initech.example", 202, 61, "evt_1QKx2L-2"),
		send(s.hubTime(13, 55), "accounts@contoso.example", 202, 58, "evt_1QL0aB"),
	} {
		if err := insertHubCall(ctx, s, id, key, ip, client, c); err != nil {
			return err
		}
	}
	for _, w := range []struct {
		key, name string
		uses      any
		first     time.Time
		last      time.Time
	}{
		{beta.key, beta.name, []map[string]string{{"item": "upsert_contact", "detail": "Beta Program · tag beta-applicant · double opt-in"}}, s.hubDay(21, 10, 30), s.hubTime(14, 18)},
		{hub.key, hub.name, []map[string]string{{"item": "add_tag", "detail": "customer"}}, s.hubDay(19, 9, 0), s.hubTime(12, 51)},
		{stripe.key, stripe.name, []map[string]string{{"item": "send_transactional", "detail": "payment-failed"}}, s.hubTime(11, 52), s.hubTime(13, 55)},
	} {
		uses, _ := json.Marshal(w.uses)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO integration_workflows (integration_id, key, external_id, name, uses, first_seen_at, last_seen_at) VALUES ($1,$2,$2,$3,$4,$5,$6)`,
			id, w.key, w.name, uses, w.first, w.last); err != nil {
			return err
		}
	}
	return nil
}

// seedHubN8n adds n8n's workflows, their calls and the managed webhook of
// "Unsubscribes → Acme Cloud CRM", healthy, with 14 deliveries today.
func seedHubN8n(ctx context.Context, s *Seeder, id, key, linh uuid.UUID) error {
	ws := s.WS()
	const ip, client = "198.51.100.61", "n8n, ReputeCast node 1.1.0"
	events := "/api/v1/workspaces/" + ws.Slug + "/automations/events"
	people := []string{"ana@northwind.example", "dev@contoso.example", "it@fabrikam.example", "ops@wingtip.example", "team@tailspin.example"}
	for _, wf := range []struct {
		key, name, event string
		n                int
		last             time.Time
	}{
		{"wf_7Hq2", "Acme Cloud: workspace created", "workspace.created", 61, s.hubTime(14, 14)},
		{"wf_9Kx4", "Acme Cloud: teammate invited", "teammate.invited", 23, s.hubTime(13, 37)},
	} {
		for i, at := range spread(wf.n, s.hubDay(6, 14, 0), wf.last) {
			email := people[i%len(people)]
			if err := insertHubCall(ctx, s, id, key, ip, client, hubCall{at: at, wfKey: wf.key, wfName: wf.name, op: "send-automation-event", item: "record_event",
				method: "POST", path: events, status: 202, ms: 21 + (i*3)%17, target: map[string]any{"kind": "record_event", "event": wf.event, "email": email},
				request:  map[string]any{"headers": map[string]string{"X-ReputeCast-Workflow": wf.name + "; id=" + wf.key}, "body": map[string]any{"event": wf.event, "email": email}},
				response: map[string]any{"accepted": true, "contactId": ID().String(), "flows": 1}}); err != nil {
				return err
			}
		}
		uses, _ := json.Marshal([]map[string]string{{"item": "record_event", "detail": wf.event}})
		if _, err := s.Tx.Exec(ctx, `INSERT INTO integration_workflows (integration_id, key, external_id, name, uses, first_seen_at, last_seen_at) VALUES ($1,$2,$2,$3,$4,$5,$6)`,
			id, wf.key, wf.name, uses, s.hubDay(16, 12, 0), wf.last); err != nil {
			return err
		}
	}
	// The managed webhook n8n created when the workflow was turned on.
	hookID := ID()
	created := s.hubDay(16, 12, 4)
	secret, err := s.Box.SealString("whsec_demo_"+hookID.String()[:8], "webhook:"+hookID.String()+":secret")
	if err != nil {
		return err
	}
	url := "https://n8n.acme.io/webhook/7c1e9d42-3b8a-4f6e-a1c5-2d9e8f7b6a30/reputecast"
	// Demo endpoint: its address is fictional, so nothing is sent to it.
	if _, err := s.Tx.Exec(ctx, `INSERT INTO webhooks (id, workspace_id, name, description, url, events, secret_enc, enabled, created_by, created_at, secret_rotated_at,
		last_status, last_delivery_at, demo) VALUES ($1,$2,'n8n · Unsubscribes → Acme Cloud CRM','',$3,$4,$5,true,$6,$7,$7,200,$8,true)`,
		hookID, ws.ID, url, []string{"contact.unsubscribed"}, secret, linh, created, s.Ago(16*time.Minute)); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO integration_webhooks (webhook_id, integration_id, api_key_id, event, workflow_key, workflow_name, created_at)
		VALUES ($1,$2,$3,'contact.unsubscribed','wf_2Lm8','Unsubscribes → Acme Cloud CRM',$4)`, hookID, id, key, created); err != nil {
		return err
	}
	leavers := []string{"nora.lind@example.com", "k.osei@example.net", "pablo.diaz@example.org", "m.tanaka@example.com", "eli.brooks@example.net"}
	for i, at := range spread(14, s.Ago(23*time.Hour), s.Ago(16*time.Minute)) {
		email := leavers[i%len(leavers)]
		payload, _ := json.Marshal(map[string]any{"id": ID().String(), "type": "contact.unsubscribed", "created_at": at.UTC(), "workspace": ws.Slug,
			"data": map[string]any{"email": email, "list_name": "Weekly Digest", "source": "unsubscribe_page"}})
		ms := 120 + (i*11)%60
		if i == 0 {
			ms = 140
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO webhook_deliveries (id, webhook_id, event, event_id, payload, attempt, max_attempts, state, status_code, duration_ms, response,
			created_at, last_attempt_at, delivered_at, enqueued_at) VALUES ($1,$2,'contact.unsubscribed',$3,$4,1,8,'delivered',200,$5,'ok',$6,$6,$6,$6)`,
			ID(), hookID, ID(), payload, ms, at); err != nil {
			return err
		}
	}
	uses, _ := json.Marshal([]map[string]string{{"item": "contact.unsubscribed"}})
	if _, err := s.Tx.Exec(ctx, `INSERT INTO integration_workflows (integration_id, key, external_id, name, uses, first_seen_at, last_seen_at) VALUES ($1,'wf_2Lm8','wf_2Lm8',$2,$3,$4,$5)`,
		id, "Unsubscribes → Acme Cloud CRM", uses, created, created); err != nil {
		return err
	}
	if err := insertHubCall(ctx, s, id, key, ip, client, hubCall{at: created, wfKey: "wf_2Lm8", wfName: "Unsubscribes → Acme Cloud CRM", op: "subscribe-integration-hook",
		item: "contact.unsubscribed", method: "POST", path: "/api/v1/workspaces/" + ws.Slug + "/integration/hooks", status: 201, ms: 48,
		target: map[string]any{"kind": "subscribe", "event": "contact.unsubscribed", "endpoint": "https://n8n.acme.io/webhook/7c1e…/reputecast"}}); err != nil {
		return err
	}
	return hubAudit(ctx, s, key, "n8n · Acme automations", "api_key", "webhook.created", hookID, url,
		map[string]any{"summary": "managed by n8n · contact.unsubscribed · workflow Unsubscribes → Acme Cloud CRM", "permission": string(authz.WebhooksManage),
			"actor_label": "integration key", "source": "api", "via": "rc_live_9a40…e1c3"}, created)
}

// blogPostMJML is the "Blog post" template: the post's image, title,
// excerpt and button fill its post section.
const blogPostMJML = `<mjml>
  <mj-head><mj-attributes><mj-all font-family="Georgia, 'Times New Roman', serif"/></mj-attributes></mj-head>
  <mj-body background-color="#E9ECF0" width="600px">
    <mj-section background-color="#FFFFFF" padding="18px 24px"><mj-column>
      <mj-text font-family="Helvetica, Arial, sans-serif" font-size="15px" font-weight="700" color="#0B1220" padding="0">Acme Product</mj-text>
    </mj-column></mj-section>
    <mj-section background-color="#FFFFFF" padding="0"><mj-column>
      <mj-image src="{{ post.image_url }}" alt="{{ post.title }}" href="{{ post.url }}" padding="0"/>
      <mj-text font-size="24px" font-weight="700" line-height="1.25" color="#0B1220" padding="24px 24px 8px"><h1 style="margin:0;font-size:24px">{{ post.title }}</h1></mj-text>
      <mj-text font-size="16px" line-height="1.55" color="#374151" padding="0 24px 16px">{{ post.excerpt }}</mj-text>
      <mj-button href="{{ post.url }}" background-color="#0F766E" color="#FFFFFF" align="left" font-family="Helvetica, Arial, sans-serif" font-weight="600" padding="0 24px 28px">Read more</mj-button>
    </mj-column></mj-section>
    <mj-section padding="16px 24px"><mj-column>
      <mj-text font-family="Helvetica, Arial, sans-serif" font-size="12px" color="#5B6472" align="center" padding="0">Acme Inc · 500 Howard St, San Francisco, CA · <a href="{{ unsubscribe_url }}" style="color:#5B6472">Unsubscribe</a></mj-text>
    </mj-column></mj-section>
  </mj-body>
</mjml>`

// seedHubWordPress adds the two sites, the "Blog post" template, the rule
// on blog.acme.io and the draft it made at 10:05, with Sofia Rossi's
// notification, and the attention notification about staging.acme.io.
func seedHubWordPress(ctx context.Context, s *Seeder, ids, keys map[string]uuid.UUID, marcus, sofia uuid.UUID) error {
	ws := s.WS()
	html := mjml.Compile(blogPostMJML).HTML
	tplID := ID()
	if _, err := s.Tx.Exec(ctx, `INSERT INTO templates (id, workspace_id, kind, name, description, mjml, html, thumb, size_bytes, created_by, updated_by, created_at, updated_at)
		VALUES ($1,$2,'template','Blog post','Saved template · for post drafts from WordPress',$3,$4,$5,$6,$7,$7,$8,$8)`,
		tplID, ws.ID, blogPostMJML, html, json.RawMessage(`[{"h":10,"w":40},{"h":34},{"h":10},{"h":8}]`), len(html), marcus, s.hubDay(20, 15, 50)); err != nil {
		return err
	}
	cats := []map[string]string{{"id": "3", "name": "Announcements", "slug": "announcements"}, {"id": "4", "name": "Engineering", "slug": "engineering"},
		{"id": "5", "name": "Releases", "slug": "releases"}, {"id": "6", "name": "Customers", "slug": "customers"}, {"id": "7", "name": "Company news", "slug": "company-news"}}
	blogCats, _ := json.Marshal(cats)
	blogForms, _ := json.Marshal([]map[string]any{
		{"form": "frm_8c2e1a", "kind": "block", "places": []map[string]string{{"title": "Sidebar", "url": "https://blog.acme.io/"}, {"title": "About", "url": "https://blog.acme.io/about/"}}},
		{"form": "frm_b7a412", "kind": "shortcode", "places": []map[string]string{{"title": "Join the beta", "url": "https://blog.acme.io/join-the-beta/"}}},
	})
	stagingForms, _ := json.Marshal([]map[string]any{
		{"form": "frm_8c2e1a", "kind": "block", "places": []map[string]string{{"title": "Home", "url": "https://staging.acme.io/"}}},
	})
	product := s.Lists["Product Updates"]
	sender, err := composerEnsureSender(ctx, s, "news.acme.io", "Acme Product", "product@news.acme.io")
	if err != nil {
		return err
	}
	rule, _ := json.Marshal(map[string]any{"categories": []string{"announcements"}, "listIds": []string{product.String()}, "templateId": tplID.String(),
		"senderId": sender.String(), "notifyUserId": sofia.String()})
	blog, staging := ids["blog.acme.io"], ids["staging.acme.io"]
	if _, err := s.Tx.Exec(ctx, `INSERT INTO wordpress_sites (integration_id, workspace_id, address, site_url, name, plugin_version, wp_version, php_version, feed_url,
		categories, forms, first_report_at, last_report_at, rule_enabled, rule, rule_updated_by, rule_updated_at)
		VALUES ($1,$2,'blog.acme.io','https://blog.acme.io','Acme Blog','1.2.0','6.9.2','8.3','https://blog.acme.io/feed/',$3,$4,$5,$6,true,$7,$8,$9)`,
		blog, ws.ID, blogCats, blogForms, s.hubDay(20, 15, 33), s.hubTime(14, 14), rule, marcus, s.hubDay(20, 16, 0)); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO wordpress_sites (integration_id, workspace_id, address, site_url, name, plugin_version, wp_version, php_version, feed_url,
		categories, forms, first_report_at, last_report_at, last_check_at, last_check_ok, last_check_detail)
		VALUES ($1,$2,'staging.acme.io','https://staging.acme.io','Acme Blog (staging)','1.3.0','6.9.2','8.2','https://staging.acme.io/feed/',$3,$4,$5,$6,$7,false,'Timed out after 10 s')`,
		staging, ws.ID, blogCats, stagingForms, s.hubDay(13, 9, 24), s.hubDay(1, 22, 10), s.hubTime(14, 2)); err != nil {
		return err
	}
	// The post published at 10:04 and its draft, a minute later.
	post := struct{ title, excerpt, url string }{"Scheduled rollouts are here", "You can now schedule flag changes ahead of time, in any timezone.",
		"https://blog.acme.io/scheduled-rollouts/"}
	src := blogPostMJML
	for k, v := range map[string]string{"{{ post.title }}": post.title, "{{ post.excerpt }}": post.excerpt, "{{ post.url }}": post.url,
		"{{ post.image_url }}": "https://blog.acme.io/wp-content/uploads/2026/10/scheduled-rollouts.png"} {
		src = strings.ReplaceAll(src, k, v)
	}
	draftHTML := mjml.Compile(src).HTML
	campaign := ID()
	created := s.hubTime(10, 5)
	audience, _ := json.Marshal(map[string]any{"lists": []uuid.UUID{product}})
	if _, err := s.Tx.Exec(ctx, `INSERT INTO campaigns (id, workspace_id, name, status, subject, preheader, sender_id, audience, editor_mode, mjml, html, plain_text,
		version, content_hash, origin, note, created_by, updated_by, created_at, updated_at, autosaved_at)
		VALUES ($1,$2,$3,'draft',$3,$4,$5,$6,'visual',$7,$8,'',1,'','wordpress','draft from blog.acme.io',$9,$9,$10,$10,$10)`,
		campaign, ws.ID, post.title, post.excerpt, sender, audience, src, draftHTML, marcus, created); err != nil {
		return err
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO campaign_versions (campaign_id, version, subject, preheader, mjml, html, saved_by, saved_at, label, source, size_bytes)
		VALUES ($1,1,$2,$3,$4,$5,$6,$7,'created from the post on blog.acme.io','api',$8)`, campaign, post.title, post.excerpt, src, draftHTML, marcus, created, len(draftHTML)); err != nil {
		return err
	}
	s.Campaigns[post.title] = campaign
	if _, err := s.Tx.Exec(ctx, `INSERT INTO wordpress_drafts (id, integration_id, workspace_id, campaign_id, post_id, post_title, post_url, published_at, notified_user, created_at)
		VALUES ($1,$2,$3,$4,'812',$5,$6,$7,$8,$9)`, ID(), blog, ws.ID, campaign, post.title, post.url, s.hubTime(10, 4), sofia, created); err != nil {
		return err
	}
	wsID := ws.ID
	if _, err := s.Tx.Exec(ctx, `INSERT INTO notifications (id, user_id, workspace_id, kind, severity, title, summary, body, link, created_at, email_status, emailed_at)
		VALUES ($1,$2,$3,'post_draft_created','info','Post draft created','',$4,$5,$6,'sent',$6)`,
		ID(), sofia, &wsID, "“Scheduled rollouts are here” from blog.acme.io is a draft for Product Updates. Review it, then send it through Send Readiness as usual.",
		fmt.Sprintf("/w/%s/campaigns/%s/edit", ws.Slug, campaign), created); err != nil {
		return err
	}
	keyBlog := keys["blog.acme.io"]
	if err := hubAudit(ctx, s, keyBlog, "WordPress · blog.acme.io", "api_key", "wordpress.draft_created", campaign, "Product Updates · "+post.title,
		map[string]any{"summary": "draft from the post published 10:04 · rule Announcements → Product Updates · Sofia Rossi notified",
			"actor_label": "integration key", "source": "api", "via": "rc_live_2b7d…a51f"}, created.Add(9*time.Second)); err != nil {
		return err
	}
	// staging.acme.io went quiet on 13 Oct at 22:10: members with
	// integrations:manage heard about it a day later.
	silent := s.hubDay(1, 22, 10)
	attention, _ := json.Marshal([]map[string]any{{"code": "site_silent", "since": silent, "host": "staging.acme.io", "connectionId": staging.String()}})
	if _, err := s.Tx.Exec(ctx, `UPDATE integrations SET attention = $2, notified = '{site_silent}' WHERE id = $1`, staging, attention); err != nil {
		return err
	}
	for _, email := range []string{"linh@acme.io", "marcus@acme.io"} {
		at := silent.Add(24 * time.Hour)
		read := at.Add(40 * time.Minute)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO notifications (id, user_id, workspace_id, kind, severity, title, summary, body, link, created_at, read_at, email_status, emailed_at)
			VALUES ($1,$2,$3,'integration_attention','warn','Integration needs attention','',$4,$5,$6,$7,'sent',$6)`,
			ID(), s.Users[email].ID, &wsID, "WordPress: staging.acme.io hasn't reported in since 13 Oct 22:10. Check the site, or disconnect it if it's gone.",
			"/w/"+ws.Slug+"/settings/integrations/wordpress", at, read); err != nil {
			return err
		}
	}
	// A few report-ins and form lookups for the sites' call logs.
	report := "/api/v1/workspaces/" + ws.Slug + "/wordpress/report"
	for _, c := range []struct {
		site    string
		at      time.Time
		ip, ver string
		version string
	}{
		{"blog.acme.io", s.hubTime(14, 14), "203.0.113.80", "WordPress, ReputeCast plugin 1.2.0", "1.2.0"},
		{"blog.acme.io", s.hubTime(10, 4), "203.0.113.80", "WordPress, ReputeCast plugin 1.2.0", "1.2.0"},
		{"staging.acme.io", s.hubDay(1, 22, 10), "203.0.113.81", "WordPress, ReputeCast plugin 1.3.0", "1.3.0"},
	} {
		if err := insertHubCall(ctx, s, ids[c.site], keys[c.site], c.ip, c.ver, hubCall{at: c.at, op: "report-wordpress-site", method: "POST", path: report,
			status: 200, ms: 31, target: map[string]any{"kind": "report", "version": c.version, "endpoint": c.site}}); err != nil {
			return err
		}
	}
	return insertHubCall(ctx, s, blog, keyBlog, "203.0.113.80", "WordPress, ReputeCast plugin 1.2.0", hubCall{at: created, op: "create-wordpress-draft", item: "create_drafts",
		method: "POST", path: "/api/v1/workspaces/" + ws.Slug + "/wordpress/drafts", status: 200, ms: 212, target: map[string]any{"kind": "draft", "title": post.title}})
}
