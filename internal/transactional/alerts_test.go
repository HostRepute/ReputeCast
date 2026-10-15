package transactional_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/mcpserver"
)

// seedMessages inserts n transactional messages of a template accepted at
// at, with a status, a delivery time and a domain.
func (s *sendEnv) seedMessages(key string, n int, status string, at time.Time, deliverAfter time.Duration, domain string) {
	s.t.Helper()
	var tid uuid.UUID
	s.one(`SELECT id FROM transactional_templates WHERE workspace_id = $1 AND key = $2`, &tid, s.ws, key)
	for i := 0; i < n; i++ {
		id := uuid.Must(uuid.NewV7())
		var delivered *time.Time
		if status == "delivered" || status == "complained" {
			d := at.Add(deliverAfter)
			delivered = &d
		}
		s.exec(`INSERT INTO messages (id, workspace_id, stream, email, provider_id, status, queued_at, sent_at, delivered_at, kind)
			VALUES ($1, $2, 'transactional', $3, $4, $5, $6, $6, $7, 'api')`, id, s.ws, fmt.Sprintf("user%d.%s@%s", i, status, domain), s.prov["postmark"], status, at, delivered)
		s.exec(`INSERT INTO transactional_messages (message_id, workspace_id, source, template_id, template_key, template_name, template_version, accepted_at, route)
			VALUES ($1, $2, 'api', $3, $4, $4, 1, $5, '{}')`, id, s.ws, tid, key, at)
	}
}

type overview struct {
	Stats struct {
		Sent, Delivered, Bounced, Rejected, Suppressed, OptedOut int64
		DeliveryRate                                             float64
		MedianMs                                                 *int64
		Keys                                                     int64
		ComplainedOn                                             []string
	}
	Route struct {
		Configured        bool
		Primary, Fallback *struct{ Name string }
		System            *struct{ Name string }
		Policy            string
		Priority          bool
	}
	Failover *struct {
		From, To struct{ Name string }
		Messages int64
	}
	Alerts []struct {
		ID, Kind string
		Template *struct{ ID, Name string }
		Value    float64
		Domains  []struct {
			Domain string
			Count  int64
		}
	}
	Incidents []struct {
		Kind    string
		EndedAt *time.Time
	}
	Keys     []struct{ Name, Preset, Mode string }
	FirstRun struct {
		Templates, Published int64
		Keys                 int
		VerifiedSender       bool
	}
	Endpoint string
}

func TestDeliveryAlerts(t *testing.T) {
	s := newSendEnv(t)
	now := s.clock.now()
	ctx := context.Background()
	s.publish("magic-link", "Magic sign-in link", "magic-link")
	// 30 sign-in links in the last hour, 3 bounced at typo domains: 10%.
	s.seedMessages("magic-link", 27, "delivered", now.Add(-30*time.Minute), 1200*time.Millisecond, "example.com")
	s.seedMessages("magic-link", 2, "bounced", now.Add(-20*time.Minute), 0, "gmial.com")
	s.seedMessages("magic-link", 1, "bounced", now.Add(-20*time.Minute), 0, "yaho.com")
	if err := s.env.App.Transactional().CheckAlerts(ctx); err != nil {
		t.Fatal(err)
	}
	var ov overview
	s.owner.Do("GET", base+"/overview?period=24h", nil).MustStatus(t, 200).JSON(t, &ov)
	if len(ov.Alerts) != 1 || ov.Alerts[0].Kind != "bounce_rate" || ov.Alerts[0].Template.ID != "magic-link" || ov.Alerts[0].Value != 10 ||
		len(ov.Alerts[0].Domains) != 2 || ov.Alerts[0].Domains[0].Domain != "gmial.com" {
		t.Fatalf("alert: %+v", ov.Alerts)
	}
	// Members with transactional:view are told once; the next check keeps it.
	var body string
	s.one(`SELECT body FROM notifications WHERE kind = 'transactional_alert' AND user_id = (SELECT id FROM users WHERE email = 'aisha@acme.io')`, &body)
	if !strings.Contains(body, "Magic sign-in link: bounce rate 10% in one hour (limit 2%). Most bounces went to gmial.com and yaho.com.") {
		t.Fatalf("notification: %s", body)
	}
	if err := s.env.App.Transactional().CheckAlerts(ctx); err != nil {
		t.Fatal(err)
	}
	if n := s.count(`SELECT count(*) FROM notifications WHERE kind = 'transactional_alert' AND user_id = (SELECT id FROM users WHERE email = 'aisha@acme.io')`); n != 1 {
		t.Fatalf("notified %d times", n)
	}
	var center struct {
		Items []struct{ Kind, Category, CategoryLabel string }
	}
	s.owner.Do("GET", "/workspaces/acme-newsletter/notifications?filter=transactional", nil).MustStatus(t, 200).JSON(t, &center)
	if len(center.Items) == 0 || center.Items[0].Category != "transactional" || center.Items[0].CategoryLabel != "Transactional" {
		t.Fatalf("notification center: %+v", center.Items)
	}
	s.owner.Do("POST", base+"/alerts/"+ov.Alerts[0].ID+"/seen", nil).MustStatus(t, 204)

	// Failed messages and a slow hour raise their own alerts.
	s.seedMessages("magic-link", 10, "failed", now.Add(-10*time.Minute), 0, "example.org")
	s.exec(`UPDATE messages SET delivered_at = queued_at + interval '45 seconds' WHERE workspace_id = $1 AND status = 'delivered' AND stream = 'transactional'`, s.ws)
	if err := s.env.App.Transactional().CheckAlerts(ctx); err != nil {
		t.Fatal(err)
	}
	s.owner.Do("GET", base+"/overview?period=24h", nil).MustStatus(t, 200).JSON(t, &ov)
	kinds := []string{}
	for _, a := range ov.Alerts {
		kinds = append(kinds, a.Kind)
	}
	if strings.Join(kinds, ",") != "median_delivery,failed,bounce_rate" && len(kinds) != 3 {
		t.Fatalf("alerts: %v", kinds)
	}

	// The numbers recover: the alerts end and stay as incidents.
	s.exec(`UPDATE messages SET status = 'delivered', delivered_at = queued_at + interval '1 second' WHERE workspace_id = $1 AND stream = 'transactional'`, s.ws)
	if err := s.env.App.Transactional().CheckAlerts(ctx); err != nil {
		t.Fatal(err)
	}
	s.owner.Do("GET", base+"/overview?period=24h", nil).MustStatus(t, 200).JSON(t, &ov)
	if len(ov.Alerts) != 0 || len(ov.Incidents) != 3 || ov.Incidents[0].EndedAt == nil {
		t.Fatalf("after recovery: alerts %+v incidents %+v", ov.Alerts, ov.Incidents)
	}

	// An outage: messages fail over, the overview says where mail goes.
	_, secret := s.key("Production", map[string]any{"preset": "send_only"})
	s.exec(`UPDATE providers SET status = 'down' WHERE id = $1`, s.prov["postmark"])
	for i := 0; i < 3; i++ {
		s.call(secret, "/messages", "", resetBody(fmt.Sprintf("u%d@contoso.example", i))).MustStatus(t, 202)
	}
	if err := s.env.App.Transactional().CheckAlerts(ctx); err != nil {
		t.Fatal(err)
	}
	s.owner.Do("GET", base+"/overview?period=24h", nil).MustStatus(t, 200).JSON(t, &ov)
	if ov.Failover == nil || ov.Failover.From.Name != "Postmark" || ov.Failover.To.Name != "Amazon SES" || ov.Failover.Messages != 3 {
		t.Fatalf("failover: %+v", ov.Failover)
	}

	// The audit log's Transactional filter.
	var audit struct{ Items []struct{ Action string } }
	s.owner.Do("GET", "/workspaces/acme-newsletter/audit?category=transactional&range=24h", nil).MustStatus(t, 200).JSON(t, &audit)
	if len(audit.Items) == 0 {
		t.Fatal("no transactional audit entries")
	}
	for _, it := range audit.Items {
		if !strings.HasPrefix(it.Action, "transactional.") {
			t.Fatalf("filter let through %s", it.Action)
		}
	}
}

func TestOverview(t *testing.T) {
	s := newSendEnv(t)
	_, secret := s.key("Acme Cloud · production", map[string]any{"preset": "send_only"})
	s.key("Acme Cloud · staging", map[string]any{"preset": "send_only", "mode": "test", "testAddresses": []string{"qa+*@acme.io"}})
	s.exec(`INSERT INTO suppressions (id, workspace_id, email, reason, source) VALUES ($1, $2, 'gone@outlook.com', 'hard_bounce', 'webhook')`, uuid.Must(uuid.NewV7()), s.ws)
	s.call(secret, "/messages", "", resetBody("sam.rivera@outlook.com")).MustStatus(t, 202)
	s.call(secret, "/messages", "", resetBody("gone@outlook.com")).MustStatus(t, 422)
	s.step()
	var ov overview
	s.owner.Do("GET", base+"/overview", nil).MustStatus(t, 200).JSON(t, &ov)
	if ov.Stats.Sent != 1 || ov.Stats.Delivered != 1 || ov.Stats.DeliveryRate != 100 || ov.Stats.Suppressed != 1 || ov.Stats.Rejected != 1 || ov.Stats.Keys != 1 || ov.Stats.MedianMs == nil {
		t.Fatalf("stats: %+v", ov.Stats)
	}
	if !ov.Route.Configured || ov.Route.Primary.Name != "Postmark" || ov.Route.Fallback.Name != "Amazon SES" || ov.Route.Policy != "automatic" || !ov.Route.Priority {
		t.Fatalf("route: %+v", ov.Route)
	}
	if len(ov.Keys) != 2 || ov.Keys[1].Mode != "test" || ov.FirstRun.Templates != 1 || ov.FirstRun.Published != 1 || !ov.FirstRun.VerifiedSender ||
		!strings.HasSuffix(ov.Endpoint, "/api/v1/workspaces/acme-newsletter/transactional/messages") {
		t.Fatalf("keys %+v first run %+v endpoint %s", ov.Keys, ov.FirstRun, ov.Endpoint)
	}
	// No Transactional route: the System route's provider is offered.
	s.exec(`DELETE FROM routing_rules WHERE workspace_id = $1 AND stream = 'transactional'`, s.ws)
	s.owner.Do("GET", base+"/overview", nil).MustStatus(t, 200).JSON(t, &ov)
	if ov.Route.Configured || ov.Route.System == nil || ov.Route.System.Name != "Postmark" {
		t.Fatalf("no route: %+v", ov.Route)
	}
	// Viewers read it.
	aisha := s.env.Client()
	aisha.SignInDemo("aisha@acme.io")
	aisha.Do("GET", base+"/overview", nil).MustStatus(t, 200)
}

func TestAgentsReadTransactional(t *testing.T) {
	s := newSendEnv(t)
	_, secret := s.key("Production", map[string]any{"preset": "send_only"})
	s.call(secret, "/messages", "", resetBody("sam.rivera@outlook.com")).MustStatus(t, 202)
	var agent struct{ Secret string }
	s.owner.Do("POST", "/workspaces/acme-newsletter/mcp-agents", map[string]any{"name": "Support bot", "permissions": []string{"transactional:view"}}).MustStatus(t, 201).JSON(t, &agent)
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	cs, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: s.env.Server.URL + mcpserver.Path,
		HTTPClient: &http.Client{Transport: bearerTransport{agent.Secret}}, DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cs.Close() }()
	call := func(name string, args map[string]any) map[string]any {
		t.Helper()
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil || res.IsError {
			t.Fatalf("%s: %v %+v", name, err, res)
		}
		var out map[string]any
		for _, c := range res.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				_ = json.Unmarshal([]byte(tc.Text), &out)
			}
		}
		return out
	}
	list := call("list_transactional_templates", nil)
	if tpls, _ := list["templates"].([]any); len(tpls) != 1 {
		t.Fatalf("templates: %v", list)
	}
	r := call("render_transactional_template", map[string]any{"template": "password-reset", "test_data": "New user"})
	if r["ok"] != true || !strings.Contains(r["subject"].(string), "Reset your") {
		t.Fatalf("render: %v", r)
	}
	log := call("search_transactional_messages", map[string]any{"query": "sam.rivera"})
	msgs, _ := log["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("log: %v", log)
	}
	if _, ok := msgs[0].(map[string]any)["html"]; ok {
		t.Fatal("agents got content")
	}
	if n := s.count(`SELECT count(*) FROM audit_events WHERE action = 'transactional.log_read' AND actor_type = 'mcp_agent'`); n != 1 {
		t.Fatalf("agent reads audited: %d", n)
	}
}

type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

func TestSendersAndSuppressionsForTransactional(t *testing.T) {
	s := newSendEnv(t)
	var senders struct {
		Senders []struct {
			ID, Email     string
			Transactional bool
			DefaultFor    []string
		}
		Domains []struct {
			ID, Domain    string
			Transactional bool
		}
		Defaults struct{ TransactionalSenderID string }
	}
	s.owner.Do("GET", "/workspaces/acme-newsletter/senders", nil).MustStatus(t, 200).JSON(t, &senders)
	var notify, news, noReply, digest string
	for _, d := range senders.Domains {
		switch d.Domain {
		case "notify.acme.io":
			notify = d.ID
			if !d.Transactional {
				t.Fatal("notify.acme.io isn't Transactional")
			}
		case "news.acme.io":
			news = d.ID
		}
	}
	for _, sn := range senders.Senders {
		switch sn.Email {
		case "no-reply@notify.acme.io":
			noReply = sn.ID
			if !sn.Transactional {
				t.Fatal("sender not marked Transactional")
			}
		case "digest@news.acme.io":
			digest = sn.ID
		}
	}
	if notify == "" || noReply == "" || news == "" || digest == "" {
		t.Fatalf("senders: %+v", senders)
	}
	// The default From of transactional templates.
	s.owner.Do("PUT", "/workspaces/acme-newsletter/senders/defaults", map[string]any{"defaultSenderId": digest, "replyHandling": "forward", "footerAddress": "500 Market St", "transactionalSenderId": digest}).MustStatus(t, 422)
	s.owner.Do("PUT", "/workspaces/acme-newsletter/senders/defaults", map[string]any{"defaultSenderId": digest, "replyHandling": "forward", "footerAddress": "500 Market St", "transactionalSenderId": noReply}).MustStatus(t, 200)
	s.owner.Do("GET", "/workspaces/acme-newsletter/senders", nil).MustStatus(t, 200).JSON(t, &senders)
	if senders.Defaults.TransactionalSenderID != noReply {
		t.Fatalf("default: %+v", senders.Defaults)
	}
	var settings struct{ DefaultSender *struct{ ID string } }
	s.owner.Do("GET", base+"/settings", nil).MustStatus(t, 200).JSON(t, &settings)
	if settings.DefaultSender == nil || settings.DefaultSender.ID != noReply {
		t.Fatalf("settings: %+v", settings)
	}
	// A domain becomes Transactional from Settings › Senders.
	s.owner.Do("PATCH", "/workspaces/acme-newsletter/domains/"+news, map[string]any{"transactional": true}).MustStatus(t, 200)
	if n := s.count(`SELECT count(*) FROM sending_domains WHERE id = $1 AND transactional`, news); n != 1 {
		t.Fatal("domain not marked")
	}

	// Suppressions say what they apply to.
	s.exec(`INSERT INTO suppressions (id, workspace_id, email, reason, source, scope) VALUES ($1, $2, 'left@outlook.com', 'unsubscribe', 'one_click', 'marketing'),
		($3, $2, 'gone@outlook.com', 'hard_bounce', 'webhook', 'all'), ($4, $2, 'info@contoso.example', 'role', 'import', 'all')`,
		uuid.Must(uuid.NewV7()), s.ws, uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))
	var list struct {
		Items []struct {
			Address string
			Scope   string
		}
	}
	s.owner.Do("GET", "/workspaces/acme-newsletter/suppressions?appliesTo=marketing&q=contoso&pageSize=100", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].Scope != "marketing" {
		t.Fatalf("role is marketing only: %+v", list.Items)
	}
	s.owner.Do("GET", "/workspaces/acme-newsletter/suppressions?appliesTo=all&q=gone@outlook.com", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].Scope != "all" {
		t.Fatalf("hard bounce applies to all: %+v", list.Items)
	}
}
