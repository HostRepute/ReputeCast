package appearance

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
)

// Register adds Account & security › Appearance.
func Register(h huma.API, s *Service) {
	api.Put(h, api.Op{
		ID:      "update-my-appearance",
		Path:    "/me/appearance",
		Summary: "Change your theme",
		Description: "Sets whether ReputeCast's interface is light or dark for you, on every browser you sign in on. " +
			"\"system\" follows each device's light or dark setting. Emails, public pages and their previews keep their own colours.",
		Tags:   []string{"Account"},
		Access: api.Access{Kind: api.AccessUser, HumansOnly: true, AllowPending: true},
	}, func(ctx context.Context, in *struct{ Body Choice }) (*api.Body[Choice], error) {
		out, err := s.Save(ctx, in.Body)
		if err != nil {
			return nil, err
		}
		return api.OK(out), nil
	})
}
