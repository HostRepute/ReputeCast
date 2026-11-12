package sponsors

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// registerSelfServe adds self-serve booking's operations (M65): reading
// needs earn:view; the sponsor page's settings and reviewing requests need
// a person with sponsors:manage (no API keys or agents: a request's
// approval asks a sponsor for money).
func registerSelfServe(h huma.API, s *Service) {
	tags := []string{"Sponsors"}
	view := api.Workspace(authz.EarnView)
	person := api.WorkspaceHuman(authz.SponsorsManage)
	ws := "/workspaces/{workspace}"

	api.Get(h, api.Op{ID: "get-sponsor-page-settings", Path: ws + "/sponsor-page", Summary: "The sponsor page's settings: self-serve booking, its media kit and address",
		Tags: tags, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[SponsorSelfServe], error) {
			out, err := s.SelfServeSettings(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-sponsor-page-settings", Path: ws + "/sponsor-page", Summary: "Turn self-serve booking on or off and set the sponsor page's media kit",
		Description: "Turning it on needs a payment account connected in Earn: sponsors pay through its Stripe Checkout.", Tags: tags, Access: person},
		func(ctx context.Context, in *body[SponsorSelfServeInput]) (*api.Body[SponsorSelfServe], error) {
			out, err := s.SaveSelfServe(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-sponsor-requests", Path: ws + "/sponsor-requests", Summary: "The sponsor page's requests: waiting for review, approved and waiting for payment, recently declined or expired",
		Tags: tags, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[SponsorRequestList], error) {
			out, err := s.Requests(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-sponsor-request", Path: ws + "/sponsor-requests/{id}", Summary: "A request from the sponsor page: who sent it, its creative and where it can be booked",
		Tags: tags, Access: view},
		func(ctx context.Context, in *idPath) (*api.Body[SponsorRequestDetail], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, notFound(ctx, "request")
			}
			out, err := s.RequestDetail(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "approve-sponsor-request", Path: ws + "/sponsor-requests/{id}/approve", Summary: "Approve a request: hold the issue until the payment's deadline and email the sponsor the link to pay",
		Description: "The deadline is the slot's pay_days from now, at the latest when the creative is due. The issue, slot and price may change.", Tags: tags, Access: person},
		func(ctx context.Context, in *idBody[ApproveInput]) (*api.Body[SponsorRequestDetail], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, notFound(ctx, "request")
			}
			out, err := s.Approve(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "decline-sponsor-request", Path: ws + "/sponsor-requests/{id}/decline", Summary: "Decline a request; the sponsor is emailed the reason",
		Tags: tags, Access: person},
		func(ctx context.Context, in *idBody[DeclineInput]) (*api.Body[SponsorRequestDetail], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, notFound(ctx, "request")
			}
			out, err := s.Decline(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
}
