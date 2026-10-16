package growth

import (
	"context"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// Forms, popups, hosted pages and the subscribe page send where a signup
// happened: the page's address (its UTM parameters) and the page the
// visitor came from. They are saved with the signup's consent records;
// the contact's first touch is what Growth counts.

// Touch is where one signup came from.
type Touch struct {
	UTMSource, UTMMedium, UTMCampaign, UTMContent, UTMTerm string
	// Page is the page with the form and Referrer the page before it,
	// both without their query string or fragment ("acme.io/pricing").
	Page, Referrer string
}

// Empty reports whether the touch says nothing.
func (t Touch) Empty() bool { return t == Touch{} }

const maxValue = 200

func clean(s string, n int) string {
	s = strings.TrimSpace(s)
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// PageOf is an address without its scheme, query string or fragment
// ("https://acme.io/pricing?plan=team#top" → "acme.io/pricing"). It
// returns "" for anything that isn't an http(s) address.
func PageOf(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	if p := u.Port(); p != "" && p != "80" && p != "443" {
		host += ":" + p
	}
	path := strings.TrimSuffix(u.EscapedPath(), "/")
	return clean(host+path, 500)
}

// DomainOf is a page's host ("blog.acme.io/2026/10/x" → "blog.acme.io").
func DomainOf(page string) string {
	host, _, _ := strings.Cut(page, "/")
	host, _, _ = strings.Cut(host, ":")
	return host
}

// TouchOf reads a touch from the page's address and the referrer the
// browser reported.
func TouchOf(pageURL, referrerURL string) Touch {
	var t Touch
	if u, err := url.Parse(strings.TrimSpace(pageURL)); err == nil {
		q := u.Query()
		t.UTMSource = clean(q.Get("utm_source"), maxValue)
		t.UTMMedium = clean(q.Get("utm_medium"), maxValue)
		t.UTMCampaign = clean(q.Get("utm_campaign"), maxValue)
		t.UTMContent = clean(q.Get("utm_content"), maxValue)
		t.UTMTerm = clean(q.Get("utm_term"), maxValue)
	}
	t.Page = PageOf(pageURL)
	t.Referrer = PageOf(referrerURL)
	return t
}

// SaveTouch stores a touch on the consent records a signup wrote at a
// moment (inside the signup's transaction).
func SaveTouch(ctx context.Context, q *store.Queries, contact uuid.UUID, at time.Time, t Touch) error {
	if t.Empty() {
		return nil
	}
	return q.GrowthSaveTouch(ctx, store.GrowthSaveTouchParams{ContactID: contact, At: at,
		UtmSource: t.UTMSource, UtmMedium: t.UTMMedium, UtmCampaign: t.UTMCampaign, UtmContent: t.UTMContent, UtmTerm: t.UTMTerm,
		Page: t.Page, Referrer: t.Referrer})
}
