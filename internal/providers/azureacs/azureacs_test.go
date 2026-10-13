package azureacs_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/azureacs"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
)

var accessKey = base64.StdEncoding.EncodeToString([]byte("azure-test-access-key-0123456789"))

const secret = "whsec-test-azure"

// validSignature recomputes the HMAC an ACS request must carry, from the
// request as the server sees it.
func validSignature(r *http.Request, body []byte) bool {
	sum := sha256.Sum256(body)
	hash := base64.StdEncoding.EncodeToString(sum[:])
	if r.Header.Get("x-ms-content-sha256") != hash {
		return false
	}
	date := r.Header.Get("x-ms-date")
	if t, err := http.ParseTime(date); err != nil || time.Since(t) > 5*time.Minute {
		return false
	}
	key, _ := base64.StdEncoding.DecodeString(accessKey)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(r.Method + "\n" + r.URL.RequestURI() + "\n" + date + ";" + r.Host + ";" + hash))
	want := "HMAC-SHA256 SignedHeaders=x-ms-date;host;x-ms-content-sha256&Signature=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return r.Header.Get("Authorization") == want
}

func fakeAPI(tokens *atomic.Int32) func(env *contracttest.Env) http.Handler {
	return func(env *contracttest.Env) http.Handler {
		mux := http.NewServeMux()
		authorized := func(r *http.Request, body []byte) bool {
			return validSignature(r, body) || r.Header.Get("Authorization") == "Bearer entra-token"
		}
		fail := func(w http.ResponseWriter, r *http.Request, body []byte) bool {
			switch {
			case env.Mode() == contracttest.ModeOutage:
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = io.WriteString(w, `{"error":{"code":"ServiceUnavailable","message":"The service is temporarily unavailable."}}`)
				return true
			case env.Mode() == contracttest.ModeAuth || !authorized(r, body):
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = io.WriteString(w, `{"error":{"code":"Denied","message":"Denied by the resource provider."}}`)
				return true
			}
			return false
		}
		mux.HandleFunc("POST /emails:send", func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			if fail(w, r, body) {
				return
			}
			if r.URL.Query().Get("api-version") != azureacs.APIVersion {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var in struct {
				Recipients struct {
					To []struct{ Address string } `json:"to"`
				} `json:"recipients"`
				UserEngagementTrackingDisabled bool `json:"userEngagementTrackingDisabled"`
			}
			_ = json.Unmarshal(body, &in)
			if in.Recipients.To[0].Address == contracttest.RejectAddress {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":{"code":"InvalidRecipient","message":"The recipient address is not valid."}}`)
				return
			}
			op := r.Header.Get("Operation-Id")
			w.Header().Set("Operation-Location", "https://acme.communication.azure.com/emails/operations/"+op+"?api-version="+azureacs.APIVersion)
			w.Header().Set("Retry-After", "20")
			w.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(w, `{"id":"`+op+`","status":"Running"}`)
		})
		mux.HandleFunc("GET /emails/operations/{id}", func(w http.ResponseWriter, r *http.Request) {
			if fail(w, r, nil) {
				return
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":{"code":"NotFound","message":"Operation not found."}}`)
		})
		mux.HandleFunc("POST /acme-tenant/oauth2/v2.0/token", func(w http.ResponseWriter, r *http.Request) {
			_ = r.ParseForm()
			if r.PostForm.Get("client_secret") != "app-secret" || r.PostForm.Get("scope") != "https://communication.azure.com/.default" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":"invalid_client","error_description":"AADSTS7000215: Invalid client secret provided.\r\nTrace ID: x"}`)
				return
			}
			tokens.Add(1)
			_, _ = io.WriteString(w, `{"token_type":"Bearer","expires_in":3599,"access_token":"entra-token"}`)
		})
		return mux
	}
}

func newProvider(t *testing.T, env *contracttest.Env) providers.Provider {
	t.Helper()
	creds, _ := json.Marshal(map[string]string{"connection_string": "endpoint=" + env.URL + "/;accesskey=" + accessKey})
	settings, _ := json.Marshal(azureacs.Settings{PerMinute: 30, PerHour: 100})
	p, err := azureacs.New(providers.Config{Settings: settings, Credentials: creds, WebhookSecret: secret})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func delivery(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) {
	body, err := os.ReadFile("testdata/events.json")
	if err != nil {
		t.Fatal(err)
	}
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/azureacs/token", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	r.Header.Set("aeg-event-type", "Notification")
	r.Header.Set(azureacs.SecretHeader, secret)
	return r, body
}

func TestContract(t *testing.T) {
	var tokens atomic.Int32
	contracttest.Run(t, contracttest.Suite{
		Adapter: azureacs.Adapter,
		API:     fakeAPI(&tokens),
		New:     newProvider,
		Delivered: func(t *testing.T, env *contracttest.Env, msgs []providers.Message, res []providers.SendResult) {
			reqs := env.Requests()
			if len(reqs) != 1 || !strings.HasPrefix(reqs[0].Path, "/emails:send?api-version=") || reqs[0].Header.Get("Operation-Id") != msgs[0].ID.String() {
				t.Fatalf("want one signed send with the message id as operation id, got %+v", reqs)
			}
			var sent struct {
				SenderAddress                  string            `json:"senderAddress"`
				Headers                        map[string]string `json:"headers"`
				UserEngagementTrackingDisabled bool              `json:"userEngagementTrackingDisabled"`
			}
			if err := json.Unmarshal(reqs[0].Body, &sent); err != nil {
				t.Fatal(err)
			}
			if sent.SenderAddress != "news@news.acme.io" || !sent.UserEngagementTrackingDisabled || sent.Headers["List-Unsubscribe-Post"] != "List-Unsubscribe=One-Click" {
				t.Fatalf("request: %+v", sent)
			}
			if res[0].ProviderMessageID != msgs[0].ID.String() {
				t.Fatalf("the operation id is the provider id: %q", res[0].ProviderMessageID)
			}
		},
		Tamper: func(r *http.Request, body []byte) (*http.Request, []byte) {
			r.Header.Set(azureacs.SecretHeader, "wrong-secret")
			r.Body = io.NopCloser(bytes.NewReader(body))
			return r, body
		},
		Webhooks: []contracttest.Webhook{{
			Name: "DeliveryReports", Request: delivery,
			Want: []providers.Event{
				{Type: providers.EventDelivered, Recipient: "d.f@outlook.com", ProviderMessageID: "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40"), OriginalType: "EmailDeliveryReportReceived · Delivered", OriginalCode: "250 2.6.0"},
				{Type: providers.EventHardBounce, Recipient: "john@example.com", ProviderMessageID: "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e41",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e41"), OriginalType: "EmailDeliveryReportReceived · Bounced", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown"},
				{Type: providers.EventSoftBounce, Recipient: "k.n@hotmail.com", ProviderMessageID: "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e42",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e42"), OriginalType: "EmailDeliveryReportReceived · Bounced", OriginalCode: "550 5.7.1", BounceClass: "spam_block"},
				{Type: providers.EventRejected, Recipient: "old@example.org", ProviderMessageID: "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e44",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e44"), OriginalType: "EmailDeliveryReportReceived · Suppressed", BounceClass: "mailbox_unknown"},
				{Type: providers.EventSoftBounce, Recipient: "b.h@yahoo.com", ProviderMessageID: "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e47",
					MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e47"), OriginalType: "EmailDeliveryReportReceived · FilteredSpam", BounceClass: "spam_block"},
			},
		}},
	})
}

func TestSubscriptionValidation(t *testing.T) {
	env := contracttest.NewEnv(t, nil)
	env.URL = "https://acme.communication.azure.com"
	p := newProvider(t, env).(providers.WebhookHandshaker)
	body := []byte(`[{"id":"2d1781af-3a4c-4d7c-bd0c-e34b19da4e66","topic":"/subscriptions/x","subject":"","data":{"validationCode":"512d38b6-c7b8-40c8-89fe-f46f9e9622b6","validationUrl":"https://rp-eastus2.eventgrid.azure.net:553/eventsubscriptions/x/validate?id=512d38b6"},"eventType":"Microsoft.EventGrid.SubscriptionValidationEvent","eventTime":"2026-10-12T13:41:02Z","metadataVersion":"1","dataVersion":"2"}]`)
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/azureacs/token", bytes.NewReader(body))
	r.Header.Set("aeg-event-type", "SubscriptionValidation")
	handled, resp, err := p.Handshake(context.Background(), r, body)
	if err != nil || !handled || string(resp) != `{"validationResponse":"512d38b6-c7b8-40c8-89fe-f46f9e9622b6"}` {
		t.Fatalf("handshake: %v %s %v", handled, resp, err)
	}
	notif, events := delivery(t, env)
	if handled, _, _ := p.Handshake(context.Background(), notif, events); handled {
		t.Fatal("a notification is not a handshake")
	}
	r.Header.Set("aeg-event-type", "SubscriptionDeletion")
	if handled, _, err := p.Handshake(context.Background(), r, []byte(`[]`)); !handled || err != nil {
		t.Fatal("a subscription deletion is acknowledged")
	}
}

func TestSecretPlacements(t *testing.T) {
	env := contracttest.NewEnv(t, nil)
	env.URL = "https://acme.communication.azure.com"
	p := newProvider(t, env).(providers.WebhookVerifier)
	for name, set := range map[string]func(r *http.Request){
		"header": func(r *http.Request) { r.Header.Set(azureacs.SecretHeader, secret) },
		"bearer": func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+secret) },
		"query":  func(r *http.Request) { r.URL.RawQuery = "secret=" + secret },
	} {
		r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://reputecast.acme.io/hooks/azureacs/token", nil)
		if err := p.VerifyWebhook(context.Background(), r, nil); !errors.Is(err, providers.ErrBadSignature) {
			t.Fatalf("%s: a request without the secret is refused", name)
		}
		set(r)
		if err := p.VerifyWebhook(context.Background(), r, nil); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestVerifyAndEntraID(t *testing.T) {
	var tokens atomic.Int32
	env := contracttest.NewEnv(t, fakeAPI(&tokens))
	q, err := newProvider(t, env).(providers.Verifier).Verify(context.Background())
	if err != nil || q.MaxRate != 0.5 || q.Account != "Communication Services · reachable · 30 per minute · 100 per hour" {
		t.Fatalf("verify: %+v, %v", q, err)
	}
	settings, _ := json.Marshal(azureacs.Settings{Endpoint: env.URL, TenantID: "acme-tenant", ClientID: "app-id", AuthorityURL: env.URL})
	p, err := azureacs.New(providers.Config{Settings: settings, Credentials: json.RawMessage(`{"client_secret":"app-secret"}`)})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		res, err := p.Send(context.Background(), []providers.Message{contracttest.Message(0)})
		if err != nil || res[0].Err != nil {
			t.Fatalf("send with an Entra ID token: %+v, %v", res, err)
		}
	}
	if tokens.Load() != 1 {
		t.Fatalf("the token is reused until it expires: %d fetched", tokens.Load())
	}
	bad, _ := azureacs.New(providers.Config{Settings: settings, Credentials: json.RawMessage(`{"client_secret":"wrong"}`)})
	if _, err := bad.(providers.Verifier).Verify(context.Background()); !errors.Is(err, providers.ErrUnauthorized) || !strings.Contains(err.Error(), "AADSTS7000215") {
		t.Fatalf("a wrong client secret is rejected credentials: %v", err)
	}
	if _, err := azureacs.New(providers.Config{Credentials: json.RawMessage(`{"connection_string":"accesskey=abc"}`)}); err == nil {
		t.Fatal("a connection string without an endpoint is refused")
	}
	// Credentials only go over https, or plain http on this machine.
	for endpoint, ok := range map[string]bool{
		"https://acme.communication.azure.com": true,
		"http://127.0.0.1:8080":                true,
		"http://[::1]:8080":                    true,
		"http://localhost:8080":                true,
		"http://acme.communication.azure.com":  false,
		"http://127.0.0.1.attacker.example":    false,
		"ftp://acme.communication.azure.com":   false,
	} {
		s, _ := json.Marshal(azureacs.Settings{Endpoint: endpoint})
		if _, err := azureacs.New(providers.Config{Settings: s}); (err == nil) != ok {
			t.Fatalf("endpoint %s: %v", endpoint, err)
		}
		s, _ = json.Marshal(azureacs.Settings{Endpoint: "https://acme.communication.azure.com", AuthorityURL: endpoint})
		if _, err := azureacs.New(providers.Config{Settings: s}); (err == nil) != ok {
			t.Fatalf("authority %s: %v", endpoint, err)
		}
	}
}

func TestSign(t *testing.T) {
	key, _ := base64.StdEncoding.DecodeString(accessKey)
	got := azureacs.Sign(key, "POST", "/emails:send?api-version=2025-09-01", "Mon, 12 Oct 2026 13:41:02 GMT", "acme.communication.azure.com", "47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=")
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("POST\n/emails:send?api-version=2025-09-01\nMon, 12 Oct 2026 13:41:02 GMT;acme.communication.azure.com;47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU="))
	if got != base64.StdEncoding.EncodeToString(mac.Sum(nil)) {
		t.Fatalf("string to sign: VERB\\npath?query\\ndate;host;hash, got %s", got)
	}
}
