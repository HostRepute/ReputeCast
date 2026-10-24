package visibility

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type VisibilityCampaignPath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid"`
}

func (in VisibilityCampaignPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.CampaignID)
	if err != nil {
		return uuid.Nil, api.NotFound("Campaign")
	}
	return id, nil
}

type reportInput struct {
	VisibilityCampaignPath
	Body ReportInput
}

type previewInput struct {
	VisibilityCampaignPath
	Body PreviewInput
}

type usageInput struct {
	api.WorkspaceInput
	Kind string `query:"kind" enum:"segment,list,field,tag" doc:"What the rules refer to"`
	Ref  string `query:"ref" maxLength:"200" doc:"A segment or list id, a field key or a tag"`
}

// VisibilityPreviewContactList is who the canvas can preview as.
type VisibilityPreviewContactList struct {
	Items []VisibilityPreviewContact `json:"items"`
}

// Register adds the conditional content operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Conditional content"}
	api.Get(h, api.Op{ID: "visibility-catalog", Path: "/workspaces/{workspace}/visibility/catalog", Summary: "Conditions a Visibility rule can use, with their values",
		Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[VisibilityCatalog], error) {
			out, err := s.Catalog(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "campaign-visibility", Path: "/workspaces/{workspace}/campaigns/{campaignId}/visibility",
		Summary:     "Count the versions of an email's conditional content",
		Description: "Reads the Visibility rules of the content sent (the composer's unsaved MJML and HTML) or saved, and counts the versions the campaign's audience gets with the live audience. The count is kept until the rules or the audience change; recount counts again.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsView), MaxBodyBytes: 6 << 20},
		func(ctx context.Context, in *reportInput) (*api.Body[VisibilityReport], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Report(ctx, api.WorkspaceFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "campaign-visibility-preview", Path: "/workspaces/{workspace}/campaigns/{campaignId}/visibility/preview",
		Summary: "The version one contact gets, and why each block is hidden", Tags: tags, Access: api.Workspace(authz.SubscribersView), MaxBodyBytes: 6 << 20},
		func(ctx context.Context, in *previewInput) (*api.Body[VisibilityPreview], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.PreviewAs(ctx, api.WorkspaceFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "campaign-visibility-preview-contacts", Path: "/workspaces/{workspace}/campaigns/{campaignId}/visibility/preview-contacts",
		Summary: "A contact for each version, to preview as", Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *VisibilityCampaignPath) (*api.Body[VisibilityPreviewContactList], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			items, err := s.PreviewContacts(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(VisibilityPreviewContactList{Items: items}), err
		})
	api.Get(h, api.Op{ID: "visibility-usage", Path: "/workspaces/{workspace}/visibility/usage",
		Summary: "Conditional blocks that use a segment, list, field or tag", Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *usageInput) (*api.Body[VisibilityUsage], error) {
			out, err := s.Usage(ctx, api.WorkspaceFrom(ctx), in.Kind, in.Ref)
			return api.OK(out), err
		})
}
