package forms_test

// Posts of the copied plain HTML form: the form can't
// carry the timing token and anyone can post _plain=1, so a plain post
// never skips a check. It gets the Turnstile check while the workspace has
// keys; without them it has a small budget, confirms by email even on a
// single opt-in form and never gives SMS consent.

import (
	"context"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/phone"
	"github.com/hostrepute/reputecast/internal/store"
)

// fakeTexts offers texts and keeps what signups asked of them.
type fakeTexts struct {
	mu        sync.Mutex
	consents  map[uuid.UUID]bool // contact → SMS consent given
	confirmed int
}

func (f *fakeTexts) Offer(context.Context, *store.Workspace, string) (forms.TextOffer, bool) {
	return forms.TextOffer{Countries: []forms.TextCountry{{Code: "US", Name: "United States", Dial: "+1"}}, Default: "US",
		ConsentText: "Texts from Acme. Msg & data rates may apply. Reply STOP to cancel.", InvalidPhone: "Check the number.", PhoneNeeded: "Add your number."}, true
}

func (f *fakeTexts) Save(_ context.Context, _ pgx.Tx, _ *store.Workspace, contact uuid.UUID, _ phone.Number, consent bool, _ []uuid.UUID, _ forms.TextEvidence) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.consents[contact] = consent
	return consent, nil
}

func (f *fakeTexts) Confirm(context.Context, *store.Workspace, uuid.UUID, []uuid.UUID) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.confirmed++
}

func (f *fakeTexts) Texted(_, _ string) string { return "We've also texted you." }

func (f *fakeTexts) TextedContact(context.Context, *store.Workspace, uuid.UUID, string) string {
	return ""
}

func TestPlainPosts(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	a := env.App
	sql := func(query string, args ...any) {
		t.Helper()
		if _, err := a.Pool.Exec(ctx, query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	acme, err := a.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	ws := &acme
	// The footer form subscribes at once, asks for a number and SMS consent,
	// and doesn't ask for the Turnstile check itself.
	sql(`UPDATE forms SET double_opt_in = false, fields = '[{"key":"email","required":true},{"key":"phone"}]',
		content = content || '{"smsConsent": true}', spam_protection = spam_protection || '{"turnstile": false}'
		WHERE workspace_id = $1 AND public_id = 'frm_3f91d0'`, ws.ID)
	row, err := a.Q.FormByPublicID(ctx, store.FormByPublicIDParams{WorkspaceID: ws.ID, PublicID: "frm_3f91d0"})
	if err != nil {
		t.Fatal(err)
	}
	form := forms.ConfigOf(row)
	texts := &fakeTexts{consents: map[uuid.UUID]bool{}}
	now := time.Now()
	svc := &forms.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Tokens: a.Links, URLs: a.URLs, Texts: texts,
		Audience: &audience.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Mail: a.Mail, Links: a.Links, URLs: a.URLs},
		Now:      func() time.Time { return now }}
	addr := func(s string) *netip.Addr { ip := netip.MustParseAddr(s); return &ip }
	// What a script sends: an allowed site as its Origin, and _plain=1.
	plain := func(email, from string) forms.Submission {
		return forms.Submission{Email: email, Consent: true, Plain: true, Origin: "acme.io", IP: addr(from), UserAgent: "Mozilla/5.0",
			Phone: "+13125550161", SmsConsent: true}
	}
	contact := func(email string) (uuid.UUID, string, bool) {
		var id uuid.UUID
		var status string
		err := a.Pool.QueryRow(ctx, `SELECT c.id, m.status FROM contacts c JOIN list_memberships m ON m.contact_id = c.id
			WHERE c.workspace_id = $1 AND c.email = $2 AND m.list_id = $3`, ws.ID, email, form.Lists[0]).Scan(&id, &status)
		return id, status, err == nil
	}

	t.Run("with Turnstile, a plain post gets the check", func(t *testing.T) {
		set, err := svc.LoadSettings(ctx, ws)
		if err != nil {
			t.Fatal(err)
		}
		if set.TurnstileSiteKey() == "" {
			t.Fatal("the demo workspace has Turnstile keys")
		}
		// Even on a form that doesn't ask for the check otherwise.
		if out := svc.Submit(ctx, ws, set, form, plain("held@fastmail.com", "203.0.113.1")); out.Status != forms.OutcomeChallenge || out.ChallengeID == "" {
			t.Fatalf("plain post: %+v", out)
		}
		if _, _, ok := contact("held@fastmail.com"); ok {
			t.Fatal("a plain post was saved before the check")
		}
	})

	sql(`UPDATE public_page_settings SET turnstile_status = 'off', turnstile_site_key = '', turnstile_secret_enc = NULL WHERE workspace_id = $1`, ws.ID)
	set, err := svc.LoadSettings(ctx, ws)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("without Turnstile, a plain post confirms by email and gives no SMS consent", func(t *testing.T) {
		out := svc.Submit(ctx, ws, set, form, plain("victim@fastmail.com", "203.0.113.2"))
		if out.Status != forms.OutcomeConfirm || out.Texted != "" {
			t.Fatalf("plain post to a single opt-in form: %+v", out)
		}
		id, status, ok := contact("victim@fastmail.com")
		if !ok || status != "pending" {
			t.Fatalf("membership %q (%v), want pending until the confirmation click", status, ok)
		}
		if consent, saved := texts.consents[id]; !saved || consent || texts.confirmed != 0 {
			t.Fatalf("SMS consent %v (number saved %v), %d confirmations texted", consent, saved, texts.confirmed)
		}
		// The hosted page, with its token, still subscribes at once.
		hosted := plain("person@fastmail.com", "203.0.113.3")
		hosted.Plain, hosted.Token = false, svc.FormToken(form.PublicID, now.Add(-time.Minute))
		if out := svc.Submit(ctx, ws, set, form, hosted); out.Status != forms.OutcomeSubscribed || texts.confirmed != 1 {
			t.Fatalf("hosted page: %+v, %d confirmations texted", out, texts.confirmed)
		}
		if _, status, _ := contact("person@fastmail.com"); status != "subscribed" {
			t.Fatalf("hosted page's membership %q", status)
		}
	})

	t.Run("without Turnstile, plain posts have a budget", func(t *testing.T) {
		for i, email := range []string{"one@fastmail.com", "two@fastmail.com", "three@fastmail.com"} {
			if out := svc.Submit(ctx, ws, set, form, plain(email, "198.51.100.7")); out.Status != forms.OutcomeConfirm {
				t.Fatalf("plain post %d: %+v", i+1, out)
			}
		}
		out := svc.Submit(ctx, ws, set, form, plain("four@fastmail.com", "198.51.100.7"))
		if out.Status != forms.OutcomeInvalid || out.Message != "One more step: confirm you're a person." || len(out.Errors) != 0 {
			t.Fatalf("a fourth plain post from the address: %+v", out)
		}
		if _, _, ok := contact("four@fastmail.com"); ok {
			t.Fatal("the post past the budget was saved")
		}
		now = now.Add(20 * time.Minute)
		if out := svc.Submit(ctx, ws, set, form, plain("four@fastmail.com", "198.51.100.7")); out.Status != forms.OutcomeConfirm {
			t.Fatalf("20 minutes later: %+v", out)
		}
	})
}
