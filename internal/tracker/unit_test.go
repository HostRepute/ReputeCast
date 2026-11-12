package tracker

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/clicktoken"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// rc.js stays small (≤5 KB gzipped), as form.js is bounded, and is served
// so every site may load it.
func TestScriptIsSmallAndServed(t *testing.T) {
	var b bytes.Buffer
	z, _ := gzip.NewWriterLevel(&b, gzip.BestCompression)
	_, _ = z.Write(Script())
	_ = z.Close()
	if n := b.Len(); n < 1<<10 || n > 5<<10 {
		t.Fatalf("rc.js is %d bytes gzipped, want 1–5 KB", n)
	}
	for _, want := range []string{"globalPrivacyControl === true", "data-wait-consent", "rc_vid", "SameSite=Lax", "keepalive: true", "'text/plain'", "pushState", "notifyWhenAvailable"} {
		if !strings.Contains(string(Script()), want) {
			t.Errorf("rc.js lacks %q", want)
		}
	}
	w := httptest.NewRecorder()
	serveScript(w, httptest.NewRequest(http.MethodGet, "/track/v1/rc.js", nil))
	h := w.Header()
	if w.Code != 200 || h.Get("Cache-Control") != "public, max-age=600" || h.Get("Cross-Origin-Resource-Policy") != "cross-origin" ||
		h.Get("X-Content-Type-Options") != "nosniff" || !strings.HasPrefix(h.Get("Content-Type"), "text/javascript") {
		t.Fatalf("served %d %v", w.Code, h)
	}
}

// The click decorator adds rc= to links to a tracked site's host, once,
// and leaves every other link as it is.
func TestDecorator(t *testing.T) {
	signer, err := linktoken.NewSigner(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{Links: signer}
	ws, contact, msg := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	s.origins.put(ws.String(), []string{"shop.acme.io", "localhost:3000"}, time.Now().Add(time.Hour))
	at := time.Date(2026, 11, 11, 9, 30, 0, 0, time.UTC)
	click := func(target string, who *uuid.UUID) string {
		return s.Decorate(context.Background(), tracking.ClickedLink{WorkspaceID: ws, MessageID: msg, ContactID: who, Target: target, At: at})
	}
	got := click("https://shop.acme.io/products/hoodie?utm_source=acme#size", &contact)
	base, frag, _ := strings.Cut(got, "#")
	if frag != "size" || !strings.HasPrefix(base, "https://shop.acme.io/products/hoodie?utm_source=acme&rc=") {
		t.Fatalf("decorated: %s", got)
	}
	tok := strings.TrimPrefix(base, "https://shop.acme.io/products/hoodie?utm_source=acme&rc=")
	c, err := clicktoken.Verify(signer, tok)
	if err != nil || c.ContactID != contact || c.MessageID != msg || c.WorkspaceID != ws || !c.At.Equal(at) {
		t.Fatalf("token: %+v %v", c, err)
	}
	for _, keep := range []string{
		"https://shop.acme.io/a?rc=already",  // a store's tagging added it
		"https://www.shop.acme.io/a",         // another host
		"https://blog.acme.io/a",             // not a tracked site
		"mailto:sam@acme.io",                 // not a web link
		"https://shop.acme.io.evil.example/", // a lookalike
	} {
		if got := click(keep, &contact); got != keep {
			t.Errorf("%s became %s", keep, got)
		}
	}
	if got := click("https://shop.acme.io/a", nil); got != "https://shop.acme.io/a" {
		t.Errorf("a test send's click got a token: %s", got)
	}
	if got := click("http://localhost:3000/", &contact); !strings.HasPrefix(got, "http://localhost:3000/?rc=") {
		t.Errorf("a port: %s", got)
	}
}

// Events keep their path and utm_* only, the referrer its domain, and
// their fields within bounds.
func TestEventChecks(t *testing.T) {
	now := time.Now()
	const origin = "https://shop.acme.io"
	ok := func(ev collectEvent) item {
		t.Helper()
		it, good := eventOf(ev, origin, now)
		if !good {
			t.Fatalf("refused %+v", ev)
		}
		return it
	}
	it := ok(collectEvent{T: "page", URL: "https://shop.acme.io/p/hoodie?utm_source=news&email=sam%40x.io&utm_medium=email#top", Ref: "https://www.google.com/search?q=x"})
	if it.path != "/p/hoodie?utm_source=news&utm_medium=email" || it.ref != "google.com" || it.kind != KindPage {
		t.Fatalf("page: %+v", it)
	}
	if it := ok(collectEvent{T: "page", URL: origin + "/b", Ref: origin + "/a"}); it.ref != "" {
		t.Fatalf("own referrer kept: %q", it.ref)
	}
	if it := ok(collectEvent{T: "product", URL: origin + "/p", ID: []byte(`"gid://shopify/Product/8841203"`)}); it.extID != "8841203" {
		t.Fatalf("gid: %q", it.extID)
	}
	if it := ok(collectEvent{T: "cart", URL: origin + "/p", ID: []byte(`77`), Qty: 3}); it.extID != "77" || it.qty != 3 {
		t.Fatalf("cart: %+v", it)
	}
	if it := ok(collectEvent{T: "track", URL: origin + "/", Name: "quiz_done", Props: []byte(`{"score":3}`), At: float64(now.Add(-time.Hour).UnixMilli())}); !it.at.Equal(now.Add(-skew)) {
		t.Fatalf("an hour-old event is kept at %v", it.at)
	}
	long := strings.Repeat("a", 600)
	if it := ok(collectEvent{T: "page", URL: origin + "/" + long}); len(it.path) != maxPath {
		t.Fatalf("path %d", len(it.path))
	}
	for _, bad := range []collectEvent{
		{T: "page", URL: "https://evil.example/"},
		{T: "page", URL: "http://shop.acme.io/"},
		{T: "spy", URL: origin + "/"},
		{T: "product", URL: origin + "/"},
		{T: "product", URL: origin + "/", ID: []byte(`"a b"`)},
		{T: "track", URL: origin + "/", Name: "Bad-Name"},
		{T: "track", URL: origin + "/", Name: "ok", Props: []byte(`[1]`)},
		{T: "track", URL: origin + "/", Name: "ok", Props: []byte(`{"x":"` + strings.Repeat("y", 2100) + `"}`)},
	} {
		if _, good := eventOf(bad, origin, now); good {
			t.Errorf("kept %+v", bad)
		}
	}
}

// A site's server signs identify calls as the WordPress plugin does
// (integrations/wordpress tests/Unit/TrackingTest.php has the same vector).
func TestSignatureVector(t *testing.T) {
	if got := Signature("rcid_0123456789abcdefghijklmnopqrstuvwxyzABCDEFG", " Sam.Rivera@Outlook.com "); got != "12a36b417d210c68b4e1cc84673dcc2e19be54e3454c68c9317271f6fcddf6df" {
		t.Fatalf("signature %s", got)
	}
}
