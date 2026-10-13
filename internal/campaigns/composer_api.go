package campaigns

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/mjml"
)

// CampaignPath is the {workspace}/{campaignId} part of composer operations.
type CampaignPath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid"`
}

func (in CampaignPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.CampaignID)
	if err != nil {
		return uuid.Nil, api.NotFound("Campaign")
	}
	return id, nil
}

type createCampaignInput struct {
	api.WorkspaceInput
	Body ComposerCreateCampaign
}

type updateCampaignInput struct {
	CampaignPath
	Body ComposerUpdateCampaign
}

type saveContentInput struct {
	CampaignPath
	Body ComposerSaveContent
}

type versionsInput struct {
	CampaignPath
	Limit int `query:"limit" minimum:"1" maximum:"300" default:"50"`
}

// VersionPath adds a {version}.
type VersionPath struct {
	CampaignPath
	Version int `path:"version" minimum:"1"`
}

type restoreInput struct {
	VersionPath
	Body ComposerRestoreVersion
}

type testSendInput struct {
	CampaignPath
	Body ComposerTestSend
}

type previewInput struct {
	CampaignPath
	Body ComposerPreviewRequest
}

type previewContactsInput struct {
	CampaignPath
	Query string `query:"q" maxLength:"200"`
}

type checksInput struct {
	CampaignPath
	Body ComposerChecksRequest
}

type contextInput struct {
	api.WorkspaceInput
}

type audienceCountInput struct {
	api.WorkspaceInput
	Body ComposerAudienceInput
}

// ComposerCompileRequest is MJML to compile on the server.
type ComposerCompileRequest struct {
	MJML string `json:"mjml" minLength:"1" maxLength:"2000000"`
}

// ComposerCompiled is the server-side compile result (gomjml, MJML 4.15).
type ComposerCompiled struct {
	HTML       string             `json:"html"`
	Errors     []mjml.MJMLProblem `json:"errors"`
	Warnings   []mjml.MJMLProblem `json:"warnings"`
	SizeBytes  int                `json:"sizeBytes"`
	DurationMs int64              `json:"durationMs"`
}

type compileInput struct {
	api.WorkspaceInput
	Body ComposerCompileRequest
}

// RegisterComposer adds the composer operations.
func RegisterComposer(h huma.API, s *Service) {
	tags := []string{"Campaigns"}
	view, edit := api.Workspace(authz.CampaignsView), api.Workspace(authz.CampaignsEdit)

	api.Post(h, api.Op{ID: "create-campaign", Path: "/workspaces/{workspace}/campaigns", Summary: "Start a draft, blank or from a template or another campaign",
		Tags: tags, Access: edit, Status: 201},
		func(ctx context.Context, in *createCampaignInput) (*api.Body[ComposerCampaign], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-campaign", Path: "/workspaces/{workspace}/campaigns/{campaignId}", Summary: "Get a campaign with its content, audience, sender and route",
		Tags: tags, Access: view},
		func(ctx context.Context, in *CampaignPath) (*api.Body[ComposerCampaign], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Detail(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-campaign", Path: "/workspaces/{workspace}/campaigns/{campaignId}", Summary: "Rename a campaign or change its sender, reply-to, audience or content settings",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *updateCampaignInput) (*api.Body[ComposerCampaign], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Update(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-campaign-content", Path: "/workspaces/{workspace}/campaigns/{campaignId}/content", Summary: "Save the content as a new version (autosave)",
		Description: "Fails with 409 version_conflict when someone saved a newer version since baseVersion; the problem's meta says who, when and which sections each side changed. Send force: true to keep yours on top.",
		Tags:        tags, Access: edit, MaxBodyBytes: 16 << 20},
		func(ctx context.Context, in *saveContentInput) (*api.Body[ComposerSavedContent], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Save(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-campaign-versions", Path: "/workspaces/{workspace}/campaigns/{campaignId}/versions", Summary: "Version history, newest first",
		Tags: tags, Access: view},
		func(ctx context.Context, in *versionsInput) (*api.Body[ComposerVersionList], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Versions(ctx, api.WorkspaceFrom(ctx), id, in.Limit)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-campaign-version", Path: "/workspaces/{workspace}/campaigns/{campaignId}/versions/{version}", Summary: "One version with its content",
		Tags: tags, Access: view},
		func(ctx context.Context, in *VersionPath) (*api.Body[ComposerVersionDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Version(ctx, api.WorkspaceFrom(ctx), id, in.Version)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "restore-campaign-version", Path: "/workspaces/{workspace}/campaigns/{campaignId}/versions/{version}/restore", Summary: "Restore a version (saved as a new version)",
		Tags: tags, Access: edit},
		func(ctx context.Context, in *restoreInput) (*api.Body[ComposerSavedContent], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Restore(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Version, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "duplicate-campaign", Path: "/workspaces/{workspace}/campaigns/{campaignId}/duplicate", Summary: "Copy a campaign into a new draft",
		Tags: tags, Access: edit, Status: 201},
		func(ctx context.Context, in *CampaignPath) (*api.Body[ComposerCampaign], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Duplicate(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "send-campaign-test", Path: "/workspaces/{workspace}/campaigns/{campaignId}/test", Summary: "Send a test email of the saved content",
		Description: "Goes through the System route. Without campaigns:send, test emails can only go to your own address.",
		Tags:        tags, Access: edit},
		func(ctx context.Context, in *testSendInput) (*api.Body[ComposerTestSendResult], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SendTest(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "preview-campaign", Path: "/workspaces/{workspace}/campaigns/{campaignId}/preview", Summary: "Render the email for one recipient (personalization, links)",
		Tags: tags, Access: view, MaxBodyBytes: 6 << 20},
		func(ctx context.Context, in *previewInput) (*api.Body[ComposerPreview], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Preview(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-preview-contacts", Path: "/workspaces/{workspace}/campaigns/{campaignId}/preview-contacts", Summary: "Contacts of the audience to preview as",
		Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *previewContactsInput) (*api.Body[ComposerPreviewContactList], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.PreviewContacts(ctx, api.WorkspaceFrom(ctx), id, in.Query)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-campaign-content", Path: "/workspaces/{workspace}/campaigns/{campaignId}/checks", Summary: "Run the content checks on the composer's current content",
		Description: "Size against Gmail's 102 KB clipping limit, alt text, image-to-text ratio, blocklisted and shortened links, unsubscribe link and postal address, personalization, subject heuristics and the plain-text part. Nothing is saved.",
		Tags:        tags, Access: view, MaxBodyBytes: 6 << 20},
		func(ctx context.Context, in *checksInput) (*api.Body[ComposerChecks], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Checks(ctx, api.WorkspaceFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "composer-context", Path: "/workspaces/{workspace}/composer", Summary: "Senders, lists, segments, personalization tokens, team and route for the composer",
		Tags: tags, Access: view},
		func(ctx context.Context, _ *contextInput) (*api.Body[ComposerContext], error) {
			out, err := s.Context(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "count-campaign-audience", Path: "/workspaces/{workspace}/campaigns/audience-count", Summary: "Live recipient count of an audience definition",
		Tags: tags, Access: view},
		func(ctx context.Context, in *audienceCountInput) (*api.Body[ComposerAudienceCount], error) {
			out, err := s.CountAudience(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body.def())
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "compile-mjml", Path: "/workspaces/{workspace}/mjml/compile", Summary: "Compile MJML to HTML on the server (gomjml, MJML 4)",
		Description: "The composer compiles in the browser; this endpoint serves API clients and automations, and reports layout warnings.",
		Tags:        tags, Access: view, MaxBodyBytes: 4 << 20},
		func(ctx context.Context, in *compileInput) (*api.Body[ComposerCompiled], error) {
			res := mjml.Compile(in.Body.MJML)
			return api.OK(ComposerCompiled{HTML: res.HTML, Errors: nonNilProblems(res.Errors), Warnings: mjml.Lint(in.Body.MJML), SizeBytes: len(res.HTML), DurationMs: res.Duration.Milliseconds()}), nil
		})
}

func nonNilProblems(p []mjml.MJMLProblem) []mjml.MJMLProblem {
	if p == nil {
		return []mjml.MJMLProblem{}
	}
	return p
}
