package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The rate limits' problems read in the language the request asks for;
// without one, in English.
func TestRateLimitInVietnamese(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := RateLimits{AuthPerMinute: 1, APIPerMinute: 1}.Middleware()(ok)
	detail := func(path, ip, lang string) string {
		var last *httptest.ResponseRecorder
		for range 2 {
			r := httptest.NewRequest(http.MethodPost, path, nil)
			r.RemoteAddr = ip + ":1234"
			if lang != "" {
				r.Header.Set("Accept-Language", lang)
			}
			last = httptest.NewRecorder()
			h.ServeHTTP(last, r)
		}
		var p struct{ Detail string }
		_ = json.Unmarshal(last.Body.Bytes(), &p)
		return p.Detail
	}
	if d := detail("/api/v1/auth/password", "198.51.100.7", "vi"); d != "Mạng của bạn đã thử đăng nhập quá nhiều lần. Hãy đợi một phút rồi thử lại." {
		t.Errorf("sign-in: %q", d)
	}
	if d := detail("/api/v1/workspaces/acme/campaigns", "198.51.100.8", "vi-VN,vi;q=0.9"); d != "Đã chạm giới hạn tần suất. Yêu cầu sẽ tiếp tục trong vòng một phút; endpoint theo lô nhận tối đa 1.000 mục mỗi lần gọi." {
		t.Errorf("API: %q", d)
	}
	if d := detail("/api/v1/workspaces/acme/campaigns", "198.51.100.9", ""); d != "Rate limit reached. Requests resume within a minute; batch endpoints accept up to 1,000 items per call." {
		t.Errorf("English: %q", d)
	}
}
