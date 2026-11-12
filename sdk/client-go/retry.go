package reputecast

import (
	"context"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// MaxRetryDelay is the longest wait between two attempts, whatever
// Retry-After asks.
const MaxRetryDelay = 30 * time.Second

const baseRetryDelay = 500 * time.Millisecond

// retryStatus reports whether a retry may fix an answer: rate limited, or
// the server briefly unavailable.
func retryStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// retryable reports whether req may be sent again: its method is
// idempotent, or it is a POST with an Idempotency-Key (the server answers a
// repeat with the first result instead of acting twice), and its body can
// be sent again.
func retryable(req *http.Request) bool {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete:
	case http.MethodPost:
		if req.Header.Get("Idempotency-Key") == "" {
			return false
		}
	default:
		return false
	}
	return req.Body == nil || req.Body == http.NoBody || req.GetBody != nil
}

// retryDelay is how long to wait before retry number attempt (from 0): the
// answer's Retry-After (seconds or an HTTP date) when it has one, else
// exponential backoff from 0.5 s with jitter; never more than 30 s.
func retryDelay(attempt int, retryAfter string, now time.Time, random func() float64) time.Duration {
	if v := strings.TrimSpace(retryAfter); v != "" {
		d, ok := time.Duration(0), false
		if s, err := strconv.ParseUint(v, 10, 32); err == nil {
			d, ok = time.Duration(s)*time.Second, true
		} else if at, err := http.ParseTime(v); err == nil {
			d, ok = max(at.Sub(now), 0), true
		}
		if ok {
			return min(d, MaxRetryDelay)
		}
	}
	ceiling := MaxRetryDelay
	if attempt < 16 {
		ceiling = min(baseRetryDelay<<attempt, MaxRetryDelay)
	}
	return ceiling/2 + time.Duration(random()*float64(ceiling/2))
}

// retryTransport sends a retryable request again after a 429, 502, 503 or
// 504, up to maxRetries times, giving each attempt timeout (none when 0)
// to send the request and read the answer.
type retryTransport struct {
	next       http.RoundTripper
	maxRetries int
	timeout    time.Duration
	sleep      func(context.Context, time.Duration) error
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	retries := 0
	if retryable(req) {
		retries = t.maxRetries
	}
	for attempt := 0; ; attempt++ {
		ctx, cancel := req.Context(), context.CancelFunc(func() {})
		if t.timeout > 0 {
			ctx, cancel = context.WithTimeout(ctx, t.timeout)
		}
		r := req.Clone(ctx)
		if attempt > 0 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				cancel()
				return nil, err
			}
			r.Body = body
		}
		res, err := t.next.RoundTrip(r)
		if err != nil {
			cancel()
			return nil, err
		}
		if attempt >= retries || !retryStatus(res.StatusCode) {
			// The attempt's time runs until the caller has read the answer.
			res.Body = cancelOnClose{res.Body, cancel}
			return res, nil
		}
		delay := retryDelay(attempt, res.Header.Get("Retry-After"), time.Now(), rand.Float64)
		// Read a little of the body so the connection can be used again.
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10))
		_ = res.Body.Close()
		cancel()
		if err := t.sleep(req.Context(), delay); err != nil {
			return nil, err
		}
	}
}

type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b cancelOnClose) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()
	return err
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
