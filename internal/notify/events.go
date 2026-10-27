package notify

// EventType is one row of the notification preferences: the person chooses
// in-app and/or email per event type.
type EventType struct {
	Key string
	// Label and Audience are the row's name and who gets the event, in
	// English. messages/<locale>.json has them as event.<Key> and
	// audience.<Key>, with their translations.
	Label    string
	Audience string
	// Short is the plural phrase used in summaries ("Approvals,
	// auto-pauses"), in English. messages/<locale>.json has it as
	// short.<Key>, with its translations.
	Short string
	// DefaultEmail is used until the person saves their preferences.
	DefaultEmail bool
	// Urgent emails ignore quiet hours and digests ("Auto-pauses and
	// provider outages always send immediately").
	Urgent bool
	// OffByDefault event types are neither in-app nor emailed until the
	// person turns them on, and summaries leave them out while off.
	OffByDefault bool
	Kinds        []string
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
	{Key: KindCommentMention, Label: "Mentioned in a comment", Audience: "When someone @mentions you in a campaign's comments", Short: "mentions", DefaultEmail: true, Kinds: []string{KindCommentMention}},
	{Key: KindHostReputeAlert, Label: "HostRepute alerts", Audience: "Everyone with deliverability:view", Short: "HostRepute alerts", DefaultEmail: false, Kinds: []string{KindHostReputeAlert}},
	{Key: KindIntegrationAttention, Label: "Integration needs attention", Audience: "Everyone with integrations:manage", Short: "integration problems", DefaultEmail: true, Kinds: []string{KindIntegrationAttention}},
	{Key: KindPostDraftCreated, Label: "Post draft created", Audience: "When a WordPress post-draft rule names you", Short: "post drafts", DefaultEmail: true, Kinds: []string{KindPostDraftCreated}},
	{Key: KindTransactionalAlert, Label: "Transactional delivery alert", Audience: "Everyone with transactional:view", Short: "transactional delivery alerts", DefaultEmail: true, Kinds: []string{KindTransactionalAlert}},
	{Key: KindSMTPRelayUnreachable, Label: "SMTP relay unreachable", Audience: "Instance admins only", Short: "SMTP relay outages", DefaultEmail: true, Urgent: true, Kinds: []string{KindSMTPRelayUnreachable}},
	{Key: KindScoreRecalculated, Label: "Score recalculation finished", Audience: "The person who changed the rules", Short: "score recalculations", DefaultEmail: false, Kinds: []string{KindScoreRecalculated}},
	// Recurring digests (design expansion, phase 3).
	{Key: KindDigestIssueHeld, Label: "Digest issue held", Audience: "The owner, and approvers with a request open", Short: "held digest issues", DefaultEmail: true, Kinds: []string{KindDigestIssueHeld}},
	{Key: KindDigestPaused, Label: "Digest paused", Audience: "The owner, or everyone with campaigns:send", Short: "paused digests", DefaultEmail: true, Kinds: []string{KindDigestPaused}},
	{Key: KindDigestFeedFailing, Label: "Digest feed failing", Audience: "The digest's owner", Short: "failing digest feeds", DefaultEmail: true, Kinds: []string{KindDigestFeedFailing}},
	{Key: KindDigestDraftReady, Label: "Digest draft ready", Audience: "The owner and everyone with campaigns:send", Short: "digest drafts", DefaultEmail: true, Kinds: []string{KindDigestDraftReady}},
	// Campaign reports (review AN-15): the email someone asked for.
	{Key: KindResultsFinal, Label: "Report results final", Audience: "When you asked to be emailed", Short: "final report results", DefaultEmail: true, Kinds: []string{KindResultsFinal}},
	// AI assistant (design expansion, phase 3).
	{Key: KindAIUsage, Label: "AI usage at 80% and cap reached", Audience: "Everyone with integrations:manage", Short: "AI usage notices", DefaultEmail: true, Kinds: []string{KindAIUsage}},
	{Key: KindAIKeyRejected, Label: "AI key rejected by the provider", Audience: "Everyone with integrations:manage", Short: "rejected AI keys", DefaultEmail: true, Kinds: []string{KindAIKeyRejected}},
	{Key: KindPaymentsAttention, Label: "Payments: key rejected, webhook missing or silent", Audience: "Everyone with payments:manage", Short: "payment problems", DefaultEmail: true, Kinds: []string{KindPaymentsAttention}},
	{Key: KindUnmappedPrice, Label: "Payments: a Stripe price isn't mapped to a tier", Audience: "Everyone with payments:manage", Short: "unmapped prices", DefaultEmail: false, Kinds: []string{KindUnmappedPrice}},
	{Key: KindPaidImportDone, Label: "Paid members import finished", Audience: "Whoever ran the import", Short: "paid-members imports", DefaultEmail: false, Kinds: []string{KindPaidImportDone}},
	{Key: KindSponsorCreative, Label: "Sponsor creative due tomorrow or late", Audience: "Everyone with sponsors:manage", Short: "sponsor creatives", DefaultEmail: true, Kinds: []string{KindSponsorCreative}},
	{Key: KindSponsorPayment, Label: "Sponsor payment overdue", Audience: "Everyone with sponsors:manage", Short: "overdue sponsor payments", DefaultEmail: false, Kinds: []string{KindSponsorPayment}},
	{Key: KindFailedPayments, Label: "Failed payments · daily summary", Audience: "Everyone with earn:view · off by default", Short: "failed payments", OffByDefault: true, Kinds: []string{KindFailedPayments}},
	{Key: KindVerificationEnded, Label: "Verification finished or stopped", Audience: "The person who started it", Short: "verification jobs", DefaultEmail: true, Kinds: []string{KindVerificationEnded}},
	{Key: KindVerificationCreditsLow, Label: "Verification credits low", Audience: "Everyone with integrations:manage", Short: "low verification credits", DefaultEmail: true, Kinds: []string{KindVerificationCreditsLow}},
	{Key: KindPlacementFinished, Label: "Placement test finished", Audience: "The person who started it", Short: "placement tests", DefaultEmail: false, Kinds: []string{KindPlacementFinished}},
	{Key: KindSeedSignInFailed, Label: "Seed mailbox sign-in failed", Audience: "Everyone with placement:manage", Short: "seed sign-in failures", DefaultEmail: true, Kinds: []string{KindSeedSignInFailed}},
	// The publication website (design expansion, phase 7).
	{Key: KindWebsiteDomain, Label: "Website domain or certificate problem", Audience: "Everyone with website:manage", Short: "website domain problems", DefaultEmail: true, Kinds: []string{KindWebsiteDomain}},
	// Digital products (design expansion, phase 7).
	{Key: KindProductDeliveryFailed, Label: "Product delivery failed", Audience: "Everyone with orders:manage", Short: "failed product deliveries", DefaultEmail: true, Kinds: []string{KindProductDeliveryFailed}},
	{Key: KindOrderDisputed, Label: "Dispute opened on an order", Audience: "Everyone with orders:manage", Short: "order disputes", DefaultEmail: true, Kinds: []string{KindOrderDisputed}},
	// Grow (design expansion, phase 5).
	{Key: KindReferralsHeld, Label: "Referrals held for review", Audience: "Everyone with grow:review · then at most once a day while the queue isn't empty", Short: "held referrals", DefaultEmail: true, Kinds: []string{KindReferralsHeld}},
	{Key: KindRewardCodesLow, Label: "Reward codes running low or out", Audience: "Everyone with grow:edit", Short: "reward codes running low", DefaultEmail: true, Kinds: []string{KindRewardCodesLow}},
	{Key: KindRewardWebhookFailing, Label: "Reward webhook failing", Audience: "Everyone with grow:edit", Short: "failing reward webhooks", DefaultEmail: true, Kinds: []string{KindRewardWebhookFailing}},
	{Key: KindPartnerLinkProblem, Label: "Partner link invalid or unreachable", Audience: "Everyone with grow:connect", Short: "partner link problems", DefaultEmail: true, Kinds: []string{KindPartnerLinkProblem}},
	// SMS (design expansion, phase 8).
	{Key: KindSMSBudget, Label: "SMS budget at 80% or used up", Audience: "Everyone with providers:manage", Short: "SMS budget alerts", DefaultEmail: true, Kinds: []string{KindSMSBudget}},
	{Key: KindSMSRegistration, Label: "Sender registration approved or rejected", Audience: "Everyone with providers:view", Short: "sender registrations", DefaultEmail: true, Kinds: []string{KindSMSRegistration}},
	{Key: KindSMSProviderDown, Label: "SMS provider down or receipts silent", Audience: "Everyone with providers:view", Short: "SMS provider problems", DefaultEmail: true, Urgent: true, Kinds: []string{KindSMSProviderDown}},
	{Key: KindSMSSendPaused, Label: "SMS campaign auto-paused", Audience: "Everyone with campaigns:pause", Short: "SMS auto-pauses", DefaultEmail: true, Urgent: true, Kinds: []string{KindSMSSendPaused}},
	{Key: KindStoreProblem, Label: "Store disconnected or its webhooks missing", Audience: "Everyone with integrations:manage", Short: "store problems", DefaultEmail: true, Kinds: []string{KindStoreProblem}},
	// Web push and WhatsApp (design expansion, phase 8).
	{Key: KindPushWorkerMissing, Label: "Push service worker missing", Audience: "Everyone with providers:view", Short: "missing push service workers", DefaultEmail: false, Kinds: []string{KindPushWorkerMissing}},
	{Key: KindWhatsAppAttention, Label: "WhatsApp template rejected or quality dropped", Audience: "Everyone with providers:view", Short: "WhatsApp templates and quality", DefaultEmail: true, Kinds: []string{KindWhatsAppAttention, KindWhatsAppTokenFails}},
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
	// CategoryIntegrations holds the integrations hub's notifications.
	CategoryIntegrations = "integrations"
	// CategoryTransactional holds transactional mail's delivery alerts and
	// the SMTP relay.
	CategoryTransactional = "transactional"
	// CategoryAI holds the AI assistant's cap and key notices.
	CategoryAI = "ai"
	// CategoryEarn holds payments and paid members.
	CategoryEarn = "earn"
	// CategoryWebsite holds the publication website's notifications.
	CategoryWebsite = "website"
	// CategoryGrow holds referrals' and partner links' notifications.
	CategoryGrow = "grow"
)

var categoryKinds = map[string][]string{
	CategoryApprovals:     {KindApprovalRequested, KindApprovalDecided, KindApprovalInvalidated},
	CategorySending:       {KindSendPaused, KindSendBlocked, KindSendFinished, KindResultsFinal, KindDigestIssueHeld, KindDigestPaused, KindDigestFeedFailing, KindDigestDraftReady, KindSMSSendPaused},
	CategoryProviders:     {KindProviderDown, KindProviderDegraded, KindWebhooksSilent, KindSMSProviderDown, KindSMSRegistration, KindSMSBudget, KindPushWorkerMissing, KindWhatsAppAttention, KindWhatsAppTokenFails},
	CategoryTeam:          {KindInviteAccepted, KindInviteRequested, KindCommentMention},
	CategoryHostRepute:    {KindHostReputeAlert},
	CategoryIntegrations:  {KindIntegrationAttention, KindPostDraftCreated, KindStoreProblem},
	CategoryTransactional: {KindTransactionalAlert, KindSMTPRelayUnreachable},
	CategoryAI:            {KindAIUsage, KindAIKeyRejected},
	CategoryEarn:          {KindPaymentsAttention, KindUnmappedPrice, KindPaidImportDone, KindFailedPayments, KindSponsorCreative, KindSponsorPayment},
	CategoryWebsite:       {KindWebsiteDomain, KindProductDeliveryFailed, KindOrderDisputed},
	CategoryGrow:          {KindReferralsHeld, KindRewardCodesLow, KindRewardWebhookFailing, KindPartnerLinkProblem},
}

var categoryLabels = map[string]string{
	CategoryApprovals:     "Approvals",
	CategorySending:       "Sending",
	CategoryProviders:     "Providers",
	CategoryTeam:          "Team",
	CategoryHostRepute:    "HostRepute",
	CategoryOther:         "Other",
	CategoryIntegrations:  "Integrations",
	CategoryTransactional: "Transactional",
	CategoryAI:            "AI",
	CategoryEarn:          "Earn",
	CategoryWebsite:       "Website",
	CategoryGrow:          "Grow",
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

// CategoryLabel is the tab name of a category, in English
// (messages/<locale>.json has it as category.<tab>, with its translations).
func CategoryLabel(c string) string { return categoryLabels[c] }

// KindsOf returns the kinds of a category tab.
func KindsOf(category string) []string { return categoryKinds[category] }
