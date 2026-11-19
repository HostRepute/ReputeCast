package mcpclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/mcpclient"
	"github.com/hostrepute/reputecast/internal/mcpclient/mcptest"
)

// https serves the test's app on https too and makes that its public URL
// (an authorization server fetches the client metadata document there).
func (f *fx) https() string {
	f.t.Helper()
	srv := httptest.NewTLSServer(f.env.Server.Config.Handler)
	f.t.Cleanup(srv.Close)
	f.env.App.PublicURL.Set(srv.URL)
	return srv.URL
}

// Without a client entered, an authorization server that takes Client ID
// Metadata Documents gets the instance's: its https address is the
// client_id, and the server fetches it (public, cached briefly, JSON, the
// exact redirect URI, a public client). One that doesn't falls back to
// dynamic registration, and so does an instance whose public URL isn't
// https (which serves no document); without registration either, a
// manager has to enter a client.
func TestMCPClientMetadataDocument(t *testing.T) {
	cases := []struct {
		name          string
		https         bool
		opts          mcptest.AuthOptions
		document      bool
		registrations int32
		problem       string
	}{
		{name: "supported", https: true, opts: mcptest.AuthOptions{CIMD: true}, document: true},
		{name: "unsupported", https: true, registrations: 1},
		{name: "http public URL", opts: mcptest.AuthOptions{CIMD: true}, registrations: 1},
		{name: "no client", https: true, opts: mcptest.AuthOptions{NoRegistration: true}, problem: "mcp_oauth_client"},
		{name: "no client over http", opts: mcptest.AuthOptions{CIMD: true, NoRegistration: true}, problem: "mcp_oauth_client"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFx(t)
			public := f.env.App.PublicURL.Get()
			if c.https {
				public = f.https()
			}
			doc := public + "/api/v1/ai/mcp/oauth/client-metadata.json"
			callback := public + "/api/v1/ai/mcp/oauth/callback"
			// The document, without signing in.
			r := f.env.Client().Do(http.MethodGet, "/ai/mcp/oauth/client-metadata.json", nil)
			if !c.https {
				r.MustStatus(t, 404)
			} else {
				var got mcpclient.MCPClientMetadataDocument
				r.MustStatus(t, 200).JSON(t, &got)
				if got.ClientID != doc || got.ClientName == "" || len(got.RedirectURIs) != 1 || got.RedirectURIs[0] != callback ||
					got.TokenEndpointAuthMethod != "none" || strings.Join(got.GrantTypes, " ") != "authorization_code refresh_token" ||
					r.Header.Get("Cache-Control") != "public, max-age=300" || r.Header.Get("Content-Type") != "application/json" {
					t.Fatalf("the document: %+v %v", got, r.Header)
				}
			}

			fake := mcptest.NewServer(t, "", "")
			fake.Tool("search", "Search", true, mcptest.Text("hit"))
			auth := mcptest.NewAuthWith(t, fake, c.opts)
			s := f.create(map[string]any{"name": "Docs", "url": fake.Endpoint(), "auth": "oauth"})
			if c.problem != "" {
				if p := f.do(f.oc, http.MethodPost, f.base+"/"+s.ID.String()+"/oauth", nil, 422).Problem(t); p["code"] != c.problem {
					t.Fatalf("without a client: %v", p)
				}
				return
			}
			if loc := f.signIn(f.oc, auth, s.ID, "owner").MustStatus(t, 303).Header.Get("Location"); !strings.HasSuffix(loc, "&connected=1") {
				t.Fatalf("back to %s", loc)
			}
			docs := auth.Documents()
			if c.document != (len(docs) == 1) || auth.Registrations.Load() != c.registrations {
				t.Fatalf("%d documents fetched, %d registrations", len(docs), auth.Registrations.Load())
			}
			if c.document && (docs[0].Fields["client_id"] != doc || docs[0].ContentType != "application/json" || docs[0].CacheControl != "public, max-age=300") {
				t.Fatalf("the document fetched: %+v", docs[0])
			}
			var cur mcpclient.RemoteMCPServerList
			f.do(f.oc, http.MethodGet, f.base, nil, 200).JSON(t, &cur)
			if v := cur.Servers[0]; v.ClientMetadata != c.document || v.ClientRegistered == c.document || (c.document && v.ClientID != doc) {
				t.Fatalf("the server's client: %+v", v)
			}
			s = f.decide(f.sync(f.oc, s.ID), "search", "read_only")
			if r, err := f.callAs(f.svc(true), f.owner, s, "search"); err != nil || r.Text != "hit" {
				t.Fatalf("a call with the token: %q %v", r.Text, err)
			}
		})
	}
}

// A client with a secret authenticates at the token endpoint, for the
// code and for refreshes, and at the revocation endpoint (on disconnect,
// for a member removed and for a deleted server, from the queue) as its
// authorization server lists: client_secret_basic when offered, else
// client_secret_post, or what its dynamic registration returned; a public
// client sends none. A server that lists only methods ReputeCast doesn't
// support (private_key_jwt) is refused with a problem that says so, and so
// is a secret entered for one.
func TestMCPClientAuthentication(t *testing.T) {
	cases := []struct {
		name     string
		opts     mcptest.AuthOptions
		secret   bool
		want     string
		everyway bool
	}{
		{name: "basic preferred", opts: mcptest.AuthOptions{Methods: []string{"client_secret_post", "client_secret_basic"}}, secret: true, want: "client_secret_basic"},
		{name: "post", opts: mcptest.AuthOptions{Methods: []string{"client_secret_post"}}, secret: true, want: "client_secret_post", everyway: true},
		{name: "registered post", opts: mcptest.AuthOptions{Methods: []string{"client_secret_basic", "client_secret_post"}, RegisterPost: true}, want: "client_secret_post"},
		{name: "public", opts: mcptest.AuthOptions{Methods: []string{"none", "client_secret_basic"}}, want: "none"},
		{name: "private_key_jwt only", opts: mcptest.AuthOptions{Methods: []string{"private_key_jwt"}}, secret: true},
	}
	const secret = "mcp-client-auth-secret-7f3e"
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFx(t)
			ctx := context.Background()
			fake := mcptest.NewServer(t, "", "")
			fake.Tool("search", "Search", true, mcptest.Text("hit"))
			auth := mcptest.NewAuthWith(t, fake, c.opts)
			in := map[string]any{"name": "Docs", "url": fake.Endpoint(), "auth": "oauth"}
			if c.secret {
				in["clientId"], in["clientSecret"] = "acme-client", secret
				auth.Preregister("acme-client", secret, f.env.App.PublicURL.Get()+"/api/v1/ai/mcp/oauth/callback")
			}
			s := f.create(in)
			if c.want == "" {
				p := f.do(f.oc, http.MethodPost, f.base+"/"+s.ID.String()+"/oauth", nil, 502).Problem(t)
				if p["code"] != "mcp_oauth_client_auth" || !strings.Contains(p["detail"].(string), "private_key_jwt") {
					t.Fatalf("a server with private_key_jwt only: %v", p)
				}
				return
			}
			f.signIn(f.oc, auth, s.ID, "owner").MustStatus(t, 303)
			s = f.decide(f.sync(f.oc, s.ID), "search", "read_only")
			// The access token expires: the call refreshes it.
			if _, err := f.env.App.Pool.Exec(ctx, `UPDATE mcp_tokens SET expires_at = now() - interval '1 minute' WHERE server_id = $1`, s.ID); err != nil {
				t.Fatal(err)
			}
			svc := f.svc(true)
			if r, err := f.callAs(svc, f.owner, s, "search"); err != nil || r.Text != "hit" || auth.Refreshes.Load() != 1 {
				t.Fatalf("a call after a refresh: %q %v (%d refreshes)", r.Text, err, auth.Refreshes.Load())
			}
			f.do(f.oc, http.MethodDelete, f.base+"/"+s.ID.String()+"/oauth", nil, 204)
			want := map[string]int{"token": 1, "refresh": 1, "revoke": 1}
			if c.everyway {
				// A member removed, then the server deleted: the queue
				// authenticates as the client did.
				f.signIn(f.ec, auth, s.ID, "editor").MustStatus(t, 303)
				f.signIn(f.cc, auth, s.ID, "colleague").MustStatus(t, 303)
				var member string
				if err := f.env.App.Pool.QueryRow(ctx, `SELECT id::text FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`, f.ws.ID, f.colleague.ID).Scan(&member); err != nil {
					t.Fatal(err)
				}
				f.do(f.oc, http.MethodDelete, "/workspaces/"+f.ws.Slug+"/members/"+member, nil, 204)
				f.do(f.oc, http.MethodDelete, f.base+"/"+s.ID.String(), nil, 204)
				if n, err := svc.Revoke(ctx, 100); err != nil || n != 2 {
					t.Fatalf("the queued revocations: %d %v", n, err)
				}
				want = map[string]int{"token": 3, "refresh": 1, "revoke": 3}
			}
			got := map[string]int{}
			for _, a := range auth.ClientAuth() {
				endpoint, method, _ := strings.Cut(a, " ")
				if method != c.want {
					t.Fatalf("authenticated as %q, want %s: %q", a, c.want, auth.ClientAuth())
				}
				got[endpoint]++
			}
			if len(got) != len(want) || got["token"] != want["token"] || got["refresh"] != want["refresh"] || got["revoke"] != want["revoke"] {
				t.Fatalf("requests %v, want %v", got, want)
			}
			if n := len(auth.Revoked()); n != want["revoke"] {
				t.Fatalf("%d revoked, want %d", n, want["revoke"])
			}
			f.noSecret(secret, "registered-secret-")
		})
	}
}

// A secret entered for a client whose authorization server lists only
// methods ReputeCast doesn't support besides none is refused.
func TestMCPClientSecretWithoutMethod(t *testing.T) {
	f := newFx(t)
	fake := mcptest.NewServer(t, "", "")
	auth := mcptest.NewAuthWith(t, fake, mcptest.AuthOptions{Methods: []string{"none", "private_key_jwt"}})
	s := f.create(map[string]any{"name": "Docs", "url": fake.Endpoint(), "auth": "oauth"})
	f.signIn(f.oc, auth, s.ID, "owner").MustStatus(t, 303)
	p := f.do(f.oc, http.MethodPatch, f.base+"/"+s.ID.String(), map[string]any{"clientId": "acme-client", "clientSecret": secretSentinel}, 422).Problem(t)
	if p["code"] != "mcp_oauth_client_auth" {
		t.Fatalf("a secret for a server without client_secret methods: %v", p)
	}
	f.noSecret(secretSentinel)
}

const secretSentinel = "mcp-client-secret-unsupported-90ab"
