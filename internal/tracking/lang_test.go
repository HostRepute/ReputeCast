package tracking

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Every verdict tracking stores reads in Vietnamese; names stay as they are,
// and English readers get the stored English.
func TestDiagnosticIn(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	cases := map[string]string{
		reasonApple:                                 "Apple Mail Privacy Protection tải trước",
		reasonHead:                                  "Yêu cầu HEAD từ một trình quét liên kết",
		"Security scanner (Barracuda)":              "Trình quét bảo mật (Barracuda)",
		"Security scanner (a script)":               "Trình quét bảo mật (một tập lệnh)",
		"Link scanner (Microsoft Defender)":         "Trình quét liên kết (Microsoft Defender)",
		"Link scanner (a link preview)":             "Trình quét liên kết (một bản xem trước liên kết)",
		"Chrome on macOS":                           "Chrome trên macOS",
		"Apple Mail on iOS":                         "Apple Mail trên iOS",
		"A mail client on Android":                  "Một ứng dụng email trên Android",
		clientUnknown:                               "Ứng dụng email không xác định",
		"Gmail":                                     "Gmail",
		"550 5.1.1 The email account doesn't exist": "550 5.1.1 The email account doesn't exist",
	}
	for en, want := range cases {
		if got := DiagnosticIn(vi, en); got != want {
			t.Errorf("%q: %q, want %q", en, got, want)
		}
		if got := DiagnosticIn(i18n.Default, en); got != en {
			t.Errorf("English %q: %q", en, got)
		}
	}
	// Every verdict the classifiers give is known.
	signals := []Signal{
		{Method: "HEAD", UserAgent: "x", SinceSend: time.Hour}, {SinceSend: time.Hour}, {UserAgent: "Mozilla/5.0", SinceSend: time.Hour},
		{UserAgent: gmailPx, SinceSend: time.Second}, {UserAgent: yahooPx, SinceSend: time.Second}, {UserAgent: "curl/8.0", SinceSend: time.Hour},
		{UserAgent: "HeadlessChrome", SinceSend: time.Hour}, {UserAgent: chromeMac, SinceSend: time.Second}, {UserAgent: "Linux x86_64", SinceSend: time.Hour},
		{UserAgent: "Something", SinceSend: time.Hour},
	}
	for _, s := range signals {
		for _, v := range []Verdict{ClassifyOpen(s), ClassifyClick(s, false), ClassifyClick(Signal{UserAgent: chromeMac, SinceSend: time.Hour}, true)} {
			en := v.Reason
			if !v.Machine {
				en = v.Client
			}
			if got := DiagnosticIn(vi, en); got == en && en != "Gmail" && en != "Yahoo Mail" && en != "Outlook" {
				t.Errorf("not translated: %q", en)
			}
		}
	}
}

// A broken tracked link answers in the visitor's browser language.
func TestGoneInVietnamese(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/t/c/x", nil)
	r.Header.Set("Accept-Language", "vi-VN,vi;q=0.9")
	w := httptest.NewRecorder()
	gone(w, r)
	if w.Code != http.StatusNotFound || !strings.HasPrefix(w.Body.String(), "Liên kết này không còn hợp lệ.") {
		t.Fatalf("Vietnamese: %d %q", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	gone(w, httptest.NewRequest(http.MethodGet, "/t/c/x", nil))
	if w.Body.String() != "This link is no longer valid. If you copied it, check that the whole address came along.\n" {
		t.Fatalf("English: %q", w.Body.String())
	}
}
