package mcpconnect_test

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pquerna/otp/totp"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/mcpconnect"
	"github.com/hostrepute/reputecast/internal/server"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/testdb"
)

type fixture struct {
	t     *testing.T
	env   *apptest.Env
	human *apptest.Client
	owner store.User
	ws    store.Workspace
	base  string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	e := apptest.New(t)
	if _, err := e.App.Pool.Exec(context.Background(), `UPDATE instance_settings SET twofa_enforcement = 'off'`); err != nil {
		t.Fatal(err)
	}
	u := e.CreateUser(apptest.UserOpts{Email: "editor@connect.test", Name: "Connect Editor"})
	w := e.CreateWorkspace("Editorial", u.ID)
	c := e.Client()
	c.SignIn(u.Email, "")
	return &fixture{t: t, env: e, human: c, owner: u, ws: w, base: "/workspaces/" + w.Slug + "/mcp-connect"}
}

func request(t *testing.T, base, method, path, contentType, body string, headers http.Header) *apptest.Response {
	t.Helper()
	r, err := http.NewRequestWithContext(context.Background(), method, base+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header = headers.Clone()
	if r.Header == nil {
		r.Header = http.Header{}
	}
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	c := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return &apptest.Response{Status: resp.StatusCode, Header: resp.Header, Body: b}
}

func (f *fixture) client(method string) mcpconnect.RegisteredClient {
	f.t.Helper()
	b, _ := json.Marshal(mcpconnect.ClientRegistration{Name: "Unverified editorial client", RedirectURIs: []string{"https://client.example/callback?flow=editor"}, AuthMethod: method})
	var out mcpconnect.RegisteredClient
	r := request(f.t, f.env.Server.URL, "POST", "/oauth/mcp/register", "application/json", string(b), nil).MustStatus(f.t, 201)
	r.JSON(f.t, &out)
	if r.Header.Get("Cache-Control") != "no-store" || out.ID == "" || (method != "none" && out.Secret == "") {
		f.t.Fatal("registration or no-store missing")
	}
	return out
}

func (f *fixture) authorization(c mcpconnect.RegisteredClient) (url.Values, string) {
	verifier, _ := authn.NewToken()
	sum := sha256.Sum256([]byte(verifier))
	return url.Values{"client_id": {c.ID}, "redirect_uri": {c.RedirectURIs[0]}, "response_type": {"code"}, "scope": {"editorial:read reports:read drafts:propose"}, "state": {"state + / unchanged"}, "resource": {f.env.Server.URL + mcpconnect.Path}, "code_challenge_method": {"S256"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}}, verifier
}

func (f *fixture) pending(values url.Values) string {
	f.t.Helper()
	r := request(f.t, f.env.Server.URL, "GET", "/oauth/mcp/authorize?"+values.Encode(), "", "", nil).MustStatus(f.t, 303)
	u, err := url.Parse(r.Header.Get("Location"))
	if err != nil || u.Scheme+"://"+u.Host != f.env.Server.URL || u.Path != "/mcp-consent" {
		f.t.Fatal("unsafe consent redirect")
	}
	return u.Query().Get("request")
}

func (f *fixture) consentBody(handle string) mcpconnect.MCPConsentDecision {
	return mcpconnect.MCPConsentDecision{Request: handle, Scopes: []string{"editorial:read", "reports:read", "drafts:propose"}, Lists: []uuid.UUID{}, SenderDomains: []string{}, AllowedIPs: []string{}}
}

func (f *fixture) allow(c mcpconnect.RegisteredClient) (string, string) {
	f.t.Helper()
	v, verifier := f.authorization(c)
	handle := f.pending(v)
	var consent mcpconnect.MCPConsent
	f.human.Do("GET", "/mcp-connect/consent?request="+handle, nil).MustStatus(f.t, 200).JSON(f.t, &consent)
	if consent.Verified || consent.CallbackHost != "client.example" || !slices.Equal(consent.Scopes, f.consentBody(handle).Scopes) {
		f.t.Fatalf("consent identity/scopes: %+v", consent)
	}
	var out mcpconnect.MCPConsentResult
	f.human.Do("POST", f.base+"/consent", f.consentBody(handle)).MustStatus(f.t, 200).JSON(f.t, &out)
	u, err := url.Parse(out.Redirect)
	if err != nil || u.Host != "client.example" || u.Path != "/callback" || u.Query().Get("state") != v.Get("state") || u.Query().Get("iss") != f.env.Server.URL || u.Query().Get("flow") != "editor" {
		f.t.Fatal("authorization response lost callback, issuer or state binding")
	}
	f.human.Do("POST", f.base+"/consent", f.consentBody(handle)).MustStatus(f.t, 409)
	return u.Query().Get("code"), verifier
}

func (f *fixture) exchangeAt(base string, c mcpconnect.RegisteredClient, values url.Values) *apptest.Response {
	f.t.Helper()
	headers := http.Header{}
	switch c.AuthMethod {
	case "client_secret_basic":
		headers.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(url.QueryEscape(c.ID)+":"+url.QueryEscape(c.Secret))))
	case "client_secret_post":
		values.Set("client_id", c.ID)
		values.Set("client_secret", c.Secret)
	default:
		values.Set("client_id", c.ID)
	}
	return request(f.t, base, "POST", "/oauth/mcp/token", "application/x-www-form-urlencoded", values.Encode(), headers)
}

func (f *fixture) codeValues(c mcpconnect.RegisteredClient, code, verifier string) url.Values {
	return url.Values{"grant_type": {"authorization_code"}, "code": {code}, "code_verifier": {verifier}, "redirect_uri": {c.RedirectURIs[0]}, "resource": {f.env.Server.URL + mcpconnect.Path}}
}

func (f *fixture) refreshValues(token string) url.Values {
	return url.Values{"grant_type": {"refresh_token"}, "refresh_token": {token}, "resource": {f.env.Server.URL + mcpconnect.Path}}
}

func (f *fixture) exchange(c mcpconnect.RegisteredClient, code, verifier string) mcpconnect.TokenResponse {
	f.t.Helper()
	var out mcpconnect.TokenResponse
	r := f.exchangeAt(f.env.Server.URL, c, f.codeValues(c, code, verifier)).MustStatus(f.t, 200)
	r.JSON(f.t, &out)
	if out.ExpiresIn != 600 || out.AccessToken == "" || out.RefreshToken == "" || r.Header.Get("Cache-Control") != "no-store" {
		f.t.Fatal("incomplete token response")
	}
	return out
}

type bearer string

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+string(b))
	return http.DefaultTransport.RoundTrip(r)
}

func (f *fixture) mcp(token string) *mcp.ClientSession {
	f.t.Helper()
	c := mcp.NewClient(&mcp.Implementation{Name: "oauth-http-test", Version: "1"}, nil)
	s, err := c.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: f.env.Server.URL + mcpconnect.Path, HTTPClient: &http.Client{Transport: bearer(token)}, DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	if err != nil {
		f.t.Fatal(err)
	}
	f.t.Cleanup(func() { _ = s.Close() })
	return s
}

func (f *fixture) deniedToken(token string) {
	f.t.Helper()
	request(f.t, f.env.Server.URL, "POST", mcpconnect.Path, "application/json", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, http.Header{"Authorization": {"Bearer " + token}, "Accept": {"application/json, text/event-stream"}}).MustStatus(f.t, 401)
}

func TestOAuthHTTP(t *testing.T) {
	for _, method := range []string{"none", "client_secret_basic", "client_secret_post"} {
		t.Run(method, func(t *testing.T) {
			f := newFixture(t)
			ctx := context.Background()
			discovery := request(t, f.env.Server.URL, "GET", "/.well-known/oauth-authorization-server", "", "", nil).MustStatus(t, 200)
			var metadata map[string]any
			discovery.JSON(t, &metadata)
			if metadata["issuer"] != f.env.Server.URL || metadata["token_endpoint"] != f.env.Server.URL+"/oauth/mcp/token" {
				t.Fatal("incorrect discovery")
			}
			challenge := request(t, f.env.Server.URL, "POST", mcpconnect.Path, "application/json", `{}`, nil).MustStatus(t, 401)
			if !strings.Contains(challenge.Header.Get("WWW-Authenticate"), f.env.Server.URL+"/.well-known/oauth-protected-resource/mcp/connect") {
				t.Fatal("missing resource discovery challenge")
			}
			c := f.client(method)
			code, verifier := f.allow(c)
			tokens := f.exchange(c, code, verifier)
			f.exchangeAt(f.env.Server.URL, c, f.codeValues(c, code, verifier)).MustStatus(t, 400)
			cs := f.mcp(tokens.AccessToken)
			out, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "connection_status", Arguments: map[string]any{}})
			if err != nil || out.IsError {
				t.Fatalf("connection_status: %v %+v", err, out)
			}
			for _, forbidden := range []string{"send_campaign", "list_contacts", "publish_automation", "start_agent_run"} {
				out, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: forbidden, Arguments: map[string]any{}})
				if err == nil && !out.IsError {
					t.Fatalf("forbidden tool %s accepted", forbidden)
				}
			}
			// Neither REST nor the legacy MCP transport accepts this token.
			other := f.env.Client()
			other.Bearer = tokens.AccessToken
			other.Do("GET", "/workspaces/"+f.ws.Slug+"/campaigns", nil).MustStatus(t, 401)
			request(t, f.env.Server.URL, "POST", "/mcp", "application/json", `{}`, http.Header{"Authorization": {"Bearer " + tokens.AccessToken}}).MustStatus(t, 401)
			var rotated mcpconnect.TokenResponse
			f.exchangeAt(f.env.Server.URL, c, f.refreshValues(tokens.RefreshToken)).MustStatus(t, 200).JSON(t, &rotated)
			if rotated.RefreshToken == tokens.RefreshToken {
				t.Fatal("refresh token was not rotated")
			}
			f.mcp(rotated.AccessToken)
			f.exchangeAt(f.env.Server.URL, c, f.refreshValues(tokens.RefreshToken)).MustStatus(t, 400)
			f.deniedToken(tokens.AccessToken)
			f.deniedToken(rotated.AccessToken)
			f.exchangeAt(f.env.Server.URL, c, f.refreshValues(rotated.RefreshToken)).MustStatus(t, 400)
			var reason, actor string
			if err = f.env.App.Pool.QueryRow(ctx, `SELECT revoked_reason FROM mcp_connect_grants WHERE client_id=$1`, c.ID).Scan(&reason); err != nil || reason != "refresh_reuse" {
				t.Fatalf("family revocation: %s %v", reason, err)
			}
			if err = f.env.App.Pool.QueryRow(ctx, `SELECT actor_type FROM audit_events WHERE action='mcp_connect.granted'`).Scan(&actor); err != nil || actor != "user" {
				t.Fatalf("human consent audit: %s %v", actor, err)
			}
			var leaked bool
			if err = f.env.App.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit_events e WHERE row_to_json(e)::text LIKE '%' || $1 || '%' OR row_to_json(e)::text LIKE '%' || $2 || '%' OR row_to_json(e)::text LIKE '%' || $3 || '%')`, tokens.AccessToken, tokens.RefreshToken, code).Scan(&leaked); err != nil || leaked {
				t.Fatalf("token in audit: %v %v", leaked, err)
			}
			var budgets int
			if err = f.env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM ai_budget_reservations`).Scan(&budgets); err != nil || budgets != 0 {
				t.Fatalf("external MCP budget use: %d %v", budgets, err)
			}
		})
	}
}

func TestOAuthBindingsAndRaces(t *testing.T) {
	f := newFixture(t)
	c, foreign := f.client("none"), f.client("client_secret_post")
	v, _ := f.authorization(c)
	for _, change := range []struct{ key, value string }{{"redirect_uri", "https://evil.example/callback"}, {"resource", f.env.Server.URL + "/mcp"}, {"code_challenge_method", "plain"}, {"scope", "ai:use"}, {"client_id", "unknown"}} {
		bad := v.Clone()
		bad.Set(change.key, change.value)
		r := request(t, f.env.Server.URL, "GET", "/oauth/mcp/authorize?"+bad.Encode(), "", "", nil)
		if r.Status < 400 || r.Header.Get("Location") != "" {
			t.Fatalf("unsafe authorization redirect for %s", change.key)
		}
	}
	request(t, f.env.Server.URL, "GET", "/oauth/mcp/authorize?"+v.Encode()+"&client_id="+c.ID, "", "", nil).MustStatus(t, 400)
	code, verifier := f.allow(c)
	for _, change := range []struct{ key, value string }{{"redirect_uri", "https://client.example/callback"}, {"resource", f.env.Server.URL + "/mcp"}, {"code_verifier", strings.Repeat("x", 43)}} {
		bad := f.codeValues(c, code, verifier)
		bad.Set(change.key, change.value)
		f.exchangeAt(f.env.Server.URL, c, bad).MustStatus(t, 400)
	}
	f.exchangeAt(f.env.Server.URL, foreign, f.codeValues(c, code, verifier)).MustStatus(t, 400)
	// A separate OS process has no access to this process's locks or caches.
	second := f.replica()
	results := make(chan *apptest.Response, 2)
	var wg sync.WaitGroup
	for _, base := range []string{f.env.Server.URL, second} {
		wg.Go(func() { results <- f.exchangeAt(base, c, f.codeValues(c, code, verifier)) })
	}
	wg.Wait()
	var tokens mcpconnect.TokenResponse
	statuses := []int{}
	for range 2 {
		r := <-results
		statuses = append(statuses, r.Status)
		if r.Status == 200 {
			r.JSON(t, &tokens)
		}
	}
	slices.Sort(statuses)
	if !slices.Equal(statuses, []int{200, 400}) {
		t.Fatalf("code race: %v", statuses)
	}
	// A different authenticated client cannot revoke the victim through replay.
	f.exchangeAt(f.env.Server.URL, foreign, f.refreshValues(tokens.RefreshToken)).MustStatus(t, 400)
	f.mcp(tokens.AccessToken)
	for _, base := range []string{f.env.Server.URL, second} {
		wg.Go(func() { results <- f.exchangeAt(base, c, f.refreshValues(tokens.RefreshToken)) })
	}
	wg.Wait()
	statuses = nil
	var winner mcpconnect.TokenResponse
	for range 2 {
		r := <-results
		statuses = append(statuses, r.Status)
		if r.Status == 200 {
			r.JSON(t, &winner)
		}
	}
	slices.Sort(statuses)
	if !slices.Equal(statuses, []int{200, 400}) {
		t.Fatalf("refresh race: %v", statuses)
	}
	f.deniedToken(winner.AccessToken)
	var n int
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM mcp_connect_tokens`).Scan(&n); err != nil || n != 4 {
		t.Fatalf("race issued extra tokens: %d %v", n, err)
	}
}

func TestOAuthReplicaProcess(t *testing.T) {
	url := os.Getenv("REPUTECAST_OAUTH_TEST_DB")
	if url == "" {
		t.Skip("subprocess helper")
	}
	pool, err := pgxpool.New(t.Context(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	q := store.New(pool)
	s := &mcpconnect.Service{Pool: pool, Q: q, Audit: audit.New(q, slog.New(slog.NewTextHandler(io.Discard, nil))),
		Development: true, PublicURL: func() string { return os.Getenv("REPUTECAST_OAUTH_TEST_ORIGIN") }}
	router := chi.NewRouter()
	router.Use(server.AuditContext)
	s.RegisterPublic(router)
	httpServer := httptest.NewServer(router)
	defer httpServer.Close()
	fmt.Println(httpServer.URL)
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func (f *fixture) replica() string {
	f.t.Helper()
	exe, err := os.Executable()
	if err != nil {
		f.t.Fatal(err)
	}
	cmd := exec.CommandContext(f.t.Context(), exe, "-test.run=^TestOAuthReplicaProcess$")
	cmd.Env = append(os.Environ(), "REPUTECAST_OAUTH_TEST_DB="+testdb.URL(f.env.App.Pool), "REPUTECAST_OAUTH_TEST_ORIGIN="+f.env.Server.URL)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		f.t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		f.t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		f.t.Fatal(err)
	}
	f.t.Cleanup(func() { _ = stdin.Close(); _ = cmd.Wait() })
	ready := make(chan string, 1)
	go func() { line, _ := bufio.NewReader(stdout).ReadString('\n'); ready <- strings.TrimSpace(line) }()
	select {
	case url := <-ready:
		if !strings.HasPrefix(url, "http://127.0.0.1:") {
			f.t.Fatal("OAuth replica did not start")
		}
		return url
	case <-time.After(10 * time.Second):
		f.t.Fatal("OAuth replica startup timed out")
		return ""
	}
}

func TestOAuthHumanConsentAndRecovery(t *testing.T) {
	f := newFixture(t)
	c := f.client("none")
	v, _ := f.authorization(c)
	handle := f.pending(v)
	// Approval cannot silently bind a request which was never opened for review.
	f.human.Do("POST", f.base+"/consent", f.consentBody(handle)).MustStatus(t, 409)
	f.human.Do("GET", "/mcp-connect/consent?request="+handle, nil).MustStatus(t, 200)
	otherSession := f.env.Client()
	otherSession.SignIn(f.owner.Email, "")
	otherSession.Do("GET", "/mcp-connect/consent?request="+handle, nil).MustStatus(t, 409)
	otherSession.Do("POST", f.base+"/consent", f.consentBody(handle)).MustStatus(t, 409)
	f.human.Header = http.Header{"Sec-Fetch-Site": {"cross-site"}, "Origin": {"https://evil.example"}}
	f.human.Do("POST", f.base+"/consent", f.consentBody(handle)).MustStatus(t, 403)
	f.human.Header = nil
	var denied mcpconnect.MCPConsentResult
	f.human.Do("POST", "/mcp-connect/consent/deny", map[string]any{"request": handle}).MustStatus(t, 200).JSON(t, &denied)
	u, _ := url.Parse(denied.Redirect)
	if u.Query().Get("error") != "access_denied" || u.Query().Has("code") {
		t.Fatal("denial granted a code")
	}
	f.human.Do("POST", f.base+"/consent", f.consentBody(handle)).MustStatus(t, 409)
	var n int
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM mcp_connect_grants`).Scan(&n); err != nil || n != 0 {
		t.Fatal("opening or denying created a grant", err)
	}
	code, verifier := f.allow(c)
	tokens := f.exchange(c, code, verifier)
	// Losing grant authority must not remove the owner's revocation escape hatch.
	if _, err := f.env.App.Pool.Exec(context.Background(), `UPDATE workspace_members SET role_id=(SELECT id FROM roles WHERE workspace_id=$1 AND key='viewer'), scope='{"lists":["00000000-0000-0000-0000-000000000001"]}' WHERE workspace_id=$1 AND user_id=$2`, f.ws.ID, f.owner.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.env.App.Pool.Exec(context.Background(), `UPDATE workspaces SET mcp_settings='{"enabled":false}' WHERE id=$1`, f.ws.ID); err != nil {
		t.Fatal(err)
	}
	f.deniedToken(tokens.AccessToken)
	var list mcpconnect.MCPConnectionList
	f.human.Do("GET", f.base+"/connections", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 {
		t.Fatal("own connection hidden after downgrade")
	}
	path := f.base + "/connections/" + list.Items[0].ID.String()
	var saved mcpconnect.MCPConnection
	f.human.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &saved)
	if saved.ID != list.Items[0].ID || saved.RevokedAt != nil {
		t.Fatal("status recovery did not find the owned active connection")
	}
	viewer := f.env.CreateUser(apptest.UserOpts{Email: "viewer@connect.test"})
	f.env.AddMember(f.ws, viewer.ID, "viewer")
	visitor := f.env.Client()
	visitor.SignIn(viewer.Email, "")
	visitor.Do("GET", path, nil).MustStatus(t, 404)
	visitor.Do("DELETE", path, nil).MustStatus(t, 404)
	f.human.Do("DELETE", f.base+"/connections/"+list.Items[0].ID.String(), nil).MustStatus(t, 204)
	f.human.Do("DELETE", f.base+"/connections/"+list.Items[0].ID.String(), nil).MustStatus(t, 204)
	f.human.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &saved)
	if saved.RevokedAt == nil || saved.RevokedReason != "human_revoked" {
		t.Fatal("status recovery did not confirm the saved revocation")
	}
	f.exchangeAt(f.env.Server.URL, c, f.refreshValues(tokens.RefreshToken)).MustStatus(t, 400)
}

func TestOAuthExpiryAndRetention(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	c := f.client("none")
	v, _ := f.authorization(c)
	handle := f.pending(v)
	if _, err := f.env.App.Pool.Exec(ctx, `UPDATE mcp_connect_requests SET expires_at=now()-interval '1 second' WHERE handle_hash=$1`, authn.HashToken(handle)); err != nil {
		t.Fatal(err)
	}
	f.human.Do("GET", "/mcp-connect/consent?request="+handle, nil).MustStatus(t, 409)
	code, verifier := f.allow(c)
	if _, err := f.env.App.Pool.Exec(ctx, `UPDATE mcp_connect_codes SET expires_at=now()-interval '1 second' WHERE hash=$1`, authn.HashToken(code)); err != nil {
		t.Fatal(err)
	}
	f.exchangeAt(f.env.Server.URL, c, f.codeValues(c, code, verifier)).MustStatus(t, 400)
	code, verifier = f.allow(c)
	tokens := f.exchange(c, code, verifier)
	if _, err := f.env.App.Pool.Exec(ctx, `UPDATE mcp_connect_tokens SET expires_at=now()-interval '1 second' WHERE hash=$1`, authn.HashToken(tokens.AccessToken)); err != nil {
		t.Fatal(err)
	}
	f.deniedToken(tokens.AccessToken)
	var rotated mcpconnect.TokenResponse
	f.exchangeAt(f.env.Server.URL, c, f.refreshValues(tokens.RefreshToken)).MustStatus(t, 200).JSON(t, &rotated)
	// Cleanup must preserve the used refresh hash: removing it would hide reuse.
	s := &mcpconnect.Service{Q: f.env.App.Q}
	if err := s.Purge(ctx); err != nil {
		t.Fatal(err)
	}
	f.exchangeAt(f.env.Server.URL, c, f.refreshValues(tokens.RefreshToken)).MustStatus(t, 400)
	f.deniedToken(rotated.AccessToken)
	code, verifier = f.allow(c)
	tokens = f.exchange(c, code, verifier)
	if _, err := f.env.App.Pool.Exec(ctx, `UPDATE mcp_connect_grants SET expires_at=now()-interval '1 second' WHERE id=(SELECT grant_id FROM mcp_connect_tokens WHERE hash=$1)`, authn.HashToken(tokens.AccessToken)); err != nil {
		t.Fatal(err)
	}
	f.deniedToken(tokens.AccessToken)
	f.exchangeAt(f.env.Server.URL, c, f.refreshValues(tokens.RefreshToken)).MustStatus(t, 400)
	// Every granted family and unverified client is gone beyond the recovery period.
	s.Now = func() time.Time { return time.Now().Add(38 * 24 * time.Hour) }
	if err := s.Purge(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM mcp_connect_grants)+(SELECT count(*) FROM mcp_connect_codes)+(SELECT count(*) FROM mcp_connect_tokens)+(SELECT count(*) FROM mcp_connect_requests)+(SELECT count(*) FROM mcp_connect_clients)`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("expired rows remain: %d %v", n, err)
	}
}

func TestOAuthMFAAndFrozenScope(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	c := f.client("none")
	v, _ := f.authorization(c)
	handle := f.pending(v)
	if _, err := f.env.App.Pool.Exec(ctx, `UPDATE instance_settings SET twofa_enforcement='everyone'`); err != nil {
		t.Fatal(err)
	}
	f.env.App.InvalidatePolicy()
	f.human.Do("GET", "/mcp-connect/consent?request="+handle, nil).MustStatus(t, 200)
	f.human.Do("POST", f.base+"/consent", f.consentBody(handle)).MustStatus(t, 403)
	f.human.Do("POST", "/mcp-connect/consent/deny", map[string]any{"request": handle}).MustStatus(t, 200)
	if _, err := f.env.App.Pool.Exec(ctx, `UPDATE instance_settings SET twofa_enforcement='off'`); err != nil {
		t.Fatal(err)
	}
	f.env.App.InvalidatePolicy()
	role := uuid.Must(uuid.NewV7())
	if _, err := f.env.App.Pool.Exec(ctx, `INSERT INTO roles(id,workspace_id,key,name,builtin,permissions) VALUES($1,$2,'reader','Editorial reader',false,ARRAY['mcp:manage','campaigns:view'])`, role, f.ws.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.env.App.Pool.Exec(ctx, `UPDATE workspace_members SET role_id=$1 WHERE workspace_id=$2 AND user_id=$3`, role, f.ws.ID, f.owner.ID); err != nil {
		t.Fatal(err)
	}
	v, verifier := f.authorization(c)
	handle = f.pending(v)
	f.human.Do("GET", "/mcp-connect/consent?request="+handle, nil).MustStatus(t, 200)
	f.human.Do("POST", f.base+"/consent", f.consentBody(handle)).MustStatus(t, 422)
	body := f.consentBody(handle)
	body.Scopes = []string{"editorial:read"}
	var out mcpconnect.MCPConsentResult
	f.human.Do("POST", f.base+"/consent", body).MustStatus(t, 200).JSON(t, &out)
	u, _ := url.Parse(out.Redirect)
	tokens := f.exchange(c, u.Query().Get("code"), verifier)
	status := func(want bool) {
		t.Helper()
		cs := f.mcp(tokens.AccessToken)
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "connection_status", Arguments: map[string]any{}})
		if err != nil || res.IsError {
			t.Fatal("status failed", err)
		}
		b, _ := json.Marshal(res.StructuredContent)
		var data struct {
			Scopes []string `json:"scopes"`
		}
		if err = json.Unmarshal(b, &data); err != nil || slices.Contains(data.Scopes, "editorial:read") != want {
			t.Fatalf("current capabilities %s %v", b, err)
		}
		_ = cs.Close()
	}
	status(true)
	// Current authority can shrink, and a later role upgrade cannot add a
	// different resource's permission which the frozen grant never held.
	if _, err := f.env.App.Pool.Exec(ctx, `UPDATE roles SET permissions=ARRAY['mcp:manage'] WHERE id=$1`, role); err != nil {
		t.Fatal(err)
	}
	status(false)
	if _, err := f.env.App.Pool.Exec(ctx, `UPDATE roles SET permissions=ARRAY['mcp:manage','templates:view'] WHERE id=$1`, role); err != nil {
		t.Fatal(err)
	}
	status(false)
	if _, err := f.env.App.Pool.Exec(ctx, `UPDATE workspace_members SET status='deactivated' WHERE workspace_id=$1 AND user_id=$2`, f.ws.ID, f.owner.ID); err != nil {
		t.Fatal(err)
	}
	f.deniedToken(tokens.AccessToken)
}

func TestOAuthRegistrationAndAbuse(t *testing.T) {
	f := newFixture(t)
	// Unknown metadata is discarded, not fetched. A missing auth method has
	// RFC 7591's confidential-basic default, never silent public-client auth.
	var c mcpconnect.RegisteredClient
	request(t, f.env.Server.URL, "POST", "/oauth/mcp/register", "application/json", `{"redirect_uris":["https://client.example/callback"],"logo_uri":"http://127.0.0.1:1/private","unknown":{"secret":"do-not-store"}}`, nil).MustStatus(t, 201).JSON(t, &c)
	if c.AuthMethod != "client_secret_basic" || c.Secret == "" || c.Name != "Unnamed client" {
		t.Fatal("registration defaults")
	}
	for _, raw := range []string{
		`{"redirect_uris":["https://client.example/callback"],"redirect_\u0075ris":["https://evil.example/callback"]}`,
		`{"redirect_uris":["https://client.example/callback"],"REDIRECT_URIS":["https://evil.example/callback"]}`,
		`{"redirect_uris":["https://user:pass@client.example/callback"]}`,
		`{"redirect_uris":["https://client.example/callback#fragment"]}`,
		`{"redirect_uris":["https://*.example/callback"]}`,
		`{"redirect_uris":["http://client.example/callback"]}`,
		`{"redirect_uris":["https://client.example/%5c%0dcallback"]}`,
		`{"redirect_uris":["https://client.example/callback?code=attacker"]}`,
	} {
		request(t, f.env.Server.URL, "POST", "/oauth/mcp/register", "application/json", raw, nil).MustStatus(t, 400)
	}
	another := f.client("none")
	for range 3 {
		r := request(t, f.env.Server.URL, "POST", "/oauth/mcp/register", "application/json", `{"redirect_uris":["https://client.example/callback"]}`, nil).MustStatus(t, 429)
		if r.Header.Get("Retry-After") == "" {
			t.Fatal("rate limit lacks retry delay")
		}
	}
	var n int
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM mcp_connect_clients`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("registration flood created rows: %d %v", n, err)
	}
	v, _ := f.authorization(another)
	for range 30 {
		f.pending(v)
	}
	request(t, f.env.Server.URL, "GET", "/oauth/mcp/authorize?"+v.Encode(), "", "", nil).MustStatus(t, 429)
}

// A flood from one network is refused by that network's own quota and must
// not use up the global one: another network can still register a client.
func TestOAuthQuotaIsolatesNetworks(t *testing.T) {
	f := newFixture(t)
	// A fixed clock keeps every request in one ten-minute quota window.
	now := time.Now()
	s := &mcpconnect.Service{Pool: f.env.App.Pool, Q: f.env.App.Q, Audit: f.env.App.Audit, Development: true, Now: func() time.Time { return now },
		PublicURL: func() string { return f.env.Server.URL }}
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.RemoteAddr = r.Header.Get("X-Test-Peer") + ":4000"
			next.ServeHTTP(w, r)
		})
	})
	router.Use(server.AuditContext)
	s.RegisterPublic(router)
	srv := httptest.NewServer(router)
	defer srv.Close()
	register := func(peer string) int {
		t.Helper()
		return request(t, srv.URL, "POST", "/oauth/mcp/register", "application/json", `{"redirect_uris":["https://client.example/callback"]}`, http.Header{"X-Test-Peer": {peer}}).Status
	}
	// The global registration quota is 1,000 per ten minutes.
	for i := range 1000 {
		if got := register("203.0.113.10"); (i < 10 && got != 201) || (i >= 10 && got != 429) {
			t.Fatalf("flood request %d: %d", i, got)
		}
	}
	if got := register("198.51.100.20"); got != 201 {
		t.Fatalf("another network was refused after one network's flood: %d", got)
	}
}

func TestOAuthNetworkScopeAndRevocation(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	c := f.client("none")
	v, verifier := f.authorization(c)
	handle := f.pending(v)
	f.human.Do("GET", "/mcp-connect/consent?request="+handle, nil).MustStatus(t, 200)
	body := f.consentBody(handle)
	body.AllowedIPs = []string{"203.0.113.7/32"}
	var out mcpconnect.MCPConsentResult
	f.human.Do("POST", f.base+"/consent", body).MustStatus(t, 200).JSON(t, &out)
	u, _ := url.Parse(out.Redirect)
	// The human browser's IP does not constrain consent; token use does.
	f.exchangeAt(f.env.Server.URL, c, f.codeValues(c, u.Query().Get("code"), verifier)).MustStatus(t, 400)
	code, verifier := f.allow(c)
	tokens := f.exchange(c, code, verifier)
	var foreign mcpconnect.RegisteredClient
	raw := `{"client_name":"Native editor","redirect_uris":["http://127.0.0.1:1234/callback"],"token_endpoint_auth_method":"none"}`
	request(t, f.env.Server.URL, "POST", "/oauth/mcp/register", "application/json", raw, nil).MustStatus(t, 201).JSON(t, &foreign)
	v, _ = f.authorization(foreign)
	v.Set("redirect_uri", "http://127.0.0.1:5678/callback")
	f.pending(v)
	v.Set("redirect_uri", "http://127.0.0.1:5678/other")
	request(t, f.env.Server.URL, "GET", "/oauth/mcp/authorize?"+v.Encode(), "", "", nil).MustStatus(t, 400)
	revoke := func(client, token string) {
		t.Helper()
		request(t, f.env.Server.URL, "POST", "/oauth/mcp/revoke", "application/x-www-form-urlencoded", url.Values{"client_id": {client}, "token": {token}}.Encode(), nil).MustStatus(t, 200)
	}
	revoke(foreign.ID, tokens.RefreshToken)
	revoke(foreign.ID, "unknown-token")
	f.mcp(tokens.AccessToken)
	revoke(c.ID, tokens.RefreshToken)
	revoke(c.ID, tokens.RefreshToken)
	f.deniedToken(tokens.AccessToken)
	var actor, reason string
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT actor_type,metadata->>'reason' FROM audit_events WHERE action='mcp_connect.revoked'`).Scan(&actor, &reason); err != nil || actor != "mcp_agent" || reason != "client_revoked" {
		t.Fatalf("revocation audit: %s %s %v", actor, reason, err)
	}
}

func TestOAuthParallelAdmissionAndRedaction(t *testing.T) {
	f := newFixture(t)
	c := f.client("none")
	code, verifier := f.allow(c)
	tokens := f.exchange(c, code, verifier)
	second := httptest.NewServer(f.env.App.Handler())
	defer second.Close()
	// A real, incomplete HTTP body holds one lease. Another replica must
	// reject immediately, and disconnecting releases the original lease.
	reader, writer := io.Pipe()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, f.env.Server.URL+mcpconnect.Path, reader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header = http.Header{"Authorization": {"Bearer " + tokens.AccessToken}, "Content-Type": {"application/json"}, "Accept": {"application/json, text/event-stream"}}
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	t.Cleanup(func() { _ = writer.Close(); <-finished })
	if _, err = writer.Write([]byte(`{"jsonrpc":"2.0",`)); err != nil {
		t.Fatal(err)
	}
	var count int
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err = f.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM mcp_connect_slots`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if count != 1 {
		t.Fatal("request did not acquire a lease")
	}
	r := request(t, second.URL, "POST", mcpconnect.Path, "application/json", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, http.Header{"Authorization": {"Bearer " + tokens.AccessToken}, "Accept": {"application/json, text/event-stream"}}).MustStatus(t, 429)
	if r.Header.Get("Retry-After") != "1" {
		t.Fatal("concurrency rejection lacked retry delay")
	}
	_ = writer.Close()
	<-finished
	f.mcp(tokens.AccessToken)
	// A running tool call holds the connection's slot and locks its grant.
	// A parallel request must get the slot's 429, not wait on the lock and
	// answer 401, which would send the client to refresh its token.
	ctx := context.Background()
	var grant uuid.UUID
	if err = f.env.App.Pool.QueryRow(ctx, `SELECT grant_id FROM mcp_connect_tokens WHERE hash=$1`, authn.HashToken(tokens.AccessToken)).Scan(&grant); err != nil {
		t.Fatal(err)
	}
	if _, err = f.env.App.Pool.Exec(ctx, `INSERT INTO mcp_connect_slots (grant_id, lease_id, expires_at) VALUES ($1, $2, now() + interval '30 seconds')`, grant, uuid.Must(uuid.NewV7())); err != nil {
		t.Fatal(err)
	}
	held, err := f.env.App.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = held.Exec(ctx, `SELECT 1 FROM mcp_connect_grants WHERE id=$1 FOR UPDATE`, grant); err != nil {
		t.Fatal(err)
	}
	parallel := request(t, f.env.Server.URL, "POST", mcpconnect.Path, "application/json", `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, http.Header{"Authorization": {"Bearer " + tokens.AccessToken}, "Accept": {"application/json, text/event-stream"}})
	_ = held.Rollback(ctx)
	if _, err = f.env.App.Pool.Exec(ctx, `DELETE FROM mcp_connect_slots WHERE grant_id=$1`, grant); err != nil {
		t.Fatal(err)
	}
	if parallel.Status != http.StatusTooManyRequests {
		t.Fatalf("a parallel request during a tool call answered %d", parallel.Status)
	}
	f.mcp(tokens.AccessToken)
	// Malformed human API input must not echo an opaque consent handle.
	sentinel := "PRIVATE-CONSENT-SENTINEL-" + strings.Repeat("x", 110)
	r = f.human.Do("GET", "/mcp-connect/consent?request="+sentinel, nil).MustStatus(t, 422)
	if strings.Contains(string(r.Body), sentinel) {
		t.Fatal("consent validation leaked supplied handle")
	}
	for _, r := range []*apptest.Response{r,
		request(t, f.env.Server.URL, "GET", "/mcp-consent?request="+sentinel, "", "", nil).MustStatus(t, 200),
		f.human.Do("GET", f.base+"/connections", nil).MustStatus(t, 200),
	} {
		if r.Header.Get("Cache-Control") != "no-store" || r.Header.Get("Referrer-Policy") != "no-referrer" {
			t.Fatal("sensitive response can be cached or leaked in a referrer")
		}
	}
}

func TestOAuthConsentOptionsAndMFA(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	var setup accounts.TOTPSetup
	f.human.Do("POST", "/me/totp", nil).MustStatus(t, 200).JSON(t, &setup)
	code, err := totp.GenerateCode(strings.ReplaceAll(setup.Secret, " ", ""), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	f.human.Do("POST", "/me/totp/confirm", map[string]string{"challenge": setup.Challenge, "code": code}).MustStatus(t, 200)
	if _, err = f.env.App.Pool.Exec(ctx, `UPDATE instance_settings SET twofa_enforcement='everyone'`); err != nil {
		t.Fatal(err)
	}
	f.env.App.InvalidatePolicy()
	client := f.client("none")
	v, verifier := f.authorization(client)
	handle := f.pending(v)
	f.human.Do("GET", "/mcp-connect/consent?request="+handle, nil).MustStatus(t, 200)
	// Having an authenticator is not the same as verifying this session.
	f.human.Do("POST", f.base+"/consent", f.consentBody(handle)).MustStatus(t, 403)
	var login accounts.SignInResult
	// The fixture's SignIn helper submits the real password endpoint but does
	// not complete the second-factor challenge. Read its result explicitly.
	password := "OAuth session verification password 72!"
	hash, err := authn.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.env.App.Q.SetUserPassword(ctx, store.SetUserPasswordParams{ID: f.owner.ID, PasswordHash: &hash}); err != nil {
		t.Fatal(err)
	}
	f.human = f.env.Client()
	f.human.Do("POST", "/auth/password", map[string]string{"email": f.owner.Email, "password": password}).MustStatus(t, 200).JSON(t, &login)
	if login.Status != "two_factor" {
		t.Fatal("password skipped the required second factor")
	}
	// The setup code is consumed, so use the next accepted TOTP time step.
	code, err = totp.GenerateCode(strings.ReplaceAll(setup.Secret, " ", ""), time.Now().Add(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	f.human.Do("POST", "/auth/two-factor", map[string]string{"challenge": login.Challenge, "method": "totp", "code": code}).MustStatus(t, 200)
	// The old request stays bound to the old session; reconnect explicitly.
	f.human.Do("GET", "/mcp-connect/consent?request="+handle, nil).MustStatus(t, 409)
	handle = f.pending(v)
	f.human.Do("GET", "/mcp-connect/consent?request="+handle, nil).MustStatus(t, 200)
	var options mcpconnect.MCPConsentOptions
	f.human.Do("GET", f.base+"/options", nil).MustStatus(t, 200).JSON(t, &options)
	if !slices.Equal(options.Scopes, f.consentBody(handle).Scopes) || options.Endpoint != f.env.Server.URL+mcpconnect.Path {
		t.Fatal("consent options lost the owner's capabilities or configured resource")
	}
	var result mcpconnect.MCPConsentResult
	f.human.Do("POST", f.base+"/consent", f.consentBody(handle)).MustStatus(t, 200).JSON(t, &result)
	redirect, _ := url.Parse(result.Redirect)
	f.mcp(f.exchange(client, redirect.Query().Get("code"), verifier).AccessToken)
}

func TestOAuthScopedConsentOptions(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	allowed, hidden := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	for id, name := range map[uuid.UUID]string{allowed: "Weekly", hidden: "Private list sentinel"} {
		if _, err := f.env.App.Pool.Exec(ctx, `INSERT INTO lists(id,workspace_id,name) VALUES($1,$2,$3)`, id, f.ws.ID, name); err != nil {
			t.Fatal(err)
		}
	}
	for _, domain := range []string{"news.example", "private.example"} {
		if _, err := f.env.App.Pool.Exec(ctx, `INSERT INTO sending_domains(id,workspace_id,domain) VALUES($1,$2,$3)`, uuid.Must(uuid.NewV7()), f.ws.ID, domain); err != nil {
			t.Fatal(err)
		}
	}
	scope, _ := json.Marshal(map[string]any{"lists": []uuid.UUID{allowed}, "senderDomains": []string{"news.example"}})
	role := uuid.Must(uuid.NewV7())
	if _, err := f.env.App.Pool.Exec(ctx, `INSERT INTO roles(id,workspace_id,key,name,builtin,permissions) VALUES($1,$2,'scoped-reader','Scoped reader',false,ARRAY['mcp:manage','campaigns:view'])`, role, f.ws.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.env.App.Pool.Exec(ctx, `UPDATE workspace_members SET role_id=$1, scope=$2 WHERE workspace_id=$3 AND user_id=$4`, role, scope, f.ws.ID, f.owner.ID); err != nil {
		t.Fatal(err)
	}
	var options mcpconnect.MCPConsentOptions
	response := f.human.Do("GET", f.base+"/options", nil).MustStatus(t, 200)
	response.JSON(t, &options)
	if len(options.Lists) != 1 || options.Lists[0].ID != allowed || !slices.Equal(options.DefaultLists, []uuid.UUID{allowed}) ||
		!slices.Equal(options.SenderDomains, []string{"news.example"}) || !slices.Equal(options.DefaultDomains, []string{"news.example"}) ||
		!slices.Equal(options.Scopes, []string{"editorial:read"}) || strings.Contains(string(response.Body), "Private list sentinel") {
		t.Fatalf("options escaped current authority: %s", response.Body)
	}
	c := f.client("none")
	v, _ := f.authorization(c)
	handle := f.pending(v)
	f.human.Do("GET", "/mcp-connect/consent?request="+handle, nil).MustStatus(t, 200)
	body := f.consentBody(handle)
	body.Scopes = options.Scopes
	// Empty is unrestricted, not an instruction to infer the user's scope.
	f.human.Do("POST", f.base+"/consent", body).MustStatus(t, 422)
	body.Lists, body.SenderDomains = options.DefaultLists, options.DefaultDomains
	f.human.Do("POST", f.base+"/consent", body).MustStatus(t, 200)
	var grants mcpconnect.MCPConnectionList
	f.human.Do("GET", f.base+"/connections", nil).MustStatus(t, 200).JSON(t, &grants)
	if len(grants.Items) != 1 || !slices.Equal(grants.Items[0].Lists, []uuid.UUID{allowed}) || !slices.Equal(grants.Items[0].SenderDomains, []string{"news.example"}) {
		t.Fatal("grant did not preserve the explicitly chosen scope")
	}
}
