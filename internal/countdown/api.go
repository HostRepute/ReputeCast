package countdown

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type createInput struct {
	api.WorkspaceInput
	Body CountdownSettings
}

// Register adds the countdown operation.
func Register(h huma.API, s *Service) {
	api.Post(h, api.Op{ID: "create-countdown", Path: "/workspaces/{workspace}/countdowns", Summary: "The image of a Countdown block's settings",
		Description: "The image is an animated GIF drawn each time the email is opened: up to a minute of the time left, then it stays on its last frame. " +
			"The same settings always return the same image. The digits need a contrast of at least 4.5:1 on the background.",
		Tags: []string{"Campaigns"}, Access: api.Workspace(authz.CampaignsEdit)},
		func(ctx context.Context, in *createInput) (*api.Body[CountdownImage], error) {
			var actor *uuid.UUID
			if p := api.PrincipalFrom(ctx); p != nil && p.UserID != uuid.Nil {
				id := p.UserID
				actor = &id
			}
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), actor, in.Body)
			if err != nil {
				return nil, err
			}
			return api.OK(out), nil
		})
}
