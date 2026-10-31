package audience

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Saved views: the chips above the contacts table. Each keeps its counts
// (review 2026-11 AU-3), counted again once viewStale old or when its
// filters change, so a million contacts aren't scanned on every visit;
// it can be renamed, given the current filters or columns, and deleted
// (AU-8, AU-9).

// viewStale is how long a view's counts are shown before they're counted
// again.
const viewStale = 10 * time.Minute

// ContactSavedView is a chip above the contacts table.
type ContactSavedView struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Filters ContactFilter `json:"filters"`
	Count   int64         `json:"count"`
	// Columns are the table's columns in this view; empty for the usual ones.
	Columns []string `json:"columns"`
	// Shared views are visible to everyone in the workspace.
	Shared bool `json:"shared"`
	// CountedAt is when Count was counted (at most ten minutes ago).
	CountedAt *time.Time `json:"countedAt,omitempty"`
}

// ContactSavedViews is the views row: "All contacts" plus the saved views.
type ContactSavedViews struct {
	All   int64              `json:"all"`
	Items []ContactSavedView `json:"items"`
}

// viewStats are a view's numbers: how many, by tier, and how many wait for
// their opt-in.
type viewStats struct {
	Contacts int64 `json:"contacts"`
	Active   int64 `json:"active"`
	Cooling  int64 `json:"cooling"`
	AtRisk   int64 `json:"atRisk"`
	Sunset   int64 `json:"sunset"`
	Pending  int64 `json:"pending"`
}

// Views returns the saved views with their counts.
func (s *Service) Views(ctx context.Context, ws *store.Workspace) (ContactSavedViews, error) {
	var uid uuid.UUID
	if u := api.UserFrom(ctx); u != nil {
		uid = u.ID
	}
	views, err := s.Q.ListSavedViews(ctx, store.ListSavedViewsParams{WorkspaceID: ws.ID, UserID: &uid})
	if err != nil {
		return ContactSavedViews{}, err
	}
	all, err := s.Q.AudienceCounts(ctx, ws.ID)
	if err != nil {
		return ContactSavedViews{}, err
	}
	out := ContactSavedViews{All: all.Contacts, Items: make([]ContactSavedView, 0, len(views))}
	shown := make([]store.SavedView, 0, len(views))
	for _, v := range views {
		var f ContactFilter
		_ = json.Unmarshal(v.Filters, &f)
		// A view by paid status shows only to members who can see it.
		if f.Paid != "" && !canSeePaid(ctx) {
			continue
		}
		shown = append(shown, v)
	}
	stats, err := s.viewStats(ctx, ws.ID, shown)
	if err != nil {
		return out, err
	}
	for _, v := range shown {
		out.Items = append(out.Items, savedView(v, stats[v.ID]))
	}
	return out, nil
}

func savedView(v store.SavedView, st counted) ContactSavedView {
	var f ContactFilter
	_ = json.Unmarshal(v.Filters, &f)
	cols := v.Columns
	if cols == nil {
		cols = []string{}
	}
	return ContactSavedView{ID: v.ID.String(), Name: v.Name, Filters: f, Columns: cols, Shared: v.UserID == nil, Count: st.Contacts, CountedAt: st.at}
}

// counted is a view's numbers and when they were counted.
type counted struct {
	viewStats
	at *time.Time
}

// viewStats returns each view's numbers: the stored ones while fresh, the
// rest counted in one scan of the workspace's contacts and stored. A view
// whose filters no longer apply (a deleted list or segment) counts 0.
func (s *Service) viewStats(ctx context.Context, wsID uuid.UUID, views []store.SavedView) (map[uuid.UUID]counted, error) {
	now := s.now()
	out := make(map[uuid.UUID]counted, len(views))
	base := filterSQL{where: "c.workspace_id = $1 AND c.erased_at IS NULL", args: []any{wsID}}
	var sel []string
	var stale []uuid.UUID
	for _, v := range views {
		if v.CountedAt != nil && now.Sub(*v.CountedAt) < viewStale && len(v.Stats) > 0 {
			var st viewStats
			_ = json.Unmarshal(v.Stats, &st)
			at := *v.CountedAt
			out[v.ID] = counted{st, &at}
			continue
		}
		var f ContactFilter
		_ = json.Unmarshal(v.Filters, &f)
		sub := filterSQL{where: "true", args: base.args}
		if err := s.applyFilter(ctx, wsID, &sub, f); err != nil {
			out[v.ID] = counted{at: &now}
			continue
		}
		base.args = sub.args
		w := sub.where
		sel = append(sel, fmt.Sprintf(`count(*) FILTER (WHERE %[1]s), count(*) FILTER (WHERE %[1]s AND c.tier = 'active'),
			count(*) FILTER (WHERE %[1]s AND c.tier = 'cooling'), count(*) FILTER (WHERE %[1]s AND c.tier = 'at_risk'),
			count(*) FILTER (WHERE %[1]s AND c.tier = 'sunset'), count(*) FILTER (WHERE %[1]s AND c.consent = 'pending')`, w))
		stale = append(stale, v.ID)
	}
	if len(stale) == 0 {
		return out, nil
	}
	nums := make([]int64, 6*len(stale))
	ptrs := make([]any, len(nums))
	for i := range nums {
		ptrs[i] = &nums[i]
	}
	if err := s.Pool.QueryRow(ctx, "SELECT "+strings.Join(sel, ",\n")+" FROM contacts c WHERE "+base.where, base.args...).Scan(ptrs...); err != nil {
		return out, err
	}
	for i, id := range stale {
		n := nums[6*i:]
		st := viewStats{Contacts: n[0], Active: n[1], Cooling: n[2], AtRisk: n[3], Sunset: n[4], Pending: n[5]}
		b, _ := json.Marshal(st)
		if err := s.Q.SavedViewCounted(ctx, store.SavedViewCountedParams{ID: id, Stats: b, At: &now}); err != nil {
			return out, err
		}
		out[id] = counted{st, &now}
	}
	return out, nil
}

// ContactViewInput saves the current filters as a view.
type ContactViewInput struct {
	Name    string        `json:"name" minLength:"1" maxLength:"80"`
	Filters ContactFilter `json:"filters"`
	// Columns are the table's columns in this view (empty: the usual ones).
	Columns []string `json:"columns,omitempty" maxItems:"20"`
	// Personal views are only visible to their creator.
	Personal bool `json:"personal,omitempty"`
}

// ContactViewUpdate renames a view, or gives it new filters or columns;
// what's left out stays.
type ContactViewUpdate struct {
	Name    *string        `json:"name,omitempty" minLength:"1" maxLength:"80"`
	Filters *ContactFilter `json:"filters,omitempty"`
	Columns *[]string      `json:"columns,omitempty" maxItems:"20"`
}

// The table's columns a view can show, beside custom fields ("field:plan").
var viewColumns = []string{"lists", "tier", "score", "consent", "mailbox", "last_open", "joined", "source", "paid", "phone"}

// cleanColumns checks a view's columns: known ones and the workspace's
// custom fields, each once.
func (s *Service) cleanColumns(ctx context.Context, wsID uuid.UUID, cols []string) ([]string, error) {
	out := []string{}
	var fields map[string]bool
	for _, c := range cols {
		c = strings.TrimSpace(c)
		if slices.Contains(out, c) {
			continue
		}
		if key, ok := strings.CutPrefix(c, "field:"); ok {
			if fields == nil {
				fs, err := s.Q.ListCustomFields(ctx, wsID)
				if err != nil {
					return nil, err
				}
				fields = map[string]bool{}
				for _, f := range fs {
					fields[f.Key] = true
				}
			}
			if !fields[key] {
				return nil, api.Invalid("body.columns", say(ctx, mViewColumn, map[string]any{"column": c}))
			}
		} else if !slices.Contains(viewColumns, c) {
			return nil, api.Invalid("body.columns", say(ctx, mViewColumn, map[string]any{"column": c}))
		}
		out = append(out, c)
	}
	return out, nil
}

// CreateView saves the current filters (and columns) as a view.
func (s *Service) CreateView(ctx context.Context, ws *store.Workspace, in ContactViewInput) (ContactSavedView, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return ContactSavedView{}, api.Invalid("body.name", say(ctx, mViewName, nil))
	}
	if _, err := s.filterSQL(ctx, ws.ID, in.Filters); err != nil {
		return ContactSavedView{}, err
	}
	cols, err := s.cleanColumns(ctx, ws.ID, in.Columns)
	if err != nil {
		return ContactSavedView{}, err
	}
	in.Columns = cols
	filters, _ := json.Marshal(in.Filters)
	var user *uuid.UUID
	if in.Personal {
		user = actorID(ctx)
	}
	var v store.SavedView
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var err error
		v, err = q.CreateSavedView(ctx, store.CreateSavedViewParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, UserID: user, Name: in.Name, Filters: filters, Columns: cols})
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "contact_view.create", TargetType: "saved_view", TargetID: v.ID.String(), TargetName: v.Name, After: in})
	})
	if err != nil {
		return ContactSavedView{}, err
	}
	return savedView(v, counted{}), nil
}

// UpdateView renames a view or saves new filters or columns in it. A
// personal view is its creator's only.
func (s *Service) UpdateView(ctx context.Context, ws *store.Workspace, id uuid.UUID, in ContactViewUpdate) (ContactSavedView, error) {
	var out ContactSavedView
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		v, err := s.ownView(ctx, q, ws.ID, id)
		if err != nil {
			return err
		}
		name, filters, cols := v.Name, v.Filters, v.Columns
		if cols == nil {
			cols = []string{}
		}
		if in.Name != nil {
			if name = strings.TrimSpace(*in.Name); name == "" {
				return api.Invalid("body.name", say(ctx, mViewName, nil))
			}
		}
		if in.Filters != nil {
			if _, err := s.filterSQL(ctx, ws.ID, *in.Filters); err != nil {
				return err
			}
			filters, _ = json.Marshal(*in.Filters)
		}
		if in.Columns != nil {
			if cols, err = s.cleanColumns(ctx, ws.ID, *in.Columns); err != nil {
				return err
			}
		}
		u, err := q.UpdateSavedView(ctx, store.UpdateSavedViewParams{WorkspaceID: ws.ID, ID: id, Name: name, Filters: filters, Columns: cols, At: s.now()})
		if err != nil {
			return err
		}
		out = savedView(u, counted{})
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "contact_view.update", TargetType: "saved_view", TargetID: id.String(), TargetName: name,
			Before: map[string]any{"name": v.Name, "filters": v.Filters, "columns": v.Columns}, After: map[string]any{"name": name, "filters": filters, "columns": cols}})
	})
	return out, err
}

// ownView reads a view the caller may change: a shared one, or their own.
func (s *Service) ownView(ctx context.Context, q *store.Queries, wsID, id uuid.UUID) (store.SavedView, error) {
	v, err := q.GetSavedView(ctx, store.GetSavedViewParams{WorkspaceID: wsID, ID: id})
	if err != nil {
		return v, api.MapError(err, "Saved view")
	}
	if v.UserID != nil {
		if me := actorID(ctx); me == nil || *me != *v.UserID {
			return v, api.NotFound("Saved view")
		}
	}
	return v, nil
}

// DeleteView deletes a view.
func (s *Service) DeleteView(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		v, err := s.ownView(ctx, q, ws.ID, id)
		if err != nil {
			return err
		}
		if _, err := q.DeleteSavedView(ctx, store.DeleteSavedViewParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "contact_view.delete", TargetType: "saved_view", TargetID: id.String(), TargetName: v.Name, Before: v.Filters})
	})
}
