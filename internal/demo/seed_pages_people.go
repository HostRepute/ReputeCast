package demo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/emailaddr"
)

// The people of the pages and products sample (publish-sell brief §4):
// buyers, the webinar's registrants and the Launch Kit waitlist. They are
// all sample contacts already, as the store sample's customers are: the
// Audience boards keep their 48,213 contacts and the three original lists
// their numbers. What they did on the pages is in their consent history,
// on the new lists and on the orders.

type sellContact struct {
	id          uuid.UUID
	email, name string
}

// sampleFree is a sample contact no other part of the demo tells a story
// about ($1: the workspace, $2: the addresses the boards name, $3: the
// contacts the buyers already are).
const sampleFree = `c.workspace_id = $1 AND c.status = 'active' AND c.erased_at IS NULL AND c.consent IN ('double', 'single')
	  AND NOT (c.email = ANY($2)) AND NOT (c.id = ANY($3))
	  AND NOT EXISTS (SELECT 1 FROM suppressions su WHERE su.workspace_id = c.workspace_id AND su.email = c.email)
	  AND NOT EXISTS (SELECT 1 FROM automation_runs x WHERE x.contact_id = c.id)
	  AND NOT EXISTS (SELECT 1 FROM paid_subscriptions x WHERE x.contact_id = c.id)
	  AND NOT EXISTS (SELECT 1 FROM earn_checkouts x WHERE x.contact_id = c.id)
	  AND NOT EXISTS (SELECT 1 FROM form_submissions x WHERE x.contact_id = c.id)
	  AND NOT EXISTS (SELECT 1 FROM poll_answers x WHERE x.contact_id = c.id)
	  AND NOT EXISTS (SELECT 1 FROM survey_responses x WHERE x.contact_id = c.id)
	  AND NOT EXISTS (SELECT 1 FROM unsubscribe_feedback x WHERE x.contact_id = c.id)
	  AND NOT EXISTS (SELECT 1 FROM website_members x WHERE x.contact_id = c.id)`

// named are the addresses the boards show with a story of their own.
func (w *sellSeed) named() []string {
	out := []string{"sam.rivera@outlook.com"}
	for _, b := range audBoardContacts {
		out = append(out, b.email)
	}
	return out
}

func (w *sellSeed) takenIDs() []uuid.UUID {
	out := make([]uuid.UUID, 0, len(w.taken))
	for id := range w.taken {
		out = append(out, id)
	}
	return out
}

// existingContacts are sample subscribers of the Weekly Digest who bought.
func (w *sellSeed) existingContacts(ctx context.Context, n int) ([]sellContact, error) {
	rows, err := w.s.Tx.Query(ctx, `SELECT c.id, c.email, trim(c.first_name || ' ' || c.last_name) FROM contacts c
		JOIN list_memberships m ON m.contact_id = c.id AND m.list_id = $4 AND m.status = 'subscribed'
		WHERE `+sampleFree+`
		ORDER BY md5(c.email || 'x19 buyers') LIMIT $5`, w.ws.ID, w.named(), w.takenIDs(), w.digest, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []sellContact
	for rows.Next() {
		var c sellContact
		if err := rows.Scan(&c.id, &c.email, &c.name); err != nil {
			return nil, err
		}
		w.taken[c.id] = true
		out = append(out, c)
	}
	return out, rows.Err()
}

// sampleAs is a buyer the boards name (Lee Wong, Ana Ruiz): a sample
// subscriber of the Weekly Digest renamed, whose mailbox provider already
// matches the new address, so no count moves.
func (w *sellSeed) sampleAs(ctx context.Context, email, name string) (uuid.UUID, error) {
	if id, ok := w.s.contactByEmail(ctx, email); ok {
		return id, nil
	}
	first, last, _ := strings.Cut(name, " ")
	var id uuid.UUID
	var old string
	err := w.s.Tx.QueryRow(ctx, `SELECT c.id, c.email FROM contacts c
		JOIN list_memberships m ON m.contact_id = c.id AND m.list_id = $4 AND m.status = 'subscribed'
		WHERE `+sampleFree+` AND c.mailbox_provider = $5
		ORDER BY (c.first_name = $6) DESC, md5(c.email) LIMIT 1`,
		w.ws.ID, w.named(), w.takenIDs(), w.digest, emailaddr.MailboxProvider(emailaddr.Domain(email)), first).Scan(&id, &old)
	if err != nil {
		return uuid.Nil, fmt.Errorf("sell seed: a sample contact for %s: %w", name, err)
	}
	if _, err := w.s.Tx.Exec(ctx, `UPDATE contacts SET email = $2, first_name = $3, last_name = $4 WHERE id = $1`, id, email, first, last); err != nil {
		return uuid.Nil, err
	}
	if _, err := w.s.Tx.Exec(ctx, `UPDATE messages SET email = $2 WHERE contact_id = $1 AND email = $3`, id, email, old); err != nil {
		return uuid.Nil, err
	}
	w.taken[id] = true
	return id, nil
}

// joinedAtCheckout is a buyer who ticked "Also send me the Weekly Digest"
// at checkout: the consent record of the tick and of the double opt-in
// they confirmed (Ana Ruiz two minutes later, at 10:14).
func (w *sellSeed) joinedAtCheckout(ctx context.Context, contact uuid.UUID, at time.Time, product string, ana bool) error {
	source, page := "product_checkout:deliverability-field-guide", "news.acme.io/field-guide"
	if product == "lk" {
		source, page = "product_checkout:newsletter-launch-kit", "news.acme.io/launch-kit"
	}
	confirmed := at.Add(time.Duration(2+w.rng.IntN(40)) * time.Minute)
	if ana {
		confirmed = at.Add(2 * time.Minute)
	}
	for _, ev := range []struct {
		action string
		at     time.Time
	}{{"subscribed", at}, {"confirmed", confirmed}} {
		if _, err := w.s.Tx.Exec(ctx, `INSERT INTO consent_events (id, contact_id, list_id, action, source, ip, note, consent_text, page, occurred_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, ID(), contact, w.digest, ev.action, source, addr("203.0.113.77"),
			"ticked “Also send me the Weekly Digest” at checkout", "Also send me the Weekly Digest", page, ev.at); err != nil {
			return err
		}
	}
	return nil
}

// signups: the webinar's 486 registrants on "DMARC live · 29 Oct" (single
// opt-in, the session's consent text, the date in dmarc_live_at) and the
// Launch Kit waitlist's 1,406 (double opt-in: 1,298 confirmed).
func (w *sellSeed) signups(ctx context.Context) error {
	s := w.s
	pick := func(n int, salt string) ([]uuid.UUID, error) {
		rows, err := s.Tx.Query(ctx, `SELECT c.id FROM contacts c WHERE `+sampleFree+` ORDER BY md5(c.email || $4) LIMIT $5`,
			w.ws.ID, w.named(), w.takenIDs(), salt, n)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				return nil, err
			}
			out = append(out, id)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		if len(out) < n {
			return nil, fmt.Errorf("sell seed: %d sample contacts for %s, need %d", len(out), salt, n)
		}
		return out, nil
	}
	spread := func(from, to time.Time) time.Time {
		return from.Add(time.Duration(w.rng.Int64N(int64(to.Sub(from)))))
	}
	var members, consents, convs [][]any
	// The webinar.
	registrants, err := pick(486, "x19 dmarc live")
	if err != nil {
		return err
	}
	date := w.event.Format("2006-01-02")
	const eventConsent = "Send me the joining link and reminders for this session."
	for _, id := range registrants {
		at := spread(w.day(10, 8, 10, 5), w.day(10, 15, 9, 0))
		members = append(members, []any{w.dmarc, id, "subscribed", "form:dmarc-live", &at, at})
		consents = append(consents, []any{ID(), id, w.dmarc, "subscribed", "form:dmarc-live", eventConsent, "news.acme.io/dmarc-live", at})
		pid := w.pages["dmarc-live"]
		cid := id
		convs = append(convs, []any{ID(), w.ws.ID, &pid, "page", "registration", &cid, nil, int64(0), at})
	}
	if _, err := s.Tx.Exec(ctx, `UPDATE contacts SET attributes = attributes || jsonb_build_object('dmarc_live_at', $2::text) WHERE id = ANY($1)`,
		registrants, date); err != nil {
		return err
	}
	// The waitlist: 1,406 joined before the countdown ended, 1,298 confirmed.
	waiting, err := pick(1406, "x19 launch kit waitlist")
	if err != nil {
		return err
	}
	const waitConsent = "Email me when the Newsletter Launch Kit opens."
	for i, id := range waiting {
		at := spread(w.day(9, 29, 9, 12), w.opens)
		status := "subscribed"
		var subscribed *time.Time
		if i < 1298 {
			c := at.Add(time.Duration(1+w.rng.IntN(90)) * time.Minute)
			subscribed = &c
			consents = append(consents, []any{ID(), id, w.waitlist, "confirmed", "form:launch-kit-waitlist", waitConsent, "news.acme.io/launch-kit", c})
		} else {
			status = "pending"
		}
		members = append(members, []any{w.waitlist, id, status, "form:launch-kit-waitlist", subscribed, at})
		consents = append(consents, []any{ID(), id, w.waitlist, "subscribed", "form:launch-kit-waitlist", waitConsent, "news.acme.io/launch-kit", at})
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"list_memberships"}, []string{"list_id", "contact_id", "status", "source", "subscribed_at", "created_at"},
		pgx.CopyFromRows(members)); err != nil {
		return fmt.Errorf("registrants and waitlist: %w", err)
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"consent_events"}, []string{"id", "contact_id", "list_id", "action", "source", "consent_text", "page", "occurred_at"},
		pgx.CopyFromRows(consents)); err != nil {
		return fmt.Errorf("registrants' consent: %w", err)
	}
	if _, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"page_conversions"}, []string{"id", "workspace_id", "page_id", "source", "kind", "contact_id", "order_id", "amount_cents", "occurred_at"},
		pgx.CopyFromRows(convs)); err != nil {
		return fmt.Errorf("registrations: %w", err)
	}
	return nil
}

// reminders is "DMARC live reminders", which Marcus Chen made from the
// webinar page and turned on: a date-field trigger on dmarc_live_at, the
// day before at the session's time, then an hour before.
func (w *sellSeed) reminders(ctx context.Context) error {
	s := w.s
	title := "Live: Fix your DMARC setup in 30 minutes"
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		ny = time.UTC
	}
	start := w.event.In(ny)
	link := "The joining link: https://acme.zoom.us/j/88412093371"
	when := w.event.Format("2 January 2006, 15:04 MST")
	created, published := w.day(10, 8, 10, 20), w.day(10, 8, 10, 31)
	type mail struct {
		id                             uuid.UUID
		name, subject, pre, header, l1 string
	}
	mails := []mail{
		{ID(), title + " · tomorrow", "Tomorrow: " + title, "Your joining link, and when it starts.", "See you tomorrow",
			"Hi {{ first_name }}, " + title + " starts " + when + "."},
		{ID(), title + " · in an hour", "Starting in an hour: " + title, "Join from the link below.", "Starting in an hour", title + " starts in an hour."},
	}
	flow := ID()
	graph := automations.AutomationGraph{
		Nodes: []autoFlowNode{
			autoNode("n1", automations.NodeTrigger, 370, 24, autoFlowData{Trigger: &automations.AutomationTrigger{Kind: automations.TriggerDateField, Field: "dmarc_live_at",
				OffsetDays: -1, At: start.Format("15:04"), Reentry: "always"}}),
			autoNode("n2", automations.NodeEmail, 370, 144, autoFlowData{Email: &automations.AutomationEmailStep{EmailID: mails[0].id.String(), EmailVersion: 1, Subject: mails[0].subject}}),
			autoNode("n3", automations.NodeDelay, 370, 284, autoFlowData{Delay: autoDur(23, "hours")}),
			autoNode("n4", automations.NodeEmail, 370, 364, autoFlowData{Email: &automations.AutomationEmailStep{EmailID: mails[1].id.String(), EmailVersion: 1, Subject: mails[1].subject}}),
			autoNode("n5", automations.NodeExit, 370, 504, autoFlowData{}),
		},
		Edges: []automations.AutomationEdge{autoEdge("n1", "n2", ""), autoEdge("n2", "n3", ""), autoEdge("n3", "n4", ""), autoEdge("n4", "n5", "")},
	}
	if err := autoInsertFlow(ctx, s, autoFlowSeed{id: flow, name: "DMARC live reminders", description: "Reminders for " + title + ": the day before and an hour before.",
		status: "live", live: 1, created: created, updated: published, createdBy: w.marcus,
		versions: []autoVersionSeed{{graph: graph, by: w.marcus, created: created, published: &published, summary: "created from a webinar page"}}}); err != nil {
		return err
	}
	for _, m := range mails {
		c := automations.DemoEmail(m.subject, m.pre, m.header, "", m.l1, link)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_emails (id, workspace_id, automation_id, name, latest_version, created_by, created_at, updated_at) VALUES ($1,$2,$3,$4,1,$5,$6,$6)`,
			m.id, w.ws.ID, flow, m.name, w.marcus, created); err != nil {
			return err
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_email_versions (email_id, version, subject, preheader, mjml, html, plain_text, created_by, created_at) VALUES ($1,1,$2,$3,$4,$5,$6,$7,$8)`,
			m.id, c.Subject, c.Preheader, c.MJML, c.HTML, c.Text, w.marcus, created); err != nil {
			return err
		}
	}
	return nil
}

// settle: what the sample's rows queued for automations happened long ago
// (nobody waits to enter a flow because of them).
func (w *sellSeed) settle(ctx context.Context) error {
	_, err := w.s.Tx.Exec(ctx, `UPDATE automation_events SET processed_at = occurred_at, enrolled = 0 WHERE workspace_id = $1 AND processed_at IS NULL`, w.ws.ID)
	return err
}
