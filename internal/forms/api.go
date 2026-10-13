package forms

import (
	"context"
	"io"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// Reading forms and page settings needs subscribers:view: forms are where
// contacts come from, and Viewers see everything (A5 §3.6). The spec has a
// single key for this area, forms:edit, which every change needs.
const viewPerm = authz.SubscribersView

// FormPath names a form in the path. It is exported because Huma only
// reads parameters through exported embedded structs.
type FormPath struct {
	api.WorkspaceInput
	Form string `path:"form" format:"uuid" doc:"Form id"`
}

type formBody[T any] struct {
	FormPath
	Body T
}

type wsBody[T any] struct {
	api.WorkspaceInput
	Body T
}

type archiveListInput struct {
	api.WorkspaceInput
	Q    string `query:"q" maxLength:"100" doc:"Search titles, subjects and excerpts"`
	List string `query:"list" doc:"Only issues filed under this list (id)"`
	Page int    `query:"page" minimum:"0" maximum:"100000" doc:"Page, from 1 (10 issues a page)"`
}

type archiveIssueInput struct {
	api.WorkspaceInput
	Campaign string `path:"campaign" format:"uuid" doc:"The issue's campaign id"`
	Body     ArchiveIssueInput
}

type logoInput struct {
	api.WorkspaceInput
	RawBody huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" contentType:"image/svg+xml,image/png,image/jpeg,image/gif,image/webp,application/octet-stream" required:"true" doc:"SVG, PNG, JPEG, GIF or WebP, up to 256 KB"`
	}]
}

// Register adds the forms and public page settings operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Forms"}
	api.Get(h, api.Op{ID: "list-signup-forms", Path: "/workspaces/{workspace}/forms", Summary: "List signup forms with the lists, fields and branding they use",
		Tags: tags, Access: api.Workspace(viewPerm)},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[SignupFormsOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-signup-form", Path: "/workspaces/{workspace}/forms", Summary: "Create a signup form (a draft until it's published)",
		Tags: tags, Access: api.Workspace(authz.FormsEdit), Status: 201},
		func(ctx context.Context, in *wsBody[SignupFormCreateInput]) (*api.Body[SignupForm], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-signup-form", Path: "/workspaces/{workspace}/forms/{form}", Summary: "Get a signup form with its numbers for the last 30 days",
		Tags: tags, Access: api.Workspace(viewPerm)},
		func(ctx context.Context, in *FormPath) (*api.Body[SignupForm], error) {
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx), in.Form)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-signup-form", Path: "/workspaces/{workspace}/forms/{form}", Summary: "Save a signup form (Save & publish)",
		Description: "Changing the consent text starts a new consent version; every consent event records the exact text and its version.",
		Tags:        tags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *formBody[SignupFormInput]) (*api.Body[SignupForm], error) {
			out, err := s.Update(ctx, api.WorkspaceFrom(ctx), in.Form, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "rename-signup-form", Path: "/workspaces/{workspace}/forms/{form}/rename", Summary: "Rename a signup form",
		Description: "The form's hosted page follows the new name (/subscribe/<name>); embed code keeps working.",
		Tags:        tags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *formBody[SignupFormRenameInput]) (*api.Body[SignupForm], error) {
			out, err := s.Rename(ctx, api.WorkspaceFrom(ctx), in.Form, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "set-signup-form-status", Path: "/workspaces/{workspace}/forms/{form}/status", Summary: "Pause or resume a signup form",
		Tags: tags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *formBody[SignupFormStatusInput]) (*api.Body[SignupForm], error) {
			out, err := s.SetStatus(ctx, api.WorkspaceFrom(ctx), in.Form, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-signup-form", Path: "/workspaces/{workspace}/forms/{form}", Summary: "Delete a signup form",
		Description: "Contacts the form brought in stay. Embedded copies stop accepting submissions.",
		Tags:        tags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *FormPath) (*api.Empty, error) {
			return api.NoContent, s.Delete(ctx, api.WorkspaceFrom(ctx), in.Form)
		})
	api.Post(h, api.Op{ID: "allow-signup-form-origin", Path: "/workspaces/{workspace}/forms/{form}/allowed-origins", Summary: "Allow a site to submit a form",
		Tags: tags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *formBody[SignupFormOriginInput]) (*api.Body[SignupForm], error) {
			out, err := s.AllowOrigin(ctx, api.WorkspaceFrom(ctx), in.Form, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "keep-signup-form-origin-blocked", Path: "/workspaces/{workspace}/forms/{form}/blocked-origins/keep", Summary: "Keep a rejected site blocked and dismiss its alert",
		Tags: tags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *formBody[SignupFormOriginInput]) (*api.Body[SignupForm], error) {
			out, err := s.KeepBlocked(ctx, api.WorkspaceFrom(ctx), in.Form, in.Body)
			return api.OK(out), err
		})

	ptags := []string{"Public pages"}
	api.Get(h, api.Op{ID: "get-public-pages", Path: "/workspaces/{workspace}/public-pages", Summary: "Get the public pages' branding, options, domain and Turnstile set-up",
		Description: "The Turnstile secret is write-only: only whether one is stored is returned.",
		Tags:        ptags, Access: api.Workspace(viewPerm)},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[PublicPagesSettings], error) {
			out, err := s.PublicPages(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-public-branding", Path: "/workspaces/{workspace}/public-pages/branding", Summary: "Save the public page branding",
		Description: "Hover, tint and link colours are derived from the accent and the background so every text keeps WCAG 2.2 AA contrast.",
		Tags:        ptags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *wsBody[PublicBrandingInput]) (*api.Body[PublicPagesSettings], error) {
			out, err := s.SaveBranding(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "upload-public-logo", Path: "/workspaces/{workspace}/public-pages/logo", Summary: "Upload the publisher's logo",
		Tags: ptags, Access: api.Workspace(authz.FormsEdit), MaxBodyBytes: MaxLogoBytes + 64<<10},
		func(ctx context.Context, in *logoInput) (*api.Body[PublicPagesSettings], error) {
			f := in.RawBody.Data().File
			if !f.IsSet {
				return nil, api.Invalid("body.file", "Choose an image to upload.")
			}
			data, err := io.ReadAll(io.LimitReader(f, MaxLogoBytes+1))
			if err != nil {
				return nil, api.Invalid("body.file", "The upload was interrupted. Try again.")
			}
			out, err := s.UploadLogo(ctx, api.WorkspaceFrom(ctx), f.Filename, data)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-public-logo", Path: "/workspaces/{workspace}/public-pages/logo", Summary: "Remove the logo (the pages show the publisher's initial)",
		Tags: ptags, Access: api.Workspace(authz.FormsEdit), Status: 200},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[PublicPagesSettings], error) {
			out, err := s.RemoveLogo(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-public-domain", Path: "/workspaces/{workspace}/public-pages/domain", Summary: "Set the custom domain of the public pages",
		Description: "The pages move to the domain once its DNS points at this server (a CNAME, matching addresses or the TXT record _reputecast.<domain>).",
		Tags:        ptags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *wsBody[PublicDomainInput]) (*api.Body[PublicPagesSettings], error) {
			out, err := s.SetDomain(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-public-domain", Path: "/workspaces/{workspace}/public-pages/domain/check", Summary: "Check the custom domain's DNS now",
		Tags: ptags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[PublicPagesSettings], error) {
			out, err := s.CheckDomain(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-public-pages", Path: "/workspaces/{workspace}/public-pages/pages", Summary: "Save the options of the public pages",
		Description: "Send the pages to change; the others keep their options.",
		Tags:        ptags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *wsBody[PublicPageInput]) (*api.Body[PublicPagesSettings], error) {
			out, err := s.SavePage(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-archive-issues", Path: "/workspaces/{workspace}/public-pages/archive/issues", Summary: "List the archive's issues, hidden ones included",
		Description: "Sent campaigns of the archived lists are filed the first time the archive is read after they finish.",
		Tags:        ptags, Access: api.Workspace(viewPerm)},
		func(ctx context.Context, in *archiveListInput) (*api.Body[ArchiveIssuePage], error) {
			out, err := s.ArchiveIssues(ctx, api.WorkspaceFrom(ctx), in.Q, in.List, in.Page)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-archive-issue", Path: "/workspaces/{workspace}/public-pages/archive/issues/{campaign}", Summary: "Hide, show or retitle an archived issue",
		Tags: ptags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *archiveIssueInput) (*api.Body[ArchiveIssueItem], error) {
			out, err := s.UpdateArchiveIssue(ctx, api.WorkspaceFrom(ctx), in.Campaign, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-unsubscribe-stats", Path: "/workspaces/{workspace}/public-pages/unsubscribe/stats", Summary: "Unsubscribes from the public pages in the last 30 days",
		Tags: ptags, Access: api.Workspace(viewPerm)},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[UnsubscribeStats], error) {
			out, err := s.UnsubscribeNumbers(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-public-turnstile", Path: "/workspaces/{workspace}/public-pages/turnstile", Summary: "Save the Cloudflare Turnstile keys",
		Description: "The secret is stored encrypted and never returned. Cloudflare checks it when it's saved.",
		Tags:        ptags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *wsBody[PublicTurnstileInput]) (*api.Body[PublicPagesSettings], error) {
			out, err := s.SaveTurnstile(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-public-turnstile", Path: "/workspaces/{workspace}/public-pages/turnstile", Summary: "Remove the Turnstile keys",
		Tags: ptags, Access: api.Workspace(authz.FormsEdit), Status: 200},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[PublicPagesSettings], error) {
			out, err := s.RemoveTurnstile(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
}
