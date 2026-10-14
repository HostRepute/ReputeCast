package reports

import "time"

// CampaignReport is the campaign report (spec 7.7, honest analytics 5.5).
// It only ever holds aggregate numbers: no contact address or other personal
// data, so the same document serves the read-only shared link.
type CampaignReport struct {
	Campaign     ReportCampaignInfo   `json:"campaign"`
	Headline     ReportHeadline       `json:"headline"`
	Engagement   ReportEngagement     `json:"engagement"`
	ClickMap     ReportClickMap       `json:"clickMap"`
	Mailboxes    []ReportMailboxRow   `json:"mailboxes"`
	Providers    ReportProviderTable  `json:"providers"`
	Bounces      ReportBounceTable    `json:"bounces"`
	Unsubscribes ReportUnsubscribes   `json:"unsubscribes"`
	ABTest       *ReportABTest        `json:"abTest,omitempty"`
	Backlog      *ReportIngestBacklog `json:"backlog,omitempty"`
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
type ReportHeadline struct {
	Sent            int64                 `json:"sent"`
	SentBy          []ReportProviderCount `json:"sentBy"`
	Delivered       int64                 `json:"delivered"`
	DeliveredRate   *float64              `json:"deliveredRate,omitempty"`
	HumanOpens      int64                 `json:"humanOpens"`
	HumanOpenRate   *float64              `json:"humanOpenRate,omitempty"`
	MachineOpens    int64                 `json:"machineOpens"`
	MachineOpenRate *float64              `json:"machineOpenRate,omitempty"`
	Clicks          int64                 `json:"clicks"`
	ClickRate       *float64              `json:"clickRate,omitempty"`
	ClickToOpen     *float64              `json:"clickToOpen,omitempty"`
	Unsubscribes    int64                 `json:"unsubscribes"`
	UnsubscribeRate *float64              `json:"unsubscribeRate,omitempty"`
	Complaints      int64                 `json:"complaints"`
	ComplaintRate   *float64              `json:"complaintRate,omitempty"`
	Bounces         int64                 `json:"bounces"`
	HardBounces     int64                 `json:"hardBounces"`
	SoftBounces     int64                 `json:"softBounces"`
	BounceRate      *float64              `json:"bounceRate,omitempty"`
	Rejected        int64                 `json:"rejected"`
	// Undelivered were neither delivered nor bounced: deferrals that expired
	// once the send is final, mail still in flight before that.
	Undelivered int64 `json:"undelivered"`
}

// ReportEngagementPoint is the cumulative engagement at an hour after the send.
type ReportEngagementPoint struct {
	Hour         int   `json:"hour"`
	HumanOpens   int64 `json:"humanOpens"`
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
	ID           string `json:"id"`
	URL          string `json:"url"`
	Label        string `json:"label"`
	Position     int    `json:"position"`
	Clicks       int64  `json:"clicks"`
	UniqueClicks int64  `json:"uniqueClicks"`
	// Share of all link clicks, percent.
	Share float64 `json:"share"`
	// Grouped links fall under "Other" on the map (under the threshold).
	Grouped bool `json:"grouped"`
	// Hot marks the most-clicked links (drawn in red).
	Hot bool `json:"hot"`
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
	Available bool             `json:"available"`
	Total     int64            `json:"total" doc:"All link clicks"`
	Links     []ReportLink     `json:"links"`
	Threshold float64          `json:"threshold" doc:"Links under this share (percent) are grouped as Other"`
	Other     *ReportLinkGroup `json:"other,omitempty"`
}

// ReportRates are a slice of the send's results. Rates as in ReportHeadline.
type ReportRates struct {
	Sent            int64    `json:"sent"`
	Delivered       int64    `json:"delivered"`
	DeliveredRate   *float64 `json:"deliveredRate,omitempty"`
	HumanOpenRate   *float64 `json:"humanOpenRate,omitempty"`
	MachineOpenRate *float64 `json:"machineOpenRate,omitempty"`
	ClickRate       *float64 `json:"clickRate,omitempty"`
	BounceRate      *float64 `json:"bounceRate,omitempty"`
	ComplaintRate   *float64 `json:"complaintRate,omitempty"`
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
	// first), suppressed (all of them are on the list now).
	Suppression string `json:"suppression" enum:"auto,retry,add,review,suppressed"`
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

type ReportABVariant struct {
	Key   string `json:"key"`
	Label string `json:"label" doc:"The subject line, or what differs"`
	// Recipients of the test sample.
	Recipients    int64    `json:"recipients"`
	SamplePercent float64  `json:"samplePercent"`
	HumanOpenRate *float64 `json:"humanOpenRate,omitempty"`
	ClickRate     *float64 `json:"clickRate,omitempty"`
	ComplaintRate *float64 `json:"complaintRate,omitempty"`
	Winner        bool     `json:"winner"`
	// Remaining is how many recipients got the winner after the test.
	Remaining int64 `json:"remaining"`
}

// ReportABTest is "A/B test" with the confidence of the result: a
// two-proportion z-test on the decision metric at the end of the test window.
type ReportABTest struct {
	Kind  string `json:"kind" enum:"subject,content,send_time"`
	Label string `json:"label" doc:"subject line, content, send time"`
	// Metric is human_opens or clicks; machine opens never count.
	Metric      string     `json:"metric" enum:"human_opens,clicks"`
	WindowHours float64    `json:"windowHours"`
	DecidedAt   *time.Time `json:"decidedAt,omitempty"`
	// State: decided (a winner went out), testing (the window is open) or
	// no_winner (the window closed without a significant difference).
	State    string            `json:"state" enum:"decided,testing,no_winner"`
	Winner   string            `json:"winner,omitempty"`
	Variants []ReportABVariant `json:"variants"`
	// Lift is the leader's advantage on the metric, in percentage points.
	Lift *float64 `json:"lift,omitempty"`
	// PValue of the two-sided z-test; Confidence is 1 - p, in percent.
	PValue      *float64 `json:"pValue,omitempty"`
	Confidence  *float64 `json:"confidence,omitempty"`
	Significant bool     `json:"significant" doc:"p below 0.05"`
	// Remaining recipients who got the winner, and when.
	Remaining int64      `json:"remaining"`
	RolloutAt *time.Time `json:"rolloutAt,omitempty"`
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
}

// ReportShareList lists a campaign's active links.
type ReportShareList struct {
	Items []ReportShareLink `json:"items"`
}

// SharedCampaignReport is what a read-only link shows: the report of one
// campaign and the workspace it belongs to, nothing else.
type SharedCampaignReport struct {
	Workspace string         `json:"workspace"`
	ExpiresAt *time.Time     `json:"expiresAt,omitempty"`
	Report    CampaignReport `json:"report"`
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
