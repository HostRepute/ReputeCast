package publicpages_test

import (
	"context"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/linktoken"
)

// DP-3: a confirmation link confirms only the lists of the signup it was
// sent for; an expired link offers a new one, and the new one works; an old
// link without a date works while the signup is recent.
func TestConfirmationLinkLifetime(t *testing.T) {
	env, ids := demo(t)
	ctx := context.Background()
	mail := &fakeMail{}
	env.App.Mail = mail
	email := "two.signups@fastmail.com"
	contact := uuid.Must(uuid.NewV7())
	sub1, sub2 := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	stmts := []struct {
		q    string
		args []any
	}{
		{`INSERT INTO contacts (id, workspace_id, email, first_name, consent, source) VALUES ($1, $2, $3, 'Two', 'pending', 'form:test')`, []any{contact, ids.ws, email}},
		{`INSERT INTO list_memberships (list_id, contact_id, status, source) VALUES ($1, $2, 'pending', 'form:test'), ($3, $2, 'pending', 'form:test')`, []any{ids.digest, contact, ids.updates}},
		{`INSERT INTO form_submissions (id, workspace_id, contact_id, list_ids, double_opt_in, created_at, confirm_sent_at) VALUES
			($1, $3, $4, ARRAY[$5::uuid], true, now() - interval '2 days', now() - interval '2 days'),
			($2, $3, $4, ARRAY[$6::uuid], true, now() - interval '1 day', now() - interval '1 day')`, []any{sub1, sub2, ids.ws, contact, ids.digest, ids.updates}},
	}
	for _, s := range stmts {
		if _, err := env.App.Pool.Exec(ctx, s.q, s.args...); err != nil {
			t.Fatal(err)
		}
	}
	link := func(sub uuid.UUID, issued time.Time) string {
		return "/p/acme-newsletter/confirm/" + env.App.Links.Sign(linktoken.Claims{Kind: linktoken.Confirm, WorkspaceID: ids.ws, ContactID: contact,
			MessageID: sub, Link: linktoken.IssueDay(issued)})
	}

	t.Run("a link confirms its own signup's lists", func(t *testing.T) {
		first := link(sub1, time.Now().Add(-48*time.Hour))
		must(t, get(t, env, first), 200, "Confirm that <strong>"+email+"</strong> should get Weekly Digest from Acme.")
		must(t, confirmClick(t, env, first, nil), 200, "You're in, Two")
		if membership(t, env, contact, ids.digest) != "subscribed" || membership(t, env, contact, ids.updates) != "pending" {
			t.Fatalf("digest %s, updates %s", membership(t, env, contact, ids.digest), membership(t, env, contact, ids.updates))
		}
		if n := count(t, env, `SELECT count(*) FROM form_submissions WHERE contact_id = $1 AND confirmed_at IS NOT NULL`, contact); n != 1 {
			t.Fatalf("%d submissions confirmed, want only the first", n)
		}
	})

	t.Run("an expired link offers a new one, which works", func(t *testing.T) {
		old := link(sub2, time.Now().Add(-40*24*time.Hour))
		r := get(t, env, old)
		must(t, r, 200, "This link has expired", "Send a new link")
		lacks(t, r, email)
		must(t, confirmClick(t, env, old, nil), 200, "This link has expired")
		if membership(t, env, contact, ids.updates) != "pending" {
			t.Fatal("an expired link confirmed")
		}
		must(t, do(t, env, http.MethodPost, old, url.Values{"do": {"resend"}}, nil), 200, "Check your inbox")
		sent := mail.to(email)
		if len(sent) != 1 {
			t.Fatalf("%d emails after Send a new link, want 1", len(sent))
		}
		fresh, err := url.Parse(linkIn.FindString(sent[0].Text))
		if err != nil || fresh.Path == "" {
			t.Fatalf("no link in:\n%s", sent[0].Text)
		}
		must(t, confirmClick(t, env, fresh.Path, nil), 200, "You're in")
		if membership(t, env, contact, ids.updates) != "subscribed" {
			t.Fatal("the new link didn't confirm")
		}
		// A second ask within 10 minutes sends nothing, and says the same.
		must(t, do(t, env, http.MethodPost, old, url.Values{"do": {"resend"}}, nil), 200, "Check your inbox")
		if n := len(mail.to(email)); n != 1 {
			t.Fatalf("%d emails after asking twice", n)
		}
	})

	t.Run("an old link without a date lives as long as the signup", func(t *testing.T) {
		undated := "/p/acme-newsletter/confirm/" + sign(env, linktoken.Confirm, ids.ws, contact, uuid.Nil)
		must(t, get(t, env, undated), 200, "You're in")
		for _, q := range []string{
			`UPDATE form_submissions SET created_at = now() - interval '60 days', confirmed_at = NULL WHERE contact_id = $1`,
			`DELETE FROM consent_events WHERE contact_id = $1 AND action = 'optin_sent'`,
			`UPDATE list_memberships SET status = 'pending', subscribed_at = NULL WHERE contact_id = $1`,
		} {
			if _, err := env.App.Pool.Exec(ctx, q, contact); err != nil {
				t.Fatal(err)
			}
		}
		must(t, get(t, env, undated), 200, "This link has expired")
	})
}

// DP-3: however often an address is submitted, it gets one confirmation
// email every 10 minutes and five a day, and the page always says to check
// the inbox.
func TestOptInCooldown(t *testing.T) {
	env, ids := demo(t)
	ctx := context.Background()
	mail := &fakeMail{}
	env.App.Mail = mail
	email := "flooded@fastmail.com"
	// The per-address limit is under test, not the per-IP one.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE forms SET spam_protection = jsonb_set(spam_protection, '{rate_limit_per_ip}', '100') WHERE workspace_id = $1`, ids.ws); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		must(t, signUp(t, env, ids, email), 200, "Check your inbox")
	}
	if n := len(mail.to(email)); n != 1 {
		t.Fatalf("%d confirmation emails after three signups in a minute, want 1", n)
	}
	// Eleven minutes later the next goes, up to five a day.
	for i := 0; i < 8; i++ {
		if _, err := env.App.Pool.Exec(ctx, `UPDATE optin_sends SET sent_at = sent_at - interval '11 minutes'`); err != nil {
			t.Fatal(err)
		}
		must(t, signUp(t, env, ids, email), 200, "Check your inbox")
	}
	if n := len(mail.to(email)); n != 5 {
		t.Fatalf("%d confirmation emails in a day, want 5", n)
	}
}

// clientKeyOf is forms.ClientKey of an address.
func clientKeyOf(t *testing.T, s string) string {
	t.Helper()
	return forms.ClientKey(netip.MustParseAddr(s))
}

// DP-3: rotating addresses inside one IPv6 /64 is one client.
func TestClientKeyGroupsIPv6(t *testing.T) {
	for _, c := range []struct{ a, b string }{{"2001:db8:1:2::1", "2001:db8:1:2:ffff::9"}, {"::ffff:203.0.113.9", "203.0.113.9"}} {
		if ka, kb := clientKeyOf(t, c.a), clientKeyOf(t, c.b); ka != kb {
			t.Errorf("%s → %s, %s → %s: want one key", c.a, ka, c.b, kb)
		}
	}
	if clientKeyOf(t, "2001:db8:1:2::1") == clientKeyOf(t, "2001:db8:1:3::1") {
		t.Error("two /64s share a key")
	}
	if !strings.HasSuffix(clientKeyOf(t, "2001:db8:1:2::1"), "/64") {
		t.Error("an IPv6 key names its /64")
	}
}

// DP-3: a form flooded from many addresses (past 120 submissions a minute)
// asks each submission for the Turnstile check while it lasts, though the
// form doesn't ask for it otherwise.
func TestFloodedFormAsksForTurnstile(t *testing.T) {
	env, ids := demo(t)
	env.App.Mail = &fakeMail{}
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE forms SET spam_protection = spam_protection || '{"rate_limit_per_ip": 1000, "turnstile": false}' WHERE workspace_id = $1`, ids.ws); err != nil {
		t.Fatal(err)
	}
	var last resp
	for i := range 121 {
		last = signUp(t, env, ids, "flood."+strconv.Itoa(i)+"@fastmail.com")
		if i < 120 && !strings.Contains(last.Body, "Check your inbox") {
			t.Fatalf("submission %d before the flood: %d %s", i, last.Status, clip(last.Body))
		}
	}
	must(t, last, 200, "One more step")
}
