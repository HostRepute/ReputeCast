// Package hrtest is a fake HostRepute (API and OAuth server) for tests. It
// answers with the demo data and checks what ReputeCast sends: the bearer
// token, the PKCE verifier, and that only domains and IPs are submitted.
package hrtest

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/hostrepute"
)

// Fake is a running fake HostRepute.
type Fake struct {
	*httptest.Server
	mu sync.Mutex
	// Token is the API key or access token the fake accepts.
	Token string
	// Monitored assets and the plan's limit (0 = unlimited).
	Monitored []string
	Limit     int
	// Down answers 503 to every call; Revoked answers 401.
	Down, Revoked bool
	// Alerts are returned by /alerts (the demo's when nil).
	Alerts []hostrepute.Alert
	// Revocations counts token revocations (disconnect).
	Revocations int
	// Requests logs "METHOD path" of every call.
	Requests []string
	// Bodies logs request bodies of POST calls.
	Bodies    []string
	challenge map[string]string
	now       func() time.Time
}

// New starts a fake HostRepute that accepts token.
func New(t testing.TB, token string) *Fake {
	f := &Fake{Token: token, Monitored: slices.Clone(hostrepute.DemoMonitored), Limit: hostrepute.DemoLimit, challenge: map[string]string{}, now: time.Now}
	f.Server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.Close)
	return f
}

// APIURL is the API base to configure (…/v1).
func (f *Fake) APIURL() string { return f.URL + "/v1" }

// Consent plays the consent page: it reads the authorize URL ReputeCast
// produced and returns the code HostRepute would send back, plus the state.
func (f *Fake) Consent(t testing.TB, authorizeURL string) (code, state string) {
	t.Helper()
	u, err := url.Parse(authorizeURL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" {
		t.Fatalf("authorize URL without PKCE: %s", authorizeURL)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	code = "code-" + q.Get("state")[:8]
	f.challenge[code] = q.Get("code_challenge")
	return code, q.Get("state")
}

func (f *Fake) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Requests = append(f.Requests, r.Method+" "+r.URL.Path)
	if r.Method == http.MethodPost && r.URL.Path != "/oauth/token" {
		var b strings.Builder
		_ = r.ParseForm()
		if r.Header.Get("Content-Type") == "application/json" {
			var m map[string]any
			_ = json.NewDecoder(r.Body).Decode(&m)
			raw, _ := json.Marshal(m)
			b.Write(raw)
		}
		f.Bodies = append(f.Bodies, b.String())
	}
	if f.Down {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	switch r.URL.Path {
	case "/oauth/token":
		f.token(w, r)
		return
	case "/oauth/revoke":
		f.Revocations++
		w.WriteHeader(http.StatusOK)
		return
	}
	if f.Revoked || r.Header.Get("Authorization") != "Bearer "+f.Token {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	now := f.now()
	path := strings.TrimPrefix(r.URL.Path, "/v1")
	switch {
	case path == "/me":
		acct := hostrepute.DemoAccount(len(f.Monitored))
		acct.Team.MonitorLimit = f.Limit
		writeJSON(w, acct)
	case path == "/assets" && r.Method == http.MethodGet:
		writeJSON(w, map[string]any{"assets": hostrepute.DemoAssets(now, f.Monitored)})
	case path == "/assets" && r.Method == http.MethodPost:
		var in struct{ Asset, Type string }
		_ = json.Unmarshal([]byte(f.Bodies[len(f.Bodies)-1]), &in)
		if f.Limit > 0 && len(f.Monitored) >= f.Limit {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			writeJSON(w, map[string]any{"code": "monitor_limit_reached", "detail": "plan full", "limit": f.Limit, "used": len(f.Monitored), "team": hostrepute.DemoTeam})
			return
		}
		f.Monitored = append(f.Monitored, in.Asset)
		writeJSON(w, hostrepute.DemoAssets(now, []string{in.Asset})[0])
	case strings.HasPrefix(path, "/assets/") && r.Method == http.MethodDelete:
		id := strings.TrimPrefix(path, "/assets/")
		for i, a := range hostrepute.DemoAssets(now, f.Monitored) {
			if a.ID == id || a.Asset == id {
				f.Monitored = slices.Delete(f.Monitored, i, i+1)
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	case path == "/blocklists":
		writeJSON(w, hostrepute.DemoBlocklists(now, f.Monitored))
	case strings.HasPrefix(path, "/postmaster/"):
		writeJSON(w, map[string]any{"days": hostrepute.DemoPostmaster(now, strings.TrimPrefix(path, "/postmaster/"))})
	case strings.HasPrefix(path, "/snds/"):
		writeJSON(w, map[string]any{"days": hostrepute.DemoSNDS(now, strings.TrimPrefix(path, "/snds/"))})
	case strings.HasPrefix(path, "/dmarc/"):
		d := strings.TrimSuffix(strings.TrimPrefix(path, "/dmarc/"), "/reports")
		writeJSON(w, map[string]any{"reports": hostrepute.DemoDMARC(now, d)})
	case path == "/alerts":
		alerts := f.Alerts
		if alerts == nil {
			alerts = hostrepute.DemoAlerts(now)
		}
		writeJSON(w, map[string]any{"alerts": alerts})
	case path == "/checks":
		writeJSON(w, hostrepute.Check{ID: "chk_1", Status: "done", Total: 84, Answered: 84, StartedAt: now})
	case strings.HasPrefix(path, "/checks/"):
		writeJSON(w, hostrepute.Check{ID: "chk_1", Status: "done", Total: 84, Answered: 84, StartedAt: now})
	case path == "/delistings":
		writeJSON(w, hostrepute.Delisting{ID: "dl_1", Status: "requested", RequestedAt: now})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *Fake) token(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	switch r.Form.Get("grant_type") {
	case "authorization_code":
		want, ok := f.challenge[r.Form.Get("code")]
		sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
		if !ok || base64.RawURLEncoding.EncodeToString(sum[:]) != want {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		delete(f.challenge, r.Form.Get("code"))
	case "refresh_token":
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"access_token": f.Token, "refresh_token": "refresh-" + f.Token, "token_type": "Bearer", "expires_in": 3600,
		"scope": strings.Join(hostrepute.Scopes, " ")})
}

func writeJSON(w http.ResponseWriter, v any) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	_ = json.NewEncoder(w).Encode(v)
}
