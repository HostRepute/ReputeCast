package reputecast

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// server answers with statuses in turn (the last one repeats) and records
// the requests' bodies and headers.
type server struct {
	mu      sync.Mutex
	answers []int
	headers map[int]http.Header
	body    string
	bodies  []string
	seen    []http.Header
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	s.mu.Lock()
	s.bodies = append(s.bodies, string(b))
	s.seen = append(s.seen, r.Header.Clone())
	status := s.answers[min(len(s.bodies), len(s.answers))-1]
	s.mu.Unlock()
	for k, v := range s.headers[status] {
		w.Header()[k] = v
	}
	if status >= 400 {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, `{"status":`+strconv.Itoa(status)+`,"title":"Busy","detail":"Try again."}`)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, s.body)
}

func (s *server) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.bodies)
}

func newTransport(maxRetries int, delays *[]time.Duration) *retryTransport {
	return &retryTransport{next: http.DefaultTransport, maxRetries: maxRetries, timeout: 5 * time.Second, sleep: func(_ context.Context, d time.Duration) error {
		*delays = append(*delays, d)
		return nil
	}}
}

func do(t *testing.T, rt http.RoundTripper, method, url, body string, header http.Header) *http.Response {
	t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(context.Background(), method, url, rd)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range header {
		req.Header[k] = v
	}
	res, err := (&http.Client{Transport: rt}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	return res
}

func TestRetriesWithBackoff(t *testing.T) {
	s := &server{answers: []int{503, 502, 200}}
	srv := httptest.NewServer(s)
	defer srv.Close()
	var delays []time.Duration
	if res := do(t, newTransport(2, &delays), "GET", srv.URL, "", nil); res.StatusCode != 200 || s.count() != 3 {
		t.Fatalf("status %d after %d requests", res.StatusCode, s.count())
	}
	if len(delays) != 2 || delays[0] < 250*time.Millisecond || delays[0] > 500*time.Millisecond || delays[1] < 500*time.Millisecond || delays[1] > time.Second {
		t.Errorf("delays %v", delays)
	}
}

func TestRetryAfter(t *testing.T) {
	now := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	zero, one := func() float64 { return 0 }, func() float64 { return 1 }
	for _, c := range []struct {
		attempt int
		header  string
		random  func() float64
		want    time.Duration
	}{
		{0, "2", zero, 2 * time.Second},
		{0, "120", zero, MaxRetryDelay},
		{0, "Wed, 11 Nov 2026 10:00:05 GMT", zero, 5 * time.Second},
		{0, "Wed, 11 Nov 2026 09:00:00 GMT", zero, 0},
		{0, "soon", zero, 250 * time.Millisecond},
		{1, "", one, time.Second},
		{40, "", one, MaxRetryDelay},
	} {
		if got := retryDelay(c.attempt, c.header, now, c.random); got != c.want {
			t.Errorf("retryDelay(%d, %q) = %v, want %v", c.attempt, c.header, got, c.want)
		}
	}

	s := &server{answers: []int{429, 200}, headers: map[int]http.Header{429: {"Retry-After": {"3"}}}}
	srv := httptest.NewServer(s)
	defer srv.Close()
	var delays []time.Duration
	if res := do(t, newTransport(2, &delays), "DELETE", srv.URL, "", nil); res.StatusCode != 200 || len(delays) != 1 || delays[0] != 3*time.Second {
		t.Fatalf("status %d, delays %v", res.StatusCode, delays)
	}
}

func TestRetriesAPostOnlyWithAnIdempotencyKey(t *testing.T) {
	s := &server{answers: []int{503, 200}}
	srv := httptest.NewServer(s)
	defer srv.Close()
	var delays []time.Duration
	if res := do(t, newTransport(2, &delays), "POST", srv.URL, `{"a":1}`, nil); res.StatusCode != 503 || s.count() != 1 {
		t.Fatalf("without a key: status %d after %d requests", res.StatusCode, s.count())
	}

	s = &server{answers: []int{504, 200}}
	srv2 := httptest.NewServer(s)
	defer srv2.Close()
	res := do(t, newTransport(2, &delays), "POST", srv2.URL, `{"a":1}`, http.Header{"Idempotency-Key": {"k-1"}})
	if res.StatusCode != 200 || len(s.bodies) != 2 || s.bodies[0] != `{"a":1}` || s.bodies[1] != `{"a":1}` {
		t.Fatalf("with a key: status %d, bodies %q", res.StatusCode, s.bodies)
	}
}

func TestGivesUpAndNeverRetriesOtherErrors(t *testing.T) {
	var delays []time.Duration
	for _, c := range []struct {
		answers    []int
		maxRetries int
		status     int
		requests   int
	}{
		{[]int{429}, 2, 429, 3},
		{[]int{500, 200}, 2, 500, 1},
		{[]int{503, 200}, 0, 503, 1},
	} {
		s := &server{answers: c.answers}
		srv := httptest.NewServer(s)
		res := do(t, newTransport(c.maxRetries, &delays), "GET", srv.URL, "", nil)
		srv.Close()
		if res.StatusCode != c.status || s.count() != c.requests {
			t.Errorf("%v with %d retries: status %d after %d requests", c.answers, c.maxRetries, res.StatusCode, s.count())
		}
	}
}

func TestStopsWaitingWhenTheContextEnds(t *testing.T) {
	s := &server{answers: []int{503, 200}, headers: map[int]http.Header{503: {"Retry-After": {"30"}}}}
	srv := httptest.NewServer(s)
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	rt := &retryTransport{next: http.DefaultTransport, maxRetries: 2, sleep: func(ctx context.Context, d time.Duration) error {
		cancel()
		return sleep(ctx, d)
	}}
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	if _, err := rt.RoundTrip(req); !errors.Is(err, context.Canceled) || s.count() != 1 {
		t.Fatalf("err %v after %d requests", err, s.count())
	}
}

func TestEachAttemptHasItsTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)
	rt := &retryTransport{next: http.DefaultTransport, timeout: 50 * time.Millisecond, sleep: sleep}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)
	if _, err := rt.RoundTrip(req); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err %v, want the attempt's deadline", err)
	}
}
