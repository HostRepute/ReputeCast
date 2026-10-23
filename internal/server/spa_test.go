package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// The app's line for browsers without JavaScript reads in the request's
// language; the rest of index.html is served as built.
func TestSPANoscriptFollowsTheRequestsLanguage(t *testing.T) {
	page := `<!doctype html><html lang="en"><body><div id="root"></div>` + noscript + `</body></html>`
	spa := NewSPA(fstest.MapFS{"index.html": {Data: []byte(page)}})
	for _, tc := range []struct{ accept, want string }{
		{"", noscript},
		{"de-DE,de;q=0.9", noscript},
		{"vi-VN,vi;q=0.9,en;q=0.8", "<noscript>ReputeCast cần JavaScript. Các trang công khai (đăng ký, tùy chọn, lưu trữ) vẫn dùng được khi không có JavaScript.</noscript>"},
	} {
		req := httptest.NewRequest(http.MethodGet, "/w/acme-newsletter/campaigns", nil)
		if tc.accept != "" {
			req.Header.Set("Accept-Language", tc.accept)
		}
		rec := httptest.NewRecorder()
		spa.ServeHTTP(rec, req)
		body := rec.Body.String()
		if !strings.Contains(body, tc.want) || !strings.HasPrefix(body, `<!doctype html><html lang="en"><body><div id="root"></div>`) {
			t.Errorf("%q: %s", tc.accept, body)
		}
		if rec.Header().Get("Vary") != "Accept-Language" {
			t.Errorf("%q: Vary %q", tc.accept, rec.Header().Get("Vary"))
		}
	}
}
