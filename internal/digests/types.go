package digests

import "time"

// API types. Names start with Digest: they become OpenAPI schema names.

// DigestPerson is a member as the digest screens show them.
type DigestPerson struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Initials string `json:"initials"`
	Color    string `json:"color"`
}

// DigestSourceRef names a digest's feed in lists ("blog.acme.io").
type DigestSourceRef struct {
	Kind string `json:"kind" enum:"url,wordpress"`
	// Label is the site's host, or the feed address without its scheme.
	Label      string   `json:"label"`
	Categories []string `json:"categories"`
}

// DigestIssueRef is an issue in a list row: the next one or the last sent.
type DigestIssueRef struct {
	ID     string     `json:"id"`
	Number int        `json:"number"`
	Status string     `json:"status" enum:"collecting,preparing,prepared,waiting,held,draft,sending,sent,skipped,not_sent"`
	SendAt *time.Time `json:"sendAt,omitempty"`
	Posts  int        `json:"posts"`
	// NeedsApproval: the approval rules match this issue (or would at preparation).
	NeedsApproval bool `json:"needsApproval"`
	// HumanOpens of a sent issue, and how it compares with the issue before (points).
	HumanOpens      *float64 `json:"humanOpens,omitempty"`
	HumanOpensDelta *float64 `json:"humanOpensDelta,omitempty"`
}

// DigestListItem is one row of Campaigns › Recurring.
type DigestListItem struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	State    string          `json:"state" enum:"off,on,paused"`
	Mode     string          `json:"mode" enum:"auto,draft"`
	Source   DigestSourceRef `json:"source"`
	Schedule DigestSchedule  `json:"schedule"`
	Next     *DigestIssueRef `json:"next,omitempty"`
	Last     *DigestIssueRef `json:"last,omitempty"`
	Owner    *DigestPerson   `json:"owner,omitempty"`
}

// DigestList is the workspace's digests.
type DigestList struct {
	Items []DigestListItem `json:"items"`
}

// DigestSite is a WordPress site of the integrations hub, as a feed source.
type DigestSite struct {
	ID string `json:"id"`
	// Host is "blog.acme.io"; Name the site's title.
	Host          string     `json:"host"`
	Name          string     `json:"name"`
	FeedURL       string     `json:"feedUrl"`
	ConnectedAt   time.Time  `json:"connectedAt"`
	ConnectedBy   string     `json:"connectedBy"`
	PluginVersion string     `json:"pluginVersion"`
	LatestPlugin  string     `json:"latestPlugin"`
	LastReportAt  *time.Time `json:"lastReportAt,omitempty"`
	// Status: ok, silent (not heard from in 24 hours) or outdated (a newer
	// plugin is out).
	Status string `json:"status" enum:"ok,silent,outdated"`
	// Items is how many items its feed had at the last read of a digest.
	Items int `json:"items"`
	// DraftRule names the categories the site's post-draft rule turns into
	// drafts, and the list they go to ("Announcements" → "Product Updates").
	DraftCategories []string `json:"draftCategories"`
	DraftList       string   `json:"draftList,omitempty"`
}

// DigestSites lists the workspace's WordPress sites.
type DigestSites struct {
	Items []DigestSite `json:"items"`
}

// DigestFeedHop is one address answered on the way to the feed.
type DigestFeedHop struct {
	URL      string `json:"url"`
	Status   int    `json:"status"`
	Location string `json:"location,omitempty"`
}

// DigestFeedError says why a read failed, with what the guard checked.
type DigestFeedError struct {
	// Kind: unreachable, timeout, blocked (the SSRF guard), too_large,
	// status (an HTTP error), redirects, address, not_feed, no_feed_link.
	Kind   string `json:"kind"`
	Host   string `json:"host"`
	URL    string `json:"url"`
	Status int    `json:"status,omitempty"`
	// Detail is the English sentence ("blog.acme.io didn't answer: …").
	Detail string `json:"detail"`
	// Lines are the first lines of what answered instead of a feed.
	Lines []string        `json:"lines"`
	Hops  []DigestFeedHop `json:"hops"`
	// Seconds is the read's time limit, for timeouts.
	Seconds int `json:"seconds,omitempty"`
}

// DigestFeedStatus is the feed as last read.
type DigestFeedStatus struct {
	Status string `json:"status" enum:"unread,ok,unreachable,invalid"`
	// URL is the address read (a WordPress site's reported feed).
	URL   string `json:"url"`
	Kind  string `json:"kind,omitempty" enum:"rss,atom,json,"`
	Title string `json:"title"`
	Items int    `json:"items"`
	Bytes int    `json:"bytes"`
	// LastReadVia: plugin (the WordPress plugin reported a post), schedule
	// (the 30-minute read), preparation, person (Read now), setup.
	LastReadAt  *time.Time       `json:"lastReadAt,omitempty"`
	LastReadVia string           `json:"lastReadVia,omitempty"`
	LastOkAt    *time.Time       `json:"lastOkAt,omitempty"`
	Error       *DigestFeedError `json:"error,omitempty"`
	// MovedTo is offered as "Use the new address" after a permanent redirect.
	MovedTo string `json:"movedTo,omitempty"`
	// Waiting counts the new items for the next issue and LeftOut the newest
	// items the filters leave out.
	Waiting int `json:"waiting"`
	LeftOut int `json:"leftOut"`
}

// DigestFilters are the feed's filters and the first issue's starting point.
type DigestFilters struct {
	// Categories picked: an item in any of them goes in. Empty: no category filter.
	Categories []string `json:"categories"`
	Tags       []string `json:"tags"`
	// MaxAgeDays leaves out items older than this many days (nil: off).
	MaxAgeDays  *int   `json:"maxAgeDays,omitempty" minimum:"1" maximum:"365"`
	StartMode   string `json:"startMode" enum:"after_on,latest"`
	StartLatest int    `json:"startLatest" minimum:"1" maximum:"20"`
}

// DigestAudience is who each issue goes to, counted now.
type DigestAudience struct {
	ListIDs    []string `json:"listIds"`
	SegmentIDs []string `json:"segmentIds"`
	// Label is "Product Updates list".
	Label      string `json:"label"`
	Base       int64  `json:"base"`
	Suppressed int64  `json:"suppressed"`
	Role       int64  `json:"role"`
	Duplicates int64  `json:"duplicates"`
	Stopped    int64  `json:"stopped"`
	Final      int64  `json:"final"`
	Empty      bool   `json:"empty"`
}

// DigestRouteShare is a sending provider of the digest's route.
type DigestRouteShare struct {
	ProviderID string  `json:"providerId"`
	Name       string  `json:"name"`
	Short      string  `json:"short"`
	Adapter    string  `json:"adapter"`
	Percent    float64 `json:"percent"`
	Status     string  `json:"status"`
	WarmupDay  int     `json:"warmupDay,omitempty"`
	WarmupDays int     `json:"warmupDays,omitempty"`
	// WarmupEnds is the last day of the warm-up plan.
	WarmupEnds *time.Time `json:"warmupEnds,omitempty"`
}

// DigestRoute is the route issues take.
type DigestRoute struct {
	// Override is true when the digest has its own split instead of the
	// Broadcasts stream's.
	Override bool               `json:"override"`
	Shares   []DigestRouteShare `json:"shares"`
}

// DigestThresholds are the auto-pause limits issues send with (percent).
type DigestThresholds struct {
	Bounce    float64 `json:"bounce"`
	Complaint float64 `json:"complaint"`
	// Scope: route (pause the affected mailbox or sending provider) or campaign.
	Scope string `json:"scope" enum:"route,campaign"`
	// Custom is true when the digest sets its own limits.
	Custom bool `json:"custom"`
}

// DigestSender is the From of the issues.
type DigestSender struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// DigestRuleMatch is an approval rule every issue matches.
type DigestRuleMatch struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Short string `json:"short"`
	// Until is when a rule stops matching (a provider's warm-up ends).
	Until *time.Time `json:"until,omitempty"`
	// Threshold is the audience rule's number of recipients (audience_min).
	Threshold int64 `json:"threshold,omitempty"`
}

// DigestApprovals is what the approval policy does to each issue.
type DigestApprovals struct {
	Needed bool              `json:"needed"`
	Rules  []DigestRuleMatch `json:"rules"`
	// Approvers who can approve issues (not the owner).
	Approvers []string `json:"approvers"`
}

// DigestPause says why a digest is paused.
type DigestPause struct {
	At *time.Time `json:"at,omitempty"`
	By string     `json:"by,omitempty"`
	// Reason: person, owner (the owner lost campaigns:send, was
	// deactivated or left) or feed (three feed failures in a row).
	Reason string `json:"reason" enum:"person,owner,feed"`
	Note   string `json:"note,omitempty"`
	// From and To are the owner's roles before and after (owner reason);
	// Left is true when the owner was deactivated or left.
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
	Left bool   `json:"left,omitempty"`
	// Notified are the members told.
	Notified []string `json:"notified"`
	// Skips are the send times of the issues skipped because of the feed.
	Skips []DigestFeedSkip `json:"skips"`
}

// DigestFeedSkip is an issue skipped because the feed failed.
type DigestFeedSkip struct {
	SendAt time.Time `json:"sendAt"`
	// Kind: unreachable or invalid.
	Kind string `json:"kind"`
}

// DigestFeedTry is a failed read at preparation.
type DigestFeedTry struct {
	At    time.Time `json:"at"`
	Error string    `json:"error"`
	Kind  string    `json:"kind"`
}

// DigestFeedItem is a post of the feed with what the digest does with it.
type DigestFeedItem struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	URL        string     `json:"url"`
	Author     string     `json:"author"`
	Published  *time.Time `json:"published,omitempty"`
	Categories []string   `json:"categories"`
	Tags       []string   `json:"tags"`
	Excerpt    string     `json:"excerpt"`
	// Image is the library copy; ImageAlt its alt text, AltFromTitle when
	// that is the title.
	Image        string `json:"image,omitempty"`
	ImageAlt     string `json:"imageAlt,omitempty"`
	AltFromTitle bool   `json:"altFromTitle"`
	// Status: new (goes in the next issue), sent (in issue SentIn),
	// left_out (by LeftOutBy: category, tag, age), before_start (published
	// before the digest was turned on).
	Status    string `json:"status" enum:"new,sent,left_out,before_start"`
	SentIn    int    `json:"sentIn,omitempty"`
	LeftOutBy string `json:"leftOutBy,omitempty" enum:"category,tag,age,"`
	// FirstSeenVia: plugin, schedule, preparation, person, setup.
	FirstSeenAt  time.Time `json:"firstSeenAt"`
	FirstSeenVia string    `json:"firstSeenVia"`
}

// DigestSentCount is how many older items went in an issue.
type DigestSentCount struct {
	Number int `json:"number"`
	Count  int `json:"count"`
}

// DigestFeedPreview is the newest items with their badges.
type DigestFeedPreview struct {
	Items []DigestFeedItem `json:"items"`
	// Total is how many items the feed has; the older ones are counted.
	Total        int               `json:"total"`
	OlderSent    []DigestSentCount `json:"olderSent"`
	OlderLeftOut int               `json:"olderLeftOut"`
	OlderBefore  int               `json:"olderBefore"`
	OlderNew     int               `json:"olderNew"`
}

// DigestSubject is the subject with a number of items.
type DigestSubject struct {
	Items int    `json:"items"`
	Text  string `json:"text"`
	// ZeroMore: the subject reads "…and 0 more" with this many items.
	ZeroMore bool `json:"zeroMore"`
}

// DigestNextIssue is the issue being collected (or prepared).
type DigestNextIssue struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Status string `json:"status" enum:"collecting,preparing,prepared,waiting,held,draft,sending,sent,skipped,not_sent"`
	// SendAt is unknown for an item-count schedule until it is prepared.
	SendAt    *time.Time `json:"sendAt,omitempty"`
	PrepareAt *time.Time `json:"prepareAt,omitempty"`
	Deadline  *time.Time `json:"deadline,omitempty"`
	Posts     int        `json:"posts"`
	// Fit is how many posts the Feed items block shows at most.
	Fit        int              `json:"fit"`
	Recipients int64            `json:"recipients"`
	Subject    string           `json:"subject"`
	Preheader  string           `json:"preheader"`
	Items      []DigestFeedItem `json:"items"`
	Subjects   []DigestSubject  `json:"subjects"`
	CampaignID string           `json:"campaignId,omitempty"`
	// Waiting/held details.
	Approval   *DigestIssueApproval `json:"approval,omitempty"`
	HoldReason string               `json:"holdReason,omitempty" enum:"readiness,changes,rejected,no_approvers,"`
	HeldAt     *time.Time           `json:"heldAt,omitempty"`
	PreparedAt *time.Time           `json:"preparedAt,omitempty"`
	FeedTries  []DigestFeedTry      `json:"feedTries"`
	// NextTry is the next read of a feed that failed at preparation.
	NextTry *time.Time `json:"nextTry,omitempty"`
	// Readiness of a prepared issue.
	Readiness *DigestIssueReadiness `json:"readiness,omitempty"`
}

// DigestIssueApproval is an issue's approval request.
type DigestIssueApproval struct {
	ID          string        `json:"id"`
	Status      string        `json:"status"`
	RequestedAt time.Time     `json:"requestedAt"`
	DecidedAt   *time.Time    `json:"decidedAt,omitempty"`
	DecidedBy   *DigestPerson `json:"decidedBy,omitempty"`
	Rules       []string      `json:"rules"`
}

// DigestIssueReadiness is the latest Send Readiness of an issue.
type DigestIssueReadiness struct {
	RunID    string    `json:"runId"`
	Verdict  string    `json:"verdict"`
	Passed   int       `json:"passed"`
	Warnings int       `json:"warnings"`
	Blocks   int       `json:"blocks"`
	RanAt    time.Time `json:"ranAt"`
	// Blocking are the titles of the checks that block.
	Blocking []string `json:"blocking"`
}

// DigestDetail is a digest with everything its overview and settings show.
type DigestDetail struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	State      string `json:"state" enum:"off,on,paused"`
	Mode       string `json:"mode" enum:"auto,draft"`
	TemplateID string `json:"templateId"`
	// TemplateVersion and TemplateSavedAt describe the template's latest save.
	TemplateVersion int        `json:"templateVersion"`
	TemplateSavedAt *time.Time `json:"templateSavedAt,omitempty"`
	Archive         bool       `json:"archive"`
	// Source and the feed.
	Source  string        `json:"source" enum:"url,wordpress"`
	FeedURL string        `json:"feedUrl"`
	Site    *DigestSite   `json:"site,omitempty"`
	Filters DigestFilters `json:"filters"`
	// CategoryOptions and TagOptions are the feed's own terms.
	CategoryOptions []string         `json:"categoryOptions"`
	TagOptions      []string         `json:"tagOptions"`
	Feed            DigestFeedStatus `json:"feed"`
	// Schedule.
	Schedule     DigestSchedule `json:"schedule"`
	PrepareHours int            `json:"prepareHours"`
	LateHours    int            `json:"lateHours"`
	// Audience, sender, route and auto-pause limits (the template's).
	Audience   DigestAudience   `json:"audience"`
	Sender     *DigestSender    `json:"sender,omitempty"`
	Route      DigestRoute      `json:"route"`
	Thresholds DigestThresholds `json:"thresholds"`
	Approvals  DigestApprovals  `json:"approvals"`
	// Owner and history.
	Owner      *DigestPerson `json:"owner,omitempty"`
	OwnerSince *time.Time    `json:"ownerSince,omitempty"`
	FirstOnAt  *time.Time    `json:"firstOnAt,omitempty"`
	Pause      *DigestPause  `json:"pause,omitempty"`
	// Next is the issue being collected or prepared; nil while off or paused.
	Next *DigestNextIssue `json:"next,omitempty"`
	// NextSend is the next send time by the schedule (for the turn-on confirmation).
	NextSend *time.Time `json:"nextSend,omitempty"`
	// SentIssues counts sent issues; FirstSentAt is when #1 went out.
	SentIssues  int           `json:"sentIssues"`
	FirstSentAt *time.Time    `json:"firstSentAt,omitempty"`
	CreatedBy   *DigestPerson `json:"createdBy,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
	// Helpers name who can help with what a member can't do ("Ask Sofia
	// Rossi, Marcus Chen or Linh Nguyen").
	SendHelpers  []string `json:"sendHelpers"`
	EditHelpers  []string `json:"editHelpers"`
	PauseHelpers []string `json:"pauseHelpers"`
}

// DigestIssue is a row of the issues table.
type DigestIssue struct {
	ID     string `json:"id"`
	Number *int   `json:"number,omitempty"`
	// Status: collecting, preparing (the feed failed and is retried),
	// prepared, waiting (for approval), held, draft (draft mode, waiting
	// for someone to send it), sending, sent, skipped, not_sent.
	Status   string     `json:"status" enum:"collecting,preparing,prepared,waiting,held,draft,sending,sent,skipped,not_sent"`
	SendAt   *time.Time `json:"sendAt,omitempty"`
	Deadline *time.Time `json:"deadline,omitempty"`
	// SkipReason: no_items, feed, not_approved, held, person.
	SkipReason      string               `json:"skipReason,omitempty" enum:"no_items,feed,not_approved,held,person,"`
	SkipNote        string               `json:"skipNote,omitempty"`
	SkippedBy       string               `json:"skippedBy,omitempty"`
	SkippedAt       *time.Time           `json:"skippedAt,omitempty"`
	HoldReason      string               `json:"holdReason,omitempty" enum:"readiness,changes,rejected,no_approvers,"`
	FirstIssue      bool                 `json:"firstIssue"`
	Posts           int                  `json:"posts"`
	PostsDelta      *int                 `json:"postsDelta,omitempty"`
	Recipients      *int64               `json:"recipients,omitempty"`
	Estimated       bool                 `json:"estimated"`
	RecipientsDelta *int64               `json:"recipientsDelta,omitempty"`
	HumanOpens      *float64             `json:"humanOpens,omitempty"`
	HumanOpensDelta *float64             `json:"humanOpensDelta,omitempty"`
	Clicks          *float64             `json:"clicks,omitempty"`
	ClicksDelta     *float64             `json:"clicksDelta,omitempty"`
	Delivered       *float64             `json:"delivered,omitempty"`
	Approval        *DigestIssueApproval `json:"approval,omitempty"`
	NeedsApproval   bool                 `json:"needsApproval"`
	CampaignID      string               `json:"campaignId,omitempty"`
	FeedTries       []DigestFeedTry      `json:"feedTries"`
	PreparedAt      *time.Time           `json:"preparedAt,omitempty"`
	SentAt          *time.Time           `json:"sentAt,omitempty"`
}

// DigestIssues is a digest's issues, newest first.
type DigestIssues struct {
	Items []DigestIssue `json:"items"`
}

// DigestHistoryItem is a line of the digest's history. The screen words it
// from these facts.
type DigestHistoryItem struct {
	// Kind: left_out (a new post the filters leave out), goes_in (a new post
	// for the next issue), sent, skipped, not_sent, turned_on, paused, created.
	Kind string    `json:"kind" enum:"left_out,goes_in,sent,skipped,not_sent,turned_on,paused,created"`
	At   time.Time `json:"at"`
	// Post facts (left_out, goes_in).
	Title     string     `json:"title,omitempty"`
	Category  string     `json:"category,omitempty"`
	LeftOutBy string     `json:"leftOutBy,omitempty"`
	DraftList string     `json:"draftList,omitempty"`
	Via       string     `json:"via,omitempty"`
	Published *time.Time `json:"published,omitempty"`
	// Issue facts (sent, skipped, not_sent).
	Number     int        `json:"number,omitempty"`
	Recipients int64      `json:"recipients,omitempty"`
	ApprovedBy string     `json:"approvedBy,omitempty"`
	ApprovedAt *time.Time `json:"approvedAt,omitempty"`
	SkipReason string     `json:"skipReason,omitempty"`
	SendAt     *time.Time `json:"sendAt,omitempty"`
	// Who (turned_on, paused, created, sent: the owner the issue went out for).
	Actor string `json:"actor,omitempty"`
	Owner string `json:"owner,omitempty"`
	Mode  string `json:"mode,omitempty"`
	Note  string `json:"note,omitempty"`
	// BuiltBy is who created the digest, when someone else turned it on.
	BuiltBy     string     `json:"builtBy,omitempty"`
	BuiltMode   string     `json:"builtMode,omitempty"`
	BuiltAt     *time.Time `json:"builtAt,omitempty"`
	PauseReason string     `json:"pauseReason,omitempty"`
}

// DigestHistory is the digest's recent history, newest first.
type DigestHistory struct {
	Items []DigestHistoryItem `json:"items"`
}
