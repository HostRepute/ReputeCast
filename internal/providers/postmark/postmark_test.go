package postmark_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/postmark"
)

const secret = "whsec-test-postmark"

func fakeAPI(env *contracttest.Env) http.Handler {
	mux := http.NewServeMux()
	fail := func(w http.ResponseWriter) bool {
		switch env.Mode() {
		case contracttest.ModeAuth:
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"ErrorCode":10,"Message":"Bad or missing API token."}`)
			return true
		case contracttest.ModeOutage:
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"ErrorCode":0,"Message":"Service unavailable"}`)
			return true
		}
		return false
	}
	mux.HandleFunc("POST /email/batch", func(w http.ResponseWriter, r *http.Request) {
		if fail(w) {
			return
		}
		var in []struct{ To string }
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = io.WriteString(w, `{"ErrorCode":402,"Message":"Received invalid JSON input."}`)
			return
		}
		out := []map[string]any{}
		for _, m := range in {
			if strings.Contains(m.To, contracttest.RejectAddress) {
				out = append(out, map[string]any{"ErrorCode": 406, "Message": "You tried to send to a recipient that has been marked as inactive."})
				continue
			}
			out = append(out, map[string]any{"ErrorCode": 0, "Message": "OK", "MessageID": uuid.NewString(), "To": m.To, "SubmittedAt": "2026-10-12T13:41:02Z"})
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	mux.HandleFunc("GET /server", func(w http.ResponseWriter, _ *http.Request) {
		if fail(w) {
			return
		}
		_, _ = io.WriteString(w, `{"ID":23,"Name":"Acme Newsletter","TrackOpens":true,"TrackLinks":"HtmlAndText"}`)
	})
	return mux
}

func newProvider(t *testing.T, env *contracttest.Env) providers.Provider {
	t.Helper()
	settings, _ := json.Marshal(postmark.Settings{Server: "Acme Newsletter", MessageStream: "outbound", BroadcastStream: "broadcast", MonthlyLimit: 10000, BaseURL: env.URL})
	p, err := postmark.New(providers.Config{Settings: settings, Credentials: json.RawMessage(`{"server_token":"server-token"}`), WebhookSecret: secret})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func webhook(file string) func(t *testing.T, env *contracttest.Env) (*http.Request, []byte) {
	return func(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) {
		body, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		r, _ := http.NewRequest(http.MethodPost, "https://reputecast.acme.io/hooks/postmark/token", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.SetBasicAuth(postmark.WebhookUser, secret)
		return r, body
	}
}

func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{
		Adapter: postmark.Adapter,
		API:     fakeAPI,
		New:     newProvider,
		Delivered: func(t *testing.T, env *contracttest.Env, msgs []providers.Message, _ []providers.SendResult) {
			reqs := env.Requests()
			if len(reqs) != 1 || reqs[0].Path != "/email/batch" || reqs[0].Header.Get("X-Postmark-Server-Token") != "server-token" {
				t.Fatalf("want one authenticated POST /email/batch, got %+v", reqs)
			}
			var sent []struct {
				MessageStream string
				TrackOpens    bool
				TrackLinks    string
				Headers       []struct{ Name, Value string }
				Metadata      map[string]string
			}
			if err := json.Unmarshal(reqs[0].Body, &sent); err != nil {
				t.Fatal(err)
			}
			for i, m := range sent {
				if m.MessageStream != "broadcast" || m.TrackOpens || m.TrackLinks != "None" {
					t.Fatalf("message %d: stream %q, tracking %v/%s: Broadcasts go to the Broadcast stream without Postmark tracking", i, m.MessageStream, m.TrackOpens, m.TrackLinks)
				}
				if m.Metadata[postmark.MetaMessage] != msgs[i].ID.String() {
					t.Fatalf("message %d metadata: %v", i, m.Metadata)
				}
				found := false
				for _, h := range m.Headers {
					if h.Name == "List-Unsubscribe-Post" && h.Value == "List-Unsubscribe=One-Click" {
						found = true
					}
				}
				if !found {
					t.Fatalf("message %d lost List-Unsubscribe-Post: %+v", i, m.Headers)
				}
			}
		},
		Tamper: func(r *http.Request, body []byte) (*http.Request, []byte) {
			r.SetBasicAuth(postmark.WebhookUser, "wrong-secret")
			r.Body = io.NopCloser(bytes.NewReader(body))
			return r, body
		},
		Webhooks: []contracttest.Webhook{
			{Name: "Delivery", Request: webhook("delivery.json"), Want: []providers.Event{{
				Type: providers.EventDelivered, Recipient: "d.f@outlook.com", ProviderMessageID: "883953f4-6105-42a2-a16a-77a8eac79483",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e40"), OriginalType: "Delivery", OriginalCode: "250 2.6.0",
			}}},
			{Name: "HardBounce", Request: webhook("bounce-hard.json"), Want: []providers.Event{{
				Type: providers.EventHardBounce, Recipient: "john@example.com", ProviderMessageID: "9a3f1c00-6105-42a2-a16a-77a8eac79483",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e41"), OriginalType: "Bounce · HardBounce", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown",
			}}},
			{Name: "SoftBounce", Request: webhook("bounce-soft.json"), Want: []providers.Event{{
				Type: providers.EventSoftBounce, Recipient: "b.h@yahoo.com", ProviderMessageID: "9a3f1c01-6105-42a2-a16a-77a8eac79483",
				OriginalType: "Bounce · SoftBounce", OriginalCode: "452 4.2.2", BounceClass: "mailbox_full",
			}}},
			{Name: "SpamComplaint", Request: webhook("spam-complaint.json"), Want: []providers.Event{{
				Type: providers.EventComplaint, Recipient: "m.g@aol.com", ProviderMessageID: "00000000-0000-0000-0000-000000000000",
				MessageID: contracttest.UUID(t, "0199d5a8-2b6e-7c41-9a0e-5b8f1c2d3e42"), OriginalType: "SpamComplaint", OriginalCode: "Type 512",
			}}},
		},
	})
}

func TestTracking(t *testing.T) {
	env := contracttest.NewEnv(t, fakeAPI)
	p := newProvider(t, env).(providers.TrackingManager)
	tr, err := p.Tracking(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !tr.OpenPixel || !tr.LinkRewrite || tr.Detail != "Server setting TrackOpens is on, and TrackLinks is HtmlAndText." {
		t.Fatalf("tracking: %+v", tr)
	}
}
