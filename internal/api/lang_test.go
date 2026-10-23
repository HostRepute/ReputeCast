package api_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// "Campaign not found." reads in the request's language, from any package.
func TestNotFoundInVietnamese(t *testing.T) {
	env := apptest.New(t)
	linh := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	ws := env.CreateWorkspace("Acme Newsletter", linh.ID)
	path := "/workspaces/" + ws.Slug + "/campaigns/" + uuid.Must(uuid.NewV7()).String()

	en := env.Client()
	en.SignIn("linh@acme.io", "")
	if p := en.Do("GET", path, nil).MustStatus(t, 404).Problem(t); p["detail"] != "Campaign not found." || p["code"] != "not_found" {
		t.Fatalf("English: %v", p)
	}

	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignIn("linh@acme.io", "")
	if p := vi.Do("GET", path, nil).MustStatus(t, 404).Problem(t); p["detail"] != "Không tìm thấy chiến dịch." || p["code"] != "not_found" {
		t.Fatalf("Vietnamese: %v", p)
	}
}
