package growth

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type pageInput struct {
	api.WorkspaceInput
	GrowthQuery
}

type contactInput struct {
	api.WorkspaceInput
	Contact string `path:"contact" format:"uuid"`
}

// GrowthFile is a CSV download.
type GrowthFile struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

// Register adds the growth operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Growth"}
	base := "/workspaces/{workspace}/growth"
	api.Get(h, api.Op{ID: "get-growth", Path: base, Summary: "Joins and departures by source, quality by source, UTM and referring pages, and cohort retention",
		Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *pageInput) (*api.Body[GrowthPage], error) {
			out, err := s.Page(ctx, api.WorkspaceFrom(ctx), in.GrowthQuery)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-growth-sources", Path: base + "/sources", Summary: "The sources contacts joined through, with how many (the contacts filter)",
		Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[GrowthSourceList], error) {
			out, err := s.Sources(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "export-growth", Path: base + "/export", Summary: "The Growth tables as CSV: totals only, no addresses (logged)",
		Tags: tags, Access: api.Workspace(authz.ReportsView)},
		func(ctx context.Context, in *pageInput) (*GrowthFile, error) {
			body, name, err := s.Export(ctx, api.WorkspaceFrom(ctx), in.GrowthQuery)
			if err != nil {
				return nil, err
			}
			return &GrowthFile{ContentType: "text/csv; charset=utf-8", ContentDisposition: `attachment; filename="` + name + `"`, CacheControl: "no-store", Body: body}, nil
		})
	api.Get(h, api.Op{ID: "get-contact-growth", Path: "/workspaces/{workspace}/contacts/{contact}/growth", Summary: "How a contact first joined: the source, UTM parameters and referring page",
		Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *contactInput) (*api.Body[ContactJoin], error) {
			out, err := s.ContactJoin(ctx, api.WorkspaceFrom(ctx), uuid.MustParse(in.Contact))
			return api.OK(out), err
		})
}
