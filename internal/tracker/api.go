package tracker

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// TrackerSitePath names a tracked site.
type TrackerSitePath struct {
	api.WorkspaceInput
	Site string `path:"site" format:"uuid" doc:"Tracked site id"`
}

func (p TrackerSitePath) id(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(p.Site)
	if err != nil {
		return uuid.Nil, siteGone(ctx)
	}
	return id, nil
}

// Register adds the Website tracking operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Website tracking"}
	view, manage := api.Workspace(authz.IntegrationsView), api.WorkspaceHuman(authz.IntegrationsManage)
	base := "/workspaces/{workspace}/tracker/sites"

	api.Get(h, api.Op{ID: "list-tracker-sites", Path: base, Summary: "List the sites the tracker runs on",
		Description: "Each site's snippets, events today and the origins refused in the last 24 hours. Identify secrets are never returned.", Tags: tags, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[TrackerSiteList], error) {
			out, err := s.Sites(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-tracker-site", Path: base, Summary: "Add a site to track (logged)",
		Description: "The answer holds the site's identify secret, shown this once.", Tags: tags, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body TrackerSiteInput
		}) (*api.Body[TrackerSiteCreated], error) {
			out, err := s.CreateSite(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			if err != nil {
				return nil, err
			}
			return api.OK(out), nil
		})
	api.Put(h, api.Op{ID: "update-tracker-site", Path: base + "/{site}", Summary: "Change a tracked site's name, store and consent mode (logged)",
		Description: "The origin can't change: add another site for another address.", Tags: tags, Access: manage},
		func(ctx context.Context, in *struct {
			TrackerSitePath
			Body TrackerSiteUpdate
		}) (*api.Body[TrackerSite], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.UpdateSite(ctx, api.WorkspaceFrom(ctx), id, in.Body)
			if err != nil {
				return nil, err
			}
			return api.OK(out), nil
		})
	api.Delete(h, api.Op{ID: "delete-tracker-site", Path: base + "/{site}", Summary: "Stop tracking a site (logged)",
		Description: "Its events stop at once and the ones kept go with the retention.", Tags: tags, Access: manage},
		func(ctx context.Context, in *TrackerSitePath) (*api.Empty, error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			if err := s.DeleteSite(ctx, api.WorkspaceFrom(ctx), id); err != nil {
				return nil, err
			}
			return api.NoContent, nil
		})
	api.Post(h, api.Op{ID: "rotate-tracker-site-secret", Path: base + "/{site}/secret", Summary: "Give a site a new identify secret (logged)",
		Description: "The old secret stops working at once; the new one is shown this once.", Tags: tags, Access: manage},
		func(ctx context.Context, in *TrackerSitePath) (*api.Body[TrackerSecret], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.RotateSecret(ctx, api.WorkspaceFrom(ctx), id)
			if err != nil {
				return nil, err
			}
			return api.OK(out), nil
		})
	api.Get(h, api.Op{ID: "tracker-site-recent-events", Path: base + "/{site}/events/recent", Summary: "A tracked site's last 20 events",
		Description: "What happened, on which page or product, whether a known contact or a visitor, and when. Never an address.", Tags: tags, Access: view},
		func(ctx context.Context, in *TrackerSitePath) (*api.Body[TrackerRecentEvents], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.RecentEvents(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
}
