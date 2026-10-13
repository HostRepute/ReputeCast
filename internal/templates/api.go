package templates

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

type galleryInput struct {
	api.WorkspaceInput
}

type templateInput struct {
	api.WorkspaceInput
	ID string `path:"templateId" maxLength:"64" doc:"A template id, or starter:<key> for a built-in starter"`
}

type createInput struct {
	api.WorkspaceInput
	Body EmailTemplateInput
}

type updateInput struct {
	api.WorkspaceInput
	ID   string `path:"templateId" format:"uuid"`
	Body EmailTemplateInput
}

func actorOf(ctx context.Context) *uuid.UUID {
	if p := api.PrincipalFrom(ctx); p != nil && p.UserID != uuid.Nil {
		id := p.UserID
		return &id
	}
	return nil
}

// WorkspaceAddress reads the postal address from the workspace branding
// ({"address": "…"}), used to fill footers.
func WorkspaceAddress(ws *store.Workspace) string {
	if ws == nil || len(ws.Branding) == 0 {
		return ""
	}
	var b map[string]any
	if json.Unmarshal(ws.Branding, &b) != nil {
		return ""
	}
	for _, k := range []string{"address", "footer_address", "postal_address"} {
		if v, ok := b[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// Register adds the template operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Templates"}
	api.Get(h, api.Op{ID: "template-gallery", Path: "/workspaces/{workspace}/templates", Summary: "Starter templates, saved templates, saved sections and the last sent campaign",
		Tags: tags, Access: api.Workspace(authz.TemplatesView)},
		func(ctx context.Context, _ *galleryInput) (*api.Body[EmailTemplateGallery], error) {
			out, err := s.Gallery(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-template", Path: "/workspaces/{workspace}/templates/{templateId}", Summary: "Get a template with its MJML",
		Tags: tags, Access: api.Workspace(authz.TemplatesView)},
		func(ctx context.Context, in *templateInput) (*api.Body[EmailTemplateDetail], error) {
			ws := api.WorkspaceFrom(ctx)
			out, err := s.Get(ctx, ws, in.ID, WorkspaceAddress(ws))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-template", Path: "/workspaces/{workspace}/templates", Summary: "Save a template or a reusable section",
		Tags: tags, Access: api.Workspace(authz.TemplatesEdit), Status: 201, MaxBodyBytes: 5 << 20},
		func(ctx context.Context, in *createInput) (*api.Body[EmailTemplateDetail], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), actorOf(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-template", Path: "/workspaces/{workspace}/templates/{templateId}", Summary: "Rename a template or replace its content",
		Tags: tags, Access: api.Workspace(authz.TemplatesEdit), MaxBodyBytes: 5 << 20},
		func(ctx context.Context, in *updateInput) (*api.Body[EmailTemplateDetail], error) {
			out, err := s.Update(ctx, api.WorkspaceFrom(ctx), actorOf(ctx), in.ID, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-template", Path: "/workspaces/{workspace}/templates/{templateId}", Summary: "Delete a template or saved section",
		Tags: tags, Access: api.Workspace(authz.TemplatesEdit)},
		func(ctx context.Context, in *updateInputID) (*api.Empty, error) {
			return api.NoContent, s.Delete(ctx, api.WorkspaceFrom(ctx), actorOf(ctx), in.ID)
		})
}

type updateInputID struct {
	api.WorkspaceInput
	ID string `path:"templateId" format:"uuid"`
}
