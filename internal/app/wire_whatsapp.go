package app

import (
	"context"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/channels"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/whatsapp"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta/metatest"
)

// WhatsApp (design expansion phase 8, X22, optional): Meta's WhatsApp Cloud
// API, templates, WhatsApp consent, the customer service window, WhatsApp
// campaigns, the Send WhatsApp step and Meta's webhooks.
//
// This file sorts after wire_webpush.go: it chains the campaigns list's and
// the approvals' channel hooks onto the SMS and push ones.

var (
	waMu       sync.Mutex
	waServices = map[*App]*whatsapp.Service{}
	waFakes    = map[*App]*metatest.Fake{}
)

// WhatsAppFake is the in-process Graph API the demo's account (and tests)
// answer from: nothing reaches Meta.
func (a *App) WhatsAppFake() *metatest.Fake {
	waMu.Lock()
	defer waMu.Unlock()
	if f, ok := waFakes[a]; ok {
		return f
	}
	f := metatest.New(demo.WhatsAppBusinessID, demo.WhatsAppPhoneID, demo.WhatsAppToken)
	f.SetTemplates(demo.WhatsAppTemplates())
	waFakes[a] = f
	return f
}

// WhatsApp is the WhatsApp domain.
func (a *App) WhatsApp() *whatsapp.Service {
	waMu.Lock()
	if s, ok := waServices[a]; ok {
		waMu.Unlock()
		return s
	}
	waMu.Unlock()
	cfg := a.Cfg
	if cfg == nil { // the OpenAPI command registers operations on an App without config
		d := config.Defaults()
		cfg = &d
	}
	// Meta's Graph API is reached through the SSRF guard. Tests never
	// reach Meta: every call goes to the in-process Graph API.
	client := netguard.Policy{AllowPrivate: cfg.Providers.AllowPrivateNetworks}.Client(20 * time.Second)
	if cfg.Env == "test" {
		client = a.WhatsAppFake().Client()
	}
	s := &whatsapp.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log, URLs: a.URLs,
		Jobs:     func() *jobs.Client { return a.Jobs },
		HTTP:     client,
		Demo:     a.WhatsAppFake,
		Segments: a.segments(),
		// Approvals: the workspace's rules (and the cost rule) apply.
		Approvals: approvalChannelsAdapter{a},
		Changed: func(ctx context.Context, ws *store.Workspace, id uuid.UUID) {
			newApprovals(a).CampaignChanged(ctx, ws, id)
		},
	}
	waMu.Lock()
	defer waMu.Unlock()
	if prev, ok := waServices[a]; ok {
		return prev
	}
	waServices[a] = s
	return s
}

func init() {
	prevRows := campaignChannelRows
	campaignChannelRows = func(a *App) campaigns.ChannelRows {
		return channelRows{prevRows(a), a.WhatsApp()}
	}
	prevApprovals := approvalChannels
	approvalChannels = func(a *App) map[string]approvals.ChannelCampaigns {
		out := map[string]approvals.ChannelCampaigns{}
		for k, v := range prevApprovals(a) {
			out[k] = v
		}
		out["whatsapp"] = a.WhatsApp()
		return out
	}
	startHooks = append(startHooks, func(a *App) error {
		s := a.WhatsApp()
		automations.RegisterChannelStep(automations.NodeWhatsApp, whatsapp.Step{S: s})
		channels.Register(channels.Info{Channel: channels.WhatsApp, SetUp: s.Connected})
		return nil
	})
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		whatsapp.Register(h, a.WhatsApp())
	})
	ingestRegistrars = append(ingestRegistrars, func(a *App, r chi.Router) {
		a.WhatsApp().HookRoutes(r)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		whatsapp.RegisterJobs(a.JobReg, a.WhatsApp())
	})
}
