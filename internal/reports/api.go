package reports

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
)

// ReportPath names a campaign of the workspace.
type ReportPath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid" doc:"Campaign id"`
}

func (p ReportPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return uuid.Nil, api.NotFound("Campaign")
	}
	return id, nil
}

type reportInput struct {
	ReportPath
	Language string `query:"language" maxLength:"16" doc:"Show the click map of one language version (campaigns sent in several languages)"`
}

type sharePath struct {
	ReportPath
	ShareID string `path:"shareId" format:"uuid" doc:"Link id"`
}

type bounceTypePath struct {
	ReportPath
	Type string `path:"type" enum:"unknown_user,domain_invalid,mailbox_full,deferral_expired,blocked,other" doc:"Bounce type"`
}

type createShareInput struct {
	ReportPath
	Body ShareInput
}

type sharedInput struct {
	Token string `path:"token" maxLength:"64" doc:"The token from the link"`
}

// reportFile is a download.
type reportFile struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

// sharedOutput is the public report, never cached or indexed.
type sharedOutput struct {
	CacheControl string `header:"Cache-Control"`
	RobotsTag    string `header:"X-Robots-Tag"`
	Body         SharedCampaignReport
}

// Register adds the report operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Reports"}
	base := "/workspaces/{workspace}/campaigns/{campaignId}/report"
	view, share := api.Workspace(authz.ReportsView), api.Workspace(authz.ReportsShare)

	api.Get(h, api.Op{ID: "get-campaign-report", Path: base, Summary: "Campaign report: headline metrics, 72-hour engagement, click map, providers, bounces, unsubscribes, A/B",
		Description: "Aggregate numbers only. Rates are percentages: delivered and bounces of sent; opens, clicks, unsubscribes and complaints of delivered.",
		Tags:        tags, Access: view},
		func(ctx context.Context, in *reportInput) (*api.Body[CampaignReport], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.ReportIn(ctx, api.WorkspaceFrom(ctx), id, in.Language)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "export-campaign-report", Path: base + "/export.csv", Summary: "Download the report as CSV, one section per table (logged)",
		Description: "Exporting takes the report outside the workspace, so it needs the same permission as sharing it.", Tags: tags, Access: share},
		func(ctx context.Context, in *ReportPath) (*reportFile, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			f, err := s.Export(ctx, api.WorkspaceFrom(ctx), id, "csv")
			if err != nil {
				return nil, err
			}
			return &reportFile{ContentType: "text/csv; charset=utf-8", ContentDisposition: fmt.Sprintf(`attachment; filename="%s"`, f.Filename), CacheControl: "no-store", Body: f.Body}, nil
		})
	api.Post(h, api.Op{ID: "record-campaign-report-print", Path: base + "/print", Summary: "Record a PDF export of the printable report (logged)",
		Description: "The browser prints the report to PDF; this records the export in the audit log.", Tags: tags, Access: share},
		func(ctx context.Context, in *ReportPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			_, err = s.Export(ctx, api.WorkspaceFrom(ctx), id, "pdf")
			return api.NoContent, err
		})
	api.Get(h, api.Op{ID: "get-campaign-report-bounce-review", Path: base + "/bounces/{type}", Summary: "What the providers said for one bounce type (no addresses)",
		Tags: tags, Access: view},
		func(ctx context.Context, in *bounceTypePath) (*api.Body[ReportBounceReview], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.BounceReview(ctx, api.WorkspaceFrom(ctx), id, in.Type)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "suppress-campaign-report-bounces", Path: base + "/bounces/{type}/suppress",
		Summary: "Add the bounced addresses of one type to the shared suppression list", Tags: tags, Access: api.Workspace(authz.SubscribersEdit)},
		func(ctx context.Context, in *bounceTypePath) (*api.Body[ReportSuppressResult], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SuppressBounces(ctx, api.WorkspaceFrom(ctx), id, in.Type)
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-campaign-report-final-alert", Path: base + "/final-alert",
		Summary: "Whether you asked to be emailed when the report's results are final", Tags: tags, Access: view},
		func(ctx context.Context, in *ReportPath) (*api.Body[ReportFinalAlert], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.FinalAlert(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-campaign-report-final-alert", Path: base + "/final-alert",
		Summary:     "Email me when the results are final (72 hours after the send ended)",
		Description: "Sends one notification, in the app and by email as your notification preferences say. Only people can ask, not API keys.",
		Tags:        tags, Access: view},
		func(ctx context.Context, in *ReportPath) (*api.Body[ReportFinalAlert], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SetFinalAlert(ctx, api.WorkspaceFrom(ctx), id, true)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-campaign-report-final-alert", Path: base + "/final-alert", Summary: "Stop the email for when the results are final",
		Tags: tags, Access: view},
		func(ctx context.Context, in *ReportPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			if _, err := s.SetFinalAlert(ctx, api.WorkspaceFrom(ctx), id, false); err != nil {
				return nil, err
			}
			return api.NoContent, nil
		})

	api.Get(h, api.Op{ID: "list-campaign-report-shares", Path: base + "/shares", Summary: "Active read-only links of a report", Tags: tags, Access: share},
		func(ctx context.Context, in *ReportPath) (*api.Body[ReportShareList], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Shares(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-campaign-report-share", Path: base + "/shares", Summary: "Create a read-only link (no sign-in needed), optionally expiring",
		Tags: tags, Access: share, Status: http.StatusCreated},
		func(ctx context.Context, in *createShareInput) (*api.Body[ReportShareLink], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.CreateShare(ctx, api.WorkspaceFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "revoke-campaign-report-share", Path: base + "/shares/{shareId}", Summary: "Revoke a read-only link", Tags: tags, Access: share},
		func(ctx context.Context, in *sharePath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			sid, err := uuid.Parse(in.ShareID)
			if err != nil {
				return nil, api.NotFound("Link")
			}
			return api.NoContent, s.RevokeShare(ctx, api.WorkspaceFrom(ctx), id, sid)
		})

	api.Get(h, api.Op{ID: "get-shared-report", Path: "/shared-reports/{token}", Summary: "The report behind a read-only link (public, rate-limited)",
		Description: "Anyone with the link sees the campaign's aggregate numbers, never contact data. Limited per client address.", Tags: tags, Access: api.Public},
		func(ctx context.Context, in *sharedInput) (*sharedOutput, error) {
			client := "unknown"
			if ip := audit.RequestFrom(ctx).IP; ip != nil {
				client = ip.String()
			}
			out, err := s.Shared(ctx, in.Token, client)
			if err != nil {
				return nil, err
			}
			return &sharedOutput{CacheControl: "no-store", RobotsTag: "noindex, nofollow", Body: out}, nil
		})
}
