package app

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/channels"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/pagelabels"
	"github.com/hostrepute/reputecast/internal/publicpages"
	"github.com/hostrepute/reputecast/internal/server"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/sms/twilio/twiliotest"
	"github.com/hostrepute/reputecast/internal/sms/vonage/vonagetest"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/texting"
	"github.com/hostrepute/reputecast/web"

	// The SMS adapters (design expansion phase 8, X21).
	_ "github.com/hostrepute/reputecast/internal/sms/awseum"
	_ "github.com/hostrepute/reputecast/internal/sms/plivo"
	_ "github.com/hostrepute/reputecast/internal/sms/telnyx"
	_ "github.com/hostrepute/reputecast/internal/sms/twilio"
	_ "github.com/hostrepute/reputecast/internal/sms/vonage"
)

// SMS (design expansion phase 8, X21): providers and senders, routing and
// rules, SMS consent and suppressions, SMS campaigns, the Send SMS step,
// provider webhooks, short links and the text message terms page.

var (
	textingMu       sync.Mutex
	textingServices = map[*App]*texting.Service{}
	smsDemoFakes    = map[*App]*smsFakes{}
)

// smsFakes answer the demo's sample providers in process: they never reach
// Twilio or Vonage.
type smsFakes struct {
	twilio *twiliotest.Fake
	vonage *vonagetest.Fake
}

func (a *App) smsDemo() *smsFakes {
	textingMu.Lock()
	defer textingMu.Unlock()
	if f, ok := smsDemoFakes[a]; ok {
		return f
	}
	f := &smsFakes{twilio: twiliotest.New(), vonage: vonagetest.New()}
	smsDemoFakes[a] = f
	return f
}

// smsDemoClient serves a demo provider's API from its fake.
func (a *App) smsDemoClient(adapter string) (*http.Client, string) {
	f := a.smsDemo()
	switch adapter {
	case "twilio":
		return f.twilio.Client(), "https://twilio.demo.invalid"
	case "vonage":
		return f.vonage.Client(), "https://vonage.demo.invalid"
	}
	// Other adapters have no demo account: their calls fail as unreachable.
	return &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, sms.ErrUnavailable })}, "https://demo.invalid"
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// smsDemoReceipt is the receipt a demo provider's fake sends for a text.
func smsDemoReceipt(adapter string, cfg sms.Config, hookURL, providerMessageID, to, from string, at time.Time) (*http.Request, []byte, bool) {
	switch adapter {
	case "twilio":
		r, b := twiliotest.Receipt(cfg.WebhookSecret, hookURL, providerMessageID, to, from, "delivered", "")
		return r, b, true
	case "vonage":
		r, b := vonagetest.Receipt(cfg.Value("signature_secret"), hookURL, providerMessageID, to, from, "delivered", "0", at)
		return r, b, true
	}
	return nil, nil, false
}

// approvalChannelsAdapter reaches the approvals service at call time: it's
// built with the SMS service in its Channels.
type approvalChannelsAdapter struct{ a *App }

func (x approvalChannelsAdapter) ChannelGate(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (*store.ApprovalRequest, error) {
	return newApprovals(x.a).ChannelGate(ctx, ws, p, id)
}

func (x approvalChannelsAdapter) ChannelReview(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (approvals.ReviewApproval, error) {
	return newApprovals(x.a).ChannelReview(ctx, ws, p, id)
}

func (a *App) texting() *texting.Service {
	textingMu.Lock()
	if s, ok := textingServices[a]; ok {
		textingMu.Unlock()
		return s
	}
	textingMu.Unlock()
	cfg := a.Cfg
	if cfg == nil { // the OpenAPI command registers operations on an App without config
		d := config.Defaults()
		cfg = &d
	}
	// Provider calls come from workspace admins' credentials and the
	// provider's own hosts: they go through the SSRF guard.
	guard := netguard.Policy{AllowPrivate: cfg.Providers.AllowPrivateNetworks}
	resolver := dnscheck.NewResolver(dnscheck.Options{Servers: cfg.DNS.Servers, Timeout: cfg.DNS.Timeout})
	if cfg.IsDevelopment() {
		// The demo's short-link domain (go.acme.io) answers from the demo zone.
		resolver = demo.SMSDNSOverlay(resolver, a.URLs)
	}
	s := &texting.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log, URLs: a.URLs,
		Jobs:     func() *jobs.Client { return a.Jobs },
		HTTP:     guard.Client(30 * time.Second),
		DNS:      resolver,
		Segments: a.segments(),
		Demo:     a.smsDemoClient,
		// Approvals: the workspace's rules apply to SMS campaigns.
		Approvals: approvalChannelsAdapter{a},
		Changed: func(ctx context.Context, ws *store.Workspace, id uuid.UUID) {
			newApprovals(a).CampaignChanged(ctx, ws, id)
		},
		PageBase: func(ctx context.Context, ws *store.Workspace) string {
			set, err := a.formsService().LoadSettings(ctx, ws)
			if err != nil || set.CustomDomain() == "" {
				return ""
			}
			scheme := "https://"
			if strings.HasPrefix(a.URLs.Base(), "http://") {
				scheme = "http://"
			}
			return scheme + set.CustomDomain()
		},
		DemoReceipt: smsDemoReceipt,
	}
	textingMu.Lock()
	defer textingMu.Unlock()
	if prev, ok := textingServices[a]; ok {
		return prev
	}
	textingServices[a] = s
	return s
}

// textingTimeline adds texts to the contact timeline after poll answers.
type textingTimeline struct {
	first, second audience.ActivitySource
}

func (t textingTimeline) ContactActivity(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, before *time.Time, beforeID string, limit int) ([]audience.ContactActivityItem, int64, error) {
	a, na, err := t.first.ContactActivity(ctx, ws, contactID, before, beforeID, limit)
	if err != nil {
		return nil, 0, err
	}
	b, nb, err := t.second.ContactActivity(ctx, ws, contactID, before, beforeID, limit)
	if err != nil {
		return nil, 0, err
	}
	out := make([]audience.ContactActivityItem, 0, len(a)+len(b))
	i, j := 0, 0
	for (i < len(a) || j < len(b)) && len(out) < limit {
		if j >= len(b) || (i < len(a) && (a[i].At.After(b[j].At) || (a[i].At.Equal(b[j].At) && a[i].ID > b[j].ID))) {
			out = append(out, a[i])
			i++
		} else {
			out = append(out, b[j])
			j++
		}
	}
	return out, na + nb, nil
}

// textTerms writes the public text message terms page in the pages'
// language (publicpages.TextTerms).
type textTerms struct{ a *App }

func (t textTerms) TextTermsPage(ctx context.Context, ws *store.Workspace) (publicpages.TermsPage, bool) {
	code := "en"
	if set, err := t.a.formsService().LoadSettings(ctx, ws); err == nil {
		code = set.CopyLanguage()
	}
	p, ok := t.a.texting().PublicTermsPage(ctx, ws, pagelabels.Lang(code), code)
	if !ok {
		return publicpages.TermsPage{}, false
	}
	out := publicpages.TermsPage{Lang: p.Lang, Title: p.Title, Version: p.Version, Intro: p.Intro, ClosingHeading: p.ClosingHeading, ClosingText: p.ClosingText,
		ManageLabel: p.ManageLabel, BackLabel: p.BackLabel, BackURL: p.BackURL, Footer: p.Footer}
	for _, sec := range p.Sections {
		ps := publicpages.TermsSection{Heading: sec.Heading, Text: sec.Text}
		for _, l := range sec.Links {
			ps.Links = append(ps.Links, publicpages.TermsLink{Text: l.Text, Href: l.Href, NoWrap: l.NoWrap})
		}
		out.Sections = append(out.Sections, ps)
	}
	return out, true
}

func init() {
	publicTextTerms = func(a *App) publicpages.TextTerms { return textTerms{a} }
	prevAnswers := audienceAnswers
	audienceAnswers = func(a *App) audience.ActivitySource {
		if p := prevAnswers(a); p != nil {
			return textingTimeline{first: p, second: a.texting()}
		}
		return a.texting()
	}
	approvalChannels = func(a *App) map[string]approvals.ChannelCampaigns {
		return map[string]approvals.ChannelCampaigns{"sms": a.texting()}
	}
	campaignChannelRows = func(a *App) campaigns.ChannelRows { return a.texting() }
	startHooks = append(startHooks, func(a *App) error {
		s := a.texting()
		automations.RegisterChannelStep(automations.NodeSMS, texting.SmsStep{S: s})
		channels.Register(channels.Info{Channel: channels.Email, SetUp: func(ctx context.Context, ws uuid.UUID) (bool, error) {
			var n int64
			err := a.Pool.QueryRow(ctx, `SELECT count(*) FROM providers WHERE workspace_id = $1 AND disconnected_at IS NULL`, ws).Scan(&n)
			return n > 0, err
		}})
		channels.Register(channels.Info{Channel: channels.SMS, SetUp: func(ctx context.Context, ws uuid.UUID) (bool, error) {
			var n int64
			err := a.Pool.QueryRow(ctx, `SELECT count(*) FROM sms_providers WHERE workspace_id = $1 AND removed_at IS NULL`, ws).Scan(&n)
			return n > 0, err
		}})
		return nil
	})
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		texting.Register(h, a.texting())
	})
	ingestRegistrars = append(ingestRegistrars, func(a *App, r chi.Router) {
		a.texting().HookRoutes(r)
	})
	publicRegistrars = append(publicRegistrars, func(a *App, r chi.Router) {
		a.texting().LinkRoutes(r, server.NewSPA(web.Dist()))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		texting.RegisterJobs(a.JobReg, a.texting())
	})
}
