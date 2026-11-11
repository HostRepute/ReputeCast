package leads

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// LeadAdsConnectionPath names a connection in a path.
type LeadAdsConnectionPath struct {
	api.WorkspaceInput
	ID string `path:"id" doc:"The connection's id"`
}

func (in LeadAdsConnectionPath) id(ctx context.Context) (uuid.UUID, error) {
	return parseID(ctx, in.ID, mConnectionNotFound)
}

// LeadAdsFormPath names a lead form in a path.
type LeadAdsFormPath struct {
	api.WorkspaceInput
	FormID string `path:"formId" doc:"The lead form's id in ReputeCast"`
}

func (in LeadAdsFormPath) id(ctx context.Context) (uuid.UUID, error) {
	return parseID(ctx, in.FormID, mFormNotFound)
}

func parseID(ctx context.Context, raw string, missing i18n.Msg) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, api.Err(404, api.CodeNotFound, missing.Text(i18n.LangFrom(ctx), nil))
	}
	return id, nil
}

// Register adds the lead ads operations: reading needs integrations:view,
// every change integrations:manage and a signed-in member.
func Register(h huma.API, s *Service) {
	tags := []string{"Lead ads"}
	base := "/workspaces/{workspace}/lead-ads"
	view, manage := api.Workspace(authz.IntegrationsView), api.WorkspaceHuman(authz.IntegrationsManage)

	api.Get(h, api.Op{ID: "get-lead-ads-setup", Path: base + "/setup", Summary: "What connecting Meta Lead Ads needs: the token's permissions and whether Meta can reach this server",
		Tags: tags, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[LeadAdsSetup], error) {
			out, err := s.Setup(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-lead-ads-connections", Path: base + "/connections", Summary: "The workspace's Meta Lead Ads connections",
		Tags: tags, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[LeadAdsConnectionList], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "connect-lead-ads", Path: base + "/connections", Summary: "Connect a Meta app with a system user's token (checked with Meta, kept sealed)",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body LeadAdsConnectInput
		}) (*api.Body[LeadAdsConnection], error) {
			out, err := s.Connect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-lead-ads-connection", Path: base + "/connections/{id}", Summary: "A Meta connection: how leads arrive, its Pages and forms",
		Tags: tags, Access: view},
		func(ctx context.Context, in *LeadAdsConnectionPath) (*api.Body[LeadAdsConnection], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-lead-ads-pages", Path: base + "/connections/{id}/pages", Summary: "The Facebook Pages the token reaches, the chosen ones marked (read from Meta)",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *LeadAdsConnectionPath) (*api.Body[LeadAdsPageChoices], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.PageChoices(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "choose-lead-ads-pages", Path: base + "/connections/{id}/pages", Summary: "Choose the Pages whose leads are collected: they're subscribed, the others unsubscribed",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *struct {
			LeadAdsConnectionPath
			Body LeadAdsPagesInput
		}) (*api.Body[LeadAdsConnection], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.ChoosePages(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "refresh-lead-ads-forms", Path: base + "/connections/{id}/refresh", Summary: "List the chosen Pages' lead forms again",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *LeadAdsConnectionPath) (*api.Body[LeadAdsConnection], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.Refresh(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "replace-lead-ads-credentials", Path: base + "/connections/{id}/credentials", Summary: "Replace the connection's name, app id, app secret or token (blank secrets stay)",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *struct {
			LeadAdsConnectionPath
			Body LeadAdsCredentialsInput
		}) (*api.Body[LeadAdsConnection], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.ReplaceCredentials(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "disconnect-lead-ads", Path: base + "/connections/{id}/disconnect", Summary: "Stop collecting: unsubscribe the Pages and delete the credentials (contacts stay)",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *LeadAdsConnectionPath) (*api.Body[LeadAdsConnection], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.Disconnect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-lead-ads-activity", Path: base + "/connections/{id}/activity", Summary: "The connection's last 50 leads: when, which form, what happened and the contact (no answers)",
		Tags: tags, Access: view},
		func(ctx context.Context, in *LeadAdsConnectionPath) (*api.Body[LeadAdsActivity], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.Activity(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-lead-ads-form", Path: base + "/forms/{formId}", Summary: "A lead form's questions, consent checkboxes and mapping (read from Meta when it answers)",
		Tags: tags, Access: view},
		func(ctx context.Context, in *LeadAdsFormPath) (*api.Body[LeadAdsFormDetail], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.FormDetail(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "map-lead-ads-form", Path: base + "/forms/{formId}/mapping", Summary: "Map a lead form: its questions to fields, its lists, tags and consent",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *struct {
			LeadAdsFormPath
			Body LeadAdsMappingInput
		}) (*api.Body[LeadAdsFormDetail], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.SaveMapping(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	for _, st := range []struct{ id, path, summary, status string }{
		{"pause-lead-ads-form", "/pause", "Pause a lead form: its leads wait at Meta", "paused"},
		{"resume-lead-ads-form", "/resume", "Resume a lead form", "active"},
	} {
		api.Post(h, api.Op{ID: st.id, Path: base + "/forms/{formId}" + st.path, Summary: st.summary, Tags: tags, Access: manage},
			func(ctx context.Context, in *LeadAdsFormPath) (*api.Body[LeadAdsFormDetail], error) {
				id, err := in.id(ctx)
				if err != nil {
					return nil, err
				}
				out, err := s.SetFormStatus(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, st.status)
				return api.OK(out), err
			})
	}
	api.Post(h, api.Op{ID: "import-lead-ads-form", Path: base + "/forms/{formId}/import", Summary: "Import a lead form's leads of the last 90 days (each once)",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *LeadAdsFormPath) (*api.Body[LeadAdsFormDetail], error) {
			id, err := in.id(ctx)
			if err != nil {
				return nil, err
			}
			out, err := s.StartImport(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
}
