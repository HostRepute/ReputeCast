package accessibility

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/riverqueue/river"
	"golang.org/x/net/html"

	"github.com/hostrepute/reputecast/internal/contentcheck"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// Link text that says nothing gets the linked page's title as a
// suggestion. The page is read once in the background (with the link
// check's guard against internal addresses) and kept for a week.

// TitleArgs reads the titles of linked pages.
type TitleArgs struct {
	URLs []string `json:"urls"`
}

// Kind implements river.JobArgs.
func (TitleArgs) Kind() string { return "accessibility.link_titles" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (TitleArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 2, UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 10 * time.Minute}}
}

// TitleFetcher reads a page's title and language.
type TitleFetcher interface {
	Fetch(ctx context.Context, url string) (PageTitle, int, error)
}

// TitleWorker stores the titles of linked pages.
type TitleWorker struct {
	river.WorkerDefaults[TitleArgs]
	Q       *store.Queries
	Fetcher TitleFetcher
	Now     func() time.Time
}

// maxTitlesPerJob bounds one job's requests.
const maxTitlesPerJob = 20

// Work implements river.Worker.
func (w *TitleWorker) Work(ctx context.Context, job *river.Job[TitleArgs]) error {
	urls := job.Args.URLs
	if len(urls) > maxTitlesPerJob {
		urls = urls[:maxTitlesPerJob]
	}
	now := time.Now
	if w.Now != nil {
		now = w.Now
	}
	for _, u := range urls {
		if !fetchable(u) {
			continue
		}
		t, status, err := w.Fetcher.Fetch(ctx, u)
		if err != nil {
			t = PageTitle{}
		}
		if err := w.Q.AccessibilityUpsertLinkTitle(ctx, store.AccessibilityUpsertLinkTitleParams{Url: u, Title: t.Title, Lang: t.Lang, Status: int32(status), FetchedAt: now()}); err != nil {
			return err
		}
	}
	return nil
}

// RiverTitles queues title reads on River.
type RiverTitles struct {
	Client func() *jobs.Client
}

// Enqueue implements TitleQueue.
func (r RiverTitles) Enqueue(ctx context.Context, urls []string) error {
	if r.Client == nil {
		return nil
	}
	c := r.Client()
	if c == nil || len(urls) == 0 {
		return nil
	}
	sorted := append([]string(nil), urls...)
	sort.Strings(sorted)
	_, err := c.Insert(ctx, TitleArgs{URLs: sorted}, nil)
	return err
}

// HTTPTitles reads titles over HTTP, never from internal addresses.
type HTTPTitles struct {
	Client *http.Client
}

// NewHTTPTitles returns a fetcher with short timeouts and the SSRF guard.
func NewHTTPTitles() *HTTPTitles {
	dialer := &net.Dialer{Timeout: 4 * time.Second, Control: contentcheck.PublicOnly}
	tr := &http.Transport{DialContext: dialer.DialContext, TLSHandshakeTimeout: 4 * time.Second, ResponseHeaderTimeout: 5 * time.Second, Proxy: nil}
	return &HTTPTitles{Client: &http.Client{Transport: tr, Timeout: 8 * time.Second, CheckRedirect: func(_ *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		return nil
	}}}
}

// maxPageBytes is how much of a page is read to find its title.
const maxPageBytes = 256 << 10

// Fetch implements TitleFetcher.
func (h *HTTPTitles) Fetch(ctx context.Context, url string) (PageTitle, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return PageTitle{}, 0, err
	}
	req.Header.Set("User-Agent", "ReputeCast link check (+https://reputecast.com)")
	req.Header.Set("Accept", "text/html")
	resp, err := h.Client.Do(req)
	if err != nil {
		return PageTitle{}, 0, err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode >= 400 || !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "html") {
		return PageTitle{}, resp.StatusCode, nil
	}
	return ParseTitle(io.LimitReader(resp.Body, maxPageBytes)), resp.StatusCode, nil
}

// ParseTitle reads a page's <title> and <html lang>.
func ParseTitle(r io.Reader) PageTitle {
	var out PageTitle
	z := html.NewTokenizer(r)
	inTitle := false
	var b strings.Builder
	for {
		switch z.Next() {
		case html.ErrorToken:
			out.Title = strings.TrimSpace(collapse(b.String()))
			return out
		case html.StartTagToken:
			name, hasAttr := z.TagName()
			switch string(name) {
			case "html":
				for hasAttr {
					var k, v []byte
					k, v, hasAttr = z.TagAttr()
					if string(k) == "lang" {
						out.Lang = strings.TrimSpace(string(v))
					}
				}
			case "title":
				inTitle = true
			case "body":
				out.Title = strings.TrimSpace(collapse(b.String()))
				return out
			}
		case html.EndTagToken:
			if name, _ := z.TagName(); string(name) == "title" {
				out.Title = strings.TrimSpace(collapse(b.String()))
				return out
			}
		case html.TextToken:
			if inTitle {
				b.Write(z.Text())
			}
		}
	}
}
