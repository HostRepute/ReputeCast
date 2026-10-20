package webpush

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Sending (RFC 8030 §5): one POST to the subscription's push resource with
// the encrypted payload, how long the push service may keep it (TTL), its
// urgency and, to replace an undelivered message, a topic.

// Urgency is a message's urgency (RFC 8030 §5.3): push services and
// browsers deliver low-urgency messages when it suits the battery.
type Urgency string

// Urgencies.
const (
	UrgencyVeryLow Urgency = "very-low"
	UrgencyLow     Urgency = "low"
	UrgencyNormal  Urgency = "normal"
	UrgencyHigh    Urgency = "high"
)

// MaxTTL is the longest time push services keep a message (FCM's limit,
// the shortest of the major services).
const MaxTTL = 28 * 24 * time.Hour

// Message is one push message to one subscription.
type Message struct {
	// Payload is the plaintext; Send encrypts it.
	Payload []byte
	// TTL is how long the push service keeps the message while the
	// browser is offline. After that it's dropped.
	TTL time.Duration
	// Urgency is normal when empty.
	Urgency Urgency
	// Topic replaces an undelivered message with the same topic (see
	// TopicFor).
	Topic string
}

// TopicFor turns a notification tag into a Topic header value: at most 32
// characters of the base64url alphabet (RFC 8030 §5.4).
func TopicFor(tag string) string {
	if tag == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(tag))
	return Encode(sum[:])[:32]
}

// Outcome is what a push service did with a message.
type Outcome string

// Outcomes.
const (
	// Accepted: 201 Created, the push service will deliver it.
	Accepted Outcome = "accepted"
	// Gone: 404 or 410, the subscription ended (the browser unsubscribed
	// or the push service expired it). Remove it.
	Gone Outcome = "gone"
	// TooLarge: 413, the payload is over the push service's limit.
	TooLarge Outcome = "too_large"
	// RateLimited: 429, retry after RetryAfter.
	RateLimited Outcome = "rate_limited"
	// Rejected: 400, 401 or 403. The request or the key was refused; a
	// 403 usually means the subscription was made with other keys.
	Rejected Outcome = "rejected"
	// Unavailable: a 5xx answer or no answer at all. Retry later.
	Unavailable Outcome = "unavailable"
)

// Result is a push service's answer.
type Result struct {
	Outcome Outcome
	// Status is the HTTP status, 0 when nothing answered.
	Status int
	// RetryAfter is the push service's Retry-After, for 429 and 503.
	RetryAfter time.Duration
	// Location is the message resource of an accepted message.
	Location string
	// Detail is the start of the push service's explanation, for the
	// Events tab ("410 Gone · push subscription has unsubscribed or
	// expired").
	Detail string
}

// Retryable reports whether sending again later may work.
func (r Result) Retryable() bool { return r.Outcome == RateLimited || r.Outcome == Unavailable }

// Sender sends messages signed with one workspace's VAPID key.
type Sender struct {
	Key *VAPIDKey
	// Contact is the "sub" claim: "mailto:support@acme.io".
	Contact string
	// HTTP is the client push services are called with (the SSRF-guarded
	// client in production, the in-process push service in tests and the
	// demo).
	HTTP *http.Client
	Now  func() time.Time

	mu     sync.Mutex
	tokens map[string]token
}

// token is a signed Authorization value reused for an hour per push
// service.
type token struct {
	value string
	until time.Time
}

func (s *Sender) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// authorization returns the Authorization header for a push service,
// signing a new token at most once an hour.
func (s *Sender) authorization(aud string) (string, error) {
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.tokens[aud]; ok && now.Before(t.until) {
		return t.value, nil
	}
	v, err := s.Key.Authorization(aud, s.Contact, now.Add(TokenLifetime))
	if err != nil {
		return "", err
	}
	if s.tokens == nil {
		s.tokens = map[string]token{}
	}
	s.tokens[aud] = token{value: v, until: now.Add(time.Hour)}
	return v, nil
}

// Request builds the push request for a message: encrypted, signed, with
// its TTL, urgency and topic.
func (s *Sender) Request(ctx context.Context, sub Subscription, m Message) (*http.Request, error) {
	if s.Key == nil {
		return nil, errors.New("webpush: no VAPID key")
	}
	aud, err := Origin(sub.Endpoint)
	if err != nil {
		return nil, err
	}
	body, err := Encrypt(sub, m.Payload)
	if err != nil {
		return nil, err
	}
	auth, err := s.authorization(aud)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	ttl := min(max(m.TTL, 0), MaxTTL)
	req.Header.Set("TTL", strconv.FormatInt(int64(ttl/time.Second), 10))
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Authorization", auth)
	u := m.Urgency
	if u == "" {
		u = UrgencyNormal
	}
	req.Header.Set("Urgency", string(u))
	if m.Topic != "" {
		req.Header.Set("Topic", m.Topic)
	}
	return req, nil
}

// Send delivers a message to the subscription's push service and says what
// it did. A network failure is an Unavailable result, not an error; the
// error is for a request that can't be built (bad keys, a payload too
// large).
func (s *Sender) Send(ctx context.Context, sub Subscription, m Message) (Result, error) {
	req, err := s.Request(ctx, sub, m)
	if err != nil {
		return Result{}, err
	}
	client := s.HTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{Outcome: Unavailable, Detail: shorten(err.Error())}, nil
	}
	defer resp.Body.Close()
	return Classify(resp), nil
}

// Classify reads a push service's answer.
func Classify(resp *http.Response) Result {
	r := Result{Status: resp.StatusCode}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	r.Detail = shorten(strconv.Itoa(resp.StatusCode) + " " + http.StatusText(resp.StatusCode) + detailOf(b))
	switch {
	case resp.StatusCode == http.StatusCreated, resp.StatusCode == http.StatusOK, resp.StatusCode == http.StatusAccepted:
		r.Outcome = Accepted
		r.Location = resp.Header.Get("Location")
	case resp.StatusCode == http.StatusNotFound, resp.StatusCode == http.StatusGone:
		r.Outcome = Gone
	case resp.StatusCode == http.StatusRequestEntityTooLarge:
		r.Outcome = TooLarge
	case resp.StatusCode == http.StatusTooManyRequests:
		r.Outcome = RateLimited
		r.RetryAfter = retryAfter(resp.Header.Get("Retry-After"))
	case resp.StatusCode >= 500:
		r.Outcome = Unavailable
		r.RetryAfter = retryAfter(resp.Header.Get("Retry-After"))
	default:
		r.Outcome = Rejected
	}
	return r
}

// detailOf is the push service's explanation, from its JSON or text body.
func detailOf(b []byte) string {
	s := strings.TrimSpace(string(b))
	if s == "" {
		return ""
	}
	// FCM and Apple answer JSON with a reason; Mozilla with "message".
	for _, k := range []string{`"reason":"`, `"message":"`, `"error":"`} {
		if i := strings.Index(s, k); i >= 0 {
			rest := s[i+len(k):]
			if j := strings.IndexByte(rest, '"'); j > 0 {
				return " · " + rest[:j]
			}
		}
	}
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "<") {
		return ""
	}
	return " · " + s
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

func shorten(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= 200 {
		return s
	}
	cut := 200
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}
