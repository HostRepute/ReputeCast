package engine_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/sending/sendtest"
	"github.com/hostrepute/reputecast/internal/warmup"
)

// Warm-up in the send loop (review PR-8, C5): a warming provider sends no
// more than today's cap, per mailbox provider when its plan says so, most
// engaged first; past the cap the rest goes through a warm provider when
// the plan allows it, or waits as a hold with choices.
func TestWarmupInTheSendLoop(t *testing.T) {
	s := sendtest.NewSuite(t)
	// The overflow scenario runs first: it counts on Postal having sent
	// nothing yet today.
	t.Run("OverflowToWarmProvider", func(t *testing.T) { testWarmupOverflow(t, s.Fixture(t, "Warm-up overflow")) })
	t.Run("CapHoldsTheShare", func(t *testing.T) { testWarmupCapHold(t, s.Fixture(t, "Warm-up cap")) })
}

// warmupFixture sends the fixture's list through Postal alone (SES is the
// rule's fallback), without an A/B test, with today's cap left at 60.
func warmupFixture(t *testing.T, f *sendtest.Fixture, overflow, spread bool) (postal, ses uuid.UUID, plan string, sentBefore int64) {
	t.Helper()
	postal, ses = f.ProviderID("postal"), f.ProviderID("ses")
	f.Exec(`UPDATE campaigns SET ab_test = NULL WHERE id = $1`, f.Campaign)
	f.Exec(`INSERT INTO routing_rules (id, workspace_id, stream, priority, name, conditions, splits, fallback_provider_id)
		VALUES ($1, $2, 'broadcast', 1, '', $3, $4, $5)`, uuid.Must(uuid.NewV7()), f.WS,
		fmt.Sprintf(`{"lists":[%q]}`, f.List), fmt.Sprintf(`[{"provider_id":%q,"weight":100}]`, postal), ses)
	a, ok, err := warmup.AllowanceFor(f.Ctx, f.Env.App.Q, postal, f.Start, nil)
	if err != nil || !ok {
		t.Fatalf("Postal's warm-up: %v %v", ok, err)
	}
	f.Exec(`UPDATE warmup_plans SET held_cap = $2, overflow = $3, spread_mailbox = $4, engaged_first = true WHERE id = $1`, a.PlanID, a.Sent+60, overflow, spread)
	return postal, ses, a.PlanID.String(), a.Sent
}

func testWarmupOverflow(t *testing.T, f *sendtest.Fixture) {
	postal, ses, plan, before := warmupFixture(t, f, true, true)
	if before != 0 {
		t.Fatalf("Postal already sent %d today", before)
	}
	f.AddContacts(
		sendtest.ContactSpec{N: 50, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
		sendtest.ContactSpec{N: 30, Mailbox: "yahoo", Domain: "yahoo.com", Engaged: true},
	)
	f.StartSend(engine.StartOptions{})
	// The audience build keeps 60 on Postal (today's cap) and sends the
	// rest through SES.
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2`, f.Campaign, postal); n != 60 {
		f.Dump()
		t.Fatalf("%d on Postal after the audience build, want 60", n)
	}
	_, done := f.Drive(20 * time.Minute)
	if !done || f.Status() != "sent" {
		f.Dump()
		t.Fatalf("send didn't finish: %v %s", done, f.Status())
	}
	// Gmail's part of the cap is 40% (24), Yahoo's 15% (9): the rest of
	// Postal's share went through SES instead of waiting.
	sentOn := func(p uuid.UUID, mailbox string) int64 {
		return f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2 AND mailbox_provider = $3 AND sent_at IS NOT NULL`, f.Campaign, p, mailbox)
	}
	if g, y := sentOn(postal, "gmail"), sentOn(postal, "yahoo"); g != 24 || y != 9 {
		f.Dump()
		t.Fatalf("Postal sent %d Gmail and %d Yahoo, want 24 and 9", g, y)
	}
	if n := sentOn(ses, "gmail") + sentOn(ses, "yahoo"); n != 47 {
		t.Fatalf("SES sent %d, want 47", n)
	}
	if n := f.Count(`SELECT overflowed FROM warmup_plans WHERE id = $1`, plan); n != 27 {
		t.Fatalf("over the cap today: %d, want 27", n)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND metric = 'warmup_cap'`, f.Campaign); n != 0 {
		t.Fatal("an overflow isn't a hold")
	}
}

func testWarmupCapHold(t *testing.T, f *sendtest.Fixture) {
	postal, ses, plan, before := warmupFixture(t, f, false, false)
	f.AddContacts(
		sendtest.ContactSpec{N: 40, Mailbox: "gmail", Domain: "gmail.com", Tier: "cooling", Engaged: true},
		sendtest.ContactSpec{N: 50, Mailbox: "gmail", Domain: "gmail.com", Tier: "active", Engaged: true},
	)
	f.StartSend(engine.StartOptions{})
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2`, f.Campaign, postal); n != 90 {
		f.Dump()
		t.Fatalf("%d on Postal; without overflow the share stays there", n)
	}
	at, done := f.Drive(10 * time.Minute)
	if done {
		f.Dump()
		t.Fatal("the send finished past the cap")
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2 AND sent_at IS NOT NULL`, f.Campaign, postal); n != 60 {
		f.Dump()
		t.Fatalf("Postal sent %d, today's cap leaves 60 (%d sent before)", n, before)
	}
	// Most engaged first: every Active contact went out before Cooling.
	if n := f.Count(`SELECT count(*) FROM messages m JOIN contacts c ON c.id = m.contact_id WHERE m.campaign_id = $1 AND m.sent_at IS NOT NULL AND c.tier = 'active'`, f.Campaign); n != 50 {
		t.Fatalf("%d Active contacts sent of 60, want all 50 first", n)
	}
	if n := f.Count(`SELECT count(*) FROM campaign_routes WHERE campaign_id = $1 AND provider_id = $2 AND status = 'held' AND status_detail LIKE '%"reason":"warmup_cap"%'`, f.Campaign, postal); n != 1 {
		f.Dump()
		t.Fatal("Postal's share isn't held at the cap")
	}
	if n := f.Count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND kind = 'throttled' AND metric = 'warmup_cap' AND held = 30 AND resolved_at IS NULL`, f.Campaign); n != 1 {
		f.Dump()
		t.Fatal("no warm-up cap hold for 30 recipients")
	}
	if n := f.Count(`SELECT count(*) FROM notifications WHERE data->>'reason' = 'warmup_cap' AND data->>'campaignId' = $1`, f.Campaign.String()); n == 0 {
		t.Fatal("nobody was told about the hold")
	}
	if s := f.Status(); s != "sending" {
		t.Fatalf("campaign %s; a warm-up hold keeps it sending", s)
	}

	// Moving the held share sends it through SES.
	owner := f.Env.Client()
	owner.SignInDemo("linh@acme.io")
	var res warmup.WarmupHoldDecisionResult
	owner.Do("POST", "/workspaces/acme-newsletter/warmup/"+plan+"/held/"+f.Campaign.String(), map[string]any{"action": "move", "targetProviderId": ses.String()}).
		MustStatus(t, 200).JSON(t, &res)
	if res.Moved != 30 {
		t.Fatalf("moved %d", res.Moved)
	}
	if _, done, err := f.E.Drive(f.Ctx, f.Campaign, at, 10*time.Minute); err != nil || !done || f.Status() != "sent" {
		f.Dump()
		t.Fatalf("after the move: done=%v status=%s err=%v", done, f.Status(), err)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2 AND sent_at IS NOT NULL`, f.Campaign, ses); n != 30 {
		t.Fatalf("SES sent %d of the held 30", n)
	}
}

// Warm-up for Automations and Transactional (review R3.L6b, PR-8).
// Automation mail counts toward a warming provider's cap, per mailbox
// provider when the plan caps them separately, and respects a paused plan:
// what the cap or the pause stops stays queued (never dropped) until
// tomorrow's cap or the resume, unless the plan sends what's over the cap
// through the warm Automations fallback. Transactional mail never waits: it
// counts toward the cap, goes through a warm provider of its route when the
// cap or a pause stops it, and with none still goes out, counted as over
// today's cap for the warm-up page and the provider card.
func TestWarmupOtherStreams(t *testing.T) {
	s := sendtest.NewSuite(t)
	t.Run("AutomationsWaitAtTheCap", func(t *testing.T) { testAutomationsWaitAtTheCap(t, s.Fixture(t, "Warm-up automations cap")) })
	t.Run("AutomationsWaitWhilePaused", func(t *testing.T) { testAutomationsWaitWhilePaused(t, s.Fixture(t, "Warm-up automations paused")) })
	t.Run("AutomationsCapPerMailbox", func(t *testing.T) { testAutomationsCapPerMailbox(t, s.Fixture(t, "Warm-up automations mailbox")) })
	t.Run("AutomationsOverflowToWarmFallback", func(t *testing.T) { testAutomationsOverflow(t, s.Fixture(t, "Warm-up automations overflow")) })
	t.Run("AutomationsOverflowPerMailbox", func(t *testing.T) {
		testAutomationsOverflowPerMailbox(t, s.Fixture(t, "Warm-up automations mailbox overflow"))
	})
	t.Run("TransactionalNeverWaits", func(t *testing.T) { testTransactionalNeverWaits(t, s.Fixture(t, "Warm-up transactional")) })
}

// postmarkTakesAutomations verifies the Automations sender at Postmark, the
// demo's Automations fallback, so it passes its checks for overflow.
func postmarkTakesAutomations(t *testing.T, f *sendtest.Fixture) uuid.UUID {
	t.Helper()
	postmark := f.ProviderID("postmark")
	f.Exec(`UPDATE senders SET verification = coalesce(verification, '{}'::jsonb) || jsonb_build_object($2::text, 'verified')
		WHERE id = (SELECT coalesce(default_automation_sender_id, default_sender_id) FROM workspaces WHERE id = $1)`, f.WS, postmark.String())
	return postmark
}

// postalCapLeft leaves Postal's warm-up n messages for the rest of today
// (its held cap), with or without caps per mailbox provider and overflow.
// The scenarios share the demo workspace: each starts with no automation
// or transactional mail queued, so only its own mail meets the cap.
func postalCapLeft(t *testing.T, f *sendtest.Fixture, n int64, spread, overflow bool) (uuid.UUID, warmup.Allowance) {
	t.Helper()
	f.Exec(`UPDATE messages SET status = 'cancelled', claimed_at = NULL
		WHERE workspace_id = $1 AND status = 'queued' AND campaign_id IS NULL AND stream IN ('automation', 'transactional')`, f.WS)
	postal := f.ProviderID("postal")
	a, ok, err := warmup.AllowanceFor(f.Ctx, f.Env.App.Q, postal, f.Start, nil)
	if err != nil || !ok {
		t.Fatalf("Postal's warm-up: %v %v", ok, err)
	}
	f.Exec(`UPDATE warmup_plans SET held_cap = $2, spread_mailbox = $3, overflow = $4, status = 'active', paused_at = NULL WHERE id = $1`, a.PlanID, a.Sent+n, spread, overflow)
	a, _, err = warmup.AllowanceFor(f.Ctx, f.Env.App.Q, postal, f.Start, nil)
	if err != nil {
		t.Fatal(err)
	}
	return postal, a
}

// automationMail queues an automation email to each of the fixture's
// contacts at a mailbox provider, through provider (the step's own route).
func automationMail(t *testing.T, f *sendtest.Fixture, provider uuid.UUID, mailbox string) []uuid.UUID {
	t.Helper()
	email := uuid.Must(uuid.NewV7())
	html := `<html><body><p>Welcome aboard</p><p><a href="{{ unsubscribe_url }}">Unsubscribe</a> · Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105</p></body></html>`
	f.Exec(`INSERT INTO automation_emails (id, workspace_id, name, latest_version) VALUES ($1, $2, 'Warm-up welcome', 1)`, email, f.WS)
	f.Exec(`INSERT INTO automation_email_versions (email_id, version, subject, html, plain_text) VALUES ($1, 1, 'Welcome', $2, 'Welcome aboard')`, email, html)
	var flow uuid.UUID
	if err := f.Env.App.Pool.QueryRow(f.Ctx, `SELECT id FROM automations WHERE workspace_id = $1 ORDER BY created_at LIMIT 1`, f.WS).Scan(&flow); err != nil {
		t.Fatal(err)
	}
	rows, err := f.Env.App.Pool.Query(f.Ctx, `SELECT c.id, c.email FROM contacts c JOIN list_memberships m ON m.contact_id = c.id
		WHERE m.list_id = $1 AND c.mailbox_provider = $2 ORDER BY c.email`, f.List, mailbox)
	if err != nil {
		t.Fatal(err)
	}
	type contact struct {
		id    uuid.UUID
		email string
	}
	var contacts []contact
	for rows.Next() {
		var c contact
		if err := rows.Scan(&c.id, &c.email); err != nil {
			t.Fatal(err)
		}
		contacts = append(contacts, c)
	}
	rows.Close()
	ids := make([]uuid.UUID, 0, len(contacts))
	m := engine.AutomationMailer{E: f.E}
	for _, c := range contacts {
		tx, err := f.Env.App.Pool.Begin(f.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		out, err := m.QueueAutomationEmail(f.Ctx, tx, automations.OutgoingEmail{WorkspaceID: f.WS, AutomationID: flow, NodeID: "warm", ContactID: c.id,
			To: c.email, EmailID: email, EmailVersion: 1, Subject: "Welcome", Route: provider.String()})
		if err != nil {
			_ = tx.Rollback(f.Ctx)
			t.Fatal(err)
		}
		if err := tx.Commit(f.Ctx); err != nil {
			t.Fatal(err)
		}
		if out.ProviderID != provider {
			t.Fatalf("queued on %s, want %s", out.ProviderID, provider)
		}
		ids = append(ids, out.MessageID)
	}
	return ids
}

// outbox runs the automation outbox for n seconds from at.
func outbox(t *testing.T, f *sendtest.Fixture, at time.Time, n int) {
	t.Helper()
	for i := range n {
		if _, err := f.E.OutboxStep(f.Ctx, at.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
}

// statuses counts messages by status and provider ("queued@postal").
func statuses(t *testing.T, f *sendtest.Fixture, ids []uuid.UUID) map[string]int {
	t.Helper()
	names := map[uuid.UUID]string{f.ProviderID("postal"): "postal", f.ProviderID("ses"): "ses", f.ProviderID("postmark"): "postmark"}
	rows, err := f.Env.App.Pool.Query(f.Ctx, `SELECT CASE WHEN sent_at IS NULL THEN status ELSE 'sent' END, provider_id FROM messages WHERE id = ANY($1)`, ids)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var st string
		var p *uuid.UUID
		if err := rows.Scan(&st, &p); err != nil {
			t.Fatal(err)
		}
		where := "none"
		if p != nil {
			where = names[*p]
		}
		out[st+"@"+where]++
	}
	return out
}

func testAutomationsWaitAtTheCap(t *testing.T, f *sendtest.Fixture) {
	postal, _ := postalCapLeft(t, f, 2, false, false)
	f.AddContacts(sendtest.ContactSpec{N: 5, Mailbox: "gmail", Domain: "gmail.com", Engaged: true})
	ids := automationMail(t, f, postal, "gmail")
	outbox(t, f, f.Start, 8)
	// Two go out within today's cap; the others wait on Postal, queued.
	if got := statuses(t, f, ids); got["sent@postal"] != 2 || got["queued@postal"] != 3 {
		t.Fatalf("at the cap: %v", got)
	}
	a, _, err := warmup.AllowanceFor(f.Ctx, f.Env.App.Q, postal, f.Start, nil)
	if err != nil || a.Left() != 0 {
		t.Fatalf("automation mail didn't count toward the cap: left %d (%v)", a.Left(), err)
	}
	// Tomorrow, the hold over, the plan's cap sends the rest (a held cap
	// would stay the cap until the hold ends).
	f.Exec(`UPDATE warmup_plans SET held_cap = NULL WHERE id = $1`, a.PlanID)
	outbox(t, f, a.NextDay.Add(9*time.Hour), 8)
	if got := statuses(t, f, ids); got["sent@postal"] != 5 {
		t.Fatalf("the next day: %v", got)
	}
}

func testAutomationsWaitWhilePaused(t *testing.T, f *sendtest.Fixture) {
	postal, a := postalCapLeft(t, f, 50, false, false)
	f.Exec(`UPDATE warmup_plans SET status = 'paused', paused_at = $2 WHERE id = $1`, a.PlanID, f.Start)
	f.AddContacts(sendtest.ContactSpec{N: 3, Mailbox: "outlook", Domain: "outlook.com", Engaged: true})
	ids := automationMail(t, f, postal, "outlook")
	outbox(t, f, f.Start, 5)
	// A paused plan sends nothing, though today's cap has room: without
	// overflow the mail waits.
	if got := statuses(t, f, ids); got["queued@postal"] != 3 {
		t.Fatalf("while paused: %v", got)
	}
	f.Exec(`UPDATE warmup_plans SET status = 'active', paused_at = NULL WHERE id = $1`, a.PlanID)
	outbox(t, f, f.Start.Add(time.Minute), 5)
	if got := statuses(t, f, ids); got["sent@postal"] != 3 {
		t.Fatalf("after the resume: %v", got)
	}
}

func testAutomationsCapPerMailbox(t *testing.T, f *sendtest.Fixture) {
	postal, a := postalCapLeft(t, f, 100, true, false)
	gmail, yahoo := a.MailboxLeft("gmail"), a.MailboxLeft("yahoo")
	if gmail <= 0 || yahoo <= 0 || gmail > 40 {
		t.Fatalf("today's parts: gmail %d, yahoo %d", gmail, yahoo)
	}
	f.AddContacts(
		sendtest.ContactSpec{N: int(gmail) + 3, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
		sendtest.ContactSpec{N: 1, Mailbox: "yahoo", Domain: "yahoo.com", Engaged: true},
	)
	g := automationMail(t, f, postal, "gmail")
	y := automationMail(t, f, postal, "yahoo")
	outbox(t, f, f.Start, int(gmail)+10)
	// Gmail's part fills up and its mail waits; Yahoo's goes out all the
	// same: a full mailbox provider never blocks the others.
	if got := statuses(t, f, g); got["sent@postal"] != int(gmail) || got["queued@postal"] != 3 {
		t.Fatalf("Gmail (part %d): %v", gmail, got)
	}
	if got := statuses(t, f, y); got["sent@postal"] != 1 {
		t.Fatalf("Yahoo: %v", got)
	}
}

// testAutomationsOverflowPerMailbox: over a mailbox provider's part of the
// cap, with overflow on, only that mailbox provider's mail goes through the
// warm fallback; the others keep sending through Postal.
func testAutomationsOverflowPerMailbox(t *testing.T, f *sendtest.Fixture) {
	postal, a := postalCapLeft(t, f, 100, true, true)
	postmarkTakesAutomations(t, f)
	gmail := a.MailboxLeft("gmail")
	if gmail <= 0 || gmail > 40 || a.MailboxLeft("yahoo") <= 0 {
		t.Fatalf("today's parts: gmail %d, yahoo %d", gmail, a.MailboxLeft("yahoo"))
	}
	f.AddContacts(
		sendtest.ContactSpec{N: int(gmail) + 3, Mailbox: "gmail", Domain: "gmail.com", Engaged: true},
		sendtest.ContactSpec{N: 1, Mailbox: "yahoo", Domain: "yahoo.com", Engaged: true},
	)
	g := automationMail(t, f, postal, "gmail")
	y := automationMail(t, f, postal, "yahoo")
	outbox(t, f, f.Start, int(gmail)+10)
	if got := statuses(t, f, g); got["sent@postal"] != int(gmail) || got["sent@postmark"] != 3 {
		t.Fatalf("Gmail (part %d) with overflow: %v", gmail, got)
	}
	if got := statuses(t, f, y); got["sent@postal"] != 1 {
		t.Fatalf("Yahoo stays on Postal: %v", got)
	}
}

func testAutomationsOverflow(t *testing.T, f *sendtest.Fixture) {
	postal, a := postalCapLeft(t, f, 0, false, true)
	// The Automations fallback (Postmark) passes its checks: the stream's
	// sender is verified there.
	postmarkTakesAutomations(t, f)
	before := f.Count(`SELECT CASE WHEN overflowed_on = $2::date THEN overflowed ELSE 0 END FROM warmup_plans WHERE id = $1`, a.PlanID, a.NextDay.AddDate(0, 0, -1).Format("2006-01-02"))
	f.AddContacts(sendtest.ContactSpec{N: 3, Mailbox: "icloud", Domain: "icloud.com", Engaged: true})
	ids := automationMail(t, f, postal, "icloud")
	outbox(t, f, f.Start, 6)
	// Over the cap, with overflow on: the mail goes through Postmark, and
	// counts as over the cap today.
	if got := statuses(t, f, ids); got["sent@postmark"] != 3 {
		t.Fatalf("overflow: %v", got)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE id = ANY($1) AND route_note = 'Postmark instead of Postal: over today''s warm-up cap'`, ids); n != 3 {
		t.Fatalf("%d of 3 say why they went through Postmark", n)
	}
	if n := f.Count(`SELECT overflowed FROM warmup_plans WHERE id = $1`, a.PlanID); n != before+3 {
		t.Fatalf("over the cap today: %d, want %d", n, before+3)
	}
	// A paused plan with overflow on sends its share through Postmark too:
	// the pause stops Postal, not the automation.
	f.Exec(`UPDATE warmup_plans SET status = 'paused', paused_at = $2, held_cap = NULL WHERE id = $1`, a.PlanID, f.Start)
	f.AddContacts(sendtest.ContactSpec{N: 2, Mailbox: "other", Domain: "example.net", Engaged: true})
	paused := automationMail(t, f, postal, "other")
	outbox(t, f, f.Start.Add(time.Minute), 4)
	if got := statuses(t, f, paused); got["sent@postmark"] != 2 {
		t.Fatalf("overflow while paused: %v", got)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE id = ANY($1) AND route_note = 'Postmark instead of Postal: warm-up paused'`, paused); n != 2 {
		t.Fatalf("%d of 2 say the warm-up is paused", n)
	}
}

// transactionalReady routes the demo's transactional mail (Acme Cloud's,
// from no-reply@notify.acme.io, password-reset published) through primary,
// with fallback when there is one, and returns a send-only key.
func transactionalReady(t *testing.T, f *sendtest.Fixture, primary uuid.UUID, fallback *uuid.UUID) *apptest.Client {
	t.Helper()
	transactionalRoute(t, f, primary, fallback, "automatic")
	owner := f.Env.Client()
	owner.SignInDemo("linh@acme.io")
	var created struct{ Secret string }
	owner.Do("POST", "/workspaces/acme-newsletter/api-keys", map[string]any{"name": "Warm-up test", "permissions": []string{"transactional:send"},
		"transactional": map[string]any{"preset": "send_only"}}).MustStatus(t, 201).JSON(t, &created)
	key := f.Env.Client()
	key.Bearer = created.Secret
	return key
}

// transactionalRoute sends the Transactional stream to primary, then
// fallback, with a failover policy.
func transactionalRoute(t *testing.T, f *sendtest.Fixture, primary uuid.UUID, fallback *uuid.UUID, policy string) {
	t.Helper()
	f.Exec(`DELETE FROM routing_rules WHERE workspace_id = $1 AND stream = 'transactional'`, f.WS)
	f.Exec(`INSERT INTO routing_rules (id, workspace_id, stream, priority, name, conditions, splits, fallback_provider_id)
		VALUES ($1, $2, 'transactional', 100, '', '{}', $3, $4)`, uuid.Must(uuid.NewV7()), f.WS, fmt.Sprintf(`[{"provider_id":%q,"weight":100}]`, primary), fallback)
	f.Exec(`INSERT INTO stream_policies (workspace_id, stream, failover) VALUES ($1, 'transactional', $2)
		ON CONFLICT (workspace_id, stream) DO UPDATE SET failover = EXCLUDED.failover`, f.WS, policy)
}

// passwordResets sends n password resets and returns their message IDs.
func passwordResets(t *testing.T, key *apptest.Client, tag string, n int) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, 0, n)
	for i := range n {
		key.Header = http.Header{"Idempotency-Key": []string{fmt.Sprintf("warm-%s-%d", tag, i)}}
		var res struct{ ID, Status string }
		key.Do("POST", "/workspaces/acme-newsletter/transactional/messages", map[string]any{"template": "password-reset",
			"to":   map[string]any{"email": fmt.Sprintf("sam.%s.%d@outlook.com", tag, i), "name": "Sam Rivera"},
			"data": map[string]any{"first_name": "Sam", "reset_url": "https://app.acme.io/reset/tok_" + tag, "expires_in": "30 minutes"}}).
			MustStatus(t, 202).JSON(t, &res)
		if res.Status != "queued" {
			t.Fatalf("send: %+v", res)
		}
		ids = append(ids, uuid.MustParse(res.ID))
	}
	return ids
}

// transactionalOutbox runs the transactional outbox for n seconds from at.
// Eight seconds cover a retry: the demo transport throttles about one call
// in 250 at Postal, and the route waits 5 s before it tries again.
func transactionalOutbox(t *testing.T, f *sendtest.Fixture, at time.Time, n int) {
	t.Helper()
	for i := range n {
		if _, err := f.Env.App.TransactionalOutbox().Step(f.Ctx, at.Add(time.Duration(i)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
}

func testTransactionalNeverWaits(t *testing.T, f *sendtest.Fixture) {
	postal, a := postalCapLeft(t, f, 1, false, false)
	ses := f.ProviderID("ses")
	key := transactionalReady(t, f, postal, nil)
	overCap := func() int64 {
		t.Helper()
		return f.Count(`SELECT over_cap FROM warmup_plans WHERE id = $1`, a.PlanID)
	}

	// No warm provider in the route: one goes out within the cap, the
	// other two over it, through Postal all the same, counted.
	resets := passwordResets(t, key, "alone", 3)
	transactionalOutbox(t, f, f.Start, 8)
	if got := statuses(t, f, resets); got["sent@postal"] != 3 {
		t.Fatalf("password resets waited for the warm-up: %v", got)
	}
	if n := overCap(); n != 2 {
		t.Fatalf("over the cap today: %d, want 2", n)
	}
	if now, _, _ := warmup.AllowanceFor(f.Ctx, f.Env.App.Q, postal, f.Start, nil); now.Sent != a.Sent+3 || now.Left() != 0 {
		t.Fatalf("transactional mail didn't count toward the cap: sent %d (from %d), left %d", now.Sent, a.Sent, now.Left())
	}
	// The warm-up page and the provider card say so.
	owner := f.Env.Client()
	owner.SignInDemo("linh@acme.io")
	var o warmup.WarmupOverview
	owner.Do("GET", "/workspaces/acme-newsletter/warmup", nil).MustStatus(t, 200).JSON(t, &o)
	shown := false
	for _, p := range o.Plans {
		if p.ProviderID == postal.String() && p.TransactionalOverCap == 2 {
			shown = true
		}
	}
	if !shown {
		t.Fatalf("warm-up page: %+v", o.Plans)
	}
	var cards struct {
		Providers []struct {
			ID     string
			Warmup *struct{ TransactionalOverCap int64 }
		}
	}
	owner.Do("GET", "/workspaces/acme-newsletter/providers", nil).MustStatus(t, 200).JSON(t, &cards)
	found := false
	for _, c := range cards.Providers {
		if c.ID == postal.String() && c.Warmup != nil && c.Warmup.TransactionalOverCap == 2 {
			found = true
		}
	}
	if !found {
		t.Fatalf("provider cards: %+v", cards.Providers)
	}

	// With a warm fallback, what the cap stops goes through it instead, and
	// says why; nothing more goes over the cap.
	transactionalRoute(t, f, postal, &ses, "automatic")
	moved := passwordResets(t, key, "fallback", 2)
	transactionalOutbox(t, f, f.Start.Add(time.Minute), 8)
	if got := statuses(t, f, moved); got["sent@ses"] != 2 {
		t.Fatalf("over the cap with a warm fallback: %v", got)
	}
	if n := f.Count(`SELECT count(*) FROM messages WHERE id = ANY($1) AND route_note = 'SES instead of Postal: over today''s warm-up cap'`, moved); n != 2 {
		t.Fatalf("%d of 2 say why they went through SES", n)
	}
	if n := f.Count(`SELECT count(*) FROM transactional_messages WHERE message_id = ANY($1) AND failed_over_at IS NOT NULL`, moved); n != 0 {
		t.Fatal("a warm-up move isn't a failover")
	}
	if n := overCap(); n != 2 {
		t.Fatalf("over the cap today: %d, still want 2", n)
	}

	// A paused plan holds nothing back either: with "pause and ask" the
	// message doesn't move by itself, so it goes out through Postal.
	f.Exec(`UPDATE warmup_plans SET status = 'paused', paused_at = $2, held_cap = NULL WHERE id = $1`, a.PlanID, f.Start)
	transactionalRoute(t, f, postal, &ses, "pause_and_ask")
	paused := passwordResets(t, key, "paused", 1)
	transactionalOutbox(t, f, f.Start.Add(2*time.Minute), 8)
	if got := statuses(t, f, paused); got["sent@postal"] != 1 {
		t.Fatalf("while the plan is paused: %v", got)
	}
	if n := overCap(); n != 3 {
		t.Fatalf("over the cap today: %d, want 3", n)
	}
}
