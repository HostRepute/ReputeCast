package manage

import "time"

// ProviderStatusCount is one entry of "2 Healthy · 1 Warming up".
type ProviderStatusCount struct {
	Status string `json:"status" enum:"healthy,degraded,down,sandbox,warming_up,paused"`
	Count  int    `json:"count"`
}

// ProviderAmount is one provider's part of a total.
type ProviderAmount struct {
	ProviderID string  `json:"providerId"`
	Short      string  `json:"short"`
	Value      float64 `json:"value"`
}

// ProvidersSummary is the KPI row of the providers screen.
type ProvidersSummary struct {
	Connected     int                   `json:"connected"`
	ByStatus      []ProviderStatusCount `json:"byStatus"`
	SentThisMonth int64                 `json:"sentThisMonth"`
	SentBy        []ProviderAmount      `json:"sentBy"`
	// MonthStart is the first day of the month in the workspace timezone.
	MonthStart time.Time `json:"monthStart"`
	CostToDate float64   `json:"costToDate" doc:"Dollars"`
	// CostPer1000 is the blended cost per 1,000 emails this month.
	CostPer1000  *float64   `json:"costPer1000,omitempty"`
	Suppressions int64      `json:"suppressions"`
	SyncedAt     *time.Time `json:"syncedAt,omitempty"`
	// InSync is true when every provider that syncs is up to date.
	InSync bool `json:"inSync"`
	// CheckedAt is the oldest of the providers' last health checks.
	CheckedAt *time.Time `json:"checkedAt,omitempty"`
}

// ProviderRole is what a provider does in a stream's routing.
type ProviderRole struct {
	Stream string `json:"stream" enum:"broadcast,automation,system"`
	// Role: primary (100% of the default route), split (a share), fallback,
	// condition (only some recipients).
	Role    string `json:"role" enum:"primary,split,fallback,condition"`
	Percent int    `json:"percent"`
	// Active is true while a failover makes this fallback carry the stream.
	Active bool `json:"active"`
	// Held is true while this provider's share of the stream is held.
	Held bool `json:"held"`
}

// ProviderRate is a bounce or complaint rate against the provider's limit.
type ProviderRate struct {
	Value  *float64 `json:"value,omitempty" doc:"Percent over the last 7 days"`
	Limit  float64  `json:"limit"`
	Review float64  `json:"review,omitempty"`
	Pause  float64  `json:"pause,omitempty"`
	Label  string   `json:"label,omitempty"`
	// Stale is true while webhooks are silent: the rate stopped updating.
	Stale bool `json:"stale"`
}

// ProviderQuota is the sending allowance in use.
type ProviderQuota struct {
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

// ProviderCost is the provider's cost this month.
type ProviderCost struct {
	Basis   string   `json:"basis" enum:"metered,plan,self_hosted"`
	ToDate  float64  `json:"toDate" doc:"Dollars this month"`
	Per1000 *float64 `json:"per1000,omitempty"`
	// Monthly is the plan or server price per month the user entered.
	Monthly float64 `json:"monthly,omitempty"`
	Source  string  `json:"source,omitempty"`
}

// CapabilityBadge is one capability chip.
type CapabilityBadge struct {
	Label string `json:"label"`
	// Kind: yes (green), partial (amber), no (struck through), info (neutral).
	Kind string `json:"kind" enum:"yes,partial,no,info"`
}

// ProviderWebhook is the event feed of a provider.
type ProviderWebhook struct {
	Kind        string     `json:"kind"` // SNS, Registered, Bounce mailbox
	LastEventAt *time.Time `json:"lastEventAt,omitempty"`
	SilentSince *time.Time `json:"silentSince,omitempty"`
}

// ProviderWarmup is the warm-up state of a provider.
type ProviderWarmup struct {
	PlanID    string  `json:"planId"`
	Day       int     `json:"day"`
	Days      int     `json:"days"`
	Status    string  `json:"status" enum:"active,slowed,paused,done"`
	TodayCap  int64   `json:"todayCap"`
	SentToday int64   `json:"sentToday"`
	Rate      float64 `json:"rate"`
	Share     int     `json:"share"`
}

// ProviderError is one failed call.
type ProviderError struct {
	At      time.Time `json:"at"`
	Op      string    `json:"op"`
	Status  int       `json:"status"`
	Code    string    `json:"code"`
	Message string    `json:"message"`
}

// ProviderDown describes an outage or rejected credentials.
type ProviderDown struct {
	Since         time.Time       `json:"since"`
	Reason        string          `json:"reason" enum:"outage,auth"`
	LastSuccessAt *time.Time      `json:"lastSuccessAt,omitempty"`
	LastSuccessOp string          `json:"lastSuccessOp,omitempty"`
	Errors        []ProviderError `json:"errors"`
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
	ID          string          `json:"id"`
	Adapter     string          `json:"adapter"`
	Name        string          `json:"name"`
	Short       string          `json:"short"`
	Label       string          `json:"label"`
	Color       string          `json:"color" enum:"ses,postal,postmark,other"`
	Status      string          `json:"status" enum:"healthy,degraded,down,sandbox,warming_up,paused"`
	Roles       []ProviderRole  `json:"roles"`
	IPMode      string          `json:"ipMode" enum:"shared,dedicated"`
	IPs         []string        `json:"ips"`
	Webhook     ProviderWebhook `json:"webhook"`
	ErrorRate7d *float64        `json:"errorRate7d,omitempty" doc:"Percent of API calls that failed"`
	ErrorRate3m *float64        `json:"errorRate3m,omitempty"`
	// AccountKind says how to label Account: "account" ("Account status")
	// or "server" ("Server", self-hosted).
	AccountKind  string            `json:"accountKind" enum:"account,server"`
	Account      string            `json:"account"`
	Bounce       ProviderRate      `json:"bounce"`
	Complaint    ProviderRate      `json:"complaint"`
	Quota        ProviderQuota     `json:"quota"`
	Rate         ProviderSendRate  `json:"rate"`
	Cost         ProviderCost      `json:"cost"`
	Capabilities []CapabilityBadge `json:"capabilities"`
	Warmup       *ProviderWarmup   `json:"warmup,omitempty"`
	Down         *ProviderDown     `json:"down,omitempty"`
	// TakingFailover lists the streams this provider carries for a provider
	// that is down.
	TakingFailover []string   `json:"takingFailover"`
	CheckedAt      *time.Time `json:"checkedAt,omitempty"`
	ConnectedAt    time.Time  `json:"connectedAt"`
	// Credentials lists the secret fields (never their values) so the
	// settings dialog can offer "Replace".
	Credentials []CredentialField `json:"credentials"`
}

// CredentialField is a write-only credential of a provider.
type CredentialField struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Set   bool   `json:"set"`
}

// StreamRoute is one stream in the "Routing per stream" card.
type StreamRoute struct {
	Stream     string          `json:"stream" enum:"broadcast,automation,system"`
	Splits     []RouteSplitPct `json:"splits"`
	Fallback   *RouteSplitPct  `json:"fallback,omitempty"`
	Conditions int             `json:"conditions"`
	Failover   string          `json:"failover" enum:"automatic,pause_and_ask,warm_only"`
}

// RouteSplitPct is a provider's share of a stream.
type RouteSplitPct struct {
	ProviderID string `json:"providerId"`
	Short      string `json:"short"`
	Color      string `json:"color" enum:"ses,postal,postmark,other"`
	Percent    int    `json:"percent"`
}

// NamedCount is a label with a count ("welcome series 211").
type NamedCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// HeldCampaign is a broadcast whose share on a down provider is held.
type HeldCampaign struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Planned     int64           `json:"planned"`
	Sent        int64           `json:"sent"`
	Held        int64           `json:"held"`
	Decision    string          `json:"decision,omitempty" doc:"wait, move or cancel once someone decided"`
	OtherShares []RouteSplitPct `json:"otherShares"`
}

// FailoverAlternative is a provider a held broadcast could move to.
type FailoverAlternative struct {
	ProviderID string `json:"providerId"`
	Name       string `json:"name"`
	Short      string `json:"short"`
	WarmingUp  bool   `json:"warmingUp"`
	// ExceedsCap: the held recipients don't fit in today's warm-up cap.
	ExceedsCap  bool `json:"exceedsCap"`
	Recommended bool `json:"recommended"`
}

// OutageStream is how one stream is handling an outage.
type OutageStream struct {
	Stream string `json:"stream" enum:"broadcast,automation,system"`
	// Role of the down provider in this stream.
	Role string `json:"role" enum:"primary,split,fallback,condition"`
	// State: failover_active, waiting (a decision is needed), held (no
	// send in progress, mail waits), fallback_unavailable.
	State         string       `json:"state" enum:"failover_active,waiting,held,fallback_unavailable"`
	Policy        string       `json:"policy" enum:"automatic,pause_and_ask,warm_only"`
	Since         time.Time    `json:"since"`
	Fallback      string       `json:"fallback,omitempty"`
	FallbackShort string       `json:"fallbackShort,omitempty"`
	Primary       string       `json:"primary,omitempty"`
	Rerouted      int64        `json:"rerouted"`
	ReroutedBy    []NamedCount `json:"reroutedBy"`
	// Suggestion names a connected provider that could be a second fallback.
	Suggestion        string                `json:"suggestion,omitempty"`
	Campaign          *HeldCampaign         `json:"campaign,omitempty"`
	Alternatives      []FailoverAlternative `json:"alternatives"`
	SwitchBackMinutes int                   `json:"switchBackMinutes"`
}

// QuotaForecast says when a provider taking failover runs out of quota.
type QuotaForecast struct {
	ProviderID    string     `json:"providerId"`
	PerDay        int64      `json:"perDay"`
	Stream        string     `json:"stream"`
	RunsOutOn     *time.Time `json:"runsOutOn,omitempty"`
	NextTierAdds  int64      `json:"nextTierAdds,omitempty"`
	NextTierPrice float64    `json:"nextTierPrice,omitempty"`
}

// WhyNotMove explains why a held broadcast doesn't move to a warming provider.
type WhyNotMove struct {
	ProviderID string  `json:"providerId"`
	Held       int64   `json:"held"`
	TodayCap   int64   `json:"todayCap"`
	Multiple   float64 `json:"multiple"`
	AgeDays    int     `json:"ageDays"`
	Rate       float64 `json:"rate"`
}

// ProviderOutage is the outage banner and stream columns.
type ProviderOutage struct {
	ProviderID string             `json:"providerId"`
	Provider   string             `json:"provider"`
	Short      string             `json:"short"`
	Since      time.Time          `json:"since"`
	Down       ProviderDown       `json:"down"`
	ErrorRate  *float64           `json:"errorRate,omitempty"`
	Codes      []string           `json:"codes"`
	Operation  string             `json:"operation"`
	Streams    []OutageStream     `json:"streams"`
	Forecasts  []QuotaForecast    `json:"forecasts"`
	WhyNot     []WhyNotMove       `json:"whyNot"`
	Incidents  []ProviderIncident `json:"incidents"`
}

// ProviderIncident is one entry of the incident timeline.
type ProviderIncident struct {
	ID         string    `json:"id"`
	At         time.Time `json:"at"`
	Kind       string    `json:"kind"`
	Severity   string    `json:"severity" enum:"info,ok,warn,bad"`
	Title      string    `json:"title"`
	Detail     string    `json:"detail"`
	CampaignID string    `json:"campaignId,omitempty"`
}

// WebhookCheckView is one finding of "What ReputeCast checked".
type WebhookCheckView struct {
	Key    string     `json:"key" enum:"api,registered,endpoint,provider_log"`
	OK     *bool      `json:"ok,omitempty"`
	At     *time.Time `json:"at,omitempty"`
	Detail string     `json:"detail,omitempty"`
	ID     string     `json:"id,omitempty"`
	URL    string     `json:"url,omitempty"`
	Count  *int64     `json:"count,omitempty"`
	Queued *int64     `json:"queued,omitempty"`
}

// AwaitingMessage is a message a silent provider accepted.
type AwaitingMessage struct {
	At                time.Time  `json:"at"`
	Recipient         string     `json:"recipient" doc:"Masked"`
	Source            string     `json:"source"`
	ProviderMessageID string     `json:"providerMessageId"`
	Status            string     `json:"status" enum:"accepted,delivered,bounced,deferred,complained,rejected"`
	DeliveredAt       *time.Time `json:"deliveredAt,omitempty"`
}

// WebhooksSilent is the webhooks-silent banner and diagnosis.
type WebhooksSilent struct {
	ProviderID         string             `json:"providerId"`
	Provider           string             `json:"provider"`
	Short              string             `json:"short"`
	Since              time.Time          `json:"since"`
	Accepted           int64              `json:"accepted"`
	Campaigns          []NamedCount       `json:"campaigns"`
	BroadcastShareHeld bool               `json:"broadcastShareHeld"`
	Checks             []WebhookCheckView `json:"checks"`
	Causes             []string           `json:"causes"`
	// Hourly is events received per hour over the last 24 hours, oldest first.
	Hourly      []int64           `json:"hourly"`
	HourlyFrom  time.Time         `json:"hourlyFrom"`
	Awaiting    []AwaitingMessage `json:"awaiting"`
	CanBackfill bool              `json:"canBackfill"`
	CanRegister bool              `json:"canRegister"`
	CanSendTest bool              `json:"canSendTest"`
	WebhookURL  string            `json:"webhookUrl"`
	// BroadcastPolicy explains why other providers don't take the held share.
	BroadcastPolicy string `json:"broadcastPolicy" enum:"automatic,pause_and_ask,warm_only"`
}

// ProvidersOverview is the providers screen.
type ProvidersOverview struct {
	Summary   ProvidersSummary `json:"summary"`
	Providers []ProviderCard   `json:"providers"`
	Routing   []StreamRoute    `json:"routing"`
	Outage    *ProviderOutage  `json:"outage,omitempty"`
	Silent    *WebhooksSilent  `json:"silent,omitempty"`
	// AuthFailure is set when a provider rejects its credentials.
	AuthFailure *AuthFailure `json:"authFailure,omitempty"`
}

// AuthFailure is the "a provider rejected our credentials" state.
type AuthFailure struct {
	ProviderID string         `json:"providerId"`
	Provider   string         `json:"provider"`
	Short      string         `json:"short"`
	Down       ProviderDown   `json:"down"`
	Rerouted   int64          `json:"rerouted"`
	Streams    []OutageStream `json:"streams"`
}
