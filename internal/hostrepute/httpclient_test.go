package hostrepute

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/hostrepute/reputecast/internal/config"
)

func newTestClient(srv *httptest.Server, token string) *HTTPClient {
	c := NewHTTPClient(srv.URL+"/v1", oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token, TokenType: "Bearer"}), srv.Client())
	c.Wait = func(int) time.Duration { return time.Millisecond }
	return c
}

func TestHTTPClientAuthAndDecode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer hr_live_ok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/me":
			_, _ = w.Write([]byte(`{"id":"acct_1","email":"linh@acme.io","scopes":["monitor:read"],"sources":84,"team":{"id":"t1","name":"Acme Inc","monitor_limit":3,"monitored":2}}`))
		case "/v1/postmaster/news.acme.io":
			if r.URL.Query().Get("from") == "" {
				t.Errorf("missing date range: %s", r.URL)
			}
			_, _ = w.Write([]byte(`{"days":[{"day":"2026-10-01","spam_rate":0.0004,"domain_reputation":"HIGH"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	c := newTestClient(srv, "hr_live_ok")
	acct, err := c.Account(context.Background())
	if err != nil || acct.Team.Name != "Acme Inc" || acct.Team.MonitorLimit != 3 || acct.Sources != 84 {
		t.Fatalf("account: %+v %v", acct, err)
	}
	days, err := c.Postmaster(context.Background(), "news.acme.io", time.Now().AddDate(0, 0, -30), time.Now())
	if err != nil || len(days) != 1 || *days[0].SpamRate != 0.0004 {
		t.Fatalf("postmaster: %+v %v", days, err)
	}
	if _, err := c.Assets(context.Background()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("404: %v", err)
	}
	if _, err := newTestClient(srv, "hr_live_revoked").Account(context.Background()); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("401 must be ErrUnauthorized: %v", err)
	}
}

func TestHTTPClientLimitAndRetries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/assets":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"code":"monitor_limit_reached","detail":"plan full","limit":3,"used":3,"team":"Acme Inc"}`))
		case "/v1/me":
			if calls.Add(1) < 3 {
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			_, _ = w.Write([]byte(`{"id":"acct_1","team":{"name":"Acme Inc"}}`))
		case "/v1/blocklists":
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer srv.Close()
	c := newTestClient(srv, "hr_live_ok")
	_, err := c.AddAsset(context.Background(), "mta.acme.io", AssetHostname, RoleServerHostname)
	var le *LimitError
	if !errors.As(err, &le) || le.Limit != 3 || le.Used != 3 || le.Team != "Acme Inc" {
		t.Fatalf("limit: %v", err)
	}
	if _, err := c.Account(context.Background()); err != nil || calls.Load() != 3 {
		t.Fatalf("two 502s then success: %v after %d calls", err, calls.Load())
	}
	_, err = c.Blocklists(context.Background())
	var ue *UnavailableError
	if !errors.Is(err, ErrUnavailable) || !errors.As(err, &ue) || ue.Attempts != 3 || !strings.Contains(err.Error(), "HTTP 503") {
		t.Fatalf("unavailable after 3 attempts: %v", err)
	}
}

func TestHTTPClientTimeoutMessage(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-block }))
	defer srv.Close()
	defer close(block)
	c := newTestClient(srv, "hr_live_ok")
	c.HTTP = &http.Client{Timeout: 50 * time.Millisecond}
	_, err := c.Account(context.Background())
	if !errors.Is(err, ErrUnavailable) || !strings.Contains(err.Error(), "timeout after 50 ms, 3 attempts") {
		t.Fatalf("timeout: %v", err)
	}
	if !strings.HasPrefix(err.Error(), "Couldn't reach 127.0.0.1") {
		t.Fatalf("names the host: %v", err)
	}
}

func TestOAuthPKCE(t *testing.T) {
	o := NewOAuth(testConfig("https://app.hostrepute.test/oauth/authorize", "https://api.hostrepute.test/oauth/token"), "https://reputecast.acme.io/")
	v := NewVerifier()
	u := o.AuthCodeURL("state123", v)
	for _, want := range []string{"response_type=code", "client_id=reputecast", "code_challenge_method=S256", "state=state123",
		"redirect_uri=https%3A%2F%2Freputecast.acme.io%2Fapi%2Fv1%2Fintegrations%2Fhostrepute%2Fcallback", "scope=monitor%3Aread+alerts%3Aread+assets%3Awrite"} {
		if !strings.Contains(u, want) {
			t.Errorf("authorize URL lacks %s: %s", want, u)
		}
	}
	if strings.Contains(u, v) {
		t.Fatal("the verifier itself must never be in the URL")
	}
}

func TestDemoData(t *testing.T) {
	now := time.Date(2026, 10, 12, 15, 0, 0, 0, time.UTC)
	if n := len(demoSources()); n != 84 {
		t.Fatalf("demo HostRepute checks %d sources, want 84", n)
	}
	bl := DemoBlocklists(now, DemoMonitored)
	listed := 0
	for _, r := range bl.Results {
		if r.Listed {
			listed++
			if r.Source != "UCEPROTECT Level 3" || r.Asset != "203.0.113.25" || r.Delisting == nil {
				t.Fatalf("unexpected listing %+v", r)
			}
		}
	}
	if listed != 1 {
		t.Fatalf("listings: %d", listed)
	}
	var total int64
	for _, d := range DemoSNDS(now, "203.0.113.25")[2:] {
		total += d.Messages
	}
	if total != 11194 {
		t.Fatalf("SNDS 7-day messages: %d", total)
	}
	var sum float64
	for _, r := range demoSpamRates {
		sum += r
	}
	if avg := sum / float64(len(demoSpamRates)); avg < 0.045 || avg >= 0.055 {
		t.Fatalf("30-day spam rate must read 0.05%%: %.4f", avg)
	}
	c := NewDemoClient(func() time.Time { return now }, DemoMonitored)
	if _, err := c.AddAsset(context.Background(), "mta.acme.io", AssetHostname, ""); !errors.As(err, new(*LimitError)) {
		t.Fatalf("demo plan is full: %v", err)
	}
	if err := c.RemoveAsset(context.Background(), "acme.io"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AddAsset(context.Background(), "mta.acme.io", AssetHostname, ""); err != nil {
		t.Fatalf("room after removing: %v", err)
	}
}

func testConfig(authURL, tokenURL string) config.HostReputeConfig {
	c := config.Defaults().HostRepute
	c.AuthURL, c.TokenURL = authURL, tokenURL
	return c
}
