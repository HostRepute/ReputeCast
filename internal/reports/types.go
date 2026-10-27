package reports

import (
	"time"

	"github.com/hostrepute/reputecast/internal/languages"
)

// CampaignReport is the campaign report (spec 7.7, honest analytics 5.5).
// It only ever holds aggregate numbers: no contact address or other personal
// data, so the same document serves the read-only shared link.
type CampaignReport struct {
	Campaign ReportCampaignInfo `json:"campaign"`
	Headline ReportHeadline     `json:"headline"`
	// Comparisons are the tiles' changes against the median of the last
	// issues (review AN-11).
	Comparisons ReportComparisons `json:"comparisons"`
	// Growth is the issue's new subscribers and web views (review AN-12);
	// its upgrades and revenue are the campaign's revenue (Earn).
	Growth       ReportGrowth         `json:"growth"`
	Engagement   ReportEngagement     `json:"engagement"`
	ClickMap     ReportClickMap       `json:"clickMap"`
	Mailboxes    []ReportMailboxRow   `json:"mailboxes"`
	Providers    ReportProviderTable  `json:"providers"`
	Bounces      ReportBounceTable    `json:"bounces"`
	Unsubscribes ReportUnsubscribes   `json:"unsubscribes"`
	ABTest       *ReportABTest        `json:"abTest,omitempty"`
	Backlog      *ReportIngestBacklog `json:"backlog,omitempty"`
	// Languages is "Results per language", for a campaign sent in more than
	// one language.
	Languages *languages.ReportLanguages `json:"languages,omitempty"`
	// NotSent is set for campaigns that haven't sent anything yet.
	NotSent *ReportNotSent `json:"notSent,omitempty"`
}

// ReportCampaignInfo is the header of the report.
type ReportCampaignInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status" enum:"draft,approval,approved,scheduled,sending,paused,sent,cancelled,failed"`
	// Audience is a readable label ("All subscribers", "Weekly Digest").
	Audience    string     `json:"audience"`
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	// Timezone of the schedule, when one was set.
	Timezone string `json:"timezone,omitempty"`
	// Origin is the importer source for history pulled in by migration ("brevo").
	Origin string `json:"origin,omitempty"`
	// Final is true once 72 hours have passed since the send ended; until
	// then opens and clicks are still coming in.
	Final bool `json:"final"`
	// FinalAt is when the results become final (72 h after the send ended).
	FinalAt *time.Time `json:"finalAt,omitempty"`
	// Notes: "A/B subject, winner B", "first send after Brevo migration".
	Notes []string `json:"notes"`
}

// ReportProviderCount is one sending provider's part of a count ("SES 43,182").
type ReportProviderCount struct {
	ProviderID string `json:"providerId"`
	Short      string `json:"short"`
	Count      int64  `json:"count"`
}

// ReportHeadline holds the headline tiles. Rates are percentages: delivered
// and bounces of sent; opens, clicks, unsubscribes and complaints of
// delivered; click-to-open of human opens.
//
// Opens are in three buckets (review AN-10): people's (HumanOpens),
// privacy-protected (PrivacyOpens: Apple Mail opens every message on
// delivery, so whether anyone read it can't be measured) and scanners'
// (MachineOpens, left out). Measurable are the delivered recipients whose
// opens can be measured (all but the privacy-protected), and
// HumanOpenRateMeasurable is human opens of them: "41.2% of delivered ·
// 52.8% of measurable".
type ReportHeadline struct {
	Sent                    int64                 `json:"sent"`
	SentBy                  []ReportProviderCount `json:"sentBy"`
	Delivered               int64                 `json:"delivered"`
	DeliveredRate           *float64              `json:"deliveredRate,omitempty"`
	HumanOpens              int64                 `json:"humanOpens"`
	HumanOpenRate           *float64              `json:"humanOpenRate,omitempty"`
	Measurable              int64                 `json:"measurable"`
	HumanOpenRateMeasurable *float64              `json:"humanOpenRateMeasurable,omitempty"`
	PrivacyOpens            int64                 `json:"privacyOpens"`
	PrivacyOpenRate         *float64              `json:"privacyOpenRate,omitempty"`
	MachineOpens            int64                 `json:"machineOpens" doc:"Scanner opens, left out of every rate"`
	MachineOpenRate         *float64              `json:"machineOpenRate,omitempty"`
	Clicks                  int64                 `json:"clicks"`
	ClickRate               *float64              `json:"clickRate,omitempty"`
	ClickToOpen             *float64              `json:"clickToOpen,omitempty"`
	// BotClicks are the clicks on the email's links the bot filter left
	// out (scanners following every link, the hidden link's scanners).
	BotClicks       int64    `json:"botClicks"`
	Unsubscribes    int64    `json:"unsubscribes"`
	UnsubscribeRate *float64 `json:"unsubscribeRate,omitempty"`
	Complaints      int64    `json:"complaints"`
	ComplaintRate   *float64 `json:"complaintRate,omitempty"`
	// ComplaintRateReported is complaints of the messages delivered to
	// mailbox providers that report complaints; NotReportedBy names the
	// ones in this send that don't ("Gmail", "iCloud", review F-2).
	ComplaintRateReported *float64 `json:"complaintRateReported,omitempty"`
	NotReportedBy         []string `json:"notReportedBy"`
	Bounces               int64    `json:"bounces"`
	HardBounces           int64    `json:"hardBounces"`
	SoftBounces           int64    `json:"softBounces"`
	BounceRate            *float64 `json:"bounceRate,omitempty"`
	// Rejected: the providers refused them, when they were handed over or
	// afterwards; they were never sent.
	Rejected int64 `json:"rejected"`
	// Undelivered were sent and neither delivered nor bounced: deferrals
	// that expired once the send is final, mail still in flight before
	// that.
	Undelivered int64 `json:"undelivered"`
}

// ReportEngagementPoint is the cumulative engagement at an hour after the send.
type ReportEngagementPoint struct {
	Hour         int   `json:"hour"`
	HumanOpens   int64 `json:"humanOpens"`
	PrivacyOpens int64 `json:"privacyOpens"`
	MachineOpens int64 `json:"machineOpens"`
	Clicks       int64 `json:"clicks"`
}

// ReportRollout is the winning A/B variant going out to everyone else.
type ReportRollout struct {
	Variant string    `json:"variant"`
	At      time.Time `json:"at"`
	// Hour after the send started.
	Hour float64 `json:"hour"`
}

// ReportEngagement is "Engagement over the first 72 hours": cumulative
// counts per hour, as shares of delivered on the chart.
type ReportEngagement struct {
	// Available is false when there is no hourly data (imported history).
	Available bool                    `json:"available"`
	Points    []ReportEngagementPoint `json:"points"`
	// Hours of data so far (72 once final).
	Hours int `json:"hours"`
	// Delivered is the base the chart's percentages use.
	Delivered int64 `json:"delivered"`
	// HumanWithin6h is the share of the 72-hour human opens that came in
	// the first 6 hours (percent).
	HumanWithin6h *float64       `json:"humanWithin6h,omitempty"`
	Rollout       *ReportRollout `json:"rollout,omitempty"`
}

// ReportLink is one tracked link of the email.
type ReportLink struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Label    string `json:"label"`
	Position int    `json:"position"`
	// Clicks are every human click on the link; UniqueClicks the people who
	// clicked it, which the map and its shares count.
	Clicks       int64 `json:"clicks"`
	UniqueClicks int64 `json:"uniqueClicks"`
	// BotClicks landed on this link and were left out of its clicks.
	BotClicks int64 `json:"botClicks"`
	// Share of the map's human clicks (people per link), percent.
	Share float64 `json:"share"`
	// Grouped links fall under "Other" on the map (under the threshold);
	// a link of one of the map's Rows isn't grouped but drawn with its row.
	Grouped bool `json:"grouped"`
	// Hot marks the most-clicked links (drawn in red).
	Hot bool `json:"hot"`
}

// ReportLinkRow is links that sit on one line of the email with only
// separators between them (a navigation bar: "Blog · Docs · Changelog"),
// each under the threshold: the map shows them as one entry and one bubble.
type ReportLinkRow struct {
	// Place is where the row is: header (at the top, after a logo at
	// most), footer (after every other link) or body.
	Place     string   `json:"place" enum:"header,footer,body"`
	Positions []int    `json:"positions"`
	Labels    []string `json:"labels"`
	Clicks    int64    `json:"clicks" doc:"Human clicks, each person once per link"`
	BotClicks int64    `json:"botClicks"`
	Share     float64  `json:"share"`
}

// ReportLinkGroup is "Other": the links under the threshold, as one bubble.
type ReportLinkGroup struct {
	Share  float64  `json:"share"`
	Links  int      `json:"links"`
	Clicks int64    `json:"clicks"`
	Labels []string `json:"labels" doc:"The most-clicked links in the group, for the caption"`
}

// ReportClickMap is the click map: the email as sent and the clicks per link.
type ReportClickMap struct {
	// HTML is the campaign's email as sent (rendered in a sandboxed frame).
	HTML string `json:"html"`
	// Available is false when the stored content is missing; the links
	// are still listed.
	Available bool         `json:"available"`
	Total     int64        `json:"total" doc:"Human clicks on the links, each person once per link"`
	BotTotal  int64        `json:"botTotal" doc:"Clicks the bot filter left out"`
	Links     []ReportLink `json:"links"`
	Threshold float64      `json:"threshold" doc:"Links under this share (percent) are grouped as Other"`
	// Rows are the links on one line of the email shown as one entry; their
	// links aren't in Other.
	Rows  []ReportLinkRow  `json:"rows"`
	Other *ReportLinkGroup `json:"other,omitempty"`
	// Language is the language version the map shows, for a campaign sent
	// in several languages.
	Language string `json:"language,omitempty"`
	// Variants are the A/B test's maps, one per variant (review AN-5): each
	// variant's recipients' clicks (its sample and, for the winner, everyone
	// who got it after the test) on the variant's email. Empty without a
	// test, or when the clicks weren't kept per variant.
	Variants []ReportVariantClickMap `json:"variants"`
}

// ReportVariantClickMap is one A/B variant's click map.
type ReportVariantClickMap struct {
	Key string `json:"key"`
	// SameEmail: the variants share the email (a subject or send-time
	// test), so the map is drawn on the campaign's email (ReportClickMap's
	// HTML) and HTML is empty.
	SameEmail bool             `json:"sameEmail"`
	HTML      string           `json:"html"`
	Available bool             `json:"available"`
	Total     int64            `json:"total"`
	BotTotal  int64            `json:"botTotal"`
	Links     []ReportLink     `json:"links"`
	Rows      []ReportLinkRow  `json:"rows"`
	Other     *ReportLinkGroup `json:"other,omitempty"`
}

// ReportRates are a slice of the send's results. Rates as in ReportHeadline.
type ReportRates struct {
	Sent          int64    `json:"sent"`
	Delivered     int64    `json:"delivered"`
	DeliveredRate *float64 `json:"deliveredRate,omitempty"`
	HumanOpenRate *float64 `json:"humanOpenRate,omitempty"`
	// Measurable and HumanOpenRateMeasurable: human opens of the recipients
	// whose opens can be measured; PrivacyOpenRate: privacy-protected opens
	// of delivered (review AN-10).
	Measurable              int64    `json:"measurable"`
	HumanOpenRateMeasurable *float64 `json:"humanOpenRateMeasurable,omitempty"`
	PrivacyOpenRate         *float64 `json:"privacyOpenRate,omitempty"`
	MachineOpenRate         *float64 `json:"machineOpenRate,omitempty"`
	ClickRate               *float64 `json:"clickRate,omitempty"`
	BounceRate              *float64 `json:"bounceRate,omitempty"`
	ComplaintRate           *float64 `json:"complaintRate,omitempty"`
	// ComplaintRateNA says why ComplaintRate is empty: not_reported (the
	// mailbox provider doesn't report complaints to senders, review F-2).
	ComplaintRateNA string   `json:"complaintRateNa,omitempty" enum:"not_reported"`
	UnsubscribeRate *float64 `json:"unsubscribeRate,omitempty"`
}

// ReportMailboxRow is one mailbox provider (where recipients read it).
type ReportMailboxRow struct {
	Key  string `json:"key" enum:"gmail,outlook,yahoo,icloud,other"`
	Name string `json:"name"`
	ReportRates
}

// ReportProviderColumn is one sending provider in the side-by-side table.
type ReportProviderColumn struct {
	ProviderID string `json:"providerId"`
	Name       string `json:"name"`
	Short      string `json:"short"`
	Adapter    string `json:"adapter"`
	Color      string `json:"color" enum:"ses,postal,postmark,other"`
	ReportRates
	// Percent of the planned send routed here.
	Percent float64 `json:"percent"`
	// MedianDeliveryMs is the median time from handing a message over to its
	// delivery event.
	MedianDeliveryMs *int `json:"medianDeliveryMs,omitempty"`
	// Cost in dollars; absent when unknown (history imported from another
	// platform, which ReputeCast didn't pay for).
	Cost *float64 `json:"cost,omitempty" doc:"Dollars"`
	// WarmupDay is the warm-up day the provider was on at the send.
	WarmupDay *int `json:"warmupDay,omitempty"`
	// EngagedFirst is true when the warm-up sent to the most engaged contacts only.
	EngagedFirst bool `json:"engagedFirst"`
}

// ReportProviderTable is "Results per sending provider".
type ReportProviderTable struct {
	Columns []ReportProviderColumn `json:"columns"`
	// Split is true when more than one provider delivered the send.
	Split bool `json:"split"`
}

// ReportCodeCount is one provider's original code within a bounce type.
type ReportCodeCount struct {
	Provider string `json:"provider" doc:"Short name: SES, Postal"`
	Type     string `json:"type" doc:"The provider's event name: Permanent/General, HardFail"`
	Code     string `json:"code" doc:"SMTP reply or provider code: 550 5.1.1"`
	Count    int64  `json:"count"`
}

// ReportBounceRow is one type of the shared bounce taxonomy.
type ReportBounceRow struct {
	Type  string            `json:"type" enum:"unknown_user,domain_invalid,mailbox_full,deferral_expired,blocked,other"`
	Label string            `json:"label"`
	Tone  string            `json:"tone" enum:"bad,warn,neutral"`
	Count int64             `json:"count"`
	Codes []ReportCodeCount `json:"codes"`
	// Suppression: auto (every address was suppressed when it bounced),
	// retry (soft bounces are retried next send), add (addresses aren't
	// suppressed; a person can add them), review (look at the diagnostics
	// first), suppressed (all of them are on the list now), investigate (a
	// block for the sending IP or domain: the addresses are valid and stay
	// on the list; never suppressed, review AN-7).
	Suppression string `json:"suppression" enum:"auto,retry,add,review,suppressed,investigate"`
	// Unsuppressed is how many bounced addresses aren't on the list.
	Unsuppressed int64 `json:"unsuppressed"`
}

// ReportBounceTable is "Bounces by type".
type ReportBounceTable struct {
	Total int64             `json:"total"`
	Rows  []ReportBounceRow `json:"rows"`
}

// ReportUnsubscribeReason is one answer on the unsubscribe page.
type ReportUnsubscribeReason struct {
	Key   string  `json:"key" enum:"too_frequent,not_relevant,none,never_signed_up,spam,other"`
	Label string  `json:"label"`
	Count int64   `json:"count"`
	Share float64 `json:"share" doc:"Share of unsubscribes, percent"`
	Tone  string  `json:"tone" enum:"ink,muted,bad"`
}

// ReportUnsubscribes is "Unsubscribe reasons".
type ReportUnsubscribes struct {
	Total   int64                     `json:"total"`
	Reasons []ReportUnsubscribeReason `json:"reasons"`
	// ChoseFewer chose "fewer emails" instead of leaving.
	ChoseFewer int64 `json:"choseFewer"`
}

// ReportABVariant is one variant of the A/B test: its test sample's
// results (the recipients who got it after the test are counted apart, as
// the remaining).
type ReportABVariant struct {
	Key   string `json:"key"`
	Label string `json:"label" doc:"The subject line, or what differs"`
	// Default is the variant the rest get when the test has no clear
	// winner (A).
	Default bool `json:"default"`
	// Recipients of the test sample.
	Recipients    int64   `json:"recipients"`
	SamplePercent float64 `json:"samplePercent"`
	// The sample's results so far (final after 72 hours): human opens of
	// the measurable recipients, clicks and complaints of delivered.
	HumanOpenRate *float64 `json:"humanOpenRate,omitempty"`
	ClickRate     *float64 `json:"clickRate,omitempty"`
	ComplaintRate *float64 `json:"complaintRate,omitempty"`
	// MetricRate is the rate on the test's metric now (the bar).
	MetricRate *float64 `json:"metricRate,omitempty"`
	// DecisionRate is the rate on the metric the decision was made on.
	DecisionRate *float64 `json:"decisionRate,omitempty"`
	// Eligible and Conversions are what the decision counted: recipients
	// whose result could be measured, and how many converted.
	Eligible    int64 `json:"eligible"`
	Conversions int64 `json:"conversions"`
	// Winner is the variant the rest got (the leader, or the default
	// without a clear winner).
	Winner bool `json:"winner"`
	// Remaining is how many recipients got this variant after the test.
	Remaining int64 `json:"remaining"`
}

// ReportABTest is "A/B test" with the confidence of the result: the
// sending engine's decision (internal/abstats: at least MinSample
// recipients per variant, a two-proportion z-test between the leader and
// the runner-up at the Threshold confidence; otherwise the default goes
// out), and each sample's results since.
type ReportABTest struct {
	Kind  string `json:"kind" enum:"subject,content,send_time"`
	Label string `json:"label" doc:"subject line, content, send time"`
	// Metric is human_opens (of measurable recipients) or clicks; machine
	// and privacy-protected opens never count.
	Metric      string     `json:"metric" enum:"human_opens,clicks"`
	WindowHours float64    `json:"windowHours"`
	DecidedAt   *time.Time `json:"decidedAt,omitempty"`
	// DecidedAfter is how many hours after the sample went out it was
	// decided.
	DecidedAfter *float64 `json:"decidedAfter,omitempty"`
	// State: decided (a winner went out), testing (the window is open) or
	// no_winner (no clear winner: the default went out).
	State string `json:"state" enum:"decided,testing,no_winner"`
	// Outcome is the decision: winner, not_significant, too_small or
	// manual (a person picked); empty while testing.
	Outcome string `json:"outcome,omitempty" enum:"winner,not_significant,too_small,manual,"`
	// Winner is the variant the rest got; Default the test's default (A).
	Winner   string            `json:"winner,omitempty"`
	Default  string            `json:"default"`
	Variants []ReportABVariant `json:"variants"`
	// Leader and RunnerUp of the decision (or of the test so far).
	Leader   string `json:"leader,omitempty"`
	RunnerUp string `json:"runnerUp,omitempty"`
	// Lift is the leader's advantage on the metric, in percentage points.
	Lift *float64 `json:"lift,omitempty"`
	// PValue of the two-sided z-test; Confidence is 1 - p, in percent.
	PValue      *float64 `json:"pValue,omitempty"`
	Confidence  *float64 `json:"confidence,omitempty"`
	Significant bool     `json:"significant" doc:"Confidence reached the threshold, with enough recipients per variant"`
	// MinSample is the fewest recipients each variant needed; Threshold the
	// confidence a winner needed, in percent (95).
	MinSample int64   `json:"minSample"`
	Threshold float64 `json:"threshold"`
	// Remaining recipients who got the variant after the test, and when.
	Remaining int64      `json:"remaining"`
	RolloutAt *time.Time `json:"rolloutAt,omitempty"`
	// BarMax is the top of the bars' scale, in percent ("on a 0 to 10%
	// scale").
	BarMax float64 `json:"barMax"`
}

// ReportBacklogProvider is one provider whose events haven't been processed.
type ReportBacklogProvider struct {
	ProviderID string    `json:"providerId"`
	Name       string    `json:"name"`
	Short      string    `json:"short"`
	Feed       string    `json:"feed" doc:"How its events arrive: SNS, webhook, bounce mailbox"`
	Waiting    int64     `json:"waiting"`
	Since      time.Time `json:"since"`
}

// ReportIngestBacklog says that part of the report is behind: provider
// events wait to be processed, so counts they feed are as of Since.
type ReportIngestBacklog struct {
	Since     time.Time               `json:"since"`
	Minutes   int                     `json:"minutes"`
	Providers []ReportBacklogProvider `json:"providers"`
	// Current names the other providers, whose events are up to date.
	Current []string `json:"current"`
}

// ReportNotSent is the empty state of a campaign without results.
type ReportNotSent struct {
	Status      string     `json:"status"`
	ScheduledAt *time.Time `json:"scheduledAt,omitempty"`
	Timezone    string     `json:"timezone,omitempty"`
	Readiness   string     `json:"readiness,omitempty" enum:"ready,warnings,blocked,"`
	// LastReport is the most recent campaign with results.
	LastReport *ReportCampaignRef `json:"lastReport,omitempty"`
}

// ReportCampaignRef points to another campaign.
type ReportCampaignRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ReportShareLink is an active read-only link.
type ReportShareLink struct {
	ID string `json:"id"`
	// URL is the public link, shown to members who may share reports.
	URL           string     `json:"url"`
	CreatedAt     time.Time  `json:"createdAt"`
	CreatedByName string     `json:"createdByName"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	Views         int64      `json:"views"`
	LastViewedAt  *time.Time `json:"lastViewedAt,omitempty"`
	// Includes is what the link shares beyond the numbers.
	Includes ReportShareIncludes `json:"includes"`
}

// ReportShareIncludes is what a read-only link shares beyond the numbers,
// charts and tables (review AN-16).
type ReportShareIncludes struct {
	Cost   bool `json:"cost" doc:"The cost per sending provider"`
	Warmup bool `json:"warmup" doc:"The warm-up day of each sending provider"`
	Email  bool `json:"email" doc:"The email as sent, in the click map"`
}

// ReportShareList lists a campaign's active links.
type ReportShareList struct {
	Items []ReportShareLink `json:"items"`
}

// SharedCampaignReport is what a read-only link shows: the report of one
// campaign and the workspace it belongs to, nothing else.
type SharedCampaignReport struct {
	Workspace string              `json:"workspace"`
	ExpiresAt *time.Time          `json:"expiresAt,omitempty"`
	Includes  ReportShareIncludes `json:"includes"`
	Report    CampaignReport      `json:"report"`
}

// ReportBounceDiagnostic is one provider message behind a bounce type.
type ReportBounceDiagnostic struct {
	Provider   string `json:"provider"`
	Type       string `json:"type"`
	Code       string `json:"code"`
	Diagnostic string `json:"diagnostic"`
	Count      int64  `json:"count"`
}

// ReportBounceReview is the Review dialog of a bounce type.
type ReportBounceReview struct {
	Type         string                   `json:"type"`
	Label        string                   `json:"label"`
	Count        int64                    `json:"count"`
	Unsuppressed int64                    `json:"unsuppressed"`
	Diagnostics  []ReportBounceDiagnostic `json:"diagnostics"`
}

// ReportSuppressResult says how many bounced addresses were suppressed.
type ReportSuppressResult struct {
	Added   int `json:"added"`
	Already int `json:"already" doc:"Already on the list"`
}
