package authn

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestPasswordHashVerify(t *testing.T) {
	h, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$") {
		t.Fatalf("unexpected format %s", h)
	}
	ok, err := VerifyPassword("correct horse battery staple", h)
	if err != nil || !ok {
		t.Fatalf("verify: %v %v", ok, err)
	}
	if ok, _ := VerifyPassword("wrong", h); ok {
		t.Fatal("wrong password verified")
	}
	if NeedsRehash(h) {
		t.Fatal("fresh hash should not need rehash")
	}
	if _, err := VerifyPassword("x", "$bcrypt$nope"); err == nil {
		t.Fatal("malformed hash should error")
	}
}

func TestPasswordProblem(t *testing.T) {
	if p := PasswordProblem("short", 12); p == "" {
		t.Fatal("short password accepted")
	}
	if p := PasswordProblem("a long enough passphrase", 12); p != "" {
		t.Fatalf("good password rejected: %s", p)
	}
}

func TestTokens(t *testing.T) {
	tok, h := NewToken()
	if len(tok) < 40 || !EqualHash(h, HashToken(tok)) {
		t.Fatal("token hash mismatch")
	}
	key, kh, masked := NewAPIKey(APIKeyPrefix)
	if !regexp.MustCompile(`^rc_live_[0-9a-f]{64}$`).MatchString(key) {
		t.Fatalf("bad key %s", key)
	}
	if !EqualHash(kh, HashToken(key)) || !strings.HasPrefix(masked, "rc_live_") || !strings.Contains(masked, "…") {
		t.Fatalf("bad mask %s", masked)
	}
	if !LooksLikeAPIKey(key) || LooksLikeAPIKey("abc") {
		t.Fatal("LooksLikeAPIKey wrong")
	}
}

func TestTOTP(t *testing.T) {
	s, err := NewTOTP("Acme ReputeCast", "linh@acme.io")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	code, _ := totp.GenerateCode(s.Secret, now)
	if !VerifyTOTP(s.Secret, code[:3]+" "+code[3:], now) {
		t.Fatal("valid code rejected")
	}
	if VerifyTOTP(s.Secret, "000000", now.Add(-time.Hour)) && code != "000000" {
		t.Fatal("stale code accepted")
	}
	if g := GroupSecret("JBSWY3DPEHPK3PXP"); g != "JBSW Y3DP EHPK 3PXP" {
		t.Fatalf("group %q", g)
	}
}

func TestRecoveryCodes(t *testing.T) {
	codes := NewRecoveryCodes(10)
	re := regexp.MustCompile(`^[0-9a-hj-km-np-tv-z]{4}-[0-9a-hj-km-np-tv-z]{4}-[0-9a-hj-km-np-tv-z]{4}$`)
	seen := map[string]bool{}
	for _, c := range codes {
		if !re.MatchString(c) || seen[c] {
			t.Fatalf("bad or duplicate code %s", c)
		}
		seen[c] = true
	}
	if NormalizeRecoveryCode(" B4XK 9Q2M 7HDN ") != "b4xk-9q2m-7hdn" {
		t.Fatal("normalize failed")
	}
}

func TestBreachChecker(t *testing.T) {
	// SHA-1("password") = 5BAA61E4C9B93F3F0682250B6CF8331B7EE68FD8
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/5BAA6") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		fmt.Fprintln(w, "1E4C9B93F3F0682250B6CF8331B7EE68FD8:9545824")
		fmt.Fprintln(w, "0000000000000000000000000000000000A:0")
	}))
	defer srv.Close()
	c := &BreachChecker{Client: srv.Client(), BaseURL: srv.URL + "/range/"}
	if hit, err := c.Breached(context.Background(), "password"); err != nil || !hit {
		t.Fatalf("expected breach hit: %v %v", hit, err)
	}
}
