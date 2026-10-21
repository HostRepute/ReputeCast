package agency

import (
	"context"
	"time"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Staff hear about incidents and approvals in their clients through their
// memberships there, following their own notification preferences. The
// Agency area's bell gathers them across the agency's workspaces; marking
// one read goes through its workspace's notification center.

// AgencyNotification is one of a staff member's notifications.
type AgencyNotification struct {
	ID        string             `json:"id"`
	Workspace AgencyWorkspaceRef `json:"workspace"`
	Kind      string             `json:"kind"`
	Severity  string             `json:"severity" enum:"info,ok,warn,bad"`
	Title     string             `json:"title"`
	Summary   string             `json:"summary"`
	Link      string             `json:"link"`
	CreatedAt time.Time          `json:"createdAt"`
	Read      bool               `json:"read"`
}

// AgencyNotificationList is the bell's latest notifications.
type AgencyNotificationList struct {
	Items  []AgencyNotification `json:"items"`
	Unread int64                `json:"unread"`
}

// Notifications returns the caller's latest notifications in the agency's
// workspaces they're a member of, unread first.
func (s *Service) Notifications(ctx context.Context, limit int) (AgencyNotificationList, error) {
	me, _, err := s.require(ctx, authz.AgencyView)
	if err != nil {
		return AgencyNotificationList{}, err
	}
	out := AgencyNotificationList{Items: []AgencyNotification{}}
	set, err := s.workspacesOf(ctx, s.Q, me)
	if err != nil {
		return out, err
	}
	ids := set.ids()
	u := api.UserFrom(ctx)
	if limit <= 0 || limit > 50 {
		limit = 6
	}
	rows, err := s.Q.AgencyNotifications(ctx, store.AgencyNotificationsParams{UserID: u.ID, WorkspaceIds: ids, Lim: int32(limit)})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		if r.WorkspaceID == nil {
			continue
		}
		w, ok := set.find(*r.WorkspaceID)
		if !ok {
			continue
		}
		out.Items = append(out.Items, AgencyNotification{ID: r.ID.String(), Workspace: wsRef(w), Kind: r.Kind, Severity: r.Severity, Title: r.Title, Summary: r.Summary,
			Link: r.Link, CreatedAt: r.CreatedAt, Read: r.Read})
	}
	out.Unread, err = s.Q.AgencyUnreadNotifications(ctx, store.AgencyUnreadNotificationsParams{UserID: u.ID, WorkspaceIds: ids})
	return out, err
}
