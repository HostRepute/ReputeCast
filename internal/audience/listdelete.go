package audience

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Deleting a list (Dialogs board) shows its impact first. A deleted list
// disappears from every screen and sends skip it; after RestoreDays days the
// maintenance job removes it and its memberships for good.

// RestoreDays is how long a deleted list can be restored.
const RestoreDays = 30

// ListImpactItem is a form, automation or campaign that uses a list.
type ListImpactItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListImpact is what deleting a list affects.
type ListImpact struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Members are the list's subscribers. Elsewhere of them are subscribed
	// to another list and stay subscribed there; the rest become contacts
	// without a subscription.
	Members   int64 `json:"members"`
	Elsewhere int64 `json:"elsewhere"`
	// OtherLists are the other lists those subscribers are on, most shared
	// first (up to three); OtherListCount counts them all.
	OtherLists     []string `json:"otherLists"`
	OtherListCount int      `json:"otherListCount"`
	// Forms, Automations and Campaigns are what deleting the list pauses.
	Forms       []ListImpactItem `json:"forms"`
	Automations []ListImpactItem `json:"automations"`
	Campaigns   []ListImpactItem `json:"campaigns"`
}

// ListRestore is what restoring a list turned back on.
type ListRestore struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Forms and Automations are live again; Campaigns are back on their
	// schedule.
	Forms       []string `json:"forms"`
	Automations []string `json:"automations"`
	Campaigns   []string `json:"campaigns"`
	// Left were paused by the delete but changed since, or their send time
	// passed: they stay as they are.
	Left []string `json:"left"`
}

// deletePaused is what deleting a list paused, kept on the list for a
// restore. At is when: what was paused carries that time, so a restore
// only resumes what nobody touched since.
type deletePaused struct {
	At          time.Time        `json:"at"`
	Forms       []pausedItem     `json:"forms,omitempty"`
	Automations []pausedItem     `json:"automations,omitempty"`
	Campaigns   []pausedCampaign `json:"campaigns,omitempty"`
}

type pausedItem struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type pausedCampaign struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	ScheduledAt time.Time `json:"scheduledAt"`
	// Status is where coming off the schedule left it: approved when its
	// approval stood, otherwise draft.
	Status string `json:"status"`
}

// listInScope refuses a list outside the person's list scope: a member
// limited to some lists manages only those.
func listInScope(ctx context.Context, id uuid.UUID) error {
	if scopeAllowsList(ctx, id) {
		return nil
	}
	return api.Err(http.StatusForbidden, api.CodePermissionDenied, "Your role is limited to other lists.")
}

// ListImpact reports what deleting a list would affect.
func (s *Service) ListImpact(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ListImpact, error) {
	l, err := s.Q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ListImpact{}, api.MapError(err, "List")
	}
	if err := listInScope(ctx, id); err != nil {
		return ListImpact{}, err
	}
	out := ListImpact{ID: l.ID.String(), Name: l.Name, OtherLists: []string{}, Forms: []ListImpactItem{}, Automations: []ListImpactItem{}, Campaigns: []ListImpactItem{}}
	m, err := s.Q.ListImpactMembers(ctx, l.ID)
	if err != nil {
		return ListImpact{}, err
	}
	out.Members, out.Elsewhere = m.Members, m.Elsewhere
	others, err := s.Q.ListImpactOtherLists(ctx, l.ID)
	if err != nil {
		return ListImpact{}, err
	}
	// Lists outside the person's scope are counted, not named.
	out.OtherListCount = len(others)
	for _, o := range others {
		if len(out.OtherLists) < 3 && scopeAllowsList(ctx, o.ID) {
			out.OtherLists = append(out.OtherLists, o.Name)
		}
	}
	d, err := dependents(ctx, s.Q, ws.ID, l.ID)
	if err != nil {
		return ListImpact{}, err
	}
	for _, f := range d.forms {
		out.Forms = append(out.Forms, ListImpactItem{ID: f.ID.String(), Name: f.Name})
	}
	for _, a := range d.automations {
		out.Automations = append(out.Automations, ListImpactItem{ID: a.ID.String(), Name: a.Name})
	}
	for _, c := range d.campaigns {
		out.Campaigns = append(out.Campaigns, ListImpactItem{ID: c.ID.String(), Name: c.Name})
	}
	return out, nil
}

type listDependents struct {
	forms       []store.ListImpactFormsRow
	automations []store.ListImpactAutomationsRow
	campaigns   []store.ListImpactCampaignsRow
}

func dependents(ctx context.Context, q *store.Queries, wsID, id uuid.UUID) (listDependents, error) {
	var d listDependents
	var err error
	if d.forms, err = q.ListImpactForms(ctx, store.ListImpactFormsParams{WorkspaceID: wsID, ID: id}); err != nil {
		return d, err
	}
	if d.automations, err = q.ListImpactAutomations(ctx, store.ListImpactAutomationsParams{WorkspaceID: wsID, ID: id.String()}); err != nil {
		return d, err
	}
	d.campaigns, err = q.ListImpactCampaigns(ctx, store.ListImpactCampaignsParams{WorkspaceID: wsID, ID: id.String()})
	return d, err
}

// DeleteList deletes a list. Its live signup forms and the live automations
// it starts pause, and scheduled campaigns sending to it come off the
// schedule (back to approved when their approval stands, otherwise draft).
// Contacts, their other lists and suppressions stay. It can be restored for
// RestoreDays days.
func (s *Service) DeleteList(ctx context.Context, ws *store.Workspace, id uuid.UUID) error {
	now := s.now().Truncate(time.Microsecond)
	by := actorID(ctx)
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		l, err := q.LockList(ctx, store.LockListParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "List")
		}
		if err := listInScope(ctx, id); err != nil {
			return err
		}
		m, err := q.ListImpactMembers(ctx, id)
		if err != nil {
			return err
		}
		d, err := dependents(ctx, q, ws.ID, id)
		if err != nil {
			return err
		}
		reason := fmt.Sprintf("List %q was deleted", l.Name)
		paused := deletePaused{At: now}
		for _, f := range d.forms {
			n, err := q.ListPauseForm(ctx, store.ListPauseFormParams{By: by, At: now, WorkspaceID: ws.ID, ID: f.ID})
			if err != nil {
				return err
			}
			if n == 0 {
				continue
			}
			paused.Forms = append(paused.Forms, pausedItem{ID: f.ID, Name: f.Name})
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.pause", TargetType: "form", TargetID: f.ID.String(), TargetName: f.Name,
				Before: map[string]any{"status": "live"}, After: map[string]any{"status": "paused"}, Reason: reason}); err != nil {
				return err
			}
		}
		for _, a := range d.automations {
			n, err := q.ListPauseAutomation(ctx, store.ListPauseAutomationParams{At: &now, By: by, Reason: reason, WorkspaceID: ws.ID, ID: a.ID})
			if err != nil {
				return err
			}
			if n == 0 {
				continue
			}
			paused.Automations = append(paused.Automations, pausedItem{ID: a.ID, Name: a.Name})
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "automation.pause", TargetType: "automation", TargetID: a.ID.String(), TargetName: a.Name,
				Before: map[string]any{"status": "live"}, After: map[string]any{"status": "paused"}, Reason: reason}); err != nil {
				return err
			}
		}
		for _, c := range d.campaigns {
			if c.ScheduledAt == nil {
				continue
			}
			status := "draft"
			cid := c.ID
			if r, err := q.ApprovalOpenForCampaign(ctx, &cid); err == nil && r.Status == "approved" {
				status = "approved"
			} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			n, err := q.ListUnscheduleCampaign(ctx, store.ListUnscheduleCampaignParams{Status: status, By: by, At: now, WorkspaceID: ws.ID, ID: c.ID})
			if err != nil {
				return err
			}
			if n == 0 {
				continue
			}
			if err := q.ApprovalResolveHold(ctx, store.ApprovalResolveHoldParams{ResolvedAt: &now, Resolution: "cancelled", ResolvedBy: by, CampaignID: c.ID}); err != nil {
				return err
			}
			paused.Campaigns = append(paused.Campaigns, pausedCampaign{ID: c.ID, Name: c.Name, ScheduledAt: *c.ScheduledAt, Status: status})
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.unscheduled", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
				Before: map[string]any{"status": "scheduled", "scheduled_at": c.ScheduledAt.UTC()}, After: map[string]any{"status": status}, Reason: reason}); err != nil {
				return err
			}
		}
		raw, err := json.Marshal(paused)
		if err != nil {
			return err
		}
		if _, err := q.SoftDeleteList(ctx, store.SoftDeleteListParams{At: &now, Paused: raw, WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "list.delete", TargetType: "list", TargetID: id.String(), TargetName: l.Name,
			Before: map[string]any{"name": l.Name, "subscribed": m.Members},
			After: map[string]any{"restorableUntil": now.AddDate(0, 0, RestoreDays).UTC(), "paused": map[string]any{
				"forms": names(paused.Forms), "automations": names(paused.Automations), "campaigns": campaignNames(paused.Campaigns),
			}}})
	})
}

// RestoreList brings back a list deleted in the last RestoreDays days with
// its memberships, and turns back on what deleting it paused, unless that
// changed since: forms and automations go live again, campaigns go back on
// their schedule when their send time hasn't passed.
func (s *Service) RestoreList(ctx context.Context, ws *store.Workspace, id uuid.UUID) (ListRestore, error) {
	now := s.now().Truncate(time.Microsecond)
	by := actorID(ctx)
	out := ListRestore{Forms: []string{}, Automations: []string{}, Campaigns: []string{}, Left: []string{}}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		l, err := q.LockDeletedList(ctx, store.LockDeletedListParams{WorkspaceID: ws.ID, ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			if live, err := q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: id}); err == nil {
				return api.Conflict(fmt.Sprintf("%q isn't deleted.", live.Name))
			}
			return api.NotFound("List")
		}
		if err != nil {
			return err
		}
		if err := listInScope(ctx, id); err != nil {
			return err
		}
		if l.DeletedAt == nil || l.DeletedAt.Before(now.AddDate(0, 0, -RestoreDays)) {
			return api.Conflict(fmt.Sprintf("Deleted lists can be restored for %d days; %q is past that.", RestoreDays, l.Name))
		}
		taken, err := q.ListNameTaken(ctx, store.ListNameTakenParams{WorkspaceID: ws.ID, Name: l.Name, ExceptID: l.ID})
		if err != nil {
			return err
		}
		if taken {
			return api.Conflict(fmt.Sprintf("Another list is called %q now. Rename it, then restore this one.", l.Name))
		}
		var p deletePaused
		_ = json.Unmarshal(l.DeletePaused, &p)
		r, err := q.RestoreList(ctx, l.ID)
		if err != nil {
			return err
		}
		out.ID, out.Name = r.ID.String(), r.Name
		reason := fmt.Sprintf("List %q was restored", l.Name)
		for _, f := range p.Forms {
			n, err := q.ListResumeForm(ctx, store.ListResumeFormParams{By: by, At: now, WorkspaceID: ws.ID, ID: f.ID, PausedAt: p.At})
			if err != nil {
				return err
			}
			if n == 0 {
				out.Left = append(out.Left, f.Name)
				continue
			}
			out.Forms = append(out.Forms, f.Name)
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "form.resume", TargetType: "form", TargetID: f.ID.String(), TargetName: f.Name,
				Before: map[string]any{"status": "paused"}, After: map[string]any{"status": "live"}, Reason: reason}); err != nil {
				return err
			}
		}
		for _, a := range p.Automations {
			n, err := q.ListResumeAutomation(ctx, store.ListResumeAutomationParams{By: by, At: now, WorkspaceID: ws.ID, ID: a.ID, PausedAt: &p.At})
			if err != nil {
				return err
			}
			if n == 0 {
				out.Left = append(out.Left, a.Name)
				continue
			}
			out.Automations = append(out.Automations, a.Name)
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "automation.resume", TargetType: "automation", TargetID: a.ID.String(), TargetName: a.Name,
				Before: map[string]any{"status": "paused"}, After: map[string]any{"status": "live"}, Reason: reason}); err != nil {
				return err
			}
		}
		for _, c := range p.Campaigns {
			// A send time that passed while the list was deleted is the
			// author's to set again.
			var n int64
			if c.ScheduledAt.After(now.Add(time.Minute)) {
				at := c.ScheduledAt
				if n, err = q.ListRescheduleCampaign(ctx, store.ListRescheduleCampaignParams{ScheduledAt: &at, By: by, At: now, WorkspaceID: ws.ID, ID: c.ID, Status: c.Status, UnscheduledAt: p.At}); err != nil {
					return err
				}
			}
			if n == 0 {
				out.Left = append(out.Left, c.Name)
				continue
			}
			out.Campaigns = append(out.Campaigns, c.Name)
			if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.scheduled", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
				Before: map[string]any{"status": c.Status}, After: map[string]any{"status": "scheduled", "scheduled_at": c.ScheduledAt.UTC()}, Reason: reason}); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "list.restore", TargetType: "list", TargetID: id.String(), TargetName: l.Name,
			Before: map[string]any{"deletedAt": l.DeletedAt.UTC()},
			After:  map[string]any{"resumed": map[string]any{"forms": out.Forms, "automations": out.Automations, "campaigns": out.Campaigns}, "left": out.Left}})
	})
	return out, err
}

func names(items []pausedItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Name)
	}
	return out
}

func campaignNames(items []pausedCampaign) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Name)
	}
	return out
}
