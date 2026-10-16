package feed

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/netguard"
)

// Limits of a read (brief 3.1): 15 seconds, 5 MB, up to 3 redirects.
const (
	Timeout      = 15 * time.Second
	MaxBytes     = 5 << 20
	MaxRedirects = 3
)

// Fetcher reads feeds, pages and images from addresses people type, through
// the same SSRF guard as importers and webhooks: every address (the first
// and each redirect's) is checked before connecting, and again at connect
// time against the resolved IP, so a name can't point at loopback,
// private, link-local or metadata addresses.
type Fetcher struct {
	Policy netguard.Policy
	// Transport replaces the guarded transport (tests, and the demo's
	// recorded sites); the address checks still run.
	Transport http.RoundTripper
	Timeout   time.Duration
	MaxBytes  int64
	UserAgent string
}

// Request is a conditional read.
type Request struct {
	URL      string
	ETag     string
	Modified string
	// Accept overrides the feed Accept header (images).
	Accept string
}

// Hop is one address answered on the way: a redirect or the final answer.
type Hop struct {
	URL      string `json:"url"`
	Status   int    `json:"status"`
	Location string `json:"location,omitempty"`
}

// Response is what answered.
type Response struct {
	// URL is the final address, after redirects.
	URL         string
	Status      int
	NotModified bool
	Body        []byte
	ContentType string
	ETag        string
	Modified    string
	Hops        []Hop
	// Moved is the target of a permanent redirect (301, 308) on the way.
	Moved    string
	Duration time.Duration
}

// Error kinds.
const (
	ErrUnreachable = "unreachable" // DNS, connection refused or reset, TLS
	ErrTimeout     = "timeout"
	ErrBlocked     = "blocked" // the SSRF guard refused the address
	ErrTooLarge    = "too_large"
	ErrStatus      = "status" // an HTTP error status
	ErrRedirects   = "redirects"
	ErrAddress     = "address" // not a usable URL
)

// FetchError explains why a read failed, in terms a person can act on.
type FetchError struct {
	Kind string
	// URL is the address that failed (the redirect target when it was one).
	URL    string
	Status int
	Detail string
	Hops   []Hop
}

func (e *FetchError) Error() string {
	if e.Detail != "" {
		return e.Detail
	}
	return e.Kind
}

// Host of the address that failed ("blog.acme.io").
func (e *FetchError) Host() string {
	if u, err := url.Parse(e.URL); err == nil && u.Host != "" {
		return u.Hostname()
	}
	return e.URL
}

// AsFetchError returns err as a *FetchError.
func AsFetchError(err error) (*FetchError, bool) {
	var fe *FetchError
	ok := errors.As(err, &fe)
	return fe, ok
}

func (f *Fetcher) timeout() time.Duration {
	if f.Timeout > 0 {
		return f.Timeout
	}
	return Timeout
}

func (f *Fetcher) maxBytes() int64 {
	if f.MaxBytes > 0 {
		return f.MaxBytes
	}
	return MaxBytes
}

func (f *Fetcher) client() *http.Client {
	c := f.Policy.Client(f.timeout())
	if f.Transport != nil {
		c.Transport = f.Transport
	}
	return c
}

// NormalizeURL turns what someone typed ("blog.acme.io", "blog.acme.io/feed/")
// into an absolute http(s) URL.
func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("type a feed or page address, like blog.acme.io/feed/")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + strings.TrimLeft(raw, "/")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || !strings.Contains(u.Hostname(), ".") && u.Hostname() != "localhost" {
		return "", errors.New("that isn't an address: type one like blog.acme.io/feed/")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", errors.New("feeds are read over https:// or http://")
	}
	u.Fragment = ""
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String(), nil
}

// Get reads an address, following up to MaxRedirects redirects, each
// target checked by the guard first. A 304 answer to a conditional request
// is NotModified with no body.
func (f *Fetcher) Get(ctx context.Context, r Request) (*Response, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, f.timeout())
	defer cancel()
	current, err := NormalizeURL(r.URL)
	if err != nil {
		return nil, &FetchError{Kind: ErrAddress, URL: r.URL, Detail: err.Error()}
	}
	client := f.client()
	out := &Response{}
	// The first address answered a permanent redirect: the final address
	// is offered as the feed's new one.
	permanent := false
	for hop := 0; ; hop++ {
		u, err := f.Policy.ValidateURL(current)
		if err != nil {
			fe := &FetchError{Kind: ErrBlocked, URL: current, Detail: blockedDetail(current, err), Hops: out.Hops}
			if !netguard.IsBlocked(err) {
				fe.Kind = ErrAddress
			}
			return nil, fe
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, &FetchError{Kind: ErrAddress, URL: current, Detail: err.Error(), Hops: out.Hops}
		}
		ua := f.UserAgent
		if ua == "" {
			ua = "ReputeCast feed reader (+https://reputecast.com)"
		}
		req.Header.Set("User-Agent", ua)
		accept := r.Accept
		if accept == "" {
			accept = "application/rss+xml, application/atom+xml, application/feed+json, application/json;q=0.9, application/xml;q=0.9, text/xml;q=0.9, text/html;q=0.5, */*;q=0.1"
		}
		req.Header.Set("Accept", accept)
		// Conditional only for the address the feed was last read at.
		if hop == 0 {
			if r.ETag != "" {
				req.Header.Set("If-None-Match", r.ETag)
			}
			if r.Modified != "" {
				req.Header.Set("If-Modified-Since", r.Modified)
			}
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, transportError(ctx, current, err, out.Hops, f.timeout())
		}
		h := Hop{URL: current, Status: resp.StatusCode}
		switch resp.StatusCode {
		case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
			loc := resp.Header.Get("Location")
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
			_ = resp.Body.Close()
			if loc == "" {
				return nil, &FetchError{Kind: ErrStatus, URL: current, Status: resp.StatusCode, Detail: fmt.Sprintf("%s answered %d without saying where to go", hostOf(current), resp.StatusCode), Hops: out.Hops}
			}
			next, err := u.Parse(loc)
			if err != nil {
				return nil, &FetchError{Kind: ErrAddress, URL: current, Detail: "the redirect target isn't an address", Hops: out.Hops}
			}
			h.Location = next.String()
			out.Hops = append(out.Hops, h)
			if hop == 0 && (resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusPermanentRedirect) {
				permanent = true
			}
			if hop+1 > MaxRedirects {
				return nil, &FetchError{Kind: ErrRedirects, URL: current, Detail: fmt.Sprintf("%s redirected more than %d times", hostOf(r.URL), MaxRedirects), Hops: out.Hops}
			}
			current = next.String()
			continue
		}
		out.Hops = append(out.Hops, h)
		out.URL, out.Status = current, resp.StatusCode
		if permanent {
			out.Moved = current
		}
		out.ContentType = resp.Header.Get("Content-Type")
		out.ETag, out.Modified = resp.Header.Get("ETag"), resp.Header.Get("Last-Modified")
		if resp.StatusCode == http.StatusNotModified {
			_ = resp.Body.Close()
			out.NotModified = true
			out.Duration = time.Since(start)
			return out, nil
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
			_ = resp.Body.Close()
			return nil, &FetchError{Kind: ErrStatus, URL: current, Status: resp.StatusCode, Detail: statusDetail(current, resp.StatusCode), Hops: out.Hops}
		}
		if resp.ContentLength > f.maxBytes() {
			_ = resp.Body.Close()
			return nil, tooLarge(current, out.Hops, f.maxBytes())
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, f.maxBytes()+1))
		_ = resp.Body.Close()
		if err != nil {
			return nil, transportError(ctx, current, err, out.Hops, f.timeout())
		}
		if int64(len(body)) > f.maxBytes() {
			return nil, tooLarge(current, out.Hops, f.maxBytes())
		}
		out.Body = body
		out.Duration = time.Since(start)
		return out, nil
	}
}

func tooLarge(u string, hops []Hop, limit int64) *FetchError {
	return &FetchError{Kind: ErrTooLarge, URL: u, Hops: hops, Detail: fmt.Sprintf("%s sent more than %d MB, the limit for a feed", hostOf(u), limit>>20)}
}

func hostOf(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return u.Hostname()
	}
	return raw
}

func statusDetail(u string, status int) string {
	host := hostOf(u)
	switch {
	case status == http.StatusNotFound || status == http.StatusGone:
		return fmt.Sprintf("%s answered %d %s: there is no feed at %s", host, status, http.StatusText(status), shortURL(u))
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return fmt.Sprintf("%s answered %d %s: the feed needs a sign-in, which digests don't support", host, status, http.StatusText(status))
	case status == http.StatusTooManyRequests:
		return fmt.Sprintf("%s answered 429 Too Many Requests: it limits how often it can be read", host)
	case status >= 500:
		return fmt.Sprintf("%s answered %d %s", host, status, http.StatusText(status))
	}
	return fmt.Sprintf("%s answered %d %s", host, status, http.StatusText(status))
}

// shortURL is an address without its scheme ("blog.acme.io/feed/").
func shortURL(u string) string {
	return strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
}

func blockedDetail(u string, err error) string {
	msg := err.Error()
	if i := strings.Index(msg, netguard.ErrBlocked.Error()+": "); i >= 0 {
		msg = msg[i+len(netguard.ErrBlocked.Error())+2:]
	}
	return fmt.Sprintf("%s can't be read: %s. An instance admin can allow private addresses for self-hosted feeds", hostOf(u), msg)
}

// transportError words a failed connection: "blog.acme.io didn't answer:
// connection timed out after 15 s."
func transportError(ctx context.Context, u string, err error, hops []Hop, timeout time.Duration) *FetchError {
	host := hostOf(u)
	fe := &FetchError{Kind: ErrUnreachable, URL: u, Hops: hops}
	var dnsErr *net.DNSError
	var certErr *tls.CertificateVerificationError
	switch {
	case netguard.IsBlocked(err):
		fe.Kind = ErrBlocked
		fe.Detail = blockedDetail(u, err)
	case errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil || strings.Contains(err.Error(), "Client.Timeout") || isTimeout(err):
		fe.Kind = ErrTimeout
		fe.Detail = fmt.Sprintf("%s didn't answer: connection timed out after %d s", host, int(timeout/time.Second))
	case errors.As(err, &dnsErr):
		fe.Detail = fmt.Sprintf("%s didn't answer: its name doesn't resolve (DNS lookup failed)", host)
	case errors.As(err, &certErr):
		fe.Detail = fmt.Sprintf("%s didn't answer securely: its TLS certificate isn't valid", host)
	case strings.Contains(err.Error(), "connection refused"):
		fe.Detail = fmt.Sprintf("%s didn't answer: connection refused", host)
	case strings.Contains(err.Error(), "connection reset"):
		fe.Detail = fmt.Sprintf("%s didn't answer: the connection was reset", host)
	default:
		fe.Detail = fmt.Sprintf("%s didn't answer: %s", host, netguard.Describe(ctx, err))
	}
	return fe
}

func isTimeout(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}
