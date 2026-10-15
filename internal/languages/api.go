package languages

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/campaigns"
)

type wsInput struct {
	api.WorkspaceInput
}

type catalogInput struct {
	api.WorkspaceInput
	Query string `query:"q" maxLength:"100" doc:"Search by name, own name or code"`
}

// LanguageCatalogList is the languages that can be added.
type LanguageCatalogList struct {
	Items []CatalogLanguage `json:"items"`
}

// AddWorkspaceLanguage adds a language to the workspace.
type AddWorkspaceLanguage struct {
	Code string `json:"code" pattern:"^[a-z]{2,3}$" doc:"A base language code, such as fr"`
}

type addInput struct {
	api.WorkspaceInput
	Body AddWorkspaceLanguage
}

type codeInput struct {
	api.WorkspaceInput
	Code string `path:"code" pattern:"^[a-z]{2,3}$"`
}

type defaultsInput struct {
	api.WorkspaceInput
	Body LanguageDefaults
}

type fallbackInput struct {
	api.WorkspaceInput
	Code string `query:"code" pattern:"^[a-z]{2,3}$" required:"true"`
}

// campaignPath is {workspace}/campaigns/{campaignId}.
type campaignPath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid"`
}

func (in campaignPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.CampaignID)
	if err != nil {
		return uuid.Nil, api.NotFound("Campaign")
	}
	return id, nil
}

// campaignLangPath adds a {language}.
type campaignLangPath struct {
	campaignPath
	Language string `path:"language" pattern:"^[a-z]{2,3}$"`
}

type addVersionInput struct {
	campaignPath
	Body AddLanguageInput
}

type markInput struct {
	campaignLangPath
	Body MarkInput
}

type holdInput struct {
	campaignLangPath
	Body HoldInput
}

type keepInput struct {
	campaignPath
	Body KeepUpToDateInput
}

type applyInput struct {
	campaignPath
	Body ApplyInput
}

type sourceInput struct {
	campaignPath
	Body SourceInput
}

type blockInput struct {
	campaignLangPath
	Body BlockInput
}

type historyInput struct {
	campaignLangPath
	Limit int `query:"limit" minimum:"1" maximum:"300" default:"50"`
}

// Register adds the content-languages operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Languages"}
	manage := api.Workspace(authz.WorkspaceManage)

	api.Get(h, api.Op{ID: "list-workspace-languages", Path: "/workspaces/{workspace}/languages", Summary: "The workspace's languages, default and fallback, with their contacts",
		Tags: tags, Access: api.Workspace("")},
		func(ctx context.Context, _ *wsInput) (*api.Body[WorkspaceLanguages], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-language-catalog", Path: "/workspaces/{workspace}/languages/catalog", Summary: "Languages that can be added, with the contacts who already have them",
		Tags: tags, Access: api.Workspace("")},
		func(ctx context.Context, in *catalogInput) (*api.Body[LanguageCatalogList], error) {
			out, err := s.LanguageCatalog(ctx, api.WorkspaceFrom(ctx), in.Query)
			return api.OK(LanguageCatalogList{Items: out}), err
		})
	api.Post(h, api.Op{ID: "add-workspace-language", Path: "/workspaces/{workspace}/languages", Summary: "Add a language to the workspace",
		Description: "Right-to-left languages aren't supported yet and answer 422. Adding a language that was removed brings back its versions and page copy.",
		Tags:        tags, Access: manage},
		func(ctx context.Context, in *addInput) (*api.Body[WorkspaceLanguages], error) {
			out, err := s.AddLanguage(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body.Code)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-language-removal", Path: "/workspaces/{workspace}/languages/{code}/removal", Summary: "What removing a language affects",
		Tags: tags, Access: api.Workspace("")},
		func(ctx context.Context, in *codeInput) (*api.Body[LanguageRemoval], error) {
			out, err := s.Removal(ctx, api.WorkspaceFrom(ctx), in.Code)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-workspace-language", Path: "/workspaces/{workspace}/languages/{code}", Summary: "Remove a language from the workspace",
		Description: "Its email versions and page copy are kept but no longer used; its contacts keep their language and get the fallback. The default and the fallback can't be removed (409).",
		Tags:        tags, Access: manage, Status: http.StatusOK},
		func(ctx context.Context, in *codeInput) (*api.Body[WorkspaceLanguages], error) {
			out, err := s.RemoveLanguage(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Code)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-language-defaults", Path: "/workspaces/{workspace}/languages/defaults", Summary: "Change the default and fallback languages",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *defaultsInput) (*api.Body[WorkspaceLanguages], error) {
			out, err := s.SetDefaults(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-fallback-impact", Path: "/workspaces/{workspace}/languages/fallback-impact", Summary: "How many contacts a new fallback language moves",
		Tags: tags, Access: api.Workspace("")},
		func(ctx context.Context, in *fallbackInput) (*api.Body[FallbackImpact], error) {
			out, err := s.FallbackPreview(ctx, api.WorkspaceFrom(ctx), in.Code)
			return api.OK(out), err
		})

	registerCampaign(h, s)
}

func registerCampaign(h huma.API, s *Service) {
	tags := []string{"Campaigns"}
	view, edit := api.Workspace(authz.CampaignsView), api.Workspace(authz.CampaignsEdit)
	base := "/workspaces/{workspace}/campaigns/{campaignId}/languages"

	api.Get(h, api.Op{ID: "list-campaign-languages", Path: base, Summary: "A campaign's language versions with their status and readers",
		Description: "Every workspace language is listed: the source, versions that are Up to date, Outdated (with the blocks that changed) or Draft, and Missing ones, whose readers get the fallback.",
		Tags:        tags, Access: view},
		func(ctx context.Context, in *campaignPath) (*api.Body[EmailLanguages], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.CampaignLanguages(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "add-campaign-language", Path: base, Summary: "Add a language version (a Draft) to a campaign",
		Description: "It starts from a copy of the source (layout and text) or from the layout only. A Draft is never sent: its readers get the fallback until it is marked translated.",
		Tags:        tags, Access: edit, Status: http.StatusCreated},
		func(ctx context.Context, in *addVersionInput) (*api.Body[EmailLanguages], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.AddCampaignLanguage(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-campaign-language", Path: base + "/{language}", Summary: "Remove a campaign's language version",
		Tags: tags, Access: edit, Status: http.StatusOK},
		func(ctx context.Context, in *campaignLangPath) (*api.Body[EmailLanguages], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.RemoveCampaignLanguage(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Language)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "mark-campaign-language-translated", Path: base + "/{language}/mark-translated", Summary: "Mark a Draft language version translated, so it is sent",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *campaignLangPath) (*api.Body[EmailLanguages], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.MarkCampaignLanguage(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Language, true, MarkInput{})
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "mark-campaign-language-up-to-date", Path: base + "/{language}/mark-up-to-date", Summary: "Mark an Outdated language version (or some of its blocks) up to date",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *markInput) (*api.Body[EmailLanguages], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.MarkCampaignLanguage(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Language, false, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "hold-campaign-language", Path: base + "/{language}/hold", Summary: "Send the fallback to a language's readers instead, or stop doing so",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *holdInput) (*api.Body[EmailLanguages], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.HoldCampaignLanguage(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Language, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "keep-campaign-languages-up-to-date", Path: base + "/keep-up-to-date", Summary: "Keep every language version up to date after a source edit that needs no translating",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *keepInput) (*api.Body[EmailLanguages], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.KeepCampaignUpToDate(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "apply-to-campaign-languages", Path: base + "/apply", Summary: "Apply a source link or image change to every language version",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *applyInput) (*api.Body[ApplyResult], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.ApplyCampaignChange(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-campaign-source-language", Path: base + "/source", Summary: "Change the language a campaign is written in (while it has one version)",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *sourceInput) (*api.Body[EmailLanguages], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SetCampaignSource(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "compare-campaign-language", Path: base + "/{language}/compare", Summary: "Compare a language version with the source, block by block",
		Tags: tags, Access: view},
		func(ctx context.Context, in *campaignLangPath) (*api.Body[LanguageCompare], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.CompareCampaign(ctx, api.WorkspaceFrom(ctx), id, in.Language)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "copy-block-to-campaign-language", Path: base + "/{language}/copy-block", Summary: "Copy a block new in the source into a language version",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *blockInput) (*api.Body[LanguageCompare], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.CopyBlockToCampaignLanguage(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Language, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "remove-block-from-campaign-language", Path: base + "/{language}/remove-block", Summary: "Remove a block the source no longer has from a language version",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *blockInput) (*api.Body[LanguageCompare], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.RemoveBlockFromCampaignLanguage(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Language, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-campaign-language-versions", Path: base + "/{language}/versions", Summary: "Saved versions of a campaign's language version, newest first",
		Tags: tags, Access: view},
		func(ctx context.Context, in *historyInput) (*api.Body[campaigns.ComposerVersionList], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.CampaignHistory(ctx, api.WorkspaceFrom(ctx), id, in.Language, in.Limit)
			return api.OK(out), err
		})
}
