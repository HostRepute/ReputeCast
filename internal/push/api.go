package push

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

// PushSitePath names a push site.
type PushSitePath struct {
	api.WorkspaceInput
	SiteID string `path:"siteId" format:"uuid" doc:"Push site id"`
}

// PushPromptPath names a push prompt (its form).
type PushPromptPath struct {
	api.WorkspaceInput
	FormID string `path:"formId" format:"uuid" doc:"Push prompt (form) id"`
}

// PushCampaignPath names a push campaign.
type PushCampaignPath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid" doc:"Campaign id"`
}

// PushContactPath names a contact.
type PushContactPath struct {
	api.WorkspaceInput
	ContactID string `path:"contactId" format:"uuid" doc:"Contact id"`
}

// PushCSV is a CSV download.
type PushCSV struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	Body               []byte
}

// Register adds the push operations.
func Register(h huma.API, s *Service) {
	ptags, ctags := []string{"Push"}, []string{"Push campaigns"}
	pview, pmanage := api.Workspace(authz.ProvidersView), api.Workspace(authz.ProvidersManage)
	cedit := api.Workspace(authz.CampaignsEdit)

	// Setup: keys, guards, sites, subscribers, prompts.
	api.Get(h, api.Op{ID: "push-setup", Path: "/workspaces/{workspace}/push/setup", Summary: "Push setup: keys, sites, subscribers and prompts",
		Description: "The private key is never returned.", Tags: ptags, Access: pview},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[PushSetup], error) {
			out, err := s.Setup(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "enable-push", Path: "/workspaces/{workspace}/push/enable", Summary: "Generate the keys and turn push on (logged)",
		Description: "Makes the VAPID key pair (the private key is sealed and never returned) and adds the publication website as a push site when there is one.",
		Tags:        ptags, Access: pmanage},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body PushEnableInput
		}) (*api.Body[PushSetup], error) {
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			if err := s.Enable(ctx, ws, p, in.Body); err != nil {
				return nil, err
			}
			out, err := s.Setup(ctx, ws, p)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-push-contact", Path: "/workspaces/{workspace}/push/contact", Summary: "Change the contact the push services get (logged)",
		Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body PushContactInput
		}) (*api.Body[PushSetup], error) {
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			if err := s.SetContact(ctx, ws, in.Body); err != nil {
				return nil, err
			}
			out, err := s.Setup(ctx, ws, p)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "push-rotation", Path: "/workspaces/{workspace}/push/keys/rotation", Summary: "What rotating the keys would end",
		Tags: ptags, Access: pmanage},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[PushRotation], error) {
			out, err := s.Rotation(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "rotate-push-keys", Path: "/workspaces/{workspace}/push/keys/rotate", Summary: "Rotate the keys: every subscription ends (logged)",
		Description: "Needs the number of subscriptions that end, typed. Readers have to allow notifications again.", Tags: ptags,
		Access: api.WorkspaceHuman(authz.ProvidersManage)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body PushRotateInput
		}) (*api.Body[PushRotation], error) {
			out, err := s.Rotate(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-push-settings", Path: "/workspaces/{workspace}/push/settings", Summary: "Change the frequency cap and quiet hours (logged)",
		Tags: ptags, Access: api.WorkspaceHuman(authz.ProvidersManage)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body PushSettingsInput
		}) (*api.Body[PushSetup], error) {
			ws, p := api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx)
			if err := s.SaveSettings(ctx, ws, p, in.Body); err != nil {
				return nil, err
			}
			out, err := s.Setup(ctx, ws, p)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "add-push-site", Path: "/workspaces/{workspace}/push/sites", Summary: "Add a site readers can turn notifications on from (logged)",
		Tags: ptags, Access: pmanage, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body PushSiteInput
		}) (*api.Body[PushSite], error) {
			out, err := s.AddSite(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-push-site", Path: "/workspaces/{workspace}/push/sites/{siteId}", Summary: "Change a site's label, worker and icon (logged)",
		Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			SiteID string `path:"siteId" format:"uuid"`
			Body   PushSiteUpdate
		}) (*api.Body[PushSite], error) {
			id, err := parseID(in.SiteID, "Push site")
			if err != nil {
				return nil, err
			}
			out, err := s.UpdateSite(ctx, api.WorkspaceFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-push-site", Path: "/workspaces/{workspace}/push/sites/{siteId}", Summary: "Remove a site: its subscriptions end (logged)",
		Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *PushSitePath) (*struct{}, error) {
			id, err := parseID(in.SiteID, "Push site")
			if err != nil {
				return nil, err
			}
			return nil, s.RemoveSite(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
		})
	api.Post(h, api.Op{ID: "check-push-site", Path: "/workspaces/{workspace}/push/sites/{siteId}/check", Summary: "Fetch the site's service worker now",
		Description: "Reports whether the file was found, its version and its scope.", Tags: ptags, Access: pmanage},
		func(ctx context.Context, in *PushSitePath) (*api.Body[PushSite], error) {
			id, err := parseID(in.SiteID, "Push site")
			if err != nil {
				return nil, err
			}
			out, err := s.CheckSite(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-push-events", Path: "/workspaces/{workspace}/push/events", Summary: "What the push services and the service workers reported",
		Tags: ptags, Access: pview},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Before time.Time `query:"before" doc:"Events before this time"`
			Limit  int       `query:"limit" minimum:"0" maximum:"200"`
		}) (*api.Body[PushEventList], error) {
			var before *time.Time
			if !in.Before.IsZero() {
				before = &in.Before
			}
			out, err := s.Events(ctx, api.WorkspaceFrom(ctx), before, in.Limit)
			return api.OK(out), err
		})

	// Prompts.
	api.Post(h, api.Op{ID: "create-push-prompt", Path: "/workspaces/{workspace}/push/prompts", Summary: "Make a push prompt, a form type (logged)",
		Tags: ptags, Access: api.Workspace(authz.FormsEdit), Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body PushPromptCreate
		}) (*api.Body[PushPrompt], error) {
			out, err := s.CreatePrompt(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-push-prompt", Path: "/workspaces/{workspace}/push/prompts/{formId}", Summary: "The push prompt designer",
		Tags: ptags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *PushPromptPath) (*api.Body[PushPrompt], error) {
			id, err := parseID(in.FormID, "Push prompt")
			if err != nil {
				return nil, err
			}
			out, err := s.Prompt(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-push-prompt", Path: "/workspaces/{workspace}/push/prompts/{formId}", Summary: "Save the prompt's draft",
		Description: "Readers see it once it's published.", Tags: ptags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			FormID string `path:"formId" format:"uuid"`
			Body   PushPromptSave
		}) (*api.Body[PushPrompt], error) {
			id, err := parseID(in.FormID, "Push prompt")
			if err != nil {
				return nil, err
			}
			out, err := s.SaveDraft(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "publish-push-prompt", Path: "/workspaces/{workspace}/push/prompts/{formId}/publish", Summary: "Publish the prompt (logged)",
		Tags: ptags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *PushPromptPath) (*api.Body[PushPrompt], error) {
			id, err := parseID(in.FormID, "Push prompt")
			if err != nil {
				return nil, err
			}
			out, err := s.Publish(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "discard-push-prompt", Path: "/workspaces/{workspace}/push/prompts/{formId}/discard", Summary: "Throw the draft away",
		Tags: ptags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *PushPromptPath) (*api.Body[PushPrompt], error) {
			id, err := parseID(in.FormID, "Push prompt")
			if err != nil {
				return nil, err
			}
			out, err := s.Discard(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-push-prompt-status", Path: "/workspaces/{workspace}/push/prompts/{formId}/status", Summary: "Pause or resume a prompt (logged)",
		Tags: ptags, Access: api.Workspace(authz.FormsEdit)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			FormID string `path:"formId" format:"uuid"`
			Body   PushPromptStatus
		}) (*api.Body[PushPrompt], error) {
			id, err := parseID(in.FormID, "Push prompt")
			if err != nil {
				return nil, err
			}
			out, err := s.SetStatus(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})

	// Campaigns.
	api.Post(h, api.Op{ID: "create-push-campaign", Path: "/workspaces/{workspace}/push-campaigns", Summary: "Make a push campaign (logged)",
		Tags: ctags, Access: cedit, Status: http.StatusCreated},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body PushCampaignCreate
		}) (*api.Body[PushCampaign], error) {
			out, err := s.CreateCampaign(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-push-campaign", Path: "/workspaces/{workspace}/push-campaigns/{campaignId}", Summary: "The push composer",
		Tags: ctags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *PushCampaignPath) (*api.Body[PushCampaign], error) {
			id, err := parseID(in.CampaignID, "Push campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.Campaign(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-push-campaign", Path: "/workspaces/{workspace}/push-campaigns/{campaignId}", Summary: "Autosave a push campaign (logged)",
		Tags: ctags, Access: cedit},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			CampaignID string `path:"campaignId" format:"uuid"`
			Body       PushCampaignUpdate
		}) (*api.Body[PushCampaign], error) {
			id, err := parseID(in.CampaignID, "Push campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.UpdateCampaign(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "duplicate-push-campaign", Path: "/workspaces/{workspace}/push-campaigns/{campaignId}/duplicate", Summary: "Copy a push campaign as a draft (logged)",
		Tags: ctags, Access: cedit, Status: http.StatusCreated},
		func(ctx context.Context, in *PushCampaignPath) (*api.Body[PushCampaign], error) {
			id, err := parseID(in.CampaignID, "Push campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.DuplicateCampaign(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "send-push-test", Path: "/workspaces/{workspace}/push-campaigns/{campaignId}/test", Summary: "Send the notification to my browser (logged)",
		Description: "The body is this browser's subscription, made with the workspace's public key.", Tags: ctags, Access: api.WorkspaceHuman(authz.CampaignsEdit)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			CampaignID string `path:"campaignId" format:"uuid"`
			UserAgent  string `header:"User-Agent"`
			Body       PushTestInput
		}) (*api.Body[PushTestResult], error) {
			id, err := parseID(in.CampaignID, "Push campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.SendTest(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body, in.UserAgent)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "review-push-campaign", Path: "/workspaces/{workspace}/push-campaigns/{campaignId}/review", Summary: "Review & send: the push checks, schedule and approvals",
		Tags: ctags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *PushCampaignPath) (*api.Body[PushReview], error) {
			id, err := parseID(in.CampaignID, "Push campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.Review(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-push-send-settings", Path: "/workspaces/{workspace}/push-campaigns/{campaignId}/send-settings",
		Summary: "Choose when it goes and what quiet hours do", Tags: ctags, Access: cedit},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			CampaignID string `path:"campaignId" format:"uuid"`
			Body       PushSendSettings
		}) (*api.Body[PushReview], error) {
			id, err := parseID(in.CampaignID, "Push campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.SaveSendSettings(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "send-push-campaign", Path: "/workspaces/{workspace}/push-campaigns/{campaignId}/send", Summary: "Send or schedule a push campaign as reviewed (logged)",
		Description: "Needs campaigns:send and push:send; the workspace's approval rules apply.", Tags: ctags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			CampaignID string `path:"campaignId" format:"uuid"`
			Body       PushSendInput
		}) (*api.Body[PushSendResult], error) {
			id, err := parseID(in.CampaignID, "Push campaign")
			if err != nil {
				return nil, err
			}
			out, err := s.Send(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "unschedule-push-campaign", Path: "/workspaces/{workspace}/push-campaigns/{campaignId}/unschedule",
		Summary: "Take a scheduled push campaign back to a draft (logged)", Tags: ctags, Access: api.Workspace(authz.CampaignsSend)},
		func(ctx context.Context, in *PushCampaignPath) (*struct{}, error) {
			id, err := parseID(in.CampaignID, "Push campaign")
			if err != nil {
				return nil, err
			}
			return nil, s.Unschedule(ctx, api.WorkspaceFrom(ctx), id)
		})
	api.Post(h, api.Op{ID: "cancel-push-campaign", Path: "/workspaces/{workspace}/push-campaigns/{campaignId}/cancel",
		Summary: "Stop a push campaign: what wasn't handed over doesn't go (logged)", Tags: ctags, Access: api.Workspace(authz.CampaignsPause)},
		func(ctx context.Context, in *PushCampaignPath) (*struct{}, error) {
			id, err := parseID(in.CampaignID, "Push campaign")
			if err != nil {
				return nil, err
			}
			return nil, s.Cancel(ctx, api.WorkspaceFrom(ctx), id)
		})
	api.Get(h, api.Op{ID: "get-push-report", Path: "/workspaces/{workspace}/push-campaigns/{campaignId}/report",
		Summary: "The push report: sent, shown, clicks, unsubscribes and expired subscriptions, by browser and site", Tags: ctags,
		Access: api.Workspace(authz.ReportsView)},
		func(ctx context.Context, in *PushCampaignPath) (*api.Body[PushReport], error) {
			id, err := parseID(in.CampaignID, "Push campaign")
			if err != nil {
				return nil, err
			}
			p := api.PrincipalFrom(ctx)
			out, err := s.CampaignReport(ctx, api.WorkspaceFrom(ctx), id, p.Can(authz.CampaignsEdit))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "export-push-report", Path: "/workspaces/{workspace}/push-campaigns/{campaignId}/report.csv",
		Summary: "The push report as CSV, by browser and site (logged)", Tags: ctags, Access: api.Workspace(authz.ReportsView)},
		func(ctx context.Context, in *PushCampaignPath) (*PushCSV, error) {
			id, err := parseID(in.CampaignID, "Push campaign")
			if err != nil {
				return nil, err
			}
			b, name, err := s.ReportCSV(ctx, api.WorkspaceFrom(ctx), id)
			if err != nil {
				return nil, err
			}
			return &PushCSV{ContentType: "text/csv; charset=utf-8", ContentDisposition: `attachment; filename="` + name + `"`, Body: b}, nil
		})

	// The automation step.
	api.Get(h, api.Op{ID: "get-push-step-estimate", Path: "/workspaces/{workspace}/automations/{id}/push-estimate",
		Summary: "Who a flow's push steps reach: its contacts with a linked subscription", Tags: ptags, Access: api.Workspace(authz.AutomationsView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			ID string `path:"id" format:"uuid"`
		}) (*api.Body[PushStepEstimate], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			out, err := s.StepEstimate(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-automation-push-stats", Path: "/workspaces/{workspace}/automations/{id}/push-stats",
		Summary: "A flow's push steps over 30 days: sent, shown, clicks, skipped", Tags: ptags, Access: api.Workspace(authz.AutomationsView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			ID string `path:"id" format:"uuid"`
		}) (*api.Body[PushStepStatsList], error) {
			id, err := parseID(in.ID, "Automation")
			if err != nil {
				return nil, err
			}
			out, err := s.StepStats(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})

	// The contact profile.
	api.Get(h, api.Op{ID: "get-contact-push", Path: "/workspaces/{workspace}/contacts/{contactId}/push", Summary: "A contact's linked browser subscriptions",
		Tags: ptags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *PushContactPath) (*api.Body[PushContactCard], error) {
			id, err := parseID(in.ContactID, "Contact")
			if err != nil {
				return nil, err
			}
			out, err := s.ContactCard(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "remove-contact-push", Path: "/workspaces/{workspace}/contacts/{contactId}/push/{subscriptionId}",
		Summary: "End one of a contact's subscriptions (logged)", Tags: ptags, Access: api.Workspace(authz.SubscribersEdit)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			ContactID      string `path:"contactId" format:"uuid"`
			SubscriptionID string `path:"subscriptionId" format:"uuid"`
		}) (*struct{}, error) {
			cid, err := parseID(in.ContactID, "Contact")
			if err != nil {
				return nil, err
			}
			sid, err := parseID(in.SubscriptionID, "Push subscription")
			if err != nil {
				return nil, err
			}
			return nil, s.RemoveContactSubscription(ctx, api.WorkspaceFrom(ctx), cid, sid)
		})
}
