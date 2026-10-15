package integrations

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// IntegrationKindPath names an integration in the path.
type IntegrationKindPath struct {
	api.WorkspaceInput
	Kind string `path:"kind" enum:"zapier,make,n8n,wordpress"`
}

// IntegrationPath names a connection (an app's, or a WordPress site's).
type IntegrationPath struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid"`
}

func (in *IntegrationPath) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.ID)
	if err != nil {
		return uuid.Nil, api.NotFound("Integration")
	}
	return id, nil
}

type callsInput struct {
	IntegrationPath
	Errors bool   `query:"errors" doc:"Only errors"`
	Window string `query:"window" enum:"24h,7d" default:"7d"`
	Offset int    `query:"offset" minimum:"0" maximum:"10000" default:"0"`
	Limit  int    `query:"limit" minimum:"1" maximum:"100" default:"20"`
}

type callInput struct {
	IntegrationPath
	CallID string `path:"callId" format:"uuid"`
}

type deadLettersInput struct {
	IntegrationPath
	WebhookID string `path:"webhookId" format:"uuid"`
}

type ownedInput struct {
	api.WorkspaceInput
	UserID string `path:"userId" format:"uuid"`
}

type hookInput struct {
	api.WorkspaceInput
	Workflow string `header:"X-ReputeCast-Workflow" doc:"The calling workflow: \"<name>; id=<id>\", the name percent-encoded when it isn't ASCII"`
	Body     IntegrationHookInput
}

type hookPathInput struct {
	api.WorkspaceInput
	HookID string `path:"hookId" format:"uuid"`
}

type sampleInput struct {
	api.WorkspaceInput
	Event string `path:"event" example:"contact.subscribed"`
}

type listAddInput struct {
	api.WorkspaceInput
	ID   string `path:"id" format:"uuid" doc:"The list"`
	Body ContactListAddInput
}

// IntegrationAddressCheckInput checks a workflow tool's address.
type IntegrationAddressCheckInput struct {
	Address string `json:"address" maxLength:"300" example:"https://n8n.acme.io"`
}

// IntegrationListItem is a list in an app's dropdown.
type IntegrationListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DoubleOptIn bool   `json:"doubleOptIn"`
}

// IntegrationListItems are the lists the apps offer.
type IntegrationListItems struct {
	Items []IntegrationListItem `json:"items"`
}

// IntegrationFieldItem is a custom field in an app's field mapping.
type IntegrationFieldItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

// IntegrationFieldItems are the custom fields the apps map.
type IntegrationFieldItems struct {
	Items []IntegrationFieldItem `json:"items"`
}

// Register adds the integrations hub's operations: the people-facing hub
// (integrations:view to read, integrations:manage to change, people only)
// and the API the apps and the WordPress plugin call.
func Register(h huma.API, s *Service) {
	hub := []string{"Integrations"}
	view := api.Workspace(authz.IntegrationsView)
	manage := api.WorkspaceHuman(authz.IntegrationsManage)
	base := "/workspaces/{workspace}/integrations"

	api.Get(h, api.Op{ID: "get-integration-catalog", Path: base + "/catalog", Summary: "The integrations catalog: groups, statuses and what needs attention", Tags: hub, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[IntegrationHub], error) {
			out, err := s.Hub(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-integration-app", Path: base + "/apps/{kind}", Summary: "An automation platform's page: numbers, workflows, webhooks, what it can do and its key",
		Tags: hub, Access: view},
		func(ctx context.Context, in *IntegrationKindPath) (*api.Body[IntegrationApp], error) {
			out, err := s.App(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Kind)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-integration-wordpress", Path: base + "/wordpress", Summary: "WordPress sites: health, forms, post drafts and keys", Tags: hub, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[IntegrationWordPressPage], error) {
			out, err := s.WordPressPage(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-integration-connect", Path: base + "/apps/{kind}/connect", Summary: "What the connect dialog offers you: triggers and actions, locks, lists and reachability",
		Tags: hub, Access: manage},
		func(ctx context.Context, in *IntegrationKindPath) (*api.Body[IntegrationConnectInfo], error) {
			out, err := s.ConnectInfo(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Kind)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "connect-integration", Path: base + "/apps/{kind}/connect", Summary: "Connect an integration; its key is shown once",
		Description: "Creates the integration's own key with the permissions of the triggers and actions turned on. The key works only for those.",
		Tags:        hub, Access: manage, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			IntegrationKindPath
			Body IntegrationConnectInput
		}) (*api.Body[IntegrationSecret], error) {
			out, err := s.Connect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Kind, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-integration-status", Path: base + "/connections/{id}/status", Summary: "Whether a new key has been called yet (the connect dialog's last step)",
		Tags: hub, Access: view},
		func(ctx context.Context, in *IntegrationPath) (*api.Body[IntegrationStatus], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Status(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-integration-calls", Path: base + "/connections/{id}/calls", Summary: "The 7-day call log: workflow, action, target, result and duration",
		Description: "Addresses show to members with subscribers:view; others see \"a contact\".", Tags: hub, Access: view},
		func(ctx context.Context, in *callsInput) (*api.Body[IntegrationCallList], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Calls(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, IntegrationCallFilter{ErrorsOnly: in.Errors, Window: in.Window, Offset: in.Offset, Limit: in.Limit})
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-integration-call", Path: base + "/connections/{id}/calls/{callId}", Summary: "One call with its request and response, secrets masked",
		Tags: hub, Access: view},
		func(ctx context.Context, in *callInput) (*api.Body[IntegrationCallDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			callID, err := uuid.Parse(in.CallID)
			if err != nil {
				return nil, api.NotFound("Call")
			}
			out, err := s.Call(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, callID)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-integration-dead-letters", Path: base + "/connections/{id}/webhooks/{webhookId}/dead-letters", Summary: "A managed webhook's dead-letter queue",
		Tags: hub, Access: view},
		func(ctx context.Context, in *deadLettersInput) (*api.Body[IntegrationDeadLetterList], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			hookID, err := uuid.Parse(in.WebhookID)
			if err != nil {
				return nil, api.NotFound("Webhook")
			}
			out, err := s.DeadLetters(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, hookID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "change-integration", Path: base + "/connections/{id}/change", Summary: "Change what an integration can do; its key changes in place",
		Tags: hub, Access: manage},
		func(ctx context.Context, in *struct {
			IntegrationPath
			Body IntegrationChangeInput
		}) (*api.Body[IntegrationChanged], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Change(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "rotate-integration-key", Path: base + "/connections/{id}/rotate", Summary: "Rotate an integration's key; the old one works for 24 hours",
		Tags: hub, Access: manage},
		func(ctx context.Context, in *IntegrationPath) (*api.Body[IntegrationSecret], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Rotate(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "stop-integration-old-key", Path: base + "/connections/{id}/stop-old-key", Summary: "Stop a rotated key before its 24 hours run out",
		Tags: hub, Access: manage, Status: http.StatusNoContent},
		func(ctx context.Context, in *IntegrationPath) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.StopOldKey(ctx, api.WorkspaceFrom(ctx), id)
		})
	api.Post(h, api.Op{ID: "reconnect-integration", Path: base + "/connections/{id}/reconnect", Summary: "Give an integration whose key was revoked or expired a new key, shown once",
		Tags: hub, Access: manage},
		func(ctx context.Context, in *struct {
			IntegrationPath
			Body IntegrationConnectInput
		}) (*api.Body[IntegrationSecret], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Reconnect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-integration-take-over", Path: base + "/connections/{id}/take-over", Summary: "What taking over the key would change: the permissions you hold and don't",
		Tags: hub, Access: manage},
		func(ctx context.Context, in *IntegrationPath) (*api.Body[IntegrationTakeOverPreview], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.TakeOverPreview(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "take-over-integration", Path: base + "/connections/{id}/take-over", Summary: "Become the owner of an integration's key; it keeps its value",
		Tags: hub, Access: manage},
		func(ctx context.Context, in *IntegrationPath) (*api.Body[IntegrationTakeOverPreview], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.TakeOver(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-integration-disconnect", Path: base + "/connections/{id}/disconnect", Summary: "What disconnecting stops, removes and keeps",
		Tags: hub, Access: manage},
		func(ctx context.Context, in *IntegrationPath) (*api.Body[IntegrationDisconnectPreview], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.DisconnectPreview(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "disconnect-integration", Path: base + "/connections/{id}/disconnect", Summary: "Disconnect: revoke the key and remove the webhooks it created",
		Description: "Contacts, events and messages created through the integration stay.", Tags: hub, Access: manage, Status: http.StatusNoContent},
		func(ctx context.Context, in *struct {
			IntegrationPath
			Body IntegrationDisconnectInput
		}) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, s.Disconnect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
		})
	api.Post(h, api.Op{ID: "check-wordpress-site", Path: base + "/connections/{id}/check", Summary: "Check now: call a site's status endpoint once, through the SSRF guard",
		Tags: hub, Access: manage},
		func(ctx context.Context, in *IntegrationPath) (*api.Body[WordPressCheck], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.CheckSite(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-wordpress-rule", Path: base + "/connections/{id}/rule", Summary: "Set a site's post-draft rule: categories, audience, template, sender, who's notified",
		Tags: hub, Access: manage},
		func(ctx context.Context, in *struct {
			IntegrationPath
			Body WordPressRuleInput
		}) (*api.Body[WordPressRule], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SetRule(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-integration-reachability", Path: base + "/reachability/check", Summary: "Run the public URL check again: can Zapier and Make reach this instance?",
		Tags: hub, Access: manage},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[IntegrationReachability], error) {
			out, err := s.CheckReachability(ctx)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-integration-address", Path: base + "/address-check", Summary: "Whether triggers can reach a workflow tool's address (private addresses need an instance admin)",
		Tags: hub, Access: manage},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body IntegrationAddressCheckInput
		}) (*api.Body[IntegrationAddressCheck], error) {
			out, err := s.CheckAddress(ctx, in.Body.Address)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-integration-address", Path: base + "/connections/{id}/address", Summary: "Keep n8n's address, to open it and to check that triggers can reach it",
		Tags: hub, Access: manage},
		func(ctx context.Context, in *struct {
			IntegrationPath
			Body IntegrationAddressInput
		}) (*api.Body[IntegrationAddressCheck], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.SetAddress(ctx, api.WorkspaceFrom(ctx), id, in.Body.Address)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-member-integrations", Path: base + "/owned-by/{userId}", Summary: "Integrations whose key a member owns (deactivate and remove dialogs)",
		Tags: hub, Access: api.Workspace(authz.TeamManage)},
		func(ctx context.Context, in *ownedInput) (*api.Body[IntegrationOwnedList], error) {
			uid, err := uuid.Parse(in.UserID)
			if err != nil {
				return nil, api.NotFound("Member")
			}
			out, err := s.OwnedBy(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), uid)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-integration-keys", Path: base + "/keys", Summary: "Keys managed by integrations (Keys & MCP's second table)",
		Tags: hub, Access: api.Workspace(authz.APIKeysManage)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[IntegrationManagedKeyList], error) {
			out, err := s.ManagedKeys(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})

	// The API the apps and the plugin call with their key.
	app := []string{"Integration API"}
	api.Get(h, api.Op{ID: OpMe, Path: "/integration", Summary: "Test connection: the key's workspace and what it can do",
		Description: "Needs only the key: the apps call it first to learn the workspace slug the other paths take.", Tags: app, Access: api.CallingKey},
		func(ctx context.Context, _ *struct{}) (*api.Body[IntegrationMe], error) {
			out, err := s.Me(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: OpSubscribe, Path: "/workspaces/{workspace}/integration/hooks", Summary: "Subscribe a workflow to a trigger (REST hook); the signing secret is shown once",
		Description: "Creates a managed webhook. It can't be edited by hand, answers to the integration only, and is removed when the endpoint answers 410 Gone.",
		Tags:        app, Access: api.IntegrationOnly("*", authz.WebhooksManage), Status: http.StatusCreated},
		func(ctx context.Context, in *hookInput) (*api.Body[IntegrationHook], error) {
			out, err := s.Subscribe(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body, in.Workflow)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: OpUnsubscribe, Path: "/workspaces/{workspace}/integration/hooks/{hookId}", Summary: "Unsubscribe: remove a workflow's managed webhook",
		Tags: app, Access: api.IntegrationOnly("*", authz.WebhooksManage)},
		func(ctx context.Context, in *hookPathInput) (*api.Empty, error) {
			id, err := uuid.Parse(in.HookID)
			if err != nil {
				return nil, api.NotFound("Webhook")
			}
			return api.NoContent, s.Unsubscribe(ctx, api.WorkspaceFrom(ctx), id)
		})
	api.Get(h, api.Op{ID: OpSample, Path: "/workspaces/{workspace}/integration/samples/{event}", Summary: "A sample payload of a trigger, for field mapping",
		Tags: app, Access: api.IntegrationOnly("*", authz.WebhooksManage)},
		func(ctx context.Context, in *sampleInput) (*api.Body[IntegrationTriggerSample], error) {
			out, err := s.Sample(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Event)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: OpLists, Path: "/workspaces/{workspace}/integration/lists", Summary: "Lists, for the apps' dropdowns",
		Tags: app, Access: api.IntegrationOnly("*", authz.SubscribersView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[IntegrationListItems], error) {
			out, err := s.Lists(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: OpFields, Path: "/workspaces/{workspace}/integration/fields", Summary: "Custom fields, for the apps' field mapping",
		Tags: app, Access: api.IntegrationOnly("*", authz.SubscribersView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[IntegrationFieldItems], error) {
			out, err := s.Fields(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})

	contacts := []string{"Contacts"}
	api.Post(h, api.Op{ID: OpUpsert, Path: "/workspaces/{workspace}/contacts/upsert", Summary: "Create or update a contact, matched by email",
		Description: "Adds to lists with single or double opt-in. Never adds someone back to a list they left (409 unsubscribed) or a suppressed address (409 suppressed). " +
			"Updating an existing contact needs subscribers:edit.",
		Tags: contacts, Access: api.Workspace(authz.SubscribersCreate)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body ContactUpsertInput
		}) (*api.Body[ContactUpsertResult], error) {
			out, err := s.UpsertContact(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: OpAddToList, Path: "/workspaces/{workspace}/lists/{id}/contacts", Summary: "Add a contact to a list, single or double opt-in",
		Description: "Never adds someone back to a list they left, or a suppressed address: both answer 409 with the date and the reason.",
		Tags:        contacts, Access: api.Workspace(authz.SubscribersEdit)},
		func(ctx context.Context, in *listAddInput) (*api.Body[ContactUpsertResult], error) {
			out, err := s.AddToList(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: OpTag, Path: "/workspaces/{workspace}/contacts/tags", Summary: "Add tags to a contact, matched by email",
		Tags: contacts, Access: api.Workspace(authz.SubscribersEdit)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body ContactTagInput
		}) (*api.Body[ContactUpsertResult], error) {
			out, err := s.TagContact(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})

	wp := []string{"WordPress plugin"}
	api.Post(h, api.Op{ID: OpWPReport, Path: "/workspaces/{workspace}/wordpress/report", Summary: "Report in: plugin, WordPress and PHP versions, the feed and where forms are used",
		Description: "The plugin reports when it connects, every 12 hours and with every post. ReputeCast never polls a site.", Tags: wp, Access: api.IntegrationOnly(KindWordPress, "")},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body WordPressReportInput
		}) (*api.Body[WordPressReportResult], error) {
			out, err := s.Report(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: OpWPForms, Path: "/workspaces/{workspace}/wordpress/forms", Summary: "The signup forms, for the block and the shortcode",
		Description: "subscribers:view lets a site's key list forms only: it can't read contacts.", Tags: wp, Access: api.IntegrationOnly(KindWordPress, authz.SubscribersView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[WordPressFormList], error) {
			out, err := s.Forms(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: OpWPDraft, Path: "/workspaces/{workspace}/wordpress/drafts", Summary: "Create a campaign draft from a published post, as the site's rule says",
		Description: "campaigns:edit lets a site's key create drafts from posts only: it can't open, change or send any other campaign. Nothing is sent.",
		Tags:        wp, Access: api.IntegrationOnly(KindWordPress, authz.CampaignsEdit)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body WordPressDraftInput
		}) (*api.Body[WordPressDraftResult], error) {
			out, err := s.CreateDraft(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
}
