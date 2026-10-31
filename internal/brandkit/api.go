package brandkit

import (
	"context"
	"net/http"

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
	Kind string `path:"kind" pattern:"^(header|footer|b-[0-9a-f]{12})$" doc:"header, footer, or a named block's key"`
	Body SyncedInput
}

type syncedCreateInput struct {
	api.WorkspaceInput
	Body SyncedCreateInput
}

type syncedKindInput struct {
	api.WorkspaceInput
	Kind string `path:"kind" pattern:"^(header|footer|b-[0-9a-f]{12})$" doc:"A named block's key (the Header and Footer can't be deleted)"`
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
	api.Post(h, api.Op{ID: "create-synced-block", Path: base + "/synced", Summary: "Save a section as a named synced block",
		Description: "The block comes back marked with its key (rc-synced): the composer puts it in place of the section it came from. " +
			"A section that is, or holds, a synced block already is refused.",
		Tags: tags, Access: api.Workspace(authz.CampaignsEdit)},
		func(ctx context.Context, in *syncedCreateInput) (*api.Body[SyncedBlock], error) {
			out, err := s.CreateSynced(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-synced-block", Path: base + "/synced/{kind}", Status: http.StatusOK, Summary: "Delete a named synced block",
		Description: "Drafts and scheduled emails holding it keep its content as ordinary sections, in a new version; emails in an approval, sending, paused or sent keep theirs. " +
			"The Header and Footer can't be deleted.",
		Tags: tags, Access: api.Workspace(authz.CampaignsEdit)},
		func(ctx context.Context, in *syncedKindInput) (*api.Body[SyncedDeleted], error) {
			out, err := s.DeleteSynced(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Kind)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-synced-block", Path: base + "/synced/{kind}", Summary: "Change a synced block (the Header, the Footer or a named block)",
		Description: "Drafts and scheduled emails holding the block get the change as a new version; emails in an approval, sending, paused or sent keep theirs. " +
			"A named block can be renamed with it.",
		Tags: tags, Access: api.Workspace(authz.CampaignsEdit)},
		func(ctx context.Context, in *syncedInput) (*api.Body[SyncedSaved], error) {
			out, err := s.SaveSynced(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Kind, in.Body)
			return api.OK(out), err
		})
}
