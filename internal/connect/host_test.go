package connect

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/store"
)

// storeProvider stands for a connected provider in canContinue.
var storeProvider = store.Provider{}

// PR-5: where ReputeCast runs decides the SES access it recommends.
func TestHostDetection(t *testing.T) {
	now := time.Date(2026, 10, 26, 9, 7, 0, 0, time.UTC)
	env := func(vars map[string]string) func(string) string {
		return func(k string) string { return vars[k] }
	}
	noFile := filepath.Join(t.TempDir(), "missing")
	probes := 0
	off := func(context.Context) bool { probes++; return false }
	on := func(context.Context) bool { probes++; return true }

	cases := []struct {
		name  string
		vars  map[string]string
		probe func(context.Context) bool
		want  HostInfo
	}{
		{"laptop", map[string]string{"AWS_SHARED_CREDENTIALS_FILE": noFile}, off, HostInfo{}},
		{"laptop with keys", map[string]string{"AWS_SHARED_CREDENTIALS_FILE": noFile, "AWS_ACCESS_KEY_ID": "AKIAEXAMPLE"}, off, HostInfo{BaseCredentials: true}},
		{"ec2", map[string]string{"AWS_SHARED_CREDENTIALS_FILE": noFile}, on, HostInfo{OnAWS: true, Kind: "ec2", BaseCredentials: true}},
		{"ecs", map[string]string{"ECS_CONTAINER_METADATA_URI_V4": "http://169.254.170.2/v4/x"}, off, HostInfo{OnAWS: true, Kind: "ecs", BaseCredentials: true}},
		{"eks", map[string]string{"AWS_WEB_IDENTITY_TOKEN_FILE": "/var/run/token", "AWS_ROLE_ARN": "arn:aws:iam::1:role/x"}, off, HostInfo{OnAWS: true, Kind: "eks", BaseCredentials: true}},
		{"lambda", map[string]string{"AWS_LAMBDA_FUNCTION_NAME": "fn"}, off, HostInfo{OnAWS: true, Kind: "lambda", BaseCredentials: true}},
	}
	for _, c := range cases {
		d := &HostDetector{Probe: c.probe, Env: env(c.vars), Now: func() time.Time { return now }}
		got := d.Detect(context.Background())
		got.CheckedAt = time.Time{}
		if got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
		want := "keys"
		if c.want.OnAWS {
			want = "role"
		}
		if got.RecommendedAccess() != want {
			t.Errorf("%s: recommends %s, want %s", c.name, got.RecommendedAccess(), want)
		}
	}

	// A shared credentials file counts as base credentials.
	file := filepath.Join(t.TempDir(), "credentials")
	if err := os.WriteFile(file, []byte("[default]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	d := &HostDetector{Probe: off, Env: env(map[string]string{"AWS_SHARED_CREDENTIALS_FILE": file})}
	if h := d.Detect(context.Background()); h.OnAWS || !h.BaseCredentials {
		t.Errorf("credentials file: %+v", h)
	}

	// The answer is kept for ten minutes: no second probe.
	probes = 0
	clock := now
	d = &HostDetector{Probe: off, Env: env(nil), Now: func() time.Time { return clock }}
	d.Detect(context.Background())
	clock = clock.Add(9 * time.Minute)
	d.Detect(context.Background())
	if probes != 1 {
		t.Errorf("probed %d times within ten minutes", probes)
	}
	clock = clock.Add(2 * time.Minute)
	d.Detect(context.Background())
	if probes != 2 {
		t.Errorf("probed %d times after the answer expired", probes)
	}
}

// PR-3: the event check's state, and when the setup may go on.
func TestEventsCheckState(t *testing.T) {
	at := time.Date(2026, 10, 26, 9, 8, 31, 0, time.UTC)
	rows := func(send, check string) []ProvisionState {
		return []ProvisionState{{Key: "tenant", Status: "done"}, {Key: eventSendKey, Status: send}, {Key: eventCheckKey, Status: check, At: &at}}
	}
	cases := []struct {
		rows  []ProvisionState
		state string
		cont  bool
	}{
		{nil, "none", true},
		{[]ProvisionState{{Key: "sns_topic", Status: "done"}}, "none", true},
		{rows("done", "done"), "verified", true},
		{rows(statusDeferred, "waiting"), "deferred", true},
		{rows("done", "waiting"), "waiting", false},
		{rows("running", "waiting"), "waiting", false},
		{rows("done", "failed"), "failed", false},
		{rows("failed", "waiting"), "waiting", false},
	}
	for i, c := range cases {
		got, when := eventsCheckOf(c.rows)
		if got != c.state {
			t.Errorf("case %d: state %s, want %s", i, got, c.state)
		}
		if got == "verified" && (when == nil || !when.Equal(at)) {
			t.Errorf("case %d: verified at %v", i, when)
		}
		cn := Connection{Status: "connected", Provisioning: c.rows}
		if canContinue("ses", cn, &storeProvider) != c.cont {
			t.Errorf("case %d: canContinue = %v", i, !c.cont)
		}
	}
}

// PR-9: the provider's test webhook, as the wizard shows it.
func TestWebhookTestView(t *testing.T) {
	asked := time.Date(2026, 10, 26, 9, 12, 5, 0, time.UTC)
	w := &WebhookState{TestEventAt: &asked}
	if v := webhookTestView(w, asked.Add(30*time.Second)); v.Status != "waiting" || !v.Until.Equal(asked.Add(time.Minute)) {
		t.Errorf("waiting: %+v", v)
	}
	if v := webhookTestView(w, asked.Add(61*time.Second)); v.Status != "missing" {
		t.Errorf("missing: %+v", v)
	}
	got := asked.Add(3 * time.Second)
	w.TestArrivedAt = &got
	if v := webhookTestView(w, asked.Add(5*time.Minute)); v.Status != "arrived" || !v.ArrivedAt.Equal(got) {
		t.Errorf("arrived: %+v", v)
	}
	w = &WebhookState{TestEventAt: &asked, TestEventErr: "403 Forbidden"}
	if v := webhookTestView(w, asked); v.Status != "refused" || v.Error != "403 Forbidden" {
		t.Errorf("refused: %+v", v)
	}

	// Continue: only once the test arrived, when the provider can send one;
	// otherwise a registered webhook at a reachable address.
	ok := true
	cn := Connection{Status: "connected", Webhook: &WebhookState{Mode: "webhook", CanTest: true, TestEventAt: &asked,
		Registration: &Registration{Status: "done"}, SelfTest: &SelfTestState{OK: ok}}}
	if canContinue("api", cn, &storeProvider) {
		t.Error("continued before the test webhook arrived")
	}
	cn.Webhook.TestArrivedAt = &got
	if !canContinue("api", cn, &storeProvider) {
		t.Error("blocked after the test webhook arrived")
	}
	cn.Webhook = &WebhookState{Mode: "webhook", Registration: &Registration{Status: "done"}, SelfTest: &SelfTestState{OK: true}}
	if !canContinue("api", cn, &storeProvider) {
		t.Error("a provider without test webhooks is confirmed by the test send")
	}
	cn.Webhook.SelfTest = &SelfTestState{OK: false, Private: true}
	if canContinue("api", cn, &storeProvider) {
		t.Error("continued with a private address")
	}
}
