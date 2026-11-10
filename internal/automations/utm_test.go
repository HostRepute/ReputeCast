package automations_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/clicktoken"
	sendingengine "github.com/hostrepute/reputecast/internal/sending/engine"
)

// G-13: "Add UTM tags to links" reaches a flow when it is published and
// stays with that version. A version published with the setting on tags its
// emails' links even after the setting is turned off, and its stored links
// never change; the next publish, with the setting off, stores no tags, and
// the same email version then goes out untagged with links of its own.
func TestUTMTagsFrozenAtPublish(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	clk := &clock{t: time.Now()}
	svc := engine(env, clk)
	eng := env.App.SendingEngine()
	svc.Mailer = sendingengine.AutomationMailer{E: eng}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	setUTM := func(on bool) {
		t.Helper()
		if on {
			exec(`UPDATE workspaces SET sending_defaults = sending_defaults || '{"utm_links": true}' WHERE slug = 'acme-newsletter'`)
		} else {
			exec(`UPDATE workspaces SET sending_defaults = sending_defaults - 'utm_links' WHERE slug = 'acme-newsletter'`)
		}
	}
	email := uuid.Must(uuid.NewV7())
	html := `<html><body><p>Welcome aboard</p><p><a href="https://acme.io/start">Get started</a></p>` +
		`<p><a href="{{ unsubscribe_url }}">Unsubscribe</a> · Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105</p></body></html>`
	exec(`INSERT INTO automation_emails (id, workspace_id, name, latest_version) SELECT $1, id, 'UTM welcome', 1 FROM workspaces WHERE slug = 'acme-newsletter'`, email)
	exec(`INSERT INTO automation_email_versions (email_id, version, subject, html, plain_text) VALUES ($1, 1, 'Welcome', $2, 'Welcome aboard: https://acme.io/start')`, email, html)
	product := listID(t, env, "Product Updates")
	graph := func(subject string) automations.AutomationGraph {
		return automations.AutomationGraph{
			Nodes: []automations.AutomationNode{
				node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product})),
				node("m", automations.NodeEmail, automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: email.String(), EmailVersion: 1, Subject: subject}}),
			},
			Edges: []automations.AutomationEdge{link("t", "m", "")},
		}
	}
	utmOf := func(flow string, version int) *clicktoken.UTM {
		t.Helper()
		var raw []byte
		if err := env.App.Pool.QueryRow(ctx, `SELECT utm FROM automation_versions WHERE automation_id = $1 AND version = $2`, flow, version).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		if raw == nil {
			return nil
		}
		var u clicktoken.UTM
		if err := json.Unmarshal(raw, &u); err != nil {
			t.Fatal(err)
		}
		return &u
	}
	// One contact joins, its run queues the email and the outbox sends it.
	send := func(addr string) uuid.UUID {
		t.Helper()
		contact := addContact(t, env, addr, nil)
		joinList(t, env, product, contact)
		if _, err := svc.Dispatch(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Tick(ctx); err != nil {
			t.Fatal(err)
		}
		var msg uuid.UUID
		if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM messages WHERE contact_id = $1 AND stream = 'automation'`, contact).Scan(&msg); err != nil {
			t.Fatalf("the email to %s wasn't queued: %v", addr, err)
		}
		for i := range 10 {
			if _, err := eng.OutboxStep(ctx, clk.now().Add(time.Duration(i)*time.Second)); err != nil {
				t.Fatal(err)
			}
		}
		return msg
	}
	// The email version's stored link sets: tags (automation_email_links.utm) → address.
	links := func() map[string]string {
		t.Helper()
		rows, err := env.App.Pool.Query(ctx, `SELECT utm, url FROM automation_email_links WHERE email_id = $1 ORDER BY utm, position`, email)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[string]string{}
		for rows.Next() {
			var k, u string
			if err := rows.Scan(&k, &u); err != nil {
				t.Fatal(err)
			}
			out[k] = u
		}
		return out
	}
	tagsOf := func(msg uuid.UUID) string {
		t.Helper()
		var campaign *string
		if err := env.App.Pool.QueryRow(ctx, `SELECT content_ref->'utm'->>'campaign' FROM messages WHERE id = $1`, msg).Scan(&campaign); err != nil {
			t.Fatal(err)
		}
		if campaign == nil {
			return ""
		}
		return *campaign
	}

	setUTM(true)
	flow := publishFlow(t, c, "UTM welcome", graph("Welcome"))
	want := clicktoken.UTM{Source: "acme-newsletter", Medium: "email", Campaign: "utm-welcome"}
	if u := utmOf(flow, 1); u == nil || *u != want {
		t.Fatalf("version 1 stored %+v, want %+v", u, want)
	}
	// Turned off after the publish: version 1 still tags.
	setUTM(false)
	tagged := "https://acme.io/start?utm_source=acme-newsletter&utm_medium=email&utm_campaign=utm-welcome"
	if m := send("utm-1@example.com"); tagsOf(m) != "utm-welcome" {
		t.Fatalf("the email of version 1 was queued without its tags")
	}
	first := links()
	if len(first) != 1 || first[want.Key()] != tagged {
		t.Fatalf("version 1's links: %v", first)
	}
	if m := send("utm-2@example.com"); tagsOf(m) != "utm-welcome" {
		t.Fatalf("a later email of version 1 lost its tags")
	}
	if again := links(); len(again) != 1 || again[want.Key()] != tagged {
		t.Fatalf("version 1's links changed: %v", again)
	}

	// The next publish, with the setting off, stores no tags; the same email
	// version goes out untagged, and version 1's links stay as they were.
	var saved struct{ DraftVersion int }
	c.Do("PUT", wsPath+"/"+flow+"/draft", map[string]any{"graph": graph("Welcome again"), "revision": 0}).MustStatus(t, 200).JSON(t, &saved)
	c.Do("POST", wsPath+"/"+flow+"/publish", map[string]any{"version": saved.DraftVersion, "migration": "finish"}).MustStatus(t, 200)
	if u := utmOf(flow, saved.DraftVersion); u != nil {
		t.Fatalf("version %d stored %+v with the setting off", saved.DraftVersion, u)
	}
	if m := send("utm-3@example.com"); tagsOf(m) != "" {
		t.Fatalf("an email of version %d was queued with tags", saved.DraftVersion)
	}
	if last := links(); len(last) != 2 || last[""] != "https://acme.io/start" || last[want.Key()] != tagged {
		t.Fatalf("links after the next publish: %v", last)
	}
}
