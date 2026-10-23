package instance

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/auditlog"
	"github.com/hostrepute/reputecast/internal/i18n"
)

func TestVersions(t *testing.T) {
	for _, c := range []struct {
		tag, installed string
		want           bool
	}{
		{"v1.5.0", "v1.4.2", true},
		{"v1.4.2", "v1.4.2", false},
		{"v1.4.10", "v1.4.9", true},
		{"v1.4.2", "v1.5.0", false},
		{"v1.5.0", "v1.5.0-rc.1", true},
		{"v1.5.0-rc.2", "v1.5.0", false},
		{"v1.5.0", "dev", false},
		{"nightly", "v1.4.2", false},
	} {
		if got := newer(c.tag, c.installed); got != c.want {
			t.Errorf("newer(%s, %s) = %v", c.tag, c.installed, got)
		}
	}
}

func TestChangelog(t *testing.T) {
	notes, note := changelog("## What's new\n- **Send approvals**: two approvers\n* Passkeys as a [second factor](https://x)\n\n" +
		"- a\n- b\n- c\n- d\n- e (dropped)\n\nNo database migration downtime.")
	if len(notes) != 6 || notes[0] != "Send approvals: two approvers" || notes[1] != "Passkeys as a second factor" || note != "No database migration downtime." {
		t.Fatalf("notes %q, note %q", notes, note)
	}
}

func TestPublicURL(t *testing.T) {
	for in, want := range map[string]string{
		"https://reputecast.acme.io/": "https://reputecast.acme.io",
		" http://localhost:8080 ":     "http://localhost:8080",
		"https://acme.io/mail/":       "https://acme.io/mail",
	} {
		if got, err := normalizePublicURL(context.Background(), in); err != nil || got != want {
			t.Errorf("%q: %q %v", in, got, err)
		}
	}
	for _, bad := range []string{"reputecast.acme.io", "ftp://acme.io", "https://acme.io/?x=1", "https://acme.io/#x", ""} {
		if _, err := normalizePublicURL(context.Background(), bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestLogo(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 32))
	img.Set(1, 1, color.RGBA{R: 37, G: 99, B: 235, A: 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	svg, err := logoSVG(i18n.Default, "data:image/png;base64,"+base64.StdEncoding.EncodeToString(buf.Bytes()))
	if err != nil || !strings.HasPrefix(svg, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"`) || !strings.Contains(svg, `y="16"`) {
		t.Fatalf("png: %v %.120s", err, svg)
	}
	for _, bad := range []string{
		`<svg xmlns="http://www.w3.org/2000/svg"><style>@import url(https://evil.example/x.css);</style></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><rect style="fill:url( 'https://evil.example/p.svg#a')"/></svg>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><div/></foreignObject></svg>`,
		`<html><body/></html>`,
		`<svg xmlns="http://www.w3.org/2000/svg"><rect></svg>`,
	} {
		if _, err := logoSVG(i18n.Default, bad); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
	if _, err := logoSVG(i18n.Default, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><defs><linearGradient id="g"/></defs><rect fill="url(#g)" width="10" height="10"/></svg>`); err != nil {
		t.Errorf("internal url(#g) refused: %v", err)
	}
}

func TestSignInLabels(t *testing.T) {
	for _, c := range []struct{ method, provider, want string }{
		{"password+totp", "", "Password + TOTP"},
		{"sso", "Google Workspace", "Google Workspace SSO"},
		{"sso+recovery", "Okta", "Okta SSO + recovery code"},
		{"passkey", "", "Passkey"},
		{"password+passkey", "", "Password + passkey"},
		{"github", "", "GitHub"},
	} {
		if got := auditlog.SignInMethod(i18n.Default, c.method, c.provider); got != c.want {
			t.Errorf("methodLabel(%s) = %s", c.method, got)
		}
	}
	if got := auditlog.SignInFailed(i18n.Default, "Password", 2); got != "Password · 2 failed" {
		t.Errorf("failed: %s", got)
	}
	if got := auditlog.SignInFailed(i18n.For(i18n.Vietnamese), auditlog.SignInMethod(i18n.For(i18n.Vietnamese), "password+totp", ""), 3); got != "Mật khẩu + TOTP · 3 lần thất bại" {
		t.Errorf("Vietnamese: %s", got)
	}
}

// Why the last update check failed is stored in English and reads in the
// admin's language; what GitHub or the network said stays as it was.
func TestCheckErrorIn(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	for stored, want := range map[string]string{
		"GitHub's answer had no release.":             "Câu trả lời của GitHub không có bản phát hành nào.",
		"GitHub answered 503 Service Unavailable.":    "GitHub trả về 503 Service Unavailable.",
		"GitHub didn't answer: dial tcp: i/o timeout": "GitHub không trả lời: dial tcp: i/o timeout",
		"the release feed answered something unknown": "the release feed answered something unknown",
	} {
		if got := checkErrorIn(vi, stored); got != want {
			t.Errorf("%q: %q, want %q", stored, got, want)
		}
		if got := checkErrorIn(i18n.Default, stored); got != stored {
			t.Errorf("English %q: %q", stored, got)
		}
	}
	if got := mUpdateStatus.Text(i18n.Default, i18n.Args{"status": "503", "statusText": "Service Unavailable"}); got != "GitHub answered 503 Service Unavailable." {
		t.Errorf("stored English: %q", got)
	}
}
