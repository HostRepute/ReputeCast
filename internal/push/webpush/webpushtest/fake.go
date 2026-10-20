// Package webpushtest is an in-process push service for tests and the demo:
// it answers the requests FCM, Mozilla's autopush, Apple and Windows would
// get, checks the VAPID token as they do, decrypts the payload when it made
// the subscription, and records every message. Nothing it's given ever
// reaches the network.
package webpushtest

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/push/webpush"
)

// Hosts are the push services' hosts, by browser.
var Hosts = map[string]string{
	"chrome":  "https://fcm.googleapis.com/fcm/send/",
	"android": "https://fcm.googleapis.com/fcm/send/",
	"edge":    "https://wns2-bl2p.notify.windows.com/w/?token=",
	"firefox": "https://updates.push.services.mozilla.com/wpush/v2/",
	"safari":  "https://web.push.apple.com/",
	"ios":     "https://web.push.apple.com/",
}

// Received is one message the service got.
type Received struct {
	At       time.Time
	Endpoint string
	Header   http.Header
	Body     []byte
	// Claims are the VAPID token's claims; Key the public key that signed it.
	Claims webpush.Claims
	Key    []byte
	// Payload is the decrypted message, for subscriptions this service made.
	Payload []byte
	// Status is what the service answered.
	Status int
}

// Service is the fake push service.
type Service struct {
	// Now is the clock tokens are checked against.
	Now func() time.Time

	mu       sync.Mutex
	keys     map[string]*ecdh.PrivateKey
	auth     map[string][]byte
	answers  map[string]int
	pages    map[string]Page
	received []Received
}

// Page is what the service answers a GET with, as the web server of a
// site would: a site's hosted service worker, the page a link opens.
type Page struct {
	Status int
	Header http.Header
	Body   []byte
}

// New makes an empty push service.
func New() *Service {
	return &Service{keys: map[string]*ecdh.PrivateKey{}, auth: map[string][]byte{}, answers: map[string]int{}, pages: map[string]Page{}}
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Subscribe makes a subscription as a browser would get from its push
// service: the endpoint and the keys, in base64url. The service keeps the
// private key, so it can read what's sent to it.
func (s *Service) Subscribe(browser string) (endpoint, p256dh, auth string) {
	prefix, ok := Hosts[browser]
	if !ok {
		prefix = Hosts["chrome"]
	}
	id := make([]byte, 24)
	_, _ = rand.Read(id)
	endpoint = prefix + hex.EncodeToString(id)
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	sec := make([]byte, 16)
	_, _ = rand.Read(sec)
	s.mu.Lock()
	s.keys[endpoint], s.auth[endpoint] = k, sec
	s.mu.Unlock()
	return endpoint, webpush.Encode(k.PublicKey().Bytes()), webpush.Encode(sec)
}

// Answer makes the service answer status for an endpoint from now on: 410
// for a subscription that ended, 429 or 503 for a busy service.
func (s *Service) Answer(endpoint string, status int) {
	s.mu.Lock()
	s.answers[endpoint] = status
	s.mu.Unlock()
}

// Serve makes GET requests for u answer p (any other GET answers 404), so
// tests check sites and links without the network.
func (s *Service) Serve(u string, p Page) {
	s.mu.Lock()
	s.pages[u] = p
	s.mu.Unlock()
}

// Received returns what the service got, oldest first.
func (s *Service) Received() []Received {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Received(nil), s.received...)
}

// Reset forgets what was received.
func (s *Service) Reset() {
	s.mu.Lock()
	s.received = nil
	s.mu.Unlock()
}

// Client sends every request to the service, whatever its host.
func (s *Service) Client() *http.Client {
	return &http.Client{Transport: transport{s}, Timeout: 10 * time.Second}
}

type transport struct{ s *Service }

func (t transport) RoundTrip(r *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	t.s.ServeHTTP(rec, r)
	resp := rec.Result()
	resp.Request = r
	return resp, nil
}

// ServeHTTP answers a push request as a push service does: 201 with a
// Location, 400 without aes128gcm, 403 for a token that doesn't verify,
// 413 over 4096 bytes, or the status set with Answer.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil { // a client's GET has none
		body, _ = io.ReadAll(io.LimitReader(r.Body, 1<<16))
	}
	endpoint := r.URL.String()
	rec := Received{At: s.now(), Endpoint: endpoint, Header: r.Header.Clone(), Body: body}
	answer := func(status int, msg string) {
		rec.Status = status
		s.mu.Lock()
		s.received = append(s.received, rec)
		s.mu.Unlock()
		if status == http.StatusCreated {
			w.Header().Set("Location", strings.TrimSuffix(endpoint, "/")+"/m/"+hex.EncodeToString(body[:min(8, len(body))]))
		}
		w.WriteHeader(status)
		if msg != "" {
			_, _ = w.Write([]byte(`{"reason":"` + msg + `"}`))
		}
	}
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		s.mu.Lock()
		p, ok := s.pages[endpoint]
		s.mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		for k, vs := range p.Header {
			w.Header()[k] = vs
		}
		w.WriteHeader(p.Status)
		if r.Method == http.MethodGet {
			_, _ = w.Write(p.Body)
		}
		return
	}
	if r.Method != http.MethodPost {
		answer(http.StatusMethodNotAllowed, "MethodNotAllowed")
		return
	}
	origin, _ := webpush.Origin(endpoint)
	claims, key, err := webpush.Verify(r.Header.Get("Authorization"), origin, s.now())
	if err != nil {
		answer(http.StatusForbidden, "BadJwtToken")
		return
	}
	rec.Claims, rec.Key = claims, key
	if r.Header.Get("Content-Encoding") != "aes128gcm" || r.Header.Get("TTL") == "" {
		answer(http.StatusBadRequest, "BadRequest")
		return
	}
	if len(body) > webpush.RecordSize {
		answer(http.StatusRequestEntityTooLarge, "PayloadTooLarge")
		return
	}
	s.mu.Lock()
	status, forced := s.answers[endpoint]
	k, sec := s.keys[endpoint], s.auth[endpoint]
	s.mu.Unlock()
	if forced {
		reason := map[int]string{404: "NotFound", 410: "Unregistered", 429: "TooManyRequests", 500: "InternalServerError", 503: "ServiceUnavailable"}[status]
		answer(status, reason)
		return
	}
	if k != nil {
		p, err := webpush.Decrypt(k, sec, body)
		if err != nil {
			answer(http.StatusBadRequest, "DecryptionFailed")
			return
		}
		rec.Payload = p
	}
	answer(http.StatusCreated, "")
}
