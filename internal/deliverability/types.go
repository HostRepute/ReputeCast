package deliverability

import "time"

// DeliverabilityOverview is the Deliverability screen (spec 7.8): health per
// sending domain and dedicated IP, shared pools, account health per sending
// provider, blocklists, Postmaster, SNDS, DMARC and the incident timeline.
type DeliverabilityOverview struct {
	HostRepute DeliverabilityHostRepute `json:"hostRepute"`
	Checks     DeliverabilityChecks     `json:"checks"`
	Assets     []AssetHealth            `json:"assets"`
	Pools      []PoolHealth             `json:"pools"`
	Providers  []ProviderAccountHealth  `json:"providers"`
	Blocklists BlocklistStatus          `json:"blocklists"`
	Postmaster *PostmasterPanel         `json:"postmaster,omitempty"`
	SNDS       *SNDSPanel               `json:"snds,omitempty"`
	DMARC      DMARCSummary             `json:"dmarc"`
	Incidents  IncidentTimeline         `json:"incidents"`
}

// DeliverabilityHostRepute is the HostRepute part of the screen header.
type DeliverabilityHostRepute struct {
	Status   string `json:"status" enum:"not_connected,connected,revoked,expired,error"`
	TeamName string `json:"teamName,omitempty"`
	// Sources is how many blocklists HostRepute checks (84).
	Sources         int32      `json:"sources"`
	IntervalMinutes int32      `json:"intervalMinutes"`
	LastSyncAt      *time.Time `json:"lastSyncAt,omitempty"`
	TokenRejectedAt *time.Time `json:"tokenRejectedAt,omitempty"`
	// Unreachable is set while HostRepute can't be reached; HostRepute data
	// on the page is from LastSyncAt.
	Unreachable *HostReputeOutage `json:"unreachable,omitempty"`
	DMARCRua    string            `json:"dmarcRua,omitempty"`
}

// HostReputeOutage describes a failing sync.
type HostReputeOutage struct {
	Message string     `json:"message" doc:"Couldn't reach api.hostrepute.com (timeout after 30 s, 3 attempts)"`
	Since   *time.Time `json:"since,omitempty"`
}

// DeliverabilityChecks says how and when the checks run.
type DeliverabilityChecks struct {
	Via      string     `json:"via" enum:"hostrepute,builtin"`
	Interval string     `json:"interval" enum:"hourly,daily"`
	LastAt   *time.Time `json:"lastAt,omitempty"`
	NextAt   *time.Time `json:"nextAt,omitempty"`
	// Running is the check in progress ("61 of 84 sources answered").
	Running *CheckRun `json:"running,omitempty"`
	// BuiltinSources is the size of the built-in DNSBL set (8).
	BuiltinSources int `json:"builtinSources"`
}

// CheckRun is one run of the checks.
type CheckRun struct {
	ID         string     `json:"id"`
	Via        string     `json:"via" enum:"hostrepute,builtin"`
	Trigger    string     `json:"trigger" enum:"schedule,manual,connect,fallback"`
	Status     string     `json:"status" enum:"running,done,failed"`
	Assets     []string   `json:"assets"`
	Total      int32      `json:"total"`
	Answered   int32      `json:"answered"`
	Listed     int32      `json:"listed"`
	Error      string     `json:"error,omitempty"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// AssetHealth is one sending domain or dedicated IP.
type AssetHealth struct {
	Asset string `json:"asset"`
	Type  string `json:"type" enum:"domain,ip"`
	Role  string `json:"role" enum:"sending_domain,dedicated_ip"`
	// UsedBy are the providers that send for the domain ("SES", "Postmark").
	UsedBy []string `json:"usedBy"`
	// Provider sends from the dedicated IP ("Postal").
	Provider       string          `json:"provider,omitempty"`
	ProviderID     string          `json:"providerId,omitempty"`
	ProviderStatus string          `json:"providerStatus,omitempty"`
	Warmup         *WarmupProgress `json:"warmup,omitempty"`
	// Score and Verdict come from HostRepute.
	Score   *int   `json:"score,omitempty"`
	Verdict string `json:"verdict,omitempty"`
	Tone    string `json:"tone" enum:"ok,warn,bad,neutral"`
	// Blocklists from the current source; Fallback holds the built-in
	// results while HostRepute data is stale.
	Blocklists AssetBlocklists  `json:"blocklists"`
	Fallback   *AssetBlocklists `json:"fallback,omitempty"`
	// Stale is when the HostRepute data on the card was last fresh.
	Stale *time.Time `json:"stale,omitempty"`
	// Auth is the SPF / DKIM / DMARC check of a domain.
	Auth *DomainAuthSummary `json:"auth,omitempty"`
	// DMARCAligned is the share of DMARC-aligned mail, last 7 days (percent).
	DMARCAligned *float64 `json:"dmarcAligned,omitempty"`
	// GmailReputation is Postmaster's domain reputation: high, medium, low, bad.
	GmailReputation string      `json:"gmailReputation,omitempty"`
	ReverseDNS      *ReverseDNS `json:"reverseDns,omitempty"`
	SNDS            *SNDSBrief  `json:"snds,omitempty"`
}

// WarmupProgress is "day 9 of 21".
type WarmupProgress struct {
	Day  int `json:"day"`
	Days int `json:"days"`
}

// AssetBlocklists summarizes an asset's blocklist checks.
type AssetBlocklists struct {
	Via       string     `json:"via" enum:"hostrepute,builtin,none"`
	Sources   int        `json:"sources"`
	Listed    int        `json:"listed"`
	Listings  []string   `json:"listings"`
	CheckedAt *time.Time `json:"checkedAt,omitempty"`
	// HighImpact is true when one listing is on a high-impact list.
	HighImpact bool `json:"highImpact"`
}

// DomainAuthSummary is "pass · pass · p=quarantine".
type DomainAuthSummary struct {
	SPF        string     `json:"spf"`
	DKIM       string     `json:"dkim"`
	DMARC      string     `json:"dmarc" doc:"Policy (none, quarantine, reject) or missing / invalid"`
	BIMI       string     `json:"bimi"`
	ReturnPath string     `json:"returnPath"`
	CheckedAt  *time.Time `json:"checkedAt,omitempty"`
}

// ReverseDNS is the forward-confirmed reverse DNS of an IP.
type ReverseDNS struct {
	PTR     string `json:"ptr,omitempty"`
	Matches bool   `json:"matches"`
	Status  string `json:"status"`
}

// SNDSBrief is "Green · 0 trap hits".
type SNDSBrief struct {
	FilterResult string `json:"filterResult"`
	TrapHits     int    `json:"trapHits"`
}

// PoolHealth is a provider's shared IP pool, checked at domain level.
type PoolHealth struct {
	ProviderID string `json:"providerId"`
	Provider   string `json:"provider"`
	Short      string `json:"short"`
	Adapter    string `json:"adapter"`
	Region     string `json:"region,omitempty"`
	// OnlyStream is set when the provider is primary for one stream only.
	OnlyStream        string `json:"onlyStream,omitempty" enum:"broadcast,automation,system"`
	AccountStatus     string `json:"accountStatus"`
	Status            string `json:"status"`
	GmailIPReputation string `json:"gmailIpReputation,omitempty"`
	DedicatedIPOption string `json:"dedicatedIpOption,omitempty"`
	VolumeThisMonth   *int64 `json:"volumeThisMonth,omitempty"`
	// ManagedReputation is true when the provider manages pool reputation (SES).
	ManagedReputation bool `json:"managedReputation"`
	// Current figures for the error state ("Bounce · complaint").
	BounceRate    *float64 `json:"bounceRate,omitempty"`
	ComplaintRate *float64 `json:"complaintRate,omitempty"`
}

// ProviderAccountHealth is a sending provider's account health against its
// own thresholds.
type ProviderAccountHealth struct {
	ProviderID string    `json:"providerId"`
	Name       string    `json:"name" doc:"Postal (self-hosted)"`
	Adapter    string    `json:"adapter"`
	Status     string    `json:"status"`
	Badge      string    `json:"badge" doc:"Healthy · no enforcement"`
	Tone       string    `json:"tone" enum:"ok,warn,bad,info,neutral"`
	Bounce     RateLimit `json:"bounce"`
	Complaint  RateLimit `json:"complaint"`
	// Note is one extra fact ("Reputation dashboard", "Stream", "Thresholds").
	Note *ProviderNote `json:"note,omitempty"`
}

// RateLimit is a rate against the provider's limits.
type RateLimit struct {
	Label string `json:"label" doc:"Bounce rate, Complaint rate, Spam complaint rate"`
	// Value is the current rate in percent.
	Value *float64 `json:"value,omitempty"`
	// Thresholds in words: "review 5%, pause 10%", "limit 10%", "warm-up guard 2%".
	Thresholds string `json:"thresholds"`
	// Usage is Value as a percent of the first limit (fills the bar).
	Usage float64 `json:"usage"`
	Tone  string  `json:"tone" enum:"ok,warn,bad"`
}

// ProviderNote is a label and a value.
type ProviderNote struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// BlocklistStatus is the blacklist table.
type BlocklistStatus struct {
	Via       string     `json:"via" enum:"hostrepute,builtin,none"`
	Sources   int        `json:"sources"`
	Interval  string     `json:"interval" enum:"hourly,daily"`
	Assets    []string   `json:"assets"`
	Clean     int        `json:"clean"`
	Listed    int        `json:"listed"`
	CheckedAt *time.Time `json:"checkedAt,omitempty"`
	// Stale is true while HostRepute data can't be refreshed.
	Stale   bool                    `json:"stale"`
	Results []BlocklistSourceResult `json:"results"`
}

// BlocklistSourceResult is one source and the assets checked on it.
type BlocklistSourceResult struct {
	Source       string     `json:"source"`
	Impact       string     `json:"impact" enum:"high,medium,low"`
	Assets       []string   `json:"assets"`
	Listed       bool       `json:"listed"`
	ListedAssets []string   `json:"listedAssets"`
	Since        *time.Time `json:"since,omitempty"`
	DelistURL    string     `json:"delistUrl,omitempty"`
	Detail       string     `json:"detail,omitempty"`
	Delisting    string     `json:"delisting,omitempty" enum:"requested,pending,delisted,denied"`
	DelistingAt  *time.Time `json:"delistingAt,omitempty"`
}

// PostmasterPanel is Google Postmaster Tools for the main sending domain.
type PostmasterPanel struct {
	Domain string `json:"domain"`
	Via    string `json:"via" enum:"hostrepute,direct"`
	Stale  bool   `json:"stale"`
	// Reputations: high, medium, low, bad.
	DomainReputation string `json:"domainReputation,omitempty"`
	IPReputation     string `json:"ipReputation,omitempty"`
	// IPReputationPool names the shared pool the IP reputation is about ("SES").
	IPReputationPool string `json:"ipReputationPool,omitempty"`
	// Rates in percent. SpamRate is the 30-day average.
	SpamRate *float64 `json:"spamRate,omitempty"`
	SPF      *float64 `json:"spf,omitempty"`
	DKIM     *float64 `json:"dkim,omitempty"`
	DMARC    *float64 `json:"dmarc,omitempty"`
	TLS      *float64 `json:"tls,omitempty"`
	// FeedbackLoopFlagged is the number of flagged FBL identifiers (nil: no FBL data).
	FeedbackLoopFlagged *int `json:"feedbackLoopFlagged,omitempty"`
	// Series is the spam rate in two-day buckets, oldest first.
	Series []PostmasterPoint `json:"series"`
	Peak   *PostmasterPeak   `json:"peak,omitempty"`
	// From and To are dates (YYYY-MM-DD) in the workspace's timezone.
	From string `json:"from" format:"date"`
	To   string `json:"to" format:"date"`
}

// PostmasterPoint is the average spam rate of a bucket of days.
type PostmasterPoint struct {
	From     string   `json:"from" format:"date"`
	To       string   `json:"to" format:"date"`
	SpamRate *float64 `json:"spamRate,omitempty"`
}

// PostmasterPeak is the highest daily spam rate of the period.
type PostmasterPeak struct {
	Day      string  `json:"day" format:"date"`
	SpamRate float64 `json:"spamRate"`
	// Note explains the peak from the timeline ("first send after migration").
	Note string `json:"note,omitempty"`
}

// SNDSPanel is Microsoft SNDS for the dedicated IP.
type SNDSPanel struct {
	IP           string `json:"ip"`
	Via          string `json:"via" enum:"hostrepute,direct"`
	Stale        bool   `json:"stale"`
	FilterResult string `json:"filterResult" enum:"green,yellow,red,"`
	Messages7d   int64  `json:"messages7d"`
	// ComplaintRate is the latest rate in percent.
	ComplaintRate *float64 `json:"complaintRate,omitempty"`
	TrapHits      int      `json:"trapHits"`
	JMRPReports   int      `json:"jmrpReports"`
}

// DMARCSummary is the DMARC aggregate report summary, last 7 days.
type DMARCSummary struct {
	Via      string `json:"via" enum:"hostrepute,upload,none"`
	Days     int    `json:"days"`
	Rua      string `json:"rua,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Policy   string `json:"policy,omitempty"`
	Reports  int    `json:"reports"`
	Messages int64  `json:"messages"`
	// PassRate is the share of messages that passed DMARC (percent).
	PassRate *float64         `json:"passRate,omitempty"`
	Sources  []DMARCSourceRow `json:"sources"`
	// Unknown is the largest source that isn't one of the providers.
	Unknown *DMARCSourceRow `json:"unknown,omitempty"`
}

// DMARCSourceRow is one sending source in the reports.
type DMARCSourceRow struct {
	Label      string `json:"label" doc:"Amazon SES, Postal · 203.0.113.25, Unknown · 198.51.100.77"`
	ProviderID string `json:"providerId,omitempty"`
	Adapter    string `json:"adapter,omitempty"`
	IP         string `json:"ip,omitempty"`
	Known      bool   `json:"known"`
	Messages   int64  `json:"messages"`
	SPF        string `json:"spf" doc:"aligned, or the raw result: fail, softfail, none…"`
	DKIM       string `json:"dkim"`
	Result     string `json:"result" enum:"pass,quarantined,rejected,fail"`
}

// IncidentTimeline is the incident timeline.
type IncidentTimeline struct {
	Items []IncidentItem `json:"items"`
	Total int64          `json:"total"`
	// FromHostRepute is true when HostRepute alerts are part of the timeline.
	FromHostRepute bool `json:"fromHostRepute"`
}

// IncidentItem is one incident, linked to campaigns and providers.
type IncidentItem struct {
	ID         string         `json:"id"`
	OccurredAt time.Time      `json:"occurredAt"`
	Kind       string         `json:"kind"`
	Severity   string         `json:"severity" enum:"info,ok,warn,bad"`
	Title      string         `json:"title"`
	Detail     string         `json:"detail"`
	Source     string         `json:"source" enum:"reputecast,hostrepute,builtin"`
	Links      []IncidentLink `json:"links"`
}

// IncidentLink points to the screen an incident is about.
type IncidentLink struct {
	Kind  string `json:"kind" enum:"warmup,campaign,campaign_report,provider,provider_events,import,deliverability"`
	Label string `json:"label"`
	ID    string `json:"id,omitempty"`
}
