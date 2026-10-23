package connect

import (
	"cmp"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

// The connect flow's words for each adapter. Capabilities, terms and fields
// always come from the adapter itself (its Descriptor); this table only
// holds how the catalog and the connection step talk about the providers
// the design names (spec 5.2). The words are messages (adapter.<key>.* in
// messages/<locale>.json), written in the reader's language by copyFor.

// copySpec is how the flow talks about one adapter. A zero message falls
// back to the generic wording (copyFor).
type copySpec struct {
	// Display is the catalog name ("Any SMTP server"); zero: the adapter's name.
	Display i18n.Msg
	// Short is the name in "Connect SES" and the stream cards ("SES").
	Short string
	// Step1 is the provider's name in the stepper ("SMTP server"), or
	// Step1Name when it is a name ("Postal"); zero: the adapter's name.
	Step1       i18n.Msg
	Step1Name   string
	Recommended bool
	// Terms overrides the terms badge ("Bulk via Broadcasts API").
	Terms i18n.Msg
	// Pricing is the catalog's pricing line.
	Pricing i18n.Msg
	// Chips overrides chip labels by chip key (webhooks, domain,
	// suppression, batch, dedicated, sandbox, quota).
	Chips map[string]i18n.Msg
	// Title is the wizard header ("Connect an SMTP server").
	Title i18n.Msg
	// Heading and Lead open the connection step.
	Heading, Lead i18n.Msg
	// ConnectionHint is the stepper's line under "Connection".
	ConnectionHint i18n.Msg
	// DomainHint is the stepper's line under "Domain verification": the
	// records, by their names.
	DomainHint string
	// KeyHint follows a rejected key ("Check that it's a Private API key…").
	KeyHint i18n.Msg
	// KeysURL is where the provider's dashboard lists API keys.
	KeysURL string
	// VerifyCall names the read-only call that checks keys ("GET /v3/domains").
	VerifyCall string
	// WebhookEvents lists the events ReputeCast registers, by the
	// provider's own names.
	WebhookEvents string
	// EventsAPI is true when the provider has an events API ReputeCast can
	// poll instead of webhooks.
	EventsAPI bool
	// Section is the first card's title ("1. API key").
	Section i18n.Msg
}

func adapterMsg(key string) i18n.Msg { return msgs.M("adapter." + key) }

var copyTable = map[string]copySpec{
	"ses": {Short: "SES", Recommended: true, Pricing: adapterMsg("ses.pricing"), Heading: adapterMsg("ses.heading"), Lead: adapterMsg("ses.lead"),
		ConnectionHint: adapterMsg("ses.connectionHint"), DomainHint: "DKIM, MAIL FROM, SPF, DMARC"},
	"postmark": {Short: "Postmark", Terms: adapterMsg("postmark.terms"), Pricing: adapterMsg("postmark.pricing"),
		Chips:   map[string]i18n.Msg{"dedicated": mChipDedicated300k},
		Heading: adapterMsg("postmark.heading"), Lead: mInfoLeadRegisters, KeyHint: adapterMsg("postmark.keyHint"), KeysURL: "https://account.postmarkapp.com/servers",
		VerifyCall: "GET /server", WebhookEvents: "Delivery, Bounce, SpamComplaint, SubscriptionChange", Section: adapterMsg("postmark.section")},
	"mailgun": {Short: "Mailgun", Pricing: adapterMsg("mailgun.pricing"), Chips: map[string]i18n.Msg{"sandbox": mChipSandboxDomain},
		Heading: mInfoHeadingAPIKey, Lead: mInfoLeadRegisters, KeyHint: adapterMsg("mailgun.keyHint"), KeysURL: "https://app.mailgun.com/settings/api_security",
		VerifyCall: "GET /v3/domains", WebhookEvents: "delivered, permanent_fail, temporary_fail, complained, unsubscribed", EventsAPI: true, Section: mInfoSectionAPIKey},
	"sendgrid": {Short: "SendGrid", Pricing: adapterMsg("sendgrid.pricing"),
		Chips:   map[string]i18n.Msg{"webhooks": mChipEventWebhook, "dedicated": mChipDedicatedPro, "quota": mChipQuotaPlan},
		Heading: mInfoHeadingAPIKey, Lead: adapterMsg("sendgrid.lead"), KeyHint: adapterMsg("sendgrid.keyHint"), KeysURL: "https://app.sendgrid.com/settings/api_keys",
		VerifyCall: "GET /v3/scopes", WebhookEvents: "delivered, bounce, dropped, deferred, spamreport, unsubscribe", Section: mInfoSectionAPIKey},
	"resend": {Short: "Resend", Terms: adapterMsg("resend.terms"), Pricing: adapterMsg("resend.pricing"),
		Chips:   map[string]i18n.Msg{"dedicated": mChipDedicatedScale, "quota": mChipQuotaPlan},
		Heading: mInfoHeadingAPIKey, Lead: mInfoLeadRegisters, KeyHint: adapterMsg("resend.keyHint"), KeysURL: "https://resend.com/api-keys", VerifyCall: "GET /domains",
		WebhookEvents: "email.delivered, email.bounced, email.complained, email.delivery_delayed", Section: mInfoSectionAPIKey},
	"sparkpost": {Short: "SparkPost", Pricing: adapterMsg("sparkpost.pricing"), Chips: map[string]i18n.Msg{"sandbox": mChipSandboxDomain},
		Heading: mInfoHeadingAPIKey, Lead: mInfoLeadRegisters, KeyHint: adapterMsg("sparkpost.keyHint"), KeysURL: "https://app.sparkpost.com/account/api-keys",
		VerifyCall: "GET /api/v1/account", WebhookEvents: "delivery, bounce, out_of_band, spam_complaint, list_unsubscribe", EventsAPI: true, Section: mInfoSectionAPIKey},
	"mailjet": {Short: "Mailjet", Pricing: adapterMsg("mailjet.pricing"), Heading: adapterMsg("mailjet.heading"), Lead: adapterMsg("mailjet.lead"),
		KeyHint: adapterMsg("mailjet.keyHint"), KeysURL: "https://app.mailjet.com/account/apikeys", VerifyCall: "GET /v3/REST/apikey",
		WebhookEvents: "sent, bounce, blocked, spam, unsub", Section: adapterMsg("mailjet.section")},
	"brevo": {Short: "Brevo", Pricing: adapterMsg("brevo.pricing"), Heading: mInfoHeadingAPIKey, Lead: mInfoLeadRegisters,
		KeyHint: adapterMsg("brevo.keyHint"), KeysURL: "https://app.brevo.com/settings/keys/api", VerifyCall: "GET /v3/account",
		WebhookEvents: "delivered, hardBounce, softBounce, spam, unsubscribed", Section: mInfoSectionAPIKey},
	"azureacs": {Short: "Azure", Pricing: adapterMsg("azureacs.pricing"), Chips: map[string]i18n.Msg{"webhooks": mChipEventGrid},
		Heading: adapterMsg("azureacs.heading"), Lead: adapterMsg("azureacs.lead"), KeyHint: adapterMsg("azureacs.keyHint"), KeysURL: "https://portal.azure.com/",
		VerifyCall: "GET /emails", WebhookEvents: "EmailDeliveryReportReceived", Section: adapterMsg("azureacs.section")},
	"postal": {Display: adapterMsg("postal.display"), Short: "Postal", Step1Name: "Postal", Pricing: adapterMsg("postal.pricing"), Title: adapterMsg("postal.title"),
		Heading: adapterMsg("postal.heading"),
		Lead:    adapterMsg("postal.lead"), KeyHint: adapterMsg("postal.keyHint"), VerifyCall: "POST /api/v1/messages/message",
		WebhookEvents: "MessageSent, MessageDelayed, MessageDeliveryFailed, MessageHeld, MessageBounced", Section: adapterMsg("postal.section")},
	"smtp": {Display: adapterMsg("smtp.display"), Short: "SMTP", Step1: adapterMsg("smtp.step1"), Pricing: adapterMsg("smtp.pricing"),
		Title: adapterMsg("smtp.title"), Heading: adapterMsg("smtp.heading"), Lead: adapterMsg("smtp.lead"),
		ConnectionHint: adapterMsg("smtp.connectionHint"), DomainHint: "DKIM, return-path, SPF, DMARC"},
}

// adapterCopy is how the flow talks about an adapter, in one language.
type adapterCopy struct {
	Display, Short, Step1 string
	Recommended           bool
	// TermsLabel overrides the terms badge.
	TermsLabel string
	Pricing    string
	// Chips are chip label overrides by chip key.
	Chips          map[string]string
	Title          string
	Heading, Lead  string
	ConnectionHint string
	DomainHint     string
	KeyHint        string
	KeysURL        string
	VerifyCall     string
	WebhookEvents  string
	EventsAPI      bool
	Section        string
}

var noMsg i18n.Msg

// copyFor returns the wording for an adapter in a language, derived from
// its Descriptor where the table has nothing.
func copyFor(l i18n.Lang, d providers.Descriptor) adapterCopy {
	spec := copyTable[d.Adapter]
	args := i18n.Args{"name": d.Name}
	text := func(m, fallback i18n.Msg) string {
		if m == noMsg {
			m = fallback
		}
		if m == noMsg {
			return ""
		}
		return m.Text(l, args)
	}
	c := adapterCopy{Short: spec.Short, Recommended: spec.Recommended, DomainHint: spec.DomainHint, KeysURL: spec.KeysURL, VerifyCall: spec.VerifyCall,
		WebhookEvents: spec.WebhookEvents, EventsAPI: spec.EventsAPI, Chips: map[string]string{}}
	c.Display, c.Step1 = text(spec.Display, noMsg), text(spec.Step1, noMsg)
	if c.Display == "" {
		c.Display = d.Name
	}
	if c.Short == "" {
		c.Short = shortName(d.Name)
	}
	if c.Step1 == "" {
		c.Step1 = cmp.Or(spec.Step1Name, d.Name)
	}
	c.TermsLabel = text(spec.Terms, noMsg)
	c.Pricing = text(spec.Pricing, noMsg)
	if c.Pricing == "" {
		c.Pricing = providers.TextIn(d.Capabilities.Pricing, d.Capabilities.PricingText, l)
	}
	for k, m := range spec.Chips {
		c.Chips[k] = m.Text(l, args)
	}
	c.Title = text(spec.Title, mInfoTitle)
	heading := mInfoHeadingConnect
	if hasSecret(d, "api_key") {
		heading = mInfoHeadingAPIKey
	}
	c.Heading = text(spec.Heading, heading)
	c.Lead = text(spec.Lead, mInfoLeadDefault)
	c.ConnectionHint = text(spec.ConnectionHint, mInfoConnectionHintDefault)
	// DomainHint stays empty for API providers: their records come from the
	// provider's API once the key works (SetupError), so the stepper only
	// previews the domain step for SES and SMTP.
	c.KeyHint = text(spec.KeyHint, noMsg)
	c.Section = text(spec.Section, mInfoSectionDefault)
	if c.WebhookEvents == "" {
		c.WebhookEvents = mInfoWebhookEventsDefault.Text(l, nil)
	}
	return c
}

// keyHint is the hint that follows a rejected key, kept for the setup's
// later readers.
func keyHint(d providers.Descriptor) (string, i18n.Text, bool) {
	m := copyTable[d.Adapter].KeyHint
	if m == noMsg {
		return "", i18n.Text{}, false
	}
	s, t := say(m, i18n.Args{"name": d.Name})
	return s, t, true
}

// webhookEvents is what a registration registered, kept for the setup's
// later readers: the provider's event names, or the generic words.
func webhookEvents(d providers.Descriptor) (string, i18n.Text) {
	if e := copyTable[d.Adapter].WebhookEvents; e != "" {
		return e, i18n.Text{}
	}
	return say(mInfoWebhookEventsDefault, nil)
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
func chips(l i18n.Lang, d providers.Descriptor, c adapterCopy) []ConnectChip {
	caps := d.Capabilities
	out := []ConnectChip{}
	add := func(key, kind string, m i18n.Msg, args i18n.Args) {
		label := m.Text(l, args)
		if o, ok := c.Chips[key]; ok && kind != "no" {
			label = o
		}
		out = append(out, ConnectChip{Label: label, Kind: kind})
	}
	switch caps.Webhooks {
	case "":
		add("webhooks", "no", mChipWebhooks, nil)
	case "Bounce mailbox":
		add("webhooks", "partial", mChipBounceMailbox, nil)
	case "Registered":
		add("webhooks", "yes", mChipWebhooks, nil)
	default:
		add("webhooks", "yes", mChipWebhooksVia, i18n.Args{"via": caps.Webhooks})
	}
	switch caps.DomainAPI {
	case "full":
		add("domain", "yes", mChipDomainAPI, nil)
	case "dkim_only":
		add("domain", "partial", mChipDKIMAPI, nil)
	default:
		add("domain", "partial", mChipDomainManual, nil)
	}
	switch {
	case caps.SuppressionSync && strings.EqualFold(d.SuppressionMode, "Pull only"):
		add("suppression", "partial", mChipSuppressionPull, nil)
	case caps.SuppressionSync:
		add("suppression", "yes", mChipSuppressionSync, nil)
	case strings.EqualFold(d.SuppressionMode, "Pull only"):
		add("suppression", "partial", mChipSuppressionPull, nil)
	default:
		add("suppression", "partial", mChipSuppressionLocal, nil)
	}
	if caps.BatchSize > 1 && d.Kind != "smtp" {
		add("batch", "yes", mChipBatchN, i18n.Args{"n": caps.BatchSize})
	} else {
		add("batch", "no", mChipBatch, nil)
	}
	switch {
	case d.Kind == "self_hosted":
		add("dedicated", "yes", mChipDedicatedIP, nil)
	case d.Kind == "smtp":
		add("dedicated", "partial", mChipDedicatedDepends, nil)
	case caps.DedicatedIPs != "":
		add("dedicated", "partial", mChipDedicatedPaid, nil)
	default:
		add("dedicated", "no", mChipDedicatedIPs, nil)
	}
	switch caps.Sandbox {
	case "detection":
		add("sandbox", "yes", mChipSandboxDetection, nil)
	case "test token":
		add("sandbox", "yes", mChipSandboxToken, nil)
	case "mode":
		add("sandbox", "yes", mChipSandboxMode, nil)
	case "domain":
		add("sandbox", "yes", mChipSandboxDomain, nil)
	case "test addresses":
		add("sandbox", "yes", mChipSandboxAddresses, nil)
	case "":
		add("sandbox", "no", mChipSandbox, nil)
	default:
		add("sandbox", "yes", mChipSandboxKind, i18n.Args{"kind": caps.Sandbox})
	}
	if caps.QuotaDiscovery {
		add("quota", "yes", mChipQuotaDiscovery, nil)
	} else {
		add("quota", "partial", mChipQuotaManual, nil)
	}
	return out
}

// termsBadge is the catalog's badge about bulk marketing.
func termsBadge(l i18n.Lang, d providers.Descriptor, c adapterCopy) (string, string) {
	if c.Recommended {
		return mTermsRecommended.Text(l, nil), "ok"
	}
	if c.TermsLabel != "" {
		return c.TermsLabel, "ok"
	}
	switch d.Capabilities.Bulk {
	case providers.BulkAllowed:
		return mTermsBulkAllowed.Text(l, nil), "ok"
	case providers.BulkBroadcastStreams:
		return mTermsBroadcastStreams.Text(l, nil), "ok"
	case providers.BulkForbidden:
		return mTermsTransactionalOnly.Text(l, nil), "warn"
	}
	if d.Kind == "smtp" {
		return mTermsDependsOnServer.Text(l, nil), "neutral"
	}
	return mTermsOwnPolicy.Text(l, nil), "ok"
}
