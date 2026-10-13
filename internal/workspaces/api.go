package workspaces

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// Register adds the workspace settings operations (Settings › General).
// Creating a workspace lives with the account endpoints (POST /workspaces).
func Register(h huma.API, s *Service) {
	tags := []string{"Workspaces"}
	base := "/workspaces/{workspace}"

	api.Get(h, api.Op{ID: "get-workspace-settings", Path: base + "/settings", Summary: "General settings, approval and team summaries, danger-zone counts", Tags: tags, Access: api.Workspace("")},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[WorkspaceSettings], error) {
			out, err := s.Settings(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})

	api.Put(h, api.Op{ID: "update-workspace-settings", Path: base + "/settings", Summary: "Change the name, timezone and default sender", Tags: tags, Access: api.Workspace(authz.WorkspaceManage), Status: 204},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body WorkspaceGeneralInput
		}) (*api.Empty, error) {
			return api.NoContent, s.UpdateGeneral(ctx, api.WorkspaceFrom(ctx), in.Body)
		})

	api.Post(h, api.Op{ID: "transfer-workspace", Path: base + "/transfer", Summary: "Make another member the Owner; you become an Admin", Tags: tags, Access: api.WorkspaceHuman(authz.WorkspaceTransfer), Status: 204},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body WorkspaceTransferInput
		}) (*api.Empty, error) {
			return api.NoContent, s.Transfer(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
		})

	api.Post(h, api.Op{ID: "delete-workspace", Path: base + "/delete", Summary: "Delete the workspace (typed confirmation; restorable for 7 days)", Tags: tags, Access: api.WorkspaceHuman(authz.WorkspaceDelete)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body WorkspaceDeleteInput
		}) (*api.Body[WorkspaceDeleteResult], error) {
			out, err := s.Delete(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "export-workspace", Path: base + "/export", Summary: "Export everything: contacts with consent, campaigns as MJML, reports as CSV, audit log as JSON (ZIP, streamed)", Tags: tags, Access: api.WorkspaceHuman(authz.SubscribersExport)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*huma.StreamResponse, error) {
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			return &huma.StreamResponse{Body: func(hctx huma.Context) {
				name := strings.Trim(strings.ToLower(ws.Slug), "-")
				hctx.SetHeader("Content-Type", "application/zip")
				hctx.SetHeader("Content-Disposition", fmt.Sprintf(`attachment; filename="reputecast-%s-%s.zip"`, name, time.Now().UTC().Format("2006-01-02")))
				hctx.SetHeader("Cache-Control", "no-store")
				hctx.SetStatus(200)
				_ = s.Export(ctx, ws, p, hctx.BodyWriter())
			}}, nil
		})
}
