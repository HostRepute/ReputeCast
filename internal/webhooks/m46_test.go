package webhooks_test

import (
	"context"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// TestRotationKeepsTheOldSecretSigningADay: after a rotation each delivery
// carries both secrets' signatures, the new first, so a receiver still on
// the old secret keeps working for 24 hours (review 2026-11 IN-1); an
// endpoint failing for an hour tells the people who manage webhooks once
// (IN-2).
func TestRotationKeepsTheOldSecretSigningADay(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Cfg.Webhooks.AllowPrivateNetworks = true
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	rcv := &receiver{}
	srv := httptest.NewServer(rcv)
	defer srv.Close()
	var created struct {
		Webhook struct{ ID string }
		Secret  string
	}
	c.Do("POST", ws+"/webhooks", map[string]any{"name": "Rotating", "url": srv.URL + "/hook", "events": []string{"email.*"}}).MustStatus(t, 201).JSON(t, &created)
	old := created.Secret
	var rotated struct{ Secret string }
	c.Do("POST", ws+"/webhooks/"+created.Webhook.ID+"/rotate-secret", nil).MustStatus(t, 200).JSON(t, &rotated)
	if rotated.Secret == "" || rotated.Secret == old {
		t.Fatalf("rotated: %+v", rotated)
	}
	test := func() (string, int64, []byte) {
		t.Helper()
		c.Do("POST", ws+"/webhooks/"+created.Webhook.ID+"/test", nil).MustStatus(t, 200)
		rcv.mu.Lock()
		defer rcv.mu.Unlock()
		i := len(rcv.sigs) - 1
		ts, _ := strconv.ParseInt(rcv.ts[i], 10, 64)
		return rcv.sigs[i], ts, rcv.bodies[i]
	}
	sig, ts, body := test()
	if parts := strings.Split(sig, ","); len(parts) != 2 || parts[0] != webhooks.Sign(rotated.Secret, ts, body) {
		t.Fatalf("both signatures, the new first: %q", sig)
	}
	if !webhooks.Verify(old, sig, ts, body, time.Now(), 5*time.Minute) || !webhooks.Verify(rotated.Secret, sig, ts, body, time.Now(), 5*time.Minute) {
		t.Fatal("either secret verifies during the overlap")
	}
	var list struct {
		Items []struct {
			ID                  string
			PreviousSecretUntil *time.Time
		}
	}
	c.Do("GET", ws+"/webhooks", nil).MustStatus(t, 200).JSON(t, &list)
	for _, it := range list.Items {
		if it.ID == created.Webhook.ID && (it.PreviousSecretUntil == nil || time.Until(*it.PreviousSecretUntil) < 23*time.Hour) {
			t.Fatalf("the overlap shows: %+v", it)
		}
	}
	// A day later only the new secret signs.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE webhooks SET previous_secret_until = now() - interval '1 minute' WHERE id = $1`, created.Webhook.ID); err != nil {
		t.Fatal(err)
	}
	sig, ts, body = test()
	if strings.Contains(sig, ",") || webhooks.Verify(old, sig, ts, body, time.Now(), 5*time.Minute) {
		t.Fatalf("after the overlap: %q", sig)
	}

	// Failing: told once the endpoint has failed for an hour.
	rcv.mu.Lock()
	rcv.status = 502
	rcv.mu.Unlock()
	w, _ := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	alerts := func() int {
		t.Helper()
		var n int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE kind = 'webhook_failing' AND data->>'webhook' = $1`, created.Webhook.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	attempt := func() {
		t.Helper()
		if _, err := webhooks.Emit(ctx, env.App.Pool, w.ID, w.Slug, "email.bounced", map[string]any{"subtype": "hard"}); err != nil {
			t.Fatal(err)
		}
		var id uuid.UUID
		if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM webhook_deliveries WHERE webhook_id = $1 AND state = 'pending' ORDER BY created_at DESC LIMIT 1`, created.Webhook.ID).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if _, err := service(env).Attempt(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	attempt()
	if alerts() != 0 {
		t.Fatal("a first failure tells nobody")
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE webhooks SET failing_since = now() - interval '2 hours' WHERE id = $1`, created.Webhook.ID); err != nil {
		t.Fatal(err)
	}
	attempt()
	n := alerts()
	if n == 0 {
		t.Fatal("an hour of failures tells the people who manage webhooks")
	}
	attempt()
	if alerts() != n {
		t.Fatal("once per failing spell")
	}
}
