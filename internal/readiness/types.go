// Package readiness is Send Readiness (spec 5.1, 7.5): every check a
// campaign must pass before it goes out, each Pass, Warn, Block or Unknown
// with its evidence and a fix, plus the estimates around the send (final
// audience, route, cost and duration, throttling per mailbox provider).
//
// A run gathers facts from the other domains and Assemble turns them into
// checks. Assemble is pure, so the demo seeds runs through it and tests feed
// it fixtures. Runs are stored in readiness_runs; the run in progress is in
// readiness_progress so the review screen can tick the groups in.
package readiness

import "time"

// Verdicts.
const (
	VerdictReady    = "ready"
	VerdictWarnings = "warnings"
	VerdictBlocked  = "blocked"
)

// Check statuses.
const (
	Pass    = "pass"
	Warn    = "warn"
	Block   = "block"
	Unknown = "unknown"
)

// Check groups, in the order of the checklist.
const (
	GroupReputation     = "reputation"
	GroupAuthentication = "authentication"
	GroupContent        = "content"
	GroupLinks          = "links"
	GroupCompliance     = "compliance"
	GroupAudience       = "audience"
	GroupRouting        = "routing"
)

// Groups lists the check groups in checklist order.
var Groups = []string{GroupReputation, GroupAuthentication, GroupContent, GroupLanguages, GroupAccessibility, GroupConditional, GroupLinks, GroupCompliance, GroupAudience, GroupRouting,
	GroupRules}

// Triggers of a run (readiness_runs.trigger).
const (
	TriggerManual   = "manual"
	TriggerComposer = "composer"
	TriggerSchedule = "schedule"
	TriggerSendTime = "send_time"
	TriggerApproval = "approval"
)

// Fix actions applied by POST …/readiness/fixes.
const (
	FixExcludeSunset = "exclude_sunset"
	// FixIncludeUnengaged puts back the unengaged contacts a send leaves
	// out ("Include them" on the audience, SR-11).
	FixIncludeUnengaged = "include_unengaged"
	FixRouteOnly        = "route_only"
	FixInsertFooter     = "insert_footer"
	FixReplaceLink      = "replace_link"
	FixUseSubject       = "use_subject"
	FixMinify           = "minify"
)

// ReadinessFix is one fix offered with a check: an action ReputeCast applies
// in one click (Kind "apply", Action and Params) or a link to the screen
// where it's fixed (Kind "link", Href is an app path).
type ReadinessFix struct {
	Kind   string            `json:"kind" enum:"apply,link"`
	Action string            `json:"action,omitempty" enum:"rule_add_param,exclude_sunset,route_only,insert_footer,replace_link,use_subject,minify,mark_language_up_to_date,mark_language_translated,hold_language,release_language,accessibility_decorative,accessibility_color,accessibility_one_click,visibility_show_everyone,visibility_hide_block,visibility_remove_condition,poll_close_days,exclude_risky,verify_addresses,sponsor_remove_block,sponsor_refresh_block"`
	Label  string            `json:"label"`
	Tone   string            `json:"tone,omitempty" enum:"primary,danger"`
	Href   string            `json:"href,omitempty"`
	Params map[string]string `json:"params,omitempty"`
}

// ReadinessCheck is one row of the checklist.
type ReadinessCheck struct {
	// Key identifies the check: "audience.unengaged", "reputation.provider.ses".
	Key    string `json:"key"`
	Group  string `json:"group" enum:"reputation,authentication,content,languages,links,compliance,audience,routing,accessibility,conditional,rules"`
	Status string `json:"status" enum:"pass,warn,block,unknown"`
	// Title states the finding; FlatTitle is the same finding prefixed with
	// its group, for lists ordered by severity ("Links: go.acme-promo.link…").
	Title     string `json:"title"`
	FlatTitle string `json:"flatTitle"`
	// Detail is the evidence. Mono is true when it is a record or a header.
	Detail string `json:"detail,omitempty"`
	Mono   bool   `json:"mono,omitempty"`
	// Label names the check in compact tables ("URL shortener"); Evidence is
	// a one-sentence version of title and detail for them.
	Label    string `json:"label,omitempty"`
	Evidence string `json:"evidence,omitempty"`
	// Short is the passing finding in a few words, used in group summaries.
	Short string `json:"short,omitempty"`
	// Ack is the sentence a person confirms before sending with this warning
	// ("I understand 412 unengaged contacts are included.").
	Ack string `json:"ack,omitempty"`
	// AckKey is the same for warnings confirmed with the same sentence,
	// whatever language it is read in: they are confirmed together.
	AckKey string `json:"ackKey,omitempty"`
	// Note is a short right-aligned note ("timed out").
	Note  string         `json:"note,omitempty"`
	Fixes []ReadinessFix `json:"fixes"`
	// Rows are a small table of the finding ("Recipients per language":
	// English 26,890 gets English · …).
	Rows []ReadinessCheckRow `json:"rows,omitempty"`
	// Technical are the check's technical details (a workspace rule, what
	// it checks and what broke it; a blocklist and its scope), and Raw the
	// records, answers or addresses behind them, shown as they are.
	Technical []ReadinessTechRow `json:"technical,omitempty"`
	Raw       []string           `json:"raw,omitempty"`

	// texts are the messages the texts above are written from (stored.go).
	texts *checkTexts
}

// ReadinessTechRow is a line of a check's technical details.
type ReadinessTechRow struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// ReadinessCheckRow is one line of a check's table.
type ReadinessCheckRow struct {
	Label  string `json:"label"`
	Count  int64  `json:"count"`
	Detail string `json:"detail"`
	// Language is the readers' language and Gets the version they get
	// (Recipients per language: Audience › By version and the
	// confirmation step read them).
	Language string `json:"language,omitempty"`
	Gets     string `json:"gets,omitempty"`
	GetsName string `json:"getsName,omitempty"`
}

// ReadinessGroup is a group heading with its caption and a summary of what
// passed (the collapsed "Passed" list and the running state show it).
type ReadinessGroup struct {
	Key     string `json:"key" enum:"reputation,authentication,content,languages,links,compliance,audience,routing,accessibility,conditional,rules"`
	Title   string `json:"title"`
	Caption string `json:"caption"`
	Status  string `json:"status" enum:"pass,warn,block,unknown"`
	Passed  int    `json:"passed"`
	Total   int    `json:"total"`
	// Summary is "SPF, DKIM, DMARC p=quarantine and custom MAIL FROM verified for SES".
	Summary string `json:"summary"`
	// Running describes the group while it runs ("checking 203.0.113.25 and
	// news.acme.io on 84 sources").
	Running string `json:"running"`
	// Note is a sentence the verdict adds when the group blocks the send
	// ("Your workspace blocks sends with serious accessibility issues…").
	Note string `json:"note,omitempty"`

	texts *groupTexts
}

// ReadinessCounts adds up the checks.
type ReadinessCounts struct {
	Passed   int `json:"passed"`
	Warnings int `json:"warnings"`
	Blocks   int `json:"blocks"`
	Unknown  int `json:"unknown"`
	Total    int `json:"total"`
}

// ReadinessAudienceRow is one line of the audience card.
type ReadinessAudienceRow struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
	// Kind: base (the lists or segments), minus (excluded), plus (added back).
	Kind string `json:"kind" enum:"base,minus,plus"`
	// Action is the fix the line offers: include_unengaged ("Include
	// them") on the unengaged contacts the send leaves out.
	Action string `json:"action,omitempty" enum:"include_unengaged"`
}

// ReadinessAudience is the final count after exclusions.
type ReadinessAudience struct {
	Rows  []ReadinessAudienceRow `json:"rows"`
	Final int64                  `json:"final"`
	// Lists names the lists and segments of the audience.
	Lists []string `json:"lists"`
	// Suppressed, Role and Duplicates repeat the excluded counts for the
	// confirmation ("331 suppressed · 27 role · 26 duplicates").
	Suppressed int64 `json:"suppressed"`
	Role       int64 `json:"role"`
	Duplicates int64 `json:"duplicates"`
	Unengaged  int64 `json:"unengaged"`
	// UnengagedExcluded is true when the unengaged (the sunset tier) are
	// left out, as they are by default, and ExcludedUnengaged counts them.
	UnengagedExcluded bool  `json:"unengagedExcluded"`
	ExcludedUnengaged int64 `json:"excludedUnengaged"`
}

// ReadinessRouteShare is one sending provider's part of the send.
type ReadinessRouteShare struct {
	ProviderID string  `json:"providerId"`
	Name       string  `json:"name"`
	Short      string  `json:"short"`
	Adapter    string  `json:"adapter"`
	Label      string  `json:"label,omitempty"`
	Color      string  `json:"color" enum:"ses,postal,postmark,other"`
	Percent    float64 `json:"percent"`
	Recipients int64   `json:"recipients"`
	Status     string  `json:"status"`
	// StatusLabel is "Healthy", "Warming up · day 9 of 21", "Down since 08:52".
	StatusLabel string `json:"statusLabel"`
	WarmupDay   int    `json:"warmupDay,omitempty"`
	WarmupDays  int    `json:"warmupDays,omitempty"`
	// WarmupCap is the warm-up plan's cap on the send day (0 without one).
	WarmupCap int64 `json:"warmupCap,omitempty"`
	// Available is false when the provider can't take mail (down, paused,
	// webhooks silent); its share is held.
	Available bool `json:"available"`
	// Cost in US dollars; PriceLabel is "$0.10 per 1,000" or "self-hosted";
	// Pricing is how the provider is paid for.
	Cost       float64 `json:"cost"`
	PriceLabel string  `json:"priceLabel"`
	Pricing    string  `json:"pricing" enum:"metered,plan,self_hosted"`
	// Rate is messages per second; RateLabel "14 /s" or "2 /s ramp".
	// RateFrom says where the sending engine takes it from: published (the
	// provider's own maximum), default (the adapter's, when the provider
	// publishes none) or warmup (the plan's ramp).
	Rate      float64 `json:"rate"`
	RateLabel string  `json:"rateLabel"`
	RateFrom  string  `json:"rateFrom,omitempty" enum:"published,default,warmup"`
	// Minutes is how long the share sends, waits left out; Days how many
	// days a warm-up plan's caps spread it over; OverQuota how many
	// messages wait for the provider's 24-hour quota to free up.
	Minutes   int   `json:"minutes"`
	Days      int   `json:"days,omitempty"`
	OverQuota int64 `json:"overQuota,omitempty"`
}

// ReadinessTimeline is when the parts of the send go out, played out with
// the sending engine's own rules (SR-5): rates, warm-up caps, quotas,
// throttles, the A/B wait and best time.
type ReadinessTimeline struct {
	Start time.Time `json:"start"`
	// End is when the last message goes out. Open is true when a person
	// picks the A/B winner: the rest waits for them and End is the
	// sample's end.
	End      time.Time `json:"end"`
	Open     bool      `json:"open,omitempty"`
	BestTime bool      `json:"bestTime,omitempty"`
	// AB is the A/B test's timeline; nil without a test.
	AB *ReadinessABTimeline `json:"ab,omitempty"`
	// Basis names what the estimate includes; BasisLabel says it ("The
	// estimate includes each provider's rate limit, …").
	Basis      []string `json:"basis"`
	BasisLabel string   `json:"basisLabel"`
}

// ReadinessABTimeline is an A/B test's sample, wait and rest.
type ReadinessABTimeline struct {
	Kind        string `json:"kind" enum:"subject,content,send_time"`
	TestPercent int    `json:"testPercent"`
	Variants    int    `json:"variants"`
	Sample      int64  `json:"sample"`
	// PerVariant is the sample of each variant.
	PerVariant int64 `json:"perVariant"`
	// SampleEnd is when the sample's last message goes out; DecideAt when
	// the winner is picked (nil when a person picks it); Hours the wait.
	SampleEnd time.Time  `json:"sampleEnd"`
	DecideAt  *time.Time `json:"decideAt,omitempty"`
	Hours     int        `json:"hours"`
}

// ReadinessHeldMove is what moving a held share to an available provider costs.
type ReadinessHeldMove struct {
	From       string  `json:"from"`
	To         string  `json:"to"`
	ToID       string  `json:"toId"`
	AddCost    float64 `json:"addCost"`
	AddMinutes int     `json:"addMinutes"`
}

// ReadinessRoute is the route breakdown with cost and duration estimates.
type ReadinessRoute struct {
	Stream string `json:"stream" enum:"broadcast,automation,system"`
	// StreamLabel is "Broadcasts"; SplitLabel "Broadcasts 90 / 10".
	StreamLabel string                `json:"streamLabel"`
	SplitLabel  string                `json:"splitLabel"`
	Policy      string                `json:"policy" enum:"automatic,pause_and_ask,warm_only"`
	Override    bool                  `json:"override"`
	Shares      []ReadinessRouteShare `json:"shares"`
	TotalCost   float64               `json:"totalCost"`
	// TotalMinutes is how long the send takes from its start, waits
	// included (Timeline); without an estimate, the longest route (routes
	// run in parallel) or the slowest mailbox provider throttle.
	TotalMinutes int                `json:"totalMinutes"`
	Down         int                `json:"down"`
	Move         *ReadinessHeldMove `json:"move,omitempty"`
	// Timeline is when the send's parts go out; nil when the sending engine
	// couldn't estimate it.
	Timeline *ReadinessTimeline `json:"timeline,omitempty"`
}

// ReadinessThrottle is the rate for one mailbox provider.
type ReadinessThrottle struct {
	Key        string `json:"key" enum:"gmail,outlook,yahoo,icloud,other"`
	Name       string `json:"name"`
	Recipients int64  `json:"recipients"`
	// Rate is messages per second; 0 means the provider's full rate.
	Rate float64 `json:"rate"`
}

// ReadinessEstimates are the numbers around the send.
type ReadinessEstimates struct {
	Audience   ReadinessAudience   `json:"audience"`
	Route      ReadinessRoute      `json:"route"`
	Throttling []ReadinessThrottle `json:"throttling"`

	texts *estimateTexts
}

// ReadinessResult is what Assemble produces and a run stores.
type ReadinessResult struct {
	Verdict string           `json:"verdict" enum:"ready,warnings,blocked"`
	Counts  ReadinessCounts  `json:"counts"`
	Groups  []ReadinessGroup `json:"groups"`
	Checks  []ReadinessCheck `json:"checks"`
	// Incomplete is true when some checks couldn't run (Unknown); the
	// verdict then can't be Ready until a person continues without them.
	Incomplete bool `json:"incomplete"`
	// Problem says why checks are unknown ("api.hostrepute.com timed out…").
	Problem   string             `json:"problem,omitempty"`
	Estimates ReadinessEstimates `json:"estimates"`

	problem Text
}

// ReadinessRun is a stored run.
type ReadinessRun struct {
	ID         string    `json:"id"`
	CampaignID string    `json:"campaignId"`
	Trigger    string    `json:"trigger" enum:"manual,composer,schedule,send_time,approval"`
	RanAt      time.Time `json:"ranAt"`
	RanBy      string    `json:"ranBy,omitempty"`
	DurationMs int       `json:"durationMs"`
	// Version is the campaign version the run checked.
	Version int `json:"version"`
	ReadinessResult
	// UnknownAccepted is set when someone chose to continue with provider
	// data only.
	UnknownAcceptedAt *time.Time `json:"unknownAcceptedAt,omitempty"`
	UnknownAcceptedBy string     `json:"unknownAcceptedBy,omitempty"`
	// Fingerprint is what the run checked; it changes with the content,
	// audience, route or schedule.
	Fingerprint string `json:"-"`
}

// ReadinessProgressGroup is one group of a run in progress.
type ReadinessProgressGroup struct {
	Key    string `json:"key" enum:"reputation,authentication,content,languages,links,compliance,audience,routing,accessibility,conditional,rules"`
	Title  string `json:"title"`
	Status string `json:"status" enum:"queued,running,pass,warn,block,unknown"`
	Text   string `json:"text"`
	// Done of Total are the group's checks that finished.
	Done    int `json:"done"`
	Total   int `json:"total"`
	Checked int `json:"checked"`
	// Count of Of Unit is how far the group's long step got: "9 of 14
	// links", "5 of 8 domains", "24,310 of 31,988 contacts".
	Unit  string `json:"unit,omitempty" enum:"links,domains,contacts"`
	Count int64  `json:"count"`
	Of    int64  `json:"of"`

	texts  *progressTexts
	hidden bool
}

// ReadinessProgress is a run waiting for a worker or in progress.
type ReadinessProgress struct {
	RunID     string    `json:"runId"`
	Trigger   string    `json:"trigger"`
	StartedAt time.Time `json:"startedAt"`
	StartedBy string    `json:"startedBy,omitempty"`
	// Queued is true until a worker takes the run up.
	Queued bool                     `json:"queued"`
	Done   int                      `json:"done"`
	Total  int                      `json:"total"`
	Groups []ReadinessProgressGroup `json:"groups"`
}

// ReadinessRunFailure is a run that stopped on an error.
type ReadinessRunFailure struct {
	RunID string    `json:"runId"`
	At    time.Time `json:"at"`
	// Error is what went wrong, as the code said it (English).
	Error string `json:"error"`
}

// ReadinessRunRef names a stored run: what a poll compares with what the
// screen shows.
type ReadinessRunRef struct {
	ID      string    `json:"id"`
	RanAt   time.Time `json:"ranAt"`
	Verdict string    `json:"verdict" enum:"ready,warnings,blocked"`
}
