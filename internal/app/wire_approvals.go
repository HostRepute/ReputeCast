package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/sending/engine"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// Review & send (Send Readiness) and send approvals.

// readinessService builds Send Readiness: the composer's content and link
// checks, deliverability's blocklists and authentication, routing and the
// audience.
func readinessService(a *App) *readiness.Service {
	prober := linkProber(a)
	links := linkBlocklist(a, 30*time.Second)
	cs := &campaigns.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit,
		Templates: &templates.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit},
		Mail:      func() mailer.Sender { return a.Mail },
		URLs:      a.URLs,
		Checker:   &contentcheck.Checker{Blocklist: links, Prober: prober, Extra: contentExtras(a)},
		// Send Readiness checks and fixes every language version.
		Translations: campaignTranslations(a),
	}
	_, dl := deliverabilityServices(a)
	rs := &readiness.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log, Campaigns: cs, Deliverability: dl, URLs: a.URLs, Segments: a.segments(),
		Languages: languagesService(a), Mailbox: a.mailboxClassifier(),
		// Lookups Deliverability couldn't make are made again during a run;
		// the return-path and tracking domains are looked up as link domains are.
		DNS: dl.DNS, Links: links,
		// Runs wait in the job queue; a worker runs them (SR-8).
		Jobs: func() readiness.JobInserter { return lazyJobs{a} },
		// Estimates come from the sending engine's own rules (SR-5, C4).
		Estimator: engineEstimator{a}}
	if acc := readinessAccessibility(a); acc != nil {
		rs.Accessibility = acc
	}
	if c := readinessConditional(a); c != nil {
		rs.Conditional = c
	}
	if v := readinessVerification(a); v != nil {
		rs.Verification = v
	}
	// Grow's Referral and Recommendations blocks (wire_grow.go).
	rs.Grow = growReadiness{a: a}
	if w := readinessWebsite(a); w != nil {
		rs.Website = w
	}
	if sp := readinessSponsor(a); sp != nil {
		rs.Sponsor = sp
	}
	if uo := readinessUpgradeOffers(a); uo != nil {
		rs.UpgradeOffers = uo
	}
	return rs
}

// engineEstimator asks the app's sending engine for a send's estimate
// (built on first use: the engine's own wiring includes Send Readiness).
type engineEstimator struct{ a *App }

// Estimate implements readiness.Estimator.
func (e engineEstimator) Estimate(ctx context.Context, campaignID uuid.UUID, o engine.EstimateOptions) (engine.Estimate, error) {
	return e.a.SendingEngine().Estimate(ctx, campaignID, o)
}

// linkBlocklist looks link domains up live on Spamhaus DBL and SURBL and
// keeps the answers for an hour (SR-1), for the composer's inline checks
// (a short budget: it checks as people type) and Send Readiness (a longer
// one: it runs in the background). The demo and the tests answer from the
// demo's zone, so nothing reaches the real lists.
func linkBlocklist(a *App, budget time.Duration) *contentcheck.LiveBlocklist {
	cfg := a.Cfg
	if cfg == nil {
		d := config.Defaults()
		cfg = &d
	}
	resolver := dnscheck.NewResolver(dnscheck.Options{Servers: cfg.DNS.Servers, Timeout: cfg.DNS.Timeout})
	if cfg.IsDevelopment() || cfg.Env == "test" {
		resolver = demo.ReadinessLinkDNS(resolver)
	}
	return &contentcheck.LiveBlocklist{Q: a.Q, DNS: resolver, Budget: budget}
}

// readinessUpgradeOffers finds Upgrade blocks whose offer ended; set from
// wire_offers.go. Nil leaves the check out.
var readinessUpgradeOffers = func(*App) readiness.UpgradeOfferChecker { return nil }

// readinessSponsor gives the Sponsor block's checks and fixes; set from
// wire_sponsors.go. Nil leaves them out.
var readinessSponsor = func(*App) readiness.SponsorChecker { return nil }

// readinessWebsite gives the web checks of campaigns that publish to the
// website; set from wire_website.go. Nil leaves them out.
var readinessWebsite = func(*App) readiness.WebsiteChecker { return nil }

// approvalsWebsite plugs the website's post approvals into the approvals
// service; set from wire_website.go.
var approvalsWebsite = func(*App, *approvals.Service) {}

// readinessAccessibility runs the Accessibility group of Send Readiness; the
// accessibility domain replaces it from its wire file. Nil leaves it out.
var readinessAccessibility = func(*App) readiness.AccessibilityChecker { return nil }

// readinessConditional runs the Conditional content group and the poll
// checks; set from wire_visibility.go. Nil leaves them out.
var readinessConditional = func(*App) readiness.ConditionalChecker { return nil }

// readinessVerification gives Audience quality the address checks; the
// verification domain replaces it from its wire file. Nil leaves them out.
var readinessVerification = func(*App) readiness.VerificationChecker { return nil }

// approvalsCanaryPlanner tells Review & send the sending engine's canary
// plan (C1: engine.CanaryPlanFor); the engine's wire file sets it. Nil
// asks the sending starter, when it tells one.
var approvalsCanaryPlanner = func(*App) sending.CanaryPlanner { return nil }

// newApprovals builds the approvals service with its own Send Readiness.
func newApprovals(a *App) *approvals.Service {
	rs := readinessService(a)
	s := &approvals.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log, Readiness: rs, Campaigns: rs.Campaigns,
		Languages:     rs.Languages,
		Sending:       func() sending.Starter { return a.Sending },
		CanaryPlanner: approvalsCanaryPlanner(a),
		// Publishing an approved automation: the gate already passed.
		Automations: &automations.Service{
			Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log,
			Jobs: func() automations.Inserter {
				if a.Jobs == nil {
					return nil
				}
				return a.Jobs
			},
			Mailer:        automationMailer(a),
			Gate:          automations.NoApproval{},
			Accessibility: automationAccessibility(a),
		},
	}
	approvalsWebsite(a, s)
	approvalsPages(a, s)
	// Fixes and send settings re-check open approvals; so do composer saves.
	rs.AfterChange = s.CampaignChanged
	rs.Campaigns.AfterChange = s.CampaignChanged
	// Campaigns on other channels (SMS) describe themselves for approvals.
	s.Channels = approvalChannels(a)
	return s
}

// approvalChannels describes campaigns on channels other than email for
// approvals; the channels' wire files set it (wire_texting.go).
var approvalChannels = func(*App) map[string]approvals.ChannelCampaigns { return nil }

// CampaignChangeHook re-checks a campaign's open approval after the composer
// saved it (the composer's campaigns.Service.AfterChange).
func CampaignChangeHook(a *App) campaigns.ChangeHook {
	return func(ctx context.Context, ws *store.Workspace, id uuid.UUID) {
		newApprovals(a).CampaignChanged(ctx, ws, id)
	}
}

// SendRecheck is the recheck the sending engine runs as a send starts
// (its sending.Readiness, via readinessRecheck): Send Readiness again, and
// the approval must still cover the campaign. It records nothing; the
// engine holds the send when the verdict is Blocked.
func SendRecheck(a *App) func(ctx context.Context, campaignID uuid.UUID) (approvals.SendRecheck, error) {
	return newApprovals(a).SendRecheck
}

// ApprovalsTickAt runs Review & send's minute job as if it were now: tests
// move the clock to a scheduled send's recheck, or past a held send's
// expiry.
func ApprovalsTickAt(a *App, now time.Time) error {
	s := newApprovals(a)
	s.Now = func() time.Time { return now }
	return s.Tick(context.Background())
}

// HeldSendCheck is the send-time check the minute job runs for scheduled
// and held sends: it holds a send that turns Blocked or lost its approval
// (and tells people), and returns an error wrapping sending.ErrNotReady.
func HeldSendCheck(a *App) func(ctx context.Context, workspaceID, campaignID uuid.UUID) error {
	return newApprovals(a).CheckSendTime
}

func init() {
	automationGate = func(a *App) automations.PublishGate { return approvals.AutomationGate{S: newApprovals(a)} }
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		s := newApprovals(a)
		readiness.Register(h, s.Readiness)
		approvals.Register(h, s)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		river.AddWorker(a.JobReg.Workers, &readiness.RunWorker{S: newApprovals(a).Readiness})
		river.AddWorker(a.JobReg.Workers, &approvals.TickWorker{S: newApprovals(a)})
		a.JobReg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return approvals.TickArgs{}, nil }, false)
	})
}

// ReadinessQueue works the Send Readiness runs waiting in the job queue, in
// this process: tests (whose app runs no workers) call it after asking for
// a run. It returns how many it worked.
func ReadinessQueue(ctx context.Context, a *App) (int, error) {
	res, err := a.Jobs.JobList(ctx, river.NewJobListParams().Kinds(readiness.RunArgs{}.Kind()).States(rivertype.JobStateAvailable).First(100))
	if err != nil {
		return 0, err
	}
	rs := newApprovals(a).Readiness
	n := 0
	for _, job := range res.Jobs {
		var args readiness.RunArgs
		if err := json.Unmarshal(job.EncodedArgs, &args); err != nil {
			return n, err
		}
		if _, err := a.Jobs.JobDelete(ctx, job.ID); err != nil {
			return n, err
		}
		if err := rs.Work(ctx, args); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
