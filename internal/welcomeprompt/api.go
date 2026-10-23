package welcomeprompt

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
)

type promptInput struct {
	api.WorkspaceInput
	Context string `query:"context" enum:"health,form,import" doc:"The screen: List health, a form in the builder, or an import's summary"`
	ID      string `query:"id" format:"uuid" doc:"The form or the import (not for List health)"`
}

// WelcomePromptDismissInput names the list whose prompt is dismissed.
type WelcomePromptDismissInput struct {
	ListID string `json:"listId" format:"uuid"`
}

type dismissInput struct {
	api.WorkspaceInput
	Body WelcomePromptDismissInput
}

// Register adds the prompt operations. Prompts are for people who can
// start an automation; agents and keys have no use for them.
func Register(h huma.API, s *Service) {
	tags := []string{"Automations"}
	api.Get(h, api.Op{ID: "get-welcome-prompt", Path: "/workspaces/{workspace}/welcome-prompt",
		Summary:     "The Welcome series prompt for a screen",
		Description: "The first list of the screen whose new subscribers get no welcome email: no live automation starts on it (or on the form), and nobody dismissed it.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.AutomationsEdit)},
		func(ctx context.Context, in *promptInput) (*api.Body[WelcomePromptResult], error) {
			id := uuid.Nil
			if in.Context != ContextHealth {
				parsed, err := uuid.Parse(in.ID)
				if err != nil {
					return nil, api.Invalid("query.id", mNameFormOrImport.Text(i18n.LangFrom(ctx), nil))
				}
				id = parsed
			}
			out, err := s.Prompt(ctx, api.WorkspaceFrom(ctx), in.Context, id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "dismiss-welcome-prompt", Path: "/workspaces/{workspace}/welcome-prompt/dismiss",
		Summary: "Stop offering the Welcome series for a list (Not now)", Tags: tags, Access: api.WorkspaceHuman(authz.AutomationsEdit)},
		func(ctx context.Context, in *dismissInput) (*api.Empty, error) {
			listID, err := uuid.Parse(in.Body.ListID)
			if err != nil {
				return nil, api.Invalid("body.listId", mNameList.Text(i18n.LangFrom(ctx), nil))
			}
			var by *uuid.UUID
			if u := api.UserFrom(ctx); u != nil {
				id := u.ID
				by = &id
			}
			return api.NoContent, s.Dismiss(ctx, api.WorkspaceFrom(ctx), by, listID)
		})
}
