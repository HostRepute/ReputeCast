package audience

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/scoring"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
)

// ContactListItem is one row of the contacts table.
type ContactListItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	// Lists the contact is on (subscribed or waiting for opt-in), in list order.
	Lists           []string   `json:"lists"`
	Tier            string     `json:"tier" enum:"active,cooling,at_risk,sunset"`
	Consent         string     `json:"consent" enum:"double,single,pending"`
	MailboxProvider string     `json:"mailboxProvider"`
	MailboxLabel    string     `json:"mailboxLabel" doc:"Outlook, or Other · Google Workspace"`
	LastOpenAt      *time.Time `json:"lastOpenAt,omitempty"`
	LastOpenMachine bool       `json:"lastOpenMachine" doc:"The last open was a machine open (Apple Mail Privacy Protection)"`
	JoinedAt        time.Time  `json:"joinedAt"`
	Source          string     `json:"source"`
	SourceLabel     string     `json:"sourceLabel"`
	Suppressed      bool       `json:"suppressed"`
	// Score is null while the workspace has no scoring; ScoreBand is its band.
	Score     *int   `json:"score,omitempty"`
	ScoreBand string `json:"scoreBand,omitempty" enum:"cold,warm,hot"`
	// Paid is the contact's paid status (Earn), for callers with earn:view.
	Paid *ContactPaid `json:"paid,omitempty"`
	// Phone is the number as people read it (+1 312 555 0123), when there is one.
	Phone string `json:"phone,omitempty"`
	// Fields are the custom fields asked for (a view's columns), by key.
	Fields map[string]any `json:"fields,omitempty"`
}

// ContactList is one page of contacts.
type ContactList struct {
	Items []ContactListItem `json:"items"`
	// Total counts the matching contacts exactly up to 10,000; above that
	// it is 10,000 and TotalCapped is set ("10,000+"), unless the exact
	// count was asked for (review 2026-11 D5).
	Total       int64 `json:"total"`
	TotalCapped bool  `json:"totalCapped,omitempty"`
	Page        int   `json:"page"`
	PageSize    int   `json:"pageSize"`
	// NextCursor and PrevCursor page on from this page's last row, and back
	// from its first; empty at either end.
	NextCursor string `json:"nextCursor,omitempty"`
	PrevCursor string `json:"prevCursor,omitempty"`
}

// countCap is how far the contacts and suppressions counts go before they
// read "10,000+".
const countCap = 10000

// ContactPage asks for one page of contacts: from a cursor, or by page
// number from the start, and whether to count past 10,000.
type ContactPage struct {
	Cursor   string
	Page     int
	PageSize int
	Exact    bool
	// Fields are custom field keys whose values the rows carry.
	Fields []string
}

// ListContacts returns one page in the filter's order, most recent
// activity first by default.
func (s *Service) ListContacts(ctx context.Context, ws *store.Workspace, f ContactFilter, pg ContactPage) (ContactList, error) {
	if pg.PageSize <= 0 {
		pg.PageSize = 10
	}
	if pg.Page <= 0 {
		pg.Page = 1
	}
	fs, err := s.filterSQL(ctx, ws.ID, f)
	if err != nil {
		return ContactList{}, err
	}
	out := ContactList{Items: []ContactListItem{}, Page: pg.Page, PageSize: pg.PageSize}
	q, err := s.listQuery(ctx, fs, f, pg)
	if err != nil {
		return out, err
	}
	sort, keys, cur, backward := q.sort, q.keys, q.cur, q.backward
	bands, scored, err := scoring.AppliedBands(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	rows, err := s.Pool.Query(ctx, q.sql, q.args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	ids := []uuid.UUID{}
	marks := []contactCursor{}
	l := i18n.LangFrom(ctx)
	for rows.Next() {
		var it ContactListItem
		var id uuid.UUID
		var first, last, detail string
		var score int
		dest := []any{&id, &it.Email, &first, &last, &it.Tier, &it.Consent, &it.MailboxProvider, &detail,
			&it.LastOpenAt, &it.LastOpenMachine, &it.JoinedAt, &it.Source, &it.Suppressed, &score}
		vals := sortDest(keys)
		dest = append(dest, vals...)
		var fields []byte
		if len(pg.Fields) > 0 {
			dest = append(dest, &fields)
		}
		if err := rows.Scan(dest...); err != nil {
			return out, err
		}
		if len(fields) > 0 {
			_ = json.Unmarshal(fields, &it.Fields)
		}
		mark := contactCursor{Sort: sort, ID: id, Values: make([]any, len(keys))}
		for i, v := range vals {
			mark.Values[i] = cursorValue(deref(v))
		}
		marks = append(marks, mark)
		if scored {
			it.Score, it.ScoreBand = &score, bands.Band(score)
		}
		it.ID = id.String()
		it.Name = strings.TrimSpace(first + " " + last)
		it.MailboxLabel = mailboxLabelIn(l, it.MailboxProvider, detail)
		it.SourceLabel = segments.SourceLabelIn(l, it.Source)
		it.Lists = []string{}
		ids = append(ids, id)
		out.Items = append(out.Items, it)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	more := len(out.Items) > pg.PageSize
	if cur == nil && pg.Page == 1 && !more {
		// The first page holds every match: its size is the count, so a
		// search that finds a few contacts reads the indexes once, not
		// twice (review 2026-11 OP-13).
		out.Total, out.TotalCapped = int64(len(out.Items)), false
	} else if out.Total, out.TotalCapped, err = s.countContacts(ctx, fs, pg.Exact); err != nil {
		return out, err
	}
	if more {
		out.Items, ids, marks = out.Items[:pg.PageSize], ids[:pg.PageSize], marks[:pg.PageSize]
	}
	if backward {
		slices.Reverse(out.Items)
		slices.Reverse(ids)
		slices.Reverse(marks)
	}
	if len(marks) > 0 {
		// Forward, more rows follow when the page overflowed; back, the
		// page it came from does. Earlier rows exist on any page after the
		// first, or when going back overflowed.
		if (!backward && more) || backward {
			out.NextCursor = marks[len(marks)-1].encode()
		}
		if (backward && more) || (!backward && (cur != nil || pg.Page > 1)) {
			first := marks[0]
			first.Before = true
			out.PrevCursor = first.encode()
		}
	}
	srcs := make([]string, len(out.Items))
	for i, it := range out.Items {
		srcs[i] = it.Source
	}
	if named := s.nameSources(ctx, ws.ID, ids, srcs); named != nil {
		for i, id := range ids {
			if n, ok := named[id]; ok {
				out.Items[i].SourceLabel = n
			}
		}
	}
	names, err := s.Q.ContactListNames(ctx, ids)
	if err != nil {
		return out, err
	}
	byID := map[string][]string{}
	for _, n := range names {
		byID[n.ContactID.String()] = append(byID[n.ContactID.String()], n.Name)
	}
	for i := range out.Items {
		if l := byID[out.Items[i].ID]; l != nil {
			out.Items[i].Lists = l
		}
	}
	paid, err := s.contactPaid(ctx, ws.ID, ids)
	if err != nil {
		return out, err
	}
	for i, id := range ids {
		out.Items[i].Paid = paid[id]
	}
	if err := s.fillPhones(ctx, out.Items, ids); err != nil {
		return out, err
	}
	return out, nil
}

// listQuery is the contacts table's page query.
type listQuery struct {
	sql      string
	args     []any
	sort     string
	keys     []sortKey
	cur      *contactCursor
	backward bool
}

// listQuery builds the page query of a filter: its order, the keyset from
// the cursor (or an offset from the page number), one row more than the
// page to tell whether more follow, and the custom fields asked for.
func (s *Service) listQuery(ctx context.Context, fs filterSQL, f ContactFilter, pg ContactPage) (listQuery, error) {
	q := listQuery{}
	q.sort, q.keys = contactSort(f.Sort)
	if pg.Cursor != "" {
		c, err := readCursor(pg.Cursor, q.sort, q.keys)
		if err != nil {
			return q, api.Invalid("cursor", say(ctx, mCursor, nil))
		}
		q.cur = &c
		fs.addNumbered(keyset(q.keys, "c.id", c, func(v any) string {
			fs.args = append(fs.args, v)
			return fmt.Sprintf("$%d", len(fs.args))
		}))
	}
	q.backward = q.cur != nil && q.cur.Before
	args := append(slices.Clip(fs.args), pg.PageSize+1)
	limit := len(args)
	offset := ""
	if q.cur == nil && pg.Page > 1 {
		// A page number without a cursor (the public API's first pages).
		args = append(args, (pg.Page-1)*pg.PageSize)
		offset = fmt.Sprintf(" OFFSET $%d", len(args))
	}
	exprs := make([]string, len(q.keys), len(q.keys)+1)
	for i, k := range q.keys {
		exprs[i] = k.expr
	}
	if len(pg.Fields) > 0 {
		args = append(args, pg.Fields)
		exprs = append(exprs, fmt.Sprintf("(SELECT jsonb_object_agg(a.key, a.value) FROM jsonb_each(c.attributes) a WHERE a.key = ANY ($%d::text[]))", len(args)))
	}
	q.args = args
	q.sql = fmt.Sprintf(`SELECT c.id, c.email, c.first_name, c.last_name, c.tier, c.consent, c.mailbox_provider, c.mailbox_detail,
		c.last_open_at, c.last_open_machine, c.created_at, c.source,
		EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email),
		coalesce(cs.score, 0)::int, %s
		FROM contacts c LEFT JOIN contact_scores cs ON cs.contact_id = c.id WHERE %s
		ORDER BY %s
		LIMIT $%d%s`, strings.Join(exprs, ", "), fs.where, orderBy(q.keys, "c.id", q.backward), limit, offset)
	return q, nil
}

// countContacts counts the filter's contacts, exactly up to countCap and
// beyond it only when asked.
func (s *Service) countContacts(ctx context.Context, fs filterSQL, exact bool) (int64, bool, error) {
	var n int64
	if exact {
		err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM contacts c WHERE "+fs.where, fs.args...).Scan(&n)
		return n, false, err
	}
	err := s.Pool.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM (SELECT 1 FROM contacts c WHERE %s LIMIT %d) x", fs.where, countCap+1), fs.args...).Scan(&n)
	if n > countCap {
		return countCap, true, err
	}
	return n, false, err
}

// sortDest are scan targets for an order's keys.
func sortDest(keys []sortKey) []any {
	out := make([]any, len(keys))
	for i, k := range keys {
		switch k.kind {
		case 't':
			out[i] = new(*time.Time)
		case 'i':
			out[i] = new(*int32)
		default:
			out[i] = new(*string)
		}
	}
	return out
}

func deref(v any) any {
	switch x := v.(type) {
	case **time.Time:
		return *x
	case **int32:
		return *x
	case **string:
		return *x
	}
	return v
}
