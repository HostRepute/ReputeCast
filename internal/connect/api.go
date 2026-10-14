package connect

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type wsInput struct {
	api.WorkspaceInput
}

type setupIDInput struct {
	api.WorkspaceInput
	SetupID string `path:"setupId" format:"uuid"`
}

type createInput struct {
	api.WorkspaceInput
	Body ConnectSetupInput
}

type listInput struct {
	api.WorkspaceInput
	Adapter string `query:"adapter" doc:"Only setups of this adapter (ses, smtp…)"`
	Status  string `query:"status" enum:"open,finished,abandoned" doc:"Only setups in this state"`
}

// Huma only reads parameters from exported embedded structs, so every input
// spells out its path parameters.
type stepInput struct {
	api.WorkspaceInput
	SetupID string `path:"setupId" format:"uuid"`
	Body    ConnectStepInput
}

type connectBody struct {
	api.WorkspaceInput
	SetupID string `path:"setupId" format:"uuid"`
	Body    ConnectInput
}

type mailboxBody struct {
	api.WorkspaceInput
	SetupID string `path:"setupId" format:"uuid"`
	Body    ConnectMailboxInput
}

type webhookBody struct {
	api.WorkspaceInput
	SetupID string `path:"setupId" format:"uuid"`
	Body    ConnectWebhookInput
}

type regionBody struct {
	api.WorkspaceInput
	SetupID string `path:"setupId" format:"uuid"`
	Body    ConnectRegionInput
}

type productionBody struct {
	api.WorkspaceInput
	SetupID string `path:"setupId" format:"uuid"`
	Body    ConnectProductionAccessInput
}

type domainBody struct {
	api.WorkspaceInput
	SetupID string `path:"setupId" format:"uuid"`
	Body    ConnectDomainInput
}

type testBody struct {
	api.WorkspaceInput
	SetupID string `path:"setupId" format:"uuid"`
	Body    ConnectTestInput
}

type streamsBody struct {
	api.WorkspaceInput
	SetupID string `path:"setupId" format:"uuid"`
	Body    ConnectStreamsInput
}

type finishBody struct {
	api.WorkspaceInput
	SetupID string `path:"setupId" format:"uuid"`
	Body    ConnectFinishInput
}

type settingsBody struct {
	api.WorkspaceInput
	Body SettingsProvidersInput
}

// Register adds the connect-a-provider and Settings › Providers operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Connect a provider"}
	view, manage := api.Workspace(authz.ProvidersView), api.Workspace(authz.ProvidersManage)
	base := "/workspaces/{workspace}/connect/setups/{setupId}"

	api.Get(h, api.Op{ID: "connect-catalog", Path: "/workspaces/{workspace}/connect/catalog", Summary: "Every sending adapter this build can connect, with capabilities, pricing and terms",
		Tags: tags, Access: view},
		func(ctx context.Context, _ *wsInput) (*api.Body[ConnectCatalog], error) {
			out, err := s.Catalog(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-provider-setups", Path: "/workspaces/{workspace}/connect/setups", Summary: "The workspace's latest provider setups, newest first",
		Tags: tags, Access: view},
		func(ctx context.Context, in *listInput) (*api.Body[ConnectSetupList], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx), in.Adapter, in.Status)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "start-provider-setup", Path: "/workspaces/{workspace}/connect/setups", Summary: "Start connecting a provider (or resume the open setup)",
		Tags: tags, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *createInput) (*api.Body[ConnectSetup], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-provider-setup", Path: base, Summary: "A provider setup with everything its steps show", Tags: tags, Access: view},
		func(ctx context.Context, in *setupIDInput) (*api.Body[ConnectSetup], error) {
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx), in.SetupID)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "set-provider-setup-step", Path: base, Summary: "Record the step a setup is on", Tags: tags, Access: manage},
		func(ctx context.Context, in *stepInput) (*api.Body[ConnectSetup], error) {
			out, err := s.SetStep(ctx, api.WorkspaceFrom(ctx), in.SetupID, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "cancel-provider-setup", Path: base, Summary: "Cancel an unfinished setup",
		Description: "A provider the setup connected that no route uses yet is disconnected again.", Tags: tags, Access: manage},
		func(ctx context.Context, in *setupIDInput) (*api.Empty, error) {
			return api.NoContent, s.Abandon(ctx, api.WorkspaceFrom(ctx), in.SetupID)
		})
	api.Post(h, api.Op{ID: "connect-provider", Path: base + "/connection", Summary: "Check the credentials and connect the provider",
		Description: "Credentials are checked with the provider before they are stored, encrypted. They are never returned. A rejected key is recorded on the setup.",
		Tags:        tags, Access: manage},
		func(ctx context.Context, in *connectBody) (*api.Body[ConnectSetup], error) {
			out, err := s.Connect(ctx, api.WorkspaceFrom(ctx), in.SetupID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "connect-bounce-mailbox", Path: base + "/mailbox", Summary: "Set up and test the SMTP bounce mailbox", Tags: tags, Access: manage},
		func(ctx context.Context, in *mailboxBody) (*api.Body[ConnectSetup], error) {
			out, err := s.ConnectMailbox(ctx, api.WorkspaceFrom(ctx), in.SetupID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "retest-provider-webhook", Path: base + "/webhook", Summary: "Test the webhook endpoint again (optionally through a tunnel URL) and register it",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *webhookBody) (*api.Body[ConnectSetup], error) {
			out, err := s.RetestWebhook(ctx, api.WorkspaceFrom(ctx), in.SetupID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "show-provider-webhook-secret", Path: base + "/webhook-secret",
		Summary:     "Issue a new webhook secret to set up in the provider by hand",
		Description: "Only for providers whose webhook ReputeCast can't register itself (Azure Event Grid, the generic HTTP adapter). The secret is returned this once and replaces the previous one; stored secrets are never returned. Audited.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.ProvidersManage)},
		func(ctx context.Context, in *setupIDInput) (*api.Body[ConnectWebhookSecret], error) {
			out, err := s.WebhookSecret(ctx, api.WorkspaceFrom(ctx), in.SetupID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "use-provider-polling", Path: base + "/polling", Summary: "Poll the provider's events API instead of webhooks", Tags: tags, Access: manage},
		func(ctx context.Context, in *setupIDInput) (*api.Body[ConnectSetup], error) {
			out, err := s.UsePolling(ctx, api.WorkspaceFrom(ctx), in.SetupID)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-provider-region", Path: base + "/region", Summary: "Move an SES connection to another region", Tags: tags, Access: manage},
		func(ctx context.Context, in *regionBody) (*api.Body[ConnectSetup], error) {
			out, err := s.SetRegion(ctx, api.WorkspaceFrom(ctx), in.SetupID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "retry-provider-provisioning", Path: base + "/provision", Summary: "Retry the failed provisioning steps", Tags: tags, Access: manage},
		func(ctx context.Context, in *setupIDInput) (*api.Body[ConnectSetup], error) {
			out, err := s.RetryProvisioning(ctx, api.WorkspaceFrom(ctx), in.SetupID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "request-production-access", Path: base + "/production-access", Summary: "Ask the provider to lift its sandbox in a region",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *productionBody) (*api.Body[ConnectSetup], error) {
			out, err := s.RequestProductionAccess(ctx, api.WorkspaceFrom(ctx), in.SetupID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "choose-setup-domain", Path: base + "/domain", Summary: "Set up the provider for a sending domain, or skip the step",
		Description: "Adding a new domain needs domains:manage.", Tags: tags, Access: manage},
		func(ctx context.Context, in *domainBody) (*api.Body[ConnectSetup], error) {
			out, err := s.ChooseDomain(ctx, api.WorkspaceFrom(ctx), in.SetupID, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-setup-test", Path: base + "/test", Summary: "The test send form and the delivery timeline of the latest test", Tags: tags, Access: view},
		func(ctx context.Context, in *setupIDInput) (*api.Body[ConnectTestStatus], error) {
			out, err := s.TestStatus(ctx, api.WorkspaceFrom(ctx), in.SetupID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "send-setup-test", Path: base + "/test", Summary: "Send a test through the new provider", Tags: tags, Access: manage},
		func(ctx context.Context, in *testBody) (*api.Body[ConnectTestStatus], error) {
			out, err := s.SendTest(ctx, api.WorkspaceFrom(ctx), in.SetupID, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-setup-streams", Path: base + "/streams", Summary: "Assign the provider to streams (routing)", Tags: tags, Access: manage},
		func(ctx context.Context, in *streamsBody) (*api.Body[ConnectSetup], error) {
			out, err := s.SaveStreams(ctx, api.WorkspaceFrom(ctx), in.SetupID, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-setup-contacts", Path: base + "/contacts", Summary: "Importer sources, the provider's suppression list and the workspace's lists",
		Tags: tags, Access: view},
		func(ctx context.Context, in *setupIDInput) (*api.Body[ConnectContacts], error) {
			out, err := s.Contacts(ctx, api.WorkspaceFrom(ctx), in.SetupID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "finish-provider-setup", Path: base + "/finish", Summary: "Finish the setup", Tags: tags, Access: manage},
		func(ctx context.Context, in *finishBody) (*api.Body[ConnectSetup], error) {
			out, err := s.Finish(ctx, api.WorkspaceFrom(ctx), in.SetupID, in.Body)
			return api.OK(out), err
		})

	stags := []string{"Sending providers"}
	api.Get(h, api.Op{ID: "get-provider-settings", Path: "/workspaces/{workspace}/settings/providers", Summary: "Provider summaries and workspace-wide sending defaults",
		Tags: stags, Access: view},
		func(ctx context.Context, _ *wsInput) (*api.Body[SettingsProviders], error) {
			out, err := s.SettingsPage(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-provider-settings", Path: "/workspaces/{workspace}/settings/providers", Summary: "Save the auto-pause thresholds, throttles and sync interval",
		Tags: stags, Access: manage},
		func(ctx context.Context, in *settingsBody) (*api.Body[SettingsProviders], error) {
			out, err := s.SaveSettings(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "rotate-webhook-secrets", Path: "/workspaces/{workspace}/settings/providers/rotate-secrets", Summary: "Rotate every provider's webhook signing secret",
		Tags: stags, Access: api.WorkspaceHuman(authz.ProvidersManage)},
		func(ctx context.Context, _ *wsInput) (*api.Body[SettingsRotationResult], error) {
			out, err := s.RotateSecrets(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
}
