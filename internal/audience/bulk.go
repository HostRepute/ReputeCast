package audience

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
)

// MaxBulkIDs bounds an explicit selection; larger actions use a filter.
const MaxBulkIDs = 10000

// Selection is either explicit contact ids or every contact matching a filter.
type Selection struct {
	IDs []string `json:"ids,omitempty" maxItems:"10000"`
	// AllMatching applies the action to every contact matching Filter.
	AllMatching bool          `json:"allMatching,omitempty"`
	Filter      ContactFilter `json:"filter,omitempty"`
}

type ContactBulkInput struct {
	Selection
	Action string `json:"action" enum:"add_to_list,add_tag,resend_optin,suppress"`
	ListID string `json:"listId,omitempty" doc:"For add_to_list"`
	Tag    string `json:"tag,omitempty" maxLength:"60" doc:"For add_tag"`
	Reason string `json:"reason,omitempty" maxLength:"300" doc:"For suppress"`
}

// ContactBulkFailure is one contact the action couldn't apply to.
type ContactBulkFailure struct {
	Email  string `json:"email"`
	Reason string `json:"reason"`
}

// ContactBulkResult reports a bulk action ("Added 2,116 of 2,118 contacts to
// 'Re-permission'; 2 failed").
type ContactBulkResult struct {
	Action   string               `json:"action"`
	Matched  int64                `json:"matched"`
	Done     int64                `json:"done"`
	Failed   int64                `json:"failed"`
	Failures []ContactBulkFailure `json:"failures" doc:"Up to 20 examples"`
	// Target names the list or tag ("Re-permission").
	Target string `json:"target,omitempty"`
	Note   string `json:"note,omitempty"`
}

// selectIDs resolves a selection to contact ids visible to the caller.
func (s *Service) selectIDs(ctx context.Context, ws *store.Workspace, sel Selection) ([]uuid.UUID, error) {
	fs, err := s.filterSQL(ctx, ws.ID, sel.Filter)
	if err != nil {
		return nil, err
	}
	if !sel.AllMatching {
		if len(sel.IDs) == 0 {
			return nil, api.Invalid("body.ids", "Select at least one contact.")
		}
		ids := make([]uuid.UUID, 0, len(sel.IDs))
		for _, raw := range sel.IDs {
			id, err := uuid.Parse(raw)
			if err != nil {
				return nil, api.Invalid("body.ids", "Some selected contacts aren't valid ids.")
			}
			ids = append(ids, id)
		}
		fs = filterSQL{where: "c.workspace_id = $1 AND c.erased_at IS NULL", args: []any{ws.ID}}
		fs.add("c.id = ANY (?)", ids)
		if p := api.PrincipalFrom(ctx); p != nil && len(p.Scope.Lists) > 0 {
			fs.add("EXISTS (SELECT 1 FROM list_memberships lm WHERE lm.contact_id = c.id AND lm.list_id = ANY (?))", p.Scope.Lists)
		}
	}
	rows, err := s.Pool.Query(ctx, "SELECT c.id FROM contacts c WHERE "+fs.where, fs.args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
}

// Bulk applies an action to a selection of contacts.
func (s *Service) Bulk(ctx context.Context, ws *store.Workspace, in ContactBulkInput) (ContactBulkResult, error) {
	out := ContactBulkResult{Action: in.Action, Failures: []ContactBulkFailure{}}
	ids, err := s.selectIDs(ctx, ws, in.Selection)
	if err != nil {
		return out, err
	}
	out.Matched = int64(len(ids))
	actor := actorID(ctx)
	switch in.Action {
	case "add_to_list":
		listID, perr := uuid.Parse(in.ListID)
		if perr != nil {
			return out, api.Invalid("body.listId", "Pick a list.")
		}
		l, lerr := s.Q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: listID})
		if lerr != nil {
			return out, api.Invalid("body.listId", "That list doesn't exist in this workspace.")
		}
		if !scopeAllowsList(ctx, listID) {
			return out, api.Invalid("body.listId", "Your role is limited to other lists.")
		}
		out.Target = l.Name
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			// Suppressed addresses can't join a list without a new opt-in.
			blocked, err := failures(ctx, tx, `SELECT c.email, s.reason FROM contacts c JOIN suppressions s ON s.workspace_id = c.workspace_id AND s.email = c.email
				WHERE c.id = ANY ($1)`, ids)
			if err != nil {
				return err
			}
			for _, b := range blocked {
				out.Failures = appendFailure(out.Failures, b.Email, fmt.Sprintf("on the suppression list (%s); needs a new opt-in", strings.ToLower(reasonName(b.Reason))))
			}
			out.Failed = int64(len(blocked))
			tag, err := tx.Exec(ctx, `INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at)
				SELECT $1, c.id, 'subscribed', 'team', now() FROM contacts c
				WHERE c.id = ANY ($2) AND NOT EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email)
				ON CONFLICT (list_id, contact_id) DO UPDATE SET status = 'subscribed', source = 'team',
				    subscribed_at = CASE WHEN list_memberships.status = 'subscribed' THEN list_memberships.subscribed_at ELSE now() END, unsubscribed_at = NULL
				WHERE list_memberships.status <> 'subscribed'`, listID, ids)
			if err != nil {
				return err
			}
			added, err := s.idsWithout(ctx, tx, ids, blocked)
			if err != nil {
				return err
			}
			if err := insertConsentEvents(ctx, tx, added, &listID, "subscribed", "team", "added by the team", actor, s.now()); err != nil {
				return err
			}
			out.Done = out.Matched - out.Failed
			_ = tag
			return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &ws.ID, Action: "contacts.add_to_list", TargetType: "list", TargetID: listID.String(), TargetName: l.Name,
				Metadata: map[string]any{"matched": out.Matched, "done": out.Done, "failed": out.Failed, "filter": selectionMeta(in.Selection)}})
		})
	case "add_tag":
		tag := cleanTags([]string{in.Tag})
		if len(tag) == 0 {
			return out, api.Invalid("body.tag", "Enter a tag.")
		}
		out.Target = tag[0]
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `UPDATE contacts SET tags = array_append(tags, $2), updated_at = now() WHERE id = ANY ($1) AND NOT ($2 = ANY (tags))`, ids, tag[0]); err != nil {
				return err
			}
			out.Done = out.Matched
			return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &ws.ID, Action: "contacts.add_tag", TargetType: "tag", TargetName: tag[0],
				Metadata: map[string]any{"matched": out.Matched, "filter": selectionMeta(in.Selection)}})
		})
	case "suppress":
		reason := strings.TrimSpace(in.Reason)
		if reason == "" {
			return out, api.Invalid("body.reason", "Say why these contacts are suppressed; it goes to the audit log.")
		}
		var emails []string
		if err := pgxSelect(ctx, s.Pool, &emails, `SELECT email FROM contacts WHERE id = ANY ($1)`, ids); err != nil {
			return out, err
		}
		res, err := s.addSuppressions(ctx, ws, emails, "manual", reason)
		if err != nil {
			return out, err
		}
		out.Done = int64(res.Added)
		out.Failed = int64(res.Already)
		if res.Already > 0 {
			out.Note = fmt.Sprintf("%d were already on the suppression list.", res.Already)
		}
	case "resend_optin":
		rows, qerr := s.Pool.Query(ctx, `SELECT * FROM contacts WHERE id = ANY ($1) LIMIT 5000`, ids)
		if qerr != nil {
			return out, qerr
		}
		contacts, cerr := pgx.CollectRows(rows, pgx.RowToStructByName[store.Contact])
		if cerr != nil {
			return out, cerr
		}
		if s.Mail == nil || !s.Mail.SystemAvailable(ctx, &ws.ID) {
			return out, api.Err(409, "no_system_provider", "No sending provider is connected to the System stream, so opt-in emails can't go out yet. Connect one in Sending providers.")
		}
		for _, c := range contacts {
			if sent, _ := s.sendConfirmation(ctx, ws, c); sent {
				out.Done++
				_ = s.Q.InsertConsentEvent(ctx, store.InsertConsentEventParams{ID: uuid.Must(uuid.NewV7()), ContactID: c.ID, Action: "optin_sent", Source: "team",
					Note: "confirmation sent via the System stream", ActorID: actor, OccurredAt: s.now()})
			} else {
				out.Failed++
				out.Failures = appendFailure(out.Failures, c.Email, "the System provider refused the email")
			}
		}
		err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "contacts.resend_optin", TargetType: "contact",
			Metadata: map[string]any{"matched": out.Matched, "sent": out.Done, "failed": out.Failed}})
	default:
		return out, api.Invalid("body.action", "Pick an action.")
	}
	return out, err
}

// ContactEraseInput erases contacts' personal data.
type ContactEraseInput struct {
	Selection
	Confirm int64 `json:"confirm" doc:"The number of contacts, typed to confirm"`
}

// Erase removes personal data of the selected contacts: names, fields,
// tags, list memberships, consent history and preferences go; the address
// is replaced. Suppressions stay, so an erased address is never mailed again.
func (s *Service) Erase(ctx context.Context, ws *store.Workspace, in ContactEraseInput) (ContactBulkResult, error) {
	out := ContactBulkResult{Action: "erase", Failures: []ContactBulkFailure{}}
	ids, err := s.selectIDs(ctx, ws, in.Selection)
	if err != nil {
		return out, err
	}
	out.Matched = int64(len(ids))
	if in.Confirm != out.Matched {
		return out, api.Invalid("body.confirm", fmt.Sprintf("Type %d to confirm erasing %d contacts.", out.Matched, out.Matched))
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		if err := eraseContacts(ctx, tx, ids); err != nil {
			return err
		}
		out.Done = out.Matched
		ev := audit.Event{WorkspaceID: &ws.ID, Action: "contacts.erase", TargetType: "contact", Metadata: map[string]any{"count": out.Done}}
		if len(ids) == 1 {
			ev.TargetID = ids[0].String()
		}
		return s.Audit.LogWith(ctx, store.New(tx), ev)
	})
	return out, err
}

// eraseContacts scrubs personal data in place (the rows stay so campaign
// statistics keep adding up).
func eraseContacts(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) error {
	stmts := []string{
		`DELETE FROM list_memberships WHERE contact_id = ANY ($1)`,
		`DELETE FROM consent_events WHERE contact_id = ANY ($1)`,
		`DELETE FROM contact_preferences WHERE contact_id = ANY ($1)`,
		`DELETE FROM segment_members WHERE contact_id = ANY ($1)`,
		`UPDATE messages SET email = '' WHERE contact_id = ANY ($1)`,
		`UPDATE delivery_events SET ip = NULL, user_agent = '', url = '' WHERE contact_id = ANY ($1)`,
		// Grow: a referrer's link, hub and rewards go; the referrals they
		// made, and their own referral, stay counted without an address.
		`DELETE FROM referral_rewards WHERE contact_id = ANY ($1)`,
		`DELETE FROM referral_links WHERE contact_id = ANY ($1)`,
		`UPDATE referrals SET ip = NULL WHERE contact_id = ANY ($1)`,
		`UPDATE partner_requests_in SET contact_id = NULL WHERE contact_id = ANY ($1)`,
		`UPDATE contacts SET erased_at = now(), email = 'erased-' || id || '@erased.invalid', first_name = '', last_name = '', company = '',
		    country = '', timezone = '', language = '', tags = '{}', attributes = '{}', consent_evidence = '{}', mailbox_detail = '',
		    status = 'suppressed', updated_at = now()
		 WHERE id = ANY ($1)`,
	}
	for _, q := range stmts {
		if _, err := tx.Exec(ctx, q, ids); err != nil {
			return err
		}
	}
	return nil
}

type failureRow struct {
	Email  string
	Reason string
}

func failures(ctx context.Context, tx pgx.Tx, q string, ids []uuid.UUID) ([]failureRow, error) {
	rows, err := tx.Query(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[failureRow])
}

func (s *Service) idsWithout(ctx context.Context, tx pgx.Tx, ids []uuid.UUID, blocked []failureRow) ([]uuid.UUID, error) {
	if len(blocked) == 0 {
		return ids, nil
	}
	emails := make([]string, len(blocked))
	for i, b := range blocked {
		emails[i] = b.Email
	}
	rows, err := tx.Query(ctx, `SELECT id FROM contacts WHERE id = ANY ($1) AND NOT (email = ANY ($2))`, ids, emails)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
}

func appendFailure(fs []ContactBulkFailure, email, reason string) []ContactBulkFailure {
	if len(fs) >= 20 {
		return fs
	}
	return append(fs, ContactBulkFailure{Email: email, Reason: reason})
}

// insertConsentEvents records one consent event per contact.
func insertConsentEvents(ctx context.Context, tx pgx.Tx, contacts []uuid.UUID, list *uuid.UUID, action, source, note string, actor *uuid.UUID, at time.Time) error {
	if len(contacts) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(contacts))
	for i := range ids {
		ids[i] = uuid.Must(uuid.NewV7())
	}
	_, err := tx.Exec(ctx, `INSERT INTO consent_events (id, contact_id, list_id, action, source, note, actor_id, occurred_at)
		SELECT x.id, x.contact_id, $3, $4, $5, $6, $7, $8 FROM unnest($1::uuid[], $2::uuid[]) AS x(id, contact_id)`,
		ids, contacts, list, action, source, note, actor, at)
	return err
}

func pgxSelect(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, dst *[]string, sql string, args ...any) error {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	out, err := pgx.CollectRows(rows, pgx.RowTo[string])
	*dst = out
	return err
}

func selectionMeta(sel Selection) map[string]any {
	if sel.AllMatching {
		return map[string]any{"allMatching": true, "filter": sel.Filter}
	}
	return map[string]any{"ids": len(sel.IDs)}
}

// ContactExportInput selects the contacts to export.
type ContactExportInput struct {
	Selection
}

// Export writes the selected contacts as CSV: built-in columns, lists and
// every custom field. Every export is logged with its size and filter.
func (s *Service) Export(ctx context.Context, ws *store.Workspace, in ContactExportInput) ([]byte, int, error) {
	ids, err := s.selectIDs(ctx, ws, in.Selection)
	if err != nil {
		return nil, 0, err
	}
	fields, err := s.Q.ListCustomFields(ctx, ws.ID)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT c.email, c.first_name, c.last_name, c.company, c.country, c.timezone, c.language, c.tier, c.consent,
		c.mailbox_provider, c.source, c.created_at, c.last_open_at, c.last_click_at, c.tags, c.attributes,
		coalesce((SELECT string_agg(l.name, '; ' ORDER BY l.created_at) FROM list_memberships lm JOIN lists l ON l.id = lm.list_id
		          WHERE lm.contact_id = c.id AND lm.status = 'subscribed' AND l.deleted_at IS NULL), '')
		FROM contacts c WHERE c.id = ANY ($1) ORDER BY c.created_at, c.id`, ids)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	header := []string{"email", "first_name", "last_name", "company", "country", "timezone", "language", "lists", "tier", "consent",
		"mailbox_provider", "source", "joined_at", "last_open_at", "last_click_at", "tags"}
	for _, f := range fields {
		header = append(header, f.Key)
	}
	_ = w.Write(header)
	n := 0
	for rows.Next() {
		var email, first, last, company, country, tz, lang, tier, consent, mbp, source, lists string
		var joined time.Time
		var lastOpen, lastClick *time.Time
		var tags []string
		var attrs json.RawMessage
		if err := rows.Scan(&email, &first, &last, &company, &country, &tz, &lang, &tier, &consent, &mbp, &source, &joined, &lastOpen, &lastClick, &tags, &attrs, &lists); err != nil {
			return nil, 0, err
		}
		var a map[string]any
		_ = json.Unmarshal(attrs, &a)
		rec := []string{email, first, last, company, country, tz, lang, lists, tier, consent, mbp, segments.SourceLabel(source),
			joined.UTC().Format(time.RFC3339), timeOrEmpty(lastOpen), timeOrEmpty(lastClick), strings.Join(tags, "; ")}
		for _, f := range fields {
			rec = append(rec, attrString(a[f.Key]))
		}
		_ = w.Write(csvSafe(rec...))
		n++
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	w.Flush()
	err = s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "contacts.export", TargetType: "contact",
		Metadata: map[string]any{"rows": n, "selection": selectionMeta(in.Selection)}})
	return buf.Bytes(), n, err
}

func timeOrEmpty(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
