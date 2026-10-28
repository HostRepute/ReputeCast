package ses_test

import (
	"context"
	"crypto/md5" //nolint:gosec // SQS's own body checksum, not a security measure
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/ses"
)

// The SQS queue path (review PR-4, R3.A1) against a fake AWS: the adapter's
// calls go through its own HTTP client, as in production, to an in-memory
// SQS that speaks SQS's JSON protocol (with its body checksums) and an SNS
// that speaks the query protocol.

const queueURL = "https://sqs.us-east-1.amazonaws.com/123456789012/reputecast-events"

// fakeAWS is SQS and SNS in memory.
type fakeAWS struct {
	mu       sync.Mutex
	waiting  []string          // bodies in the queue
	inflight map[string]string // receipt handle → body
	handles  int
	receives []map[string]any // ReceiveMessage inputs
	batches  []int            // DeleteMessageBatch sizes
	ops      []string         // every call, "sqs:ReceiveMessage", "sns:Subscribe"…
	queue    map[string]any   // CreateQueue input
	policy   string           // the queue's Policy attribute
	sub      url.Values       // SNS Subscribe input
	fail     map[string]string
}

func newFakeAWS() *fakeAWS {
	return &fakeAWS{inflight: map[string]string{}, fail: map[string]string{}}
}

func (f *fakeAWS) failNext(op, code string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fail[op] = code
}

func (f *fakeAWS) push(bodies ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.waiting = append(f.waiting, bodies...)
}

func respond(r *http.Request, status int, contentType, body string, header map[string]string) *http.Response {
	h := http.Header{"Content-Type": {contentType}}
	for k, v := range header {
		h.Set(k, v)
	}
	return &http.Response{StatusCode: status, Status: strconv.Itoa(status) + " " + http.StatusText(status), Header: h,
		Body: io.NopCloser(strings.NewReader(body)), ContentLength: int64(len(body)), Request: r}
}

// RoundTrip answers SQS (X-Amz-Target: AmazonSQS.*) and SNS (Action=…).
func (f *fakeAWS) RoundTrip(r *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	defer f.mu.Unlock()
	if target := r.Header.Get("X-Amz-Target"); strings.HasPrefix(target, "AmazonSQS.") {
		return f.sqs(r, strings.TrimPrefix(target, "AmazonSQS."), body), nil
	}
	form, _ := url.ParseQuery(string(body))
	return f.sns(r, form.Get("Action"), form), nil
}

func (f *fakeAWS) sqs(r *http.Request, op string, body []byte) *http.Response {
	f.ops = append(f.ops, "sqs:"+op)
	const jsonType = "application/x-amz-json-1.0"
	if code := f.fail[op]; code != "" {
		delete(f.fail, op)
		status := http.StatusBadRequest
		if code == "AccessDenied" {
			status = http.StatusForbidden
		}
		return respond(r, status, jsonType, fmt.Sprintf(`{"__type":"com.amazonaws.sqs#%s","message":"%s refused"}`, code, op),
			map[string]string{"x-amzn-query-error": code + ";Sender"})
	}
	var in map[string]any
	_ = json.Unmarshal(body, &in)
	ok := func(v any) *http.Response {
		b, _ := json.Marshal(v)
		return respond(r, http.StatusOK, jsonType, string(b), nil)
	}
	switch op {
	case "ReceiveMessage":
		f.receives = append(f.receives, in)
		limit, _ := in["MaxNumberOfMessages"].(float64)
		msgs := []map[string]any{}
		for len(f.waiting) > 0 && len(msgs) < int(limit) {
			b := f.waiting[0]
			f.waiting = f.waiting[1:]
			f.handles++
			h := fmt.Sprintf("handle-%02d", f.handles)
			f.inflight[h] = b
			sum := md5.Sum([]byte(b)) //nolint:gosec // SQS's body checksum
			msgs = append(msgs, map[string]any{"MessageId": fmt.Sprintf("m-%02d", f.handles), "ReceiptHandle": h, "MD5OfBody": hex.EncodeToString(sum[:]), "Body": b})
		}
		return ok(map[string]any{"Messages": msgs})
	case "DeleteMessageBatch":
		entries, _ := in["Entries"].([]any)
		f.batches = append(f.batches, len(entries))
		done := []map[string]any{}
		for _, e := range entries {
			m, _ := e.(map[string]any)
			delete(f.inflight, fmt.Sprint(m["ReceiptHandle"]))
			done = append(done, map[string]any{"Id": m["Id"]})
		}
		return ok(map[string]any{"Successful": done, "Failed": []any{}})
	case "CreateQueue":
		f.queue = in
		return ok(map[string]any{"QueueUrl": queueURL})
	case "GetQueueAttributes":
		return ok(map[string]any{"Attributes": map[string]string{"QueueArn": "arn:aws:sqs:us-east-1:123456789012:reputecast-events"}})
	case "SetQueueAttributes":
		attrs, _ := in["Attributes"].(map[string]any)
		f.policy = fmt.Sprint(attrs["Policy"])
		return ok(map[string]any{})
	}
	return respond(r, http.StatusBadRequest, jsonType, `{"__type":"com.amazonaws.sqs#UnsupportedOperation","message":"`+op+`"}`, nil)
}

func (f *fakeAWS) sns(r *http.Request, action string, form url.Values) *http.Response {
	f.ops = append(f.ops, "sns:"+action)
	xml := func(result string) *http.Response {
		return respond(r, http.StatusOK, "text/xml", `<`+action+`Response xmlns="http://sns.amazonaws.com/doc/2010-03-31/"><`+action+`Result>`+result+
			`</`+action+`Result><ResponseMetadata><RequestId>req-1</RequestId></ResponseMetadata></`+action+`Response>`, nil)
	}
	switch action {
	case "CreateTopic":
		return xml(`<TopicArn>arn:aws:sns:us-east-1:123456789012:` + form.Get("Name") + `</TopicArn>`)
	case "Subscribe":
		f.sub = form
		return xml(`<SubscriptionArn>arn:aws:sns:us-east-1:123456789012:reputecast-events:3f1c</SubscriptionArn>`)
	case "Unsubscribe":
		return xml(``)
	}
	return respond(r, http.StatusBadRequest, "text/xml", `<ErrorResponse><Error><Code>InvalidAction</Code><Message>`+action+`</Message></Error></ErrorResponse>`, nil)
}

// queueProvider is an SES connection whose events come through the queue.
func queueProvider(t *testing.T, aws *fakeAWS, settings map[string]any) providers.Provider {
	t.Helper()
	s := map[string]any{"region": "us-east-1", "access": "keys", "events": providers.EventsQueue, "queue_url": queueURL}
	for k, v := range settings {
		s[k] = v
	}
	raw, _ := json.Marshal(s)
	p, err := ses.New(providers.Config{Settings: raw,
		Credentials: json.RawMessage(`{"access_key_id":"AKIDEXAMPLE","secret_access_key":"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}`),
		HTTPClient:  &http.Client{Transport: aws, Timeout: 10 * time.Second}})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// ReceiveEvents long-polls the queue within SQS's bounds (10 messages, 20 s,
// a 2-minute visibility timeout) and hands back each SNS envelope with its
// receipt handle; AckEvents deletes stored ones ten per call.
func TestQueueReceivesAndAcks(t *testing.T) {
	ctx := context.Background()
	aws := newFakeAWS()
	p := queueProvider(t, aws, nil)
	q, ok := p.(providers.EventQueue)
	if !ok || !q.QueueEnabled() {
		t.Fatal("the queue mode isn't on")
	}
	bodies := make([]string, 12)
	for i := range bodies {
		bodies[i] = fmt.Sprintf(`{"Type":"Notification","MessageId":"sns-%02d","Message":"{\"eventType\":\"Delivery\"}"}`, i)
	}
	aws.push(bodies...)

	got, err := q.ReceiveEvents(ctx, 50, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 10 {
		t.Fatalf("received %d, want SQS's 10 at most", len(got))
	}
	in := aws.receives[0]
	if in["QueueUrl"] != queueURL || in["MaxNumberOfMessages"] != float64(10) || in["WaitTimeSeconds"] != float64(20) || in["VisibilityTimeout"] != float64(120) {
		t.Fatalf("ReceiveMessage input %v", in)
	}
	for i, ev := range got {
		if string(ev.Body) != bodies[i] || ev.ContentType != "application/json" || ev.Handle != fmt.Sprintf("handle-%02d", i+1) {
			t.Fatalf("event %d: %+v", i, ev)
		}
	}
	// A short wait for the rest, and at least one message asked for.
	rest, err := q.ReceiveEvents(ctx, 0, -time.Second)
	if err != nil || len(rest) != 1 {
		t.Fatalf("the rest: %d %v", len(rest), err)
	}
	if in := aws.receives[1]; in["MaxNumberOfMessages"] != float64(1) || in["WaitTimeSeconds"] != nil && in["WaitTimeSeconds"] != float64(0) {
		t.Fatalf("second ReceiveMessage input %v", in)
	}

	handles := make([]string, 0, 11)
	for _, ev := range append(got, rest...) {
		handles = append(handles, ev.Handle)
	}
	if err := q.AckEvents(ctx, handles); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(aws.batches) != "[10 1]" || len(aws.inflight) != 0 {
		t.Fatalf("deleted in batches %v, still in flight %v", aws.batches, aws.inflight)
	}
	calls := len(aws.ops)
	if err := q.AckEvents(ctx, nil); err != nil || len(aws.ops) != calls {
		t.Fatalf("nothing to ack made a call: %v", err)
	}
}

// Without the queue mode the adapter never calls SQS.
func TestQueueOffMakesNoCall(t *testing.T) {
	aws := newFakeAWS()
	p := queueProvider(t, aws, map[string]any{"events": "", "queue_url": ""})
	q := p.(providers.EventQueue)
	if q.QueueEnabled() {
		t.Fatal("queue mode on without a queue")
	}
	evs, err := q.ReceiveEvents(context.Background(), 10, time.Second)
	if err != nil || evs != nil || len(aws.ops) != 0 {
		t.Fatalf("events %v, err %v, calls %v", evs, err, aws.ops)
	}
}

// SQS's refusals become the adapter's call errors: lost access needs a
// person (Auth), a deleted queue is permanent.
func TestQueueErrors(t *testing.T) {
	ctx := context.Background()
	aws := newFakeAWS()
	q := queueProvider(t, aws, nil).(providers.EventQueue)
	aws.failNext("ReceiveMessage", "AccessDenied")
	_, err := q.ReceiveEvents(ctx, 10, time.Second)
	var ce *providers.CallError
	if !errors.As(err, &ce) || ce.Op != "sqs:ReceiveMessage" || !ce.Auth || ce.Temporary || ce.Status != http.StatusForbidden {
		t.Fatalf("access denied: %#v", err)
	}
	aws.failNext("ReceiveMessage", "QueueDoesNotExist")
	_, err = q.ReceiveEvents(ctx, 10, time.Second)
	if !errors.As(err, &ce) || ce.Code != "QueueDoesNotExist" || ce.Auth || ce.Temporary {
		t.Fatalf("no queue: %#v", err)
	}
	aws.failNext("DeleteMessageBatch", "AccessDenied")
	if err := q.AckEvents(ctx, []string{"handle-01"}); !errors.As(err, &ce) || ce.Op != "sqs:DeleteMessageBatch" || !ce.Auth {
		t.Fatalf("ack refused: %#v", err)
	}
}

// Provisioning the queue path: the topic, a queue that long-polls for 20 s
// and keeps four days, subscribed to the topic; then a policy that lets
// only that topic send to it.
func TestQueueProvisioning(t *testing.T) {
	ctx := context.Background()
	aws := newFakeAWS()
	p := queueProvider(t, aws, map[string]any{"events": "", "queue_url": ""})
	pr := p.(providers.Provisioner)
	in := providers.ProvisionInput{Events: providers.EventsQueue}
	steps := []string{}
	for _, st := range pr.ProvisionSteps(ctx, in) {
		steps = append(steps, st.Key)
	}
	if !strings.HasPrefix(strings.Join(steps, ","), "sns_queue,sqs_policy,config_set:") {
		t.Fatalf("queue steps %v", steps)
	}
	res, err := pr.Provision(ctx, "sns_queue", in)
	if err != nil {
		t.Fatal(err)
	}
	attrs, _ := aws.queue["Attributes"].(map[string]any)
	if aws.queue["QueueName"] != "reputecast-events" || attrs["ReceiveMessageWaitTimeSeconds"] != "20" || attrs["MessageRetentionPeriod"] != "345600" {
		t.Fatalf("CreateQueue %v", aws.queue)
	}
	topic := "arn:aws:sns:us-east-1:123456789012:reputecast-events"
	if aws.sub.Get("TopicArn") != topic || aws.sub.Get("Protocol") != "sqs" || aws.sub.Get("Endpoint") != "arn:aws:sqs:us-east-1:123456789012:reputecast-events" {
		t.Fatalf("Subscribe %v", aws.sub)
	}
	if res.Settings["events"] != providers.EventsQueue || res.Settings["queue_url"] != queueURL || res.Settings["topic_arn"] != topic ||
		res.Done != "Created SNS topic `reputecast-events` and an SQS queue subscribed to it: `reputecast-events`" {
		t.Fatalf("result %+v", res)
	}

	p = queueProvider(t, aws, map[string]any{"topic_arn": topic, "queue_arn": res.Settings["queue_arn"]})
	res, err = p.(providers.Provisioner).Provision(ctx, "sqs_policy", in)
	if err != nil {
		t.Fatal(err)
	}
	var policy struct {
		Statement []struct {
			Effect    string
			Principal map[string]string
			Action    string
			Resource  string
			Condition map[string]map[string]string
		}
	}
	if err := json.Unmarshal([]byte(aws.policy), &policy); err != nil || len(policy.Statement) != 1 {
		t.Fatalf("policy %q: %v", aws.policy, err)
	}
	st := policy.Statement[0]
	if st.Effect != "Allow" || st.Principal["Service"] != "sns.amazonaws.com" || st.Action != "sqs:SendMessage" ||
		st.Resource != "arn:aws:sqs:us-east-1:123456789012:reputecast-events" || st.Condition["ArnEquals"]["aws:SourceArn"] != topic {
		t.Fatalf("policy %+v", st)
	}
	if res.Done != "Limited the queue to that topic and started polling every 20 s" {
		t.Fatalf("done %q", res.Done)
	}
	// Without the queue's step done first, the policy step says so.
	p = queueProvider(t, aws, map[string]any{"queue_url": ""})
	if _, err := p.(providers.Provisioner).Provision(ctx, "sqs_policy", in); err == nil {
		t.Fatal("a policy for a queue that doesn't exist yet")
	}
}
