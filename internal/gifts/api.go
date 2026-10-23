package gifts

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type listInput struct {
	api.WorkspaceInput
	Status string `query:"status" enum:"all,scheduled,not_redeemed,redeemed,expired,refunded" doc:"The filter (all by default)"`
	Search string `query:"q" maxLength:"100" doc:"Buyer, recipient or email"`
	Page   int    `query:"page" minimum:"1" doc:"The page (20 gifts each)"`
}

type optionBody struct {
	api.WorkspaceInput
	Body GiftOptionInput
}

type optionPath struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid" doc:"The gift option's id"`
}

type optionIDBody struct {
	api.WorkspaceInput
	ID   string `path:"id" format:"uuid" doc:"The gift option's id"`
	Body GiftOptionInput
}

type pageBody struct {
	api.WorkspaceInput
	Body GiftPageInput
}

type giftPath struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid" doc:"The gift's id"`
}

type deliveryBody struct {
	api.WorkspaceInput
	ID   string `path:"id" format:"uuid" doc:"The gift's id"`
	Body GiftDeliveryInput
}

func parseID(ctx context.Context, s string, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, notFound(ctx, what)
	}
	return id, nil
}

// Register adds the gifts' operations: reading needs earn:view, the
// options and the gift page earn:edit, and changing a gift earn:manage.
func Register(h huma.API, s *Service) {
	tags := []string{"Gifts"}
	view := api.Workspace(authz.EarnView)
	edit := api.Workspace(authz.EarnEdit)
	manage := api.Workspace(authz.EarnManage)
	base := "/workspaces/{workspace}/earn/gifts"

	api.Get(h, api.Op{ID: "get-gifts", Path: base, Summary: "Earn › Gifts: the numbers, the gift options and page, the gift emails and the gifts that need a look",
		Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[GiftsView], error) {
			out, err := s.View(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-gifts", Path: base + "/list", Summary: "A page of the gifts bought on the gift page, the latest first", Tags: tags, Access: view},
		func(ctx context.Context, in *listInput) (*api.Body[GiftList], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx), in.Status, in.Search, in.Page)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-gift-option", Path: base + "/options", Summary: "Add a gift option: a tier for a period at a one-time price",
		Tags: tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *optionBody) (*api.Body[GiftOption], error) {
			out, err := s.CreateOption(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-gift-option", Path: base + "/options/{id}", Summary: "Change a gift option; gifts already bought keep what they were sold as",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *optionIDBody) (*api.Body[GiftOption], error) {
			id, err := parseID(ctx, in.ID, "option")
			if err != nil {
				return nil, err
			}
			out, err := s.UpdateOption(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-gift-option", Path: base + "/options/{id}", Summary: "Take a gift option off the gift page; gifts already bought keep working",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *optionPath) (*api.Empty, error) {
			id, err := parseID(ctx, in.ID, "option")
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.DeleteOption(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Put(h, api.Op{ID: "set-gift-page", Path: base + "/page", Summary: "Turn the gift page on or off, and set how long gift links work",
		Description: "Gifts already bought keep working while the page is off: their emails go out and their links can be redeemed.",
		Tags:        tags, Access: edit},
		func(ctx context.Context, in *pageBody) (*api.Body[GiftPage], error) {
			out, err := s.SetPage(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "change-gift-delivery", Path: base + "/{id}/delivery", Summary: "Move a scheduled gift's delivery to another day and time",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *deliveryBody) (*api.Body[GiftRow], error) {
			id, err := parseID(ctx, in.ID, "gift")
			if err != nil {
				return nil, err
			}
			out, err := s.ChangeDelivery(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "resend-gift", Path: base + "/{id}/resend", Summary: "Send the gift email again to a gift's recipient",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *giftPath) (*api.Body[GiftRow], error) {
			id, err := parseID(ctx, in.ID, "gift")
			if err != nil {
				return nil, err
			}
			out, err := s.Resend(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "copy-gift-link", Path: base + "/{id}/link", Summary: "Copy a gift's redeem link (recorded in the audit log: anyone with it can redeem the gift)",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *giftPath) (*api.Body[GiftLink], error) {
			id, err := parseID(ctx, in.ID, "gift")
			if err != nil {
				return nil, err
			}
			out, err := s.CopyLink(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
}
