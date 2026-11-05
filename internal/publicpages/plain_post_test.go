package publicpages_test

// The copied plain HTML form posts to the hosted page without a token and
// says so with _plain=1. A script sends the same post
// with an allowed site as its Origin, so the flag never skips a check.

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestPlainPostIsChecked(t *testing.T) {
	env, ids := demo(t)
	env.App.Mail = &fakeMail{}
	// The footer form subscribes at once and doesn't ask for the Turnstile
	// check itself; the address limit is out of the way.
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE forms SET double_opt_in = false,
		spam_protection = spam_protection || '{"turnstile": false, "rate_limit_per_ip": 1000}' WHERE id = $1`, ids.footer); err != nil {
		t.Fatal(err)
	}
	path := "/p/acme-newsletter/subscribe/footer-mini-form"
	plain := func(email string) resp {
		return do(t, env, http.MethodPost, path, url.Values{"_plain": {"1"}, "email": {email}, "consent": {"1"}, "website": {""}}, hdr{"Origin": "https://acme.io"})
	}

	t.Run("with Turnstile, the check", func(t *testing.T) {
		must(t, plain("victim.one@fastmail.com"), 200, "One more step", `name="challenge"`)
		if _, ok := contactID(t, env, ids.ws, "victim.one@fastmail.com"); ok {
			t.Error("a plain post was saved before the check")
		}
	})

	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE public_page_settings SET turnstile_status = 'off', turnstile_site_key = '', turnstile_secret_enc = NULL
		WHERE workspace_id = $1`, ids.ws); err != nil {
		t.Fatal(err)
	}
	t.Run("without Turnstile, a confirmation email rather than a subscription", func(t *testing.T) {
		must(t, plain("victim.two@fastmail.com"), 200, "Check your inbox", "victim.two@fastmail.com")
		id, ok := contactID(t, env, ids.ws, "victim.two@fastmail.com")
		if !ok || membership(t, env, id, ids.digest) != "pending" {
			t.Fatal("a plain post subscribed the address without the confirmation click")
		}
	})
	t.Run("past its budget, the hosted page asks for one more step", func(t *testing.T) {
		must(t, plain("three@fastmail.com"), 200, "Check your inbox")
		must(t, plain("four@fastmail.com"), 200, "Check your inbox")
		r := plain("five@fastmail.com")
		must(t, r, 422, "One more step: confirm you're a person.", `value="five@fastmail.com"`)
		if _, ok := contactID(t, env, ids.ws, "five@fastmail.com"); ok {
			t.Fatal("the post past the budget was saved")
		}
		// Sent again from the hosted page, with its token (a moment old by
		// then), it goes through.
		pageToken(t, r)
		f := url.Values{"_t": {formToken(env, "frm_3f91d0", time.Minute)}, "email": {"five@fastmail.com"}, "consent": {"1"}}
		must(t, do(t, env, http.MethodPost, path, f, hdr{"Origin": env.Server.URL}), 200, "You're subscribed", "five@fastmail.com")
	})
}
