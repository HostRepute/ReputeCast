package connect

import (
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/providers"
)

// The connect flow's words for each adapter. Capabilities, terms and fields
// always come from the adapter itself (its Descriptor); this table only
// holds how the catalog and the connection step talk about the providers
// the design names (spec 5.2): display name, pricing line, a few chip
// wordings, where to find API keys, which webhook events get registered.
// Adapters that aren't listed (community adapters, new built-ins) get
// wording derived from their Descriptor.

type adapterCopy struct {
	// Display is the catalog name ("Any SMTP server").
	Display string
	// Short is the name in "Connect SES" and the stream cards ("SES").
	Short string
	// Step1 is the provider's name in the stepper ("SMTP server").
	Step1       string
	Recommended bool
	// TermsLabel overrides the terms badge ("Bulk via Broadcasts API").
	TermsLabel string
	// Pricing is the catalog's pricing line.
	Pricing string
	// Chip label overrides by chip key (webhooks, domain, suppression,
	// batch, dedicated, sandbox, quota).
	Chips map[string]string
	// Title is the wizard header ("Connect an SMTP server").
	Title string
	// Heading and Lead open the connection step.
	Heading, Lead string
	// ConnectionHint is the stepper's line under "Connection".
	ConnectionHint string
	// DomainHint is the stepper's line under "Domain verification".
	DomainHint string
	// KeyHint follows a rejected key ("Check that it's a Private API key…").
	KeyHint string
	// KeysURL is where the provider's dashboard lists API keys.
	KeysURL string
	// VerifyCall names the read-only call that checks keys ("GET /v3/domains").
	VerifyCall string
	// WebhookEvents lists the events ReputeCast registers.
	WebhookEvents string
	// CanPoll is true when the provider has an events API ReputeCast can
	// poll instead of webhooks.
	EventsAPI bool
	// Section is the first card's title ("1. API key").
	Section string
}

var copyTable = map[string]adapterCopy{
	"ses": {Display: "Amazon SES", Short: "SES", Step1: "Amazon SES", Recommended: true, Pricing: "Pay as you go · $0.10 per 1,000 · bulk marketing allowed with production access",
		Chips: map[string]string{"webhooks": "Webhooks (SNS)", "dedicated": "Dedicated IPs, paid"},
		Title: "Connect Amazon SES", Heading: "Connect your AWS account",
		Lead:           "ReputeCast needs permission to send through SES, create SNS topics for events and manage configuration sets. Nothing else.",
		ConnectionHint: "AWS access, region, provisioning", DomainHint: "DKIM, MAIL FROM, SPF, DMARC"},
	"postmark": {Display: "Postmark", Short: "Postmark", Step1: "Postmark", TermsLabel: "Bulk on Broadcast streams",
		Pricing: "Plans from $15 per month for 10,000 · Transactional streams are one-to-one only",
		Chips:   map[string]string{"dedicated": "Dedicated IPs, 300k+", "sandbox": "Sandbox token"},
		Title:   "Connect Postmark", Heading: "Paste your Postmark server token",
		Lead:    "ReputeCast registers webhooks and fetches the DNS records for your domain through the Postmark API.",
		KeyHint: "Check that it's a Server API token (not the Account token) from the server you send with.", KeysURL: "https://account.postmarkapp.com/servers",
		VerifyCall: "GET /server", WebhookEvents: "Delivery, Bounce, SpamComplaint, SubscriptionChange", Section: "1. Server token"},
	"mailgun": {Display: "Mailgun", Short: "Mailgun", Step1: "Mailgun", Pricing: "Plans from $15 per month for 10,000, then per 1,000",
		Chips: map[string]string{"dedicated": "Dedicated IPs, paid", "sandbox": "Sandbox domain"},
		Title: "Connect Mailgun", Heading: "Paste your Mailgun API key",
		Lead:    "ReputeCast registers webhooks and fetches the DNS records for your domain through the Mailgun API.",
		KeyHint: "Check that it's a Private API key (not a Sending key or the HTTP webhook signing key) and that it belongs to the region selected below.",
		KeysURL: "https://app.mailgun.com/settings/api_security", VerifyCall: "GET /v3/domains",
		WebhookEvents: "delivered, permanent_fail, temporary_fail, complained, unsubscribed", EventsAPI: true, Section: "1. API key"},
	"sendgrid": {Display: "SendGrid", Short: "SendGrid", Step1: "SendGrid", Pricing: "Free tier, then plans from $19.95 per month",
		Chips: map[string]string{"webhooks": "Event webhook", "dedicated": "Dedicated IPs, Pro plan", "sandbox": "Sandbox mode", "quota": "Quota, plan only"},
		Title: "Connect SendGrid", Heading: "Paste your SendGrid API key",
		Lead:    "ReputeCast registers the event webhook and fetches the DNS records for your domain through the SendGrid API.",
		KeyHint: "Check that the key has Full Access, or at least Mail Send, Sender Authentication and Mail Settings.", KeysURL: "https://app.sendgrid.com/settings/api_keys",
		VerifyCall: "GET /v3/scopes", WebhookEvents: "delivered, bounce, dropped, deferred, spamreport, unsubscribe", Section: "1. API key"},
	"resend": {Display: "Resend", Short: "Resend", Step1: "Resend", TermsLabel: "Bulk via Broadcasts API", Pricing: "Free 3,000 per month, then $20 per month for 50,000",
		Chips: map[string]string{"dedicated": "Dedicated IPs, Scale plan", "quota": "Quota, plan only"},
		Title: "Connect Resend", Heading: "Paste your Resend API key",
		Lead:    "ReputeCast registers webhooks and fetches the DNS records for your domain through the Resend API.",
		KeyHint: "Check that the key has Full access, not Sending access only.", KeysURL: "https://resend.com/api-keys", VerifyCall: "GET /domains",
		WebhookEvents: "email.delivered, email.bounced, email.complained, email.delivery_delayed", Section: "1. API key"},
	"sparkpost": {Display: "SparkPost", Short: "SparkPost", Step1: "SparkPost", Pricing: "Plans from $20 per month for 50,000",
		Chips: map[string]string{"dedicated": "Dedicated IPs, paid", "sandbox": "Sandbox domain"},
		Title: "Connect SparkPost", Heading: "Paste your SparkPost API key",
		Lead:    "ReputeCast registers webhooks and fetches the DNS records for your domain through the SparkPost API.",
		KeyHint: "Check the key's permissions and that it belongs to the region selected below (US or EU).", KeysURL: "https://app.sparkpost.com/account/api-keys",
		VerifyCall: "GET /api/v1/account", WebhookEvents: "delivery, bounce, out_of_band, spam_complaint, list_unsubscribe", EventsAPI: true, Section: "1. API key"},
	"mailjet": {Display: "Mailjet", Short: "Mailjet", Step1: "Mailjet", Pricing: "Free 6,000 per month, then plans from $17 per month",
		Chips: map[string]string{"dedicated": "Dedicated IPs, paid"}, Title: "Connect Mailjet", Heading: "Paste your Mailjet API keys",
		Lead:    "ReputeCast registers event callbacks and fetches the DNS records for your domain through the Mailjet API.",
		KeyHint: "Check both the API key and the secret key; they come as a pair.", KeysURL: "https://app.mailjet.com/account/apikeys", VerifyCall: "GET /v3/REST/apikey",
		WebhookEvents: "sent, bounce, blocked, spam, unsub", Section: "1. API keys"},
	"brevo": {Display: "Brevo", Short: "Brevo", Step1: "Brevo", Pricing: "Transactional API, plans from $15 per month for 20,000",
		Chips: map[string]string{"dedicated": "Dedicated IPs, paid"}, Title: "Connect Brevo", Heading: "Paste your Brevo API key",
		Lead:    "ReputeCast registers webhooks and fetches the DNS records for your domain through the Brevo API.",
		KeyHint: "Check that it's an API v3 key, not an SMTP key.", KeysURL: "https://app.brevo.com/settings/keys/api", VerifyCall: "GET /v3/account",
		WebhookEvents: "delivered, hardBounce, softBounce, spam, unsubscribed", Section: "1. API key"},
	"azureacs": {Display: "Azure Communication Services", Short: "Azure", Step1: "Azure Communication Services",
		Pricing: "Pay as you go, per email and per MB",
		Chips:   map[string]string{"webhooks": "Event Grid"}, Title: "Connect Azure Communication Services", Heading: "Paste your Azure connection string",
		Lead:    "ReputeCast sends through your Email Communication resource and receives delivery reports through Event Grid.",
		KeyHint: "Copy the connection string from your Communication Services resource › Keys.", KeysURL: "https://portal.azure.com/", VerifyCall: "GET /emails",
		WebhookEvents: "EmailDeliveryReportReceived", Section: "1. Connection string"},
	"postal": {Display: "Postal (self-hosted)", Short: "Postal", Step1: "Postal", Pricing: "Free software, your server · needs a warm-up for a new IP",
		Chips: map[string]string{"dedicated": "Dedicated IP"}, Title: "Connect Postal", Heading: "Connect your Postal server",
		Lead:    "ReputeCast sends through your mail server's API and receives its webhooks. Add the webhook in Postal once the connection works.",
		KeyHint: "Check that it's an API credential of the mail server (Credentials › API), not an SMTP one.", VerifyCall: "POST /api/v1/messages/message",
		WebhookEvents: "MessageSent, MessageDelayed, MessageDeliveryFailed, MessageHeld, MessageBounced", Section: "1. Server and API key"},
	"smtp": {Display: "Any SMTP server", Short: "SMTP", Step1: "SMTP server", Pricing: "No webhooks, so bounces come from a bounce mailbox you connect over IMAP or POP3",
		Chips: map[string]string{"dedicated": "Dedicated IP, depends"}, Title: "Connect an SMTP server", Heading: "SMTP server details",
		Lead:           "Works with any relay or your own mail server. SMTP has no event webhooks, so ReputeCast reads bounces from a mailbox you connect below.",
		ConnectionHint: "Host, TLS, credentials, bounce mailbox", DomainHint: "DKIM, return-path, SPF, DMARC"},
}

// copyFor returns the wording for an adapter, derived from its Descriptor
// where the table has nothing.
func copyFor(d providers.Descriptor) adapterCopy {
	c, ok := copyTable[d.Adapter]
	if !ok {
		c = adapterCopy{}
	}
	if c.Display == "" {
		c.Display = d.Name
	}
	if c.Short == "" {
		c.Short = shortName(d.Name)
	}
	if c.Step1 == "" {
		c.Step1 = d.Name
	}
	if c.Pricing == "" {
		c.Pricing = d.Capabilities.Pricing
	}
	if c.Title == "" {
		c.Title = "Connect " + d.Name
	}
	if c.Heading == "" {
		c.Heading = "Connect " + d.Name
		if hasSecret(d, "api_key") {
			c.Heading = "Paste your " + d.Name + " API key"
		}
	}
	if c.Lead == "" {
		c.Lead = "ReputeCast checks the credentials with a read-only call and stores them encrypted. They are never shown again."
	}
	if c.ConnectionHint == "" {
		c.ConnectionHint = "API key, region, webhooks"
	}
	// DomainHint stays empty for API providers: their records come from the
	// provider's API once the key works (SetupError), so the stepper only
	// previews the domain step for SES and SMTP.
	if c.Section == "" {
		c.Section = "1. Credentials"
	}
	if c.WebhookEvents == "" {
		c.WebhookEvents = "delivery, bounce and complaint events"
	}
	return c
}

func shortName(name string) string {
	if i := strings.Index(name, " ("); i > 0 {
		return name[:i]
	}
	return name
}

func hasSecret(d providers.Descriptor, key string) bool {
	for _, f := range d.Fields {
		if f.Key == key && f.Secret {
			return true
		}
	}
	return false
}

// flowOf picks the connection step's layout: SES (AWS access, region,
// provisioning), SMTP (server and bounce mailbox) or the generic API form.
func flowOf(d providers.Descriptor) string {
	switch {
	case d.Adapter == "ses":
		return "ses"
	case d.Kind == "smtp" || d.Capabilities.Webhooks == "Bounce mailbox":
		return "smtp"
	}
	return "api"
}

// ConnectChip is one capability chip of a catalog card.
type ConnectChip struct {
	Label string `json:"label"`
	// Kind: yes (green), partial (amber), no (struck through).
	Kind string `json:"kind" enum:"yes,partial,no"`
}

// chips turns an adapter's capabilities into the catalog's chips, in the
// design's order: events, domain setup, suppressions, batching, dedicated
// IPs, sandbox, quota.
func chips(d providers.Descriptor, c adapterCopy) []ConnectChip {
	caps := d.Capabilities
	out := []ConnectChip{}
	add := func(key, kind, label string) {
		if l, ok := c.Chips[key]; ok && kind != "no" {
			label = l
		}
		out = append(out, ConnectChip{Label: label, Kind: kind})
	}
	switch caps.Webhooks {
	case "":
		add("webhooks", "no", "Webhooks")
	case "Bounce mailbox":
		add("webhooks", "partial", "Bounce mailbox")
	case "Registered":
		add("webhooks", "yes", "Webhooks")
	default:
		add("webhooks", "yes", "Webhooks ("+caps.Webhooks+")")
	}
	switch caps.DomainAPI {
	case "full":
		add("domain", "yes", "Domain via API")
	case "dkim_only":
		add("domain", "partial", "DKIM via API")
	default:
		add("domain", "partial", "Domain, manual DNS")
	}
	switch {
	case caps.SuppressionSync && strings.EqualFold(d.SuppressionMode, "Pull only"):
		add("suppression", "partial", "Suppression, pull only")
	case caps.SuppressionSync:
		add("suppression", "yes", "Suppression sync")
	case strings.EqualFold(d.SuppressionMode, "Pull only"):
		add("suppression", "partial", "Suppression, pull only")
	default:
		add("suppression", "partial", "Suppression, local")
	}
	if caps.BatchSize > 1 && d.Kind != "smtp" {
		add("batch", "yes", "Batch "+thousands(caps.BatchSize))
	} else {
		add("batch", "no", "Batch")
	}
	switch {
	case d.Kind == "self_hosted":
		add("dedicated", "yes", "Dedicated IP")
	case d.Kind == "smtp":
		add("dedicated", "partial", "Dedicated IP, depends")
	case caps.DedicatedIPs != "":
		add("dedicated", "partial", "Dedicated IPs, paid")
	default:
		add("dedicated", "no", "Dedicated IPs")
	}
	switch caps.Sandbox {
	case "detection":
		add("sandbox", "yes", "Sandbox detection")
	case "test token":
		add("sandbox", "yes", "Sandbox token")
	case "mode":
		add("sandbox", "yes", "Sandbox mode")
	case "domain":
		add("sandbox", "yes", "Sandbox domain")
	case "":
		add("sandbox", "no", "Sandbox")
	default:
		add("sandbox", "yes", "Sandbox "+caps.Sandbox)
	}
	if caps.QuotaDiscovery {
		add("quota", "yes", "Quota discovery")
	} else {
		add("quota", "partial", "Quota, set manually")
	}
	return out
}

func thousands(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// termsBadge is the catalog's badge about bulk marketing.
func termsBadge(d providers.Descriptor, c adapterCopy) (string, string) {
	if c.Recommended {
		return "Recommended", "ok"
	}
	if c.TermsLabel != "" {
		return c.TermsLabel, "ok"
	}
	switch d.Capabilities.Bulk {
	case providers.BulkAllowed:
		return "Bulk marketing allowed", "ok"
	case providers.BulkBroadcastStreams:
		return "Bulk on Broadcast streams", "ok"
	case providers.BulkForbidden:
		return "Transactional only", "warn"
	}
	if d.Kind == "smtp" {
		return "Depends on the server", "neutral"
	}
	return "Your own policy", "ok"
}
