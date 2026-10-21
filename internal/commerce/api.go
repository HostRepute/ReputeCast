package commerce

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
)

type idPath struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid" doc:"The store's id"`
}

// idBody declares its fields itself: Huma skips an unexported embedded
// struct, which would leave {id} unparsed.
type idBody[T any] struct {
	api.WorkspaceInput
	ID   string `path:"id" format:"uuid" doc:"The store's id"`
	Body T
}

type body[T any] struct {
	api.WorkspaceInput
	Body T
}

type activityInput struct {
	api.WorkspaceInput
	ID     string   `path:"id" format:"uuid" doc:"The store's id"`
	Cursor string   `query:"cursor" maxLength:"200"`
	Kinds  []string `query:"kinds" maxItems:"20" doc:"Only these kinds of entries"`
	Limit  int      `query:"limit" minimum:"0" maximum:"100"`
}

type productsInput struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid" doc:"The store's id"`
	CommerceProductsInput
}

type registerInput struct {
	api.WorkspaceInput
	ID  string `path:"id" format:"uuid" doc:"The store's id"`
	All bool   `query:"all" doc:"Every webhook, not only the missing ones"`
}

type finishInput struct {
	WithoutPlugin bool `json:"withoutPlugin" doc:"Connect without the ReputeCast plugin: products and orders only"`
}

type automationPath struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid" doc:"The automation's id"`
}

type campaignPath struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid" doc:"The campaign's id"`
}

type contactPath struct {
	api.WorkspaceInput
	ContactID string `path:"contactId" format:"uuid" doc:"The contact's id"`
}

func (s *Service) parse(ctx context.Context, id string) (uuid.UUID, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, api.Err(http.StatusNotFound, api.CodeNotFound, mStoreNotFound.Text(i18n.LangFrom(ctx), nil))
	}
	return u, nil
}

// parseAny reads another domain's id ("Campaign", "Automation", "Contact").
func parseAny(id, what string) (uuid.UUID, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, api.NotFound(what)
	}
	return u, nil
}

// Register adds the stores' operations. A store's page reads with
// integrations:view; connecting, configuring and disconnecting a store
// need integrations:manage and a person (API keys and agents can read).
func Register(h huma.API, s *Service) {
	tags := []string{"Stores"}
	base := "/workspaces/{workspace}/stores"
	view := api.Workspace(authz.IntegrationsView)
	manage := api.WorkspaceHuman(authz.IntegrationsManage)

	api.Get(h, api.Op{ID: "list-stores", Path: base, Summary: "The connected stores, with their sync row", Tags: tags, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[CommerceStoreList], error) {
			out, err := s.Stores(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "connect-store", Path: base, Summary: "Start connecting a Shopify or WooCommerce store: its address",
		Description: "The address is checked against the platform. A store disconnected before reconnects with its data.", Tags: tags, Access: manage},
		func(ctx context.Context, in *body[CommerceBeginInput]) (*api.Body[CommerceConnectState], error) {
			out, err := s.Begin(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-store", Path: base + "/{id}", Summary: "A store's page: numbers, sync, consent, attribution, webhooks and activity",
		Description: "Credentials are write-only: only hints are returned.", Tags: tags, Access: view},
		func(ctx context.Context, in *idPath) (*api.Body[CommerceStorePage], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.Page(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-store-connect", Path: base + "/{id}/connect", Summary: "Where connecting a store stands: its step and what it shows", Tags: tags, Access: view},
		func(ctx context.Context, in *idPath) (*api.Body[CommerceConnectState], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.ConnectState(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-store-app-keys", Path: base + "/{id}/app-keys", Summary: "Shopify: the app's client ID and secret (step 3)",
		Description: "The secret is write-only: sealed, and never returned here or anywhere in the API.", Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[CommerceKeysInput]) (*api.Body[CommerceConnectState], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.SaveKeys(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "install-store-app", Path: base + "/{id}/install", Summary: "Shopify: the approval page's address (step 4)",
		Description: "Shopify asks the merchant to approve the app on its own page, then sends the browser back to the redirect URL.", Tags: tags, Access: manage},
		func(ctx context.Context, in *idPath) (*api.Body[CommerceInstall], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.Install(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "start-store-sync", Path: base + "/{id}/start", Summary: "Choose what syncs and start: the list, the first import, the webhooks",
		Description: "Only customers who accepted marketing join the list.", Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[CommerceSyncInput]) (*api.Body[CommerceConnectState], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.StartSync(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-store-rest-key", Path: base + "/{id}/rest-key", Summary: "WooCommerce: the REST API key, then its webhooks",
		Description: "The key is write-only: sealed, and never returned. Webhooks are registered with a secret of their own.", Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[CommerceWooKeyInput]) (*api.Body[CommerceConnectState], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.SaveWooKey(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-store-plugin", Path: base + "/{id}/plugin-check", Summary: "WooCommerce: look for the ReputeCast plugin on the store's site", Tags: tags, Access: manage},
		func(ctx context.Context, in *idPath) (*api.Body[CommerceConnectState], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.CheckPlugin(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "finish-store-connect", Path: base + "/{id}/finish", Summary: "WooCommerce: finish, with the plugin or without it", Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[finishInput]) (*api.Body[CommerceConnectState], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.FinishWoo(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body.WithoutPlugin)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "reconnect-store", Path: base + "/{id}/reconnect", Summary: "Reconnect a disconnected store: orders placed since are fetched", Tags: tags, Access: manage},
		func(ctx context.Context, in *idPath) (*api.Body[CommerceConnectState], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.Reconnect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-store-settings", Path: base + "/{id}/settings", Summary: "What syncs, the list, consent write-back, link tagging and the attribution window",
		Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[CommerceSettingsInput]) (*api.Body[CommerceStorePage], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.UpdateSettings(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "sync-store", Path: base + "/{id}/sync", Summary: "Sync now: fetch what changed in the store", Tags: tags, Access: manage},
		func(ctx context.Context, in *idPath) (*api.Body[CommerceQueued], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.SyncNow(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "register-store-webhooks", Path: base + "/{id}/webhooks", Summary: "Register the store's missing webhooks again, or all of them", Tags: tags, Access: manage},
		func(ctx context.Context, in *registerInput) (*api.Body[CommerceStorePage], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.RegisterAgain(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.All)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "disconnect-store", Path: base + "/{id}/disconnect", Summary: "Disconnect a store: its webhooks are removed; its data is kept unless deleted",
		Description: "Deleting the store data needs the store's name typed in confirm. Contacts the store added stay, with their consent. Automations with its triggers pause and say why.",
		Tags:        tags, Access: manage},
		func(ctx context.Context, in *idBody[CommerceDisconnectInput]) (*api.Body[CommerceStoreSummary], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.Disconnect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-store-activity", Path: base + "/{id}/activity", Summary: "The store's activity log: events and errors, kept 30 days",
		Description: "Not the audit log: orders and store webhooks are kept here.", Tags: tags, Access: view},
		func(ctx context.Context, in *activityInput) (*api.Body[CommerceActivityList], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.Activity(ctx, api.WorkspaceFrom(ctx), id, in.Cursor, in.Kinds, in.Limit)
			return api.OK(out), err
		})

	// Store data in emails: the composer's picker and blocks.
	compose := api.Workspace(authz.CampaignsView)
	api.Get(h, api.Op{ID: "get-store-catalog", Path: "/workspaces/{workspace}/commerce/catalog", Summary: "The product picker's stores, with their collections", Tags: tags, Access: compose},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[CommerceCatalog], error) {
			out, err := s.Catalog(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-store-products", Path: base + "/{id}/products", Summary: "A store's products, newest first, with their prices and availability now",
		Tags: tags, Access: compose},
		func(ctx context.Context, in *productsInput) (*api.Body[CommerceProductList], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.Products(ctx, api.WorkspaceFrom(ctx), id, in.CommerceProductsInput)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "render-store-block", Path: "/workspaces/{workspace}/commerce/blocks/render", Summary: "Draw a product card or grid with its products now",
		Description: "Removed products are drawn and flagged; sends read prices and availability again.", Tags: tags, Access: compose},
		func(ctx context.Context, in *body[CommerceBlockRenderInput]) (*api.Body[CommerceBlockRender], error) {
			out, err := s.RenderBlock(ctx, api.WorkspaceFrom(ctx), in.Body.Block)
			return api.OK(out), err
		})

	// Automations: the store triggers' choices, the Checkout items block,
	// an abandoned-checkout trigger's numbers.
	flows := api.Workspace(authz.AutomationsView)
	api.Get(h, api.Op{ID: "get-store-trigger-options", Path: "/workspaces/{workspace}/commerce/trigger-options", Summary: "The store triggers' stores, products and collections",
		Tags: tags, Access: flows},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[CommerceAutomationOptions], error) {
			out, err := s.AutomationOptions(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-store-checkout-block", Path: base + "/{id}/checkout-block", Summary: "The Checkout items block, with sample items from the store",
		Description: "Each message fills it with the items of the checkout or order that started its run.", Tags: tags, Access: flows},
		func(ctx context.Context, in *idPath) (*api.Body[CommerceCheckoutBlock], error) {
			id, err := s.parse(ctx, in.ID)
			if err != nil {
				return nil, err
			}
			out, err := s.CheckoutBlock(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-automation-store-stats", Path: "/workspaces/{workspace}/automations/{id}/store-stats",
		Summary: "An abandoned-checkout trigger's numbers since the automation went live, and its recovered orders", Tags: tags, Access: flows},
		func(ctx context.Context, in *automationPath) (*api.Body[CommerceTriggerStats], error) {
			id, err := parseAny(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			out, err := s.TriggerStats(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})

	// Contacts and revenue.
	api.Get(h, api.Op{ID: "get-contact-stores", Path: "/workspaces/{workspace}/contacts/{contactId}/stores", Summary: "The contact's store orders and their consent in each store",
		Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *contactPath) (*api.Body[CommerceContactStores], error) {
			id, err := parseAny(in.ContactID, "Contact")
			if err != nil {
				return nil, err
			}
			out, err := s.ContactStores(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-campaign-store-revenue", Path: "/workspaces/{workspace}/campaigns/{id}/store-revenue", Summary: "Store orders credited to a campaign, one row per store",
		Description: "Each order is credited to one email.", Tags: tags, Access: api.Workspace(authz.ReportsView)},
		func(ctx context.Context, in *campaignPath) (*api.Body[CommerceCampaignRevenue], error) {
			id, err := parseAny(in.ID, "Campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.CampaignRevenue(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-store-revenue", Path: "/workspaces/{workspace}/commerce/revenue", Summary: "Store revenue from email in the last 30 days (Earn › Overview)",
		Tags: tags, Access: api.Workspace(authz.EarnView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[CommerceRevenueSummary], error) {
			out, err := s.RevenueSummary(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})

	// The WordPress plugin, on a WooCommerce site.
	api.Post(h, api.Op{ID: OpWooCheckouts, Path: "/workspaces/{workspace}/woocommerce/checkouts", Summary: "Report checkouts left without an order, with their consent and recovery link",
		Description: "The ReputeCast WordPress plugin calls this with its site's key. Only checkouts of the WooCommerce store connected at the site's address count.",
		Tags:        []string{"WordPress plugin"}, Access: api.IntegrationOnly("wordpress", "")},
		func(ctx context.Context, in *body[CommerceWooCheckoutsInput]) (*api.Body[CommerceWooCheckoutsResult], error) {
			out, err := s.WooCheckouts(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
}
