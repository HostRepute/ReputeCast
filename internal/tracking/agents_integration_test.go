package tracking_test

import (
	"context"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
	"github.com/hostrepute/reputecast/internal/tracking"
	"github.com/hostrepute/reputecast/internal/useragent"
)

// TestUserAgentList: the user agents of machines are a list the instance
// keeps, as its networks are (review AN-2). Releases bring the built-in
// ones; instance admins add the machines they meet and the people's devices
// a rule takes for machines, and take off what they added. Every change is
// audited, only instance admins see or change the list, and the classifier
// uses it as soon as it changes.
func TestUserAgentList(t *testing.T) {
	s := sendtest.NewSuite(t)
	f := s.Fixture(t, "User agent list")
	ids := f.HandedOver(sendtest.ContactSpec{N: 4, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	tr, ok := f.E.Recorder.(*tracking.Service)
	if !ok {
		t.Fatalf("recorder %T", f.E.Recorder)
	}
	ctx := context.Background()
	arrived := f.Start.Add(time.Minute)
	deliver(t, f, arrived, ids...)

	// The migration's built-in list is the package's.
	rows, err := f.Env.App.Pool.Query(f.Ctx, `SELECT pattern, kind, name FROM tracking_user_agents WHERE source = 'builtin'`)
	if err != nil {
		t.Fatal(err)
	}
	var stored []useragent.Pattern
	for rows.Next() {
		var p useragent.Pattern
		if err := rows.Scan(&p.Text, &p.Kind, &p.Name); err != nil {
			t.Fatal(err)
		}
		stored = append(stored, p)
	}
	rows.Close()
	want := useragent.Builtin()
	byText := func(ps []useragent.Pattern) func(i, j int) bool {
		return func(i, j int) bool { return ps[i].Text < ps[j].Text }
	}
	sort.Slice(stored, byText(stored))
	sort.Slice(want, byText(want))
	if len(stored) != len(want) {
		t.Fatalf("%d built-in user agents stored, the package has %d", len(stored), len(want))
	}
	for i := range want {
		if stored[i] != want[i] {
			t.Fatalf("built-in user agent %+v stored as %+v", want[i], stored[i])
		}
	}

	admin := f.Env.Client()
	admin.SignInDemo("linh@acme.io")
	viewer := f.Env.Client()
	viewer.SignInDemo("aisha@acme.io")
	var list tracking.TrackingUserAgentList
	admin.Do("GET", "/admin/tracking-user-agents", nil).MustStatus(t, http.StatusOK).JSON(t, &list)
	if len(list.Items) != len(want) {
		t.Fatalf("the list has %d user agents, want the %d built-in ones", len(list.Items), len(want))
	}
	viewer.Do("GET", "/admin/tracking-user-agents", nil).MustStatus(t, http.StatusForbidden)
	viewer.Do("POST", "/admin/tracking-user-agents", map[string]any{"pattern": "x-scanner", "kind": "scanner", "name": "x"}).MustStatus(t, http.StatusForbidden)

	// What a machine's pattern may not be: in people's browsers, too
	// short; a person's is one word. And no pattern twice.
	admin.Do("POST", "/admin/tracking-user-agents", map[string]any{"pattern": "Chrome", "kind": "scanner", "name": "x"}).MustStatus(t, http.StatusUnprocessableEntity)
	admin.Do("POST", "/admin/tracking-user-agents", map[string]any{"pattern": "ab", "kind": "scanner", "name": "x"}).MustStatus(t, http.StatusUnprocessableEntity)
	admin.Do("POST", "/admin/tracking-user-agents", map[string]any{"pattern": "kiosk bot", "kind": "person", "name": "x"}).MustStatus(t, http.StatusUnprocessableEntity)
	admin.Do("POST", "/admin/tracking-user-agents", map[string]any{"pattern": "barracuda", "kind": "scanner", "name": "x"}).MustStatus(t, http.StatusConflict)

	click := func(id uuid.UUID, ua string, at time.Time) string {
		t.Helper()
		if err := tr.RecordEngagement(ctx, []sending.Engagement{{MessageID: id, Click: true, Link: 0, UserAgent: ua, IP: "198.51.100.40", Method: "GET", At: at}}); err != nil {
			t.Fatal(err)
		}
		if _, err := tr.SettleEngagement(ctx, at.Add(2*time.Minute)); err != nil {
			t.Fatal(err)
		}
		var class string
		if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT class FROM delivery_events WHERE message_id = $1 AND type = 'click' ORDER BY occurred_at DESC LIMIT 1`,
			id).Scan(&class); err != nil {
			t.Fatal(err)
		}
		return class
	}
	later := arrived.Add(30 * time.Minute)

	// A gateway's own link checker, unknown to the built-in list: a person
	// until an admin lists it, a machine right after.
	gateway := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AcmeGate-LinkCheck/3.1"
	if c := click(ids[0], gateway, later); c != tracking.ClassHuman {
		t.Fatalf("an unlisted gateway's click: %s", c)
	}
	var added tracking.TrackingUserAgent
	admin.Do("POST", "/admin/tracking-user-agents", map[string]any{"pattern": "AcmeGate-LinkCheck", "kind": "scanner", "name": "Our mail gateway",
		"note": "Follows every link in inbound mail"}).MustStatus(t, http.StatusCreated).JSON(t, &added)
	if added.Pattern != "acmegate-linkcheck" || added.Source != "admin" || added.CreatedByName == "" {
		t.Fatalf("added %+v", added)
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE action = 'tracking_user_agent.create' AND target_id = 'acmegate-linkcheck'`); n != 1 {
		t.Fatalf("%d audit entries for the new user agent", n)
	}
	if c := click(ids[1], gateway, later.Add(5*time.Minute)); c != tracking.ClassBot {
		t.Fatalf("a listed gateway's click: %s", c)
	}
	var reason string
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT diagnostic FROM delivery_events WHERE message_id = $1 AND type = 'click'`, ids[1]).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if reason != "Link scanner (Our mail gateway)" {
		t.Fatalf("the listed gateway's click is %q", reason)
	}

	// A person's device the "…bot" rule takes for a machine counts as a
	// person's once listed as one, and as a machine's again once taken off.
	kiosk := "Mozilla/5.0 (Linux; Android 13; KioskBot) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0 Mobile Safari/537.36"
	admin.Do("POST", "/admin/tracking-user-agents", map[string]any{"pattern": "KioskBot", "kind": "person", "name": "Lobby kiosks"}).MustStatus(t, http.StatusCreated)
	if c := click(ids[2], kiosk, later.Add(10*time.Minute)); c != tracking.ClassHuman {
		t.Fatalf("a listed person's device: %s", c)
	}
	admin.Do("DELETE", "/admin/tracking-user-agents?pattern=kioskbot", nil).MustStatus(t, http.StatusNoContent)
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE action = 'tracking_user_agent.delete' AND target_id = 'kioskbot'`); n != 1 {
		t.Fatalf("%d audit entries for the removed user agent", n)
	}
	if c := click(ids[3], kiosk, later.Add(15*time.Minute)); c != tracking.ClassBot {
		t.Fatalf("the device taken off the list: %s", c)
	}

	// Only what an admin added comes off the list.
	admin.Do("DELETE", "/admin/tracking-user-agents?pattern=barracuda", nil).MustStatus(t, http.StatusNotFound)
	viewer.Do("DELETE", "/admin/tracking-user-agents?pattern=acmegate-linkcheck", nil).MustStatus(t, http.StatusForbidden)
	admin.Do("GET", "/admin/tracking-user-agents", nil).MustStatus(t, http.StatusOK).JSON(t, &list)
	if len(list.Items) != len(want)+1 {
		t.Fatalf("the list has %d user agents, want the built-in ones and the gateway", len(list.Items))
	}
}
