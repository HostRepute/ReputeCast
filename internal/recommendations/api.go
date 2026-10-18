package recommendations

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type recInput struct {
	api.WorkspaceInput
	Recommendation string `path:"recommendation" format:"uuid"`
}

func (in recInput) id() uuid.UUID { return uuid.MustParse(in.Recommendation) }

type linkInput struct {
	api.WorkspaceInput
	Link string `path:"link" format:"uuid"`
}

func (in linkInput) id() uuid.UUID { return uuid.MustParse(in.Link) }

// PartnerCodeText is a pasted partner code.
type PartnerCodeText struct {
	Code string `json:"code" maxLength:"500"`
}

// Register adds the recommendation and partner operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Grow"}
	base := "/workspaces/{workspace}/grow/recommendations"
	view, edit, connect := api.Workspace(authz.GrowView), api.WorkspaceHuman(authz.GrowEdit), api.WorkspaceHuman(authz.GrowConnect)

	api.Get(h, api.Op{ID: "get-recommendations", Path: base, Summary: "What you recommend, who recommends you, where recommendations show and your partner profile",
		Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[RecommendationsView], error) {
			out, err := s.View(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "add-link-recommendation", Path: base, Summary: "Recommend a newsletter by its subscribe page (readers subscribe there)",
		Tags: tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body RecLinkInput
		}) (*api.Body[RecommendationItem], error) {
			out, err := s.AddLink(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "prefill-recommendation", Path: base + "/prefill", Summary: "Read a subscribe page's name, description and logo (through the SSRF guard)",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body RecPrefillInput
		}) (*api.Body[RecPrefill], error) {
			out, err := s.Prefill(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "reorder-recommendations", Path: base + "/order", Summary: "Put the recommendations in the order readers see them", Tags: tags, Access: edit},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body RecOrderInput
		}) (*api.Empty, error) {
			return api.NoContent, s.Reorder(ctx, api.WorkspaceFrom(ctx), in.Body)
		})
	api.Put(h, api.Op{ID: "update-recommendation-settings", Path: base + "/settings", Summary: "Show recommendations on the opt-in confirmation page, or not",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body RecSettingsInput
		}) (*api.Empty, error) {
			return api.NoContent, s.SetConfirmPage(ctx, api.WorkspaceFrom(ctx), in.Body.ConfirmPage)
		})
	api.Put(h, api.Op{ID: "update-partner-profile", Path: base + "/profile", Summary: "Change the partner profile: what partners see and show",
		Description: "Partners that recommend you get the new profile over the signed channel.", Tags: tags, Access: edit},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body RecProfileInput
		}) (*api.Body[RecProfile], error) {
			out, err := s.UpdateProfile(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-recommendation", Path: base + "/{recommendation}", Summary: "Change a recommendation (a partner's: only where it shows)",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *struct {
			recInput
			Body RecLinkInput
		}) (*api.Body[RecommendationItem], error) {
			out, err := s.Update(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "pause-recommendation", Path: base + "/{recommendation}/pause", Summary: "Pause a link recommendation: hidden until resumed, numbers kept",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *recInput) (*api.Body[RecommendationItem], error) {
			out, err := s.SetPaused(ctx, api.WorkspaceFrom(ctx), in.id(), true)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "resume-recommendation", Path: base + "/{recommendation}/resume", Summary: "Resume a paused link recommendation", Tags: tags, Access: edit},
		func(ctx context.Context, in *recInput) (*api.Body[RecommendationItem], error) {
			out, err := s.SetPaused(ctx, api.WorkspaceFrom(ctx), in.id(), false)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-recommendation", Path: base + "/{recommendation}", Summary: "Remove a recommendation (a partner's also ends its link, which needs grow:connect)",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *recInput) (*api.Empty, error) {
			return api.NoContent, s.Remove(ctx, api.WorkspaceFrom(ctx), in.id())
		})

	// Partner links.
	pbase := "/workspaces/{workspace}/grow/partners"
	api.Post(h, api.Op{ID: "preview-partner", Path: pbase + "/preview", Summary: "Ask a partner's instance what a pasted partner code connects to",
		Tags: tags, Access: connect},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body PartnerCodeText
		}) (*api.Body[PartnerPreview], error) {
			out, err := s.PreviewPartner(ctx, api.WorkspaceFrom(ctx), in.Body.Code)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "connect-partner", Path: pbase + "/connect", Summary: "Connect a partner with its code: the instances agree a signing secret over HTTPS",
		Tags: tags, Access: connect, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body PartnerConnectInput
		}) (*api.Body[RecommendationItem], error) {
			out, err := s.ConnectPartner(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-partner-code", Path: "/workspaces/{workspace}/grow/partner-codes", Summary: "Create a partner code for one newsletter (shown once, valid 7 days, works once)",
		Tags: tags, Access: connect, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body PartnerCodeInput
		}) (*api.Body[PartnerCodeCreated], error) {
			out, err := s.CreateCode(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "revoke-partner-code", Path: "/workspaces/{workspace}/grow/partner-codes/{code}", Summary: "Revoke a partner code nobody has used",
		Tags: tags, Access: connect},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Code string `path:"code" format:"uuid"`
		}) (*api.Empty, error) {
			return api.NoContent, s.RevokeCode(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Code))
		})
	lbase := "/workspaces/{workspace}/grow/partner-links/{link}"
	api.Post(h, api.Op{ID: "pause-partner-link", Path: lbase + "/pause", Summary: "Pause a partner link: requests stop until it resumes", Tags: tags, Access: connect},
		func(ctx context.Context, in *linkInput) (*api.Empty, error) {
			return api.NoContent, s.PauseLink(ctx, api.WorkspaceFrom(ctx), in.id(), true)
		})
	api.Post(h, api.Op{ID: "resume-partner-link", Path: lbase + "/resume", Summary: "Resume a partner link this side paused", Tags: tags, Access: connect},
		func(ctx context.Context, in *linkInput) (*api.Empty, error) {
			return api.NoContent, s.PauseLink(ctx, api.WorkspaceFrom(ctx), in.id(), false)
		})
	api.Post(h, api.Op{ID: "end-partner-link", Path: lbase + "/end", Summary: "End a partnership for good; reconnecting needs a new partner code",
		Tags: tags, Access: connect},
		func(ctx context.Context, in *linkInput) (*api.Empty, error) {
			return api.NoContent, s.EndLink(ctx, api.WorkspaceFrom(ctx), in.id())
		})
	api.Put(h, api.Op{ID: "set-partner-limit", Path: lbase + "/limit", Summary: "Change the daily limit of a partner that recommends you", Tags: tags, Access: connect},
		func(ctx context.Context, in *struct {
			linkInput
			Body PartnerLimitInput
		}) (*api.Empty, error) {
			return api.NoContent, s.SetLimit(ctx, api.WorkspaceFrom(ctx), in.id(), in.Body.DailyLimit)
		})
	api.Post(h, api.Op{ID: "rotate-partner-secret", Path: lbase + "/rotate", Summary: "Agree a new signing secret with the partner; both work for 24 hours",
		Tags: tags, Access: connect},
		func(ctx context.Context, in *linkInput) (*api.Empty, error) {
			return api.NoContent, s.RotateSecret(ctx, api.WorkspaceFrom(ctx), in.id())
		})
	api.Post(h, api.Op{ID: "check-partner-link", Path: lbase + "/check", Summary: "Retry a partner's waiting requests now", Tags: tags, Access: connect},
		func(ctx context.Context, in *linkInput) (*api.Body[RecommendationItem], error) {
			out, err := s.CheckAgain(ctx, api.WorkspaceFrom(ctx), in.id())
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-partner-requests", Path: lbase + "/requests", Summary: "A partner link's request log (90 days; never the addresses)",
		Tags: tags, Access: view},
		func(ctx context.Context, in *struct {
			linkInput
			Page int `query:"page" minimum:"1" default:"1"`
		}) (*api.Body[PartnerRequestLog], error) {
			out, err := s.RequestLog(ctx, api.WorkspaceFrom(ctx), in.id(), in.Page)
			return api.OK(out), err
		})

	// The composer's block.
	api.Post(h, api.Op{ID: "render-recommendations-block", Path: "/workspaces/{workspace}/grow/blocks/recommendations", Summary: "Render the Recommendations block for the composer's canvas",
		Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body BlockRenderInput
		}) (*api.Body[BlockRender], error) {
			out, err := s.RenderBlock(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
}
