package forms_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/forms"
)

// A member reads the forms' problems and the settings' findings in their
// app language; API clients that don't ask read English, as before.
func TestFormsInVietnamese(t *testing.T) {
	verify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"error-codes":["invalid-input-secret"]}`))
	}))
	defer verify.Close()
	prev := forms.SiteverifyURL
	forms.SiteverifyURL = verify.URL
	defer func() { forms.SiteverifyURL = prev }()

	env := apptest.New(t)
	env.SeedDemo()
	en := signedIn(env, "linh@acme.io")
	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignInDemo("linh@acme.io")

	if p := vi.Do("POST", api+"/forms", map[string]any{"name": "   ", "kind": "embedded"}).MustStatus(t, 422).Problem(t); p["detail"] != "Hãy đặt tên cho biểu mẫu." {
		t.Errorf("unnamed form, in Vietnamese: %v", p["detail"])
	}
	if p := en.Do("POST", api+"/forms", map[string]any{"name": "   ", "kind": "embedded"}).MustStatus(t, 422).Problem(t); p["detail"] != "Name the form." {
		t.Errorf("unnamed form, in English: %v", p["detail"])
	}
	dark := map[string]any{"publisherName": "Acme", "accent": "#4338CA", "background": "#333333", "font": "plex", "poweredBy": true}
	if p := vi.Do("PUT", api+"/public-pages/branding", dark).MustStatus(t, 422).Problem(t); p["detail"] != "#333333 quá tối cho nền trang: chữ ở chân trang cần độ tương phản 4,5:1 với nền. Hãy chọn màu sáng hơn." {
		t.Errorf("dark background, in Vietnamese: %v", p["detail"])
	}
	if p := en.Do("PUT", api+"/public-pages/branding", dark).MustStatus(t, 422).Problem(t); p["detail"] != "#333333 is too dark for the page background: the footer text needs 4.5:1 against it. Pick a lighter colour." {
		t.Errorf("dark background, in English: %v", p["detail"])
	}

	// Cloudflare's refusal is stored in English and read in each member's language.
	type turnstile struct {
		Turnstile struct{ Status, Error string } `json:"turnstile"`
	}
	var out turnstile
	vi.Do("PUT", api+"/public-pages/turnstile", map[string]any{"siteKey": "0x4AAAAAAAnewkey", "secret": "0xBAD"}).MustStatus(t, 200).JSON(t, &out)
	if out.Turnstile.Status != "error" || out.Turnstile.Error != "Cloudflare đã từ chối khóa bí mật (invalid-input-secret)" {
		t.Errorf("Turnstile, in Vietnamese: %+v", out.Turnstile)
	}
	en.Do("GET", api+"/public-pages", nil).MustStatus(t, 200).JSON(t, &out)
	if out.Turnstile.Error != "Cloudflare rejected the secret key (invalid-input-secret)" {
		t.Errorf("Turnstile, in English: %+v", out.Turnstile)
	}
}
