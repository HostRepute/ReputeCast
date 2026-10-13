package ses

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"

	"github.com/hostrepute/reputecast/internal/providers"
)

// This file is the SES side of "Connect a sending provider" (spec 5.7):
// the CloudFormation access template, the account id, a scan of every
// region's standing, the production-access request, and provisioning of
// the SNS topic, its HTTPS subscription and one configuration set per
// stream, each publishing events to the topic with open and click tracking
// off (ReputeCast tracks those itself).

var (
	_ providers.Provisioner               = (*Provider)(nil)
	_ providers.RegionScanner             = (*Provider)(nil)
	_ providers.ProductionAccessRequester = (*Provider)(nil)
	_ providers.AccountIdentifier         = (*Provider)(nil)
	_ providers.AccessTemplater           = (*Provider)(nil)
)

// Resource names created by provisioning. A suffix is added when another
// connection of the same instance already uses them in the account.
const (
	TopicName = "reputecast-events"
	RoleName  = "ReputeCastSES"
	// EventDestination names the configuration sets' event destination.
	EventDestination = "reputecast"
)

// configSetNames are the per-stream configuration sets.
var configSetNames = map[string]string{"broadcast": "rc-broadcasts", "automation": "rc-automations", "system": "rc-system"}

var streamOrder = []string{"broadcast", "automation", "system"}

// RegionNames are the SES regions with the names the AWS console uses.
var RegionNames = map[string]string{
	"us-east-1": "N. Virginia", "us-east-2": "Ohio", "us-west-1": "N. California", "us-west-2": "Oregon",
	"ca-central-1": "Canada (Central)", "eu-west-1": "Ireland", "eu-west-2": "London", "eu-west-3": "Paris",
	"eu-central-1": "Frankfurt", "eu-north-1": "Stockholm", "eu-south-1": "Milan", "ap-south-1": "Mumbai",
	"ap-northeast-1": "Tokyo", "ap-northeast-2": "Seoul", "ap-northeast-3": "Osaka", "ap-southeast-1": "Singapore",
	"ap-southeast-2": "Sydney", "sa-east-1": "São Paulo", "me-south-1": "Bahrain", "af-south-1": "Cape Town",
	"il-central-1": "Tel Aviv",
}

// configurationSet is the configuration set for a stream's messages.
func (p *Provider) configurationSet(stream providers.Stream) string {
	if cs := p.s.ConfigurationSets[string(stream)]; cs != "" {
		return cs
	}
	return p.s.ConfigurationSet
}

func withSuffix(name, suffix string) string {
	if suffix == "" {
		return name
	}
	return name + "-" + suffix
}

// ProvisionSteps implements providers.Provisioner.
func (p *Provider) ProvisionSteps(_ context.Context, in providers.ProvisionInput) []providers.ProvisionStep {
	topic := withSuffix(TopicName, in.Suffix)
	out := []providers.ProvisionStep{
		{Key: "sns_topic", Label: "Create SNS topic `" + topic + "` and subscribe `" + in.WebhookURL + "`"},
		{Key: "sns_confirm", Label: "Confirm SNS subscription (handshake)"},
	}
	for _, s := range streamOrder {
		out = append(out, providers.ProvisionStep{Key: "config_set:" + s, Label: "Create configuration set `" + withSuffix(configSetNames[s], in.Suffix) + "`"})
	}
	return out
}

// Provision implements providers.Provisioner.
func (p *Provider) Provision(ctx context.Context, step string, in providers.ProvisionInput) (providers.ProvisionResult, error) {
	switch {
	case step == "sns_topic":
		return p.provisionTopic(ctx, in)
	case step == "sns_confirm":
		return p.confirmSubscription(ctx, in)
	case strings.HasPrefix(step, "config_set:"):
		return p.provisionConfigSet(ctx, strings.TrimPrefix(step, "config_set:"), in)
	}
	return providers.ProvisionResult{}, fmt.Errorf("ses: unknown provisioning step %q", step)
}

func (p *Provider) snsClient() *sns.Client {
	return sns.NewFromConfig(p.aws, func(o *sns.Options) { o.Retryer = aws.NopRetryer{} })
}

func (p *Provider) provisionTopic(ctx context.Context, in providers.ProvisionInput) (providers.ProvisionResult, error) {
	name := withSuffix(TopicName, in.Suffix)
	c := p.snsClient()
	t, err := c.CreateTopic(ctx, &sns.CreateTopicInput{Name: aws.String(name)})
	if err != nil {
		return providers.ProvisionResult{}, callError("sns:CreateTopic", err)
	}
	topicARN := aws.ToString(t.TopicArn)
	proto := "https"
	if strings.HasPrefix(in.WebhookURL, "http://") {
		proto = "http"
	}
	sub, err := c.Subscribe(ctx, &sns.SubscribeInput{TopicArn: aws.String(topicARN), Protocol: aws.String(proto), Endpoint: aws.String(in.WebhookURL), ReturnSubscriptionArn: true})
	if err != nil {
		return providers.ProvisionResult{}, callError("sns:Subscribe", err)
	}
	arns := append([]string{}, p.s.TopicARNs...)
	if !contains(arns, topicARN) {
		arns = append(arns, topicARN)
	}
	return providers.ProvisionResult{
		Done:     "Created SNS topic `" + name + "` and subscribed `" + in.WebhookURL + "`",
		Settings: map[string]any{"topic_arns": arns, "topic_arn": topicARN, "subscription_arn": aws.ToString(sub.SubscriptionArn)},
	}, nil
}

// SubscriptionSettings are the provisioning results kept in the settings.
type subscriptionSettings struct {
	TopicARN        string `json:"topic_arn"`
	SubscriptionARN string `json:"subscription_arn"`
}

func (p *Provider) confirmSubscription(ctx context.Context, _ providers.ProvisionInput) (providers.ProvisionResult, error) {
	var s subscriptionSettings
	if err := decodeSettings(p.cfg.Settings, &s); err != nil {
		return providers.ProvisionResult{}, err
	}
	if s.SubscriptionARN == "" || !strings.HasPrefix(s.SubscriptionARN, "arn:") {
		return providers.ProvisionResult{Pending: true}, nil
	}
	out, err := p.snsClient().GetSubscriptionAttributes(ctx, &sns.GetSubscriptionAttributesInput{SubscriptionArn: aws.String(s.SubscriptionARN)})
	if err != nil {
		return providers.ProvisionResult{}, callError("sns:GetSubscriptionAttributes", err)
	}
	if out.Attributes["PendingConfirmation"] == "true" {
		return providers.ProvisionResult{Pending: true}, nil
	}
	return providers.ProvisionResult{Done: "Confirmed SNS subscription (handshake received)"}, nil
}

func (p *Provider) provisionConfigSet(ctx context.Context, stream string, in providers.ProvisionInput) (providers.ProvisionResult, error) {
	base, ok := configSetNames[stream]
	if !ok {
		return providers.ProvisionResult{}, fmt.Errorf("ses: unknown stream %q", stream)
	}
	var s subscriptionSettings
	if err := decodeSettings(p.cfg.Settings, &s); err != nil {
		return providers.ProvisionResult{}, err
	}
	if s.TopicARN == "" {
		return providers.ProvisionResult{}, errors.New("the SNS topic doesn't exist yet: run its step first")
	}
	name := withSuffix(base, in.Suffix)
	_, err := p.client.CreateConfigurationSet(ctx, &sesv2.CreateConfigurationSetInput{
		ConfigurationSetName: aws.String(name),
		ReputationOptions:    &types.ReputationOptions{ReputationMetricsEnabled: true},
		SendingOptions:       &types.SendingOptions{SendingEnabled: true},
	})
	if err != nil && !isAWSCode(err, "AlreadyExistsException") {
		return providers.ProvisionResult{}, callError("ses:CreateConfigurationSet", err)
	}
	// Every delivery event except opens and clicks: ReputeCast tracks those
	// itself, so SES must not add a pixel or rewrite links.
	dest := &types.EventDestinationDefinition{
		Enabled: true,
		MatchingEventTypes: []types.EventType{types.EventTypeSend, types.EventTypeDelivery, types.EventTypeBounce, types.EventTypeComplaint,
			types.EventTypeReject, types.EventTypeDeliveryDelay, types.EventTypeRenderingFailure, types.EventTypeSubscription},
		SnsDestination: &types.SnsDestination{TopicArn: aws.String(s.TopicARN)},
	}
	_, err = p.client.CreateConfigurationSetEventDestination(ctx, &sesv2.CreateConfigurationSetEventDestinationInput{
		ConfigurationSetName: aws.String(name), EventDestinationName: aws.String(EventDestination), EventDestination: dest,
	})
	if isAWSCode(err, "AlreadyExistsException") {
		_, err = p.client.UpdateConfigurationSetEventDestination(ctx, &sesv2.UpdateConfigurationSetEventDestinationInput{
			ConfigurationSetName: aws.String(name), EventDestinationName: aws.String(EventDestination), EventDestination: dest,
		})
		if err != nil {
			return providers.ProvisionResult{}, callError("ses:UpdateConfigurationSetEventDestination", err)
		}
	} else if err != nil {
		return providers.ProvisionResult{}, callError("ses:CreateConfigurationSetEventDestination", err)
	}
	sets := map[string]string{}
	for k, v := range p.s.ConfigurationSets {
		sets[k] = v
	}
	sets[stream] = name
	settings := map[string]any{"configuration_sets": sets}
	if p.s.ConfigurationSet == "" || stream == "broadcast" {
		settings["configuration_set"] = sets["broadcast"]
		if settings["configuration_set"] == "" {
			settings["configuration_set"] = name
		}
	}
	return providers.ProvisionResult{Done: "Created configuration set `" + name + "` with event destination, tracking off", Settings: settings}, nil
}

func isAWSCode(err error, code string) bool {
	var ae smithy.APIError
	return err != nil && errors.As(err, &ae) && ae.ErrorCode() == code
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// AccountID implements providers.AccountIdentifier.
func (p *Provider) AccountID(ctx context.Context) (string, error) {
	out, err := sts.NewFromConfig(p.aws).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", callError("sts:GetCallerIdentity", err)
	}
	return aws.ToString(out.Account), nil
}

// ScanRegions implements providers.RegionScanner: GetAccount in every SES
// region, ten at a time.
func (p *Provider) ScanRegions(ctx context.Context) ([]providers.RegionStatus, error) {
	regions := make([]string, 0, len(RegionNames))
	for r := range RegionNames {
		regions = append(regions, r)
	}
	sort.Strings(regions)
	out := make([]providers.RegionStatus, len(regions))
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup
	for i, r := range regions {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, region string) {
			defer wg.Done()
			defer func() { <-sem }()
			st := providers.RegionStatus{Region: region, Name: RegionNames[region]}
			cfg := p.aws.Copy()
			cfg.Region = region
			cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			acct, err := sesv2.NewFromConfig(cfg).GetAccount(cctx, &sesv2.GetAccountInput{})
			cancel()
			if err != nil {
				st.Err = callError("GetAccount", err).Error()
			} else {
				st.Production = acct.ProductionAccessEnabled
				if acct.SendQuota != nil {
					st.Max24h, st.MaxRate = int64(acct.SendQuota.Max24HourSend), acct.SendQuota.MaxSendRate
				}
			}
			out[i] = st
		}(i, r)
	}
	wg.Wait()
	return out, nil
}

// RequestProductionAccess implements providers.ProductionAccessRequester.
func (p *Provider) RequestProductionAccess(ctx context.Context, r providers.ProductionAccessRequest) error {
	cfg := p.aws.Copy()
	if r.Region != "" {
		cfg.Region = r.Region
	}
	mailType := types.MailTypeMarketing
	if r.MailType == "transactional" {
		mailType = types.MailTypeTransactional
	}
	lang := types.ContactLanguageEn
	if r.Language == "ja" {
		lang = types.ContactLanguageJa
	}
	useCase := r.UseCase
	if r.DailyVolume > 0 {
		useCase += fmt.Sprintf("\n\nExpected daily volume: %d emails.", r.DailyVolume)
	}
	_, err := sesv2.NewFromConfig(cfg).PutAccountDetails(ctx, &sesv2.PutAccountDetailsInput{
		MailType: mailType, WebsiteURL: aws.String(r.WebsiteURL), ContactLanguage: lang,
		// Optional for AWS now, but the request form asks for it and the
		// review reads it when present.
		UseCaseDescription:              aws.String(useCase), //nolint:staticcheck // see above
		AdditionalContactEmailAddresses: r.ContactEmails, ProductionAccessEnabled: aws.Bool(true),
	})
	if err != nil {
		return callError("ses:PutAccountDetails", err)
	}
	return nil
}

// Policy is the least-privilege IAM policy ReputeCast needs: send, read the
// account, manage configuration sets, identities and the suppression list,
// request production access, and create and subscribe topics named
// reputecast-*.
const Policy = `{ "Version": "2012-10-17", "Statement": [
  { "Effect": "Allow", "Action": ["ses:SendEmail","ses:SendRawEmail","ses:GetAccount","ses:GetSendQuota",
    "ses:*ConfigurationSet*","ses:*EmailIdentity*","ses:*SuppressedDestination*","ses:PutAccountDetails"], "Resource": "*" },
  { "Effect": "Allow", "Action": ["sns:CreateTopic","sns:Subscribe","sns:SetTopicAttributes","sns:GetSubscriptionAttributes"],
    "Resource": "arn:aws:sns:*:*:reputecast-*" } ] }`

// Template is the CloudFormation template that creates the role. It takes
// the external id ReputeCast generated for the connection and the identity
// this server runs as.
const Template = `AWSTemplateFormatVersion: "2010-09-09"
Description: >-
  ReputeCast access to Amazon SES: send mail, read the account, manage
  configuration sets, identities and the suppression list, and create the
  SNS topics (reputecast-*) that carry delivery events back to ReputeCast.
Parameters:
  ExternalId:
    Type: String
    MinLength: 16
    NoEcho: true
    Description: Generated by ReputeCast for this connection.
  ReputeCastPrincipal:
    Type: String
    Description: ARN of the AWS identity your ReputeCast server runs as, or its account id.
Resources:
  ReputeCastSESRole:
    Type: AWS::IAM::Role
    Properties:
      RoleName: ReputeCastSES
      MaxSessionDuration: 3600
      AssumeRolePolicyDocument:
        Version: "2012-10-17"
        Statement:
          - Effect: Allow
            Principal:
              AWS: !Ref ReputeCastPrincipal
            Action: sts:AssumeRole
            Condition:
              StringEquals:
                sts:ExternalId: !Ref ExternalId
      Policies:
        - PolicyName: ReputeCastSES
          PolicyDocument:
            Version: "2012-10-17"
            Statement:
              - Effect: Allow
                Action:
                  - ses:SendEmail
                  - ses:SendRawEmail
                  - ses:GetAccount
                  - ses:GetSendQuota
                  - ses:*ConfigurationSet*
                  - ses:*EmailIdentity*
                  - ses:*SuppressedDestination*
                  - ses:PutAccountDetails
                Resource: "*"
              - Effect: Allow
                Action:
                  - sns:CreateTopic
                  - sns:Subscribe
                  - sns:SetTopicAttributes
                  - sns:GetSubscriptionAttributes
                Resource: arn:aws:sns:*:*:reputecast-*
Outputs:
  RoleArn:
    Description: Paste this into ReputeCast.
    Value: !GetAtt ReputeCastSESRole.Arn
`

var (
	principalMu    sync.Mutex
	principalValue string
	principalUntil time.Time
)

// hostPrincipal detects the AWS identity this server runs as (instance
// role, environment credentials), cached for ten minutes. Assumed-role
// sessions are turned into their role ARN.
func hostPrincipal(ctx context.Context, region string) string {
	principalMu.Lock()
	defer principalMu.Unlock()
	if time.Now().Before(principalUntil) {
		return principalValue
	}
	principalValue, principalUntil = "", time.Now().Add(10*time.Minute)
	cctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	cfg, err := config.LoadDefaultConfig(cctx, config.WithRegion(region))
	if err != nil {
		return ""
	}
	out, err := sts.NewFromConfig(cfg).GetCallerIdentity(cctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return ""
	}
	principalValue = RolePrincipal(aws.ToString(out.Arn))
	return principalValue
}

// RolePrincipal turns an assumed-role session ARN
// (arn:aws:sts::123:assumed-role/Name/session) into the role ARN a trust
// policy can name; other ARNs are returned unchanged.
func RolePrincipal(s string) string {
	a, err := arn.Parse(s)
	if err != nil || a.Service != "sts" || !strings.HasPrefix(a.Resource, "assumed-role/") {
		return s
	}
	parts := strings.Split(strings.TrimPrefix(a.Resource, "assumed-role/"), "/")
	return "arn:" + a.Partition + ":iam::" + a.AccountID + ":role/" + parts[0]
}

// AccessTemplate implements providers.AccessTemplater.
func (p *Provider) AccessTemplate(ctx context.Context, in providers.AccessTemplateInput) (providers.AccessTemplate, error) {
	region := in.Region
	if region == "" {
		region = p.s.Region
	}
	principal := hostPrincipal(ctx, region)
	q := "templateURL=" + url.QueryEscape(in.TemplateURL) + "&stackName=" + RoleName + "&param_ExternalId=" + url.QueryEscape(in.ExternalID)
	if principal != "" {
		q += "&param_ReputeCastPrincipal=" + url.QueryEscape(principal)
	}
	return providers.AccessTemplate{
		Format:    "cloudformation",
		Body:      Template,
		LaunchURL: "https://console.aws.amazon.com/cloudformation/home?region=" + url.QueryEscape(region) + "#/stacks/quickcreate?" + q,
		Policy:    Policy,
		Principal: principal,
		RoleName:  RoleName,
	}, nil
}

func decodeSettings(raw []byte, v any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, v)
}
