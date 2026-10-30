package connect_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/connect"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/systemmail"
)

// PR-3: an SES setup finishes only once a real event came back. ReputeCast
// sends a message to the SES mailbox simulator from a verified identity and
// waits for that message's Delivery event from the simulator (review R3.A1:
// not the Send event SES publishes first); until then Finish is refused.
// Without a verified identity yet, the check waits for the domain step.
func TestSESEventCheckGatesFinish(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	// A fake SES API: identities verified or not, sends accepted.
	var mu sync.Mutex
	verified := false
	sent := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/email/identities/"):
			_, _ = io.WriteString(w, `{"IdentityType":"DOMAIN","VerifiedForSendingStatus":`+map[bool]string{true: "true", false: "false"}[verified]+`}`)
		case r.URL.Path == "/v2/email/identities" || r.URL.Path == "/v2/email/list-identities":
			_, _ = io.WriteString(w, `{"EmailIdentities":[]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/email/outbound-emails":
			body, _ := io.ReadAll(r.Body)
			sent = append(sent, string(body))
			_, _ = io.WriteString(w, `{"MessageId":"0100019a-sim-check-0001"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"__type":"NotFoundException","message":"unexpected `+r.Method+` `+r.URL.Path+`"}`)
		}
	}))
	defer srv.Close()

	var st struct {
		ID         string
		Connection struct {
			Status, EventsCheck string
			CanContinue         bool
		}
		Done *struct {
			Rows []struct{ Status, Lead, Text string }
		}
		Status string
	}
	c.Do("POST", ws+"/connect/setups", map[string]any{"adapter": "ses"}).MustStatus(t, 201).JSON(t, &st)
	base := ws + "/connect/setups/" + st.ID

	// The connection step as it ends for SES keys (its AWS calls are the
	// adapter's tests): the provider is saved and the event check waits.
	wsRow, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	pid := uuid.Must(uuid.NewV7())
	cfg, _ := json.Marshal(map[string]any{"region": "us-east-1", "access": "keys", "endpoint": srv.URL, "events": "webhook",
		"topic_arn": "arn:aws:sns:us-east-1:123456789012:reputecast-events"})
	creds, err := env.App.Secrets.Seal([]byte(`{"access_key_id":"AKIDEXAMPLE","secret_access_key":"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}`),
		manage.CredentialsAD(store.Provider{ID: pid}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, status, ip_mode, ips,
		capabilities, limits, health, pricing, connected_at)
		VALUES ($1, $2, 'ses', 'Amazon SES (check)', '', $3, $4, 'token-sescheck-0001', 'healthy', 'shared', '{}', '{"webhooks":"SNS"}', '{}', '{}', '{}', now())`,
		pid, wsRow.ID, cfg, creds); err != nil {
		t.Fatal(err)
	}
	state := `{"connection":{"status":"connected","access":"keys","region":"us-east-1","events":"webhook","provisioning":[
		{"key":"verify","status":"done"},{"key":"account","status":"done"},{"key":"sns_topic","status":"done"},
		{"key":"event_send","status":"waiting","label":"Send a test to success@simulator.amazonses.com"},{"key":"event_check","status":"waiting","label":"Wait for its delivery event"}]}}`
	if _, err := env.App.Pool.Exec(ctx, `UPDATE provider_setups SET provider_id = $1, state = $2 WHERE id = $3`, pid, state, st.ID); err != nil {
		t.Fatal(err)
	}
	// rows is the setup's provisioning, for failure messages.
	rows := func() string {
		var raw string
		_ = env.App.Pool.QueryRow(ctx, `SELECT state->'connection'->>'provisioning' FROM provider_setups WHERE id = $1`, st.ID).Scan(&raw)
		return raw
	}
	finish := map[string]any{"pullSuppressions": false, "startEmpty": false}
	refused := func(when string) {
		t.Helper()
		r := c.Do("POST", base+"/finish", finish).MustStatus(t, 409)
		if !strings.Contains(r.Problem(t)["detail"].(string), "events aren't verified yet") {
			t.Fatalf("%s: %s", when, r.Body)
		}
	}
	refused("before the check")

	svc := &connect.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Factory: &manage.Factory{Box: env.App.Secrets},
		Deliver: &systemmail.Deliverer{Pool: env.App.Pool, Q: env.App.Q, Factory: &manage.Factory{Box: env.App.Secrets}}, URLs: env.App.URLs}
	setupID := uuid.MustParse(st.ID)

	// Nothing verified at SES yet: the check waits for the domain step, and
	// the person may go on to it, but not finish.
	if again, err := svc.RunProvisioning(ctx, setupID); err != nil || again {
		t.Fatalf("deferred run: again %v, %v", again, err)
	}
	c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &st)
	if st.Connection.EventsCheck != "deferred" || !st.Connection.CanContinue {
		t.Fatalf("deferred: %+v\n%s", st.Connection, rows())
	}
	refused("deferred")

	// Past the domain step the check goes out again: from a verified
	// sender, to the simulator.
	mu.Lock()
	verified = true
	mu.Unlock()
	c.Do("PATCH", base, map[string]any{"step": "test"}).MustStatus(t, 200)
	again, err := svc.RunProvisioning(ctx, setupID)
	if err != nil || !again {
		t.Fatalf("waiting run: again %v, %v", again, err)
	}
	mu.Lock()
	if len(sent) != 1 || !strings.Contains(sent[0], "success@simulator.amazonses.com") {
		t.Fatalf("sent %v", sent)
	}
	mu.Unlock()
	c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &st)
	if st.Connection.EventsCheck != "waiting" || st.Connection.CanContinue {
		t.Fatalf("waiting: %+v\n%s", st.Connection, rows())
	}
	refused("waiting for the event")

	// SES's events come back through the webhook as it publishes them and
	// are normalized like any other. Only the Delivery event of the check's
	// message, for the simulator's address, verifies the events: not
	// another message's, and not the Send event SES publishes first.
	norm := &ingest.Normalizer{Pool: env.App.Pool, Q: env.App.Q, Factory: &manage.Factory{Box: env.App.Secrets}, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	raw := func(eventType, messageID string, after time.Duration) {
		t.Helper()
		var sent time.Time
		if err := env.App.Pool.QueryRow(ctx, `SELECT sent_at FROM messages WHERE provider_id = $1 AND kind = 'event_check'`, pid).Scan(&sent); err != nil {
			t.Fatal(err)
		}
		at := sent.Add(after).UTC().Format("2006-01-02T15:04:05.000Z")
		ev := map[string]any{"eventType": eventType, "mail": map[string]any{"timestamp": sent.UTC().Format("2006-01-02T15:04:05.000Z"), "source": "reputecast-check@news.acme.io",
			"messageId": messageID, "destination": []string{"success@simulator.amazonses.com"}}}
		if eventType == "Delivery" {
			ev["delivery"] = map[string]any{"timestamp": at, "processingTimeMillis": 412, "recipients": []string{"success@simulator.amazonses.com"},
				"smtpResponse": "250 2.6.0 Message received", "reportingMTA": "a8-50.smtp-out.amazonses.com"}
		}
		inner, _ := json.Marshal(ev)
		body, _ := json.Marshal(map[string]any{"Type": "Notification", "MessageId": uuid.NewString(), "TopicArn": "arn:aws:sns:us-east-1:123456789012:reputecast-events",
			"Message": string(inner), "Timestamp": at})
		id := uuid.Must(uuid.NewV7())
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO provider_raw_events (id, workspace_id, provider_id, received_at, content_type, body, signature_ok, source, events)
			SELECT $1, workspace_id, provider_id, sent_at + $4::interval, 'text/plain', $2, true, 'webhook', 1 FROM messages WHERE provider_id = $3 AND kind = 'event_check'`,
			id, body, pid, after.String()); err != nil {
			t.Fatal(err)
		}
		if err := norm.Normalize(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	raw("Delivery", "0100019a-someone-else", 2*time.Second)
	if again, err := svc.RunProvisioning(ctx, setupID); err != nil || !again {
		t.Fatalf("another message's event counted: again %v, %v", again, err)
	}
	raw("Send", "0100019a-sim-check-0001", time.Second)
	if again, err := svc.RunProvisioning(ctx, setupID); err != nil || !again {
		t.Fatalf("the Send event counted as the delivery: again %v, %v", again, err)
	}
	c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &st)
	if st.Connection.EventsCheck != "waiting" || st.Connection.CanContinue {
		t.Fatalf("after the Send event: %+v\n%s", st.Connection, rows())
	}
	raw("Delivery", "0100019a-sim-check-0001", 3*time.Second)
	if again, err := svc.RunProvisioning(ctx, setupID); err != nil || again {
		t.Fatalf("verified run: again %v, %v", again, err)
	}
	c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &st)
	if st.Connection.EventsCheck != "verified" || !st.Connection.CanContinue {
		t.Fatalf("verified: %+v\n%s", st.Connection, rows())
	}
	if audits(t, env, "provider.events_verified") != 1 {
		t.Fatal("no audit entry for the verified events")
	}
	c.Do("POST", base+"/finish", finish).MustStatus(t, 200).JSON(t, &st)
	if st.Status != "finished" || st.Done == nil {
		t.Fatalf("finished: %+v", st)
	}
	found := false
	for _, r := range st.Done.Rows {
		if r.Lead == "Events" && r.Status == "verified" && strings.Contains(r.Text, "from the SES mailbox simulator") {
			found = true
		}
	}
	if !found {
		t.Fatalf("summary rows: %+v", st.Done.Rows)
	}
}
