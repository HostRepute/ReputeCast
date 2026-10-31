package audience

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/store"
)

// exportMessage is one email to the contact in their data export.
type exportMessage struct {
	Stream      string     `json:"stream"`
	Email       string     `json:"email"`
	Campaign    string     `json:"campaign,omitempty"`
	Provider    string     `json:"provider,omitempty"`
	Status      string     `json:"status"`
	QueuedAt    time.Time  `json:"queuedAt"`
	DeliveredAt *time.Time `json:"deliveredAt,omitempty"`
	OpenedAt    *time.Time `json:"openedAt,omitempty"`
	ClickedAt   *time.Time `json:"clickedAt,omitempty"`
}

// exportCut is a part of the export that couldn't be written in full.
type exportCut struct {
	Section string `json:"section"`
	After   int    `json:"after"`
	Reason  string `json:"reason"`
}

// ContactExport is a contact's data export ready to stream: its file name
// and the writer. Everything that can fail before the first byte (a contact
// that doesn't exist) fails when it is made, as a proper API error.
type ContactExport struct {
	Name  string
	Write func(w io.Writer) error
}

// ExportContact prepares a JSON document with everything ReputeCast stores
// about one contact, for a data subject access request ("Export data
// (GDPR)"): the profile, every email and event, and a section per erasure
// source (SMS, paid membership, orders, referrals… review 2026-11 DP-2).
// Nothing is capped; a section that fails part way is named in "cut" and
// "complete" is false. A section whose source needs a permission the caller
// lacks says so instead of its rows. The export is logged.
func (s *Service) ExportContact(ctx context.Context, ws *store.Workspace, id uuid.UUID) (*ContactExport, error) {
	prof, err := s.Profile(ctx, ws, id)
	if err != nil {
		return nil, err
	}
	c, err := s.getContact(ctx, s.Q, ws, id)
	if err != nil {
		return nil, err
	}
	p := api.PrincipalFrom(ctx)
	write := func(w io.Writer) error {
		bw := bufio.NewWriterSize(w, 64<<10)
		ew := &exportWriter{w: bw}
		counts := map[string]int{}
		var cuts []exportCut
		ew.raw("{")
		ew.field("exportedAt", s.now().UTC())
		ew.field("workspace", ws.Name)
		ew.field("contact", prof)
		ew.field("customFields", json.RawMessage(nonEmptyJSON(c.Attributes)))
		ew.field("consentEvidence", json.RawMessage(nonEmptyJSON(c.ConsentEvidence)))
		if prof.Suppression != nil {
			ew.field("suppression", prof.Suppression)
		}
		// One read-only snapshot for what the sources read.
		err := pgx.BeginTxFunc(ctx, s.Pool, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
			n, err := withSavepoint(ctx, tx, func(sp pgx.Tx) (int, error) { return s.exportMessages(ctx, sp, id, ew) })
			counts["messages"] = n
			if err != nil {
				cuts = append(cuts, exportCut{Section: "messages", After: n, Reason: err.Error()})
			}
			n, err = s.exportEvents(ctx, ws, id, ew)
			counts["events"] = n
			if err != nil {
				cuts = append(cuts, exportCut{Section: "events", After: n, Reason: err.Error()})
			}
			ew.key("sections")
			ew.raw("{")
			first := true
			for _, src := range s.registry().Sources() {
				for _, ex := range src.Exports {
					if !first {
						ew.raw(",")
					}
					first = false
					ew.str(ex.Section)
					ew.raw(":")
					if ex.Needs != "" && (p == nil || !p.Can(ex.Needs)) {
						ew.value(map[string]string{"omitted": "needs " + string(ex.Needs)})
						continue
					}
					// A savepoint each, so one failing section leaves the
					// snapshot usable for the next.
					n, err := withSavepoint(ctx, tx, func(sp pgx.Tx) (int, error) {
						return exportRows(ctx, sp, ex.SQL, ew, id, ws.ID, strings.ToLower(c.Email), c.Phone)
					})
					counts[ex.Section] = n
					if err != nil {
						cuts = append(cuts, exportCut{Section: ex.Section, After: n, Reason: err.Error()})
					}
				}
			}
			ew.raw("}")
			return nil
		})
		if err != nil {
			cuts = append(cuts, exportCut{Section: "sections", Reason: err.Error()})
		}
		if cuts == nil {
			cuts = []exportCut{}
		}
		ew.field("cut", cuts)
		ew.field("complete", len(cuts) == 0)
		ew.raw("}\n")
		if ew.err == nil {
			ew.err = bw.Flush()
		}
		// The export is logged once written, even if the reader went away.
		if err := s.Audit.Log(context.WithoutCancel(ctx), audit.Event{WorkspaceID: &ws.ID, Action: "contact.export", TargetType: "contact", TargetID: id.String(),
			TargetName: c.Email, Metadata: map[string]any{"counts": counts, "complete": len(cuts) == 0}}); err != nil {
			return err
		}
		return ew.err
	}
	return &ContactExport{Name: "contact-" + id.String()[:8] + ".json", Write: write}, nil
}

// withSavepoint runs fn in a savepoint, rolled back if fn fails.
func withSavepoint(ctx context.Context, tx pgx.Tx, fn func(pgx.Tx) (int, error)) (int, error) {
	sp, err := tx.Begin(ctx)
	if err != nil {
		return 0, err
	}
	n, err := fn(sp)
	if err != nil {
		_ = sp.Rollback(ctx)
		return n, err
	}
	return n, sp.Commit(ctx)
}

// exportMessages writes every email to the contact, newest first.
func (s *Service) exportMessages(ctx context.Context, tx pgx.Tx, id uuid.UUID, ew *exportWriter) (int, error) {
	ew.key("messages")
	ew.raw("[")
	defer ew.raw("]")
	rows, err := tx.Query(ctx, `SELECT m.stream, m.email, coalesce(cp.name, a.name, ''), coalesce(p.name, ''), m.status, m.queued_at,
		m.delivered_at, m.opened_at, m.clicked_at
		FROM messages m LEFT JOIN campaigns cp ON cp.id = m.campaign_id LEFT JOIN automations a ON a.id = m.automation_id
		LEFT JOIN providers p ON p.id = m.provider_id
		WHERE m.contact_id = $1 ORDER BY m.queued_at DESC, m.id`, id)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var m exportMessage
		if err := rows.Scan(&m.Stream, &m.Email, &m.Campaign, &m.Provider, &m.Status, &m.QueuedAt, &m.DeliveredAt, &m.OpenedAt, &m.ClickedAt); err != nil {
			return n, err
		}
		if n > 0 {
			ew.raw(",")
		}
		ew.value(m)
		n++
	}
	return n, rows.Err()
}

// exportEvents writes the whole activity timeline, page by page.
func (s *Service) exportEvents(ctx context.Context, ws *store.Workspace, id uuid.UUID, ew *exportWriter) (int, error) {
	ew.key("events")
	ew.raw("[")
	defer ew.raw("]")
	n, cursor := 0, ""
	for {
		page, err := s.Activity(ctx, ws, id, "all", cursor, 100)
		if err != nil {
			return n, err
		}
		for _, it := range page.Items {
			if n > 0 {
				ew.raw(",")
			}
			ew.value(it)
			n++
		}
		if page.NextCursor == "" || page.NextCursor == cursor {
			return n, nil
		}
		cursor = page.NextCursor
	}
}

// exportRows writes a source's export query as an array of objects, one
// per row, column by column.
func exportRows(ctx context.Context, tx pgx.Tx, sql string, ew *exportWriter, contact, ws uuid.UUID, email, phone string) (int, error) {
	ew.raw("[")
	defer ew.raw("]")
	rows, err := tx.Query(ctx, "WITH erasure_params AS (SELECT $1::uuid, $2::uuid, $3::text, $4::text) "+sql, contact, ws, email, phone)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	fields := rows.FieldDescriptions()
	n := 0
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return n, err
		}
		row := make(map[string]any, len(vals))
		for i, f := range fields {
			row[f.Name] = vals[i]
		}
		if n > 0 {
			ew.raw(",")
		}
		ew.value(row)
		n++
	}
	return n, rows.Err()
}

// exportWriter writes JSON piece by piece, keeping the first error.
type exportWriter struct {
	w      io.Writer
	err    error
	fields int
}

func (e *exportWriter) raw(s string) {
	if e.err == nil {
		_, e.err = io.WriteString(e.w, s)
	}
}

func (e *exportWriter) str(s string) {
	b, _ := json.Marshal(s)
	e.raw(string(b))
}

func (e *exportWriter) value(v any) {
	if e.err != nil {
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(fmt.Sprint(v))
	}
	_, e.err = e.w.Write(b)
}

// key starts an object field ("name":), with a comma after the first.
func (e *exportWriter) key(name string) {
	if e.fields > 0 {
		e.raw(",")
	}
	e.fields++
	e.str(name)
	e.raw(":")
}

func (e *exportWriter) field(name string, v any) {
	e.key(name)
	e.value(v)
}

func nonEmptyJSON(b []byte) []byte {
	if len(b) == 0 {
		return []byte("{}")
	}
	return b
}
