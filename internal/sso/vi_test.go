package sso_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/sso/ssotest"
)

// SSO reads in the admin's language: the presets' hints, the problems and
// what's missing before a provider can be enabled; a discovery test's error
// and a rejected sign-in's reason, both stored while the admin read
// English; and the audit log's lines about providers and sign-ins.
func TestSSOInVietnamese(t *testing.T) {
	env, admin := setup(t)
	idp := ssotest.New()
	defer idp.Close()
	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignInDemo("linh@acme.io")

	var list struct {
		Presets []struct {
			Key, Name, ShortName, DiscoveryHint, App string
		} `json:"presets"`
	}
	vi.Do("GET", "/admin/sso", nil).MustStatus(t, 200).JSON(t, &list)
	for _, p := range list.Presets {
		switch p.Key {
		case "google":
			if p.Name != "Google Workspace" || p.DiscoveryHint != "Nhà phát hành của Google luôn là https://accounts.google.com." {
				t.Errorf("google: %+v", p)
			}
		case "oidc":
			if p.Name != "OIDC chung" || p.ShortName != "IdP" || p.App != "phần cài đặt client của nhà cung cấp danh tính" {
				t.Errorf("generic: %+v", p)
			}
		}
	}

	var d detail
	vi.Do("POST", "/admin/sso", map[string]any{"preset": "entra"}).MustStatus(t, 201).JSON(t, &d)
	if !strings.HasPrefix(d.Name, "Microsoft Entra ID") || len(d.EnableBlockers) != 3 || d.EnableBlockers[0] != "Hãy chạy thành công một lần kiểm tra discovery cho URL nhà phát hành." {
		t.Fatalf("new provider: %+v", d)
	}
	if p := vi.Do("POST", "/admin/sso/"+d.ID+"/test", map[string]any{"workspace": "acme-newsletter"}).MustStatus(t, 409).Problem(t); !strings.HasPrefix(p["detail"].(string), "Trước khi đăng nhập thử: Hãy chạy thành công") {
		t.Errorf("test before setup: %v", p)
	}
	body := providerBody(env, t, idp)
	body["name"] = " "
	p := vi.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 422).Problem(t)
	if errs, _ := p["errors"].([]any); p["detail"] != "Hãy đặt cho nhà cung cấp một cái tên mà mọi người nhận ra trên trang đăng nhập." || len(errs) != 1 || errs[0].(map[string]any)["location"] != "body.name" {
		t.Errorf("name: %v", p)
	}

	// A failed discovery test, run while reading English.
	admin.Do("POST", "/admin/sso/"+d.ID+"/discovery", map[string]any{"issuerUrl": "http://127.0.0.1:1"}).MustStatus(t, 200)
	vi.Do("GET", "/admin/sso/"+d.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Discovery == nil || d.Discovery.Error != "Nhà phát hành không trả lời: kết nối bị từ chối." {
		t.Errorf("discovery: %+v", d.Discovery)
	}

	// A sign-in from a domain the provider doesn't allow.
	live := addProvider(t, env, admin, idp, func(b map[string]any) { b["allowedDomains"] = []string{"acme-labs.io"} })
	idp.Claims = map[string]any{"sub": "kim-1", "email": "kim@gmail.com", "email_verified": true, "amr": []string{"mfa"}}
	newBrowser(t, env).follow("/api/v1/auth/sso/" + live.ID + "/start")
	var got detail
	vi.Do("GET", "/admin/sso/"+live.ID, nil).MustStatus(t, 200).JSON(t, &got)
	if len(got.Rejections) != 1 || got.Rejections[0].Reason != "tên miền email không được phép với SSO Test IdP" {
		t.Errorf("rejections: %+v", got.Rejections)
	}
	admin.Do("GET", "/admin/sso/"+live.ID, nil).MustStatus(t, 200).JSON(t, &got)
	if len(got.Rejections) != 1 || got.Rejections[0].Reason != "email domain not allowed for Test IdP SSO" {
		t.Errorf("rejections in English: %+v", got.Rejections)
	}

	var audit struct {
		Items []struct{ Action, Summary, Reason string }
	}
	vi.Do("GET", "/admin/audit?workspace=instance&limit=50", nil).MustStatus(t, 200).JSON(t, &audit)
	seen := map[string]string{}
	for _, e := range audit.Items {
		if _, ok := seen[e.Action]; !ok {
			seen[e.Action] = e.Summary + " | " + e.Reason
		}
	}
	for action, want := range map[string]string{
		"sso.provider.updated":          "đã lưu bản nháp · đã đổi nhà phát hành · đã thay client secret",
		"sso.provider.discovery_tested": "discovery thành công · 4 endpoint · 1 khóa ký",
		"signin.rejected":               "tên miền email không được phép với SSO Test IdP",
	} {
		if !strings.Contains(seen[action], want) {
			t.Errorf("%s: %q, want %q", action, seen[action], want)
		}
	}
}
