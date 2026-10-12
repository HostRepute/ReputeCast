package notify

// EventType is one row of the notification preferences: the person chooses
// in-app and/or email per event type.
type EventType struct {
	Key      string
	Label    string
	Audience string
	// Short is the plural phrase used in summaries ("Approvals, auto-pauses").
	Short string
	// DefaultEmail is used until the person saves their preferences.
	DefaultEmail bool
	// Urgent emails ignore quiet hours and digests ("Auto-pauses and
	// provider outages always send immediately").
	Urgent bool
	Kinds  []string
}

// Events lists the event types in the order of the preferences table.
var Events = []EventType{
	{Key: KindApprovalRequested, Label: "Approval requested", Audience: "Only for approvers", Short: "approval requests", DefaultEmail: true, Kinds: []string{KindApprovalRequested}},
	{Key: KindApprovalDecided, Label: "Approval approved, rejected or changes requested", Audience: "Only for the author", Short: "approval decisions", DefaultEmail: true, Kinds: []string{KindApprovalDecided}},
	{Key: KindApprovalInvalidated, Label: "Approval invalidated", Audience: "Author and approvers", Short: "invalidated approvals", DefaultEmail: true, Kinds: []string{KindApprovalInvalidated}},
	{Key: KindSendPaused, Label: "Send auto-paused", Audience: "Everyone with campaigns:pause", Short: "auto-pauses", DefaultEmail: true, Urgent: true, Kinds: []string{KindSendPaused, KindSendBlocked}},
	{Key: KindProviderDown, Label: "Provider down or degraded", Audience: "Everyone with providers:view", Short: "provider down", DefaultEmail: true, Urgent: true, Kinds: []string{KindProviderDown, KindProviderDegraded}},
	{Key: KindWebhooksSilent, Label: "Webhooks silent", Short: "silent webhooks", DefaultEmail: false, Kinds: []string{KindWebhooksSilent}},
	{Key: KindInviteAccepted, Label: "Invite accepted", Audience: "Inviter only", Short: "invites", DefaultEmail: false, Kinds: []string{KindInviteAccepted, KindInviteRequested}},
	{Key: KindHostReputeAlert, Label: "HostRepute alerts", Audience: "Everyone with deliverability:view", Short: "HostRepute alerts", DefaultEmail: false, Kinds: []string{KindHostReputeAlert}},
}

// otherEvent covers kinds without a preference row (send finished, jobs):
// in-app only.
var otherEvent = EventType{Key: "other", Label: "Other", DefaultEmail: false}

// EventFor returns the event type a notification kind belongs to.
func EventFor(kind string) EventType {
	for _, e := range Events {
		for _, k := range e.Kinds {
			if k == kind {
				return e
			}
		}
	}
	return otherEvent
}

// Categories of the notification center tabs.
const (
	CategoryApprovals  = "approvals"
	CategorySending    = "sending"
	CategoryProviders  = "providers"
	CategoryTeam       = "team"
	CategoryHostRepute = "hostrepute"
	CategoryOther      = "other"
)

var categoryKinds = map[string][]string{
	CategoryApprovals:  {KindApprovalRequested, KindApprovalDecided, KindApprovalInvalidated},
	CategorySending:    {KindSendPaused, KindSendBlocked, KindSendFinished},
	CategoryProviders:  {KindProviderDown, KindProviderDegraded, KindWebhooksSilent},
	CategoryTeam:       {KindInviteAccepted, KindInviteRequested},
	CategoryHostRepute: {KindHostReputeAlert},
}

var categoryLabels = map[string]string{
	CategoryApprovals:  "Approvals",
	CategorySending:    "Sending",
	CategoryProviders:  "Providers",
	CategoryTeam:       "Team",
	CategoryHostRepute: "HostRepute",
	CategoryOther:      "Other",
}

// CategoryOf returns the center tab a kind is listed under.
func CategoryOf(kind string) string {
	for c, kinds := range categoryKinds {
		for _, k := range kinds {
			if k == kind {
				return c
			}
		}
	}
	return CategoryOther
}

// CategoryLabel is the tab name of a category.
func CategoryLabel(c string) string { return categoryLabels[c] }

// KindsOf returns the kinds of a category tab.
func KindsOf(category string) []string { return categoryKinds[category] }
