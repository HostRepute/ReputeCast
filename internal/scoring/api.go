package scoring

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type saveInput struct {
	api.WorkspaceInput
	Body ScoringSaveInput
}

type previewInput struct {
	api.WorkspaceInput
	Body ScoringPreviewInput
}

type contactScoreInput struct {
	api.WorkspaceInput
	Contact string `path:"contact" format:"uuid"`
}

// Register adds the scoring operations. member answers "is this contact
// in that segment" for the profile's "Where it counts".
func Register(h huma.API, s *Service, member SegmentMembership) {
	tags := []string{"Scoring"}
	base := "/workspaces/{workspace}/scoring"
	api.Get(h, api.Op{ID: "get-scoring", Path: base, Summary: "Scoring rules, decay and bands, how contacts spread, bands by tier, what uses the score and the latest recalculation",
		Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[ScoringPage], error) {
			out, err := s.Page(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-scoring", Path: base, Summary: "Save the rules, decay and bands (recalculates every contact in the background when the rules change)",
		Tags: tags, Access: api.Workspace(authz.ScoringEdit)},
		func(ctx context.Context, in *saveInput) (*api.Body[ScoringPage], error) {
			out, err := s.Save(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "preview-scoring", Path: base + "/preview", Summary: "How an edited, unsaved set-up would move contacts between bands (nothing is saved)",
		Tags: tags, Access: api.Workspace(authz.ScoringEdit)},
		func(ctx context.Context, in *previewInput) (*api.Body[ScoringImpact], error) {
			out, err := s.Preview(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "retry-scoring-recalculation", Path: base + "/recalculate", Summary: "Start a stopped recalculation again where it stopped",
		Tags: tags, Access: api.Workspace(authz.ScoringEdit)},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[ScoringPage], error) {
			out, err := s.Retry(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-contact-score", Path: "/workspaces/{workspace}/contacts/{contact}/score", Summary: "A contact's score: its band, the breakdown by rule, the last change, the 90-day line and where it counts",
		Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *contactScoreInput) (*api.Body[ContactScore], error) {
			out, err := s.ContactScore(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Contact), member)
			return api.OK(out), err
		})
}
