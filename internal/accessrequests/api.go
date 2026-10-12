package accessrequests

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
)

type suggestionInput struct {
	api.WorkspaceInput
	Permission string `query:"permission" required:"true" maxLength:"64" doc:"The missing permission, from a 403 problem" example:"providers:manage"`
}

type requestInput struct {
	api.WorkspaceInput
	Body struct {
		Permission string `json:"permission" maxLength:"64" doc:"The missing permission" example:"providers:manage"`
		Note       string `json:"note,omitempty" maxLength:"500" doc:"Why you need it; shown to whoever decides"`
	}
}

type requestOutput struct {
	Status int
	Body   AccessRequestResult
}

// Register adds the access request operations. Any member may call them
// (no permission needed); API keys and agents may not.
func Register(h huma.API, s *Service) {
	tags := []string{"Team"}
	access := api.WorkspaceHuman("")
	api.Get(h, api.Op{
		ID: "suggest-access", Path: "/workspaces/{workspace}/access-requests/suggestion", Tags: tags, Access: access,
		Summary: "Which role grants a missing permission, and who can grant it",
	}, func(ctx context.Context, in *suggestionInput) (*api.Body[AccessSuggestion], error) {
		r, err := s.Suggest(ctx, in.Permission)
		return api.OK(r), err
	})

	api.Post(h, api.Op{
		ID: "request-access", Path: "/workspaces/{workspace}/access-requests", Tags: tags, Access: access, Status: statusAccepted,
		Summary:     "Ask for the role that grants a missing permission",
		Description: "Notifies the members who can change roles, with your note. One request per role a day. Recorded as access.request.",
	}, func(ctx context.Context, in *requestInput) (*requestOutput, error) {
		r, err := s.Request(ctx, in.Body.Permission, in.Body.Note)
		if err != nil {
			return nil, err
		}
		return &requestOutput{Status: statusAccepted, Body: r}, nil
	})
}
