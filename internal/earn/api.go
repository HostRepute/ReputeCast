package earn

import (
	"context"
	"fmt"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type idPath struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid" doc:"Id"`
}

type contactPath struct {
	api.WorkspaceInput
	ContactID string `path:"contactId" format:"uuid" doc:"Contact id"`
}

type body[T any] struct {
	api.WorkspaceInput
	Body T
}

// idBody declares its fields itself: Huma skips an unexported embedded
// struct, which would leave {id} unparsed.
type idBody[T any] struct {
	api.WorkspaceInput
	ID   string `path:"id" format:"uuid" doc:"Id"`
	Body T
}

type eventLogInput struct {
	api.WorkspaceInput
	EarnEventFilter
}

type membersInput struct {
	api.WorkspaceInput
	EarnMemberFilter
}

type importUpload struct {
	api.WorkspaceInput
	Source   string `query:"source" enum:"substack,ghost,beehiiv,kit,csv" doc:"Where the export comes from"`
	FileName string `query:"fileName" maxLength:"200" doc:"The file's name"`
	RawBody  []byte `contentType:"application/octet-stream" doc:"The export: a CSV, or the zip Substack exports"`
}

type csvFile struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	CacheControl       string `header:"Cache-Control"`
	Body               []byte
}

// Register adds Earn's operations: the payment connection and its event
// log, tiers and the upgrade page, paid members, imports and the Earn
// settings. Changing anything that touches money or keys needs a person
// (API keys and agents can read).
func Register(h huma.API, s *Service) {
	tags := []string{"Earn"}
	base := "/workspaces/{workspace}/earn"
	view := api.Workspace(authz.EarnView)
	pay := api.WorkspaceHuman(authz.PaymentsManage)
	edit := api.WorkspaceHuman(authz.EarnEdit)
	manage := api.WorkspaceHuman(authz.EarnManage)

	// The connection chip and Settings › Payments.
	api.Get(h, api.Op{ID: "get-earn-status", Path: base + "/status", Summary: "The payment connection's chip: provider, account, mode and health", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[EarnStatus], error) {
			out, err := s.Status(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-earn-payments", Path: base + "/payments", Summary: "Settings › Payments: the connections, their webhook, checkout and tax",
		Description: "Keys and webhook signing secrets are write-only: only a hint of the key is returned.", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[EarnPayments], error) {
			out, err := s.Payments(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-payment-events", Path: base + "/payments/events", Summary: "The payments event log: each provider event and what ReputeCast did with it",
		Description: "Raw events are kept 90 days. Payment events never go to the audit log.", Tags: tags, Access: view},
		func(ctx context.Context, in *eventLogInput) (*api.Body[EarnEventLog], error) {
			out, err := s.EventLog(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.EarnEventFilter)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "connect-payment-provider", Path: base + "/connections", Summary: "Check a restricted key and start connecting",
		Description: "Restricted keys only: a secret key is refused before anything is sent to the provider. The key is sealed and never returned.",
		Tags:        tags, Access: pay},
		func(ctx context.Context, in *body[EarnConnectInput]) (*api.Body[EarnConnection], error) {
			out, err := s.Connect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-payment-connection", Path: base + "/connections/{id}", Summary: "A payment connection: the connect wizard's state", Tags: tags, Access: view},
		func(ctx context.Context, in *idPath) (*api.Body[EarnConnection], error) {
			out, err := s.Connection(ctx, api.WorkspaceFrom(ctx), in.ID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-payment-connection", Path: base + "/connections/{id}/check", Summary: "Check the key's account and permissions again", Tags: tags, Access: pay},
		func(ctx context.Context, in *idPath) (*api.Body[EarnConnection], error) {
			out, err := s.Recheck(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "register-payment-webhook", Path: base + "/connections/{id}/webhook", Summary: "Register the webhook endpoint with its events",
		Description: "Checks that the public URL is reachable, then creates the endpoint; its signing secret is sealed and never returned.", Tags: tags, Access: pay},
		func(ctx context.Context, in *idPath) (*api.Body[EarnConnection], error) {
			out, err := s.RegisterWebhook(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-payment-webhook-secret", Path: base + "/connections/{id}/webhook-secret", Summary: "Save the signing secret of a webhook added by hand",
		Description: "Write-only: sealed and never returned.", Tags: tags, Access: pay},
		func(ctx context.Context, in *idBody[EarnWebhookSecretInput]) (*api.Body[EarnConnection], error) {
			out, err := s.WebhookSecret(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-payment-connection-step", Path: base + "/connections/{id}/step", Summary: "Move the connect wizard to a step", Tags: tags, Access: pay},
		func(ctx context.Context, in *idBody[EarnStepInput]) (*api.Body[EarnConnection], error) {
			out, err := s.SetStep(ctx, api.WorkspaceFrom(ctx), in.ID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "finish-payment-connection", Path: base + "/connections/{id}/finish", Summary: "Save checkout and tax, set up the customer portal and finish connecting",
		Description: "Creates the eight member emails as drafts in Transactional › Templates the first time.", Tags: tags, Access: pay},
		func(ctx context.Context, in *idBody[EarnFinishInput]) (*api.Body[EarnFinished], error) {
			out, err := s.Finish(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "replace-payment-key", Path: base + "/connections/{id}/key", Summary: "Replace the restricted key with one of the same account and mode",
		Description: "Write-only: the new key is sealed and never returned.", Tags: tags, Access: pay},
		func(ctx context.Context, in *idBody[EarnReplaceKeyInput]) (*api.Body[EarnConnection], error) {
			out, err := s.ReplaceKey(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "register-payment-webhook-again", Path: base + "/connections/{id}/webhook/again", Summary: "Register the webhook again, then read any event it missed", Tags: tags, Access: pay},
		func(ctx context.Context, in *idPath) (*api.Body[EarnConnection], error) {
			out, err := s.RegisterAgain(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "disconnect-payment-provider", Path: base + "/connections/{id}/disconnect", Summary: "Disconnect: delete the key and the webhook secret",
		Description: "Subscriptions keep billing in the provider; paid members keep access until the end of what they've paid for.", Tags: tags, Access: pay, Status: 204},
		func(ctx context.Context, in *idBody[EarnDisconnectInput]) (*api.Empty, error) {
			return api.NoContent, s.Disconnect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
		})
	api.Put(h, api.Op{ID: "save-earn-checkout", Path: base + "/checkout", Summary: "Checkout and tax: currency, tax at checkout and the customer portal", Tags: tags, Access: pay},
		func(ctx context.Context, in *body[EarnCheckoutInput]) (*api.Body[EarnCheckoutSettings], error) {
			out, err := s.SaveCheckout(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "clear-earn-test-data", Path: base + "/test-data/clear", Summary: "Delete test paid members, test checkouts and test events", Tags: tags, Access: pay},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[EarnTestDataCleared], error) {
			out, err := s.ClearTestData(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-earn-checklist", Path: base + "/checklist", Summary: "The launch checklist: connect, a tier, member emails, the upgrade page, a test checkout, the live key", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[EarnChecklist], error) {
			out, err := s.Checklist(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})

	// Plans: tiers, their prices and the upgrade page.
	api.Get(h, api.Op{ID: "get-earn-plans", Path: base + "/plans", Summary: "Plans & offers: the tiers with their prices, members and MRR, and the upgrade page", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[EarnPlans], error) {
			out, err := s.Plans(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-earn-tier", Path: base + "/tiers/{id}", Summary: "A tier with its prices and their sync with the provider", Tags: tags, Access: view},
		func(ctx context.Context, in *idPath) (*api.Body[EarnTier], error) {
			out, err := s.TierDetail(ctx, api.WorkspaceFrom(ctx), in.ID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-earn-tier", Path: base + "/tiers", Summary: "Create a tier and its prices, in the provider too", Tags: tags, Access: edit},
		func(ctx context.Context, in *body[EarnTierInput]) (*api.Body[EarnTierSaved], error) {
			out, err := s.CreateTier(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-earn-tier", Path: base + "/tiers/{id}", Summary: "Change a tier",
		Description: "A new price becomes the tier's price for new members; members on the old one keep it as a legacy price.", Tags: tags, Access: edit},
		func(ctx context.Context, in *idBody[EarnTierInput]) (*api.Body[EarnTierSaved], error) {
			out, err := s.UpdateTier(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-earn-tier", Path: base + "/tiers/{id}", Summary: "Delete a tier nobody has", Tags: tags, Access: edit},
		func(ctx context.Context, in *idPath) (*api.Empty, error) {
			return api.NoContent, s.DeleteTier(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID)
		})
	api.Post(h, api.Op{ID: "sync-earn-tier", Path: base + "/tiers/{id}/sync", Summary: "Create the tier's product and prices in the provider again", Tags: tags, Access: edit},
		func(ctx context.Context, in *idPath) (*api.Body[EarnTierSaved], error) {
			out, err := s.SyncTier(ctx, api.WorkspaceFrom(ctx), in.ID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "map-earn-price", Path: base + "/unmapped-prices/{id}/map", Summary: "Map a provider price nobody set up here to a tier", Tags: tags, Access: edit, Status: 204},
		func(ctx context.Context, in *idBody[EarnMapPriceInput]) (*api.Empty, error) {
			return api.NoContent, s.MapPrice(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
		})
	api.Get(h, api.Op{ID: "get-upgrade-page", Path: base + "/upgrade-page", Summary: "The upgrade page editor: headline, tiers and their order, FAQ, the welcome page", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[EarnUpgradeEditor], error) {
			out, err := s.UpgradeEditor(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "preview-upgrade-page", Path: base + "/upgrade-page/preview", Summary: "Render the upgrade page with unsaved settings",
		Description: "Nothing is saved; the page's links and buttons don't work in the preview.", Tags: tags, Access: view},
		func(ctx context.Context, in *body[EarnPreviewInput]) (*api.Body[EarnPreview], error) {
			out, err := s.PreviewUpgrade(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-upgrade-page", Path: base + "/upgrade-page", Summary: "Save and publish the upgrade page", Tags: tags, Access: edit},
		func(ctx context.Context, in *body[UpgradePage]) (*api.Body[EarnUpgradeEditor], error) {
			out, err := s.SaveUpgradePage(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})

	// Paid members.
	api.Get(h, api.Op{ID: "list-paid-members", Path: base + "/members", Summary: "Paid members, with their counts by status",
		Description: "Amounts are in minor units, before tax. Test paid members show only with test=true.", Tags: tags, Access: view},
		func(ctx context.Context, in *membersInput) (*api.Body[EarnMemberList], error) {
			out, err := s.Members(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.EarnMemberFilter)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "export-paid-members", Path: base + "/members/export", Summary: "Export the paid members of a filter as CSV", Tags: tags,
		Access: api.Workspace(authz.SubscribersExport)},
		func(ctx context.Context, in *membersInput) (*csvFile, error) {
			ws := api.WorkspaceFrom(ctx)
			b, _, err := s.Export(ctx, ws, api.PrincipalFrom(ctx), in.EarnMemberFilter)
			if err != nil {
				return nil, err
			}
			return &csvFile{ContentType: "text/csv; charset=utf-8", CacheControl: "no-store",
				ContentDisposition: fmt.Sprintf(`attachment; filename="paid-members-%s-%s.csv"`, ws.Slug, s.now().UTC().Format("2006-01-02")), Body: b}, nil
		})
	api.Get(h, api.Op{ID: "get-paid-member", Path: base + "/members/{id}", Summary: "A paid member: status, price, source and timeline", Tags: tags, Access: view},
		func(ctx context.Context, in *idPath) (*api.Body[EarnMember], error) {
			out, err := s.Member(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-contact-paid-status", Path: base + "/contacts/{contactId}", Summary: "A contact's paid access, for the contact profile",
		Description: "Null when the contact has no paid access and never had.", Tags: tags, Access: view},
		func(ctx context.Context, in *contactPath) (*api.Body[*EarnMember], error) {
			out, err := s.ContactPaid(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ContactID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "give-complimentary-access", Path: base + "/members/complimentary", Summary: "Give complimentary access to a contact or a new address",
		Description: "A new address joins the tier's lists with double opt-in: access starts once they confirm. Nothing is created in the provider.", Tags: tags, Access: manage},
		func(ctx context.Context, in *body[EarnComplimentaryInput]) (*api.Body[EarnMember], error) {
			out, err := s.GiveComplimentary(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "change-complimentary-access", Path: base + "/members/{id}/complimentary", Summary: "Change complimentary access: tier, end date, reason", Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[EarnGrantInput]) (*api.Body[EarnMember], error) {
			out, err := s.ChangeComplimentary(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-complimentary-access", Path: base + "/members/{id}/complimentary", Summary: "End complimentary access now", Tags: tags, Access: manage},
		func(ctx context.Context, in *idPath) (*api.Empty, error) {
			return api.NoContent, s.RemoveComplimentary(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID)
		})
	api.Post(h, api.Op{ID: "cancel-paid-subscription", Path: base + "/members/{id}/cancel", Summary: "Cancel a subscription in the provider, now or at the end of the period", Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[EarnCancelInput]) (*api.Body[EarnMember], error) {
			out, err := s.CancelSubscription(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "resend-member-email", Path: base + "/members/{id}/resend", Summary: "Send a member email to a paid member again", Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[EarnResendInput]) (*api.Body[EarnResent], error) {
			out, err := s.Resend(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
			return api.OK(out), err
		})

	// Settings tabs.
	api.Get(h, api.Op{ID: "get-earn-access-rules", Path: base + "/settings/access", Summary: "Access: what happens when a payment fails, and who has access", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[EarnAccessRules], error) {
			out, err := s.AccessRules(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-earn-access-rules", Path: base + "/settings/access", Summary: "Save the access rules", Tags: tags, Access: pay},
		func(ctx context.Context, in *body[EarnAccessInput]) (*api.Body[EarnAccessRules], error) {
			out, err := s.SaveAccessRules(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-earn-pages", Path: base + "/settings/pages", Summary: "Pages: the Earn pages' addresses, terms of sale and support email", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[EarnPagesSettings], error) {
			out, err := s.Pages(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-earn-pages", Path: base + "/settings/pages", Summary: "Save the terms of sale and support email", Tags: tags, Access: edit},
		func(ctx context.Context, in *body[EarnPagesInput]) (*api.Body[EarnPagesSettings], error) {
			out, err := s.SavePages(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-member-emails", Path: base + "/settings/member-emails", Summary: "Member emails: the eight templates, their versions and when they're sent", Tags: tags, Access: view},
		func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[EarnMemberEmails], error) {
			out, err := s.MemberEmails(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})

	// Importing paid members.
	api.Post(h, api.Op{ID: "upload-paid-member-import", Path: base + "/imports", Summary: "Upload an export of paid members and find its columns",
		Description: "Needs subscribers:import too. Nothing changes in contacts or the provider until the import is started.", Tags: tags, Access: manage, MaxBodyBytes: MaxImportFile},
		func(ctx context.Context, in *importUpload) (*api.Body[EarnImport], error) {
			data := append([]byte(nil), in.RawBody...)
			out, err := s.UploadImport(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Source, in.FileName, data)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-paid-member-import", Path: base + "/imports/{id}", Summary: "An import of paid members: its columns, matches, choices and progress", Tags: tags, Access: view},
		func(ctx context.Context, in *idPath) (*api.Body[EarnImport], error) {
			out, err := s.Import(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-paid-member-import-columns", Path: base + "/imports/{id}/columns", Summary: "Pick the export's columns by hand", Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[EarnImportColumnsInput]) (*api.Body[EarnImport], error) {
			out, err := s.SetImportColumns(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "match-paid-member-import", Path: base + "/imports/{id}/match", Summary: "Match the rows with the provider's customers and subscriptions (read-only)", Tags: tags, Access: manage},
		func(ctx context.Context, in *idPath) (*api.Body[EarnImport], error) {
			out, err := s.MatchImport(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-paid-member-import-decisions", Path: base + "/imports/{id}/decisions", Summary: "Rows without a subscription, conflicts, prices and the welcome email", Tags: tags, Access: manage},
		func(ctx context.Context, in *idBody[EarnImportDecisions]) (*api.Body[EarnImport], error) {
			out, err := s.SetImportDecisions(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "start-paid-member-import", Path: base + "/imports/{id}/start", Summary: "Start importing in the background",
		Description: "Nothing changes in the provider: subscriptions keep billing as before.", Tags: tags, Access: manage},
		func(ctx context.Context, in *idPath) (*api.Body[EarnImport], error) {
			out, err := s.StartImport(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "cancel-paid-member-import", Path: base + "/imports/{id}", Summary: "Drop an import that hasn't started, with its file", Tags: tags, Access: manage},
		func(ctx context.Context, in *idPath) (*api.Empty, error) {
			return api.NoContent, s.CancelImport(ctx, api.WorkspaceFrom(ctx), in.ID)
		})
}
