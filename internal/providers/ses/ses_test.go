package ses_test

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // SNS signature version 1 fixtures
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/ses"
)

const (
	certURL  = "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-contract.pem"
	topicARN = "arn:aws:sns:us-east-1:123456789012:reputecast-acme-newsletter"
)

var (
	once    sync.Once
	snsKey  *rsa.PrivateKey
	snsCert []byte
)

func signer(t *testing.T) (*rsa.PrivateKey, []byte) {
	once.Do(func() {
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		tpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "sns.amazonaws.com"},
			NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour)}
		der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k)
		if err != nil {
			t.Fatal(err)
		}
		snsKey, snsCert = k, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	})
	return snsKey, snsCert
}

// transport serves the SNS signing certificate and the subscription link;
// everything else (the SDK's calls to the fake API) goes to the network.
type transport struct {
	t         *testing.T
	confirmed *bool
}

func (tr transport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host == "sns.us-east-1.amazonaws.com" {
		_, cert := signer(tr.t)
		body := cert
		if r.URL.Query().Get("Action") == "ConfirmSubscription" {
			*tr.confirmed = true
			body = []byte("<ConfirmSubscriptionResponse/>")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{}, Request: r}, nil
	}
	return http.DefaultTransport.RoundTrip(r)
}

func fakeAPI(env *contracttest.Env) http.Handler {
	mux := http.NewServeMux()
	apiErr := func(w http.ResponseWriter, status int, code, msg string) {
		w.Header().Set("X-Amzn-ErrorType", code)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"__type": code, "message": msg})
	}
	guard := func(w http.ResponseWriter) bool {
		switch env.Mode() {
		case contracttest.ModeOutage:
			apiErr(w, http.StatusServiceUnavailable, "ServiceUnavailable", "Service Unavailable")
			return true
		case contracttest.ModeAuth:
			apiErr(w, http.StatusForbidden, "UnrecognizedClientException", "The security token included in the request is invalid.")
			return true
		}
		return false
	}
	mux.HandleFunc("POST /v2/email/outbound-emails", func(w http.ResponseWriter, r *http.Request) {
		if guard(w) {
			return
		}
		var in struct {
			Destination struct{ ToAddresses []string }
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		if len(in.Destination.ToAddresses) == 1 && in.Destination.ToAddresses[0] == contracttest.RejectAddress {
			apiErr(w, http.StatusBadRequest, "MessageRejected", "Email address is not verified. The following identities failed the check in region US-EAST-1: "+contracttest.RejectAddress)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"MessageId": "0100018f2b4d5c6e-" + strings.Repeat("a", 8) + "-000000"})
	})
	mux.HandleFunc("GET /v2/email/account", func(w http.ResponseWriter, _ *http.Request) {
		if guard(w) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"EnforcementStatus":"HEALTHY","ProductionAccessEnabled":true,"SendingEnabled":true,"SendQuota":{"Max24HourSend":50000,"MaxSendRate":14,"SentLast24Hours":2140}}`)
	})
	// GetDedicatedIps in two pages, a standard pool and a managed one.
	mux.HandleFunc("GET /v2/email/dedicated-ips", func(w http.ResponseWriter, r *http.Request) {
		if guard(w) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("NextToken") == "" {
			_, _ = io.WriteString(w, `{"DedicatedIps":[{"Ip":"198.51.100.21","WarmupStatus":"DONE","WarmupPercentage":100,"PoolName":"ses-default-dedicated-pool"},`+
				`{"Ip":"198.51.100.7","WarmupStatus":"IN_PROGRESS","WarmupPercentage":40,"PoolName":"ses-default-dedicated-pool"}],"NextToken":"page-2"}`)
			return
		}
		_, _ = io.WriteString(w, `{"DedicatedIps":[{"Ip":"203.0.113.40","WarmupStatus":"NOT_APPLICABLE","WarmupPercentage":-1,"PoolName":"rc-managed"}]}`)
	})
	return mux
}

func newProvider(t *testing.T, env *contracttest.Env) providers.Provider {
	t.Helper()
	confirmed := false
	return newProviderWith(t, env, &confirmed)
}

func newProviderWith(t *testing.T, env *contracttest.Env, confirmed *bool) providers.Provider {
	t.Helper()
	settings, _ := json.Marshal(ses.Settings{Region: "us-east-1", ConfigurationSet: "rc-broadcasts", Access: "keys", Endpoint: env.URL, TopicARNs: []string{topicARN}})
	p, err := ses.New(providers.Config{
		Settings:    settings,
		Credentials: json.RawMessage(`{"access_key_id":"AKIDEXAMPLE","secret_access_key":"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}`),
		HTTPClient:  &http.Client{Transport: transport{t: t, confirmed: confirmed}, Timeout: 10 * time.Second},
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func envelope(t *testing.T, typ, message, version string) []byte {
	key, _ := signer(t)
	e := ses.Envelope{
		Type: typ, MessageID: "d9b9c7a3-3a9c-5f3c-9a1b-2c3d4e5f6a7b", TopicArn: topicARN, Message: message,
		Timestamp: "2026-10-12T13:41:02.200Z", SignatureVersion: version, SigningCertURL: certURL,
		UnsubscribeURL: "https://sns.us-east-1.amazonaws.com/?Action=Unsubscribe&SubscriptionArn=" + topicARN + ":1",
	}
	if typ == "SubscriptionConfirmation" {
		e.Token = "2336412f37fb687f5d51e6e2425f004aed"
		e.SubscribeURL = "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription&TopicArn=" + topicARN + "&Token=" + e.Token
		e.UnsubscribeURL = ""
	}
	var sig []byte
	var err error
	if version == "1" {
		sum := sha1.Sum([]byte(e.StringToSign())) //nolint:gosec // fixture
		sig, err = rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA1, sum[:])
	} else {
		sum := sha256.Sum256([]byte(e.StringToSign()))
		sig, err = rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	}
	if err != nil {
		t.Fatal(err)
	}
	e.Signature = base64.StdEncoding.EncodeToString(sig)
	b, _ := json.Marshal(e)
	return b
}

func notification(file, version string) func(t *testing.T, env *contracttest.Env) (*http.Request, []byte) {
	return func(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) {
		msg, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		var compact bytes.Buffer
		_ = json.Compact(&compact, msg)
		body := envelope(t, "Notification", compact.String(), version)
		r, _ := http.NewRequest(http.MethodPost, "https://reputecast.acme.io/hooks/ses/token", bytes.NewReader(body))
		r.Header.Set("Content-Type", "text/plain; charset=UTF-8")
		r.Header.Set("X-Amz-Sns-Message-Type", "Notification")
		return r, body
	}
}

func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{
		Adapter:    ses.Adapter,
		API:        fakeAPI,
		New:        newProvider,
		SendingIPs: &providers.SendingIPs{Dedicated: true, IPs: []string{"198.51.100.7", "198.51.100.21", "203.0.113.40"}},
		Delivered: func(t *testing.T, env *contracttest.Env, msgs []providers.Message, _ []providers.SendResult) {
			reqs := env.Requests()
			if len(reqs) != len(msgs) {
				t.Fatalf("want %d SendEmail calls, got %d", len(msgs), len(reqs))
			}
			if !strings.HasPrefix(reqs[0].Header.Get("Authorization"), "AWS4-HMAC-SHA256") {
				t.Fatalf("requests must be SigV4 signed: %v", reqs[0].Header)
			}
			var in struct {
				ConfigurationSetName string
				EmailTags            []struct{ Name, Value string }
				Content              struct{ Raw struct{ Data string } }
			}
			if err := json.Unmarshal(reqs[0].Body, &in); err != nil {
				t.Fatal(err)
			}
			if in.ConfigurationSetName != "rc-broadcasts" {
				t.Fatalf("configuration set: %q", in.ConfigurationSetName)
			}
			raw, _ := base64.StdEncoding.DecodeString(in.Content.Raw.Data)
			if !strings.Contains(string(raw), "List-Unsubscribe-Post: List-Unsubscribe=One-Click") {
				t.Fatalf("raw MIME lost List-Unsubscribe-Post:\n%s", raw)
			}
			tagged := false
			for _, tg := range in.EmailTags {
				for _, m := range msgs {
					if tg.Name == ses.TagMessage && tg.Value == m.ID.String() {
						tagged = true
					}
				}
			}
			if !tagged {
				t.Fatalf("message tag missing: %+v", in.EmailTags)
			}
		},
		Webhooks: []contracttest.Webhook{
			{Name: "Bounce", Request: notification("bounce.json", "2"), Want: []providers.Event{{
				Type: providers.EventHardBounce, Recipient: "r.w@outlook.com", ProviderMessageID: "0100018f2b4d5c6e-6f7a8b9c-0d1e-4f2a-8b3c-4d5e6f7a8b9c-000000",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e60"), OriginalType: "Bounce · Permanent · General", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown",
			}}},
			{Name: "Delivery", Request: notification("delivery.json", "1"), Want: []providers.Event{{
				Type: providers.EventDelivered, Recipient: "a.k@gmail.com", ProviderMessageID: "0100018f2b4d5c6e-11111111-0d1e-4f2a-8b3c-4d5e6f7a8b9c-000000",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e61"), OriginalType: "Delivery", OriginalCode: "250 2.0.0",
			}}},
			{Name: "Complaint", Request: notification("complaint.json", "2"), Want: []providers.Event{{
				Type: providers.EventComplaint, Recipient: "m.g@yahoo.com", ProviderMessageID: "0100018f2b4d5c6e-22222222-0d1e-4f2a-8b3c-4d5e6f7a8b9c-000000",
				OriginalType: "Complaint · abuse",
			}}},
			{Name: "DeliveryDelay", Request: notification("delivery-delay.json", "2"), Want: []providers.Event{{
				Type: providers.EventDeferred, Recipient: "b.h@yahoo.com", ProviderMessageID: "0100018f2b4d5c6e-33333333-0d1e-4f2a-8b3c-4d5e6f7a8b9c-000000",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e63"), OriginalType: "DeliveryDelay · MailboxFull", OriginalCode: "452 4.2.2", BounceClass: "mailbox_full",
			}}},
			{Name: "Reject", Request: notification("reject.json", "2"), Want: []providers.Event{{
				Type: providers.EventRejected, Recipient: "j.c@acme-partner.co", ProviderMessageID: "0100018f2b4d5c6e-44444444-0d1e-4f2a-8b3c-4d5e6f7a8b9c-000000",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e64"), OriginalType: "Reject · Bad content", BounceClass: "spam_block",
			}}},
		},
	})
}

func TestSubscriptionHandshake(t *testing.T) {
	env := contracttest.NewEnv(t, fakeAPI)
	confirmed := false
	p := newProviderWith(t, env, &confirmed).(providers.WebhookHandshaker)
	body := envelope(t, "SubscriptionConfirmation", "You have chosen to subscribe to the topic.", "2")
	r, _ := http.NewRequest(http.MethodPost, "https://reputecast.acme.io/hooks/ses/token", bytes.NewReader(body))
	handled, _, err := p.Handshake(context.Background(), r, body)
	if err != nil || !handled || !confirmed {
		t.Fatalf("handshake: handled %v confirmed %v err %v", handled, confirmed, err)
	}
	n := envelope(t, "Notification", `{"eventType":"Send"}`, "2")
	if handled, _, _ := p.Handshake(context.Background(), r, n); handled {
		t.Fatal("a notification is not a handshake")
	}
}

func TestRejectsForeignTopicsAndCertHosts(t *testing.T) {
	env := contracttest.NewEnv(t, fakeAPI)
	p := newProvider(t, env)
	for _, u := range []string{"https://evil.example/cert.pem", "http://sns.us-east-1.amazonaws.com/x.pem", "https://sns.us-east-1.amazonaws.com.evil.io/x.pem"} {
		if ses.ValidCertURL(u) {
			t.Fatalf("%s must not be accepted as a signing certificate URL", u)
		}
	}
	key, _ := signer(t)
	e := ses.Envelope{Type: "Notification", MessageID: "x", TopicArn: "arn:aws:sns:us-east-1:999999999999:other", Message: "{}", Timestamp: "2026-10-12T13:41:02.200Z", SignatureVersion: "2", SigningCertURL: certURL}
	sum := sha256.Sum256([]byte(e.StringToSign()))
	sig, _ := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	e.Signature = base64.StdEncoding.EncodeToString(sig)
	body, _ := json.Marshal(e)
	r, _ := http.NewRequest(http.MethodPost, "https://reputecast.acme.io/hooks/ses/token", bytes.NewReader(body))
	if _, err := p.ParseWebhook(context.Background(), r, body); err == nil {
		t.Fatal("a notification from another topic must be rejected")
	}
}

func TestQuota(t *testing.T) {
	env := contracttest.NewEnv(t, fakeAPI)
	q, err := newProvider(t, env).(providers.QuotaReporter).Quota(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if q.Max != 50000 || q.Used != 2140 || q.MaxRate != 14 || q.Sandbox || q.Account != "Production · no enforcement" {
		t.Fatalf("quota: %+v", q)
	}
}

// A custom CA bundle (AWS_CA_BUNDLE, as behind a TLS-inspecting proxy)
// doesn't stop the role path from loading: the SDK gets a client it can add
// the bundle to.
func TestRoleWithCABundle(t *testing.T) {
	_, cert := signer(t)
	bundle := t.TempDir() + "/ca.pem"
	if err := os.WriteFile(bundle, cert, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_CA_BUNDLE", bundle)
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIDEXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	settings, _ := json.Marshal(map[string]string{"region": "us-east-1", "access": "role", "role_arn": "arn:aws:iam::123456789012:role/ReputeCastSES"})
	if _, err := ses.New(providers.Config{Settings: settings, HTTPClient: &http.Client{Timeout: time.Second}}); err != nil {
		t.Fatal(err)
	}
}
