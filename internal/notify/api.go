package notify

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/store"
)

// signedIn is a person (never an API key or agent): notifications and their
// preferences are personal.
var signedIn = api.Access{Kind: api.AccessUser, HumansOnly: true}

// member is any member of the workspace in the path, in person.
var member = api.WorkspaceHuman("")

type listNotificationsInput struct {
	api.WorkspaceInput
	Filter string `query:"filter" enum:"all,unread,approvals,sending,providers,team,hostrepute,transactional,earn,website,grow" default:"all" doc:"Notification center tab"`
	Cursor string `query:"cursor" doc:"Opaque cursor from a previous page"`
	Limit  int    `query:"limit" default:"50" minimum:"1" maximum:"100"`
}

type notificationInput struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid"`
}

// NotificationsMarked says how many notifications were marked as read.
type NotificationsMarked struct {
	Marked int64 `json:"marked"`
}

type preferencesQuery struct {
	Workspace string `query:"workspace" doc:"Workspace slug whose System-stream provider is shown"`
}

// Register adds the notification center and preference operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Notifications"}
	api.Get(h, api.Op{ID: "list-notifications", Path: "/workspaces/{workspace}/notifications", Summary: "Your notifications in this workspace, unread first", Tags: tags, Access: member},
		func(ctx context.Context, in *listNotificationsInput) (*api.Body[NotificationList], error) {
			r, err := s.List(ctx, in.Filter, in.Cursor, in.Limit)
			return api.OK(r), err
		})

	api.Post(h, api.Op{ID: "read-notification", Path: "/workspaces/{workspace}/notifications/{id}/read", Summary: "Mark a notification as read", Tags: tags, Access: member, Status: http.StatusNoContent},
		func(ctx context.Context, in *notificationInput) (*api.Empty, error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return nil, api.NotFound("Notification")
			}
			return api.NoContent, s.MarkRead(ctx, id)
		})

	api.Post(h, api.Op{ID: "read-all-notifications", Path: "/workspaces/{workspace}/notifications/read-all", Summary: "Mark every notification in this workspace as read", Tags: tags, Access: member},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[NotificationsMarked], error) {
			n, err := s.MarkAllRead(ctx)
			return api.OK(NotificationsMarked{Marked: n}), err
		})

	api.Get(h, api.Op{ID: "get-notification-preferences", Path: "/me/notification-preferences", Summary: "Your notification preferences: in-app and email per event type", Tags: tags, Access: signedIn},
		func(ctx context.Context, in *preferencesQuery) (*api.Body[NotificationPreferences], error) {
			u := api.UserFrom(ctx)
			var wsID *uuid.UUID
			if in.Workspace != "" {
				if ws, err := s.Q.GetWorkspaceBySlug(ctx, in.Workspace); err == nil {
					if m, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: ws.ID, UserID: u.ID}); err == nil && m.Status == "active" {
						id := ws.ID
						wsID = &id
					}
				}
			}
			r, err := s.Preferences(ctx, *u, wsID)
			return api.OK(r), err
		})

	api.Put(h, api.Op{ID: "save-notification-preferences", Path: "/me/notification-preferences", Summary: "Save your notification preferences", Tags: tags, Access: signedIn, Status: http.StatusNoContent},
		func(ctx context.Context, in *struct{ Body NotificationPreferencesInput }) (*api.Empty, error) {
			return api.NoContent, s.SavePreferences(ctx, *api.UserFrom(ctx), in.Body)
		})
}
