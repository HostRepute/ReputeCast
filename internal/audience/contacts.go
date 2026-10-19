package audience

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
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
}

// ContactList is one page of contacts.
type ContactList struct {
	Items    []ContactListItem `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

// ListContacts returns one page, most recent activity first.
func (s *Service) ListContacts(ctx context.Context, ws *store.Workspace, f ContactFilter, page, pageSize int) (ContactList, error) {
	if pageSize <= 0 {
		pageSize = 10
	}
	if page <= 0 {
		page = 1
	}
	fs, err := s.filterSQL(ctx, ws.ID, f)
	if err != nil {
		return ContactList{}, err
	}
	out := ContactList{Items: []ContactListItem{}, Page: page, PageSize: pageSize}
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM contacts c WHERE "+fs.where, fs.args...).Scan(&out.Total); err != nil {
		return out, err
	}
	args := append(fs.args, pageSize, (page-1)*pageSize)
	bands, scored, err := scoring.AppliedBands(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	rows, err := s.Pool.Query(ctx, fmt.Sprintf(`SELECT c.id, c.email, c.first_name, c.last_name, c.tier, c.consent, c.mailbox_provider, c.mailbox_detail,
		c.last_open_at, c.last_open_machine, c.created_at, c.source,
		EXISTS (SELECT 1 FROM suppressions s WHERE s.workspace_id = c.workspace_id AND s.email = c.email),
		coalesce(cs.score, 0)::int
		FROM contacts c LEFT JOIN contact_scores cs ON cs.contact_id = c.id WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, fs.where, contactOrder(f.ScoreGrowthFilter), len(args)-1, len(args)), args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	ids := []uuid.UUID{}
	for rows.Next() {
		var it ContactListItem
		var id uuid.UUID
		var first, last, detail string
		var score int
		if err := rows.Scan(&id, &it.Email, &first, &last, &it.Tier, &it.Consent, &it.MailboxProvider, &detail,
			&it.LastOpenAt, &it.LastOpenMachine, &it.JoinedAt, &it.Source, &it.Suppressed, &score); err != nil {
			return out, err
		}
		if scored {
			it.Score, it.ScoreBand = &score, bands.Band(score)
		}
		it.ID = id.String()
		it.Name = strings.TrimSpace(first + " " + last)
		it.MailboxLabel = MailboxLabel(it.MailboxProvider, detail)
		it.SourceLabel = segments.SourceLabel(it.Source)
		it.Lists = []string{}
		ids = append(ids, id)
		out.Items = append(out.Items, it)
	}
	if err := rows.Err(); err != nil {
		return out, err
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
	return out, nil
}

// ContactSavedView is a chip above the contacts table.
type ContactSavedView struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Filters ContactFilter `json:"filters"`
	Count   int64         `json:"count"`
	// Shared views are visible to everyone in the workspace.
	Shared bool `json:"shared"`
}

// ContactSavedViews is the views row: "All contacts" plus the saved views.
type ContactSavedViews struct {
	All   int64              `json:"all"`
	Items []ContactSavedView `json:"items"`
}

// Views returns the saved views with their counts, counted in one scan.
func (s *Service) Views(ctx context.Context, ws *store.Workspace) (ContactSavedViews, error) {
	var uid uuid.UUID
	if u := api.UserFrom(ctx); u != nil {
		uid = u.ID
	}
	views, err := s.Q.ListSavedViews(ctx, store.ListSavedViewsParams{WorkspaceID: ws.ID, UserID: &uid})
	if err != nil {
		return ContactSavedViews{}, err
	}
	out := ContactSavedViews{Items: make([]ContactSavedView, 0, len(views))}
	base, err := s.filterSQL(ctx, ws.ID, ContactFilter{})
	if err != nil {
		return out, err
	}
	// One query: count(*) FILTER (WHERE <view>) per view, each view's
	// arguments numbered after the ones already used.
	sel := []string{"count(*)"}
	for _, v := range views {
		var f ContactFilter
		_ = json.Unmarshal(v.Filters, &f)
		out.Items = append(out.Items, ContactSavedView{ID: v.ID.String(), Name: v.Name, Filters: f, Shared: v.UserID == nil})
		sub := filterSQL{where: "true", args: base.args}
		if err := s.applyFilter(ctx, ws.ID, &sub, f); err != nil {
			sel = append(sel, "0")
			continue
		}
		base.args = sub.args
		sel = append(sel, "count(*) FILTER (WHERE "+sub.where+")")
	}
	args := base.args
	counts := make([]int64, len(sel))
	ptrs := make([]any, len(sel))
	for i := range counts {
		ptrs[i] = &counts[i]
	}
	if err := s.Pool.QueryRow(ctx, "SELECT "+strings.Join(sel, ", ")+" FROM contacts c WHERE "+base.where, args...).Scan(ptrs...); err != nil {
		return out, err
	}
	out.All = counts[0]
	for i := range out.Items {
		out.Items[i].Count = counts[i+1]
	}
	return out, nil
}

// ContactViewInput saves the current filters as a view.
type ContactViewInput struct {
	Name    string        `json:"name" minLength:"1" maxLength:"80"`
	Filters ContactFilter `json:"filters"`
	// Personal views are only visible to their creator.
	Personal bool `json:"personal,omitempty"`
}

func (s *Service) CreateView(ctx context.Context, ws *store.Workspace, in ContactViewInput) (ContactSavedView, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return ContactSavedView{}, api.Invalid("body.name", "Give the view a name.")
	}
	if _, err := s.filterSQL(ctx, ws.ID, in.Filters); err != nil {
		return ContactSavedView{}, err
	}
	filters, _ := json.Marshal(in.Filters)
	var user *uuid.UUID
	if in.Personal {
		user = actorID(ctx)
	}
	var v store.SavedView
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		v, err = q.CreateSavedView(ctx, store.CreateSavedViewParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, UserID: user, Name: in.Name, Filters: filters})
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "contact_view.create", TargetType: "saved_view", TargetID: v.ID.String(), TargetName: v.Name, After: in})
	})
	if err != nil {
		return ContactSavedView{}, err
	}
	return ContactSavedView{ID: v.ID.String(), Name: v.Name, Filters: in.Filters, Shared: user == nil}, nil
}

func (s *Service) DeleteView(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		v, err := q.GetSavedView(ctx, store.GetSavedViewParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "Saved view")
		}
		if v.UserID != nil {
			if me := actorID(ctx); me == nil || *me != *v.UserID {
				return api.NotFound("Saved view")
			}
		}
		if _, err := q.DeleteSavedView(ctx, store.DeleteSavedViewParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "contact_view.delete", TargetType: "saved_view", TargetID: id.String(), TargetName: v.Name, Before: v.Filters})
	})
}
