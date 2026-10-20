package offers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type idPath struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid" doc:"The offer's id"`
}

func (p idPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(p.ID)
	if err != nil {
		return uuid.Nil, notFound()
	}
	return id, nil
}

type offerBody struct {
	api.WorkspaceInput
	Body OfferInput
}

type idOfferBody struct {
	api.WorkspaceInput
	ID   string `path:"id" format:"uuid" doc:"The offer's id"`
	Body OfferInput
}

// Register adds the offers' operations: reading needs earn:view, changing
// needs earn:edit.
func Register(h huma.API, s *Service) {
	tags := []string{"Offers"}
	view := api.Workspace(authz.EarnView)
	edit := api.Workspace(authz.EarnEdit)
	base := "/workspaces/{workspace}/earn/offers"

	api.Get(h, api.Op{ID: "list-offers", Path: base, Summary: "Plans & offers: the offers and what they did", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[OfferList], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "new-offer", Path: base + "/new", Summary: "A new offer's editor: the tiers and periods it can apply to", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[OfferDetail], error) {
			out, err := s.New(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-offer", Path: base, Summary: "Create an offer, and its coupon and codes in the payment provider",
		Description: "Without a connected provider the offer is saved as a draft and created there once one is connected and it's saved again.",
		Tags:        tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *offerBody) (*api.Body[OfferDetail], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-offer", Path: base + "/{id}", Summary: "An offer: its rules, results, codes and its coupon in the provider", Tags: tags, Access: view},
		func(ctx context.Context, in *idPath) (*api.Body[OfferDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Detail(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-offer", Path: base + "/{id}", Summary: "Change an offer",
		Description: "Changing the discount creates a new coupon; codes already used keep theirs. Raising the limit or moving the date of an offer that ended by them brings it back.",
		Tags:        tags, Access: edit},
		func(ctx context.Context, in *idOfferBody) (*api.Body[OfferDetail], error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return nil, notFound()
			}
			out, err := s.Update(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "end-offer", Path: base + "/{id}/end", Summary: "End an offer now: its code stops working; paid members who used it keep their discount",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *idPath) (*api.Body[OfferDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.End(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "recreate-offer", Path: base + "/{id}/recreate", Summary: "Create the offer's coupon again after it was deleted in the provider",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *idPath) (*api.Body[OfferDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Recreate(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
}
