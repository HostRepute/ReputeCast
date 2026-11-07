package app

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/server"
)

// secretParams are the path parameters that carry a bearer secret: a link
// token, a push click, a short link's code.
var secretParams = map[string]bool{"token": true, "click": true, "code": true}

// publicCodes are routes whose {code} is no secret, and why.
var publicCodes = map[string]string{
	"/p/{ws}/join/{code}":                                      "a referral code its owner shares in public",
	"/api/v1/workspaces/{workspace}/languages/{code}":          "a language code",
	"/api/v1/workspaces/{workspace}/languages/{code}/removal":  "a language code",
	"/api/v1/workspaces/{workspace}/grow/partner-codes/{code}": "a partner code's id, behind sign-in",
}

// TestTokenRoutesAreRedacted reads every route the server mounts (the API,
// public pages, tracking, short links, provider and store webhooks) and
// every route of the web app from the code. A secret
// in a path ({token}, {click}, {code}) must never reach the request log, at
// /p/{workspace}/… and at a custom domain's root alike; a path with nothing
// secret must reach it as it is. A new route with a secret the redaction
// doesn't know fails here.
func TestTokenRoutesAreRedacted(t *testing.T) {
	r := chi.NewRouter()
	a := &App{}
	registerAPI(a, humachi.New(r, NewHumaConfig()))
	registerPublic(a, r)
	registerIngest(a, r)
	routes := map[string]bool{"/healthz": true, "/readyz": true} // mounted by Handler
	err := chi.Walk(r, func(_, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		routes[route] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range webRoutes(t) {
		routes[route] = true
	}
	// A custom domain answers the workspace's public pages at its root.
	for route := range routes {
		if rest, ok := strings.CutPrefix(route, "/p/{ws}"); ok && rest != "" && publicCodes[route] == "" && servedAtRoot(routes, rest) {
			routes[rest] = true
		}
	}
	for route := range publicCodes {
		if !routes[route] {
			t.Errorf("publicCodes names %s, which no longer exists", route)
		}
	}

	all := make([]string, 0, len(routes))
	for route := range routes {
		all = append(all, route)
	}
	sort.Strings(all)
	checked := 0
	for _, route := range all {
		path, secrets := sample(t, route)
		got := server.RedactPath(path)
		if !strings.ContainsAny(route, "{*") && got != path {
			t.Errorf("%s: logged as %s, but nothing in it is secret", route, got)
		}
		for _, s := range secrets {
			checked++
			if strings.Contains(got, s) {
				t.Errorf("%s: %s is logged as %s, with its secret", route, path, got)
			}
		}
	}
	if checked < 40 {
		t.Fatalf("only %d secrets checked: the routes weren't all found", checked)
	}
}

var param = regexp.MustCompile(`\{([a-zA-Z]+)(?::([^{}]*(?:\{[0-9,]+\}[^{}]*)*))?\}`)

// sample fills a route's parameters: each secret with a value of its own
// (matching the route's pattern), the rest with a plain slug. It returns
// the path and the secrets in it.
func sample(t *testing.T, route string) (string, []string) {
	t.Helper()
	var secrets []string
	n := 0
	path := param.ReplaceAllStringFunc(route, func(m string) string {
		sub := param.FindStringSubmatch(m)
		name, pattern := sub[1], sub[2]
		if !secretParams[name] || publicCodes[route] != "" {
			return "acme"
		}
		n++
		v := ""
		for _, c := range []string{"Xk7Pq2", "H3RT-6QWN-8ZKB", "S3cr3tTokEn0" + string(rune('a'+n))} {
			if pattern == "" || regexp.MustCompile(`^(?:`+pattern+`)$`).MatchString(c) {
				v = c
				break
			}
		}
		if v == "" {
			t.Fatalf("%s: no sample value matches {%s:%s}", route, name, pattern)
		}
		secrets = append(secrets, v)
		return v
	})
	return strings.ReplaceAll(path, "*", "file.txt"), secrets
}

// servedAtRoot reports whether a custom domain's root answers a public
// page's path ("/confirm/{token}" through "/confirm/*").
func servedAtRoot(routes map[string]bool, path string) bool {
	for r := range routes {
		if prefix, ok := strings.CutSuffix(r, "*"); ok && !strings.Contains(r, "{") && prefix != "/" && strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// webRoutes are the web app's routes with a secret in them, from its route
// files (invite.$token.tsx is /invite/{token}).
func webRoutes(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join("..", "..", "web", "src", "routes")
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".tsx") || strings.HasPrefix(d.Name(), "-") {
			return err
		}
		rel, _ := filepath.Rel(dir, strings.TrimSuffix(path, ".tsx"))
		segs := []string{}
		for _, s := range strings.FieldsFunc(filepath.ToSlash(rel), func(r rune) bool { return r == '/' || r == '.' }) {
			switch {
			case s == "index" || s == "route" || strings.HasPrefix(s, "_"):
			case strings.HasPrefix(s, "$"):
				segs = append(segs, "{"+s[1:]+"}")
			default:
				segs = append(segs, s)
			}
		}
		route := "/" + strings.Join(segs, "/")
		if slices.ContainsFunc(segs, func(s string) bool { return secretParams[strings.Trim(s, "{}")] }) {
			out = append(out, route)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 4 {
		t.Fatalf("web routes with a secret: %v", out)
	}
	return out
}
