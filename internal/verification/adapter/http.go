package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ErrRefused means the service refused the request for another reason (a
// malformed request); retrying won't help.
var ErrRefused = errors.New("the service refused the request")

// maxBody caps how much of an answer is read.
const maxBody = 4 << 20

// Request is one call to a service's JSON API.
type Request struct {
	Method string
	URL    string
	Query  url.Values
	Header http.Header
	// Body is sent as JSON when set.
	Body any
}

// Response is a 2xx answer.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// Classify maps a non-2xx answer to the adapter errors. noCredits tells a
// service's "out of credits" answers apart when it shares a status with
// another error (Kickbox answers 403 for both a bad key and no balance).
func Classify(status int, header http.Header, body []byte, noCredits func(status int, body []byte) bool) error {
	msg := strings.TrimSpace(string(body))
	if len(msg) > 300 {
		msg = msg[:300]
	}
	ce := &CallError{Status: status, Message: msg}
	switch {
	case noCredits != nil && noCredits(status, body):
		ce.Kind = ErrNoCredits
	case status == http.StatusPaymentRequired:
		ce.Kind = ErrNoCredits
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		ce.Kind = ErrAuth
	case status == http.StatusTooManyRequests:
		ce.Kind = ErrRateLimited
		ce.RetryAfter = retryAfter(header.Get("Retry-After"))
	case status >= 500 || status == http.StatusRequestTimeout:
		ce.Kind = ErrUnavailable
		ce.RetryAfter = retryAfter(header.Get("Retry-After"))
	default:
		ce.Kind = ErrRefused
	}
	return ce
}

func retryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// Do sends req and returns a 2xx answer, or an error from Classify. Network
// failures and timeouts are ErrUnavailable.
func Do(ctx context.Context, hc *http.Client, req Request, noCredits func(status int, body []byte) bool) (*Response, error) {
	u := req.URL
	if len(req.Query) > 0 {
		sep := "?"
		if strings.Contains(u, "?") {
			sep = "&"
		}
		u += sep + req.Query.Encode()
	}
	var body io.Reader
	if req.Body != nil {
		b, err := json.Marshal(req.Body)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	}
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	hr, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, &CallError{Kind: ErrRefused, Message: err.Error()}
	}
	for k, vs := range req.Header {
		for _, v := range vs {
			hr.Header.Add(k, v)
		}
	}
	hr.Header.Set("Accept", "application/json")
	if req.Body != nil {
		hr.Header.Set("Content-Type", "application/json")
	}
	hr.Header.Set("User-Agent", "ReputeCast")
	resp, err := hc.Do(hr)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, &CallError{Kind: ErrUnavailable, Message: scrub(err.Error(), req.Query)}
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, &CallError{Kind: ErrUnavailable, Status: resp.StatusCode, Message: err.Error()}
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, Classify(resp.StatusCode, resp.Header, b, noCredits)
	}
	return &Response{Status: resp.StatusCode, Header: resp.Header, Body: b}, nil
}

// scrub removes query values that carry the key from an error message
// (net/http errors quote the whole URL).
func scrub(msg string, q url.Values) string {
	for _, k := range []string{"api_key", "apikey", "key"} {
		if v := q.Get(k); v != "" {
			msg = strings.ReplaceAll(msg, url.QueryEscape(v), "…")
			msg = strings.ReplaceAll(msg, v, "…")
		}
	}
	return msg
}

// Decode reads a JSON answer into v; a body that isn't JSON is an
// ErrUnavailable (proxies answer HTML pages).
func Decode(resp *Response, v any) error {
	if err := json.Unmarshal(resp.Body, v); err != nil {
		return &CallError{Kind: ErrUnavailable, Status: resp.Status, Message: "the answer wasn't JSON"}
	}
	return nil
}

// OneByOne checks emails with verify, up to parallel at a time, and returns
// the answers in order. The first error stops it; answers before it are
// returned with the error so a job keeps them. An error stops only the
// addresses after the one that failed: checks of the addresses before it
// run to the end, however late they are, since the service answers (and
// charges for) them whatever order the parallel checks reach it in.
func OneByOne(ctx context.Context, emails []string, parallel int, verify func(ctx context.Context, email string) (Answer, error)) ([]Answer, error) {
	if parallel < 1 {
		parallel = 1
	}
	out := make([]Answer, len(emails))
	errs := make([]error, len(emails))
	cancels := make([]context.CancelFunc, len(emails))
	var mu sync.Mutex
	failed := len(emails) // the first address that failed
	stopAfter := func(i int) {
		mu.Lock()
		defer mu.Unlock()
		if i >= failed {
			return
		}
		failed = i
		for _, c := range cancels[i+1:] {
			if c != nil {
				c()
			}
		}
	}
	sem := make(chan struct{}, parallel)
	var wg sync.WaitGroup
	for i, e := range emails {
		sem <- struct{}{}
		mu.Lock()
		if failed < i {
			mu.Unlock()
			<-sem
			errs[i] = context.Canceled
			continue
		}
		cctx, cancel := context.WithCancel(ctx)
		cancels[i] = cancel
		mu.Unlock()
		wg.Add(1)
		go func() {
			defer func() { cancel(); <-sem; wg.Done() }()
			a, err := verify(cctx, e)
			if err != nil {
				errs[i] = err
				stopAfter(i)
				return
			}
			out[i] = a
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			// Keep the answers before the first failure, in order.
			first := err
			for j := range errs {
				if errs[j] != nil && !errors.Is(errs[j], context.Canceled) {
					first = errs[j]
					break
				}
			}
			return out[:i], first
		}
	}
	return out, nil
}
