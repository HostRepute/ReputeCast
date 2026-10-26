package manage

import (
	"time"

	"github.com/hostrepute/reputecast/internal/providers/state"
)

// ProviderStatusCount is one entry of "2 Healthy · 1 Warming up".
type ProviderStatusCount struct {
	Status string `json:"status" enum:"healthy,degraded,down,sandbox,warming_up,paused"`
	Count  int    `json:"count"`
}

// ProviderMonthAmount is one provider's part of a total.
type ProviderMonthAmount struct {
	ProviderID string  `json:"providerId"`
	Short      string  `json:"short"`
	Value      float64 `json:"value"`
}

// ProvidersSummary is the KPI row of the providers screen.
type ProvidersSummary struct {
	Connected     int                   `json:"connected"`
	ByStatus      []ProviderStatusCount `json:"byStatus"`
	SentThisMonth int64                 `json:"sentThisMonth"`
	SentBy        []ProviderMonthAmount `json:"sentBy"`
	// MonthStart is the first day of the month in the workspace timezone.
	MonthStart time.Time `json:"monthStart"`
	CostToDate float64   `json:"costToDate" doc:"Dollars"`
	// CostPer1000 is the blended cost per 1,000 emails this month.
	CostPer1000  *float64   `json:"costPer1000,omitempty"`
	Suppressions int64 `json:"suppressions"`
	// Pushed counts the suppressions providers get: hard bounces and
	// complaints that apply to all mail (review PR-1).
	Pushed   int64      `json:"pushed"`
	SyncedAt *time.Time `json:"syncedAt,omitempty"`
	// InSync is true when every provider that syncs is up to date.
	InSync bool `json:"inSync"`
	// CheckedAt is the oldest of the providers' last health checks.
	CheckedAt *time.Time `json:"checkedAt,omitempty"`
}

// ProviderStreamRole is what a provider does in a stream's routing.
type ProviderStreamRole struct {
	Stream string `json:"stream" enum:"broadcast,automation,system,transactional"`
	// Role: primary (100% of the default route), split (a share), fallback,
	// condition (only some recipients).
	Role    string `json:"role" enum:"primary,split,fallback,condition"`
	Percent int    `json:"percent"`
	// Active is true while a failover makes this fallback carry the stream.
	Active bool `json:"active"`
	// Held is true while this provider's share of the stream is held.
	Held bool `json:"held"`
}

// ProviderLimitRate is a bounce or complaint rate against the provider's limit.
type ProviderLimitRate struct {
	Value  *float64 `json:"value,omitempty" doc:"Percent over the last 7 days"`
	Limit  float64  `json:"limit"`
	Review float64  `json:"review,omitempty"`
	Pause  float64  `json:"pause,omitempty"`
	Label  string   `json:"label,omitempty"`
	// Stale is true while webhooks are silent: the rate stopped updating.
	Stale bool `json:"stale"`
}

// ProviderQuotaUse is the sending allowance in use.
type ProviderQuotaUse struct {
	Kind string `json:"kind" enum:"rolling_24h,monthly_plan,warmup_daily,none"`
	Max  int64  `json:"max"`
	Used int64  `json:"used"`
	// FromFailover is how much of Used came from mail rerouted here.
	FromFailover int64 `json:"fromFailover"`
}

// ProviderSendRate is the send rate in use.
type ProviderSendRate struct {
	Max       float64 `json:"max"`
	Now       float64 `json:"now"`
	PeakToday float64 `json:"peakToday"`
	Label     string  `json:"label,omitempty"`
	Throttled bool    `json:"throttled"`
}

// ProviderCostToDate is the provider's cost this month.
type ProviderCostToDate struct {
	Basis   string   `json:"basis" enum:"metered,plan,self_hosted"`
	ToDate  float64  `json:"toDate" doc:"Dollars this month"`
	Per1000 *float64 `json:"per1000,omitempty"`
	// Monthly is the plan or server price per month the user entered.
	Monthly float64 `json:"monthly,omitempty"`
	Source  string  `json:"source,omitempty"`
}

// ProviderCapabilityBadge is one capability chip.
type ProviderCapabilityBadge struct {
	Label string `json:"label"`
	// Kind: yes (green), partial (amber), no (struck through), info (neutral).
	Kind string `json:"kind" enum:"yes,partial,no,info"`
}

// ProviderWebhookFeed is the event feed of a provider.
type ProviderWebhookFeed struct {
	Kind        string     `json:"kind"` // SNS, Registered, Bounce mailbox
	LastEventAt *time.Time `json:"lastEventAt,omitempty"`
	SilentSince *time.Time `json:"silentSince,omitempty"`
}

// ProviderWarmupBrief is the warm-up state of a provider.
type ProviderWarmupBrief struct {
	PlanID    string  `json:"planId"`
	Day       int     `json:"day"`
	Days      int     `json:"days"`
	Status    string  `json:"status" enum:"active,slowed,paused,done"`
	TodayCap  int64   `json:"todayCap"`
	SentToday int64   `json:"sentToday"`
	Rate      float64 `json:"rate"`
	Share     int     `json:"share"`
}

// ProviderCallError is one failed call.
type ProviderCallError struct {
	At      time.Time `json:"at"`
	Op      string    `json:"op"`
	Status  int       `json:"status"`
	Code    string    `json:"code"`
	Message string    `json:"message"`
}

// ProviderDownState describes an outage or rejected credentials.
type ProviderDownState struct {
	Since         time.Time           `json:"since"`
	Reason        string              `json:"reason" enum:"outage,auth"`
	LastSuccessAt *time.Time          `json:"lastSuccessAt,omitempty"`
	LastSuccessOp string              `json:"lastSuccessOp,omitempty"`
	Errors        []ProviderCallError `json:"errors"`
	// StatusSummary is the provider's public status page ("AWS reports an
	// SES incident in us-east-1.").
	StatusSummary string `json:"statusSummary,omitempty"`
	StatusLabel   string `json:"statusLabel,omitempty"`
	StatusURL     string `json:"statusUrl,omitempty"`
	// CredentialLabel names the credential to replace ("Server API token").
	CredentialLabel string `json:"credentialLabel,omitempty"`
}

// ProviderCard is one provider on the providers screen.
type ProviderCard struct {
	ID          string               `json:"id"`
	Adapter     string               `json:"adapter"`
	Name        string               `json:"name"`
	Short       string               `json:"short"`
	Label       string               `json:"label"`
	Color       string               `json:"color" enum:"ses,postal,postmark,other"`
	Status      string               `json:"status" enum:"healthy,degraded,down,sandbox,warming_up,paused"`
	Roles       []ProviderStreamRole `json:"roles"`
	IPMode      string               `json:"ipMode" enum:"shared,dedicated"`
	IPs         []string             `json:"ips"`
	Webhook     ProviderWebhookFeed  `json:"webhook"`
	ErrorRate7d *float64             `json:"errorRate7d,omitempty" doc:"Percent of API calls that failed"`
	ErrorRate3m *float64             `json:"errorRate3m,omitempty"`
	// AccountKind says how to label Account: "account" ("Account status")
	// or "server" ("Server", self-hosted).
	AccountKind string `json:"accountKind" enum:"account,server"`
	Account     string `json:"account"`
	// Tenant is the workspace's own SES tenant (PR-11).
	Tenant       *ProviderTenant           `json:"tenant,omitempty"`
	Bounce       ProviderLimitRate         `json:"bounce"`
	Complaint    ProviderLimitRate         `json:"complaint"`
	Quota        ProviderQuotaUse          `json:"quota"`
	Rate         ProviderSendRate          `json:"rate"`
	Cost         ProviderCostToDate        `json:"cost"`
	Capabilities []ProviderCapabilityBadge `json:"capabilities"`
	Warmup       *ProviderWarmupBrief      `json:"warmup,omitempty"`
	Down         *ProviderDownState        `json:"down,omitempty"`
	// TakingFailover lists the streams this provider carries for a provider
	// that is down.
	TakingFailover []string   `json:"takingFailover"`
	CheckedAt      *time.Time `json:"checkedAt,omitempty"`
	ConnectedAt    time.Time  `json:"connectedAt"`
	// Credentials lists the secret fields (never their values) so the
	// settings dialog can offer "Replace".
	Credentials []ProviderCredentialField `json:"credentials"`
}

// ProviderCredentialField is a write-only credential of a provider.
type ProviderCredentialField struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Set   bool   `json:"set"`
}

// ProviderStreamRoute is one stream in the "Routing per stream" card.
type ProviderStreamRoute struct {
	Stream     string               `json:"stream" enum:"broadcast,automation,system,transactional"`
	Splits     []ProviderRouteShare `json:"splits"`
	Fallback   *ProviderRouteShare  `json:"fallback,omitempty"`
	Conditions int                  `json:"conditions"`
	Failover   string               `json:"failover" enum:"automatic,pause_and_ask,warm_only"`
}

// ProviderRouteShare is a provider's share of a stream.
type ProviderRouteShare struct {
	ProviderID string `json:"providerId"`
	Short      string `json:"short"`
	Color      string `json:"color" enum:"ses,postal,postmark,other"`
	Percent    int    `json:"percent"`
}

// ProviderNamedCount is a label with a count ("welcome series 211").
type ProviderNamedCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// ProviderHeldCampaign is a broadcast whose share on a down provider is held.
type ProviderHeldCampaign struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Planned     int64                `json:"planned"`
	Sent        int64                `json:"sent"`
	Held        int64                `json:"held"`
	Decision    string               `json:"decision,omitempty" doc:"wait, move or cancel once someone decided"`
	OtherShares []ProviderRouteShare `json:"otherShares"`
}

// ProviderFailoverAlternative is a provider a held broadcast could move to.
type ProviderFailoverAlternative struct {
	ProviderID string `json:"providerId"`
	Name       string `json:"name"`
	Short      string `json:"short"`
	WarmingUp  bool   `json:"warmingUp"`
	// ExceedsCap: the held recipients don't fit in today's warm-up cap.
	ExceedsCap bool `json:"exceedsCap"`
	// Allowed is false when one of the target checks failed: moving there
	// is refused. Recommended: every check passed.
	Allowed     bool `json:"allowed"`
	Recommended bool `json:"recommended"`
	// Checks are what was checked about the target for the held share
	// (available, bulk terms, sandbox, sender domain, quota, warm-up).
	Checks []state.VetCheck `json:"checks"`
}

// ProviderOutageStream is how one stream is handling an outage.
type ProviderOutageStream struct {
	Stream string `json:"stream" enum:"broadcast,automation,system,transactional"`
	// Role of the down provider in this stream.
	Role string `json:"role" enum:"primary,split,fallback,condition"`
	// State: failover_active, waiting (a decision is needed), held (no
	// send in progress, mail waits), fallback_unavailable.
	State         string               `json:"state" enum:"failover_active,waiting,held,fallback_unavailable"`
	Policy        string               `json:"policy" enum:"automatic,pause_and_ask,warm_only"`
	Since         time.Time            `json:"since"`
	Fallback      string               `json:"fallback,omitempty"`
	FallbackShort string               `json:"fallbackShort,omitempty"`
	Primary       string               `json:"primary,omitempty"`
	Rerouted      int64                `json:"rerouted"`
	ReroutedBy    []ProviderNamedCount `json:"reroutedBy"`
	// Suggestion names a connected provider that could be a second fallback.
	Suggestion        string                        `json:"suggestion,omitempty"`
	Campaign          *ProviderHeldCampaign         `json:"campaign,omitempty"`
	Alternatives      []ProviderFailoverAlternative `json:"alternatives"`
	SwitchBackMinutes int                           `json:"switchBackMinutes"`
	// FallbackChecks are the checks the fallback was vetted with before any
	// mail moved to it (PR-7): all passed while a failover is active; a
	// failed one says why the stream waits instead. Each carries its
	// sentence in the reader's language.
	FallbackChecks []state.VetCheck `json:"fallbackChecks"`
}

// ProviderQuotaForecast says when a provider taking failover runs out of quota.
type ProviderQuotaForecast struct {
	ProviderID    string     `json:"providerId"`
	PerDay        int64      `json:"perDay"`
	Stream        string     `json:"stream"`
	RunsOutOn     *time.Time `json:"runsOutOn,omitempty"`
	NextTierAdds  int64      `json:"nextTierAdds,omitempty"`
	NextTierPrice float64    `json:"nextTierPrice,omitempty"`
}

// ProviderWhyNotMove explains why a held broadcast doesn't move to a warming provider.
type ProviderWhyNotMove struct {
	ProviderID string  `json:"providerId"`
	Held       int64   `json:"held"`
	TodayCap   int64   `json:"todayCap"`
	Multiple   float64 `json:"multiple"`
	AgeDays    int     `json:"ageDays"`
	Rate       float64 `json:"rate"`
}

// ProviderOutage is the outage banner and stream columns.
type ProviderOutage struct {
	ProviderID string                  `json:"providerId"`
	Provider   string                  `json:"provider"`
	Short      string                  `json:"short"`
	Since      time.Time               `json:"since"`
	Down       ProviderDownState       `json:"down"`
	ErrorRate  *float64                `json:"errorRate,omitempty"`
	Codes      []string                `json:"codes"`
	Operation  string                  `json:"operation"`
	Streams    []ProviderOutageStream  `json:"streams"`
	Forecasts  []ProviderQuotaForecast `json:"forecasts"`
	WhyNot     []ProviderWhyNotMove    `json:"whyNot"`
	Incidents  []ProviderIncidentEntry `json:"incidents"`
}

// ProviderIncidentEntry is one entry of the incident timeline.
type ProviderIncidentEntry struct {
	ID         string    `json:"id"`
	At         time.Time `json:"at"`
	Kind       string    `json:"kind"`
	Severity   string    `json:"severity" enum:"info,ok,warn,bad"`
	Title      string    `json:"title"`
	Detail     string    `json:"detail"`
	CampaignID string    `json:"campaignId,omitempty"`
}

// ProviderWebhookCheck is one finding of "What ReputeCast checked".
type ProviderWebhookCheck struct {
	Key    string     `json:"key" enum:"api,registered,endpoint,provider_log"`
	OK     *bool      `json:"ok,omitempty"`
	At     *time.Time `json:"at,omitempty"`
	Detail string     `json:"detail,omitempty"`
	ID     string     `json:"id,omitempty"`
	URL    string     `json:"url,omitempty"`
	Count  *int64     `json:"count,omitempty"`
	Queued *int64     `json:"queued,omitempty"`
}

// ProviderAwaitingMessage is a message a silent provider accepted.
type ProviderAwaitingMessage struct {
	At                time.Time  `json:"at"`
	Recipient         string     `json:"recipient" doc:"Masked"`
	Source            string     `json:"source"`
	ProviderMessageID string     `json:"providerMessageId"`
	Status            string     `json:"status" enum:"accepted,delivered,bounced,deferred,complained,rejected"`
	DeliveredAt       *time.Time `json:"deliveredAt,omitempty"`
}

// ProviderWebhooksSilent is the webhooks-silent banner and diagnosis.
type ProviderWebhooksSilent struct {
	ProviderID string    `json:"providerId"`
	Provider   string    `json:"provider"`
	Short      string    `json:"short"`
	Since      time.Time `json:"since"`
	Accepted   int64     `json:"accepted"`
	// Overdue messages were sent more than 30 minutes ago and still have no
	// event; InGrace are still inside those 30 minutes (PR-6).
	Overdue            int64                  `json:"overdue"`
	InGrace            int64                  `json:"inGrace"`
	Campaigns          []ProviderNamedCount   `json:"campaigns"`
	BroadcastShareHeld bool                   `json:"broadcastShareHeld"`
	Checks             []ProviderWebhookCheck `json:"checks"`
	Causes             []string               `json:"causes"`
	// Hourly is events received per hour over the last 24 hours, oldest first.
	Hourly      []int64                   `json:"hourly"`
	HourlyFrom  time.Time                 `json:"hourlyFrom"`
	Awaiting    []ProviderAwaitingMessage `json:"awaiting"`
	CanBackfill bool                      `json:"canBackfill"`
	CanRegister bool                      `json:"canRegister"`
	CanSendTest bool                      `json:"canSendTest"`
	WebhookURL  string                    `json:"webhookUrl"`
	// BroadcastPolicy explains why other providers don't take the held share.
	BroadcastPolicy string `json:"broadcastPolicy" enum:"automatic,pause_and_ask,warm_only"`
}

// ProvidersOverview is the providers screen.
type ProvidersOverview struct {
	Summary   ProvidersSummary        `json:"summary"`
	Providers []ProviderCard          `json:"providers"`
	Routing   []ProviderStreamRoute   `json:"routing"`
	Outage    *ProviderOutage         `json:"outage,omitempty"`
	Silent    *ProviderWebhooksSilent `json:"silent,omitempty"`
	// ProviderAuthFailure is set when a provider rejects its credentials.
	ProviderAuthFailure *ProviderAuthFailure `json:"authFailure,omitempty"`
}

// ProviderAuthFailure is the "a provider rejected our credentials" state.
type ProviderAuthFailure struct {
	ProviderID string                 `json:"providerId"`
	Provider   string                 `json:"provider"`
	Short      string                 `json:"short"`
	Down       ProviderDownState      `json:"down"`
	Rerouted   int64                  `json:"rerouted"`
	Streams    []ProviderOutageStream `json:"streams"`
}

// ProviderTenant is the workspace's own SES tenant: its own suppression
// list and reputation policy.
type ProviderTenant struct {
	Name string `json:"name"`
	// Policy is the reputation policy's name ("Standard"); empty while SES
	// hasn't said.
	Policy string `json:"policy,omitempty"`
}
