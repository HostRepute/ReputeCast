package recommendations

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/media"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
)

// RecPrefillInput asks for a subscribe page's name, description and logo.
type RecPrefillInput struct {
	URL string `json:"url" maxLength:"2000"`
}

// RecPrefill is what a page says about itself (its title, description and
// image), to start a link recommendation.
type RecPrefill struct {
	URL         string    `json:"url"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Logo        *RecImage `json:"logo,omitempty"`
	// Found: the page answered and named itself.
	Found bool `json:"found"`
}

// pageMeta is a page's metadata.
type pageMeta struct {
	title, siteName, description, image string
}

// readMeta reads a page's <head>: Open Graph first, then the usual tags.
func readMeta(page []byte, base *url.URL) pageMeta {
	var m pageMeta
	var ogTitle, ogDesc, ogImage, icon, touch string
	z := xhtml.NewTokenizer(bytes.NewReader(page))
	inTitle := false
	for {
		tt := z.Next()
		if tt == xhtml.ErrorToken {
			break
		}
		switch tt {
		case xhtml.TextToken:
			if inTitle && m.title == "" {
				m.title = collapse(string(z.Text()))
			}
		case xhtml.EndTagToken:
			name, _ := z.TagName()
			if atom.Lookup(name) == atom.Title {
				inTitle = false
			}
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			a := atom.Lookup(name)
			if a == atom.Body {
				goto done
			}
			if a == atom.Title {
				inTitle = true
				continue
			}
			if !hasAttr || (a != atom.Meta && a != atom.Link) {
				continue
			}
			attrs := map[string]string{}
			for {
				k, v, more := z.TagAttr()
				attrs[strings.ToLower(string(k))] = string(v)
				if !more {
					break
				}
			}
			if a == atom.Meta {
				key := strings.ToLower(attrs["property"])
				if key == "" {
					key = strings.ToLower(attrs["name"])
				}
				switch key {
				case "og:title", "twitter:title":
					if ogTitle == "" {
						ogTitle = collapse(attrs["content"])
					}
				case "og:site_name":
					m.siteName = collapse(attrs["content"])
				case "og:description", "twitter:description":
					if ogDesc == "" {
						ogDesc = collapse(attrs["content"])
					}
				case "description":
					if m.description == "" {
						m.description = collapse(attrs["content"])
					}
				case "og:image", "og:image:url", "twitter:image":
					if ogImage == "" {
						ogImage = attrs["content"]
					}
				}
				continue
			}
			rel := " " + strings.ToLower(attrs["rel"]) + " "
			switch {
			case strings.Contains(rel, " apple-touch-icon "):
				touch = attrs["href"]
			case strings.Contains(rel, " icon "):
				if icon == "" {
					icon = attrs["href"]
				}
			}
		}
	}
done:
	if ogTitle != "" {
		m.title = ogTitle
	}
	if ogDesc != "" {
		m.description = ogDesc
	}
	for _, img := range []string{touch, ogImage, icon} {
		if u := resolve(base, img); u != "" {
			m.image = u
			break
		}
	}
	return m
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

func resolve(base *url.URL, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.HasPrefix(ref, "data:") {
		return ""
	}
	u, err := base.Parse(ref)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return ""
	}
	return u.String()
}

// fetch gets a page or an image through the SSRF guard: no redirects,
// bounded in size and time.
func (s *Service) fetch(ctx context.Context, raw string, limit int64) ([]byte, string, error) {
	if _, err := s.guard().ValidateURL(raw); err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "ReputeCast/"+version.Version+" (recommendation preview)")
	req.Header.Set("Accept", "text/html,image/*;q=0.9,*/*;q=0.5")
	resp, err := s.client(10 * time.Second).Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, "", &httpStatusError{status: resp.StatusCode}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	return body, resp.Header.Get("Content-Type"), err
}

type httpStatusError struct{ status int }

func (e *httpStatusError) Error() string { return "answered " + http.StatusText(e.status) }

// fetchImage keeps an image from the web in the media library.
func (s *Service) fetchImage(ctx context.Context, ws *store.Workspace, raw string) (uuid.UUID, bool) {
	if s.Media == nil {
		return uuid.Nil, false
	}
	data, _, err := s.fetch(ctx, raw, 2<<20)
	if err != nil || len(data) == 0 {
		return uuid.Nil, false
	}
	if _, _, _, _, err := media.Inspect(data); err != nil {
		return uuid.Nil, false
	}
	name := path.Base(strings.SplitN(raw, "?", 2)[0])
	a, err := s.Media.Upload(ctx, ws, actor(ctx), name, data)
	if err != nil {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(a.ID)
	return id, err == nil
}

var mPrefillPrivate = msgs.M("prefill.private")

// Prefill reads a subscribe page's name, description and logo, through the
// SSRF guard ("Filled in the name, description and logo from this page").
func (s *Service) Prefill(ctx context.Context, ws *store.Workspace, in RecPrefillInput) (RecPrefill, error) {
	u, err := s.guard().ValidateURL(in.URL)
	if err != nil || (u.Scheme != "https" && !s.allowPrivate()) {
		if err != nil && netguard.IsBlocked(err) {
			return RecPrefill{}, api.Invalid("body.url", s.text(ctx, mPrefillPrivate, nil))
		}
		return RecPrefill{}, api.Invalid("body.url", s.text(ctx, mURL, nil))
	}
	out := RecPrefill{URL: u.String()}
	page, ctype, err := s.fetch(ctx, u.String(), 512<<10)
	if err != nil {
		if netguard.IsBlocked(err) {
			return out, api.Invalid("body.url", s.text(ctx, mPrefillPrivate, nil))
		}
		return out, nil // the member fills it in
	}
	if ctype != "" && !strings.Contains(strings.ToLower(ctype), "html") {
		return out, nil
	}
	m := readMeta(page, u)
	out.Name = m.siteName
	if out.Name == "" {
		out.Name = m.title
	}
	if r := []rune(out.Name); len(r) > 120 {
		out.Name = string(r[:120])
	}
	out.Description = m.description
	if r := []rune(out.Description); len(r) > 120 {
		out.Description = strings.TrimSpace(string(r[:119])) + "…"
	}
	out.Found = out.Name != "" || out.Description != ""
	if m.image != "" {
		if id, ok := s.fetchImage(ctx, ws, m.image); ok {
			out.Logo = s.image(ctx, ws, &id)
		}
	}
	return out, nil
}
