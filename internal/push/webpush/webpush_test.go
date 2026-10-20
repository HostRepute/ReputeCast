package webpush

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func mustDecode(t *testing.T, s string) []byte {
	t.Helper()
	b, err := Decode(strings.Join(strings.Fields(s), ""))
	if err != nil {
		t.Fatalf("decode %q: %v", s, err)
	}
	return b
}

// RFC 8291 Appendix A: the intermediate values of the example in §5.
func TestRFC8291AppendixA(t *testing.T) {
	plaintext := mustDecode(t, "V2hlbiBJIGdyb3cgdXAsIEkgd2FudCB0byBiZSBhIHdhdGVybWVsb24")
	asPublic := mustDecode(t, "BP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIg Dll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A8")
	asPrivate := mustDecode(t, "yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw")
	uaPublic := mustDecode(t, "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV- JvLexhqUzORcx aOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4")
	uaPrivate := mustDecode(t, "q1dXpw3UpT5VOmu_cf_v6ih07Aems3njxI-JWgLcM94")
	salt := mustDecode(t, "DGv6ra1nlYgDCS1FRnbzlw")
	auth := mustDecode(t, "BTBZMqHH6r4Tts7J_aSIgg")

	as, err := ecdh.P256().NewPrivateKey(asPrivate)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(as.PublicKey().Bytes(), asPublic) {
		t.Fatal("as_public doesn't match as_private")
	}
	ua, err := ecdh.P256().NewPrivateKey(uaPrivate)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ua.PublicKey().Bytes(), uaPublic) {
		t.Fatal("ua_public doesn't match ua_private")
	}

	// The intermediate values.
	secret, err := as.ECDH(ua.PublicKey())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := Encode(secret), "kyrL1jIIOHEzg3sM2ZWRHDRB62YACZhhSlknJ672kSs"; got != want {
		t.Fatalf("ecdh_secret %s, want %s", got, want)
	}
	ikm, err := inputKey(secret, auth, uaPublic, asPublic)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := Encode(ikm), "S4lYMb_L0FxCeq0WhDx813KgSYqU26kOyzWUdsXYyrg"; got != want {
		t.Fatalf("IKM %s, want %s", got, want)
	}

	// The whole body: the 86-octet header, then the ciphertext.
	body, err := encrypt(uaPublic, auth, as, salt, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	wantHeader := mustDecode(t, "DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z 9KsN6nGRTbVYI_c7VJSPQTBtkgcy27ml mlMoZIIgDll6e3vCYLocInmYWAmS6Tlz AC8wEqKK6PBru3jl7A8")
	wantCipher := mustDecode(t, "8pfeW0KbunFT06SuDKoJH9Ql87S1QUrd irN6GcG7sFz1y1sqLgVi1VhjVkHsUoEs bI_0LpXMuGvnzQ")
	if len(wantHeader) != 86 {
		t.Fatalf("the RFC's header is %d octets", len(wantHeader))
	}
	if !bytes.Equal(body[:86], wantHeader) {
		t.Fatalf("header\n got %s\nwant %s", Encode(body[:86]), Encode(wantHeader))
	}
	if !bytes.Equal(body[86:], wantCipher) {
		t.Fatalf("ciphertext\n got %s\nwant %s", Encode(body[86:]), Encode(wantCipher))
	}

	// The browser's side decrypts it back.
	got, err := Decrypt(ua, auth, body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("decrypted %q", got)
	}
}

// RFC 8188 §3.1 and §3.2: content coding examples with known keys, one
// record and two records with a key id.
func TestRFC8188Examples(t *testing.T) {
	for _, c := range []struct {
		name, ikm, body string
	}{
		{"one record", "yqdlZ-tYemfogSmv7Ws5PQ", "I1BsxtFttlv3u_Oo94xnmwAAEAAA-NAVub2qFgBEuQKRapoZu-IxkIva3MEB1PD- ly8Thjg"},
		{"two records", "BO3ZVPxUlnLORbVGMpbT1Q", "uNCkWiNYzKTnBN9ji3-qWAAAABkCYTHOG8chz_gnvgOqdGYovxyjuqRyJFjEDyoF 1Fvkj6hQPdPHI51OEUKEpgz3SsLWIqS_uA"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := decryptContent(mustDecode(t, c.ikm), mustDecode(t, c.body))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != "I am the walrus" {
				t.Fatalf("got %q", got)
			}
		})
	}
	// A tampered record doesn't authenticate.
	b := mustDecode(t, "I1BsxtFttlv3u_Oo94xnmwAAEAAA-NAVub2qFgBEuQKRapoZu-IxkIva3MEB1PD- ly8Thjg")
	b[len(b)-1] ^= 1
	if _, err := decryptContent(mustDecode(t, "yqdlZ-tYemfogSmv7Ws5PQ"), b); !errors.Is(err, ErrBody) {
		t.Fatalf("tampered: %v", err)
	}
}

func TestEncryptRoundTripAndLimit(t *testing.T) {
	ua, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	_, _ = rand.Read(auth)
	sub := Subscription{Endpoint: "https://push.example.net/p/abc", P256dh: ua.PublicKey().Bytes(), Auth: auth}
	for _, n := range []int{0, 1, 200, MaxPayload} {
		msg := bytes.Repeat([]byte{'x'}, n)
		body, err := Encrypt(sub, msg)
		if err != nil {
			t.Fatalf("%d bytes: %v", n, err)
		}
		if len(body) > RecordSize {
			t.Fatalf("%d bytes made a %d-byte body", n, len(body))
		}
		got, err := Decrypt(ua, auth, body)
		if err != nil || !bytes.Equal(got, msg) {
			t.Fatalf("%d bytes: %v", n, err)
		}
	}
	if _, err := Encrypt(sub, make([]byte, MaxPayload+1)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("over the limit: %v", err)
	}
	// Two messages to the same browser never share a key or a salt.
	a, _ := Encrypt(sub, []byte("same"))
	b, _ := Encrypt(sub, []byte("same"))
	if bytes.Equal(a[:86], b[:86]) {
		t.Fatal("the salt and key were reused")
	}
}

// RFC 8292 §2.4: the example token verifies with its key and carries the
// example's claims.
func TestRFC8292Example(t *testing.T) {
	h := "vapid t=eyJ0eXAiOiJKV1QiLCJhbGciOiJFUzI1NiJ9.eyJhdWQiOiJodHRwczovL3B1c2guZXhhbXBsZS5uZXQiLCJleHAiOjE0NTM1MjM3NjgsInN1YiI6Im1haWx0bzpwdXNoQGV4YW1wbGUuY29tIn0.i3CYb7t4xfxCDquptFOepC9GAu_HLGkMlMuCGSK2rpiUfnK9ojFwDXb1JrErtmysazNjjvW2L9OkSSHzvoD1oA, k=BA1Hxzyi1RUM1b5wjxsn7nGxAszw2u61m164i3MrAIxHF6YK5h4SDYic-dRuU_RCPCfA5aq9ojSwk5Y2EmClBPs"
	tok, key, err := ParseAuthorization(h)
	if err != nil {
		t.Fatal(err)
	}
	c, err := VerifyToken(tok, key)
	if err != nil {
		t.Fatal(err)
	}
	if c.Aud != "https://push.example.net" || c.Exp != 1453523768 || c.Sub != "mailto:push@example.com" {
		t.Fatalf("claims %+v", c)
	}
	// It expired on 2016-01-23T04:36:08Z.
	if _, _, err := Verify(h, "https://push.example.net", time.Unix(1453523768, 0).Add(-time.Hour)); err != nil {
		t.Fatalf("before expiry: %v", err)
	}
	if _, _, err := Verify(h, "https://push.example.net", time.Unix(1453523768, 0)); !errors.Is(err, ErrClaims) {
		t.Fatalf("at expiry: %v", err)
	}
	// Another key doesn't verify it.
	other, _ := GenerateVAPIDKey()
	if _, err := VerifyToken(tok, other.PublicKey()); !errors.Is(err, ErrSignature) {
		t.Fatalf("other key: %v", err)
	}
}

func TestVAPIDSignAndVerify(t *testing.T) {
	k, err := GenerateVAPIDKey()
	if err != nil {
		t.Fatal(err)
	}
	if len(k.PublicKey()) != 65 || k.PublicKey()[0] != 4 || len(k.PrivateBytes()) != 32 {
		t.Fatal("key encodings")
	}
	// The sealed scalar reads back as the same key.
	k2, err := ParseVAPIDKey(k.PrivateBytes())
	if err != nil || k2.PublicKeyString() != k.PublicKeyString() {
		t.Fatalf("parse: %v", err)
	}
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	h, err := k.Authorization("https://fcm.googleapis.com", "mailto:support@acme.io", now.Add(TokenLifetime))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "vapid t=") || !strings.Contains(h, ", k="+k.PublicKeyString()) {
		t.Fatalf("header %q", h)
	}
	c, key, err := Verify(h, "https://fcm.googleapis.com", now)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(key, k.PublicKey()) || c.Sub != "mailto:support@acme.io" || c.Exp != now.Add(TokenLifetime).Unix() {
		t.Fatalf("claims %+v", c)
	}
	// The header is the one the RFC writes.
	tok, _, _ := ParseAuthorization(h)
	if hb, _ := Decode(strings.Split(tok, ".")[0]); string(hb) != `{"typ":"JWT","alg":"ES256"}` {
		t.Fatalf("header %s", hb)
	}
	if _, _, err := Verify(h, "https://updates.push.services.mozilla.com", now); !errors.Is(err, ErrClaims) {
		t.Fatalf("audience: %v", err)
	}
	long, _ := k.Authorization("https://fcm.googleapis.com", "", now.Add(25*time.Hour))
	if _, _, err := Verify(long, "https://fcm.googleapis.com", now); !errors.Is(err, ErrClaims) {
		t.Fatalf("over 24 hours: %v", err)
	}
	if _, _, err := ParseAuthorization("Bearer abc"); !errors.Is(err, ErrAuthorization) {
		t.Fatalf("scheme: %v", err)
	}
}

func TestParseSubscription(t *testing.T) {
	ua, _ := ecdh.P256().GenerateKey(rand.Reader)
	p256dh := Encode(ua.PublicKey().Bytes())
	auth := Encode(make([]byte, 16))
	if _, err := ParseSubscription("https://fcm.googleapis.com/fcm/send/abc", p256dh, auth); err != nil {
		t.Fatal(err)
	}
	// Padded base64 is accepted, as some browsers send it.
	if _, err := ParseSubscription("https://fcm.googleapis.com/fcm/send/abc", p256dh+"=", auth+"=="); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ endpoint, p256dh, auth string }{
		{"http://fcm.googleapis.com/fcm/send/abc", p256dh, auth},
		{"fcm.googleapis.com/abc", p256dh, auth},
		{"https://fcm.googleapis.com/abc", Encode([]byte("short")), auth},
		{"https://fcm.googleapis.com/abc", p256dh, Encode(make([]byte, 8))},
	} {
		if _, err := ParseSubscription(c.endpoint, c.p256dh, c.auth); err == nil {
			t.Errorf("accepted %+v", c)
		}
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func answer(status int, hdr map[string]string, body string) *http.Response {
	h := http.Header{}
	for k, v := range hdr {
		h.Set(k, v)
	}
	return &http.Response{StatusCode: status, Header: h, Body: io.NopCloser(strings.NewReader(body))}
}

func TestSenderRequestAndOutcomes(t *testing.T) {
	k, _ := GenerateVAPIDKey()
	ua, _ := ecdh.P256().GenerateKey(rand.Reader)
	auth := make([]byte, 16)
	sub := Subscription{Endpoint: "https://fcm.googleapis.com/fcm/send/abc", P256dh: ua.PublicKey().Bytes(), Auth: auth}
	now := time.Date(2026, 10, 14, 9, 5, 0, 0, time.UTC)
	var seen []*http.Request
	status := 201
	hdr := map[string]string{"Location": "https://fcm.googleapis.com/m/1"}
	body := ""
	s := &Sender{Key: k, Contact: "mailto:support@acme.io", Now: func() time.Time { return now },
		HTTP: &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
			seen = append(seen, r)
			return answer(status, hdr, body), nil
		})}}
	res, err := s.Send(context.Background(), sub, Message{Payload: []byte(`{"title":"Feature flags at scale"}`), TTL: 4 * time.Hour, Topic: TopicFor("new-post")})
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != Accepted || res.Location != "https://fcm.googleapis.com/m/1" {
		t.Fatalf("result %+v", res)
	}
	r := seen[0]
	if r.Method != http.MethodPost || r.Header.Get("TTL") != "14400" || r.Header.Get("Content-Encoding") != "aes128gcm" ||
		r.Header.Get("Urgency") != "normal" || len(r.Header.Get("Topic")) != 32 {
		t.Fatalf("headers %v", r.Header)
	}
	if _, _, err := Verify(r.Header.Get("Authorization"), "https://fcm.googleapis.com", now); err != nil {
		t.Fatalf("authorization: %v", err)
	}
	b, _ := io.ReadAll(r.Body)
	if got, err := Decrypt(ua, auth, b); err != nil || string(got) != `{"title":"Feature flags at scale"}` {
		t.Fatalf("payload %q %v", got, err)
	}
	// The token is reused within the hour.
	_, _ = s.Send(context.Background(), sub, Message{Payload: []byte("x")})
	if seen[0].Header.Get("Authorization") != seen[1].Header.Get("Authorization") {
		t.Fatal("token not reused")
	}
	now = now.Add(61 * time.Minute)
	_, _ = s.Send(context.Background(), sub, Message{Payload: []byte("x")})
	if seen[1].Header.Get("Authorization") == seen[2].Header.Get("Authorization") {
		t.Fatal("token reused after an hour")
	}

	for _, c := range []struct {
		status int
		hdr    map[string]string
		body   string
		want   Outcome
		retry  time.Duration
		detail string
	}{
		{410, nil, `{"reason":"Unregistered"}`, Gone, 0, "410 Gone · Unregistered"},
		{404, nil, "", Gone, 0, "404 Not Found"},
		{413, nil, "", TooLarge, 0, "413 Request Entity Too Large"},
		{429, map[string]string{"Retry-After": "120"}, "", RateLimited, 2 * time.Minute, "429 Too Many Requests"},
		{503, map[string]string{"Retry-After": "30"}, "", Unavailable, 30 * time.Second, "503 Service Unavailable"},
		{403, nil, `{"code":403,"errno":109,"error":"Forbidden","message":"Request did not validate"}`, Rejected, 0, "403 Forbidden · Request did not validate"},
	} {
		status, hdr, body = c.status, c.hdr, c.body
		res, err := s.Send(context.Background(), sub, Message{Payload: []byte("x")})
		if err != nil {
			t.Fatal(err)
		}
		if res.Outcome != c.want || res.RetryAfter != c.retry || res.Detail != c.detail {
			t.Errorf("%d: %+v", c.status, res)
		}
		if res.Retryable() != (c.want == RateLimited || c.want == Unavailable) {
			t.Errorf("%d: retryable %v", c.status, res.Retryable())
		}
	}
	// Nothing answering is Unavailable, not an error.
	s.HTTP = &http.Client{Transport: roundTripper(func(*http.Request) (*http.Response, error) { return nil, errors.New("connection refused") })}
	res, err = s.Send(context.Background(), sub, Message{Payload: []byte("x")})
	if err != nil || res.Outcome != Unavailable || res.Status != 0 {
		t.Fatalf("unreachable: %+v %v", res, err)
	}
}

func TestTopicFor(t *testing.T) {
	if TopicFor("") != "" {
		t.Fatal("empty tag")
	}
	a, b := TopicFor("new-post"), TopicFor("new-post")
	if a != b || len(a) != 32 || strings.ContainsAny(a, "+/=") {
		t.Fatalf("topic %q", a)
	}
	if TopicFor("other") == a {
		t.Fatal("collision")
	}
}
