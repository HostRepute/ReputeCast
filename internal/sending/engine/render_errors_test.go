package engine_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
)

// testRenderErrors: one recipient's own data breaks the template (review
// CO-4): their message fails with the error, the send keeps going to the
// others and finishes, and one notice counts them with a masked sample,
// told once and audited. It's a notice, not a decision: nothing is held.
func testRenderErrors(t *testing.T, f *sendtest.Fixture) {
	// The digest with a line that prints the contact's signup source (a
	// custom field) five times; its footer stays, so Send Readiness passes
	// it.
	f.Exec(`UPDATE campaigns SET ab_test = NULL, html = replace(html, '</body>', $2 || '</body>') WHERE id = $1`, f.Campaign,
		`<p>{{ signup_source }}{{ signup_source }}{{ signup_source }}{{ signup_source }}{{ signup_source }}</p>`)
	f.AddContacts(sendtest.ContactSpec{N: 40, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	// One recipient's signup source is 1 MB: five of them are over the 4 MB
	// an email may render to. The template itself is fine for everyone else.
	f.Exec(`UPDATE contacts SET attributes = coalesce(attributes, '{}'::jsonb) || jsonb_build_object('signup_source', repeat('x', 1000000))
		WHERE id = (SELECT contact_id FROM list_memberships WHERE list_id = $1 ORDER BY contact_id LIMIT 1)`, f.List)
	f.StartSend(engine.StartOptions{})
	f.Drive(3 * time.Minute)
	if st := f.Status(); st != "sent" {
		f.Dump()
		t.Fatalf("status %s, want sent: the others keep getting it", st)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND status = 'failed' AND error LIKE '%personalization%'`, f.Campaign); n != 1 {
		f.Dump()
		t.Fatalf("%d failed for the template, want 1", n)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, f.Campaign); n != 39 {
		t.Fatalf("%d sent, want the 39 others", n)
	}
	var raw []byte
	var resolved bool
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT detail, resolved_at IS NOT NULL FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'render_error'`,
		f.Campaign).Scan(&raw, &resolved); err != nil {
		t.Fatalf("render notice: %v", err)
	}
	var d struct {
		Events int64  `json:"events"`
		Action string `json:"action"`
		Log    []struct {
			Text string `json:"text"`
		} `json:"log"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if d.Events != 1 || d.Action != "alerted" || len(d.Log) != 1 || !strings.Contains(d.Log[0].Text, "···@gmail.com") || !strings.Contains(d.Log[0].Text, "4 MB") {
		t.Fatalf("notice %s", raw)
	}
	if resolved {
		t.Fatal("the notice is open until someone acknowledges it")
	}
	if n := f.Count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'campaign.render_failed'`, f.Campaign.String()); n != 1 {
		t.Fatalf("%d audit entries, want 1", n)
	}
	snap, err := f.E.Snapshot(f.Ctx, f.WS, f.Campaign)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Incident != nil {
		t.Fatalf("a notice isn't a decision: %+v", snap.Incident)
	}
}
