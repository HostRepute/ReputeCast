package app

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/growth"
	"github.com/hostrepute/reputecast/internal/integrations"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/leads"
	"github.com/hostrepute/reputecast/internal/leads/meta/metatest"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/store"
)

// Lead ads from Meta (M66, G-41): the connection page and its dialogs, the
// callback Meta posts leads to, the jobs that apply, poll and import them,
// the Lead ads group of Settings › Integrations, the "Lead ads · {form}"
// growth source and the confirmation that starts "Submitted a lead ad
// form" for double opt-in leads.

var (
	leadsMu       sync.Mutex
	leadsServices = map[*App]*leads.Service{}
	leadsFakes    = map[*App]*metatest.Fake{}
)

// LeadAdsFake is the in-process Graph API the demo's connection (and
// tests) answer from: nothing reaches Meta.
func (a *App) LeadAdsFake() *metatest.Fake {
	leadsMu.Lock()
	defer leadsMu.Unlock()
	if f, ok := leadsFakes[a]; ok {
		return f
	}
	f := demo.LeadAdsFake()
	leadsFakes[a] = f
	return f
}

// LeadAds is the lead ads domain (tests set its clock).
func (a *App) LeadAds() *leads.Service {
	leadsMu.Lock()
	if s, ok := leadsServices[a]; ok {
		leadsMu.Unlock()
		return s
	}
	leadsMu.Unlock()
	cfg := a.Cfg
	if cfg == nil { // the OpenAPI command registers operations on an App without config
		d := config.Defaults()
		cfg = &d
	}
	// Meta's Graph API is reached through the SSRF guard. Tests never
	// reach Meta: every call goes to the in-process Graph API.
	client := netguard.Policy{AllowPrivate: cfg.Providers.AllowPrivateNetworks}.Client(20 * time.Second)
	if cfg.Env == "test" {
		client = a.LeadAdsFake().Client()
	}
	s := &leads.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log, URLs: a.URLs, Audience: a.audience(),
		Jobs: func() *jobs.Client { return a.Jobs },
		HTTP: client,
		Demo: a.LeadAdsFake,
		Kick: func(ctx context.Context) { newAutomations(a).Kick(ctx) },
		Reachable: func(ctx context.Context) (string, bool) {
			r := a.integrations().Reachability(ctx)
			return r.URL, r.Reachable
		},
	}
	leadsMu.Lock()
	defer leadsMu.Unlock()
	if prev, ok := leadsServices[a]; ok {
		return prev
	}
	leadsServices[a] = s
	return s
}

// leadsCatalog is the Lead ads group of Settings › Integrations: Meta Lead
// Ads' card with its connection.
func leadsCatalog(a *App) integrations.GroupSource {
	return func(ctx context.Context, ws *store.Workspace) ([]integrations.IntegrationEntry, error) {
		e := integrations.IntegrationEntry{Key: "meta_lead_ads", Status: "not_connected", Reasons: []integrations.IntegrationReason{}}
		sum, ok, err := a.LeadAds().Summary(ctx, ws)
		if err != nil {
			return nil, err
		}
		if ok {
			e.ConnectionID = sum.ID.String()
			e.Status = "connected"
			if sum.Problem != "" {
				e.Status = "attention"
			}
			e.Facts.Leads = &integrations.IntegrationLeadFacts{Name: sum.Name, Pages: sum.Pages, Leads30d: sum.Leads30d, Polling: sum.Polling, Problem: sum.Problem}
		}
		return []integrations.IntegrationEntry{e}, nil
	}
}

// leadsHook starts "Submitted a lead ad form" when a double opt-in lead
// confirms.
type leadsHook struct{ s *leads.Service }

func (leadsHook) DoubleOptIn(context.Context, *store.Workspace, forms.Submission) bool { return false }

func (leadsHook) Accepted(context.Context, pgx.Tx, *store.Workspace, forms.SignupEvent) error {
	return nil
}

func (h leadsHook) Confirmed(ctx context.Context, tx pgx.Tx, ws *store.Workspace, e forms.ConfirmEvent) error {
	return h.s.Confirmed(ctx, tx, ws, e.ContactID, e.Lists, e.At)
}

func init() {
	integrationLeads = leadsCatalog
	// Joins through a lead form count under "Lead ads · {form}", listed
	// after integrations and before people adding contacts, as
	// GrowthSources draws it.
	growth.RegisterKind(growth.Kind{Key: "lead_ad", Order: 35})
	growth.RegisterAttributor(func(contactSource, consentSource string) (string, string, bool) {
		for _, src := range []string{consentSource, contactSource} {
			if id, ok := strings.CutPrefix(src, leads.ConsentSourcePrefix); ok {
				return "lead_ad", id, true
			}
		}
		return "", "", false
	})
	growth.RegisterNamer("lead_ad", func(ctx context.Context, q *store.Queries, ws *store.Workspace) (map[string]string, error) {
		rows, err := q.LeadFormNames(ctx, ws.ID)
		if err != nil {
			return nil, err
		}
		out := make(map[string]string, len(rows))
		for _, r := range rows {
			out[r.ID.String()] = r.Name
		}
		return out, nil
	})
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		leads.Register(h, a.LeadAds())
	})
	// Meta's callback: verified, stored as it came, answered at once.
	ingestRegistrars = append(ingestRegistrars, func(a *App, r chi.Router) {
		a.LeadAds().Routes(r)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		leads.RegisterJobs(a.JobReg, a.LeadAds())
	})
	signupHookRegistrars = append(signupHookRegistrars, func(a *App) forms.SignupHook {
		return leadsHook{s: a.LeadAds()}
	})
}
