package accounts_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/store"
)

// withTOTP gives a person an authenticator and returns its secret.
func withTOTP(t *testing.T, env *apptest.Env, email string) string {
	t.Helper()
	ctx := context.Background()
	u, err := env.App.Q.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	sec, err := authn.NewTOTP("Acme ReputeCast", email)
	if err != nil {
		t.Fatal(err)
	}
	enc, err := env.App.Secrets.SealString(sec.Secret, "user:"+u.ID.String()+":totp")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := env.App.Q.SetUserTOTP(ctx, store.SetUserTOTPParams{ID: u.ID, TotpSecretEnc: enc, TotpEnabledAt: &now}); err != nil {
		t.Fatal(err)
	}
	return sec.Secret
}

// codeAt is the authenticator code of a moment.
func codeAt(t *testing.T, secret string, at time.Time) string {
	t.Helper()
	code, err := totp.GenerateCode(secret, at)
	if err != nil {
		t.Fatal(err)
	}
	return code
}

// wrongCode is a code no step around now accepts.
func wrongCode(t *testing.T, secret string) string {
	t.Helper()
	now := time.Now()
	taken := map[string]bool{}
	for _, d := range []time.Duration{-time.Minute, -30 * time.Second, 0, 30 * time.Second, time.Minute} {
		taken[codeAt(t, secret, now.Add(d))] = true
	}
	for _, c := range []string{"000000", "111111", "222222"} {
		if !taken[c] {
			return c
		}
	}
	t.Fatal("no wrong code")
	return ""
}

// passwordStep signs in with the password and returns the two-factor challenge.
func passwordStep(t *testing.T, c *apptest.Client, email string) string {
	t.Helper()
	var step map[string]any
	c.Do("POST", "/auth/password", map[string]any{"email": email, "password": "correct horse battery staple"}).MustStatus(t, 200).JSON(t, &step)
	if step["status"] != "two_factor" {
		t.Fatalf("password step: %v", step)
	}
	return step["challenge"].(string)
}

func twoFactor(c *apptest.Client, challenge, method, code string) *apptest.Response {
	return c.Do("POST", "/auth/two-factor", map[string]any{"challenge": challenge, "method": method, "code": code})
}

// Wrong second-factor codes count toward the email's sign-in lockout, which
// outlasts every challenge: knowing the password no longer buys five fresh
// guesses per sign-in.
func TestWrongCodesLockSignInAcrossChallenges(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "aisha@acme.io"})
	secret := withTOTP(t, env, "aisha@acme.io")
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE instance_settings SET lockout_attempts = 6, lockout_minutes = 15`); err != nil {
		t.Fatal(err)
	}
	c := env.Client()
	spare := passwordStep(t, c, "aisha@acme.io")
	first := passwordStep(t, c, "aisha@acme.io")
	wrong := wrongCode(t, secret)
	for i := 1; i <= 4; i++ {
		p := twoFactor(c, first, "totp", wrong).MustStatus(t, 401).Problem(t)
		if p["code"] != "invalid_code" || p["meta"].(map[string]any)["attemptsLeft"].(float64) != float64(5-i) {
			t.Fatalf("wrong code %d: %v", i, p)
		}
	}
	if p := twoFactor(c, first, "recovery", "aaaa-bbbb-cccc").MustStatus(t, 401).Problem(t); p["code"] != "challenge_expired" {
		t.Fatalf("fifth wrong code: %v", p)
	}
	// The password is right, so a new challenge comes, but the sixth wrong
	// code of the email locks sign-in.
	second := passwordStep(t, c, "aisha@acme.io")
	if p := twoFactor(c, second, "totp", wrong).MustStatus(t, 429).Problem(t); p["code"] != "locked" {
		t.Fatalf("sixth wrong code: %v", p)
	}
	// Locked: the password and a challenge made before, even with the right code.
	c.Do("POST", "/auth/password", map[string]any{"email": "aisha@acme.io", "password": "correct horse battery staple"}).MustStatus(t, 429)
	twoFactor(c, spare, "totp", codeAt(t, secret, time.Now())).MustStatus(t, 429)
	if n := countRows(t, env, `SELECT count(*) FROM audit_events WHERE action = 'signin.failed' AND target_name = 'aisha@acme.io' AND metadata->>'method' LIKE 'password+%'`); n != 6 {
		t.Fatalf("%d wrong codes audited", n)
	}
}

// Only a whole sign-in clears the email's failures: a right password with
// wrong codes after it still counts toward the lockout.
func TestRightPasswordAloneDoesNotClearFailures(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "aisha@acme.io"})
	secret := withTOTP(t, env, "aisha@acme.io")
	c := env.Client()
	c.Do("POST", "/auth/password", map[string]any{"email": "aisha@acme.io", "password": "wrong"}).MustStatus(t, 401)
	challenge := passwordStep(t, c, "aisha@acme.io")
	twoFactor(c, challenge, "totp", wrongCode(t, secret)).MustStatus(t, 401)
	th, err := env.App.Q.GetSignInThrottle(context.Background(), "aisha@acme.io")
	if err != nil || th.Failures != 2 {
		t.Fatalf("failures after the password step: %+v %v", th, err)
	}
	twoFactor(c, challenge, "totp", codeAt(t, secret, time.Now())).MustStatus(t, 200)
	if _, err := env.App.Q.GetSignInThrottle(context.Background(), "aisha@acme.io"); err == nil {
		t.Fatal("a whole sign-in leaves the failures")
	}
}

// An authenticator code works once, even within its 30 seconds; the code
// that confirmed the authenticator doesn't sign in either.
func TestAuthenticatorCodeWorksOnce(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "aisha@acme.io"})
	c := env.Client()
	c.SignIn("aisha@acme.io", "")
	var setup map[string]any
	c.Do("POST", "/me/totp", nil).MustStatus(t, 200).JSON(t, &setup)
	secret := setup["secret"].(string)
	confirm := totpCode(t, secret)
	c.Do("POST", "/me/totp/confirm", map[string]any{"challenge": setup["challenge"], "code": confirm}).MustStatus(t, 200)

	other := env.Client()
	if p := twoFactor(other, passwordStep(t, other, "aisha@acme.io"), "totp", confirm).MustStatus(t, 401).Problem(t); p["code"] != "invalid_code" {
		t.Fatalf("the setup code signed in: %v", p)
	}
	next := codeAt(t, strings.ReplaceAll(secret, " ", ""), time.Now().Add(30*time.Second))
	twoFactor(other, passwordStep(t, other, "aisha@acme.io"), "totp", next).MustStatus(t, 200)
	third := env.Client()
	if p := twoFactor(third, passwordStep(t, third, "aisha@acme.io"), "totp", next).MustStatus(t, 401).Problem(t); p["code"] != "invalid_code" {
		t.Fatalf("a code replayed: %v", p)
	}
}

// Parallel guesses at one challenge can't exceed its five attempts: each is
// counted before its code is checked.
func TestParallelCodesStayWithinTheLimit(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "aisha@acme.io"})
	secret := withTOTP(t, env, "aisha@acme.io")
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE instance_settings SET lockout_attempts = 50`); err != nil {
		t.Fatal(err)
	}
	challenge := passwordStep(t, env.Client(), "aisha@acme.io")
	body, _ := json.Marshal(map[string]any{"challenge": challenge, "method": "totp", "code": wrongCode(t, secret)})
	codes := make([]string, 20)
	var wg sync.WaitGroup
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, env.Server.URL+"/api/v1/auth/two-factor", bytes.NewReader(body))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Sec-Fetch-Site", "same-origin")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}
			defer func() { _ = resp.Body.Close() }()
			var p struct {
				Code string `json:"code"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&p)
			codes[i] = p.Code
		}()
	}
	wg.Wait()
	checked := 0
	for _, c := range codes {
		if c == "invalid_code" {
			checked++
		}
	}
	th, err := env.App.Q.GetSignInThrottle(context.Background(), "aisha@acme.io")
	if err != nil || checked != 4 || th.Failures != 5 {
		t.Fatalf("%d codes checked, %d failures counted (%v): %v", checked, th.Failures, err, codes)
	}
}
