package applang

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
)

type saveLanguageInput struct {
	Body MemberLanguage
}

// Register adds Account & security › Language & region.
func Register(h huma.API, s *Service) {
	api.Put(h, api.Op{
		ID:      "update-my-language",
		Path:    "/me/language",
		Summary: "Change your app language and region format",
		Description: "Sets the language of ReputeCast's interface and of the emails it sends you, and how dates, " +
			"times and numbers are written. \"default\" follows the instance default; \"auto\" follows the language " +
			"(region format) or the region format (clock, first day of the week). Times stay in each workspace's timezone.",
		Tags:   []string{"Account"},
		Access: api.Access{Kind: api.AccessUser, HumansOnly: true, AllowPending: true},
	}, func(ctx context.Context, in *saveLanguageInput) (*api.Body[MemberLanguage], error) {
		out, err := s.Save(ctx, in.Body)
		if err != nil {
			return nil, err
		}
		return api.OK(out), nil
	})

	api.Put(h, api.Op{
		ID:          "answer-language-suggestion",
		Path:        "/me/language/suggestion",
		Summary:     "Answer the browser's language suggestion",
		Description: "Records that you answered \"Your browser prefers …\", so it isn't offered again on any device.",
		Tags:        []string{"Account"},
		Access:      api.Access{Kind: api.AccessUser, HumansOnly: true, AllowPending: true},
		Status:      http.StatusNoContent,
	}, func(ctx context.Context, in *struct{ Body SuggestionAnswer }) (*api.Empty, error) {
		return api.NoContent, s.AnswerSuggestion(ctx, in.Body)
	})
}
