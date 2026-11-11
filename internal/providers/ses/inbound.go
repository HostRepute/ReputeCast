package ses

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/received"
)

// Receiving mail (M58). Amazon SES receives mail for a domain whose MX
// points at its regional receiving endpoint and runs the active receipt
// rule set: a rule with an SNS action publishes each message, base64
// encoded, to a topic people choose, and the topic posts it to the inbound
// hook (https://docs.aws.amazon.com/ses/latest/dg/receiving-email-action-sns.html).
// Setting the rule up stays in the AWS console (InboundManual): people
// enter the topic's ARN, and the hook confirms that topic's subscription
// and accepts that topic's messages only, apart from the topics of the
// connection's own events.

// SettingTopicARN is the route setting naming the receipt rule's topic.
const SettingTopicARN = "topic_arn"

// MaxReceived is the largest message an SNS action publishes; SES bounces
// larger ones.
const MaxReceived = 150 << 10

// receivingRegions are the regions with an email receiving endpoint,
// inbound-smtp.<region>.amazonaws.com
// (https://docs.aws.amazon.com/general/latest/gr/ses.html#ses_inbound_endpoints,
// checked 2026-11-11).
var receivingRegions = []string{
	"us-east-1", "us-east-2", "us-west-1", "us-west-2", "af-south-1", "ap-southeast-3", "ap-south-1", "ap-northeast-3",
	"ap-northeast-2", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ca-central-1", "eu-central-1", "eu-west-1",
	"eu-west-2", "eu-south-1", "eu-west-3", "eu-north-1", "il-central-1", "me-south-1", "sa-east-1",
}

var (
	_ providers.InboundReceiver   = (*Provider)(nil)
	_ providers.InboundHandshaker = (*Provider)(nil)
)

// receivingRegion is where the route receives: the topic's region once its
// ARN is entered (the rule, its topic and the MX are in one region), else
// the connection's.
func (p *Provider) receivingRegion(route providers.InboundRoute) string {
	if parts := strings.Split(route.Settings[SettingTopicARN], ":"); len(parts) >= 6 && parts[0] == "arn" && parts[3] != "" {
		return parts[3]
	}
	return p.s.Region
}

// InboundSetup implements providers.InboundReceiver.
func (p *Provider) InboundSetup(route providers.InboundRoute) providers.InboundSetup {
	region := p.receivingRegion(route)
	args := i18n.Args{"domain": route.Domain, "url": route.URL, "region": region, "size": MaxReceived >> 10}
	setup := providers.InboundSetup{
		Settings: []providers.Field{{Key: SettingTopicARN, Required: true}},
	}
	setup.Settings[0].Label, setup.Settings[0].LabelText = say(mInboundTopicLabel, nil)
	setup.Settings[0].Help, setup.Settings[0].HelpText = say(mInboundTopicHelp, i18n.Args{"region": region})
	if slices.Contains(receivingRegions, region) {
		setup.MX = []providers.DNSRecord{{Purpose: "inbound", Type: "MX", Name: route.Domain, Value: "10 inbound-smtp." + region + ".amazonaws.com"}}
	} else {
		setup.Steps = append(setup.Steps, mInboundNoReceiving.Keep(i18n.Args{"region": region, "regions": strings.Join(receivingRegions, ", ")}))
	}
	setup.Steps = append(setup.Steps, mInboundRule.Keep(args), mInboundTopic.Keep(args), mInboundSubscribe.Keep(args), mInboundSize.Keep(args))
	return setup
}

// inboundEnvelope parses an SNS message to the inbound hook and checks it
// comes from the route's topic.
func inboundEnvelope(route providers.InboundRoute, body []byte) (Envelope, error) {
	e, err := parseEnvelope(body)
	if err != nil {
		return e, err
	}
	topic := strings.TrimSpace(route.Settings[SettingTopicARN])
	if topic == "" {
		return e, fmt.Errorf("%w: the route has no SNS topic ARN yet", providers.ErrBadSignature)
	}
	if e.TopicArn != topic {
		return e, fmt.Errorf("%w: SNS topic %q isn't the route's", providers.ErrBadSignature, e.TopicArn)
	}
	return e, nil
}

// HandshakeInbound implements providers.InboundHandshaker: it confirms the
// subscription of the route's topic, once SNS signed it, and refuses every
// other topic's.
func (p *Provider) HandshakeInbound(ctx context.Context, route providers.InboundRoute, _ *http.Request, body []byte) (bool, []byte, error) {
	e, err := parseEnvelope(body)
	if err != nil || e.Type == "Notification" {
		return false, nil, nil
	}
	if _, err := inboundEnvelope(route, body); err != nil {
		return true, nil, err
	}
	if err := p.verifySignature(ctx, e); err != nil {
		return true, nil, err
	}
	return true, nil, p.confirm(ctx, e)
}

// VerifyInbound implements providers.InboundReceiver: SNS signed the
// message and it comes from the route's topic.
func (p *Provider) VerifyInbound(ctx context.Context, route providers.InboundRoute, _ *http.Request, body []byte) error {
	e, err := inboundEnvelope(route, body)
	if err != nil {
		return err
	}
	return p.verifySignature(ctx, e)
}

// receivedNotification is an SES email receiving notification
// (https://docs.aws.amazon.com/ses/latest/dg/receiving-email-notifications-contents.html).
type receivedNotification struct {
	NotificationType string `json:"notificationType"`
	Receipt          struct {
		Timestamp    string   `json:"timestamp"`
		Recipients   []string `json:"recipients"`
		SpamVerdict  verdict  `json:"spamVerdict"`
		VirusVerdict verdict  `json:"virusVerdict"`
		SPFVerdict   verdict  `json:"spfVerdict"`
		DKIMVerdict  verdict  `json:"dkimVerdict"`
		DMARCVerdict verdict  `json:"dmarcVerdict"`
		Action       struct {
			Type     string `json:"type"`
			Encoding string `json:"encoding"`
		} `json:"action"`
	} `json:"receipt"`
	Mail struct {
		Timestamp     string                         `json:"timestamp"`
		MessageID     string                         `json:"messageId"`
		Headers       []struct{ Name, Value string } `json:"headers"`
		CommonHeaders struct {
			From      []string `json:"from"`
			To        []string `json:"to"`
			MessageID string   `json:"messageId"`
			Subject   string   `json:"subject"`
		} `json:"commonHeaders"`
	} `json:"mail"`
	Content *string `json:"content"`
}

type verdict struct {
	Status string `json:"status"`
}

// ParseInbound implements providers.InboundReceiver: a Received
// notification carries the whole message (content) when an SNS action
// published it; one from another action carries the headers only.
func (p *Provider) ParseInbound(_ context.Context, _ providers.InboundRoute, _ *http.Request, body []byte) ([]providers.InboundMessage, error) {
	e, err := parseEnvelope(body)
	if err != nil {
		return nil, err
	}
	if e.Type != "Notification" {
		return []providers.InboundMessage{}, nil
	}
	var n receivedNotification
	if err := json.Unmarshal([]byte(e.Message), &n); err != nil || n.NotificationType != "Received" {
		// Something else on the topic (SES's subscription test): no mail.
		return []providers.InboundMessage{}, nil
	}
	m := providers.InboundMessage{
		ProviderMessageID: n.Mail.MessageID,
		Recipients:        []string{},
		SPF:               received.Verdict(n.Receipt.SPFVerdict.Status),
		DKIM:              received.Verdict(n.Receipt.DKIMVerdict.Status),
		DMARC:             received.Verdict(n.Receipt.DMARCVerdict.Status),
		ReceivedAt:        parseTime(n.Mail.Timestamp),
	}
	for _, r := range n.Receipt.Recipients {
		m.Recipients = append(m.Recipients, strings.ToLower(strings.TrimSpace(r)))
	}
	switch {
	case strings.EqualFold(n.Receipt.SpamVerdict.Status, "FAIL") || strings.EqualFold(n.Receipt.VirusVerdict.Status, "FAIL"):
		m.Spam = "spam"
	case strings.EqualFold(n.Receipt.SpamVerdict.Status, "PASS"):
		m.Spam = "pass"
	}
	if m.ReceivedAt.IsZero() {
		m.ReceivedAt = parseTime(n.Receipt.Timestamp)
	}
	if n.Content != nil && *n.Content != "" {
		m.Raw = content(*n.Content, n.Receipt.Action.Encoding)
		return []providers.InboundMessage{m}, nil
	}
	// No content: a rule whose action isn't SNS notified the topic.
	h := n.Mail.CommonHeaders
	m.Subject, m.MessageID = h.Subject, h.MessageID
	if len(h.From) > 0 {
		m.From = h.From[0]
	}
	m.To = append([]string{}, h.To...)
	m.Headers = map[string][]string{}
	for _, x := range n.Mail.Headers {
		m.Headers[x.Name] = append(m.Headers[x.Name], x.Value)
	}
	m.InReplyTo = received.Header(m.Headers, "In-Reply-To")
	m.References = received.MessageIDs(received.Header(m.Headers, "References"))
	return []providers.InboundMessage{m}, nil
}

// content decodes the message an SNS action published: base64 with the
// rule's Base64 encoding (the setup asks for it), the message itself with
// UTF-8. A notification that doesn't say is read as base64 when it is.
func content(s, encoding string) []byte {
	enc := strings.ToUpper(strings.ReplaceAll(encoding, "-", ""))
	if enc == "UTF8" {
		return []byte(s)
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b
	}
	return []byte(s)
}
