// Package smsfake holds what the SMS providers' fakes share: an in-process
// transport, so the demo's sample providers answer without any network,
// and a recorder of the requests a fake received.
package smsfake

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
)

// Client serves h in process: requests never leave the machine.
func Client(h http.Handler) *http.Client {
	return &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		if r.Body == nil {
			r.Body = http.NoBody
		}
		h.ServeHTTP(rec, r)
		resp := rec.Result()
		resp.Request = r
		return resp, nil
	})}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Recorded is one request a fake received.
type Recorded struct {
	Method string
	Path   string
	Body   string
}

// Recorder keeps the requests a fake received.
type Recorder struct {
	mu   sync.Mutex
	reqs []Recorded
}

// Record notes a request and gives its body back for reading.
func (r *Recorder) Record(req *http.Request) []byte {
	var b []byte
	if req.Body != nil {
		b, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(b))
	}
	r.mu.Lock()
	r.reqs = append(r.reqs, Recorded{Method: req.Method, Path: req.URL.Path, Body: string(b)})
	r.mu.Unlock()
	return b
}

// Requests returns what was received, oldest first.
func (r *Recorder) Requests() []Recorded {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Recorded(nil), r.reqs...)
}

// Mode is a fake's failure mode: "" answers normally, "outage" answers
// 503, "auth" answers 401.
type Mode func() string

// Of returns the current mode, "" when unset.
func (m Mode) Of() string {
	if m == nil {
		return ""
	}
	return m()
}
