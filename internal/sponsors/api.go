package sponsors

import (
	"context"
	"io"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// Inputs declare their path fields themselves: Huma skips an embedded
// struct of an unexported type, so a shared "id" struct would parse an
// empty id.

func parseID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, api.NotFound("Booking")
	}
	return id, nil
}

type idPath struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid" doc:"Id"`
}

func (p idPath) id() (uuid.UUID, error) { return parseID(p.ID) }

type issuePath struct {
	api.WorkspaceInput
	ID     string `path:"id" format:"uuid" doc:"The slot's id"`
	Number int    `path:"number" minimum:"1" doc:"The issue's number"`
}

type overviewInput struct {
	api.WorkspaceInput
	OverviewQuery
}

type body[T any] struct {
	api.WorkspaceInput
	Body T
}

type idBody[T any] struct {
	api.WorkspaceInput
	ID   string `path:"id" format:"uuid" doc:"Id"`
	Body T
}

func (p idBody[T]) id() (uuid.UUID, error) { return parseID(p.ID) }

type issueBody[T any] struct {
	api.WorkspaceInput
	ID     string `path:"id" format:"uuid" doc:"The slot's id"`
	Number int    `path:"number" minimum:"1" doc:"The issue's number"`
	Body   T
}

type blockInput struct {
	api.WorkspaceInput
	ID     string `path:"id" format:"uuid" doc:"The campaign's id"`
	Layout string `query:"layout" enum:"logo-left,image-top,text," doc:"The block's layout"`
	Label  string `query:"label" enum:"plain,name," doc:"Plain \"Sponsored\", or \"Sponsored by\" the sponsor"`
}

type uploadInput struct {
	api.WorkspaceInput
	ID      string `path:"id" format:"uuid" doc:"The booking's id"`
	RawBody huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" contentType:"image/png,image/jpeg,image/gif,application/octet-stream" required:"true" doc:"PNG, JPEG or GIF, up to 5 MB"`
	}]
}

func (p uploadInput) id() (uuid.UUID, error) { return parseID(p.ID) }

type logoInput struct {
	api.WorkspaceInput
	RawBody huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" contentType:"image/png,image/jpeg,image/gif,application/octet-stream" required:"true" doc:"PNG, JPEG or GIF, up to 5 MB"`
	}]
}

// upload keeps an uploaded logo or image in the media library.
func (s *Service) upload(ctx context.Context, f huma.FormFile) (SponsorImage, error) {
	if !f.IsSet {
		return SponsorImage{}, api.Invalid("body.file", text(ctx, "problem.imageMissing", nil))
	}
	data, err := io.ReadAll(io.LimitReader(f, maxImage+1))
	if err != nil || len(data) > maxImage {
		return SponsorImage{}, api.Invalid("body.file", text(ctx, "problem.imageTooLarge", nil))
	}
	if s.Images == nil {
		return SponsorImage{}, api.Err(http.StatusServiceUnavailable, api.CodeUnavailable, text(ctx, "problem.imagesUnavailable", nil))
	}
	u, err := s.Images.Upload(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), f.Filename, data)
	if err != nil {
		return SponsorImage{}, err
	}
	return SponsorImage{URL: u}, nil
}

// SponsorImage is an uploaded logo or image.
type SponsorImage struct {
	URL string `json:"url"`
}

// SponsorShareList is a booking's or a sponsor's read-only links.
type SponsorShareList struct {
	Items []SponsorShareLink `json:"items"`
}

// maxImage is the largest logo or image upload.
const maxImage = 5 << 20

// Register adds the sponsorship operations. Reading needs earn:view;
// sponsors, slots and bookings need sponsors:manage (recording a payment
// and confirming an agent's proposal need a person); read-only report
// links need reports:share, like the campaign report's.
func Register(h huma.API, s *Service) {
	tags := []string{"Sponsors"}
	view := api.Workspace(authz.EarnView)
	manage := api.Workspace(authz.SponsorsManage)
	person := api.WorkspaceHuman(authz.SponsorsManage)
	share := api.Workspace(authz.ReportsShare)
	ws := "/workspaces/{workspace}"

	api.Get(h, api.Op{ID: "get-sponsor-overview", Path: ws + "/sponsors/overview", Summary: "Earn › Sponsors: income, a slot's issues with their bookings, the library and what needs a person",
		Tags: tags, Access: view},
		func(ctx context.Context, in *overviewInput) (*api.Body[SponsorOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.OverviewQuery)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-sponsor", Path: ws + "/sponsors", Summary: "Add a sponsor to the library", Tags: tags, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *body[SponsorInput]) (*api.Body[SponsorItem], error) {
			out, err := s.CreateSponsor(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "upload-sponsor-logo", Path: ws + "/sponsors/logo", Summary: "Upload a sponsor's logo (PNG, JPEG or GIF, up to 5 MB)",
		Description: "The logo is kept in the media library; save its URL as the sponsor's logoUrl.", Tags: tags, Access: manage, MaxBodyBytes: maxImage + 64<<10, Status: http.StatusCreated},
		func(ctx context.Context, in *logoInput) (*api.Body[SponsorImage], error) {
			out, err := s.upload(ctx, in.RawBody.Data().File)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-sponsor", Path: ws + "/sponsors/{id}", Summary: "Change a sponsor", Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[SponsorInput]) (*api.Body[SponsorItem], error) {
			id, err := in.id()
			if err != nil {
				return nil, api.NotFound("Sponsor")
			}
			out, err := s.UpdateSponsor(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-sponsor", Path: ws + "/sponsors/{id}", Summary: "Remove a sponsor without bookings to come", Tags: tags, Access: manage},
		func(ctx context.Context, in *idPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, api.NotFound("Sponsor")
			}
			return api.NoContent, s.DeleteSponsor(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Post(h, api.Op{ID: "create-sponsor-share", Path: ws + "/sponsors/{id}/shares", Summary: "Share every booking of a sponsor in one read-only report",
		Description: "The page shows only the sponsor's numbers and the creatives as sent, never contact data.", Tags: tags, Access: share, Status: http.StatusCreated},
		func(ctx context.Context, in *idBody[ShareInput]) (*api.Body[SponsorShareLink], error) {
			id, err := in.id()
			if err != nil {
				return nil, api.NotFound("Sponsor")
			}
			out, err := s.CreateShare(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), nil, &id, in.Body)
			return api.OK(out), err
		})

	api.Post(h, api.Op{ID: "create-sponsor-slot", Path: ws + "/sponsor-slots", Summary: "Add a slot: a placement in each issue of a list", Tags: tags, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *body[SlotInput]) (*api.Body[SponsorSlotItem], error) {
			out, err := s.CreateSlot(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-sponsor-slot", Path: ws + "/sponsor-slots/{id}", Summary: "Change a slot", Description: "Issues already booked keep their dates and prices.",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[SlotInput]) (*api.Body[SponsorSlotItem], error) {
			id, err := in.id()
			if err != nil {
				return nil, api.NotFound("Slot")
			}
			out, err := s.UpdateSlot(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-sponsor-slot", Path: ws + "/sponsor-slots/{id}", Summary: "Remove a slot without bookings to come", Tags: tags, Access: manage},
		func(ctx context.Context, in *idPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, api.NotFound("Slot")
			}
			return api.NoContent, s.DeleteSlot(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Get(h, api.Op{ID: "get-sponsor-issue", Path: ws + "/sponsor-slots/{id}/issues/{number}", Summary: "An issue of a slot: its dates, price, campaign and any proposal",
		Tags: tags, Access: view},
		func(ctx context.Context, in *issuePath) (*api.Body[SponsorIssueDetail], error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return nil, api.NotFound("Slot")
			}
			out, err := s.Issue(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Number)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "book-sponsor-issue", Path: ws + "/sponsor-slots/{id}/issues/{number}/book", Summary: "Book an open issue for a sponsor, or hold it until a date",
		Tags: tags, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *issueBody[BookInput]) (*api.Body[SponsorBookingDetail], error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return nil, api.NotFound("Slot")
			}
			out, err := s.Book(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Number, in.Body)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-sponsor-booking", Path: ws + "/sponsor-bookings/{id}", Summary: "A booking: its terms, creative, payment, activity and report", Tags: tags, Access: view},
		func(ctx context.Context, in *idPath) (*api.Body[SponsorBookingDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Booking(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-sponsor-booking", Path: ws + "/sponsor-bookings/{id}", Summary: "Change a booking's sponsor, price, status, hold date or notes", Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[BookingInput]) (*api.Body[SponsorBookingDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.UpdateBooking(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "link-sponsor-booking-campaign", Path: ws + "/sponsor-bookings/{id}/campaign", Summary: "Link the issue to its campaign by hand, or by list and date again",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[CampaignInput]) (*api.Body[SponsorBookingDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SetCampaign(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-sponsor-creative", Path: ws + "/sponsor-bookings/{id}/creative", Summary: "Save the creative: logo or image, headline, text, button and link",
		Description: "ReputeCast adds the UTM parameters to the link; the click token is never added to sponsor links.", Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[SponsorCreative]) (*api.Body[SponsorBookingDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SetCreative(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "mark-sponsor-creative-ready", Path: ws + "/sponsor-bookings/{id}/creative/ready", Summary: "Mark the creative ready: the Sponsor block shows it",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *idPath) (*api.Body[SponsorBookingDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.MarkReady(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "upload-sponsor-image", Path: ws + "/sponsor-bookings/{id}/creative/image", Summary: "Upload the creative's logo or image (PNG, JPEG or GIF, up to 5 MB)",
		Tags: tags, Access: manage, MaxBodyBytes: maxImage + 64<<10, Status: http.StatusCreated},
		func(ctx context.Context, in *uploadInput) (*api.Body[SponsorImage], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			if _, _, err := s.mustBooking(ctx, api.WorkspaceFrom(ctx), id); err != nil {
				return nil, err
			}
			out, err := s.upload(ctx, in.RawBody.Data().File)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "record-sponsor-payment", Path: ws + "/sponsor-bookings/{id}/payment", Summary: "Record a booking's payment by hand, or change it",
		Description: "ReputeCast sends no invoices and moves no money: this records what the sponsor paid.", Tags: tags, Access: person},
		func(ctx context.Context, in *idBody[PaymentInput]) (*api.Body[SponsorBookingDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.RecordPayment(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "cancel-sponsor-booking", Path: ws + "/sponsor-bookings/{id}/cancel", Summary: "Cancel a booking, release a hold or dismiss a proposal",
		Description: "The issue opens again and its Sponsor block is hidden.", Tags: tags, Access: manage, Status: http.StatusNoContent},
		func(ctx context.Context, in *idBody[CancelInput]) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.CancelBooking(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
		})
	api.Post(h, api.Op{ID: "confirm-sponsor-proposal", Path: ws + "/sponsor-bookings/{id}/confirm", Summary: "Book an agent's proposal",
		Description: "MCP agents can propose a booking; a person with sponsors:manage confirms it.", Tags: tags, Access: person},
		func(ctx context.Context, in *idPath) (*api.Body[SponsorBookingDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.ConfirmProposal(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-sponsor-report", Path: ws + "/sponsor-bookings/{id}/report", Summary: "A sent booking's report: who got the block, human opens and clicks",
		Description: "Machine opens and bot clicks are left out. Results are final 72 hours after the send.", Tags: tags, Access: view},
		func(ctx context.Context, in *idPath) (*api.Body[SponsorReport], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Report(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-sponsor-shares", Path: ws + "/sponsor-bookings/{id}/shares", Summary: "A booking's active read-only links", Tags: tags, Access: share},
		func(ctx context.Context, in *idPath) (*api.Body[SponsorShareList], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			ws := api.WorkspaceFrom(ctx)
			if _, _, err := s.mustBooking(ctx, ws, id); err != nil {
				return nil, err
			}
			items, err := s.shares(ctx, ws, &id, nil)
			return api.OK(SponsorShareList{Items: items}), err
		})
	api.Post(h, api.Op{ID: "create-sponsor-booking-share", Path: ws + "/sponsor-bookings/{id}/shares", Summary: "Share a booking's report as a read-only link, optionally expiring",
		Description: "The page shows only the sponsor's numbers and the creative as sent, never contact data.", Tags: tags, Access: share, Status: http.StatusCreated},
		func(ctx context.Context, in *idBody[ShareInput]) (*api.Body[SponsorShareLink], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.CreateShare(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), &id, nil, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "revoke-sponsor-share", Path: ws + "/sponsor-report-shares/{id}", Summary: "Revoke a sponsor report's read-only link", Tags: tags, Access: share},
		func(ctx context.Context, in *idPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, api.NotFound("Link")
			}
			return api.NoContent, s.RevokeShare(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})

	api.Get(h, api.Op{ID: "list-sponsor-campaign-chips", Path: ws + "/sponsors/campaign-chips", Summary: "The sponsor of each campaign with a booking, for the Campaigns list",
		Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[CampaignChips], error) {
			out, err := s.Chips(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-campaign-sponsor", Path: ws + "/campaigns/{id}/sponsor", Summary: "A campaign report's Sponsor section: the issue's booking and its report",
		Description: "Answers without a booking when the issue had no sponsor.", Tags: tags, Access: view},
		func(ctx context.Context, in *idPath) (*api.Body[CampaignSponsor], error) {
			id, err := in.id()
			if err != nil {
				return nil, api.NotFound("Campaign")
			}
			out, err := s.CampaignSponsor(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-campaign-sponsor-block", Path: ws + "/campaigns/{id}/sponsor-block", Summary: "The Sponsor block of a campaign: its booking, creative and content",
		Description: "The price shows only to people who hold earn:view.", Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *blockInput) (*api.Body[SponsorBlock], error) {
			id, err := uuid.Parse(in.ID)
			if err != nil {
				return nil, api.NotFound("Campaign")
			}
			out, err := s.CampaignBlock(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Layout, in.Label)
			return api.OK(out), err
		})
}
