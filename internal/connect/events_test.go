package connect_test

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Switching SES events between the webhook and an SQS queue changes the
// provider's settings in the transaction that records it: when the audit
// entry can't be written, the provider keeps its settings.
func TestEventsModeChangesWithItsAuditEntry(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var list struct{ Items []struct{ ID string } }
	c.Do("GET", ws+"/connect/setups?adapter=ses&status=finished", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 {
		t.Fatalf("setups: %+v", list.Items)
	}
	setup := list.Items[0].ID
	events := func() string {
		t.Helper()
		var e string
		if err := env.App.Pool.QueryRow(ctx, `SELECT coalesce(p.config->>'events', '') FROM providers p JOIN provider_setups s ON s.provider_id = p.id WHERE s.id = $1`, setup).Scan(&e); err != nil {
			t.Fatal(err)
		}
		return e
	}
	for _, sql := range []string{
		`CREATE FUNCTION test_refuse_events_mode() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.action = 'provider.events_mode' THEN
				RAISE EXCEPTION 'the audit log is unavailable';
			END IF;
			RETURN NEW;
		END $$`,
		`CREATE TRIGGER test_refuse_events_mode BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION test_refuse_events_mode()`,
	} {
		if _, err := env.App.Pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	c.Do("PUT", ws+"/connect/setups/"+setup+"/events", map[string]any{"events": "queue"}).MustStatus(t, 500)
	if e := events(); e != "" {
		t.Fatalf("the provider's events became %q without an audit entry", e)
	}

	if _, err := env.App.Pool.Exec(ctx, `DROP TRIGGER test_refuse_events_mode ON audit_events`); err != nil {
		t.Fatal(err)
	}
	c.Do("PUT", ws+"/connect/setups/"+setup+"/events", map[string]any{"events": "queue"}).MustStatus(t, 200)
	if e, n := events(), audits(t, env, "provider.events_mode"); e != "queue" || n != 1 {
		t.Fatalf("events %q with %d audit entries", e, n)
	}
}
