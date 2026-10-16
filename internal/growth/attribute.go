package growth

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Attribution: a contact's source is how they first joined (confirmed onto
// a list for the first time), read from the first consent record of the
// signup: the form or page, the import, the API key or integration that
// called, or the person who added them. An import that carried a contact's
// original join date and source keeps both ("carried"); otherwise the
// import is the source and its day the join date. A departure is the first
// time after joining that the contact left every list, hard-bounced,
// complained or was erased.

const (
	attributeBatch   = 5000
	attributeOverlap = 2 * time.Minute
	// carriedSlack: a contact created more than a day before the import that
	// brought it kept its original date.
	carriedSlack = 24 * time.Hour
)

type candidate struct {
	id           uuid.UUID
	createdAt    time.Time
	source       string
	joinedAt     *time.Time
	action       *string
	evSource     *string
	evActor      *uuid.UUID
	evImport     *uuid.UUID
	evAt         *time.Time
	utm          [5]string
	page, refPg  string
	contactImpID *uuid.UUID
}

const candidatesSQL = `WITH c AS (
    SELECT c.id, c.created_at, c.source, c.import_id FROM contacts c
    WHERE c.id = ANY ($1) AND c.erased_at IS NULL
      AND NOT EXISTS (SELECT 1 FROM contact_growth g WHERE g.contact_id = c.id)
), joined AS (
    SELECT lm.contact_id, min(lm.subscribed_at) AS at FROM list_memberships lm
    WHERE lm.contact_id = ANY ($1) AND lm.subscribed_at IS NOT NULL AND lm.status IN ('subscribed', 'unsubscribed')
    GROUP BY 1
), first_ev AS (
    SELECT DISTINCT ON (ce.contact_id) ce.contact_id, ce.action, ce.source, ce.actor_id, ce.import_id, ce.occurred_at,
           ce.utm_source, ce.utm_medium, ce.utm_campaign, ce.utm_content, ce.utm_term, ce.page, ce.referrer
    FROM consent_events ce
    WHERE ce.contact_id = ANY ($1) AND ce.action IN ('subscribed', 'confirmed', 'resubscribed', 'imported')
    ORDER BY ce.contact_id, ce.occurred_at, ce.id
), touch AS (
    SELECT DISTINCT ON (ce.contact_id) ce.contact_id, ce.utm_source, ce.utm_medium, ce.utm_campaign, ce.utm_content, ce.utm_term, ce.page, ce.referrer
    FROM consent_events ce
    WHERE ce.contact_id = ANY ($1) AND ce.action IN ('subscribed', 'confirmed', 'resubscribed') AND (ce.page <> '' OR ce.utm_source <> '' OR ce.referrer <> '')
    ORDER BY ce.contact_id, ce.occurred_at, ce.id
)
SELECT c.id, c.created_at, c.source, c.import_id, j.at, f.action, f.source, f.actor_id, f.import_id, f.occurred_at,
       coalesce(t.utm_source, ''), coalesce(t.utm_medium, ''), coalesce(t.utm_campaign, ''), coalesce(t.utm_content, ''), coalesce(t.utm_term, ''),
       coalesce(t.page, ''), coalesce(t.referrer, '')
FROM c
LEFT JOIN joined j ON j.contact_id = c.id
LEFT JOIN first_ev f ON f.contact_id = c.id
LEFT JOIN touch t ON t.contact_id = c.id`

type attribution struct {
	contact   uuid.UUID
	kind, id  string
	joinedAt  time.Time
	utm       [5]string
	page, ref string
	captured  bool
}

// attributeContacts writes the growth rows of contacts that joined and
// have none yet.
func (s *Service) attributeContacts(ctx context.Context, tx pgx.Tx, ws *store.Workspace, n *names, captureFrom *time.Time, ids []uuid.UUID) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	rows, err := tx.Query(ctx, candidatesSQL, ids)
	if err != nil {
		return 0, err
	}
	var cs []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.createdAt, &c.source, &c.contactImpID, &c.joinedAt, &c.action, &c.evSource, &c.evActor, &c.evImport, &c.evAt,
			&c.utm[0], &c.utm[1], &c.utm[2], &c.utm[3], &c.utm[4], &c.page, &c.refPg); err != nil {
			rows.Close()
			return 0, err
		}
		cs = append(cs, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	// Who called for contacts that came through the API or were added in
	// ReputeCast: the integration's call log, else the audit log.
	var apiIDs []uuid.UUID
	var lo, hi time.Time
	for _, c := range cs {
		if c.evSource != nil && (*c.evSource == "api" || *c.evSource == "team") && c.evAt != nil {
			apiIDs = append(apiIDs, c.id)
			if lo.IsZero() || c.evAt.Before(lo) {
				lo = *c.evAt
			}
			if c.evAt.After(hi) {
				hi = *c.evAt
			}
		}
	}
	calls := map[uuid.UUID]string{}
	actors := map[uuid.UUID][2]string{}
	if len(apiIDs) > 0 {
		r, err := tx.Query(ctx, `SELECT DISTINCT ON (contact_id) contact_id, integration_id, workflow_key FROM integration_calls
			WHERE workspace_id = $1 AND contact_id = ANY ($2) ORDER BY contact_id, occurred_at`, ws.ID, apiIDs)
		if err != nil {
			return 0, err
		}
		for r.Next() {
			var c, i uuid.UUID
			var wf string
			if err := r.Scan(&c, &i, &wf); err != nil {
				r.Close()
				return 0, err
			}
			calls[c] = i.String() + ":" + wf
		}
		r.Close()
		texts := make([]string, len(apiIDs))
		for i, id := range apiIDs {
			texts[i] = id.String()
		}
		r, err = tx.Query(ctx, `SELECT DISTINCT ON (target_id) target_id, actor_type, coalesce(actor_id::text, '') FROM audit_events
			WHERE workspace_id = $1 AND target_type = 'contact' AND target_id = ANY ($2::text[]) AND action IN ('contact.create', 'contact.update')
			  AND occurred_at >= $3 AND occurred_at <= $4
			ORDER BY target_id, occurred_at`, ws.ID, texts, lo.Add(-time.Hour), hi.Add(time.Hour))
		if err != nil {
			return 0, err
		}
		for r.Next() {
			var target, typ, actor string
			if err := r.Scan(&target, &typ, &actor); err != nil {
				r.Close()
				return 0, err
			}
			if id, ok := parseUUID(target); ok {
				actors[id] = [2]string{typ, actor}
			}
		}
		r.Close()
	}
	var out []attribution
	for _, c := range cs {
		a, ok := s.attribute(c, n, calls, actors)
		if !ok {
			continue
		}
		a.captured = (a.kind == KindForm || a.kind == KindSubscribePage) &&
			(c.page != "" || c.utm[0] != "" || c.refPg != "" || (captureFrom != nil && !a.joinedAt.Before(*captureFrom)))
		out = append(out, a)
	}
	if len(out) == 0 {
		return 0, nil
	}
	return len(out), insertAttributions(ctx, tx, ws.ID, out)
}

// attribute decides one contact's source and join date.
func (s *Service) attribute(c candidate, n *names, calls map[uuid.UUID]string, actors map[uuid.UUID][2]string) (attribution, bool) {
	a := attribution{contact: c.id, utm: c.utm, page: c.page, ref: c.refPg}
	src := ""
	if c.evSource != nil {
		src = *c.evSource
	}
	switch {
	case c.action != nil && *c.action == "imported":
		imp := c.evImport
		if imp == nil {
			imp = c.contactImpID
		}
		importedAt := c.createdAt
		if c.evAt != nil {
			importedAt = *c.evAt
		}
		// The import kept an older date and the contact's own source.
		if c.createdAt.Before(importedAt.Add(-carriedSlack)) && c.source != "" && !strings.HasPrefix(src, c.source) {
			a.kind, a.id, a.joinedAt = KindCarried, c.source, c.createdAt
			return a, true
		}
		a.kind, a.joinedAt = KindImport, importedAt
		if imp != nil {
			a.id = imp.String()
		}
		return a, true
	case c.joinedAt == nil:
		return a, false // still waiting for the confirmation click
	}
	a.joinedAt = *c.joinedAt
	kind, detail, _ := strings.Cut(src, ":")
	switch {
	case kind == "form":
		a.kind = KindForm
		if f, ok := n.formsBySlug[detail]; ok {
			a.id = f.ID.String()
		} else {
			a.id = "slug:" + detail
		}
	case src == "subscribe_page":
		a.kind = KindSubscribePage
	case src == "preference_center":
		a.kind = KindPreferenceCenter
	case src == "api" || src == "team":
		if call, ok := calls[c.id]; ok {
			a.kind, a.id = KindIntegration, call
			break
		}
		act := actors[c.id]
		switch {
		case act[0] == "api_key" || act[0] == "mcp_agent":
			a.kind, a.id = KindAPIKey, act[1]
			if k, ok := n.keys[act[1]]; ok && k.IntegrationID != nil {
				a.kind, a.id = KindIntegration, k.IntegrationID.String()+":"
			}
		case src == "team":
			a.kind = KindPerson
		default:
			a.kind = KindAPIKey
		}
	case kind == "import":
		a.kind = KindImport
		if c.contactImpID != nil {
			a.id = c.contactImpID.String()
		}
	default:
		a.kind = KindUnknown
	}
	return a, true
}

func insertAttributions(ctx context.Context, tx pgx.Tx, wsID uuid.UUID, as []attribution) error {
	n := len(as)
	ids := make([]uuid.UUID, n)
	kinds, srcIDs := make([]string, n), make([]string, n)
	joined := make([]time.Time, n)
	utm := [5][]string{make([]string, n), make([]string, n), make([]string, n), make([]string, n), make([]string, n)}
	pages, refs, domains := make([]string, n), make([]string, n), make([]string, n)
	captured := make([]bool, n)
	for i, a := range as {
		ids[i], kinds[i], srcIDs[i], joined[i] = a.contact, a.kind, a.id, a.joinedAt
		for j := range utm {
			utm[j][i] = a.utm[j]
		}
		pages[i], refs[i], domains[i], captured[i] = a.page, a.ref, DomainOf(a.ref), a.captured
	}
	_, err := tx.Exec(ctx, `INSERT INTO contact_growth (contact_id, workspace_id, source_kind, source_id, joined_at,
		    utm_source, utm_medium, utm_campaign, utm_content, utm_term, page, referrer, referrer_domain, captured)
		SELECT u.id, $1, u.kind, u.sid, u.joined, u.s1, u.s2, u.s3, u.s4, u.s5, u.page, u.ref, u.dom, u.cap
		FROM unnest($2::uuid[], $3::text[], $4::text[], $5::timestamptz[], $6::text[], $7::text[], $8::text[], $9::text[], $10::text[],
		            $11::text[], $12::text[], $13::text[], $14::boolean[])
		     AS u(id, kind, sid, joined, s1, s2, s3, s4, s5, page, ref, dom, cap)
		ON CONFLICT (contact_id) DO NOTHING`,
		wsID, ids, kinds, srcIDs, joined, utm[0], utm[1], utm[2], utm[3], utm[4], pages, refs, domains, captured)
	return err
}

// departSQL records first departures in a window: $1 workspace, $2 from, $3 to.
var departSQL = []string{
	// Unsubscribed from every list.
	`UPDATE contact_growth g SET left_at = x.at, left_kind = 'unsubscribed', updated_at = now()
	FROM (SELECT ce.contact_id, max(ce.occurred_at) AS at FROM consent_events ce JOIN contacts c ON c.id = ce.contact_id
	      WHERE c.workspace_id = $1 AND ce.action = 'unsubscribed' AND ce.occurred_at > $2 AND ce.occurred_at <= $3 GROUP BY 1) x
	WHERE g.contact_id = x.contact_id AND g.left_at IS NULL AND x.at >= g.joined_at
	  AND NOT EXISTS (SELECT 1 FROM list_memberships lm WHERE lm.contact_id = x.contact_id AND lm.status IN ('subscribed', 'pending'))`,
	// Hard-bounced or complained.
	`UPDATE contact_growth g SET left_at = s.created_at, left_kind = CASE s.reason WHEN 'complaint' THEN 'complaint' ELSE 'hard_bounce' END, updated_at = now()
	FROM suppressions s JOIN contacts c ON c.workspace_id = s.workspace_id AND c.email = s.email
	WHERE s.workspace_id = $1 AND s.created_at > $2 AND s.created_at <= $3 AND s.reason IN ('hard_bounce', 'complaint')
	  AND g.contact_id = c.id AND g.left_at IS NULL AND s.created_at >= g.joined_at`,
	// Erased.
	`UPDATE contact_growth g SET left_at = c.erased_at, left_kind = 'erased', updated_at = now()
	FROM contacts c WHERE c.workspace_id = $1 AND c.erased_at > $2 AND c.erased_at <= $3 AND g.contact_id = c.id AND g.left_at IS NULL`,
}

// Attribute runs the attribution step for one workspace: the first time it
// attributes every existing contact; after that what joined or left since.
func (s *Service) Attribute(ctx context.Context, ws *store.Workspace) error {
	now := s.now()
	st, err := s.Q.GrowthStateEnsure(ctx, store.GrowthStateEnsureParams{WorkspaceID: ws.ID, At: now})
	if err != nil {
		return err
	}
	n, err := s.names(ctx, s.Q, ws)
	if err != nil {
		return err
	}
	to := now.Add(-2 * time.Second)
	if st.AttributedThrough == nil {
		// Everyone already here, in id order.
		after := uuid.Nil
		for {
			var ids []uuid.UUID
			err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
				r, err := tx.Query(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND erased_at IS NULL AND id > $2 ORDER BY id LIMIT $3`, ws.ID, after, attributeBatch)
				if err != nil {
					return err
				}
				if ids, err = pgx.CollectRows(r, pgx.RowTo[uuid.UUID]); err != nil || len(ids) == 0 {
					return err
				}
				_, err = s.attributeContacts(ctx, tx, ws, n, st.CaptureStartedAt, ids)
				return err
			})
			if err != nil {
				return err
			}
			if len(ids) < attributeBatch {
				break
			}
			after = ids[len(ids)-1]
		}
		if err := s.departures(ctx, ws.ID, time.Time{}, to); err != nil {
			return err
		}
		return s.Q.GrowthStateSetAttributed(ctx, store.GrowthStateSetAttributedParams{WorkspaceID: ws.ID, AttributedThrough: &to})
	}
	from := st.AttributedThrough.Add(-attributeOverlap)
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		r, err := tx.Query(ctx, `SELECT DISTINCT lm.contact_id FROM list_memberships lm JOIN lists l ON l.id = lm.list_id
			WHERE l.workspace_id = $1 AND lm.subscribed_at > $2 AND lm.subscribed_at <= $3
			  AND NOT EXISTS (SELECT 1 FROM contact_growth g WHERE g.contact_id = lm.contact_id)`, ws.ID, from, to)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(r, pgx.RowTo[uuid.UUID])
		if err != nil {
			return err
		}
		for start := 0; start < len(ids); start += attributeBatch {
			if _, err := s.attributeContacts(ctx, tx, ws, n, st.CaptureStartedAt, ids[start:min(start+attributeBatch, len(ids))]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := s.departures(ctx, ws.ID, from, to); err != nil {
		return err
	}
	return s.Q.GrowthStateSetAttributed(ctx, store.GrowthStateSetAttributedParams{WorkspaceID: ws.ID, AttributedThrough: &to})
}

func (s *Service) departures(ctx context.Context, wsID uuid.UUID, from, to time.Time) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		for _, sql := range departSQL {
			if _, err := tx.Exec(ctx, sql, wsID, from, to); err != nil {
				return err
			}
		}
		return nil
	})
}
