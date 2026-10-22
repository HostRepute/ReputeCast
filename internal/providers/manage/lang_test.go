package manage_test

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
)

// Sending providers reads in each reader's language: its own words now,
// and what the health job stored while nobody was reading.
func TestProvidersReadInTheReadersLanguage(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := demoWorkspace(t, env)
	fp := addFake(t, env, w, nil)
	ses := providerByAdapter(t, env, "ses")

	// The health job marks the fake provider down, in no reader's language.
	svc := service(env, time.Now)
	setFakeErr(&providers.CallError{Op: "GET /quota", Status: 503, Code: "ServiceUnavailable", Temporary: true, Cause: providers.ErrUnavailable})
	defer setFakeErr(nil)
	for range 2 {
		var err error
		if fp, err = svc.Check(ctx, fp, true); err != nil {
			t.Fatal(err)
		}
	}
	if fp.Status != state.Down {
		t.Fatalf("status = %s", fp.Status)
	}

	type overview struct {
		Providers []struct {
			Adapter, Account string
			Capabilities     []struct{ Label string }
		}
		Outage *struct {
			Incidents []struct{ Kind, Title string }
		}
	}
	read := func(lang string) overview {
		c := env.Client()
		c.SignInDemo("linh@acme.io")
		if lang != "" {
			c.Header = http.Header{"Accept-Language": {lang}}
		}
		var o overview
		c.Do("GET", ws+"/providers", nil).MustStatus(t, 200).JSON(t, &o)
		return o
	}
	title := func(o overview) string {
		if o.Outage != nil {
			for _, i := range o.Outage.Incidents {
				if i.Kind == "provider_down" {
					return i.Title
				}
			}
		}
		t.Fatalf("no provider_down incident: %+v", o.Outage)
		return ""
	}
	chips := func(o overview, adapter string) []string {
		var out []string
		for _, p := range o.Providers {
			if p.Adapter == adapter {
				for _, c := range p.Capabilities {
					out = append(out, c.Label)
				}
			}
		}
		return out
	}

	en, vi := read(""), read("vi")
	if got := title(en); got != "Fake ESP: 503 ServiceUnavailable on GET /quota · health check failed." {
		t.Errorf("English incident: %q", got)
	}
	if got := title(vi); got != "Fake ESP: 503 ServiceUnavailable ở GET /quota · kiểm tra sức khỏe thất bại." {
		t.Errorf("Vietnamese incident: %q", got)
	}
	if !slices.Contains(chips(en, "ses"), "Bulk marketing allowed") || !slices.Contains(chips(vi, "ses"), "Cho phép tiếp thị hàng loạt") ||
		!slices.Contains(chips(vi, "ses"), "Giá · 0,10 US$ cho mỗi 1.000 email") {
		t.Errorf("chips: %q / %q", chips(en, "ses"), chips(vi, "ses"))
	}
	// The demo's account standing was stored before the messages were kept:
	// it reads in English for everyone.
	for _, p := range vi.Providers {
		if p.Adapter == "ses" && p.Account != "Production · no enforcement" {
			t.Errorf("SES account: %q", p.Account)
		}
	}

	// A problem reads in the request's language.
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	c.Header = http.Header{"Accept-Language": {"vi"}}
	var problem struct{ Detail string }
	c.Do("DELETE", ws+"/providers/"+ses.ID.String(), nil).MustStatus(t, 409).JSON(t, &problem)
	if !strings.Contains(problem.Detail, " vẫn đang gửi luồng ") || !strings.HasSuffix(problem.Detail, "Hãy đổi định tuyến trước, rồi ngắt kết nối nó.") {
		t.Errorf("problem: %q", problem.Detail)
	}
}
