package ses

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

// SES events through an SQS queue ReputeCast polls (review PR-4): SES
// publishes to the SNS topic, the topic delivers to a queue, and
// ReputeCast long-polls the queue with the connection's own AWS access.
// Nothing has to reach ReputeCast from the internet, so it works behind
// NAT and firewalls; the queue only accepts messages from that topic.

// QueueWait is one long poll of the queue: the cadence the setup promises
// ("polls every 20 s").
const QueueWait = 20 * time.Second

var _ providers.EventQueue = (*Provider)(nil)

func (p *Provider) sqsClient() *sqs.Client {
	return sqs.NewFromConfig(p.aws, func(o *sqs.Options) {
		o.Retryer = aws.NopRetryer{}
		if p.s.Endpoint != "" {
			o.BaseEndpoint = aws.String(p.s.Endpoint)
		}
	})
}

// QueueEnabled implements providers.EventQueue.
func (p *Provider) QueueEnabled() bool {
	return p.s.Events == providers.EventsQueue && p.s.QueueURL != ""
}

// ReceiveEvents implements providers.EventQueue: up to max SNS envelopes,
// each one SES event, waiting up to wait for the first.
func (p *Provider) ReceiveEvents(ctx context.Context, maxEvents int, wait time.Duration) ([]providers.QueuedEvent, error) {
	if !p.QueueEnabled() {
		return nil, nil
	}
	out, err := p.sqsClient().ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(p.s.QueueURL),
		MaxNumberOfMessages: int32(min(max(maxEvents, 1), 10)),
		WaitTimeSeconds:     int32(min(max(wait, 0), QueueWait) / time.Second),
		VisibilityTimeout:   120,
	})
	if err != nil {
		return nil, callError("sqs:ReceiveMessage", err)
	}
	evs := make([]providers.QueuedEvent, 0, len(out.Messages))
	for _, m := range out.Messages {
		evs = append(evs, providers.QueuedEvent{Body: []byte(aws.ToString(m.Body)), ContentType: "application/json", Handle: aws.ToString(m.ReceiptHandle)})
	}
	return evs, nil
}

// AckEvents implements providers.EventQueue: stored messages leave the
// queue, ten per call.
func (p *Provider) AckEvents(ctx context.Context, handles []string) error {
	if len(handles) == 0 {
		return nil
	}
	c := p.sqsClient()
	for len(handles) > 0 {
		n := min(len(handles), 10)
		entries := make([]sqstypes.DeleteMessageBatchRequestEntry, 0, n)
		for i, h := range handles[:n] {
			entries = append(entries, sqstypes.DeleteMessageBatchRequestEntry{Id: aws.String(strconv.Itoa(i)), ReceiptHandle: aws.String(h)})
		}
		if _, err := c.DeleteMessageBatch(ctx, &sqs.DeleteMessageBatchInput{QueueUrl: aws.String(p.s.QueueURL), Entries: entries}); err != nil {
			return callError("sqs:DeleteMessageBatch", err)
		}
		handles = handles[n:]
	}
	return nil
}

// SimulatorAddress is SES's mailbox simulator: it accepts mail and makes
// SES report a delivery, without touching the account's bounce or
// complaint rates.
const SimulatorAddress = "success@simulator.amazonses.com"

var _ providers.EventChecker = (*Provider)(nil)

// EventCheckAddress implements providers.EventChecker (PR-3).
func (p *Provider) EventCheckAddress() string { return SimulatorAddress }

// EventCheckFrom implements providers.EventChecker: SES sends only from
// verified identities, the simulator's mail included.
func (p *Provider) EventCheckFrom(ctx context.Context, candidates []string) (string, error) {
	verified := func(identity string) bool {
		out, err := p.client.GetEmailIdentity(ctx, &sesv2.GetEmailIdentityInput{EmailIdentity: aws.String(identity)})
		return err == nil && out.VerifiedForSendingStatus
	}
	for i, c := range candidates {
		at := strings.LastIndexByte(c, '@')
		if i >= 5 || at < 1 {
			continue
		}
		if verified(c) || verified(c[at+1:]) {
			return c, nil
		}
	}
	list, err := p.client.ListEmailIdentities(ctx, &sesv2.ListEmailIdentitiesInput{PageSize: aws.Int32(100)})
	if err != nil {
		return "", callError("ListEmailIdentities", err)
	}
	for _, id := range list.EmailIdentities {
		if id.VerificationStatus != sestypes.VerificationStatusSuccess || !id.SendingEnabled {
			continue
		}
		switch name := aws.ToString(id.IdentityName); id.IdentityType {
		case sestypes.IdentityTypeEmailAddress:
			return name, nil
		case sestypes.IdentityTypeDomain:
			return "reputecast-check@" + name, nil
		}
	}
	return "", nil
}

// queuePolicy lets only the topic send to the queue.
func queuePolicy(queueARN, topicARN string) string {
	b, _ := json.Marshal(map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{{
			"Sid":       "ReputeCastTopicOnly",
			"Effect":    "Allow",
			"Principal": map[string]string{"Service": "sns.amazonaws.com"},
			"Action":    "sqs:SendMessage",
			"Resource":  queueARN,
			"Condition": map[string]any{"ArnEquals": map[string]string{"aws:SourceArn": topicARN}},
		}},
	})
	return string(b)
}

// provisionQueue creates the event topic, the queue and the topic's
// subscription of it ("Created SNS topic reputecast-events and an SQS
// queue subscribed to it: reputecast-events").
func (p *Provider) provisionQueue(ctx context.Context, in providers.ProvisionInput) (providers.ProvisionResult, error) {
	topicARN, err := p.createTopic(ctx, in)
	if err != nil {
		return providers.ProvisionResult{}, err
	}
	name := withSuffix(QueueName, in.Suffix)
	c := p.sqsClient()
	q, err := c.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: aws.String(name), Attributes: map[string]string{
		"ReceiveMessageWaitTimeSeconds": strconv.Itoa(int(QueueWait / time.Second)),
		// Four days: enough to catch up after ReputeCast was stopped.
		"MessageRetentionPeriod": "345600",
	}})
	if err != nil {
		return providers.ProvisionResult{}, callError("sqs:CreateQueue", err)
	}
	queueURL := aws.ToString(q.QueueUrl)
	attrs, err := c.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{QueueUrl: aws.String(queueURL),
		AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn}})
	if err != nil {
		return providers.ProvisionResult{}, callError("sqs:GetQueueAttributes", err)
	}
	queueARN := attrs.Attributes[string(sqstypes.QueueAttributeNameQueueArn)]
	if queueARN == "" {
		return providers.ProvisionResult{}, errors.New("sqs:GetQueueAttributes: the queue has no ARN")
	}
	// SNS confirms the subscription of a queue in the same account by itself.
	sub, err := p.snsClient().Subscribe(ctx, &sns.SubscribeInput{TopicArn: aws.String(topicARN), Protocol: aws.String("sqs"), Endpoint: aws.String(queueARN),
		ReturnSubscriptionArn: true})
	if err != nil {
		return providers.ProvisionResult{}, callError("sns:Subscribe", err)
	}
	p.dropSubscription(ctx, aws.ToString(sub.SubscriptionArn))
	arns := append([]string{}, p.s.TopicARNs...)
	if !contains(arns, topicARN) {
		arns = append(arns, topicARN)
	}
	res := providers.ProvisionResult{Settings: map[string]any{"topic_arns": arns, "topic_arn": topicARN, "subscription_arn": aws.ToString(sub.SubscriptionArn),
		"events": providers.EventsQueue, "queue_url": queueURL, "queue_arn": queueARN}}
	res.Done, res.DoneText = say(mDoneQueue, i18n.Args{"topic": withSuffix(TopicName, in.Suffix), "queue": name})
	return res, nil
}

// limitQueue lets only the event topic send to the queue; polling starts
// with it ("Limited the queue to that topic and started polling every
// 20 s").
func (p *Provider) limitQueue(ctx context.Context, _ providers.ProvisionInput) (providers.ProvisionResult, error) {
	var s struct {
		TopicARN string `json:"topic_arn"`
		QueueURL string `json:"queue_url"`
		QueueARN string `json:"queue_arn"`
	}
	if err := decodeSettings(p.cfg.Settings, &s); err != nil {
		return providers.ProvisionResult{}, err
	}
	if s.QueueURL == "" || s.TopicARN == "" {
		return providers.ProvisionResult{}, errors.New("the queue doesn't exist yet: run its step first")
	}
	if _, err := p.sqsClient().SetQueueAttributes(ctx, &sqs.SetQueueAttributesInput{QueueUrl: aws.String(s.QueueURL),
		Attributes: map[string]string{"Policy": queuePolicy(s.QueueARN, s.TopicARN)}}); err != nil {
		return providers.ProvisionResult{}, callError("sqs:SetQueueAttributes", err)
	}
	var res providers.ProvisionResult
	res.Done, res.DoneText = say(mDoneQueuePolicy, i18n.Args{"seconds": int(QueueWait / time.Second)})
	return res, nil
}
