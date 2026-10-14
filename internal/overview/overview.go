// Package overview builds the workspace Overview (spec 7.1): sender health,
// the next scheduled campaign, provider cards, subscriber growth and tiers,
// and recent campaigns.
//
// Each card has its own operation so one failing source never blanks the
// page (OverviewStates board), and the numbers come from the domains that
// own them so the Overview always agrees with the screens behind it.
package overview

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/deliverability"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

// Audience supplies subscriber growth and engagement tiers (the audience
// domain's contacts, lists, tiers and sunset policy).
type Audience interface {
	Audience(ctx context.Context, ws *store.Workspace, days int) (OverviewAudience, error)
}

type Service struct {
	Pool           *pgxpool.Pool
	Q              *store.Queries
	Deliverability *deliverability.Service
	Providers      *manage.Service
	Campaigns      *campaigns.Service
	Audience       Audience
	Now            func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// OverviewHealthAction is the link at the end of a reason.
type OverviewHealthAction struct {
	Label string `json:"label" doc:"request delisting"`
	// Kind says where it goes: request delisting of Asset, Deliverability,
	// Sending providers, the warm-up plan or Settings › Integrations.
	Kind  string `json:"kind" enum:"delisting,deliverability,providers,warmup,integrations"`
	Asset string `json:"asset,omitempty"`
}

// OverviewHealthReason is one line under the score.
type OverviewHealthReason struct {
	Key    string                `json:"key" enum:"blocklist_domain,auth,rates,blocklist_ip,warmup,provider_status,checks"`
	Tone   string                `json:"tone" enum:"ok,warn,bad"`
	Text   string                `json:"text"`
	Action *OverviewHealthAction `json:"action,omitempty"`
}

// OverviewSenderHealth is "Sender health": a 0–100 score with a word and
// the reasons behind it, from the same checks the Deliverability screen shows.
type OverviewSenderHealth struct {
	// State is pending until a sending domain is set up (first run).
	State string `json:"state" enum:"ready,pending"`
	Score int    `json:"score"`
	Label string `json:"label" enum:"Good,Fair,Poor,"`
	Tone  string `json:"tone" enum:"ok,warn,bad,neutral"`
	// Domain is the main sending domain ("news.acme.io").
	Domain    string                 `json:"domain"`
	Sources   int                    `json:"sources" doc:"Blocklists checked"`
	Via       string                 `json:"via" enum:"hostrepute,builtin,none"`
	CheckedAt *time.Time             `json:"checkedAt,omitempty"`
	Reasons   []OverviewHealthReason `json:"reasons"`
}

// OverviewReadinessIssue is a warning or block of the latest readiness run.
type OverviewReadinessIssue struct {
	Status string `json:"status" enum:"warn,block"`
	Title  string `json:"title"`
}

// OverviewReadiness is the campaign's latest Send Readiness verdict.
type OverviewReadiness struct {
	Verdict string                   `json:"verdict" enum:"ready,warnings,blocked"`
	Issues  []OverviewReadinessIssue `json:"issues"`
	RanAt   time.Time                `json:"ranAt"`
}

// OverviewRouteShare is a sending provider's share of the planned send.
type OverviewRouteShare struct {
	Short   string  `json:"short"`
	Adapter string  `json:"adapter"`
	Color   string  `json:"color" enum:"ses,postal,postmark,other"`
	Percent float64 `json:"percent"`
	Planned int64   `json:"planned"`
}

// OverviewScheduledCampaign is the next campaign due to go out.
type OverviewScheduledCampaign struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	ScheduledAt time.Time `json:"scheduledAt"`
	Timezone    string    `json:"timezone"`
	Recipients  int64     `json:"recipients"`
	// Readiness is nil until Send Readiness has run.
	Readiness *OverviewReadiness   `json:"readiness,omitempty"`
	Routes    []OverviewRouteShare `json:"routes"`
	// Cost is the estimate in dollars from each provider's price.
	Cost float64 `json:"cost"`
	// DurationMinutes estimates the send from each provider's rate limit
	// (the warm-up ramp for a warming provider); nil when a rate is unknown.
	DurationMinutes *float64 `json:"durationMinutes,omitempty"`
}

// OverviewNextCampaign is "Next scheduled campaign".
type OverviewNextCampaign struct {
	Campaign *OverviewScheduledCampaign `json:"campaign,omitempty"`
	// Drafts is the draft count, for the card when nothing is scheduled.
	Drafts int64 `json:"drafts"`
}

// OverviewRefreshFailure says a provider's account health couldn't be
// refreshed while it still sends: the card shows the last good numbers.
type OverviewRefreshFailure struct {
	Op      string `json:"op" doc:"GetAccount"`
	Status  int    `json:"status,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	// Attempts are the failed calls since the last success, oldest first.
	Attempts      []time.Time `json:"attempts"`
	LastSuccessAt *time.Time  `json:"lastSuccessAt,omitempty"`
	StatusLabel   string      `json:"statusLabel,omitempty"`
	StatusURL     string      `json:"statusUrl,omitempty"`
}

// OverviewProviderCard is one sending provider on the Overview: the
// providers screen's card (same numbers) and how the Overview names it.
type OverviewProviderCard struct {
	Card manage.ProviderCard `json:"card"`
	// DisplayName: "Postal (self-hosted)".
	DisplayName string `json:"displayName"`
	// Detail: "us-east-1 · Broadcasts, Automations".
	Detail string `json:"detail"`
	// Refresh is set while account health can't be refreshed.
	Refresh *OverviewRefreshFailure `json:"refresh,omitempty"`
	// StatusLabel and StatusURL point to the provider's public status page.
	StatusLabel string `json:"statusLabel,omitempty"`
	StatusURL   string `json:"statusUrl,omitempty"`
}

// OverviewProviders is "Sending providers".
type OverviewProviders struct {
	// Connected is 0 on the first run: the Overview leads into connecting one.
	Connected  int                    `json:"connected"`
	MonthStart time.Time              `json:"monthStart"`
	Providers  []OverviewProviderCard `json:"providers"`
}

// OverviewAudiencePoint is the subscribers on a day of the period (at the
// time of day the period started; the last point is now).
type OverviewAudiencePoint struct {
	Day         string `json:"day" format:"date"`
	Subscribers int64  `json:"subscribers"`
}

// OverviewListGrowth is one list's subscribers and their net change over
// the period (joins minus departures, as on List health).
type OverviewListGrowth struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Contacts int64  `json:"contacts"`
	Net      int64  `json:"net"`
}

type OverviewTier struct {
	Tier  string  `json:"tier" enum:"active,cooling,at_risk,sunset"`
	Count int64   `json:"count"`
	Pct   float64 `json:"pct" doc:"Share of all contacts, percent"`
	// Description: "opened in the last 30 days".
	Description string `json:"description"`
}

// OverviewAudience is "Subscriber growth" and "Engagement tiers".
type OverviewAudience struct {
	// Total is the unique contacts (the Audience header's count).
	Total int64 `json:"total"`
	// Subscribers is the lists' subscribed members added up.
	Subscribers int64 `json:"subscribers"`
	Days        int   `json:"days"`
	// Net is the subscribers' change over Days (the lists' joins minus
	// departures); NetPercent is relative to the start.
	Net        int64                   `json:"net"`
	NetPercent *float64                `json:"netPercent,omitempty"`
	Series     []OverviewAudiencePoint `json:"series"`
	Lists      []OverviewListGrowth    `json:"lists"`
	Tiers      []OverviewTier          `json:"tiers"`
	// Policy is the sunset policy in one sentence.
	Policy        string `json:"policy"`
	PolicyEnabled bool   `json:"policyEnabled"`
}

// OverviewRecentCampaigns is "Recent campaigns": the campaigns list's rows.
type OverviewRecentCampaigns struct {
	Items []campaigns.CampaignListItem `json:"items"`
}
