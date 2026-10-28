package readiness

import (
	"time"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/sending/engine"
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
	// Sponsor is the email's Sponsor block, UpgradeOffers the offers its
	// Upgrade blocks still carry after they ended (Earn, design expansion X17).
	Sponsor       SponsorFacts
	UpgradeOffers []EndedUpgradeOffer
	// MailboxRates are the per-mailbox-provider throttles (messages per
	// second, 0 = full rate).
	MailboxRates map[string]float64
	// Grow are the Referral and Recommendations blocks' rows (grow.go).
	Grow []GrowRow
	// Rules are the workspace's sending rules for the send's stream
	// (rules.go, SR-10).
	Rules []RuleFacts
	// Confirm is what makes the send unusual enough for a typed
	// confirmation (confirm.go, SR-13).
	Confirm ConfirmFacts
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
	// HTMLOnly is true for a campaign written in HTML (no MJML): fixes
	// that edit MJML aren't offered.
	HTMLOnly bool
	// ABKind is the kind of the campaign's A/B test (subject, content or
	// send_time), empty without one; the variants' checks are
	// ContentFacts.Variants.
	ABKind string
}

// Kinds of audience a base label names (AudienceFacts.BaseKind).
const (
	BaseAll     = "all"     // every public list: "All subscribers"
	BaseList    = "list"    // one list: "Weekly Digest list"
	BaseSegment = "segment" // one segment: "Customers segment"
	BaseNone    = "none"    // nothing named: "Audience"
	BaseMany    = "many"    // several: "Weekly Digest + Product Updates"
)

// AudienceFacts are the audience counts after exclusions.
type AudienceFacts struct {
	// BaseLabel names the audience ("Weekly Digest list", "All subscribers").
	BaseLabel string
	// GroupLabel is the caption of the audience-quality group ("Weekly Digest").
	GroupLabel string
	// BaseKind says what BaseLabel names and BaseName the list or segment,
	// so the labels read in each reader's language; empty shows the labels
	// as they are.
	BaseKind string
	BaseName string
	Lists    []string
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
	// NoFirstName counts final recipients without a first name (what a
	// first-name tag's fallback is for).
	NoFirstName int64
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
	// Paid describes a paid-only campaign's recipients (paid.go); nil for
	// everyone on the lists.
	Paid *PaidAudienceFacts `json:"-"`
	// Retry describes a retry's recipients (retry.go); nil when the
	// campaign isn't a retry.
	Retry *RetryAudienceFacts `json:"-"`
}

// RetryAudienceFacts are a retry's: the issue it sends again, and how many
// of its addresses were blocked for the sender's reputation and haven't had
// a retry yet (before the audience's own rules leave some out).
type RetryAudienceFacts struct {
	Issue   string
	Blocked int64
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
	// Fallback is the route's fallback provider, nil without one.
	Fallback *FallbackFacts
	// Estimate is the send played out with the sending engine's rules
	// (SR-5, C4); nil when there is no engine to ask.
	Estimate *engine.Estimate
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
	// connections since 08:52 (ECONNREFUSED on port 25 and 443)"); Down says
	// it in parts, so it reads in each reader's language.
	DownSince  *time.Time
	DownDetail string
	Down       *DownFacts
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
	// ProviderReported is true when the provider measured them. Sent7d is
	// the volume ReputeCast's rates count: below MinRateVolume they can't
	// judge a provider (SR-5).
	Bounce           *float64
	Complaint        *float64
	ProviderReported bool
	Sent7d           int64
	Limits           state.Limits
	// Account is the account standing ("Production · no enforcement").
	Account string
	Sandbox bool
	Quota   *state.Quota
	// BulkAllowed says whether the provider's terms allow bulk marketing;
	// Terms explains it ("yes, with production access") and TermsKind names
	// the explanation (terms.<kind> in the messages; empty shows Terms).
	BulkAllowed bool
	Terms       string
	TermsKind   string
	Tracking    state.Tracking
	// IPs are the provider's dedicated IPs (empty for shared pools).
	IPs         []string
	ConnectedAt time.Time
	// OneClick is how the provider carries the one-click unsubscribe
	// headers (providers.Capabilities.OneClick): "" as given, "replaced"
	// or "unsigned".
	OneClick string
}

// FallbackFacts is the provider the route fails over to and what
// state.Vet found about it taking the whole send (C11).
type FallbackFacts struct {
	ProviderID string
	Name       string
	// Failed is the first vetting check that failed, empty when the
	// fallback can take the mail; Text says it in each reader's language.
	Failed string
	Text   Text
}

// DownFacts is why a provider can't take mail: since when, the host or
// provider it's about, and either rejected credentials (Auth) or refused
// connections with the error code (Code).
type DownFacts struct {
	Since time.Time
	Who   string
	Auth  bool
	Code  string
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
	// ReasonText is Reason when Send Readiness wrote it itself.
	Reason     string
	ReasonText Text
	Outage     string
	CheckedAt  *time.Time
	// DomainSources and IPSources name the lists that check domains and IPs,
	// most important first.
	DomainSources []string
	IPSources     []string
	// Listings of the campaign's assets.
	Listings []ListingFacts
	// Domains are the domains the send's reputation rides on: the sending
	// domain, its parent (organizational) domain, the providers' return-path
	// domains and the tracking domain (SR-2). Empty: the sending domain only.
	Domains []DomainAssetFacts
	// ReverseDNS is the reverse DNS of the dedicated IPs carrying the send.
	ReverseDNS []ReverseDNSFacts
}

// ReverseDNSFacts is the forward-confirmed reverse DNS of a dedicated IP.
type ReverseDNSFacts struct {
	IP       string
	Provider string
	// PTR is the IP's reverse DNS name; Status is pass (it resolves back to
	// the IP), fail (it doesn't), missing (no PTR) or error (DNS didn't
	// answer: Trouble says how, a Trouble* kind).
	PTR     string
	Status  string
	Trouble string
}

// Roles of the domains a send's reputation rides on (DomainAssetFacts.Role).
const (
	DomainSending    = "sending"     // news.acme.io, the From address's domain
	DomainParent     = "parent"      // acme.io, its organizational domain
	DomainReturnPath = "return_path" // bounce.news.acme.io, a provider's custom MAIL FROM
	DomainTracking   = "tracking"    // the domain of the open pixel and the tracked links
)

// DomainAssetFacts is one domain of a send's reputation.
type DomainAssetFacts struct {
	Domain string
	Role   string
	// Providers are the providers whose return path the domain is.
	Providers []string
	// Live is true when Send Readiness looked the domain up itself on the
	// URI blocklists (Deliverability doesn't watch it): Lists are the lists
	// it is on, Unknown why it isn't known (a contentcheck.Trouble* kind, or
	// UnknownUnchecked when nothing could look it up).
	Live    bool
	Lists   []string
	Unknown string
}

// UnknownUnchecked is why a domain of the send's reputation is unknown when
// Deliverability doesn't watch it and the run can't look it up live.
const UnknownUnchecked = "unchecked"

// ListingFacts is one asset on one blocklist.
type ListingFacts struct {
	Asset     string
	Type      string // domain, ip
	Source    string
	Impact    string // high, medium, low
	Detail    string
	DelistURL string
	Since     *time.Time
	// Answer and TXT are the list's DNS answers to the lookup when they
	// are known (the technical details' raw DNS, SR-13).
	Answer, TXT string
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
	// BIMIStatus is pass, missing, invalid or error.
	BIMIStatus string
	BIMILogo   string
	// DMARCError and BIMIError say why a lookup failed (Trouble* kinds):
	// the record's state is unknown, not wrong. BIMIProblem is what's wrong
	// with an invalid BIMI record ("l= must be an https URL…").
	DMARCError string
	BIMIError  string
	// SPFError says why the domain's own SPF record couldn't be read.
	SPFError    string
	BIMIProblem string
}

// What a provider's record detail says (ProviderAuthFacts.DetailKind).
const (
	AuthCNAMEs     = "cnames"     // CNAMEs Easy DKIM CNAMEs verified
	AuthSelector   = "selector"   // selector Selector verified
	AuthUnverified = "unverified" // DKIM records not verified
	AuthSPFAdd     = "spf_add"    // add Value to the SPF record of Name
)

// ProviderAuthFacts is one provider's record of one kind.
type ProviderAuthFacts struct {
	Short string
	// OK is true when the record is verified (and aligned, for MAIL FROM).
	OK bool
	// Detail: "3 Easy DKIM CNAMEs verified", "selector postal-2026 verified",
	// the MAIL FROM domain, or what's wrong. DetailKind says which, with
	// its parts (CNAMEs, Selector, Value and Name), so it reads in each
	// reader's language; empty shows Detail as it is.
	Detail     string
	DetailKind string
	CNAMEs     int
	Selector   string
	Value      string
	Name       string
	KeyBits    int
	Domain     string
	// Unknown is true when DNS didn't answer for the record (even after the
	// run looked it up again); Trouble says how (Trouble* kinds).
	Unknown bool
	Trouble string
}

// ContentFacts are the composer's content and link checks.
type ContentFacts struct {
	Report contentcheck.ContentCheckReport
	// FooterText is the text of the email's footer (workspace rules on
	// what it shows).
	FooterText string
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
	// Variants are the content checks of an A/B test's variants B and C
	// (the campaign itself is variant A).
	Variants []VariantReport
	// TrackedBytes is the HTML's size as sent, with the links rewritten for
	// tracking and the open pixel added (0 when nothing is tracked or the
	// email couldn't be prepared).
	TrackedBytes int
	// TrackOpens and TrackClicks say what the campaign tracks;
	// TrackingHost is the domain of the open pixel and the tracked links,
	// TrackingHTTPS whether they use HTTPS, and TrackingDNS whether that
	// domain resolves: ok, missing, or a Trouble* kind when DNS didn't
	// answer ("" when it wasn't looked up).
	TrackOpens, TrackClicks bool
	TrackingHost            string
	TrackingHTTPS           bool
	TrackingDNS             string
	// LibraryHost is where the media library's images load from (the
	// public URL), LibraryHTTPS whether over HTTPS.
	LibraryHost  string
	LibraryHTTPS bool
	// Fields are the personalization tags of the email (every version and
	// variant), each field once.
	Fields []FieldUse
}

// FieldUse is a personalization field the email uses, and its fallback.
type FieldUse struct {
	Field       string
	HasFallback bool
	Fallback    string
}

// VariantReport is the content check of one A/B variant.
type VariantReport struct {
	Key       string
	Subject   string
	Preheader string
	// Content is true for a content test's variant: its own body.
	Content bool
	Report  contentcheck.ContentCheckReport
}

// ComplianceFacts are what the bulk-sender rules need.
type ComplianceFacts struct {
	// UnsubscribeURL is the one-click unsubscribe link with a {token} placeholder.
	UnsubscribeURL string
	HTTPS          bool
	// ComplaintRate is the route's 7-day complaint rate in percent.
	ComplaintRate *float64
	// Gmail is Gmail Postmaster Tools' latest day for the sending domain
	// (or its parent), nil when there is none. PostmasterConnected says
	// Postmaster data comes in for the workspace at all.
	Gmail               *PostmasterFacts
	PostmasterConnected bool
}

// PostmasterFacts is what Gmail Postmaster Tools reported for a domain.
type PostmasterFacts struct {
	Domain string
	// Day is the latest day Google reported; SpamRate is that day's
	// user-reported spam rate in percent (nil when Google had too little
	// volume to report one).
	Day      time.Time
	SpamRate *float64
	// SpamDay is the day of SpamRate when it isn't Day (the latest day
	// with a rate).
	SpamDay *time.Time
	// SPF, DKIM, DMARC and TLS are Day's pass rates in percent, nil when
	// Google didn't report one.
	SPF, DKIM, DMARC, TLS *float64
}

// PostmasterFloor is the pass rate (percent) under which Gmail Postmaster's
// authentication and TLS rates make its compliance status "Needs work",
// and PostmasterSpamLimit the user-reported spam rate at which it does
// (Google's bulk-sender rules, SR-12).
const (
	PostmasterFloor     = 98.0
	PostmasterSpamLimit = 0.3
)
