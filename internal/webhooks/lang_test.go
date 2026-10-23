package webhooks_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Settings › Webhooks reads the events' descriptions and the problems in
// the app's language; API clients that don't ask get English. What goes
// to an endpoint stays English.
func TestWebhooksInVietnamese(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Olive Owner"})
	w := env.CreateWorkspace("Hooks", owner.ID)
	base := "/workspaces/" + w.Slug + "/webhooks"
	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignIn("owner@example.com", "")
	en := env.Client()
	en.SignIn("owner@example.com", "")

	type list struct {
		Events []struct{ Name, Description string }
	}
	describe := func(l list, name string) string {
		for _, e := range l.Events {
			if e.Name == name {
				return e.Description
			}
		}
		return ""
	}
	var lv, le list
	vi.Do(http.MethodGet, base, nil).MustStatus(t, 200).JSON(t, &lv)
	en.Do(http.MethodGet, base, nil).MustStatus(t, 200).JSON(t, &le)
	if got := describe(lv, "email.bounced"); got != "Thư trả lại vĩnh viễn hoặc tạm thời, kèm mã gốc của nhà cung cấp." {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := describe(le, "email.bounced"); got != "Hard or soft bounce, with the provider's original code." {
		t.Errorf("English: %q", got)
	}
	for _, e := range le.Events {
		if e.Description == "" {
			t.Errorf("%s has no description", e.Name)
		}
	}

	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	env.App.Cfg.Webhooks.AllowPrivateNetworks = true
	body := map[string]any{"name": "Hook", "url": srv.URL + "/hook", "events": []string{"nope.event"}}
	if p := vi.Do(http.MethodPost, base, body).MustStatus(t, 422).Problem(t); p["detail"] != `Sự kiện "nope.event" không xác định.` {
		t.Errorf("unknown event: %v", p["detail"])
	}
	if p := en.Do(http.MethodPost, base, body).MustStatus(t, 422).Problem(t); p["detail"] != `Unknown event "nope.event".` {
		t.Errorf("unknown event in English: %v", p["detail"])
	}
}
