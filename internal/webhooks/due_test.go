package webhooks_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// While an endpoint is failing only its oldest open delivery is tried; a
// healthy endpoint's ready deliveries all are.
func TestDueProbesFailingEndpoints(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	hook := func(failing bool) uuid.UUID {
		id := uuid.Must(uuid.NewV7())
		var since *time.Time
		if failing {
			now := time.Now()
			since = &now
		}
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO webhooks (id, workspace_id, url, events, secret_enc, failing_since) VALUES ($1,$2,'https://example.test/hook','{email.*}','\x00',$3)`, id, w.ID, since); err != nil {
			t.Fatal(err)
		}
		return id
	}
	delivery := func(hook uuid.UUID, age time.Duration, state string) uuid.UUID {
		id := uuid.Must(uuid.NewV7())
		at := time.Now().Add(-age)
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO webhook_deliveries (id, webhook_id, event, payload, state, created_at, next_attempt_at) VALUES ($1,$2,'email.bounced','{}',$3,$4,$4)`,
			id, hook, state, at); err != nil {
			t.Fatal(err)
		}
		return id
	}
	failing, healthy := hook(true), hook(false)
	oldest := delivery(failing, 3*time.Hour, "retrying")
	waiting := delivery(failing, 2*time.Hour, "pending")
	h1, h2 := delivery(healthy, 2*time.Hour, "pending"), delivery(healthy, time.Hour, "pending")
	due := func() map[uuid.UUID]bool {
		ids, err := env.App.Q.HooksDue(ctx, 10000)
		if err != nil {
			t.Fatal(err)
		}
		out := map[uuid.UUID]bool{}
		for _, id := range ids {
			out[id] = true
		}
		return out
	}
	got := due()
	if !got[oldest] || got[waiting] || !got[h1] || !got[h2] {
		t.Fatalf("due: probe %v, waiting %v, healthy %v %v", got[oldest], got[waiting], got[h1], got[h2])
	}
	// The probe is in flight: nothing else of that endpoint goes out.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE webhook_deliveries SET state = 'delivering', enqueued_at = NULL WHERE id = $1`, oldest); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE webhook_deliveries SET enqueued_at = NULL WHERE id = $1`, waiting); err != nil {
		t.Fatal(err)
	}
	if got := due(); got[waiting] || got[oldest] {
		t.Fatalf("while the probe is in flight: %v %v", got[waiting], got[oldest])
	}
}
