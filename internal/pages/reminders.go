package pages

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// A webinar page's reminder automation (the builder's "Create the
// reminder automation?"): the registration form writes the event's date
// to a contact date field (dmarc_live_at); a draft flow with a date-field
// trigger on it sends the joining link the day before and an hour before.

// ReminderRequest is the flow to make (internal/automations makes it).
type ReminderRequest struct {
	Name, Title, Field, At, Link, When string
}

// PageReminderFlow is the reminder automation.
type PageReminderFlow struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// PageReminders is the offer and what exists.
type PageReminders struct {
	// Available: the page has event details with a date field.
	Available bool              `json:"available"`
	Field     string            `json:"field"`
	Title     string            `json:"title"`
	Reminders string            `json:"reminders"`
	Flow      *PageReminderFlow `json:"automation,omitempty"`
}

// Reminders reads a page's reminder offer.
func (s *Service) Reminders(ctx context.Context, ws *store.Workspace, id uuid.UUID) (PageReminders, error) {
	pg, err := s.get(ctx, s.Q, ws, id)
	if err != nil {
		return PageReminders{}, err
	}
	ev, ok := FirstEvent(pg.Html)
	if !ok || ev.Field == "" {
		return PageReminders{}, nil
	}
	out := PageReminders{Available: true, Field: ev.Field, Title: ev.Title, Reminders: ev.Reminders}
	if out.Title == "" {
		out.Title = pg.Name
	}
	var f PageReminderFlow
	var fid uuid.UUID
	err = s.Pool.QueryRow(ctx, `SELECT id, name, status FROM automations WHERE workspace_id = $1 AND deleted_at IS NULL
		AND trigger->>'kind' = 'date_field' AND trigger->>'field' = $2 ORDER BY created_at DESC LIMIT 1`, ws.ID, ev.Field).Scan(&fid, &f.Name, &f.Status)
	switch {
	case err == nil:
		f.ID = fid.String()
		out.Flow = &f
	case !errors.Is(err, pgx.ErrNoRows):
		return out, err
	}
	return out, nil
}

// CreateReminders makes the reminder automation, as a draft.
func (s *Service) CreateReminders(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (PageReminders, error) {
	cur, err := s.Reminders(ctx, ws, id)
	if err != nil {
		return cur, err
	}
	if !cur.Available {
		return cur, api.Conflict(word(ctx, "reminders.noEvent", nil))
	}
	if cur.Flow != nil {
		return cur, api.Err(http.StatusConflict, "reminders_exist", word(ctx, "reminders.exist", map[string]any{"name": cur.Flow.Name})).WithMeta("automation", cur.Flow.ID)
	}
	if s.ReminderFlow == nil {
		return cur, api.Err(http.StatusServiceUnavailable, "unavailable", word(ctx, "reminders.unavailable", nil))
	}
	pg, err := s.get(ctx, s.Q, ws, id)
	if err != nil {
		return cur, err
	}
	ev, _ := FirstEvent(pg.Html)
	start, ok := Local(ev.Date+"T"+ev.Start, ev.TZ)
	if !ok {
		return cur, api.Invalid("body", word(ctx, "reminders.noTime", nil))
	}
	loc := time.UTC
	if l, err := time.LoadLocation(ws.Timezone); err == nil && ws.Timezone != "" {
		loc = l
	}
	local := start.In(loc)
	name := strings.TrimSpace(cur.Title)
	if len([]rune(name)) > 60 {
		name = string([]rune(name)[:60])
	}
	req := ReminderRequest{Name: word(ctx, "reminders.name", map[string]any{"title": name}), Title: cur.Title, Field: ev.Field,
		At: local.Format("15:04"), Link: ev.Link, When: start.In(start.Location()).Format("2 January 2006, 15:04 MST")}
	if _, err := s.ReminderFlow(ctx, ws, actor(p), req); err != nil {
		return cur, err
	}
	return s.Reminders(ctx, ws, id)
}

// RecordEventDate writes a webinar's date to a registrant's date field
// (dmarc_live_at), when the workspace has that date field. The date is the
// session's start in the workspace's time zone: the reminder automation's
// date-field trigger counts days and times there.
func (s *Service) RecordEventDate(ctx context.Context, ws *store.Workspace, contact uuid.UUID, ev EventInfo) error {
	field := ev.Field
	start, ok := Local(ev.Date+"T"+ev.Start, ev.TZ)
	if !fieldKeyRe.MatchString(field) || !ok {
		return nil
	}
	loc := time.UTC
	if l, err := time.LoadLocation(ws.Timezone); err == nil && ws.Timezone != "" {
		loc = l
	}
	date := start.In(loc).Format("2006-01-02")
	_, err := s.Pool.Exec(ctx, `UPDATE contacts SET attributes = attributes || jsonb_build_object($3::text, $4::text), updated_at = now()
		WHERE id = $1 AND workspace_id = $2
		  AND EXISTS (SELECT 1 FROM custom_fields WHERE workspace_id = $2 AND key = $3 AND type = 'date')`, contact, ws.ID, field, date)
	return err
}

func registerReminders(h huma.API, s *Service) {
	tags := []string{"Pages"}
	path := "/workspaces/{workspace}/pages/{page}/reminder-automation"
	api.Get(h, api.Op{ID: "get-page-reminders", Path: path, Summary: "A webinar page's reminder automation: the offer, or the flow made", Tags: tags,
		Access: api.Workspace(authz.WebsiteView)},
		func(ctx context.Context, in *PagePath) (*api.Body[PageReminders], error) {
			out, err := s.Reminders(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-page-reminders", Path: path,
		Summary: "Make the reminder automation of a webinar page: a draft flow with a date-field trigger, the day before and an hour before",
		Tags:    tags, Access: api.Workspace(authz.AutomationsEdit), Status: http.StatusCreated},
		func(ctx context.Context, in *PagePath) (*api.Body[PageReminders], error) {
			out, err := s.CreateReminders(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.id())
			return api.OK(out), err
		})
}
