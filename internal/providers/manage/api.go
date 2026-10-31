package manage

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

type providerInput struct {
	api.WorkspaceInput
	ProviderID string `path:"providerId" format:"uuid"`
}

// Huma only reads parameters from exported embedded structs, so every input
// spells out its path parameters.
type credentialsInput struct {
	api.WorkspaceInput
	ProviderID string `path:"providerId" format:"uuid"`
	Body       ProviderCredentialsInput
}

type settingsInput struct {
	api.WorkspaceInput
	ProviderID string `path:"providerId" format:"uuid"`
	Body       ProviderSettingsInput
}

type ipsInput struct {
	api.WorkspaceInput
	ProviderID string `path:"providerId" format:"uuid"`
	Body       ProviderIPsInput
}

type trackingInput struct {
	api.WorkspaceInput
	ProviderID string `path:"providerId" format:"uuid"`
	Body       ProviderTrackingActionInput
}

type heldInput struct {
	api.WorkspaceInput
	ProviderID string `path:"providerId" format:"uuid"`
	CampaignID string `path:"campaignId" format:"uuid"`
	Body       ProviderHeldDecisionInput
}

type adapterInput struct {
	api.WorkspaceInput
	AdapterID string `path:"adapterId" format:"uuid"`
}

type installInput struct {
	api.WorkspaceInput
	Body AdapterInstallInput
}

type httpAdapterInput struct {
	api.WorkspaceInput
	Body AdapterHTTPInput
}

type httpPreviewInput struct {
	api.WorkspaceInput
	Body AdapterHTTPMapping
}

// ProviderTrackingChecks is the tracking-check list.
type ProviderTrackingChecks struct {
	Items []ProviderTrackingCheck `json:"items"`
}

// Register adds the sending-providers operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Sending providers"}
	view, manage := api.Workspace(authz.ProvidersView), api.Workspace(authz.ProvidersManage)
	api.Get(h, api.Op{ID: "list-providers", Path: "/workspaces/{workspace}/providers", Summary: "Connected providers with health, quota, cost, capabilities and routing",
		Description: "Credentials are never returned. Outage, webhooks-silent and rejected-credential states are included when they apply.", Tags: tags, Access: view},
		func(ctx context.Context, _ *wsInput) (*api.Body[ProvidersOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-provider", Path: "/workspaces/{workspace}/providers/{providerId}/check", Summary: "Check a provider's health now", Tags: tags, Access: manage},
		func(ctx context.Context, in *providerInput) (*api.Body[ProviderCard], error) {
			out, err := s.CheckNow(ctx, api.WorkspaceFrom(ctx), in.ProviderID)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "replace-provider-credentials", Path: "/workspaces/{workspace}/providers/{providerId}/credentials", Summary: "Replace write-only credentials",
		Description: "The provider checks the new credentials before they are stored, encrypted. They are never returned.", Tags: tags, Access: manage},
		func(ctx context.Context, in *credentialsInput) (*api.Body[ProviderCard], error) {
			out, err := s.ReplaceCredentials(ctx, api.WorkspaceFrom(ctx), in.ProviderID, in.Body)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-provider", Path: "/workspaces/{workspace}/providers/{providerId}", Summary: "Change a provider's label, pricing or limits", Tags: tags, Access: manage},
		func(ctx context.Context, in *settingsInput) (*api.Body[ProviderCard], error) {
			out, err := s.UpdateSettings(ctx, api.WorkspaceFrom(ctx), in.ProviderID, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-provider-ips", Path: "/workspaces/{workspace}/providers/{providerId}/ips", Summary: "Set the sending IPs of a provider whose API doesn't list them",
		Description: "For self-hosted servers, SMTP and providers without an IP API: their dedicated IPs (up to 64 public IPv4 or IPv6 addresses) or the provider's shared pool. " +
			"A provider whose API reports its IPs is refused with 409: ReputeCast reads them when it's connected and every day.", Tags: tags, Access: manage},
		func(ctx context.Context, in *ipsInput) (*api.Body[ProviderCard], error) {
			out, err := s.SetIPs(ctx, api.WorkspaceFrom(ctx), in.ProviderID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-provider-ips", Path: "/workspaces/{workspace}/providers/{providerId}/ips/check", Summary: "Ask a provider for its sending IPs now",
		Description: "For a provider whose API reports its IPs; the card says what it answered, or why it couldn't. Others are refused with 409: their IPs are set by hand.", Tags: tags, Access: manage},
		func(ctx context.Context, in *providerInput) (*api.Body[ProviderCard], error) {
			out, err := s.CheckIPs(ctx, api.WorkspaceFrom(ctx), in.ProviderID)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "disconnect-provider", Path: "/workspaces/{workspace}/providers/{providerId}", Summary: "Disconnect a provider no route uses, and remove the webhook ReputeCast registered there", Tags: tags, Access: manage, Status: http.StatusOK},
		func(ctx context.Context, in *providerInput) (*api.Body[ProviderDisconnected], error) {
			out, err := s.Disconnect(ctx, api.WorkspaceFrom(ctx), in.ProviderID)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-tracking-checks", Path: "/workspaces/{workspace}/tracking-checks", Summary: "Provider-side open and click tracking per provider", Tags: tags, Access: view},
		func(ctx context.Context, _ *wsInput) (*api.Body[ProviderTrackingChecks], error) {
			out, err := s.ProviderTrackingChecks(ctx, api.WorkspaceFrom(ctx))
			return api.OK(ProviderTrackingChecks{Items: out}), err
		})
	api.Post(h, api.Op{ID: "provider-tracking-action", Path: "/workspaces/{workspace}/providers/{providerId}/tracking", Summary: "Turn off provider-side tracking or keep it", Tags: tags, Access: manage},
		func(ctx context.Context, in *trackingInput) (*api.Body[ProviderTrackingCheck], error) {
			out, err := s.TrackingAction(ctx, api.WorkspaceFrom(ctx), in.ProviderID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "test-provider-webhook", Path: "/workspaces/{workspace}/providers/{providerId}/webhook/test", Summary: "Self-test the webhook endpoint and ask the provider for a test event",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *providerInput) (*api.Body[ProviderWebhookTestResult], error) {
			out, err := s.TestWebhook(ctx, api.WorkspaceFrom(ctx), in.ProviderID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "register-provider-webhook", Path: "/workspaces/{workspace}/providers/{providerId}/webhook/register", Summary: "Register the webhook at the provider again",
		Tags: tags, Access: manage, Status: http.StatusNoContent},
		func(ctx context.Context, in *providerInput) (*api.Empty, error) {
			return api.NoContent, s.RegisterWebhook(ctx, api.WorkspaceFrom(ctx), in.ProviderID)
		})
	api.Post(h, api.Op{ID: "backfill-provider-events", Path: "/workspaces/{workspace}/providers/{providerId}/backfill", Summary: "Pull message status from the provider's API",
		Tags: tags, Access: manage, Status: http.StatusAccepted},
		func(ctx context.Context, in *providerInput) (*api.Body[ProviderBackfillResult], error) {
			out, err := s.Backfill(ctx, api.WorkspaceFrom(ctx), in.ProviderID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-broadcast-stream", Path: "/workspaces/{workspace}/providers/{providerId}/broadcast-stream", Summary: "Create the stream a provider's terms require for bulk mail",
		Tags: tags, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *providerInput) (*api.Body[ProviderBroadcastStreamResult], error) {
			out, err := s.CreateBroadcastStream(ctx, api.WorkspaceFrom(ctx), in.ProviderID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "decide-held-broadcast", Path: "/workspaces/{workspace}/providers/{providerId}/held/{campaignId}", Summary: "Wait, move or cancel a broadcast share held on a provider that is down",
		Description: "Cancelling needs campaigns:pause; waiting for the provider and moving the held recipients resume the send and need campaigns:send.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsPause)},
		func(ctx context.Context, in *heldInput) (*api.Body[ProviderHeldDecisionResult], error) {
			out, err := s.DecideHeld(ctx, api.WorkspaceFrom(ctx), in.ProviderID, in.CampaignID, in.Body)
			return api.OK(out), err
		})

	atags := []string{"Adapters"}
	admin := api.WorkspaceHuman(authz.ProvidersManage)
	api.Get(h, api.Op{ID: "list-adapters", Path: "/workspaces/{workspace}/adapters", Summary: "Built-in and community adapters", Tags: atags, Access: view},
		func(ctx context.Context, _ *wsInput) (*api.Body[AdaptersOverview], error) {
			out, err := s.Adapters(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "install-adapter", Path: "/workspaces/{workspace}/adapters", Summary: "Install a community adapter from its manifest (instance admins)",
		Tags: atags, Access: admin, Status: http.StatusCreated},
		func(ctx context.Context, in *installInput) (*api.Body[AdapterCommunity], error) {
			out, err := s.InstallAdapter(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "update-adapter", Path: "/workspaces/{workspace}/adapters/{adapterId}/update", Summary: "Update a community adapter to its latest version (instance admins)",
		Tags: atags, Access: admin},
		func(ctx context.Context, in *adapterInput) (*api.Body[AdapterCommunity], error) {
			out, err := s.UpdateAdapter(ctx, api.WorkspaceFrom(ctx), in.AdapterID)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-adapter", Path: "/workspaces/{workspace}/adapters/{adapterId}", Summary: "Remove a community adapter (instance admins)", Tags: atags, Access: admin},
		func(ctx context.Context, in *adapterInput) (*api.Empty, error) {
			return api.NoContent, s.RemoveAdapter(ctx, api.WorkspaceFrom(ctx), in.AdapterID)
		})
	api.Post(h, api.Op{ID: "save-http-adapter", Path: "/workspaces/{workspace}/adapters/http", Summary: "Save a generic HTTP adapter mapping (instance admins)",
		Tags: atags, Access: admin, Status: http.StatusCreated},
		func(ctx context.Context, in *httpAdapterInput) (*api.Body[AdapterCommunity], error) {
			out, err := s.SaveHTTPAdapter(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "preview-http-adapter", Path: "/workspaces/{workspace}/adapters/http/preview", Summary: "Render the request a generic HTTP mapping sends for a sample message (instance admins)",
		Description: "Nothing is sent. Secrets are shown as ••••.", Tags: atags, Access: admin},
		func(ctx context.Context, in *httpPreviewInput) (*api.Body[AdapterHTTPPreview], error) {
			out, err := s.PreviewHTTPAdapter(ctx, in.Body)
			return api.OK(out), err
		})
}
