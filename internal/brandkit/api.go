package brandkit

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type saveInput struct {
	api.WorkspaceInput
	Body BrandKitInput
}

type importInput struct {
	api.WorkspaceInput
	Body BrandKitImportInput
}

type syncedInput struct {
	api.WorkspaceInput
	Kind string `path:"kind" enum:"header,footer"`
	Body SyncedInput
}

// Register adds the brand kit operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Brand kit"}
	base := "/workspaces/{workspace}/brand-kit"
	api.Get(h, api.Op{ID: "get-brand-kit", Path: base, Summary: "The brand kit, its synced blocks and the emails holding them",
		Description: "Without a saved kit, the values are ReputeCast's defaults (saved is false).",
		Tags:        tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, _ *struct{ api.WorkspaceInput }) (*api.Body[BrandKit], error) {
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-brand-kit", Path: base, Summary: "Save the brand kit",
		Description: "New emails, templates and blocks start from it; emails already written keep their own styles. The first save creates the synced Header and Footer; " +
			"a new logo reaches the synced Header, and the drafts and scheduled emails holding it when the caller also holds campaigns:edit.",
		Tags: tags, Access: api.Workspace(authz.TemplatesEdit)},
		func(ctx context.Context, in *saveInput) (*api.Body[BrandKit], error) {
			out, err := s.Save(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "import-brand-kit", Path: base + "/import", Summary: "Read the logo, colours and fonts from a website",
		Description: "Nothing in the kit changes until it's saved. Logos found are added to the media library when the caller also holds campaigns:edit, as uploads need; " +
			"otherwise a note says they weren't.",
		Tags: tags, Access: api.Workspace(authz.TemplatesEdit)},
		func(ctx context.Context, in *importInput) (*api.Body[BrandKitImport], error) {
			out, err := s.Import(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-synced-block", Path: base + "/synced/{kind}", Summary: "Change a synced block (Header or Footer)",
		Description: "Drafts and scheduled emails holding the block get the change as a new version; emails in an approval, sending, paused or sent keep theirs.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsEdit)},
		func(ctx context.Context, in *syncedInput) (*api.Body[SyncedSaved], error) {
			out, err := s.SaveSynced(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Kind, in.Body)
			return api.OK(out), err
		})
}
