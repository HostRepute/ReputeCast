// Package ses is the Amazon SES adapter (SES API v2 through aws-sdk-go-v2),
// with events delivered through SNS.
package ses

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/mimemsg"
	"github.com/hostrepute/reputecast/internal/version"
)

// Adapter is the registry key.
const Adapter = "ses"

// Tag names SES echoes back on events.
const (
	TagMessage = "rc_message"
	TagStream  = "rc_stream"
)

// BatchSize is how many messages one Send call handles (in parallel).
const BatchSize = 50

const parallel = 10

// Settings are the non-secret connection settings.
type Settings struct {
	Region           string `json:"region"`
	ConfigurationSet string `json:"configuration_set,omitempty"`
	// ConfigurationSets names a configuration set per stream ("broadcast",
	// "automation", "system"), as provisioned by the connect flow; streams
	// without one use ConfigurationSet.
	ConfigurationSets map[string]string `json:"configuration_sets,omitempty"`
	// Access is "keys" (access key in the credentials) or "role" (assume
	// RoleARN with ExternalID from the instance's own AWS credentials, as
	// set up by the CloudFormation template).
	Access     string `json:"access,omitempty"`
	RoleARN    string `json:"role_arn,omitempty"`
	ExternalID string `json:"external_id,omitempty"`
	// TopicARNs restricts which SNS topics may post events.
	TopicARNs []string `json:"topic_arns,omitempty"`
	// Tenant is the SES tenant of the workspace (PR-11): its own suppression
	// list and reputation, so a bounce in this workspace never stops mail of
	// another workspace or app on the same AWS account. Sends name it.
	Tenant string `json:"tenant,omitempty"`
	// Events is how SES events come back: "" (SNS calls the webhook) or
	// "queue" (an SQS queue ReputeCast polls, PR-4), with the queue's URL
	// and ARN once provisioned.
	Events   string `json:"events,omitempty"`
	QueueURL string `json:"queue_url,omitempty"`
	QueueARN string `json:"queue_arn,omitempty"`
	// AccountID is the AWS account (the tenant's resource ARNs).
	AccountID string `json:"account_id,omitempty"`
	// Endpoint overrides the SES API endpoint (tests, VPC endpoints).
	Endpoint string `json:"endpoint,omitempty"`
}

// Credentials are write-only.
type Credentials struct {
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	SessionToken    string `json:"session_token,omitempty"`
}

func init() { providers.Register(Descriptor()) }

// Descriptor describes the adapter for the catalog.
func Descriptor() providers.Descriptor {
	return providers.Descriptor{
		Adapter: Adapter,
		Name:    "Amazon SES",
		Kind:    "api",
		Capabilities: providers.Capabilities{
			Bulk:            providers.BulkAllowed,
			Webhooks:        "SNS",
			DomainAPI:       "full",
			SuppressionSync: true,
			BatchSize:       BatchSize,
			Sandbox:         "detection",
			QuotaDiscovery:  true,
			RateDiscovery:   true,
			EventQueue:      "SQS",
			Tenants:         true,
			// Pricing, DedicatedIPs and every other text people read (the
			// terms' note, the Adapters tab's words, the fields' labels and
			// hints) are in messages/en.json: WithMessages fills them.
		},
		Version:    version.Version,
		Maintainer: "HostRepute",
		Terms:      []providers.Terms{{Bulk: providers.BulkAllowed, Streams: []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}}},
		Fields: []providers.Field{
			{Key: "region", Required: true, Options: []string{"us-east-1", "us-east-2", "us-west-1", "us-west-2", "ca-central-1", "eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "eu-north-1", "eu-south-1", "ap-south-1", "ap-northeast-1", "ap-northeast-2", "ap-northeast-3", "ap-southeast-1", "ap-southeast-2", "sa-east-1", "me-south-1", "af-south-1", "il-central-1"}},
			{Key: "access", Required: true, Options: []string{"role", "keys"}},
			{Key: "role_arn"},
			{Key: "access_key_id", Secret: true},
			{Key: "secret_access_key", Secret: true},
			{Key: "configuration_set"},
		},
		New: New,
	}.WithMessages(msgs)
}

// Provider talks to SES in one region.
type Provider struct {
	cfg    providers.Config
	s      Settings
	client *sesv2.Client
	http   *http.Client
	// aws is the resolved SDK configuration, reused by provisioning (SNS)
	// and region scans (provision.go).
	aws aws.Config
	// callerARN is who the credentials belong to, read with AccountID.
	callerARN string
}

var _ providers.Provider = (*Provider)(nil)

// New builds the adapter. Credentials are resolved lazily by the SDK.
func New(cfg providers.Config) (providers.Provider, error) {
	p := &Provider{cfg: cfg, http: cfg.HTTPClient}
	if len(cfg.Settings) > 0 {
		if err := json.Unmarshal(cfg.Settings, &p.s); err != nil {
			return nil, fmt.Errorf("ses settings: %w", err)
		}
	}
	var c Credentials
	if len(cfg.Credentials) > 0 {
		if err := json.Unmarshal(cfg.Credentials, &c); err != nil {
			return nil, fmt.Errorf("ses credentials: %w", err)
		}
	}
	if p.s.Region == "" {
		return nil, errors.New("ses: region is required")
	}
	if p.http == nil {
		p.http = &http.Client{Timeout: 30 * time.Second}
	}
	awsCfg := aws.Config{Region: p.s.Region, HTTPClient: p.http, Retryer: func() aws.Retryer { return aws.NopRetryer{} }}
	switch {
	case p.s.Access == "role" && p.s.RoleARN != "":
		base, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(p.s.Region), config.WithHTTPClient(sdkHTTP(p.http)))
		if err != nil {
			return nil, fmt.Errorf("ses: load AWS credentials to assume %s: %w", p.s.RoleARN, err)
		}
		prov := stscreds.NewAssumeRoleProvider(sts.NewFromConfig(base), p.s.RoleARN, func(o *stscreds.AssumeRoleOptions) {
			o.RoleSessionName = "reputecast"
			if p.s.ExternalID != "" {
				o.ExternalID = aws.String(p.s.ExternalID)
			}
		})
		awsCfg.Credentials = aws.NewCredentialsCache(prov)
	case c.AccessKeyID != "":
		awsCfg.Credentials = credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.SecretAccessKey, c.SessionToken)
	default:
		return nil, errors.New("ses: access keys or a role are required")
	}
	p.aws = awsCfg
	p.client = sesv2.NewFromConfig(awsCfg, func(o *sesv2.Options) {
		if p.s.Endpoint != "" {
			o.BaseEndpoint = aws.String(p.s.Endpoint)
		}
	})
	return p, nil
}

// sdkHTTP is the client the AWS SDK's config loader calls through: the
// adapter's own transport (netguard's dialer, proxy, TLS settings) in a
// client the loader can add a custom CA bundle to (AWS_CA_BUNDLE or the
// shared config's ca_bundle). Given a plain *http.Client, the loader fails
// whenever a bundle is set.
func sdkHTTP(c *http.Client) config.HTTPClient {
	tr, ok := c.Transport.(*http.Transport)
	if c.Transport != nil && !ok {
		return c
	}
	b := awshttp.NewBuildableClient().WithTimeout(c.Timeout)
	if tr == nil {
		return b
	}
	return b.WithTransportOptions(func(t *http.Transport) {
		t.Proxy, t.DialContext, t.DialTLSContext = tr.Proxy, tr.DialContext, tr.DialTLSContext
		if tr.TLSClientConfig != nil {
			t.TLSClientConfig = tr.TLSClientConfig.Clone()
		}
	})
}

// Adapter implements providers.Provider.
func (p *Provider) Adapter() string { return Adapter }

// Capabilities implements providers.Provider.
func (p *Provider) Capabilities() providers.Capabilities { return Descriptor().Capabilities }

// Send implements providers.Provider: SendEmail with raw MIME per message,
// up to ten at a time. A batch fails as a whole only when every message
// failed for the same account-level reason (outage, credentials).
func (p *Provider) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	return p.send(ctx, msgs, nil)
}

// SendWithExtras implements providers.ExtrasSender: copies and attachments
// (transactional mail) go in the raw message, every recipient in the
// destination.
func (p *Provider) SendWithExtras(ctx context.Context, msgs []providers.Message, extras []providers.MessageExtras) ([]providers.SendResult, error) {
	return p.send(ctx, msgs, extras)
}

func (p *Provider) send(ctx context.Context, msgs []providers.Message, extras []providers.MessageExtras) ([]providers.SendResult, error) {
	if len(msgs) > BatchSize {
		return nil, fmt.Errorf("ses: batch of %d exceeds %d", len(msgs), BatchSize)
	}
	results := make([]providers.SendResult, len(msgs))
	callErrs := make([]error, len(msgs))
	sem := make(chan struct{}, parallel)
	var wg sync.WaitGroup
	for i := range msgs {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			var x providers.MessageExtras
			if i < len(extras) {
				x = extras[i]
			}
			results[i], callErrs[i] = p.sendOne(ctx, msgs[i], x)
		}(i)
	}
	wg.Wait()
	var first error
	allFailed := len(msgs) > 0
	for i, err := range callErrs {
		if err == nil {
			allFailed = false
			continue
		}
		if first == nil {
			first = err
		}
		var ce *providers.CallError
		errors.As(err, &ce)
		results[i].Err = &providers.SendError{Temporary: true, Throttled: ce != nil && ce.Throttled, Code: errCode(ce), Message: err.Error()}
	}
	if allFailed && first != nil {
		return nil, first
	}
	return results, nil
}

func errCode(ce *providers.CallError) string {
	if ce == nil {
		return "Error"
	}
	return ce.Code
}

// sendOne returns a result for message-level outcomes and an error for
// account- or service-level failures.
func (p *Provider) sendOne(ctx context.Context, m providers.Message, x providers.MessageExtras) (providers.SendResult, error) {
	res := providers.SendResult{MessageID: m.ID}
	raw, _, err := mimemsg.BuildWithExtras(m, x, time.Now(), nil)
	if err != nil {
		// A message that can't be rendered fails on its own; the rest of the
		// batch still goes out.
		res.Err = &providers.SendError{Code: "Render", Message: err.Error()}
		return res, nil //nolint:nilerr // reported on the result, not as a batch error
	}
	in := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(m.From.Address),
		Destination:      &types.Destination{ToAddresses: []string{m.To.Address}, CcAddresses: addresses(x.Cc), BccAddresses: addresses(x.Bcc)},
		Content:          &types.EmailContent{Raw: &types.RawMessage{Data: raw}},
		EmailTags: []types.MessageTag{
			{Name: aws.String(TagMessage), Value: aws.String(m.ID.String())},
			{Name: aws.String(TagStream), Value: aws.String(string(m.Stream))},
		},
	}
	for k, v := range m.Tags {
		if len(in.EmailTags) < 10 && validTag(k) && validTag(v) {
			in.EmailTags = append(in.EmailTags, types.MessageTag{Name: aws.String(k), Value: aws.String(v)})
		}
	}
	if cs := p.configurationSet(m.Stream); cs != "" {
		in.ConfigurationSetName = aws.String(cs)
	}
	// The workspace's tenant: its own suppression list and reputation.
	in.TenantName = p.tenant()
	out, err := p.client.SendEmail(ctx, in)
	if err != nil {
		ce := callError("SendEmail", err)
		if ce.Auth || ce.Temporary {
			return res, ce
		}
		res.Err = &providers.SendError{Code: ce.Code, Message: ce.Message}
		return res, nil
	}
	res.ProviderMessageID = aws.ToString(out.MessageId)
	return res, nil
}

// addresses lists the bare addresses of a destination.
func addresses(as []mail.Address) []string {
	if len(as) == 0 {
		return nil
	}
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.Address
	}
	return out
}

// validTag reports whether s is allowed in an SES message tag.
func validTag(s string) bool {
	if s == "" || len(s) > 256 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

// callError maps an SDK error to providers.CallError.
func callError(op string, err error) *providers.CallError {
	ce := &providers.CallError{Op: op, Message: err.Error(), Cause: err}
	var re *awshttp.ResponseError
	if errors.As(err, &re) {
		ce.Status = re.HTTPStatusCode()
	}
	var ae smithy.APIError
	if errors.As(err, &ae) {
		ce.Code, ce.Message = ae.ErrorCode(), ae.ErrorMessage()
	} else if ce.Status == 0 {
		// No response at all: DNS, connection, timeout.
		ce.Temporary = true
		return ce
	}
	switch ce.Code {
	case "UnrecognizedClientException", "InvalidSignatureException", "AccessDeniedException", "ExpiredTokenException",
		"InvalidClientTokenId", "SignatureDoesNotMatch", "AccessDenied", "IncompleteSignature", "MissingAuthenticationToken":
		ce.Auth = true
	case "TooManyRequestsException", "ThrottlingException", "Throttling", "LimitExceededException":
		ce.Temporary, ce.Throttled = true, true
	case "SendingPausedException", "AccountSuspendedException", "ServiceUnavailable", "ServiceUnavailableException",
		"InternalFailure", "InternalServiceErrorException", "InternalServerError":
		ce.Temporary = true
	}
	switch {
	case ce.Status == http.StatusUnauthorized || ce.Status == http.StatusForbidden && !ce.Throttled:
		ce.Auth = true
	case ce.Status == http.StatusTooManyRequests:
		ce.Temporary, ce.Throttled = true, true
	case ce.Status >= 500:
		ce.Temporary = true
	}
	if ce.Auth {
		ce.Temporary = false
	}
	return ce
}

// Verify implements providers.Verifier.
func (p *Provider) Verify(ctx context.Context) (providers.Quota, error) { return p.Quota(ctx) }

// Quota implements providers.QuotaReporter from GetAccount.
func (p *Provider) Quota(ctx context.Context) (providers.Quota, error) {
	out, err := p.client.GetAccount(ctx, &sesv2.GetAccountInput{})
	if err != nil {
		return providers.Quota{}, callError("GetAccount", err)
	}
	q := providers.Quota{Kind: "rolling_24h", Sandbox: !out.ProductionAccessEnabled}
	if out.SendQuota != nil {
		if out.SendQuota.Max24HourSend > 0 {
			q.Max = int64(out.SendQuota.Max24HourSend)
		}
		q.Used = int64(out.SendQuota.SentLast24Hours)
		q.MaxRate = out.SendQuota.MaxSendRate
	}
	mode := "production"
	if q.Sandbox {
		mode = "sandbox"
	}
	enforcement := "none"
	switch strings.ToUpper(aws.ToString(out.EnforcementStatus)) {
	case "PROBATION":
		enforcement = "probation"
	case "SHUTDOWN":
		enforcement = "shutdown"
	}
	if !out.SendingEnabled {
		enforcement = "disabled"
	}
	// "Production · no enforcement"
	q.Account, q.AccountText = say(mAccount, i18n.Args{"mode": mode, "enforcement": enforcement})
	if out.VdmAttributes != nil && out.VdmAttributes.VdmEnabled == types.FeatureStatusEnabled {
		q.VDM = true
	}
	return q, nil
}

// SuppressionScope implements providers.SuppressionScoper: the tenant's own
// list when the workspace has a tenant; otherwise the account-level list,
// which ReputeCast only reads, because every app on the AWS account shares it.
func (p *Provider) SuppressionScope() providers.SuppressionScope {
	if p.s.Tenant != "" {
		return providers.SuppressionScope{Scope: providers.SuppressionScopeTenant, Writable: true}
	}
	return providers.SuppressionScope{Scope: providers.SuppressionScopeAccount}
}

// tenant names the tenant for suppression calls, nil for the account list.
func (p *Provider) tenant() *string {
	if p.s.Tenant == "" {
		return nil
	}
	return aws.String(p.s.Tenant)
}

// ListSuppressions implements providers.SuppressionSyncer: the tenant's list,
// or the account-level list without a tenant.
func (p *Provider) ListSuppressions(ctx context.Context, since time.Time) ([]providers.Suppression, error) {
	out := []providers.Suppression{}
	in := &sesv2.ListSuppressedDestinationsInput{PageSize: aws.Int32(1000), TenantName: p.tenant()}
	if !since.IsZero() {
		in.StartDate = aws.Time(since)
	}
	for {
		page, err := p.client.ListSuppressedDestinations(ctx, in)
		if err != nil {
			return nil, callError("ListSuppressedDestinations", err)
		}
		for _, d := range page.SuppressedDestinationSummaries {
			reason := "hard_bounce"
			if d.Reason == types.SuppressionListReasonComplaint {
				reason = "complaint"
			}
			s := providers.Suppression{Email: strings.ToLower(aws.ToString(d.EmailAddress)), Reason: reason, Code: string(d.Reason)}
			if d.LastUpdateTime != nil {
				s.CreatedAt = *d.LastUpdateTime
			}
			out = append(out, s)
		}
		if page.NextToken == nil || aws.ToString(page.NextToken) == "" {
			return out, nil
		}
		in.NextToken = page.NextToken
	}
}

// AddSuppressions implements providers.SuppressionSyncer as hard bounces.
// It never writes the account-level list: without a tenant it does nothing
// and says so (providers.ErrNotSupported).
func (p *Provider) AddSuppressions(ctx context.Context, emails []string) error {
	list := make([]providers.Suppression, 0, len(emails))
	for _, e := range emails {
		list = append(list, providers.Suppression{Email: e, Reason: "hard_bounce"})
	}
	return p.AddSuppressionsWithReason(ctx, list)
}

// AddSuppressionsWithReason implements providers.SuppressionReasonAdder: the
// tenant's list, a complaint as COMPLAINT and a hard bounce as BOUNCE.
func (p *Provider) AddSuppressionsWithReason(ctx context.Context, list []providers.Suppression) error {
	if p.s.Tenant == "" {
		return providers.ErrNotSupported
	}
	for _, s := range list {
		reason := types.SuppressionListReasonBounce
		if s.Reason == "complaint" {
			reason = types.SuppressionListReasonComplaint
		}
		_, err := p.client.PutSuppressedDestination(ctx, &sesv2.PutSuppressedDestinationInput{EmailAddress: aws.String(s.Email), Reason: reason, TenantName: p.tenant()})
		if err != nil {
			ce := callError("PutSuppressedDestination", err)
			if ce.Code == "BadRequestException" {
				continue // invalid address: nothing to suppress at SES
			}
			return ce
		}
	}
	return nil
}

// RemoveSuppressions implements providers.SuppressionRemover on the
// tenant's list only.
func (p *Provider) RemoveSuppressions(ctx context.Context, emails []string) error {
	if p.s.Tenant == "" {
		return providers.ErrNotSupported
	}
	for _, e := range emails {
		_, err := p.client.DeleteSuppressedDestination(ctx, &sesv2.DeleteSuppressedDestinationInput{EmailAddress: aws.String(e), TenantName: p.tenant()})
		if err != nil {
			ce := callError("DeleteSuppressedDestination", err)
			if ce.Code == "NotFoundException" || ce.Code == "BadRequestException" {
				continue // not on the list (any more)
			}
			return ce
		}
	}
	return nil
}

// AddDomain implements providers.DomainManager: Easy DKIM and a custom MAIL
// FROM domain (bounce.<domain>), associated with the workspace's tenant.
func (p *Provider) AddDomain(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	_, err := p.client.CreateEmailIdentity(ctx, &sesv2.CreateEmailIdentityInput{EmailIdentity: aws.String(domain), ConfigurationSetName: optional(p.s.ConfigurationSet)})
	if err != nil {
		var ae smithy.APIError
		if !errors.As(err, &ae) || ae.ErrorCode() != "AlreadyExistsException" {
			return nil, callError("CreateEmailIdentity", err)
		}
	}
	if err := p.associate(ctx, p.resourceARN("identity", domain)); err != nil {
		return nil, err
	}
	_, err = p.client.PutEmailIdentityMailFromAttributes(ctx, &sesv2.PutEmailIdentityMailFromAttributesInput{
		EmailIdentity: aws.String(domain), MailFromDomain: aws.String("bounce." + domain), BehaviorOnMxFailure: types.BehaviorOnMxFailureUseDefaultValue,
	})
	if err != nil {
		return nil, callError("PutEmailIdentityMailFromAttributes", err)
	}
	return p.DomainStatus(ctx, domain)
}

// DomainStatus implements providers.DomainManager.
func (p *Provider) DomainStatus(ctx context.Context, domain string) ([]providers.DNSRecord, error) {
	out, err := p.client.GetEmailIdentity(ctx, &sesv2.GetEmailIdentityInput{EmailIdentity: aws.String(domain)})
	if err != nil {
		return nil, callError("GetEmailIdentity", err)
	}
	recs := []providers.DNSRecord{}
	if out.DkimAttributes != nil {
		st := dnsStatus(string(out.DkimAttributes.Status))
		for _, t := range out.DkimAttributes.Tokens {
			recs = append(recs, providers.DNSRecord{Purpose: "dkim", Type: "CNAME", Name: t + "._domainkey." + domain, Value: t + ".dkim.amazonses.com", Status: st})
		}
	}
	if mf := out.MailFromAttributes; mf != nil && aws.ToString(mf.MailFromDomain) != "" {
		st := dnsStatus(string(mf.MailFromDomainStatus))
		name := aws.ToString(mf.MailFromDomain)
		recs = append(recs,
			providers.DNSRecord{Purpose: "return_path", Type: "MX", Name: name, Value: "10 feedback-smtp." + p.s.Region + ".amazonses.com", Status: st},
			providers.DNSRecord{Purpose: "spf", Type: "TXT", Name: name, Value: "v=spf1 include:amazonses.com ~all", Status: st},
		)
	}
	return recs, nil
}

func dnsStatus(s string) string {
	switch strings.ToUpper(s) {
	case "SUCCESS":
		return "verified"
	case "FAILED", "TEMPORARY_FAILURE":
		return "wrong_value"
	}
	return "pending"
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return aws.String(s)
}

// Tracking implements providers.TrackingManager: SES rewrites links and adds
// an open pixel when the configuration set publishes OPEN or CLICK events.
func (p *Provider) Tracking(ctx context.Context) (providers.Tracking, error) {
	set := i18n.Args{"name": p.s.ConfigurationSet}
	t := providers.Tracking{}
	t.Scope, t.ScopeText = say(mScopeSet, set)
	if p.s.ConfigurationSet == "" {
		t.Scope, t.ScopeText = say(mScopeNone, nil)
		t.Detail, t.DetailText = say(mTrackingNone, nil)
		return t, nil
	}
	out, err := p.client.GetConfigurationSetEventDestinations(ctx, &sesv2.GetConfigurationSetEventDestinationsInput{ConfigurationSetName: aws.String(p.s.ConfigurationSet)})
	if err != nil {
		return t, callError("GetConfigurationSetEventDestinations", err)
	}
	for _, d := range out.EventDestinations {
		if !d.Enabled {
			continue
		}
		for _, et := range d.MatchingEventTypes {
			switch et {
			case types.EventTypeOpen:
				t.OpenPixel = true
			case types.EventTypeClick:
				t.LinkRewrite = true
			}
		}
	}
	switch {
	case t.OpenPixel && t.LinkRewrite:
		t.Detail, t.DetailText = say(mTrackingBoth, set)
	case t.OpenPixel:
		t.Detail, t.DetailText = say(mTrackingOpen, set)
	case t.LinkRewrite:
		t.Detail, t.DetailText = say(mTrackingClick, set)
	default:
		t.Detail, t.DetailText = say(mTrackingOff, set)
	}
	return t, nil
}

// DisableTracking implements providers.TrackingManager by removing OPEN and
// CLICK from every event destination of the configuration set.
func (p *Provider) DisableTracking(ctx context.Context) error {
	if p.s.ConfigurationSet == "" {
		return nil
	}
	out, err := p.client.GetConfigurationSetEventDestinations(ctx, &sesv2.GetConfigurationSetEventDestinationsInput{ConfigurationSetName: aws.String(p.s.ConfigurationSet)})
	if err != nil {
		return callError("GetConfigurationSetEventDestinations", err)
	}
	for _, d := range out.EventDestinations {
		kept := make([]types.EventType, 0, len(d.MatchingEventTypes))
		for _, et := range d.MatchingEventTypes {
			if et != types.EventTypeOpen && et != types.EventTypeClick {
				kept = append(kept, et)
			}
		}
		if len(kept) == len(d.MatchingEventTypes) {
			continue
		}
		def := &types.EventDestinationDefinition{
			Enabled: d.Enabled, MatchingEventTypes: kept, SnsDestination: d.SnsDestination,
			CloudWatchDestination: d.CloudWatchDestination, EventBridgeDestination: d.EventBridgeDestination,
			KinesisFirehoseDestination: d.KinesisFirehoseDestination, PinpointDestination: d.PinpointDestination,
		}
		if _, err := p.client.UpdateConfigurationSetEventDestination(ctx, &sesv2.UpdateConfigurationSetEventDestinationInput{
			ConfigurationSetName: aws.String(p.s.ConfigurationSet), EventDestinationName: d.Name, EventDestination: def,
		}); err != nil {
			return callError("UpdateConfigurationSetEventDestination", err)
		}
	}
	return nil
}

// StatusPageURL implements providers.StatusPage.
func (p *Provider) StatusPageURL() (string, string) {
	return "AWS status", "https://health.aws.amazon.com/health/status"
}
