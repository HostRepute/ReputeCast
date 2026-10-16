package transactional

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// Base is the path of every transactional operation.
const Base = "/workspaces/{workspace}/transactional"

var (
	tagTemplates = []string{"Transactional templates"}
	tagSettings  = []string{"Transactional settings"}
)

// TemplatePath names a template by its ID (password-reset) or its UUID.
type TemplatePath struct {
	api.WorkspaceInput
	Template string `path:"template" maxLength:"64" doc:"The template's ID, such as password-reset, or its UUID" example:"password-reset"`
}

type templateListInput struct {
	api.WorkspaceInput
	Period      string `query:"period" enum:"24h,7d,30d" default:"7d" doc:"The period of the numbers"`
	IncludeTest bool   `query:"includeTest" doc:"Count messages sent with test-mode keys"`
}

type templateCreateInput struct {
	api.WorkspaceInput
	Body TransactionalTemplateCreate
}

type templateUpdateInput struct {
	TemplatePath
	Body TransactionalTemplateUpdate
}

type draftSaveInput struct {
	TemplatePath
	Language string `query:"language" maxLength:"16" doc:"Saves a language version instead of the draft (internal/languages)"`
	Body     TransactionalDraftSave
}

type previewInput struct {
	TemplatePath
	Body TransactionalPreviewInput
}

type checksInput struct {
	TemplatePath
	Body TransactionalChecksInput
}

type publishPreviewInput struct {
	TemplatePath
	Body TransactionalPublishPreviewInput
}

type publishInput struct {
	TemplatePath
	Body TransactionalPublishInput
}

type rollbackInput struct {
	TemplatePath
	Body TransactionalRollbackInput
}

type versionInput struct {
	TemplatePath
	Version int `path:"version" minimum:"1"`
}

type renderInput struct {
	TemplatePath
	Body TransactionalRenderInput
}

// CategoryPath names a category.
type CategoryPath struct {
	api.WorkspaceInput
	CategoryID string `path:"categoryId" format:"uuid"`
}

func (in CategoryPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.CategoryID)
	if err != nil {
		return uuid.Nil, api.NotFound("Category")
	}
	return id, nil
}

type categoryInput struct {
	api.WorkspaceInput
	Body TransactionalCategoryInput
}

type categoryUpdateInput struct {
	CategoryPath
	Body TransactionalCategoryInput
}

type settingsInput struct {
	api.WorkspaceInput
	Body TransactionalSettingsInput
}

// RegisterTemplates adds the template, category and settings operations.
func RegisterTemplates(h huma.API, s *Service) {
	view, edit, publish := api.Workspace(authz.TransactionalView), api.Workspace(authz.TransactionalEdit), api.Workspace(authz.TransactionalPublish)

	api.Get(h, api.Op{ID: "list-transactional-templates", Path: Base + "/templates", Summary: "List templates with their numbers over a period",
		Description: "Each template's live version, draft state, sent, delivery, bounce and complaint rates, median delivery time, last send and a sparkline. Test messages are left out unless includeTest.",
		Tags:        tagTemplates, Access: view},
		func(ctx context.Context, in *templateListInput) (*api.Body[TransactionalTemplateList], error) {
			out, err := s.Templates(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Period, in.IncludeTest)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-transactional-template", Path: Base + "/templates", Summary: "Create a template: blank, from a starter, as a copy, or with imported content",
		Description: "The template starts as a draft. Nothing is sent until it is published.", Tags: tagTemplates, Access: edit, Status: 201, MaxBodyBytes: 8 << 20},
		func(ctx context.Context, in *templateCreateInput) (*api.Body[TransactionalTemplate], error) {
			out, err := s.CreateTemplate(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-transactional-template", Path: Base + "/templates/{template}", Summary: "Get a template with its draft, variables, test data and live version",
		Tags: tagTemplates, Access: view},
		func(ctx context.Context, in *templateGetInput) (*api.Body[TransactionalTemplate], error) {
			out, err := s.TemplateIn(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Template, in.Language)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-transactional-template", Path: Base + "/templates/{template}", Summary: "Rename a template, change its category, or its ID before the first publish",
		Tags: tagTemplates, Access: edit},
		func(ctx context.Context, in *templateUpdateInput) (*api.Body[TransactionalTemplate], error) {
			out, err := s.UpdateTemplate(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Template, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-transactional-template-draft", Path: Base + "/templates/{template}/draft", Summary: "Save the draft (autosave)",
		Description: "Content, variables, test data, sender, reply-to and tracking. Fails with 409 version_conflict when someone saved a newer revision since baseRevision; the problem's meta says who, when and what each side changed. Send force: true to keep yours.",
		Tags:        tagTemplates, Access: edit, MaxBodyBytes: 16 << 20},
		func(ctx context.Context, in *draftSaveInput) (*api.Body[TransactionalDraftSaved], error) {
			out, err := s.SaveDraftIn(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Template, in.Language, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "preview-transactional-template", Path: Base + "/templates/{template}/preview", Summary: "Render the draft or a version with test data (nothing is sent)",
		Tags: tagTemplates, Access: view, MaxBodyBytes: 8 << 20},
		func(ctx context.Context, in *previewInput) (*api.Body[TransactionalPreview], error) {
			out, err := s.Preview(ctx, api.WorkspaceFrom(ctx), in.Template, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-transactional-template", Path: Base + "/templates/{template}/checks", Summary: "Run the template's checks on the draft or the editor's unsaved content",
		Description: "The composer's content checks (size, alt text, links, plain text; unsubscribe link and footer address for Optional categories) and the transactional ones: Liquid errors, undeclared and unused variables, test data that fails to render, a sensitive variable in the subject, tracking on an Essential template.",
		Tags:        tagTemplates, Access: view, MaxBodyBytes: 8 << 20},
		func(ctx context.Context, in *checksInput) (*api.Body[TransactionalChecks], error) {
			out, err := s.TemplateChecks(ctx, api.WorkspaceFrom(ctx), in.Template, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "preview-transactional-publish", Path: Base + "/templates/{template}/publish-preview", Summary: "What publishing the draft would change",
		Description: "The diff since the live version, the breaking-change check against the calls of the last 7 days, the checks, test renders with each test data set and the last real call, and whether the workspace's approval rule applies. makeOptional previews the recommended fix.",
		Tags:        tagTemplates, Access: view},
		func(ctx context.Context, in *publishPreviewInput) (*api.Body[TransactionalPublishPreview], error) {
			out, err := s.PublishPreview(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Template, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "publish-transactional-template", Path: Base + "/templates/{template}/publish", Summary: "Publish the draft as the next version",
		Description: "Fails with 409 breaking_change when a newly required variable is missing from recent calls (fix it with makeOptional, or send publishAnyway), 409 approval_required when the workspace's rule applies, 409 version_conflict when the draft changed since baseRevision, and 409 template_does_not_render when a check blocks.",
		Tags:        tagTemplates, Access: publish},
		func(ctx context.Context, in *publishInput) (*api.Body[TransactionalPublished], error) {
			out, err := s.Publish(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Template, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "rollback-transactional-template", Path: Base + "/templates/{template}/rollback", Summary: "Publish an earlier version again (roll back, or undo a publish)",
		Tags: tagTemplates, Access: publish},
		func(ctx context.Context, in *rollbackInput) (*api.Body[TransactionalPublished], error) {
			out, err := s.Rollback(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Template, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-transactional-template-versions", Path: Base + "/templates/{template}/versions", Summary: "Published versions, newest first",
		Tags: tagTemplates, Access: view},
		func(ctx context.Context, in *TemplatePath) (*api.Body[TransactionalVersionList], error) {
			out, err := s.Versions(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Template)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-transactional-template-version", Path: Base + "/templates/{template}/versions/{version}", Summary: "One published version with its content",
		Tags: tagTemplates, Access: view},
		func(ctx context.Context, in *versionInput) (*api.Body[TransactionalVersionDetail], error) {
			out, err := s.Version(ctx, api.WorkspaceFrom(ctx), in.Template, in.Version)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "archive-transactional-template", Path: Base + "/templates/{template}/archive", Summary: "Archive a template: calls answer template_not_found until it is restored",
		Tags: tagTemplates, Access: publish},
		func(ctx context.Context, in *TemplatePath) (*api.Body[TransactionalTemplate], error) {
			out, err := s.Archive(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Template, true)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "restore-transactional-template", Path: Base + "/templates/{template}/restore", Summary: "Restore an archived template",
		Tags: tagTemplates, Access: publish},
		func(ctx context.Context, in *TemplatePath) (*api.Body[TransactionalTemplate], error) {
			out, err := s.Archive(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Template, false)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-transactional-template", Path: Base + "/templates/{template}", Summary: "Delete a draft or archived template (its messages stay in the log)",
		Tags: tagTemplates, Access: publish},
		func(ctx context.Context, in *TemplatePath) (*api.Empty, error) {
			return api.NoContent, s.DeleteTemplate(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Template)
		})
	api.Post(h, api.Op{ID: "render-transactional-template", Path: Base + "/templates/{template}/render", Summary: "Render the live version with a call's data, without sending",
		Description: "Returns the subject, HTML and text a send would use. Errors: template_not_found, template_not_published, missing_variables (listing them in missing).",
		Tags:        tagTemplates, Access: api.Workspace(authz.TransactionalSend), MaxBodyBytes: 4 << 20},
		func(ctx context.Context, in *renderInput) (*api.Body[TransactionalRendered], error) {
			if err := refuseAgents(ctx); err != nil {
				return nil, err
			}
			out, err := s.Render(ctx, api.WorkspaceFrom(ctx), in.Template, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-transactional-starters", Path: Base + "/starters", Summary: "Starter templates with their variables and test data",
		Tags: tagTemplates, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[TransactionalStarterList], error) {
			out, err := s.Starters(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "list-transactional-categories", Path: Base + "/categories", Summary: "Categories with their templates and opt-outs",
		Tags: tagSettings, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[TransactionalCategoryList], error) {
			out, err := s.Categories(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-transactional-category", Path: Base + "/categories", Summary: "Add a category", Tags: tagSettings, Access: publish, Status: 201},
		func(ctx context.Context, in *categoryInput) (*api.Body[TransactionalCategory], error) {
			out, err := s.CreateCategory(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-transactional-category", Path: Base + "/categories/{categoryId}", Summary: "Change a category (its kind is fixed while it has templates)",
		Tags: tagSettings, Access: publish},
		func(ctx context.Context, in *categoryUpdateInput) (*api.Body[TransactionalCategory], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.UpdateCategory(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-transactional-category", Path: Base + "/categories/{categoryId}", Summary: "Delete a category without templates",
		Tags: tagSettings, Access: publish},
		func(ctx context.Context, in *CategoryPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.DeleteCategory(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Get(h, api.Op{ID: "get-transactional-settings", Path: Base + "/settings", Summary: "Contact linking, content retention, delivery alerts and template defaults",
		Tags: tagSettings, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[TransactionalSettings], error) {
			out, err := s.Settings(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-transactional-settings", Path: Base + "/settings", Summary: "Save Settings › Transactional", Tags: tagSettings, Access: publish},
		func(ctx context.Context, in *settingsInput) (*api.Body[TransactionalSettings], error) {
			out, err := s.SaveSettings(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
}

// refuseAgents keeps MCP agents from sending: they propose, people and
// send-only keys send.
func refuseAgents(ctx context.Context) error {
	if p := api.PrincipalFrom(ctx); p != nil && p.Type == authz.ActorMCPAgent {
		return problem(403, CodeAgentsNeverSend, "MCP agents never send transactional mail. Your app sends with a send-only key.")
	}
	return nil
}
