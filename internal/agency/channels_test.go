package agency_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/agency"
	"github.com/hostrepute/reputecast/internal/apptest"
)

// SMS usage comes from the channel message log: what reached the provider
// counts on the day it was sent, per provider and per country; texts that
// were held, skipped or are still queued don't.
func TestSMSUsageFromTheMessageLog(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@coho.example", Name: "Owner"})
	ws := env.CreateWorkspace("Coho Winery", owner.ID)
	pool := env.App.Pool
	twilio, vonage := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	day := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
	for _, p := range []struct {
		id   uuid.UUID
		name string
	}{{twilio, "Twilio"}, {vonage, "Vonage"}} {
		if _, err := pool.Exec(ctx, `INSERT INTO sms_providers (id, workspace_id, adapter, name, webhook_token, connected_at) VALUES ($1, $2, $3, $4, $5, $6)`,
			p.id, ws.ID, p.name, p.name, p.id.String(), day.AddDate(0, -1, 0)); err != nil {
			t.Fatal(err)
		}
	}
	msg := func(provider uuid.UUID, country, status string, units int, cost float64, sent time.Time) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO channel_messages (id, workspace_id, channel, kind, recipient, country, provider_id, units, cost, status, created_at, sent_at, updated_at)
			VALUES ($1, $2, 'sms', 'campaign', '+15550100', $3, $4, $5, $6, $7, $8, $8, $8)`,
			uuid.Must(uuid.NewV7()), ws.ID, country, provider, units, cost, status, sent); err != nil {
			t.Fatal(err)
		}
	}
	msg(twilio, "US", "delivered", 2, 0.0158, day.Add(9*time.Hour))
	msg(twilio, "US", "undelivered", 1, 0.0079, day.Add(10*time.Hour))
	msg(twilio, "CA", "failed", 1, 0.0079, day.Add(11*time.Hour))
	msg(vonage, "GB", "sent", 3, 0.1203, day.Add(23*time.Hour+59*time.Minute))
	// Not counted: held, skipped and queued texts, and the next day's.
	msg(twilio, "US", "held", 1, 0, day.Add(12*time.Hour))
	msg(twilio, "US", "skipped", 1, 0, day.Add(12*time.Hour))
	msg(twilio, "US", "delivered", 1, 0.0079, day.AddDate(0, 0, 1))

	rows, err := agency.MessageChannel{Name: "sms"}.Day(ctx, pool, ws.ID, day.Add(15*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]agency.UsageRow{}
	for _, r := range rows {
		got[r.Dimension+"|"+r.Label] = r
	}
	if r := got["provider|Twilio"]; r.Units != 4 || r.CostMicros != 31600 {
		t.Fatalf("twilio: %+v", r)
	}
	if r := got["provider|Vonage"]; r.Units != 3 || r.CostMicros != 120300 {
		t.Fatalf("vonage: %+v", r)
	}
	if got["country|United States"].Units != 3 || got["country|Canada"].Units != 1 || got["country|United Kingdom"].Units != 3 {
		t.Fatalf("countries: %+v", rows)
	}
	// A channel with nothing in the log reads zero.
	if rows, err := (agency.MessageChannel{Name: "whatsapp"}).Day(ctx, pool, ws.ID, day); err != nil || len(rows) != 0 {
		t.Fatalf("whatsapp: %v %+v", err, rows)
	}
}
