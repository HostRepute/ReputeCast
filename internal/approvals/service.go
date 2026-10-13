package approvals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service holds the approvals and Review & send dependencies.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Log   *slog.Logger
	Now   func() time.Time
	// Readiness runs Send Readiness.
	Readiness *readiness.Service
	// Campaigns restores versions ("Restore v9 subject").
	Campaigns *campaigns.Service
	// Sending starts and schedules sends (App.Sending, read at call time).
	Sending func() sending.Starter
	// Automations publishes an automation once its approval is granted.
	Automations *automations.Service
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) starter() sending.Starter {
	if s.Sending != nil {
		if st := s.Sending(); st != nil {
			return st
		}
	}
	return sending.Unavailable{}
}

// member is an active member of the workspace.
type member struct {
	ID          uuid.UUID
	Name        string
	Email       string
	Color       string
	RoleKey     string
	RoleName    string
	Perms       authz.Set
	LastActive  *time.Time
	HasPassword bool
}

func (s *Service) members(ctx context.Context, q *store.Queries, wsID uuid.UUID) ([]member, error) {
	rows, err := q.ApprovalMembers(ctx, wsID)
	if err != nil {
		return nil, err
	}
	out := make([]member, 0, len(rows))
	for _, r := range rows {
		out = append(out, member{ID: r.ID, Name: r.Name, Email: r.Email, Color: r.AvatarColor, RoleKey: r.RoleKey, RoleName: r.RoleName,
			Perms: authz.Effective(r.RoleKey, r.Builtin, r.Permissions), LastActive: r.LastActiveAt, HasPassword: r.HasPassword})
	}
	return out, nil
}

// approversOf are the members holding campaigns:approve (Owners first).
func approversOf(ms []member) []member {
	out := []member{}
	for _, m := range ms {
		if m.Perms.Has(authz.CampaignsApprove) {
			out = append(out, m)
		}
	}
	return out
}

// ApprovalPerson is a member as approval screens show them.
type ApprovalPerson struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Initials string `json:"initials"`
	Color    string `json:"color"`
	Role     string `json:"role"`
	// Presence is "online" (active in the last 15 minutes) or "active 2 h ago".
	Presence string `json:"presence,omitempty"`
}

// Initials of a name: "Marcus Chen" → "MC".
func Initials(name string) string {
	parts := strings.Fields(name)
	out := ""
	for _, p := range parts {
		r := []rune(p)
		if len(r) > 0 {
			out += strings.ToUpper(string(r[0]))
		}
		if len([]rune(out)) == 2 {
			break
		}
	}
	return out
}

func (s *Service) person(m member) ApprovalPerson {
	p := ApprovalPerson{ID: m.ID.String(), Name: m.Name, Email: m.Email, Initials: Initials(m.Name), Color: m.Color, Role: m.RoleName}
	if m.LastActive != nil {
		ago := s.now().Sub(*m.LastActive)
		switch {
		case ago < 15*time.Minute:
			p.Presence = "online"
		case ago < time.Hour:
			p.Presence = fmt.Sprintf("active %d min ago", int(ago.Minutes()))
		case ago < 24*time.Hour:
			p.Presence = fmt.Sprintf("active %d h ago", int(ago.Hours()))
		default:
			p.Presence = fmt.Sprintf("active %d days ago", int(ago.Hours()/24))
		}
	}
	return p
}

func memberByID(ms []member, id *uuid.UUID) *member {
	if id == nil {
		return nil
	}
	for i := range ms {
		if ms[i].ID == *id {
			return &ms[i]
		}
	}
	return nil
}

// current describes the campaign as it stands, in snapshot terms.
func (s *Service) current(ctx context.Context, ws *store.Workspace, c store.ReadinessCampaignRow) (Current, error) {
	cur := Current{Version: int(c.Version), ContentHash: c.ContentHash, Subject: c.Subject, Preheader: c.Preheader, From: fromLine(c.SenderName, c.SenderEmail),
		MJML: c.Mjml, HTML: c.Html, Audience: c.Audience, Schedule: c.Schedule, UpdatedBy: c.UpdatedByName, AudienceCount: -1}
	if c.SenderID != nil {
		cur.SenderID = c.SenderID.String()
	}
	aud, err := s.Readiness.Audience(ctx, ws, c.Audience)
	if err != nil {
		return cur, err
	}
	cur.AudienceNames = aud.Lists
	if !aud.Empty {
		cur.AudienceCount = aud.Final
	}
	rc, err := s.Readiness.Route(ctx, ws, c.RouteOverride)
	if err != nil {
		return cur, err
	}
	cur.RouteKey, cur.SplitLabel = rc.Key, rc.Label
	return cur, nil
}

func fromLine(name, email string) string {
	switch {
	case email == "":
		return ""
	case name == "":
		return email
	}
	return name + " <" + email + ">"
}

// CampaignChanged re-checks the open approval of a campaign after a change
// (the composer's and Send Readiness's change hooks call it).
func (s *Service) CampaignChanged(ctx context.Context, ws *store.Workspace, id uuid.UUID) {
	if _, err := s.Revalidate(ctx, ws, id); err != nil && s.Log != nil {
		s.Log.WarnContext(ctx, "approval revalidation failed", "campaign", id, "err", err)
	}
}

// Revalidate compares the open request of a campaign with the campaign and
// invalidates it when anything it covers changed or the audience moved more
// than 10%. It returns the invalidated request's changes (none when valid).
func (s *Service) Revalidate(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) ([]ApprovalChange, error) {
	open, err := s.Q.ApprovalOpenForCampaign(ctx, &campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c, err := s.Readiness.Campaign(ctx, ws, campaignID)
	if err != nil {
		return nil, err
	}
	switch c.Status {
	case "draft", "approval", "approved", "scheduled":
	default:
		return nil, nil // the send started: the approval did its job
	}
	cur, err := s.current(ctx, ws, c)
	if err != nil {
		return nil, err
	}
	snap := ParseSnapshot(open.Snapshot)
	changes := Diff(snap, cur, state.Location(ws.Timezone))
	if len(changes) == 0 {
		return nil, nil
	}
	return changes, s.invalidate(ctx, ws, open, c, changes)
}

func (s *Service) invalidate(ctx context.Context, ws *store.Workspace, r store.ApprovalRequest, c store.ReadinessCampaignRow, changes []ApprovalChange) error {
	now := s.now()
	reason := InvalidationReason(changes)
	cb, _ := json.Marshal(changes)
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return err
	}
	editor := memberByID(ms, c.UpdatedBy)
	var by *uuid.UUID
	byName := ""
	if editor != nil {
		by, byName = &editor.ID, editor.Name
	}
	decider := ""
	if r.DecidedBy != nil {
		if m := memberByID(ms, r.DecidedBy); m != nil {
			decider = m.Name
		}
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		// Only an open request can be invalidated; a concurrent change may
		// already have done it.
		locked, err := q.ApprovalLock(ctx, store.ApprovalLockParams{WorkspaceID: ws.ID, ID: r.ID})
		if err != nil || (locked.Status != "pending" && locked.Status != "approved") {
			return err
		}
		if err := q.ApprovalInvalidate(ctx, store.ApprovalInvalidateParams{ID: r.ID, InvalidatedAt: &now, InvalidatedReason: reason, Changes: cb, InvalidatedBy: by,
			InvalidatedFrom: locked.Status}); err != nil {
			return err
		}
		// The campaign goes back to draft; a scheduled send keeps its slot but
		// won't start without a new approval (the send-time check holds it).
		if err := q.ApprovalSetCampaignStatus(ctx, store.ApprovalSetCampaignStatusParams{Status: "draft", Now: now, ID: c.ID, FromStatuses: []string{"approval", "approved"}}); err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "approval.invalidated", TargetType: "approval_request", TargetID: r.ID.String(), TargetName: c.Name,
			Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}, Reason: reason,
			Before: map[string]any{"status": locked.Status}, After: map[string]any{"status": "invalidated", "changes": changes},
			Metadata: map[string]any{"campaign_id": c.ID.String(), "edited_by": byName, "summary": reason}}); err != nil {
			return err
		}
		// Author and approvers hear about it.
		link := "/w/" + ws.Slug + "/approvals/" + r.ID.String()
		whose := "the approval"
		if decider != "" {
			whose = decider + "'s approval"
		}
		n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindApprovalInvalidated, Severity: "warn", Title: "Approval invalidated",
			Summary: fmt.Sprintf("“%s”: %s after %s", c.Name, strings.TrimSuffix(reason, " after approval"), whose),
			Body:    fmt.Sprintf("“%s”: %s after %s. A new approval is needed before it can be sent.", c.Name, strings.TrimSuffix(reason, " after approval"), whose),
			Link:    link, Data: map[string]any{"request_id": r.ID.String(), "campaign_id": c.ID.String()}}
		seen := map[uuid.UUID]bool{}
		targets := []uuid.UUID{}
		if r.RequestedBy != nil {
			targets = append(targets, *r.RequestedBy)
		}
		for _, a := range approversOf(ms) {
			targets = append(targets, a.ID)
		}
		for _, id := range targets {
			if seen[id] || (by != nil && id == *by) {
				continue
			}
			seen[id] = true
			if err := notify.ToUser(ctx, tx, id, n); err != nil {
				return err
			}
		}
		return nil
	})
}

// open returns the open (pending or approved) request of a campaign, nil when none.
func (s *Service) open(ctx context.Context, campaignID uuid.UUID) (*store.ApprovalRequest, error) {
	r, err := s.Q.ApprovalOpenForCampaign(ctx, &campaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}
