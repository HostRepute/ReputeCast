package pages

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"

	"github.com/jackc/pgx/v5"
)

type overviewInput struct {
	api.WorkspaceInput
	Period string `query:"period" enum:"7d,30d,90d" default:"30d"`
}

// PagePath is a page of the workspace (exported: Huma reads embedded path
// structs only when their type is exported).
type PagePath struct {
	api.WorkspaceInput
	Page string `path:"page" format:"uuid" doc:"The page's id"`
}

func (p PagePath) id() uuid.UUID { return uuid.MustParse(p.Page) }

type createInput struct {
	api.WorkspaceInput
	Body PageCreate
}

type contentInput struct {
	PagePath
	Body PageContentInput
}

type settingsInput struct {
	PagePath
	Body PageSettingsInput
}

type publishInput struct {
	PagePath
	Body PagePublishInput
}

type unpublishInput struct {
	PagePath
	Body PageUnpublishInput
}

type approvalInput struct {
	PagePath
	Body PageApprovalInput
}

type versionInput struct {
	PagePath
	Version int32 `path:"version" minimum:"1"`
}

// ShopInput turns the shop on or off.
type ShopInput struct {
	On bool `json:"on"`
}

type shopInput struct {
	api.WorkspaceInput
	Body ShopInput
}

// Register adds the pages operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Pages"}
	base := "/workspaces/{workspace}/pages"
	view, edit, publish := api.Workspace(authz.WebsiteView), api.Workspace(authz.WebsiteEdit), api.Workspace(authz.WebsitePublish)

	api.Get(h, api.Op{ID: "list-pages", Path: base, Summary: "Your pages with their numbers, the shop's switch and the theme", Tags: tags, Access: view},
		func(ctx context.Context, in *overviewInput) (*api.Body[PagesOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Period)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-page", Path: base, Summary: "Start a page from a template, as a draft", Tags: tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *createInput) (*api.Body[PageDetail], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-shop", Path: base + "/shop", Summary: "Turn the shop (/products) on or off", Tags: tags, Access: api.Workspace(authz.ProductsPublish)},
		func(ctx context.Context, in *shopInput) (*api.Body[PagesShop], error) {
			out, err := s.SetShop(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body.On)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-page", Path: base + "/{page}", Summary: "A page: its draft, what's live, the publish checks and its numbers", Tags: tags, Access: view},
		func(ctx context.Context, in *PagePath) (*api.Body[PageDetail], error) {
			out, err := s.Detail(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-page-content", Path: base + "/{page}/content",
		Summary:     "Save the draft's blocks (autosave or Save): a new version; the live page changes only when published",
		Description: "A newer version saved meanwhile by someone else answers 409 version_conflict with who saved it; send force to keep yours on top.",
		Tags:        tags, Access: edit, MaxBodyBytes: MaxHTML + MaxProject + 64<<10},
		func(ctx context.Context, in *contentInput) (*api.Body[PageSaved], error) {
			out, err := s.SaveContent(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-page-settings", Path: base + "/{page}/settings",
		Summary: "Save the Page tab: title, address, visibility, header and navigation, SEO", Description: "Visibility and navigation need website:manage.",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *settingsInput) (*api.Body[PageDetail], error) {
			out, err := s.SaveSettings(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-page-diff", Path: base + "/{page}/diff", Summary: "What publishing would change since the live version", Tags: tags, Access: view},
		func(ctx context.Context, in *PagePath) (*api.Body[PageDiff], error) {
			out, err := s.Diff(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "publish-page", Path: base + "/{page}/publish", Summary: "Publish a page now, or schedule it",
		Description: "Blocked until every blocking check passes. When publishing needs approval the answer is 409 approval_required with the rule and the approvers.",
		Tags:        tags, Access: publish},
		func(ctx context.Context, in *publishInput) (*api.Body[PageDetail], error) {
			out, err := s.Publish(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "undo-page-publish", Path: base + "/{page}/undo-publish", Summary: "Take back a publish within seconds (the toast's Undo)", Tags: tags, Access: publish},
		func(ctx context.Context, in *PagePath) (*api.Body[PageDetail], error) {
			out, err := s.Undo(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "unpublish-page", Path: base + "/{page}/unpublish", Summary: "Take a page off the site, or unschedule it", Tags: tags, Access: publish},
		func(ctx context.Context, in *unpublishInput) (*api.Body[PageDetail], error) {
			out, err := s.Unpublish(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "request-page-approval", Path: base + "/{page}/approval", Summary: "Ask for approval to publish a page as it is",
		Tags: tags, Access: api.WorkspaceHuman(authz.WebsiteEdit), Status: http.StatusCreated},
		func(ctx context.Context, in *approvalInput) (*api.Body[PageDetail], error) {
			out, err := s.RequestApproval(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-page", Path: base + "/{page}", Summary: "Delete a page (a published one answers 410 at its address)", Tags: tags, Access: edit},
		func(ctx context.Context, in *PagePath) (*api.Empty, error) {
			return api.NoContent, s.Delete(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.id())
		})
	api.Get(h, api.Op{ID: "list-page-versions", Path: base + "/{page}/versions", Summary: "Version history: every saved and published version", Tags: tags, Access: view},
		func(ctx context.Context, in *PagePath) (*api.Body[PageVersionList], error) {
			out, err := s.Versions(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-page-version", Path: base + "/{page}/versions/{version}", Summary: "One version's blocks and settings", Tags: tags, Access: view},
		func(ctx context.Context, in *versionInput) (*api.Body[PageVersionDetail], error) {
			out, err := s.Version(ctx, api.WorkspaceFrom(ctx), in.id(), in.Version)
			return api.OK(out), err
		})
	registerReminders(h, s)
	registerFormPages(h, s)
	api.Post(h, api.Op{ID: "restore-page-version", Path: base + "/{page}/versions/{version}/restore", Summary: "Make an earlier version the draft again, as a new version",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *versionInput) (*api.Body[PageDetail], error) {
			out, err := s.Restore(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.id(), in.Version)
			return api.OK(out), err
		})
}

// SetShop turns the shop on or off.
func (s *Service) SetShop(ctx context.Context, ws *store.Workspace, p *authz.Principal, on bool) (PagesShop, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ShopSettingsEnsure(ctx, ws.ID); err != nil {
			return err
		}
		cur, err := q.ShopSettingsGet(ctx, ws.ID)
		if err != nil {
			return err
		}
		if cur.ShopOn == on {
			return nil
		}
		if err := q.ShopSettingsSave(ctx, store.ShopSettingsSaveParams{WorkspaceID: ws.ID, ShopOn: on, At: s.now(), Actor: actor(p)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "page.shop_updated", TargetType: "page", TargetID: "shop", TargetName: "Shop",
			Before: map[string]any{"on": cur.ShopOn}, After: map[string]any{"on": on}})
	})
	if err != nil {
		return PagesShop{}, err
	}
	return s.shop(ctx, ws, p, s.links(ctx, ws))
}
