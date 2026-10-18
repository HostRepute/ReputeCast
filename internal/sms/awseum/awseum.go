// Package awseum is the AWS End User Messaging SMS adapter (the
// pinpoint-sms-voice-v2 API, through aws-sdk-go-v2): texts from the
// workspace's own AWS account with a CloudFormation role or IAM keys,
// delivery events and two-way texts through an SNS topic ReputeCast
// subscribes to (SNS signatures checked against Amazon's certificates),
// phone numbers and sender IDs, registrations, and two-way sync with an
// opt-out list.
//
// AWS publishes SMS prices on its website, not in this API: prices are
// entered by hand.
package awseum

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	sv "github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	svtypes "github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/sms"
)

// Adapter is the registry key.
const Adapter = "aws_eum"

func init() { sms.Register(Descriptor()) }

// Regions where End User Messaging SMS runs, offered in the form.
var Regions = []string{"us-east-1", "us-east-2", "us-west-2", "ca-central-1", "eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "eu-north-1", "eu-south-2",
	"ap-south-1", "ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2", "sa-east-1", "me-central-1", "af-south-1"}

// Descriptor is the adapter's manifest.
func Descriptor() sms.Descriptor {
	return sms.Descriptor{
		Manifest: sms.Manifest{
			Adapter: Adapter, Name: "AWS End User Messaging",
			Summary: "Pay as you go · in your AWS account, with SES next door",
			Badges: []sms.Badge{
				{Capability: sms.CapTollFree, Support: sms.Yes}, {Capability: sms.Cap10DLC, Support: sms.Yes}, {Capability: sms.CapSenderIDs, Support: sms.Yes},
				{Capability: sms.CapReceipts, Support: sms.Yes}, {Capability: sms.CapInbound, Support: sms.Yes},
				{Capability: sms.CapOptOutSync, Support: sms.Yes},
				{Capability: sms.CapRegistration, Support: sms.Yes},
				{Capability: sms.CapPricing, Support: sms.No},
			},
			Fields: []sms.Field{
				{Key: "region", Label: "Region", Required: true, Options: Regions, Default: "us-east-1"},
				{Key: "access", Label: "Access", Options: []string{"role", "keys"}, Default: "role"},
				{Key: "role_arn", Label: "Role ARN", Help: "The role the CloudFormation template created. ReputeCast assumes it with the external ID."},
				{Key: "external_id", Label: "External ID"},
				{Key: "access_key_id", Label: "Access key ID"},
				{Key: "secret_access_key", Label: "Secret access key", Secret: true},
				{Key: "opt_out_list", Label: "Opt-out list", Default: "Default"},
			},
			Webhooks: "registered",
			Console: func(kind, _ string) string {
				switch kind {
				case "registration":
					return "https://console.aws.amazon.com/sms-voice/home#/registrations"
				case "numbers":
					return "https://console.aws.amazon.com/sms-voice/home#/phone-numbers"
				case "opt_outs":
					return "https://console.aws.amazon.com/sms-voice/home#/opt-out-lists"
				}
				return "https://console.aws.amazon.com/sms-voice/home"
			},
			Version: "1.0.0", Maintainer: "ReputeCast",
		},
		New: New,
	}
}

// Provider is an AWS account in a region.
type Provider struct {
	cfg     sms.Config
	region  string
	client  *sv.Client
	sns     *sns.Client
	optOuts string
	keyHint string
	http    *http.Client
}

// New opens the account.
func New(cfg sms.Config) (sms.Provider, error) {
	region := strings.TrimSpace(cfg.Value("region"))
	if region == "" {
		return nil, errors.New("aws_eum: the region is required")
	}
	p := &Provider{cfg: cfg, region: region, optOuts: cfg.Value("opt_out_list"), http: cfg.Client()}
	if p.optOuts == "" {
		p.optOuts = "Default"
	}
	awsCfg := aws.Config{Region: region, HTTPClient: p.http, Retryer: func() aws.Retryer { return aws.NopRetryer{} }}
	switch {
	case cfg.Value("access") == "role" && cfg.Value("role_arn") != "":
		base, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region), config.WithHTTPClient(p.http))
		if err != nil {
			return nil, fmt.Errorf("aws_eum: load AWS credentials to assume %s: %w", cfg.Value("role_arn"), err)
		}
		prov := stscreds.NewAssumeRoleProvider(sts.NewFromConfig(base), cfg.Value("role_arn"), func(o *stscreds.AssumeRoleOptions) {
			o.RoleSessionName = "reputecast-sms"
			if id := cfg.Value("external_id"); id != "" {
				o.ExternalID = aws.String(id)
			}
		})
		awsCfg.Credentials = aws.NewCredentialsCache(prov)
		p.keyHint = sms.Hint(cfg.Value("role_arn"), 0, 12)
	case cfg.Value("access_key_id") != "":
		awsCfg.Credentials = credentials.NewStaticCredentialsProvider(cfg.Value("access_key_id"), cfg.Value("secret_access_key"), "")
		p.keyHint = sms.Hint(cfg.Value("access_key_id"), 4, 4)
	default:
		return nil, errors.New("aws_eum: a role or access keys are required")
	}
	p.client = sv.NewFromConfig(awsCfg, func(o *sv.Options) {
		if cfg.BaseURL != "" {
			o.BaseEndpoint = aws.String(strings.TrimRight(cfg.BaseURL, "/") + "/sms-voice")
		}
	})
	p.sns = sns.NewFromConfig(awsCfg, func(o *sns.Options) {
		if cfg.BaseURL != "" {
			o.BaseEndpoint = aws.String(strings.TrimRight(cfg.BaseURL, "/") + "/sns")
		}
	})
	return p, nil
}

var (
	_ sms.SenderLister       = (*Provider)(nil)
	_ sms.WebhookRegistrar   = (*Provider)(nil)
	_ sms.WebhookHandshaker  = (*Provider)(nil)
	_ sms.OptOutSyncer       = (*Provider)(nil)
	_ sms.RegistrationReader = (*Provider)(nil)
)

// Adapter implements sms.Provider.
func (p *Provider) Adapter() string { return Adapter }

// mapErr turns AWS errors into the package's: refused credentials, an
// outage, or the provider's code.
func mapErr(op string, err error) error {
	if err == nil {
		return nil
	}
	var ae smithy.APIError
	if errors.As(err, &ae) {
		switch ae.ErrorCode() {
		case "UnrecognizedClientException", "InvalidClientTokenId", "InvalidSignatureException", "ExpiredTokenException", "AccessDeniedException", "SignatureDoesNotMatch":
			return fmt.Errorf("%w: %s", sms.ErrCredentialsRejected, ae.ErrorMessage())
		case "InternalServerException", "ServiceUnavailable", "ServiceUnavailableException":
			return fmt.Errorf("%w: %s", sms.ErrUnavailable, ae.ErrorMessage())
		}
		return &sms.CallError{Op: op, Status: 400, Code: ae.ErrorCode(), Message: ae.ErrorMessage()}
	}
	var re interface{ HTTPStatusCode() int }
	if errors.As(err, &re) && (re.HTTPStatusCode() >= 500 || re.HTTPStatusCode() == 429) {
		return fmt.Errorf("%w: %v", sms.ErrUnavailable, err)
	}
	if strings.Contains(err.Error(), "StatusCode: 5") || strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "no such host") {
		return fmt.Errorf("%w: %v", sms.ErrUnavailable, err)
	}
	return err
}

// Check implements sms.Provider: the account's tier shows the key works.
func (p *Provider) Check(ctx context.Context) (sms.Account, error) {
	out, err := p.client.DescribeAccountAttributes(ctx, &sv.DescribeAccountAttributesInput{})
	if err != nil {
		return sms.Account{}, mapErr("check", err)
	}
	tier := ""
	for _, a := range out.AccountAttributes {
		if a.Name == svtypes.AccountAttributeNameAccountTier {
			tier = aws.ToString(a.Value)
		}
	}
	name := "AWS account · " + p.region
	if tier == "SANDBOX" {
		name += " · sandbox"
	}
	return sms.Account{ID: p.keyHint, Name: name, Hint: p.keyHint, Live: tier != "SANDBOX"}, nil
}

// Send implements sms.Provider. The text carries ReputeCast's message id in
// its context, which comes back with every event.
func (p *Provider) Send(ctx context.Context, m sms.Message) (sms.SendResult, error) {
	in := &sv.SendTextMessageInput{DestinationPhoneNumber: aws.String(m.To), OriginationIdentity: aws.String(m.From), MessageBody: aws.String(m.Body),
		MessageType: svtypes.MessageTypePromotional, Context: map[string]string{"reputecast_message": m.ID.String()}}
	if cs := p.cfg.Value("configuration_set"); cs != "" {
		in.ConfigurationSetName = aws.String(cs)
	}
	out, err := p.client.SendTextMessage(ctx, in)
	if err != nil {
		var ae smithy.APIError
		if errors.As(err, &ae) {
			se := &sms.SendError{Code: ae.ErrorCode(), Message: ae.ErrorMessage()}
			switch ae.ErrorCode() {
			case "ValidationException":
				se.Reason = sms.ReasonInvalid
				if strings.Contains(ae.ErrorMessage(), "ORIGINATION") {
					se.Reason = sms.ReasonRejected
				}
				return sms.SendResult{}, se
			case "ConflictException":
				se.Reason = sms.ReasonRejected
				if strings.Contains(ae.ErrorMessage(), "OPTED_OUT") {
					se.Reason = sms.ReasonOptedOut
				}
				return sms.SendResult{}, se
			case "ThrottlingException", "ServiceQuotaExceededException":
				se.Temporary, se.Throttled = true, ae.ErrorCode() == "ThrottlingException"
				return sms.SendResult{}, se
			case "ResourceNotFoundException":
				se.Reason = sms.ReasonRejected
				return sms.SendResult{}, se
			}
		}
		return sms.SendResult{}, mapErr("send", err)
	}
	return sms.SendResult{ProviderMessageID: aws.ToString(out.MessageId)}, nil
}

// smsEvent is an SMS event as the configuration set publishes it to SNS.
type smsEvent struct {
	EventType                string            `json:"eventType"`
	EventTimestamp           int64             `json:"eventTimestamp"`
	MessageID                string            `json:"messageId"`
	OriginationPhoneNumber   string            `json:"originationPhoneNumber"`
	DestinationPhoneNumber   string            `json:"destinationPhoneNumber"`
	MessageStatus            string            `json:"messageStatus"`
	MessageStatusDescription string            `json:"messageStatusDescription"`
	TotalMessageParts        int               `json:"totalMessageParts"`
	TotalMessagePrice        float64           `json:"totalMessagePrice"`
	Context                  map[string]string `json:"context"`
}

// inboundText is a two-way text as it arrives on the SNS topic.
type inboundText struct {
	OriginationNumber string `json:"originationNumber"`
	DestinationNumber string `json:"destinationNumber"`
	MessageKeyword    string `json:"messageKeyword"`
	MessageBody       string `json:"messageBody"`
	InboundMessageID  string `json:"inboundMessageId"`
}

// eventReason maps a failed event type to what ReputeCast does about it.
func eventReason(t string) sms.Reason {
	switch t {
	case "TEXT_INVALID":
		return sms.ReasonInvalid
	case "TEXT_UNREACHABLE", "TEXT_CARRIER_UNREACHABLE", "TEXT_TTL_EXPIRED":
		return sms.ReasonUnreachable
	case "TEXT_BLOCKED", "TEXT_CARRIER_BLOCKED", "TEXT_SPAM", "TEXT_PROTECT_BLOCKED":
		return sms.ReasonFiltered
	case "TEXT_INVALID_MESSAGE":
		return sms.ReasonRejected
	}
	return sms.ReasonOther
}

// ParseWebhook implements sms.Provider: an SNS notification carrying an SMS
// event or a two-way text.
func (p *Provider) ParseWebhook(_ context.Context, _ string, body []byte) ([]sms.Event, error) {
	e, err := ParseEnvelope(body)
	if err != nil {
		return nil, err
	}
	if e.Type != "Notification" {
		return nil, nil
	}
	var ev smsEvent
	if json.Unmarshal([]byte(e.Message), &ev) == nil && ev.EventType != "" {
		at := time.UnixMilli(ev.EventTimestamp)
		if ev.EventTimestamp == 0 {
			at = p.cfg.Clock()
		}
		out := sms.Event{ProviderMessageID: ev.MessageID, To: ev.DestinationPhoneNumber, From: ev.OriginationPhoneNumber, OccurredAt: at,
			Segments: ev.TotalMessageParts, Summary: "eventType=" + ev.EventType}
		if ev.TotalMessagePrice > 0 {
			pr := ev.TotalMessagePrice
			out.Price = &pr
		}
		switch ev.EventType {
		case "TEXT_PENDING", "TEXT_QUEUED":
			out.Kind = sms.EventQueued
		case "TEXT_SUCCESSFUL":
			out.Kind = sms.EventSent
		case "TEXT_DELIVERED":
			out.Kind = sms.EventDelivered
		case "TEXT_INVALID", "TEXT_INVALID_MESSAGE", "TEXT_PROTECT_BLOCKED":
			out.Kind, out.Code, out.Reason = sms.EventFailed, ev.EventType, eventReason(ev.EventType)
		case "TEXT_UNREACHABLE", "TEXT_CARRIER_UNREACHABLE", "TEXT_BLOCKED", "TEXT_CARRIER_BLOCKED", "TEXT_SPAM", "TEXT_UNKNOWN", "TEXT_TTL_EXPIRED":
			out.Kind, out.Code, out.Reason = sms.EventUndelivered, ev.EventType, eventReason(ev.EventType)
		default:
			return nil, nil
		}
		if id := ev.Context["reputecast_message"]; id != "" {
			if u, err := uuid.Parse(id); err == nil {
				out.MessageID = &u
			}
		}
		return []sms.Event{out}, nil
	}
	var in inboundText
	if json.Unmarshal([]byte(e.Message), &in) == nil && in.OriginationNumber != "" {
		at, err := time.Parse(time.RFC3339, e.Timestamp)
		if err != nil {
			at = p.cfg.Clock()
		}
		return []sms.Event{{Kind: sms.EventInbound, ProviderMessageID: in.InboundMessageID, From: in.OriginationNumber, To: in.DestinationNumber, Body: in.MessageBody,
			OccurredAt: at, Summary: "Inbound · messageBody=" + sms.Quote(in.MessageBody)}}, nil
	}
	return nil, nil
}

// numberKind maps the API's number types.
func numberKind(t svtypes.NumberType) string {
	switch t {
	case svtypes.NumberTypeTollFree:
		return "toll_free"
	case svtypes.NumberTypeTenDlc:
		return "10dlc"
	case svtypes.NumberTypeLongCode:
		return "local"
	}
	return "mobile"
}

// Senders implements sms.SenderLister: phone numbers and sender IDs.
func (p *Provider) Senders(ctx context.Context) ([]sms.Sender, error) {
	nums, err := p.client.DescribePhoneNumbers(ctx, &sv.DescribePhoneNumbersInput{})
	if err != nil {
		return nil, mapErr("numbers", err)
	}
	out := []sms.Sender{}
	for _, n := range nums.PhoneNumbers {
		if n.NumberType == svtypes.NumberTypeSimulator || n.Status != svtypes.NumberStatusActive {
			continue
		}
		c := aws.ToString(n.IsoCountryCode)
		s := sms.Sender{Kind: numberKind(n.NumberType), Value: aws.ToString(n.PhoneNumber), Countries: []string{c}, Throughput: 1, ProviderRef: aws.ToString(n.PhoneNumberId)}
		if s.Kind == "toll_free" {
			s.Throughput = 3
			if c == "US" {
				s.Countries = []string{"US", "CA"}
			}
		}
		out = append(out, s)
	}
	ids, err := p.client.DescribeSenderIds(ctx, &sv.DescribeSenderIdsInput{})
	if err != nil {
		return nil, mapErr("sender_ids", err)
	}
	for _, id := range ids.SenderIds {
		out = append(out, sms.Sender{Kind: "sender_id", Value: aws.ToString(id.SenderId), Countries: []string{aws.ToString(id.IsoCountryCode)}, Throughput: 20})
	}
	return out, nil
}

// RegisterWebhooks implements sms.WebhookRegistrar: an SNS topic with
// ReputeCast's address subscribed to it, a configuration set publishing
// every SMS event there, and two-way texts on each number sent there too.
func (p *Provider) RegisterWebhooks(ctx context.Context, hooks sms.Webhooks, senders []sms.Sender) (sms.WebhookSetup, error) {
	topic, err := p.sns.CreateTopic(ctx, &sns.CreateTopicInput{Name: aws.String("reputecast-sms")})
	if err != nil {
		return sms.WebhookSetup{}, mapErr("topic", err)
	}
	arn := aws.ToString(topic.TopicArn)
	if _, err := p.sns.Subscribe(ctx, &sns.SubscribeInput{TopicArn: aws.String(arn), Protocol: aws.String("https"), Endpoint: aws.String(hooks.StatusURL)}); err != nil {
		return sms.WebhookSetup{}, mapErr("subscribe", err)
	}
	const set = "reputecast-sms"
	if _, err := p.client.CreateConfigurationSet(ctx, &sv.CreateConfigurationSetInput{ConfigurationSetName: aws.String(set)}); err != nil {
		var ae smithy.APIError
		if !errors.As(err, &ae) || ae.ErrorCode() != "ConflictException" {
			return sms.WebhookSetup{}, mapErr("configuration_set", err)
		}
	}
	if _, err := p.client.CreateEventDestination(ctx, &sv.CreateEventDestinationInput{ConfigurationSetName: aws.String(set), EventDestinationName: aws.String("reputecast"),
		MatchingEventTypes: []svtypes.EventType{svtypes.EventTypeTextAll}, SnsDestination: &svtypes.SnsDestination{TopicArn: aws.String(arn)}}); err != nil {
		var ae smithy.APIError
		if !errors.As(err, &ae) || ae.ErrorCode() != "ConflictException" {
			return sms.WebhookSetup{}, mapErr("event_destination", err)
		}
	}
	for _, s := range senders {
		if s.ProviderRef == "" || !strings.HasPrefix(s.Value, "+") {
			continue
		}
		if _, err := p.client.UpdatePhoneNumber(ctx, &sv.UpdatePhoneNumberInput{PhoneNumberId: aws.String(s.ProviderRef), TwoWayEnabled: aws.Bool(true),
			TwoWayChannelArn: aws.String(arn)}); err != nil {
			return sms.WebhookSetup{}, mapErr("two_way", err)
		}
	}
	return sms.WebhookSetup{Settings: map[string]string{"topic_arn": arn, "configuration_set": set}}, nil
}

// OptOuts implements sms.OptOutReader: the numbers on the opt-out list.
func (p *Provider) OptOuts(ctx context.Context) ([]sms.OptOut, error) {
	out := []sms.OptOut{}
	var next *string
	for range 20 {
		res, err := p.client.DescribeOptedOutNumbers(ctx, &sv.DescribeOptedOutNumbersInput{OptOutListName: aws.String(p.optOuts), NextToken: next})
		if err != nil {
			return nil, mapErr("opt_outs", err)
		}
		for _, n := range res.OptedOutNumbers {
			o := sms.OptOut{Phone: aws.ToString(n.OptedOutNumber)}
			if n.EndUserOptedOut {
				o.Keyword = "STOP"
			}
			if n.OptedOutTimestamp != nil {
				o.At = *n.OptedOutTimestamp
			}
			out = append(out, o)
		}
		if res.NextToken == nil {
			break
		}
		next = res.NextToken
	}
	return out, nil
}

// AddOptOuts implements sms.OptOutSyncer.
func (p *Provider) AddOptOuts(ctx context.Context, phones []string) error {
	for _, ph := range phones {
		if _, err := p.client.PutOptedOutNumber(ctx, &sv.PutOptedOutNumberInput{OptOutListName: aws.String(p.optOuts), OptedOutNumber: aws.String(ph)}); err != nil {
			return mapErr("opt_out_add", err)
		}
	}
	return nil
}

// RemoveOptOut implements sms.OptOutSyncer (a person texted START).
func (p *Provider) RemoveOptOut(ctx context.Context, phone string) error {
	_, err := p.client.DeleteOptedOutNumber(ctx, &sv.DeleteOptedOutNumberInput{OptOutListName: aws.String(p.optOuts), OptedOutNumber: aws.String(phone)})
	return mapErr("opt_out_remove", err)
}

func regStatus(s svtypes.RegistrationStatus) string {
	switch s {
	case svtypes.RegistrationStatusComplete:
		return "approved"
	case svtypes.RegistrationStatusRequiresUpdates, svtypes.RegistrationStatusClosed, svtypes.RegistrationStatusDeleted:
		return "rejected"
	case svtypes.RegistrationStatusCreated:
		return "not_registered"
	}
	return "pending"
}

func regKind(t string) string {
	switch {
	case strings.Contains(t, "TOLL_FREE"):
		return "toll_free"
	case strings.Contains(t, "BRAND"):
		return "brand"
	case strings.Contains(t, "CAMPAIGN"):
		return "campaign"
	}
	return "sender_id"
}

// Registrations implements sms.RegistrationReader: each number's and
// sender ID's registration, by its registration id.
func (p *Provider) Registrations(ctx context.Context, senders []sms.Sender) ([]sms.Registration, error) {
	want := map[string]bool{}
	for _, s := range senders {
		want[s.Value] = true
	}
	owner := map[string]string{}
	nums, err := p.client.DescribePhoneNumbers(ctx, &sv.DescribePhoneNumbersInput{})
	if err != nil {
		return nil, mapErr("numbers", err)
	}
	for _, n := range nums.PhoneNumbers {
		if id := aws.ToString(n.RegistrationId); id != "" && want[aws.ToString(n.PhoneNumber)] {
			owner[id] = aws.ToString(n.PhoneNumber)
		}
	}
	ids, err := p.client.DescribeSenderIds(ctx, &sv.DescribeSenderIdsInput{})
	if err != nil {
		return nil, mapErr("sender_ids", err)
	}
	country := map[string]string{}
	for _, s := range ids.SenderIds {
		if id := aws.ToString(s.RegistrationId); id != "" && want[aws.ToString(s.SenderId)] {
			owner[id], country[id] = aws.ToString(s.SenderId), aws.ToString(s.IsoCountryCode)
		}
	}
	if len(owner) == 0 {
		return []sms.Registration{}, nil
	}
	regIDs := make([]string, 0, len(owner))
	for id := range owner {
		regIDs = append(regIDs, id)
	}
	res, err := p.client.DescribeRegistrations(ctx, &sv.DescribeRegistrationsInput{RegistrationIds: regIDs})
	if err != nil {
		return nil, mapErr("registrations", err)
	}
	out := []sms.Registration{}
	for _, r := range res.Registrations {
		id := aws.ToString(r.RegistrationId)
		kind := regKind(aws.ToString(r.RegistrationType))
		st := regStatus(r.RegistrationStatus)
		if kind == "toll_free" && st == "approved" {
			st = "verified"
		}
		g := sms.Registration{Sender: owner[id], Kind: kind, Country: country[id], Status: st, Since: r.CreatedTimestamp, Details: map[string]string{"type": aws.ToString(r.RegistrationType)}}
		out = append(out, g)
	}
	return out, nil
}
