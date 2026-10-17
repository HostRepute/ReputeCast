package forms

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// LanguagePath names a content language in the path: a workspace
// language's code ("es"; "es-MX" is es too).
type LanguagePath struct {
	Language string `path:"language" pattern:"^[A-Za-z]{2,3}([-_][A-Za-z0-9]{2,8})*$" maxLength:"35" doc:"A workspace language (es)"`
}

type pageCopyInput struct {
	api.WorkspaceInput
	Language string `query:"language" maxLength:"35" doc:"Also return this language's strings"`
}

type pageCopySave struct {
	api.WorkspaceInput
	LanguagePath
	Body PublicPageCopyInput
}

type pageLabelsPath struct {
	api.WorkspaceInput
	LanguagePath
}

type pageLabelsSave struct {
	api.WorkspaceInput
	LanguagePath
	Body PublicPageLabelsInput
}

type formCopyPath struct {
	FormPath
	LanguagePath
}

type formCopySave struct {
	FormPath
	LanguagePath
	Body SignupFormCopyText
}

// registerCopy adds the page and form copy operations (design expansion
// M17): reading needs subscribers:view, every change forms:edit.
func registerCopy(h huma.API, s *Service) {
	ptags := []string{"Public pages"}
	api.Get(h, api.Op{ID: "get-public-page-copy", Path: "/workspaces/{workspace}/public-pages/copy", Summary: "Get the public pages' copy in every language",
		Description: "Each workspace language's status (Live, Incomplete with the strings still to write, or Not set up) and, with ?language=, that language's strings next to the pages' own.",
		Tags:        ptags, Access: api.Workspace(viewPerm)},
		func(ctx context.Context, in *pageCopyInput) (*api.Body[PublicPageCopy], error) {
			out, err := s.PageCopy(ctx, api.WorkspaceFrom(ctx), in.Language)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-public-page-copy", Path: "/workspaces/{workspace}/public-pages/copy/{language}", Summary: "Save the public pages' own text in a language",
		Description: "Send the strings to change; an empty text removes a translation. A language's pages are shown once every string and its built-in labels are written.",
		Tags:        ptags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *pageCopySave) (*api.Body[PublicPageCopy], error) {
			out, err := s.SavePageCopy(ctx, api.WorkspaceFrom(ctx), in.Language, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-public-page-labels", Path: "/workspaces/{workspace}/public-pages/labels/{language}", Summary: "Get the built-in labels of a language",
		Description: "Buttons, field labels, messages and the confirmation email. ReputeCast ships them in English and Vietnamese; for other languages the workspace writes them.",
		Tags:        ptags, Access: api.Workspace(viewPerm)},
		func(ctx context.Context, in *pageLabelsPath) (*api.Body[PublicPageLabels], error) {
			out, err := s.PageLabels(ctx, api.WorkspaceFrom(ctx), in.Language)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-public-page-labels", Path: "/workspaces/{workspace}/public-pages/labels/{language}", Summary: "Save the built-in labels the workspace writes for a language",
		Description: "Each text may use only its English label's placeholders and tags. Languages ReputeCast ships answer 409.",
		Tags:        ptags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *pageLabelsSave) (*api.Body[PublicPageLabels], error) {
			out, err := s.SavePageLabels(ctx, api.WorkspaceFrom(ctx), in.Language, in.Body)
			return api.OK(out), err
		})

	tags := []string{"Forms"}
	api.Get(h, api.Op{ID: "get-signup-form-copy", Path: "/workspaces/{workspace}/forms/{form}/copy", Summary: "Get a signup form's copy in every language",
		Description: "The form's language mode, each language's status and text, its consent text's version and the latest consent given in it.",
		Tags:        tags, Access: api.Workspace(viewPerm)},
		func(ctx context.Context, in *FormPath) (*api.Body[SignupFormCopy], error) {
			out, err := s.FormCopy(ctx, api.WorkspaceFrom(ctx), in.Form)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-signup-form-copy", Path: "/workspaces/{workspace}/forms/{form}/copy/{language}", Summary: "Save a signup form's text in a language",
		Description: "In the form's own language this is its content and consent text. Changing a consent text starts a new version of it in that language; consent evidence keeps the version and language shown.",
		Tags:        tags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *formCopySave) (*api.Body[SignupFormCopy], error) {
			out, err := s.SaveFormCopy(ctx, api.WorkspaceFrom(ctx), in.Form, in.Language, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "confirm-signup-form-consent", Path: "/workspaces/{workspace}/forms/{form}/copy/{language}/consent-reviewed", Summary: "Mark a language's consent text still right",
		Description: "Clears \"Review the translation\" after the form's own consent text changed, keeping the translation and its version.",
		Tags:        tags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *formCopyPath) (*api.Body[SignupFormCopy], error) {
			out, err := s.ConfirmFormConsent(ctx, api.WorkspaceFrom(ctx), in.Form, in.Language)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-signup-form-language", Path: "/workspaces/{workspace}/forms/{form}/language", Summary: "Pick which language a signup form shows",
		Description: "page: the lang of the page it's on; browser: the visitor's browser; fixed: always one language. A language whose copy isn't Live shows the fallback.",
		Tags:        tags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *formBody[SignupFormLanguageInput]) (*api.Body[SignupFormCopy], error) {
			out, err := s.SetFormLanguage(ctx, api.WorkspaceFrom(ctx), in.Form, in.Body)
			return api.OK(out), err
		})
}
