package connect_test

import (
	"context"
	"net/http"
	"slices"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
)

// fakeIPESP is an API provider whose API lists the account's dedicated IPs
// (providers.IPReporter): what ipAnswer says. Its name sorts before the
// other fakes', which TestCatalog expects last.
type fakeIPESP struct{}

var ipAnswer struct {
	sync.Mutex
	ips providers.SendingIPs
	err error
}

func answerIPs(ips providers.SendingIPs, err error) {
	ipAnswer.Lock()
	ipAnswer.ips, ipAnswer.err = ips, err
	ipAnswer.Unlock()
}

func init() {
	providers.Register(providers.Descriptor{
		Adapter: "fakeipesp", Name: "Fake Dedicated ESP", Kind: "api", Capabilities: fakeIPESP{}.Capabilities(),
		Fields: []providers.Field{{Key: "api_key", Label: "API key", Secret: true, Required: true}},
		New:    func(providers.Config) (providers.Provider, error) { return fakeIPESP{}, nil },
	})
}

func (fakeIPESP) Adapter() string { return "fakeipesp" }
func (fakeIPESP) Capabilities() providers.Capabilities {
	return providers.Capabilities{Bulk: providers.BulkAllowed, BatchSize: 1, QuotaDiscovery: true, Pricing: "$1 per 1,000", ReportsIPs: true}
}
func (fakeIPESP) Send(_ context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	return make([]providers.SendResult, len(msgs)), nil
}
func (fakeIPESP) ParseWebhook(context.Context, *http.Request, []byte) ([]providers.Event, error) {
	return []providers.Event{}, nil
}
func (fakeIPESP) Verify(context.Context) (providers.Quota, error) {
	return providers.Quota{Kind: "monthly_plan", Max: 50000, Account: "Active"}, nil
}
func (fakeIPESP) SendingIPs(context.Context) (providers.SendingIPs, error) {
	ipAnswer.Lock()
	defer ipAnswer.Unlock()
	return ipAnswer.ips, ipAnswer.err
}

// Connecting a provider whose API lists the account's dedicated IPs reads
// them (review GS-10), and so does repeating its connection step; a
// provider that can't answer is still connected, with its IPs as they were
// and the reason kept.
func TestConnectReadsSendingIPs(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var st setupT
	c.Do("POST", ws+"/connect/setups", map[string]any{"adapter": "fakeipesp"}).MustStatus(t, 201).JSON(t, &st)
	base := ws + "/connect/setups/" + st.ID
	ips := func() (string, []string, state.IPSource) {
		t.Helper()
		p, err := env.App.Q.ProvGetAny(ctx, uuid.MustParse(st.ProviderID))
		if err != nil {
			t.Fatal(err)
		}
		return p.IpMode, p.Ips, state.IPSourceOf(p)
	}
	updates := func() int {
		t.Helper()
		var n int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'provider.ips_update' AND target_id = $1
			AND actor_type = 'system' AND metadata->>'trigger' = 'connect'`, st.ProviderID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	answerIPs(providers.SendingIPs{Dedicated: true, IPs: []string{"198.51.100.62", "198.51.100.61"}}, nil)
	c.Do("POST", base+"/connection", map[string]any{"values": map[string]string{"api_key": "k-1"}}).MustStatus(t, 200).JSON(t, &st)
	if st.Connection.Status != "connected" || st.ProviderID == "" {
		t.Fatalf("connection: %+v", st.Connection)
	}
	mode, got, src := ips()
	if mode != "dedicated" || !slices.Equal(got, []string{"198.51.100.61", "198.51.100.62"}) || src.From != state.IPsFromProvider || src.CheckedAt == nil {
		t.Fatalf("after connecting: %s %q %+v", mode, got, src)
	}
	if n := updates(); n != 1 {
		t.Fatalf("%d audit entries for the IPs read at connect, want 1", n)
	}

	// The step again, with a new key: the account's IPs are read again.
	answerIPs(providers.SendingIPs{Dedicated: true, IPs: []string{"198.51.100.63"}}, nil)
	c.Do("POST", base+"/connection", map[string]any{"values": map[string]string{"api_key": "k-2"}}).MustStatus(t, 200).JSON(t, &st)
	if mode, got, _ := ips(); mode != "dedicated" || !slices.Equal(got, []string{"198.51.100.63"}) {
		t.Fatalf("after the step again: %s %q", mode, got)
	}
	if n := updates(); n != 2 {
		t.Fatalf("%d audit entries, want 2", n)
	}

	// A provider that can't answer is connected all the same; its IPs stay.
	answerIPs(providers.SendingIPs{}, &providers.CallError{Op: "GET /ips", Status: 403, Auth: true, Message: "This key can't read IP addresses"})
	c.Do("POST", base+"/connection", map[string]any{"values": map[string]string{"api_key": "k-3"}}).MustStatus(t, 200).JSON(t, &st)
	mode, got, src = ips()
	if st.Connection.Status != "connected" || mode != "dedicated" || !slices.Equal(got, []string{"198.51.100.63"}) || src.Error != "This key can't read IP addresses" {
		t.Fatalf("a refused check: %+v, %s %q %+v", st.Connection, mode, got, src)
	}
	if n := updates(); n != 2 {
		t.Fatalf("a refused check was audited as a change (%d entries)", n)
	}
}
