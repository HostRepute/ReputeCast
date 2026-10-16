package feed

import (
	"context"
	"errors"
	"net/url"
)

// Result is a read feed with what answered.
type Result struct {
	*Response
	// Feed is nil when the feed didn't change (NotModified).
	Feed *Feed
}

// Read fetches and parses a feed. A document that isn't a feed fails with
// a *NotFeedError carrying its first lines.
func (f *Fetcher) Read(ctx context.Context, r Request) (*Result, error) {
	resp, err := f.Get(ctx, r)
	if err != nil {
		return nil, err
	}
	out := &Result{Response: resp}
	if resp.NotModified {
		return out, nil
	}
	base, _ := url.Parse(resp.URL)
	fd, err := Parse(resp.Body, base)
	if err != nil {
		return out, err
	}
	out.Feed = fd
	return out, nil
}

// Found is the feed found from an address someone typed.
type Found struct {
	*Result
	// Page is the page address the feed was linked from ("" when the
	// address was the feed itself).
	Page string
}

// Find reads an address as a feed and, when it is a web page, reads the
// feeds the page links to in its head (blog.acme.io finds
// blog.acme.io/feed/). The first one that parses wins.
func (f *Fetcher) Find(ctx context.Context, address string) (*Found, error) {
	res, err := f.Read(ctx, Request{URL: address})
	if err == nil {
		return &Found{Result: res}, nil
	}
	var nf *NotFeedError
	if res == nil || res.Response == nil || !asNotFeed(err, &nf) || !nf.HTML {
		return nil, err
	}
	base, _ := url.Parse(res.URL)
	links := Discover(res.Body, base)
	if len(links) == 0 {
		return nil, &NoFeedLinkError{Page: res.URL}
	}
	var last error
	for i, l := range links {
		if i == 3 {
			break
		}
		r2, err := f.Read(ctx, Request{URL: l.URL})
		if err == nil {
			// The hops to the page come first.
			r2.Hops = append(append([]Hop{}, res.Hops...), r2.Hops...)
			return &Found{Result: r2, Page: res.URL}, nil
		}
		last = err
	}
	return nil, last
}

func asNotFeed(err error, nf **NotFeedError) bool { return errors.As(err, nf) }

// NoFeedLinkError means a page links to no feed.
type NoFeedLinkError struct{ Page string }

func (e *NoFeedLinkError) Error() string { return "the page links to no feed" }
