package texting_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/texting"
	"github.com/hostrepute/reputecast/migrations"
)

// The SNS topic of AWS End User Messaging webhooks set up by hand: the
// webhook trusts only the one kept.

func addSmsProvider(t *testing.T, env *apptest.Env, ws uuid.UUID, adapter string, config map[string]any) store.SmsProvider {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	cfg, _ := json.Marshal(config)
	if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO sms_providers (id, workspace_id, adapter, name, config, webhook_token, webhooks, connected_at)
		VALUES ($1, $2, $3, 'AWS End User Messaging', $4, $5, 'manual', now())`, id, ws, adapter, cfg, "token-"+id.String()); err != nil {
		t.Fatal(err)
	}
	p, err := env.App.Q.SmsProvider(context.Background(), store.SmsProviderParams{ID: id, WorkspaceID: ws})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// The first topic kept stays: of two confirmations at once one keeps its
// topic, the other gets that one back, and keeping it is audited once.
func TestPinSettingKeepsTheFirstTopic(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	u := env.CreateUser(apptest.UserOpts{Email: "owner@snstopic.test", Name: "Pat Owner"})
	ws := env.CreateWorkspace("SNS Topic Test", u.ID)
	s := &texting.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, URLs: env.App.URLs}
	p := addSmsProvider(t, env, ws.ID, "aws_eum", map[string]any{"region": "us-east-1"})

	topics := []string{"arn:aws:sns:us-east-1:123456789012:reputecast-sms", "arn:aws:sns:us-east-1:999999999999:someone-else"}
	got := make([]string, len(topics))
	var wg sync.WaitGroup
	for i, topic := range topics {
		wg.Add(1)
		go func() {
			defer wg.Done()
			kept, err := s.PinSetting(ctx, p, "topic_arn", topic)
			if err != nil {
				t.Error(err)
			}
			got[i] = kept
		}()
	}
	wg.Wait()
	stored := one[string](t, env, `SELECT config->>'topic_arn' FROM sms_providers WHERE id = $1`, p.ID)
	if got[0] != stored || got[1] != stored || (stored != topics[0] && stored != topics[1]) {
		t.Fatalf("kept %v, stored %q", got, stored)
	}
	if region := one[string](t, env, `SELECT config->>'region' FROM sms_providers WHERE id = $1`, p.ID); region != "us-east-1" {
		t.Fatalf("the other settings: region %q", region)
	}
	if kept, err := s.PinSetting(ctx, p, "topic_arn", "arn:aws:sns:us-east-1:999999999999:later"); err != nil || kept != stored {
		t.Fatalf("later: %q, %v", kept, err)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'sms_provider.settings_updated' AND target_id = $2
		AND actor_type = 'system' AND after->>'topic_arn' = $3`, ws.ID, p.ID.String(), stored); n != 1 {
		t.Fatalf("audit entries: %d", n)
	}
}

// Providers whose webhooks were set up by hand before topics were kept
// keep the one topic their verified events came from (migration 03425);
// with events from several, none, so the next confirmation decides.
func TestSnsTopicBackfill(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	u := env.CreateUser(apptest.UserOpts{Email: "owner@snsbackfill.test", Name: "Pat Owner"})
	ws := env.CreateWorkspace("SNS Backfill Test", u.ID)
	gp, err := goose.NewProvider(goose.DialectPostgres, stdlib.OpenDBFromPool(env.App.Pool), migrations.FS, goose.WithAllowOutofOrder(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gp.Close() })
	const version = 3425
	if _, err := gp.ApplyVersion(ctx, version, false); err != nil {
		t.Fatal(err)
	}
	const mine, other = "arn:aws:sns:us-east-1:123456789012:acme-sms-events", "arn:aws:sns:us-east-1:999999999999:someone-else"
	event := func(p store.SmsProvider, topic string, verified bool) {
		t.Helper()
		body := `{"Type" : "Notification", "MessageId" : "` + uuid.NewString() + `", "TopicArn" : "` + topic + `", "Message" : "{}"}`
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO sms_raw_events (id, workspace_id, provider_id, kind, received_at, body, signature_ok)
			VALUES ($1, $2, $3, 'other', $4, $5, $6)`, uuid.Must(uuid.NewV7()), ws.ID, p.ID, time.Now(), []byte(body), verified); err != nil {
			t.Fatal(err)
		}
	}
	byHand := addSmsProvider(t, env, ws.ID, "aws_eum", map[string]any{"region": "us-east-1"})
	event(byHand, mine, true)
	event(byHand, mine, true)
	event(byHand, other, false) // refused, so never trusted
	several := addSmsProvider(t, env, ws.ID, "aws_eum", map[string]any{"region": "us-east-1"})
	event(several, mine, true)
	event(several, other, true)
	registered := addSmsProvider(t, env, ws.ID, "aws_eum", map[string]any{"region": "us-east-1", "topic_arn": mine})
	event(registered, other, true)
	quiet := addSmsProvider(t, env, ws.ID, "aws_eum", map[string]any{"region": "us-east-1"})
	twilio := addSmsProvider(t, env, ws.ID, "twilio", map[string]any{"account_sid": "AC123"})
	event(twilio, mine, true)

	if _, err := gp.ApplyVersion(ctx, version, true); err != nil {
		t.Fatal(err)
	}
	topic := func(p store.SmsProvider) string {
		t.Helper()
		return one[string](t, env, `SELECT coalesce(config->>'topic_arn', '') FROM sms_providers WHERE id = $1`, p.ID)
	}
	for _, c := range []struct {
		name string
		p    store.SmsProvider
		want string
	}{{"set up by hand", byHand, mine}, {"events from several topics", several, ""}, {"registered", registered, mine}, {"no events", quiet, ""}, {"another adapter", twilio, ""}} {
		if got := topic(c.p); got != c.want {
			t.Errorf("%s: topic %q, want %q", c.name, got, c.want)
		}
	}
	if region := one[string](t, env, `SELECT config->>'region' FROM sms_providers WHERE id = $1`, byHand.ID); region != "us-east-1" {
		t.Fatalf("the other settings: region %q", region)
	}
}
