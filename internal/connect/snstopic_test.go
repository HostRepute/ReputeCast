package connect_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/connect"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

type snsFunc func(*http.Request) (*http.Response, error)

func (f snsFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// SNS asks the webhook to confirm a subscription as soon as it is made, and
// doesn't retry a refusal; the webhook trusts only the topics saved with the
// provider. So the step saves the topic before it subscribes the webhook:
// when SNS's confirmation arrives, the webhook already knows it.
func TestSESTopicSavedBeforeSubscribing(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	u := env.CreateUser(apptest.UserOpts{Email: "owner@snstopic.test", Name: "Pat Owner"})
	wsRow := env.CreateWorkspace("SNS Topic Test", u.ID)
	c := env.Client()
	c.SignIn(u.Email, "")
	var st struct{ ID string }
	c.Do("POST", "/workspaces/"+wsRow.Slug+"/connect/setups", map[string]any{"adapter": "ses"}).MustStatus(t, 201).JSON(t, &st)

	pid := uuid.Must(uuid.NewV7())
	cfg, _ := json.Marshal(map[string]any{"region": "us-east-1", "access": "keys", "events": "webhook"})
	creds, err := env.App.Secrets.Seal([]byte(`{"access_key_id":"AKIDEXAMPLE","secret_access_key":"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}`),
		manage.CredentialsAD(store.Provider{ID: pid}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, status, ip_mode, ips,
		capabilities, limits, health, pricing, connected_at)
		VALUES ($1, $2, 'ses', 'Amazon SES (topic)', '', $3, $4, 'token-snstopic-0001', 'healthy', 'shared', '{}', '{"webhooks":"SNS"}', '{}', '{}', '{}', now())`,
		pid, wsRow.ID, cfg, creds); err != nil {
		t.Fatal(err)
	}
	state := `{"connection":{"status":"connected","access":"keys","region":"us-east-1","events":"webhook","provisioning":[
		{"key":"sns_topic","status":"waiting","label":"Create SNS topic"}]}}`
	if _, err := env.App.Pool.Exec(ctx, `UPDATE provider_setups SET provider_id = $1, state = $2 WHERE id = $3`, pid, state, st.ID); err != nil {
		t.Fatal(err)
	}
	settings := func() map[string]any {
		t.Helper()
		var raw []byte
		if err := env.App.Pool.QueryRow(ctx, `SELECT config FROM providers WHERE id = $1`, pid).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		out := map[string]any{}
		_ = json.Unmarshal(raw, &out)
		return out
	}

	const topic = "arn:aws:sns:us-east-1:123456789012:reputecast-events"
	var atSubscribe map[string]any
	sns := snsFunc(func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		action := form.Get("Action")
		result := ""
		switch action {
		case "CreateTopic":
			result = `<TopicArn>` + topic + `</TopicArn>`
		case "Subscribe":
			// What the webhook reads when SNS's confirmation comes in.
			atSubscribe = settings()
			result = `<SubscriptionArn>` + topic + `:3f1c</SubscriptionArn>`
		default:
			return &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": {"text/xml"}}, Request: r,
				Body: io.NopCloser(bytes.NewBufferString(`<ErrorResponse><Error><Code>InvalidAction</Code><Message>` + action + `</Message></Error></ErrorResponse>`))}, nil
		}
		xml := `<` + action + `Response xmlns="http://sns.amazonaws.com/doc/2010-03-31/"><` + action + `Result>` + result + `</` + action + `Result>` +
			`<ResponseMetadata><RequestId>req-1</RequestId></ResponseMetadata></` + action + `Response>`
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/xml"}}, Request: r, Body: io.NopCloser(bytes.NewBufferString(xml))}, nil
	})
	svc := &connect.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets,
		Factory: &manage.Factory{Box: env.App.Secrets, HTTP: &http.Client{Transport: sns}}, URLs: env.App.URLs}
	if again, err := svc.RunProvisioning(ctx, uuid.MustParse(st.ID)); err != nil || again {
		t.Fatalf("run: again %v, %v", again, err)
	}
	if atSubscribe == nil || fmt.Sprint(atSubscribe["topic_arns"]) != "["+topic+"]" || atSubscribe["region"] != "us-east-1" {
		t.Fatalf("settings when the webhook subscribed: %v", atSubscribe)
	}
	after := settings()
	if fmt.Sprint(after["topic_arns"]) != "["+topic+"]" || after["topic_arn"] != topic || after["subscription_arn"] != topic+":3f1c" || after["access"] != "keys" {
		t.Fatalf("settings after the step: %v", after)
	}
}
