package readiness

import (
	"time"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/providers/state"
)

// Facts is everything a run looked at. Gather collects them from the other
// domains; the demo and tests build them by hand. Assemble turns them into
// the checklist and the estimates.
type Facts struct {
	Now time.Time
	// Location is the workspace time zone (clock times in the evidence).
	Location *time.Location
	// Workspace is the slug, for links to other screens.
	Workspace string

	Campaign   CampaignFacts
	Audience   AudienceFacts
	Route      RouteFacts
	Reputation ReputationFacts
	Auth       AuthFacts
	Content    ContentFacts
	Compliance ComplianceFacts
	// Languages are the email's language versions (content languages).
	Languages LanguagesFacts
	// Accessibility is the accessibility checks of every language version.
	Accessibility accessibility.ReadinessFacts
	// Conditional is the email's conditional blocks and versions, Polls its
	// polls and surveys (design expansion, X11).
	Conditional ConditionalFacts
	Polls       PollFacts
	// Verification is the audience's address-check results (verification.go).
	Verification VerificationFacts
	// Web is the campaign's post on the website (design expansion, X18).
	Web WebFacts
	// MailboxRates are the per-mailbox-provider throttles (messages per
	// second, 0 = full rate).
	MailboxRates map[string]float64
	// Grow are the Referral and Recommendations blocks' rows (grow.go).
	Grow []GrowRow
}

// CampaignFacts describes the campaign being checked.
type CampaignFacts struct {
	ID        string
	Name      string
	Subject   string
	Preheader string
	Version   int
	// CompiledAt is when the content was last compiled (saved).
	CompiledAt *time.Time
	SenderName string
	// SenderEmail is the From address; SenderDomain its domain.
	SenderEmail  string
	SenderDomain string
	// StartAt is when the send would start (the schedule, or now).
	StartAt time.Time
	// ScheduleMode is now, at or best_time.
	ScheduleMode string
}

// AudienceFacts are the audience counts after exclusions.
type AudienceFacts struct {
	// BaseLabel names the audience ("Weekly Digest list", "All subscribers").
	BaseLabel string
	// GroupLabel is the caption of the audience-quality group ("Weekly Digest").
	GroupLabel string
	Lists      []string
	// Base counts the subscribed contacts of the lists and segments (minus
	// excluded lists and segments).
	Base       int64
	Suppressed int64
	Role       int64
	Duplicates int64
	// Stopped counts contacts the sunset policy stopped Broadcasts to.
	Stopped int64
	// SunsetOverride is true when the campaign overrides the sunset policy;
	// StoppedIncluded counts the stopped contacts it sends to anyway.
	SunsetOverride  bool
	StoppedIncluded int64
	// SunsetExcluded counts the sunset-tier contacts removed because the
	// campaign excludes that tier; UnengagedExcluded says it does.
	SunsetExcluded    int64
	UnengagedExcluded bool
	Final             int64
	// Unengaged counts final recipients with no open or click in
	// UnengagedDays days (the sunset tier).
	Unengaged     int64
	UnengagedDays int
	// RecentlyBounced counts final recipients who bounced in the last 30
	// days without being suppressed (soft bounces).
	RecentlyBounced int64
	// Mailboxes counts final recipients per mailbox provider.
	Mailboxes map[string]int64
	// Languages counts final recipients per contact language code
	// (lowercase, "" for none), for content languages.
	Languages map[string]int64
	// Empty is true when the campaign has no audience yet.
	Empty bool
	// SegmentsSkipped is true when segments couldn't be counted.
	SegmentsSkipped bool
	// SuppressedEmails are the suppressed contacts' addresses, for the
	// address checks' count of invalid ones (verification.go).
	SuppressedEmails []string `json:"-"`
}

// LanguagesFacts are the email's language versions and who gets which.
type LanguagesFacts = languages.ReadinessFacts

// RouteFacts is the sending route of the campaign and its providers.
type RouteFacts struct {
	Stream string
	// Policy is the stream's failover policy.
	Policy   string
	Override bool
	// NoRoute is true when the stream has no provider.
	NoRoute bool
	Shares  []ShareFacts
}

// ShareFacts is one provider of the route with what the checks need.
type ShareFacts struct {
	ProviderID string
	Name       string
	Short      string
	Adapter    string
	Label      string
	Recipients int64
	// Percent of the final audience.
	Percent float64
	// Status is the provider status (healthy, warming_up, down…).
	Status    string
	Available bool
	// DownSince and DownDetail describe an outage ("mta.acme.io has refused
	// connections since 08:52 (ECONNREFUSED on port 25 and 443)").
	DownSince  *time.Time
	DownDetail string
	// Warm-up: day and length of the plan, the cap on the send day and
	// whether that day is today.
	WarmupDay   int
	WarmupDays  int
	WarmupCap   int64
	WarmupToday bool
	// Rate is the provider's send rate (messages per second); RateRamp is
	// true when it comes from a warm-up plan.
	Rate     float64
	RateRamp bool
	Pricing  state.Pricing
	// Bounce and Complaint are 7-day rates in percent (nil without volume);
	// ProviderReported is true when the provider measured them.
	Bounce           *float64
	Complaint        *float64
	ProviderReported bool
	Limits           state.Limits
	// Account is the account standing ("Production · no enforcement").
	Account string
	Sandbox bool
	Quota   *state.Quota
	// BulkAllowed says whether the provider's terms allow bulk marketing;
	// Terms explains it ("yes, with production access").
	BulkAllowed bool
	Terms       string
	Tracking    state.Tracking
	// IPs are the provider's dedicated IPs (empty for shared pools).
	IPs         []string
	ConnectedAt time.Time
}

// ReputationFacts are the blocklist results (HostRepute or the built-in checks).
type ReputationFacts struct {
	// Status is pass, warn, block or unknown (deliverability's verdict).
	Status string
	// Via is hostrepute, builtin or none; Sources is how many lists.
	Via     string
	Sources int
	// Reason explains an unknown status; Outage is HostRepute's error
	// ("Couldn't reach api.hostrepute.com (timeout after 30 s, 3 attempts)").
	Reason    string
	Outage    string
	CheckedAt *time.Time
	// DomainSources and IPSources name the lists that check domains and IPs,
	// most important first.
	DomainSources []string
	IPSources     []string
	// Listings of the campaign's assets.
	Listings []ListingFacts
}

// ListingFacts is one asset on one blocklist.
type ListingFacts struct {
	Asset     string
	Type      string // domain, ip
	Source    string
	Impact    string // high, medium, low
	Detail    string
	DelistURL string
	Since     *time.Time
}

// AuthFacts are the authentication results of the sending domain.
type AuthFacts struct {
	Domain string
	// Checked is false until deliverability has checked the domain.
	Checked   bool
	CheckedAt *time.Time
	SPFRecord string
	SPF       []ProviderAuthFacts
	DKIM      []ProviderAuthFacts
	MailFrom  []ProviderAuthFacts
	// DMARC: status (pass, missing, invalid), policy, the record and the
	// share of aligned mail over the last 7 days (nil without reports).
	DMARCStatus  string
	DMARCPolicy  string
	DMARCRecord  string
	DMARCAligned *float64
	// BIMIStatus is pass, missing or invalid.
	BIMIStatus string
	BIMILogo   string
}

// ProviderAuthFacts is one provider's record of one kind.
type ProviderAuthFacts struct {
	Short string
	// OK is true when the record is verified (and aligned, for MAIL FROM).
	OK bool
	// Detail: "3 Easy DKIM CNAMEs verified", "selector postal-2026 verified",
	// the MAIL FROM domain, or what's wrong.
	Detail  string
	KeyBits int
	Domain  string
}

// ContentFacts are the composer's content and link checks.
type ContentFacts struct {
	Report contentcheck.ContentCheckReport
	// Checked is false when the content check couldn't run.
	Checked bool
	// FooterRemovedAt is when the version without a footer was saved.
	FooterRemovedAt *time.Time
	// Address is the postal address of the workspace branding.
	Address string
	// InlineCSSBytes is the size of the inline style attributes.
	InlineCSSBytes int
	// URIBlocklists are the lists link domains were checked on.
	URIBlocklists []string
	// OwnDomain is the registrable domain of the sending domain ("acme.io").
	OwnDomain string
	// Versions are the content checks of the language versions that are
	// sent, besides the source's (Report).
	Versions []VersionReport
}

// ComplianceFacts are what the bulk-sender rules need.
type ComplianceFacts struct {
	// UnsubscribeURL is the one-click unsubscribe link with a {token} placeholder.
	UnsubscribeURL string
	HTTPS          bool
	// ComplaintRate is the route's 7-day complaint rate in percent.
	ComplaintRate *float64
}
