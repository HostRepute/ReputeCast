package deliverability

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/hostrepute"
	"github.com/hostrepute/reputecast/internal/i18n"
)

type overviewInput struct {
	api.WorkspaceInput
}

type runChecksInput struct {
	api.WorkspaceInput
	Builtin bool `query:"builtin" doc:"Run the built-in checks even when HostRepute is connected (while HostRepute is unreachable)"`
}

type runInput struct {
	api.WorkspaceInput
	RunID string `path:"runId" format:"uuid"`
}

type exportOutput struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	Body               []byte
}

type sourceInput struct {
	api.WorkspaceInput
	IP string `path:"ip" maxLength:"45"`
}

type uploadInput struct {
	api.WorkspaceInput
	RawBody []byte `contentType:"application/octet-stream" doc:"An aggregate report: XML, .xml.gz or .zip"`
}

type deliverabilityDelistInput struct {
	api.WorkspaceInput
	Body struct {
		Asset  string `json:"asset" minLength:"1" maxLength:"253"`
		Source string `json:"source,omitempty" maxLength:"120" doc:"One blocklist; every list the asset is on when omitted"`
	}
}

// Register adds the Deliverability operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Deliverability"}
	api.Get(h, api.Op{ID: "get-deliverability", Path: "/workspaces/{workspace}/deliverability",
		Summary: "Domain, IP and provider health, blocklists, Postmaster, SNDS, DMARC and incidents", Tags: tags, Access: api.Workspace(authz.DeliverabilityView)},
		func(ctx context.Context, _ *overviewInput) (*api.Body[DeliverabilityOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx))
			// Why HostRepute can't be reached is stored in English (Send
			// Readiness reads it as is); here it reads in the reader's language.
			if u := out.HostRepute.Unreachable; u != nil {
				u.Message = hostrepute.SyncErrorIn(i18n.LangFrom(ctx), u.Message)
			}
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-deliverability-reputation", Path: "/workspaces/{workspace}/deliverability/reputation",
		Summary: "Blocklist verdict used by Send Readiness (pass, warn, block or unknown)", Tags: tags, Access: api.Workspace(authz.DeliverabilityView)},
		func(ctx context.Context, _ *overviewInput) (*api.Body[DeliverabilityReputation], error) {
			out, err := s.Reputation(ctx, api.WorkspaceFrom(ctx))
			out.Reason = out.ReasonIn(i18n.LangFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "run-deliverability-checks", Path: "/workspaces/{workspace}/deliverability/checks",
		Summary:     "Run all checks now",
		Description: "Asks HostRepute to re-check every source when it is connected, otherwise runs the built-in checks. Returns the run in progress when one is already running.",
		Tags:        tags, Access: api.Workspace(authz.DeliverabilityView), Status: http.StatusAccepted},
		func(ctx context.Context, in *runChecksInput) (*api.Body[DeliverabilityCheckRun], error) {
			out, err := s.RunNow(ctx, api.WorkspaceFrom(ctx), in.Builtin)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-deliverability-check", Path: "/workspaces/{workspace}/deliverability/checks/{runId}",
		Summary: "Progress of a check run", Tags: tags, Access: api.Workspace(authz.DeliverabilityView)},
		func(ctx context.Context, in *runInput) (*api.Body[DeliverabilityCheckRun], error) {
			id, err := uuid.Parse(in.RunID)
			if err != nil {
				return nil, api.NotFound("Check run")
			}
			out, err := s.Run(ctx, api.WorkspaceFrom(ctx).ID, id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "export-deliverability", Path: "/workspaces/{workspace}/deliverability/export",
		Summary: "Export the deliverability report as CSV", Tags: tags, Access: api.Workspace(authz.DeliverabilityView)},
		func(ctx context.Context, _ *overviewInput) (*exportOutput, error) {
			ws := api.WorkspaceFrom(ctx)
			b, err := s.Export(ctx, ws)
			if err != nil {
				return nil, err
			}
			name := "deliverability-" + ws.Slug + "-" + time.Now().Format("2006-01-02") + ".csv"
			return &exportOutput{ContentType: "text/csv; charset=utf-8", ContentDisposition: `attachment; filename="` + name + `"`, Body: b}, nil
		})
	api.Get(h, api.Op{ID: "get-dmarc-source", Path: "/workspaces/{workspace}/deliverability/dmarc/sources/{ip}",
		Summary: "What the DMARC reports say about one sending IP", Tags: tags, Access: api.Workspace(authz.DeliverabilityView)},
		func(ctx context.Context, in *sourceInput) (*api.Body[DeliverabilityDMARCSourceDetail], error) {
			out, err := s.DMARCSource(ctx, api.WorkspaceFrom(ctx), in.IP)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "upload-dmarc-report", Path: "/workspaces/{workspace}/deliverability/dmarc/reports",
		Summary: "Upload a DMARC aggregate report (XML, .xml.gz or .zip)", Tags: tags, Access: api.Workspace(authz.DomainsManage),
		Status: http.StatusCreated, MaxBodyBytes: MaxReportBytes},
		func(ctx context.Context, in *uploadInput) (*api.Body[DeliverabilityDMARCUpload], error) {
			out, err := s.UploadReport(ctx, api.WorkspaceFrom(ctx), in.RawBody)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "request-delisting", Path: "/workspaces/{workspace}/deliverability/delistings",
		Summary: "Request delisting through HostRepute", Tags: tags, Access: api.Workspace(authz.IntegrationsHostRepute)},
		func(ctx context.Context, in *deliverabilityDelistInput) (*api.Body[DeliverabilityDelistingRequest], error) {
			out, err := s.RequestDelisting(ctx, api.WorkspaceFrom(ctx), in.Body.Asset, in.Body.Source)
			return api.OK(out), err
		})
}
