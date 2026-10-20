package whatsapp

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

func parseID(raw, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, api.NotFound(what)
	}
	return id, nil
}

// WhatsAppCampaignPath names a WhatsApp campaign.
type WhatsAppCampaignPath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid" doc:"Campaign id"`
}

// WhatsAppContactPath names a contact.
type WhatsAppContactPath struct {
	api.WorkspaceInput
	ContactID string `path:"contactId" format:"uuid" doc:"Contact id"`
}

// WhatsAppCSV is a CSV download.
type WhatsAppCSV struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	Body               []byte
}

// Register adds the WhatsApp operations.
func Register(h huma.API, s *Service) {
	wtags, ctags := []string{"WhatsApp"}, []string{"WhatsApp campaigns"}
	view, manage := api.Workspace(authz.ProvidersView), api.Workspace(authz.ProvidersManage)
	human := api.WorkspaceHuman(authz.ProvidersManage)
	cedit := api.Workspace(authz.CampaignsEdit)

	// Sending providers › WhatsApp.
	api.Get(h, api.Op{ID: "whatsapp-overview", Path: "/workspaces/{workspace}/whatsapp", Summary: "WhatsApp: the account, templates, consent and events",
		Description: "Credentials are write-only: they're never returned.", Tags: wtags, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[WhatsAppOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "connect-whatsapp", Path: "/workspaces/{workspace}/whatsapp/account", Summary: "Connect a WhatsApp Business Account (logged)",
		Description: "Checks the values with Meta, seals them and shows the webhook to paste in Meta's app settings. Nothing is sent until Meta verifies it.",
		Tags:        wtags, Access: human, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body WhatsAppConnectInput
		}) (*api.Body[WhatsAppOverview], error) {
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			if err := s.Connect(ctx, ws, p, in.Body); err != nil {
				return nil, err
			}
			out, err := s.Overview(ctx, ws, p)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "replace-whatsapp-credentials", Path: "/workspaces/{workspace}/whatsapp/account/credentials",
		Summary: "Replace the credentials, or reconnect with a new token (logged)", Description: "Held sends go again once Meta accepts the token.",
		Tags: wtags, Access: human},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body WhatsAppCredentialsInput
		}) (*api.Body[WhatsAppOverview], error) {
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			if err := s.ReplaceCredentials(ctx, ws, p, in.Body); err != nil {
				return nil, err
			}
			out, err := s.Overview(ctx, ws, p)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "disconnect-whatsapp", Path: "/workspaces/{workspace}/whatsapp/account", Summary: "Disconnect WhatsApp: waiting messages are cancelled (logged)",
		Tags: wtags, Access: human},
		func(ctx context.Context, _ *api.WorkspaceInput) (*struct{}, error) {
			return nil, s.Disconnect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
		})
	api.Post(h, api.Op{ID: "check-whatsapp", Path: "/workspaces/{workspace}/whatsapp/account/check", Summary: "Read the account from Meta now and sync the templates",
		Tags: wtags, Access: manage},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[WhatsAppOverview], error) {
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			if err := s.Check(ctx, ws, p); err != nil {
				return nil, err
			}
			out, err := s.Overview(ctx, ws, p)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "sync-whatsapp-templates", Path: "/workspaces/{workspace}/whatsapp/templates/sync", Summary: "Sync the templates from Meta now (logged when something changed)",
		Tags: wtags, Access: manage},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[WhatsAppSyncResult], error) {
			out, err := s.Sync(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), false)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "send-whatsapp-account-test", Path: "/workspaces/{workspace}/whatsapp/test", Summary: "Send a test message to a member's WhatsApp (logged)",
		Description: "An approved template; the answer says when Meta delivered it.", Tags: wtags, Access: human},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body WhatsAppTestInput
		}) (*api.Body[WhatsAppTestResult], error) {
			out, err := s.SendAccountTest(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-whatsapp-test", Path: "/workspaces/{workspace}/whatsapp/tests/{messageId}", Summary: "A test message and what Meta reported",
		Tags: wtags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			MessageID string `path:"messageId" format:"uuid"`
		}) (*api.Body[WhatsAppTestResult], error) {
			id, err := parseID(in.MessageID, "Test message")
			if err != nil {
				return nil, err
			}
			out, err := s.TestStatus(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-whatsapp-events", Path: "/workspaces/{workspace}/whatsapp/events", Summary: "Meta's webhooks and what came of them",
		Tags: wtags, Access: view},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Before time.Time `query:"before" doc:"Events before this time"`
			Limit  int       `query:"limit" minimum:"0" maximum:"200"`
		}) (*api.Body[WhatsAppEventList], error) {
			var before *time.Time
			if !in.Before.IsZero() {
				before = &in.Before
			}
			out, err := s.Events(ctx, api.WorkspaceFrom(ctx), before, in.Limit)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-whatsapp-rates", Path: "/workspaces/{workspace}/whatsapp/rates", Summary: "The rate card costs are estimated from",
		Tags: wtags, Access: view},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[WhatsAppRates], error) {
			out, err := s.Rates(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-whatsapp-rates", Path: "/workspaces/{workspace}/whatsapp/rates", Summary: "Edit the rate card (logged)",
		Tags: wtags, Access: human},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body WhatsAppRatesInput
		}) (*api.Body[WhatsAppRates], error) {
			out, err := s.SaveRates(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-whatsapp-settings", Path: "/workspaces/{workspace}/whatsapp/settings", Summary: "Whether WhatsApp keeps to the quiet hours (logged)",
		Tags: wtags, Access: human},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body WhatsAppSettingsInput
		}) (*api.Body[WhatsAppOverview], error) {
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			if err := s.SaveSettings(ctx, ws, p, in.Body); err != nil {
				return nil, err
			}
			out, err := s.Overview(ctx, ws, p)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-whatsapp-estimate", Path: "/workspaces/{workspace}/whatsapp/estimate",
		Summary: "Who a list's WhatsApp consent reaches and what one template to them costs", Tags: wtags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			List     string `query:"list" format:"uuid" required:"true"`
			Template string `query:"template"`
		}) (*api.Body[WhatsAppEstimate], error) {
			out, err := s.Estimate(ctx, api.WorkspaceFrom(ctx), in.List, in.Template)
			return api.OK(out), err
		})

	// Campaigns.
	api.Post(h, api.Op{ID: "create-whatsapp-campaign", Path: "/workspaces/{workspace}/whatsapp-campaigns", Summary: "Make a WhatsApp campaign (logged)",
		Tags: ctags, Access: cedit, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body WhatsAppCampaignCreate
		}) (*api.Body[WhatsAppCampaign], error) {
			out, err := s.CreateCampaign(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-whatsapp-campaign", Path: "/workspaces/{workspace}/whatsapp-campaigns/{campaignId}", Summary: "The WhatsApp composer",
		Tags: ctags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			CampaignID string `path:"campaignId" format:"uuid"`
			Preview    string `query:"preview" doc:"The contact the preview shows"`
		}) (*api.Body[WhatsAppCampaign], error) {
			id, err := parseID(in.CampaignID, "WhatsApp campaign")
			if err != nil {
				return nil, err
			}
			pv, _ := uuid.Parse(in.Preview)
			out, err := s.Campaign(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, pv)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-whatsapp-campaign", Path: "/workspaces/{workspace}/whatsapp-campaigns/{campaignId}", Summary: "Autosave a WhatsApp campaign (logged)",
		Tags: ctags, Access: cedit},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			CampaignID string `path:"campaignId" format:"uuid"`
			Body       WhatsAppCampaignUpdate
		}) (*api.Body[WhatsAppCampaign], error) {
			id, err := parseID(in.CampaignID, "WhatsApp campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.UpdateCampaign(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "duplicate-whatsapp-campaign", Path: "/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/duplicate",
		Summary: "Copy a WhatsApp campaign as a draft (logged)", Tags: ctags, Access: cedit, Status: http.StatusCreated},
		func(ctx context.Context, in *WhatsAppCampaignPath) (*api.Body[WhatsAppCampaign], error) {
			id, err := parseID(in.CampaignID, "WhatsApp campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.DuplicateCampaign(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "send-whatsapp-campaign-test", Path: "/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/test",
		Summary: "Send the campaign to my WhatsApp (logged)", Tags: ctags, Access: api.WorkspaceHuman(authz.CampaignsEdit)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			CampaignID string `path:"campaignId" format:"uuid"`
			Body       WhatsAppCampaignTestInput
		}) (*api.Body[WhatsAppTestResult], error) {
			id, err := parseID(in.CampaignID, "WhatsApp campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.SendCampaignTest(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "review-whatsapp-campaign", Path: "/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/review",
		Summary: "Review & send: consent, the template, quality, the messaging limit and the cost", Tags: ctags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *WhatsAppCampaignPath) (*api.Body[WhatsAppReview], error) {
			id, err := parseID(in.CampaignID, "WhatsApp campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.Review(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "send-whatsapp-campaign", Path: "/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/send",
		Summary: "Send or schedule a WhatsApp campaign as reviewed (logged)", Description: "Needs campaigns:send and whatsapp:send; the workspace's approval rules apply.",
		Tags: ctags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			CampaignID string `path:"campaignId" format:"uuid"`
			Body       WhatsAppSendInput
		}) (*api.Body[WhatsAppSendResult], error) {
			id, err := parseID(in.CampaignID, "WhatsApp campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.Send(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "unschedule-whatsapp-campaign", Path: "/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/unschedule",
		Summary: "Take a scheduled WhatsApp campaign back to a draft (logged)", Tags: ctags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *WhatsAppCampaignPath) (*struct{}, error) {
			id, err := parseID(in.CampaignID, "WhatsApp campaign")
			if err != nil {
				return nil, err
			}
			return nil, s.Unschedule(ctx, api.WorkspaceFrom(ctx), id)
		})
	api.Post(h, api.Op{ID: "cancel-whatsapp-campaign", Path: "/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/cancel",
		Summary: "Stop a WhatsApp campaign: what wasn't sent doesn't go (logged)", Tags: ctags, Access: api.Workspace(authz.CampaignsPause)},
		func(ctx context.Context, in *WhatsAppCampaignPath) (*struct{}, error) {
			id, err := parseID(in.CampaignID, "WhatsApp campaign")
			if err != nil {
				return nil, err
			}
			return nil, s.Cancel(ctx, api.WorkspaceFrom(ctx), id)
		})
	api.Get(h, api.Op{ID: "get-whatsapp-report", Path: "/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/report",
		Summary: "The WhatsApp report: sent, delivered, read, failed, button clicks, replies, opt-outs and cost", Tags: ctags,
		Access: api.Workspace(authz.ReportsView)},
		func(ctx context.Context, in *WhatsAppCampaignPath) (*api.Body[WhatsAppReport], error) {
			id, err := parseID(in.CampaignID, "WhatsApp campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.CampaignReport(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "export-whatsapp-report", Path: "/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/report.csv",
		Summary: "The WhatsApp report as CSV, by country (logged)", Tags: ctags, Access: api.Workspace(authz.ReportsView)},
		func(ctx context.Context, in *WhatsAppCampaignPath) (*WhatsAppCSV, error) {
			id, err := parseID(in.CampaignID, "WhatsApp campaign")
			if err != nil {
				return nil, err
			}
			b, name, err := s.ReportCSV(ctx, api.WorkspaceFrom(ctx), id)
			if err != nil {
				return nil, err
			}
			return &WhatsAppCSV{ContentType: "text/csv; charset=utf-8", ContentDisposition: `attachment; filename="` + name + `"`, Body: b}, nil
		})

	// The automation step.
	api.Get(h, api.Op{ID: "get-automation-whatsapp-stats", Path: "/workspaces/{workspace}/automations/{id}/whatsapp-stats",
		Summary: "A flow's WhatsApp steps over 30 days", Tags: wtags, Access: api.Workspace(authz.AutomationsView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			ID string `path:"id" format:"uuid"`
		}) (*api.Body[WhatsAppStepStatsList], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			out, err := s.StepStats(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})

	// The contact profile.
	api.Get(h, api.Op{ID: "get-contact-whatsapp", Path: "/workspaces/{workspace}/contacts/{contactId}/whatsapp",
		Summary: "A contact's WhatsApp: consent per list, the customer service window and the conversation", Tags: wtags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *WhatsAppContactPath) (*api.Body[WhatsAppContactCard], error) {
			id, err := parseID(in.ContactID, "Contact")
			if err != nil {
				return nil, err
			}
			out, err := s.ContactCard(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-contact-whatsapp-consent", Path: "/workspaces/{workspace}/contacts/{contactId}/whatsapp/consent",
		Summary: "Change a contact's WhatsApp consent on a list (logged)", Description: "Consent needs its evidence; an opt-out by the person can't be overridden.",
		Tags: wtags, Access: api.Workspace(authz.SubscribersEdit)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			ContactID string `path:"contactId" format:"uuid"`
			Body      WhatsAppConsentInput
		}) (*api.Body[WhatsAppContactCard], error) {
			id, err := parseID(in.ContactID, "Contact")
			if err != nil {
				return nil, err
			}
			out, err := s.SetConsentByTeam(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "reply-contact-whatsapp", Path: "/workspaces/{workspace}/contacts/{contactId}/whatsapp/reply",
		Summary: "Reply with free text inside the 24-hour customer service window", Tags: wtags, Access: api.WorkspaceHuman(authz.WhatsAppSend)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			ContactID string `path:"contactId" format:"uuid"`
			Body      WhatsAppReplyInput
		}) (*api.Body[WhatsAppThreadItem], error) {
			id, err := parseID(in.ContactID, "Contact")
			if err != nil {
				return nil, err
			}
			out, err := s.Reply(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
}
