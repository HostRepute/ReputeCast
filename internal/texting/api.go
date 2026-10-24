package texting

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/channels"
)

func parseID(raw, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, api.NotFound(what)
	}
	return id, nil
}

// SmsProviderPath names an SMS provider.
type SmsProviderPath struct {
	api.WorkspaceInput
	ProviderID string `path:"providerId" format:"uuid" doc:"SMS provider id"`
}

// SmsSenderPath names a sender.
type SmsSenderPath struct {
	api.WorkspaceInput
	SenderID string `path:"senderId" format:"uuid" doc:"Sender id"`
}

// SmsCampaignPath names an SMS campaign.
type SmsCampaignPath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid" doc:"Campaign id"`
}

// SmsContactPath names a contact.
type SmsContactPath struct {
	api.WorkspaceInput
	ContactID string `path:"contactId" format:"uuid" doc:"Contact id"`
}

// SmsIDPath names a row by id.
type SmsIDPath struct {
	api.WorkspaceInput
	ID string `path:"id" format:"uuid"`
}

// SmsChannels is the channel switch's state.
type SmsChannels struct {
	Channels []channels.ChannelStatus `json:"channels"`
	// SwitchShown is true once a second channel is set up; until then the
	// header offers "Add a channel".
	SwitchShown bool `json:"switchShown"`
}

// SmsCatalogList is the SMS adapters this build has.
type SmsCatalogList struct {
	Items []SmsCatalogItem `json:"items"`
}

// SmsSenderRows is the senders table.
type SmsSenderRows struct {
	Items []SmsSenderRow `json:"items"`
}

// SmsCampaignVersionList is a campaign's versions.
type SmsCampaignVersionList struct {
	Items []SmsCampaignVersion `json:"items"`
}

// SmsPreviewContactList is who the preview can be shown as.
type SmsPreviewContactList struct {
	Items []SmsPreviewContact `json:"items"`
}

// SmsTestNumberList is the test numbers with their last test.
type SmsTestNumberList struct {
	Items []SmsTestNumber `json:"items"`
}

// SmsMessageOptionList is what the route tester can count.
type SmsMessageOptionList struct {
	Items []SmsMessageOption `json:"items"`
}

// SmsSuppressionsAdded says how many numbers went on the list.
type SmsSuppressionsAdded struct {
	Added int `json:"added"`
}

// Register adds the SMS operations.
func Register(h huma.API, s *Service) {
	ptags, ctags, atags := []string{"SMS providers"}, []string{"SMS campaigns"}, []string{"SMS audience"}
	pview, pmanage := api.Workspace(authz.ProvidersView), api.Workspace(authz.ProvidersManage)
	cview, cedit := api.Workspace(authz.CampaignsView), api.Workspace(authz.CampaignsEdit)

	api.Get(h, api.Op{ID: "list-channels", Path: "/workspaces/{workspace}/channels", Summary: "Which channels the workspace can send on: the Sending providers switch",
		Tags: []string{"Channels"}, Access: pview},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[SmsChannels], error) {
			st := channels.Statuses(ctx, api.WorkspaceFrom(ctx).ID)
			return api.OK(SmsChannels{Channels: st, SwitchShown: channels.SwitchShown(st)}), nil
		})

	// Providers, senders and registrations.
	api.Get(h, api.Op{ID: "sms-overview", Path: "/workspaces/{workspace}/sms/overview", Summary: "SMS providers: health, spend, senders and routes",
		Description: "Credentials are never returned.", Tags: ptags, Access: pview},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[SmsOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "sms-catalog", Path: "/workspaces/{workspace}/sms/catalog", Summary: "The SMS adapters, with the capabilities their manifests declare",
		Tags: ptags, Access: pview},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[SmsCatalogList], error) {
			out, err := s.Catalog(ctx, api.WorkspaceFrom(ctx))
			return api.OK(SmsCatalogList{Items: out}), err
		})
	api.Post(h, api.Op{ID: "connect-sms-provider", Path: "/workspaces/{workspace}/sms/providers", Summary: "Connect an SMS provider (logged)",
		Description: "Checks the credentials, registers the receipt and inbound webhooks where the API allows it, and lists the numbers in the account. Credentials are sealed and never returned.",
		Tags:        ptags, Access: pmanage, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body SmsConnectInput
		}) (*api.Body[SmsConnectResult], error) {
			out, err := s.Connect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-sms-account-numbers", Path: "/workspaces/{workspace}/sms/providers/{providerId}/numbers",
		Summary: "The numbers in the provider's account, with their registration", Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *SmsProviderPath) (*api.Body[SmsFoundSenderList], error) {
			id, err := parseID(in.ProviderID, "SMS provider")
			if err != nil {
				return nil, err
			}
			out, err := s.AccountNumbers(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "import-sms-senders", Path: "/workspaces/{workspace}/sms/providers/{providerId}/senders", Summary: "Import numbers from the provider's account (logged)",
		Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *struct {
			SmsProviderPath
			Body SmsImportInput
		}) (*api.Body[SmsSenderRows], error) {
			id, err := parseID(in.ProviderID, "SMS provider")
			if err != nil {
				return nil, err
			}
			out, err := s.ImportSenders(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(SmsSenderRows{Items: out}), err
		})
	api.Post(h, api.Op{ID: "test-sms-provider", Path: "/workspaces/{workspace}/sms/providers/{providerId}/test", Summary: "Text a test to a member's phone through this provider (logged)",
		Description: "The timeline waits for the delivery receipt: poll the message.", Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *struct {
			SmsProviderPath
			Body SmsProviderTestInput
		}) (*api.Body[SmsMessageView], error) {
			id, err := parseID(in.ProviderID, "SMS provider")
			if err != nil {
				return nil, err
			}
			out, err := s.ProviderTest(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "replace-sms-credentials", Path: "/workspaces/{workspace}/sms/providers/{providerId}/credentials", Summary: "Replace a provider's credentials (logged)",
		Description: "Checked before they're stored, sealed, never returned. Held texts go once they work.", Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *struct {
			SmsProviderPath
			Body SmsCredentialsInput
		}) (*api.Body[SmsOverview], error) {
			id, err := parseID(in.ProviderID, "SMS provider")
			if err != nil {
				return nil, err
			}
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			if err := s.ReplaceCredentials(ctx, ws, p, id, in.Body); err != nil {
				return nil, err
			}
			out, err := s.Overview(ctx, ws, p)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "register-sms-webhooks", Path: "/workspaces/{workspace}/sms/providers/{providerId}/webhooks", Summary: "Register the receipt and inbound webhooks again (logged)",
		Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *SmsProviderPath) (*api.Body[SmsWebhookSetup], error) {
			id, err := parseID(in.ProviderID, "SMS provider")
			if err != nil {
				return nil, err
			}
			out, err := s.RegisterWebhooks(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-sms-provider", Path: "/workspaces/{workspace}/sms/providers/{providerId}/check", Summary: "Check a provider now (Retry now during an outage)",
		Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *SmsProviderPath) (*api.Body[SmsOverview], error) {
			id, err := parseID(in.ProviderID, "SMS provider")
			if err != nil {
				return nil, err
			}
			out, err := s.CheckNow(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-sms-provider", Path: "/workspaces/{workspace}/sms/providers/{providerId}", Summary: "Disconnect a provider no route uses (logged)",
		Tags: ptags, Access: pmanage, Status: http.StatusNoContent},
		func(ctx context.Context, in *SmsProviderPath) (*struct{}, error) {
			id, err := parseID(in.ProviderID, "SMS provider")
			if err != nil {
				return nil, err
			}
			return nil, s.RemoveProvider(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Post(h, api.Op{ID: "finish-sms-setup", Path: "/workspaces/{workspace}/sms/setups/{id}/finish", Summary: "Finish the connect wizard (logged)", Tags: ptags, Access: pmanage,
		Status: http.StatusNoContent},
		func(ctx context.Context, in *SmsIDPath) (*struct{}, error) {
			id, err := parseID(in.ID, "Setup")
			if err != nil {
				return nil, err
			}
			return nil, s.FinishSetup(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Get(h, api.Op{ID: "get-sms-message", Path: "/workspaces/{workspace}/sms/messages/{id}", Summary: "One text with its timeline (a test's delivery receipt)",
		Tags: ptags, Access: pview},
		func(ctx context.Context, in *SmsIDPath) (*api.Body[SmsMessageView], error) {
			id, err := parseID(in.ID, "Text")
			if err != nil {
				return nil, err
			}
			out, err := s.Message(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "add-sms-sender-id", Path: "/workspaces/{workspace}/sms/senders", Summary: "Add an alphanumeric sender ID with its countries (logged)",
		Tags: ptags, Access: pmanage, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body SmsSenderIDInput
		}) (*api.Body[SmsSenderRows], error) {
			out, err := s.AddSenderID(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(SmsSenderRows{Items: out}), err
		})
	api.Patch(h, api.Op{ID: "update-sms-sender", Path: "/workspaces/{workspace}/sms/senders/{senderId}", Summary: "Change a sender's countries (logged)", Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *struct {
			SmsSenderPath
			Body SmsSenderUpdate
		}) (*api.Body[SmsSenderRows], error) {
			id, err := parseID(in.SenderID, "Sender")
			if err != nil {
				return nil, err
			}
			out, err := s.UpdateSender(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(SmsSenderRows{Items: out}), err
		})
	api.Delete(h, api.Op{ID: "remove-sms-sender", Path: "/workspaces/{workspace}/sms/senders/{senderId}", Summary: "Remove a sender no route uses (logged)", Tags: ptags,
		Access: pmanage, Status: http.StatusNoContent},
		func(ctx context.Context, in *SmsSenderPath) (*struct{}, error) {
			id, err := parseID(in.SenderID, "Sender")
			if err != nil {
				return nil, err
			}
			return nil, s.RemoveSender(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Get(h, api.Op{ID: "get-sms-registration", Path: "/workspaces/{workspace}/sms/senders/{senderId}/registration",
		Summary: "A sender's registrations, and what the provider's form asks for", Tags: ptags, Access: pview},
		func(ctx context.Context, in *SmsSenderPath) (*api.Body[SmsRegistrationPanel], error) {
			id, err := parseID(in.SenderID, "Sender")
			if err != nil {
				return nil, err
			}
			out, err := s.Registration(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-sms-registration", Path: "/workspaces/{workspace}/sms/senders/{senderId}/registration",
		Summary: "Set a registration status by hand, with its date (logged)", Description: "For providers that don't report it through their API.",
		Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *struct {
			SmsSenderPath
			Body SmsRegistrationInput
		}) (*api.Body[SmsRegistrationPanel], error) {
			id, err := parseID(in.SenderID, "Sender")
			if err != nil {
				return nil, err
			}
			out, err := s.SetRegistration(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "read-sms-registration", Path: "/workspaces/{workspace}/sms/senders/{senderId}/registration/read",
		Summary: "Read the registration statuses from the provider now", Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *SmsSenderPath) (*api.Body[SmsRegistrationPanel], error) {
			id, err := parseID(in.SenderID, "Sender")
			if err != nil {
				return nil, err
			}
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			panel, err := s.Registration(ctx, ws, p, id)
			if err != nil {
				return nil, err
			}
			pid, _ := uuid.Parse(panel.Sender.ProviderID)
			if prov, err := s.Q.SmsProvider(ctx, smsProviderParams(pid, ws.ID)); err == nil {
				if err := s.ReadRegistrations(ctx, prov); err != nil {
					return nil, api.Err(http.StatusBadGateway, "provider_unreachable", noAnswer(ctx, prov.Name, err))
				}
				s.releasePending(ctx, ws)
			}
			out, err := s.Registration(ctx, ws, p, id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-sms-registration-pages", Path: "/workspaces/{workspace}/sms/senders/{senderId}/registration/check-pages",
		Summary: "Check the pages carriers look at: the terms page and the sign-up page", Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *SmsSenderPath) (*api.Body[SmsRegistrationPanel], error) {
			id, err := parseID(in.SenderID, "Sender")
			if err != nil {
				return nil, err
			}
			out, err := s.CheckPages(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})

	// Routing & rules.
	api.Get(h, api.Op{ID: "get-sms-rules", Path: "/workspaces/{workspace}/sms/rules", Summary: "Routes, quiet hours, keywords and replies, the budget and guards",
		Tags: ptags, Access: pview},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[SmsRules], error) {
			out, err := s.Rules(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-sms-rules", Path: "/workspaces/{workspace}/sms/rules", Summary: "Save the SMS rules (logged)",
		Description: "Quiet hours can be narrowed, never widened past 08:00–21:00. STOP and HELP stay. Applies to texts not yet sent.", Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body SmsRulesInput
		}) (*api.Body[SmsRules], error) {
			out, err := s.SaveRules(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-sms-routes", Path: "/workspaces/{workspace}/sms/routes", Summary: "Replace the routes by country (logged)",
		Description: "A country takes one route; a fallback must be registered for the route's countries.", Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body SmsRoutesInput
		}) (*api.Body[SmsRules], error) {
			out, err := s.SaveRoutes(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "add-sms-test-number", Path: "/workspaces/{workspace}/sms/test-numbers", Summary: "Add a member's phone for test texts (logged)", Tags: ptags,
		Access: pmanage, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body SmsTestNumberInput
		}) (*api.Body[SmsRules], error) {
			out, err := s.AddTestNumber(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-sms-test-number", Path: "/workspaces/{workspace}/sms/test-numbers/{id}", Summary: "Remove a test number (logged)", Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *SmsIDPath) (*api.Body[SmsRules], error) {
			id, err := parseID(in.ID, "Test number")
			if err != nil {
				return nil, err
			}
			out, err := s.RemoveTestNumber(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-sms-test-numbers", Path: "/workspaces/{workspace}/sms/test-numbers", Summary: "Test numbers with their last test", Tags: ptags, Access: cview},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[SmsTestNumberList], error) {
			out, err := s.testNumbers(ctx, api.WorkspaceFrom(ctx), nil)
			return api.OK(SmsTestNumberList{Items: out}), err
		})
	api.Post(h, api.Op{ID: "test-sms-route", Path: "/workspaces/{workspace}/sms/route-test", Summary: "Which sender would text a number, at what price, and when",
		Tags: ptags, Access: pview},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body SmsRouteTestInput
		}) (*api.Body[SmsRouteTest], error) {
			out, err := s.RouteTest(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-sms-message-options", Path: "/workspaces/{workspace}/sms/message-options", Summary: "Texts the route tester can count",
		Tags: ptags, Access: pview},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[SmsMessageOptionList], error) {
			out, err := s.MessageOptions(ctx, api.WorkspaceFrom(ctx))
			return api.OK(SmsMessageOptionList{Items: out}), err
		})
	api.Put(h, api.Op{ID: "set-sms-short-domain", Path: "/workspaces/{workspace}/sms/short-domain", Summary: "Set the short-link domain (logged)",
		Description: "Verified by a CNAME to the instance's host.", Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body SmsShortDomainInput
		}) (*api.Body[SmsRules], error) {
			out, err := s.SetShortDomain(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-sms-short-domain", Path: "/workspaces/{workspace}/sms/short-domain/check", Summary: "Check the short-link domain's DNS now",
		Tags: ptags, Access: pmanage},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[SmsRules], error) {
			out, err := s.CheckShortDomain(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-sms-events", Path: "/workspaces/{workspace}/sms/events", Summary: "Raw provider events beside their normalized ones, and opt-out sync",
		Description: "Raw events are kept 90 days.", Tags: ptags, Access: pview},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Kind     string `query:"kind" enum:"all,receipts,keywords,replies,failures" default:"all"`
			Provider string `query:"provider" doc:"An SMS provider id"`
			Page     int    `query:"page" minimum:"1" default:"1"`
		}) (*api.Body[SmsEventList], error) {
			out, err := s.Events(ctx, api.WorkspaceFrom(ctx), SmsEventFilter{Kind: in.Kind, Provider: in.Provider, Page: in.Page})
			return api.OK(out), err
		})

	// The SMS suppression list.
	api.Get(h, api.Op{ID: "list-sms-suppressions", Path: "/workspaces/{workspace}/sms/suppressions", Summary: "The SMS suppression list, numbers partly masked",
		Tags: atags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Reason   string `query:"reason" enum:"opted_out,landline,not_in_service,invalid,filtered,provider_list,manual"`
			Provider string `query:"provider" doc:"An SMS provider id"`
			Query    string `query:"q" maxLength:"40" doc:"Digits of the number"`
			Page     int    `query:"page" minimum:"1" default:"1"`
		}) (*api.Body[SmsSuppressionList], error) {
			out, err := s.Suppressions(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), SmsSuppressionFilter{Reason: in.Reason, Provider: in.Provider, Query: in.Query, Page: in.Page})
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "add-sms-suppressions", Path: "/workspaces/{workspace}/sms/suppressions", Summary: "Put numbers on the SMS suppression list (logged)",
		Tags: atags, Access: api.Workspace(authz.SubscribersEdit), Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body SmsSuppressionAdd
		}) (*api.Body[SmsSuppressionsAdded], error) {
			n, err := s.AddSuppressions(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(SmsSuppressionsAdded{Added: n}), err
		})
	api.Post(h, api.Op{ID: "remove-sms-suppression", Path: "/workspaces/{workspace}/sms/suppressions/{id}/remove", Summary: "Take a number off the SMS suppression list, with a reason (logged)",
		Description: "A number that opted out by text can't be removed: only the person can come back, by texting START.", Tags: atags,
		Access: api.Workspace(authz.SubscribersEdit), Status: http.StatusNoContent},
		func(ctx context.Context, in *struct {
			SmsIDPath
			Body SmsSuppressionRemove
		}) (*struct{}, error) {
			id, err := parseID(in.ID, "Suppressed number")
			if err != nil {
				return nil, err
			}
			return nil, s.RemoveSuppression(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
		})
	api.Get(h, api.Op{ID: "get-contact-channels", Path: "/workspaces/{workspace}/contacts/{contactId}/channels",
		Summary: "A contact's phone and consent per list on each channel, with the evidence", Tags: atags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *SmsContactPath) (*api.Body[SmsContactChannels], error) {
			id, err := parseID(in.ContactID, "Contact")
			if err != nil {
				return nil, err
			}
			out, err := s.ContactChannels(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-contact-phone", Path: "/workspaces/{workspace}/contacts/{contactId}/phone", Summary: "Set or remove a contact's phone number (logged)",
		Tags: atags, Access: api.Workspace(authz.SubscribersEdit)},
		func(ctx context.Context, in *struct {
			SmsContactPath
			Body SmsPhoneInput
		}) (*api.Body[SmsContactChannels], error) {
			id, err := parseID(in.ContactID, "Contact")
			if err != nil {
				return nil, err
			}
			out, err := s.SetPhone(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-contact-sms-consent", Path: "/workspaces/{workspace}/contacts/{contactId}/sms-consent", Summary: "Change a contact's SMS consent on a list (logged)",
		Description: "Giving consent needs its evidence: the exact consent text, where it was given and when. A text opt-out can't be overridden.",
		Tags:        atags, Access: api.Workspace(authz.SubscribersEdit)},
		func(ctx context.Context, in *struct {
			SmsContactPath
			Body SmsConsentInput
		}) (*api.Body[SmsContactChannels], error) {
			id, err := parseID(in.ContactID, "Contact")
			if err != nil {
				return nil, err
			}
			out, err := s.SetConsentByTeam(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})

	// SMS campaigns.
	api.Post(h, api.Op{ID: "create-sms-campaign", Path: "/workspaces/{workspace}/sms-campaigns", Summary: "Create an SMS campaign (logged)", Tags: ctags, Access: cedit,
		Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body SmsCampaignCreate
		}) (*api.Body[SmsCampaign], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-sms-campaign", Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}", Summary: "The SMS composer: text, counter, audience, routes and cost",
		Tags: ctags, Access: cview},
		func(ctx context.Context, in *SmsCampaignPath) (*api.Body[SmsCampaign], error) {
			id, err := parseID(in.CampaignID, "SMS campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-sms-campaign", Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}", Summary: "Autosave an SMS campaign (logged)",
		Description: "Answers 409 version_conflict when someone saved a newer version.", Tags: ctags, Access: cedit},
		func(ctx context.Context, in *struct {
			SmsCampaignPath
			Body SmsCampaignUpdate
		}) (*api.Body[SmsCampaign], error) {
			id, err := parseID(in.CampaignID, "SMS campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.Update(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-sms-campaign-versions", Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}/versions", Summary: "Saved versions of the text",
		Tags: ctags, Access: cview},
		func(ctx context.Context, in *SmsCampaignPath) (*api.Body[SmsCampaignVersionList], error) {
			id, err := parseID(in.CampaignID, "SMS campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.Versions(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(SmsCampaignVersionList{Items: out}), err
		})
	api.Post(h, api.Op{ID: "preview-sms-campaign", Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}/preview", Summary: "The phone preview, as a contact and a country see it",
		Tags: ctags, Access: cview},
		func(ctx context.Context, in *struct {
			SmsCampaignPath
			Body SmsPreviewInput
		}) (*api.Body[SmsPreview], error) {
			id, err := parseID(in.CampaignID, "SMS campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.Preview(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-sms-preview-contacts", Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}/preview-contacts", Summary: "Recipients to preview as",
		Tags: ctags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *SmsCampaignPath) (*api.Body[SmsPreviewContactList], error) {
			id, err := parseID(in.CampaignID, "SMS campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.PreviewContacts(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(SmsPreviewContactList{Items: out}), err
		})
	api.Post(h, api.Op{ID: "send-sms-campaign-test", Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}/test", Summary: "Text the campaign to test numbers (logged)",
		Description: "Each test number gets the text from the sender its country would get. Tests count toward the budget.", Tags: ctags, Access: cedit},
		func(ctx context.Context, in *struct {
			SmsCampaignPath
			Body SmsTestInput
		}) (*api.Body[SmsTestNumberList], error) {
			id, err := parseID(in.CampaignID, "SMS campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.SendTest(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(SmsTestNumberList{Items: out}), err
		})
	api.Get(h, api.Op{ID: "review-sms-campaign", Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}/review", Summary: "Review & send: the SMS checks, routes, cost, schedule and approvals",
		Tags: ctags, Access: cview},
		func(ctx context.Context, in *SmsCampaignPath) (*api.Body[SmsReview], error) {
			id, err := parseID(in.CampaignID, "SMS campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.Review(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-sms-send-settings", Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}/send-settings",
		Summary: "When the campaign goes out, pending senders, auto-pause thresholds (logged)", Tags: ctags, Access: cedit},
		func(ctx context.Context, in *struct {
			SmsCampaignPath
			Body SmsSendSettings
		}) (*api.Body[SmsReview], error) {
			id, err := parseID(in.CampaignID, "SMS campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.SaveSendSettings(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "send-sms-campaign", Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}/send", Summary: "Send or schedule an SMS campaign as reviewed (logged)",
		Description: "Needs sms:send too. Sends above 1,000 recipients or $100 need the recipient count typed. Answers 409 review_changed when the review no longer describes the campaign.",
		Tags:        ctags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *struct {
			SmsCampaignPath
			Body SmsSendInput
		}) (*api.Body[SmsSendResult], error) {
			id, err := parseID(in.CampaignID, "SMS campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.Send(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "unschedule-sms-campaign", Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}/unschedule", Summary: "Take a scheduled SMS campaign back to a draft (logged)",
		Tags: ctags, Access: api.Workspace(authz.CampaignsSend), Status: http.StatusNoContent},
		func(ctx context.Context, in *SmsCampaignPath) (*struct{}, error) {
			id, err := parseID(in.CampaignID, "SMS campaign")
			if err != nil {
				return nil, err
			}
			return nil, s.Unschedule(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	for _, a := range []struct {
		action, id, summary string
		perm                authz.Permission
	}{
		{"pause", "pause-sms-send", "Pause an SMS send (logged)", authz.CampaignsPause},
		{"resume", "resume-sms-send", "Resume a paused SMS send (logged; needs sms:send too)", authz.CampaignsSend},
		{"cancel", "cancel-sms-send", "Cancel the texts not sent yet (logged)", authz.CampaignsPause},
	} {
		api.Post(h, api.Op{ID: a.id, Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}/" + a.action, Summary: a.summary, Tags: ctags, Access: api.Workspace(a.perm),
			Status: http.StatusNoContent},
			func(ctx context.Context, in *struct {
				SmsCampaignPath
				Body struct {
					Reason string `json:"reason,omitempty" maxLength:"500"`
				}
			}) (*struct{}, error) {
				id, err := parseID(in.CampaignID, "SMS campaign")
				if err != nil {
					return nil, err
				}
				return nil, s.Act(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, SmsSendAction{Action: a.action, Reason: in.Body.Reason})
			})
	}
	api.Get(h, api.Op{ID: "get-sms-report", Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}/report",
		Summary: "The SMS report: delivery, failures by reason with the provider's codes, clicks, replies, cost, progress", Tags: ctags, Access: api.Workspace(authz.ReportsView)},
		func(ctx context.Context, in *SmsCampaignPath) (*api.Body[SmsReport], error) {
			id, err := parseID(in.CampaignID, "SMS campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.Report(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "export-sms-report", Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}/report.csv",
		Summary: "Download the SMS report by country as CSV: counts only (logged)", Tags: ctags, Access: api.Workspace(authz.ReportsView)},
		func(ctx context.Context, in *SmsCampaignPath) (*SmsCSV, error) {
			id, err := parseID(in.CampaignID, "SMS campaign")
			if err != nil {
				return nil, err
			}
			return s.ReportCSV(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Post(h, api.Op{ID: "duplicate-sms-campaign", Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}/duplicate", Summary: "Copy an SMS campaign into a new draft (logged)",
		Tags: ctags, Access: cedit, Status: http.StatusCreated},
		func(ctx context.Context, in *SmsCampaignPath) (*api.Body[SmsCampaign], error) {
			id, err := parseID(in.CampaignID, "SMS campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.Duplicate(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-sms-replies", Path: "/workspaces/{workspace}/sms-campaigns/{campaignId}/replies", Summary: "A campaign's replies: STOP, HELP and others",
		Tags: ctags, Access: api.Workspace(authz.ReportsView)},
		func(ctx context.Context, in *struct {
			SmsCampaignPath
			Kind   string `query:"kind" enum:"all,opt_out,help,other" default:"all"`
			Offset int    `query:"offset" minimum:"0" default:"0"`
			Limit  int    `query:"limit" minimum:"1" maximum:"100" default:"50"`
		}) (*api.Body[SmsReplyList], error) {
			id, err := parseID(in.CampaignID, "SMS campaign")
			if err != nil {
				return nil, err
			}
			kind := in.Kind
			if kind == "all" {
				kind = ""
			}
			out, err := s.Replies(ctx, api.WorkspaceFrom(ctx), id, kind, in.Limit, in.Offset)
			return api.OK(out), err
		})

	// Automations and the terms page.
	api.Get(h, api.Op{ID: "get-automation-sms-stats", Path: "/workspaces/{workspace}/automations/{id}/sms-stats",
		Summary: "A flow's SMS steps over 30 days: sent, delivered, clicks, opt-outs, skipped and cost", Tags: []string{"Automations"}, Access: api.Workspace(authz.AutomationsView)},
		func(ctx context.Context, in *SmsIDPath) (*api.Body[SmsStepStatsList], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			out, err := s.StepStats(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "estimate-sms-step", Path: "/workspaces/{workspace}/sms/step-estimate", Summary: "An SMS step's counter and estimated cost",
		Tags: []string{"Automations"}, Access: api.Workspace(authz.AutomationsView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body SmsStepEstimateInput
		}) (*api.Body[SmsStepEstimate], error) {
			out, err := s.StepEstimate(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-sms-terms", Path: "/workspaces/{workspace}/sms/terms", Summary: "The text message terms page: its live version and the generated text",
		Tags: []string{"Forms"}, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[SmsTerms], error) {
			out, err := s.Terms(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "publish-sms-terms", Path: "/workspaces/{workspace}/sms/terms", Summary: "Publish a new version of the text message terms (logged)",
		Description: "Consent given from now on records this version.", Tags: []string{"Forms"}, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body SmsTermsContent
		}) (*api.Body[SmsTerms], error) {
			out, err := s.PublishTerms(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
}
