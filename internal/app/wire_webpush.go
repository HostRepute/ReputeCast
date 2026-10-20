package app

import (
	"context"
	"net/http"
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
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/publicpages"
	"github.com/hostrepute/reputecast/internal/push"
	"github.com/hostrepute/reputecast/internal/push/webpush/webpushtest"
	"github.com/hostrepute/reputecast/internal/server"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
	"github.com/hostrepute/reputecast/web"
)

// Web push (design expansion phase 8, X22): VAPID keys, sites with the
// service worker, subscriptions, the prompt designer, push campaigns, the
// Send push step, and the public scripts, endpoints and click link.
//
// This file sorts after wire_texting.go: it chains the campaigns list's and
// the approvals' channel hooks onto the SMS ones.

var (
	pushMu       sync.Mutex
	pushServices = map[*App]*push.Service{}
	pushFakes    = map[*App]*webpushtest.Service{}
)

// PushFake is the in-process push service the demo's subscriptions (and
// tests) are sent to: nothing reaches FCM, Mozilla, Apple or Windows.
func (a *App) PushFake() *webpushtest.Service {
	pushMu.Lock()
	defer pushMu.Unlock()
	if f, ok := pushFakes[a]; ok {
		return f
	}
	f := webpushtest.New()
	pushFakes[a] = f
	return f
}

// Push is the web push domain.
func (a *App) Push() *push.Service {
	pushMu.Lock()
	if s, ok := pushServices[a]; ok {
		pushMu.Unlock()
		return s
	}
	pushMu.Unlock()
	cfg := a.Cfg
	if cfg == nil { // the OpenAPI command registers operations on an App without config
		d := config.Defaults()
		cfg = &d
	}
	// Push services and sites are reached through the SSRF guard. Tests
	// never reach a push service: everything goes to the in-process one.
	client := netguard.Policy{AllowPrivate: cfg.Providers.AllowPrivateNetworks}.Client(15 * time.Second)
	if cfg.Env == "test" {
		client = a.PushFake().Client()
	}
	s := &push.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log, URLs: a.URLs, Links: a.Links,
		Jobs:     func() *jobs.Client { return a.Jobs },
		HTTP:     client,
		Demo:     a.PushFake,
		Segments: a.segments(),
		// Approvals: the workspace's rules apply to push campaigns.
		Approvals: approvalChannelsAdapter{a},
		Changed: func(ctx context.Context, ws *store.Workspace, id uuid.UUID) {
			newApprovals(a).CampaignChanged(ctx, ws, id)
		},
		Website: func(ctx context.Context, ws *store.Workspace) string {
			set, err := a.formsService().LoadSettings(ctx, ws)
			if err != nil || set.CustomDomain() == "" {
				return ""
			}
			if on, err := websiteService(a).IsOn(ctx, ws); err != nil || !on {
				return ""
			}
			// Browsers allow push only on https sites.
			return "https://" + set.CustomDomain()
		},
		Member: func(ctx context.Context, ws *store.Workspace, r *http.Request) uuid.UUID {
			c, err := r.Cookie(website.SessionCookie)
			if err != nil || c.Value == "" {
				return uuid.Nil
			}
			sess, err := websiteService(a).SessionFor(ctx, ws, c.Value)
			if err != nil || sess == nil {
				return uuid.Nil
			}
			return sess.ContactID
		},
		Logo: func(ctx context.Context, ws *store.Workspace) string {
			set, err := a.formsService().LoadSettings(ctx, ws)
			if err != nil || set.Branding.Logo == nil {
				return ""
			}
			return a.formsService().Links(ws, set).Logo(set.Branding.Logo.Hash)
		},
		Brand: func(ctx context.Context, ws *store.Workspace) (string, string) {
			set, err := a.formsService().LoadSettings(ctx, ws)
			if err != nil {
				return "", ws.Name
			}
			return set.Branding.Palette.Accent, set.Branding.PublisherName
		},
	}
	pushMu.Lock()
	defer pushMu.Unlock()
	if prev, ok := pushServices[a]; ok {
		return prev
	}
	pushServices[a] = s
	push.Track(s)
	return s
}

// channelRows fills in the campaigns list's rows of several channels.
type channelRows []campaigns.ChannelRows

func (rs channelRows) FillChannelRows(ctx context.Context, ws *store.Workspace, items []campaigns.CampaignListItem) {
	for _, r := range rs {
		if r != nil {
			r.FillChannelRows(ctx, ws, items)
		}
	}
}

// pushPages gives the public pages the push prompt and the preference
// center's "Turn off".
type pushPages struct{ a *App }

func (p pushPages) SiteScripts(ctx context.Context, ws *store.Workspace, host string) []string {
	return p.a.Push().SiteScripts(ctx, ws, host)
}

func (p pushPages) TurnOff(ctx context.Context, ws *store.Workspace, host string, l i18n.Lang) *publicpages.PushTurnOff {
	label, done, none, ok := p.a.Push().TurnOff(ctx, ws, host, l)
	if !ok {
		return nil
	}
	return &publicpages.PushTurnOff{Label: label, Done: done, None: none}
}

func init() {
	prevRows := campaignChannelRows
	campaignChannelRows = func(a *App) campaigns.ChannelRows {
		return channelRows{prevRows(a), a.Push()}
	}
	prevApprovals := approvalChannels
	approvalChannels = func(a *App) map[string]approvals.ChannelCampaigns {
		out := map[string]approvals.ChannelCampaigns{}
		for k, v := range prevApprovals(a) {
			out[k] = v
		}
		out["push"] = a.Push()
		return out
	}
	prevPages := publicPush
	publicPush = func(a *App) publicpages.PushPages {
		if p := prevPages(a); p != nil {
			return p
		}
		return pushPages{a}
	}
	startHooks = append(startHooks, func(a *App) error {
		s := a.Push()
		automations.RegisterChannelStep(automations.NodePush, push.Step{S: s})
		channels.Register(channels.Info{Channel: channels.Push, SetUp: s.Enabled})
		return nil
	})
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		push.Register(h, a.Push())
	})
	publicRegistrars = append(publicRegistrars, func(a *App, r chi.Router) {
		s := a.Push()
		s.PublicRoutes(r, server.NewSPA(web.Dist()))
		r.Route("/p/{ws}/push", func(r chi.Router) {
			s.PageRoutes(r, func(req *http.Request) (*store.Workspace, bool) {
				w, err := a.Q.GetWorkspaceBySlug(req.Context(), chi.URLParam(req, "ws"))
				if err != nil {
					return nil, false
				}
				return &w, true
			})
		})
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		push.RegisterJobs(a.JobReg, a.Push())
	})
}
