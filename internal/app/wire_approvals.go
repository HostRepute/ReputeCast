package app

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/templates"
)

// Review & send (Send Readiness) and send approvals.

// readinessService builds Send Readiness: the composer's content and link
// checks, deliverability's blocklists and authentication, routing and the
// audience.
func readinessService(a *App) *readiness.Service {
	prober := linkProber(a)
	cs := &campaigns.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit,
		Templates: &templates.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit},
		Mail:      func() mailer.Sender { return a.Mail },
		URLs:      a.URLs,
		Checker:   &contentcheck.Checker{Blocklist: contentcheck.StoreBlocklist{Q: a.Q}, Prober: prober, Extra: contentExtras(a)},
		// Send Readiness checks and fixes every language version.
		Translations: campaignTranslations(a),
	}
	_, dl := deliverabilityServices(a)
	rs := &readiness.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log, Campaigns: cs, Deliverability: dl, URLs: a.URLs, Segments: a.segments(),
		Languages: languagesService(a), Mailbox: a.mailboxClassifier(),
		// Lookups Deliverability couldn't make are made again during a run.
		DNS: dl.DNS}
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

// newApprovals builds the approvals service with its own Send Readiness.
func newApprovals(a *App) *approvals.Service {
	rs := readinessService(a)
	s := &approvals.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log, Readiness: rs, Campaigns: rs.Campaigns,
		Languages: rs.Languages,
		Sending:   func() sending.Starter { return a.Sending },
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
		river.AddWorker(a.JobReg.Workers, &approvals.TickWorker{S: newApprovals(a)})
		a.JobReg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return approvals.TickArgs{}, nil }, false)
	})
}
