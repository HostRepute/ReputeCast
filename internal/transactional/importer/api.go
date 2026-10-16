package importer

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

const importBase = "/workspaces/{workspace}/transactional/imports"

// ImportPath names a run of the Import templates dialog.
type ImportPath struct {
	api.WorkspaceInput
	ImportID string `path:"importId" format:"uuid"`
}

type startInput struct {
	api.WorkspaceInput
	Body TemplateImportStart
}

type pickInput struct {
	ImportPath
	Body TemplateImportPick
}

// Register adds the template import operations. Imports hold a provider's
// key while they run, so only people (not API keys or agents) start them.
func Register(h huma.API, s *Service) {
	tags := []string{"Transactional templates"}
	edit := api.WorkspaceHuman(authz.TransactionalEdit)
	api.Post(h, api.Op{ID: "start-transactional-template-import", Path: importBase, Summary: "Connect a provider, or upload MJML and HTML files, to import templates",
		Description: "Postmark, SendGrid and Mailgun list their templates; Notifuse needs its instance's address. The key is sealed for this import and never returned. Files (up to 50, 2 MB each) are converted at once.",
		Tags:        tags, Access: edit, Status: 201, MaxBodyBytes: 105 << 20},
		func(ctx context.Context, in *startInput) (*api.Body[TemplateImport], error) {
			out, err := s.Start(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-transactional-template-import", Path: importBase + "/{importId}", Summary: "A template import: its templates, conversion report and drafts",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *ImportPath) (*api.Body[TemplateImport], error) {
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx), in.ImportID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "convert-transactional-template-import", Path: importBase + "/{importId}/convert", Summary: "Read the picked templates and convert them to ReputeCast Liquid",
		Description: "Each template gets a report row: converted, converted with notes, or needing a manual fix, naming the line and the construct.",
		Tags:        tags, Access: edit},
		func(ctx context.Context, in *pickInput) (*api.Body[TemplateImport], error) {
			out, err := s.Convert(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ImportID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "finish-transactional-template-import", Path: importBase + "/{importId}/import", Summary: "Import the converted templates as drafts",
		Description: "Nothing is published: the API keeps sending the live versions until someone publishes a draft. The provider key is dropped.",
		Tags:        tags, Access: edit},
		func(ctx context.Context, in *ImportPath) (*api.Body[TemplateImport], error) {
			out, err := s.Import(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ImportID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "cancel-transactional-template-import", Path: importBase + "/{importId}/cancel", Summary: "Stop a template import without importing; the key is dropped",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *ImportPath) (*api.Body[TemplateImport], error) {
			out, err := s.Cancel(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ImportID)
			return api.OK(out), err
		})
}
