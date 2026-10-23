package server

import (
	"bytes"
	"errors"
	"html"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// SPA serves the embedded single-page app. Hashed files under /assets/ are
// cached for a year; everything else falls back to index.html, which is
// never cached so new releases load immediately. Pre-compressed .br and .gz
// siblings produced by the build are served when the client accepts them.
type SPA struct {
	fsys fs.FS
	// index is index.html in each app language: its line for browsers
	// without JavaScript follows the request's Accept-Language.
	index map[i18n.Locale][]byte
	start time.Time
}

// noscript is index.html's line for browsers without JavaScript, as the
// build writes it.
const noscript = "<noscript>ReputeCast needs JavaScript. Public pages (subscribe, preferences, archive) work without it.</noscript>"

// NewSPA returns a handler for the built frontend in fsys (the dist folder).
// When the frontend hasn't been built, it serves a short explanation instead.
func NewSPA(fsys fs.FS) *SPA {
	idx, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		idx = []byte(`<!doctype html><html lang="en"><meta charset="utf-8"><title>ReputeCast</title>` +
			`<body style="font-family:system-ui;padding:40px"><h1>ReputeCast</h1>` +
			`<p>The web app isn't built into this binary. Run <code>make build</code>, or ` +
			`<code>make dev</code> and open the Vite dev server.</p></body></html>`)
	}
	index := map[i18n.Locale][]byte{}
	for _, l := range i18n.Locales() {
		line := "<noscript>" + html.EscapeString(msgs.M("spa.noscript").Text(i18n.For(l), nil)) + "</noscript>"
		index[l] = bytes.Replace(idx, []byte(noscript), []byte(line), 1)
	}
	return &SPA{fsys: fsys, index: index, start: time.Now()}
}

func (s *SPA) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name != "" && name != "index.html" {
		if s.serveFile(w, r, name) {
			return
		}
		// Unknown files with an extension under /assets are real 404s, not routes.
		if strings.HasPrefix(name, "assets/") {
			http.NotFound(w, r)
			return
		}
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	h.Add("Vary", "Accept-Language")
	// The app is private; public pages and the landing page are indexable.
	h.Set("X-Robots-Tag", "noindex, nofollow")
	idx, ok := s.index[requestLang(r).Locale]
	if !ok {
		idx = s.index[i18n.English]
	}
	http.ServeContent(w, r, "index.html", s.start, bytes.NewReader(idx))
}

func (s *SPA) serveFile(w http.ResponseWriter, r *http.Request, name string) bool {
	info, err := fs.Stat(s.fsys, name)
	if err != nil || info.IsDir() {
		return false
	}
	h := w.Header()
	if strings.HasPrefix(name, "assets/") {
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		h.Set("Cache-Control", "public, max-age=3600")
	}
	ctype := mime.TypeByExtension(path.Ext(name))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	h.Set("Content-Type", ctype)
	h.Add("Vary", "Accept-Encoding")
	accept := r.Header.Get("Accept-Encoding")
	for _, enc := range []struct{ ext, name string }{{".br", "br"}, {".gz", "gzip"}} {
		if !strings.Contains(accept, enc.name) {
			continue
		}
		if f, err := s.fsys.Open(name + enc.ext); err == nil {
			defer func() { _ = f.Close() }()
			if rs, ok := f.(io.ReadSeeker); ok {
				h.Set("Content-Encoding", enc.name)
				http.ServeContent(w, r, name, s.start, rs)
				return true
			}
		}
	}
	f, err := s.fsys.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		b, err := io.ReadAll(f)
		if err != nil {
			return false
		}
		rs = bytes.NewReader(b)
	}
	http.ServeContent(w, r, name, s.start, rs)
	return true
}

// ErrNoFrontend is returned by helpers when the dist folder is missing.
var ErrNoFrontend = errors.New("frontend not built")
