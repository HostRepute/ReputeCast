package meta_test

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/whatsapp/meta"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta/metatest"
)

// X-Hub-Signature-256 is HMAC-SHA256 of the raw body with the app secret,
// in hex (the expected value is computed with openssl dgst -sha256 -hmac).
func TestSignature(t *testing.T) {
	body := []byte(`{"object":"whatsapp_business_account","entry":[]}`)
	want := "sha256=e8259fe8f662b6495eca5a32132bfc7103f098398e6d759f48dccbe623b0954b"
	if got := meta.Sign("c0ffee-app-secret", body); got != want {
		t.Fatalf("sign %s", got)
	}
	if !meta.VerifySignature("c0ffee-app-secret", body, want) {
		t.Fatal("doesn't verify")
	}
	// The hex may come in upper case.
	if !meta.VerifySignature("c0ffee-app-secret", body, want[:7]+strings.ToUpper(want[7:])) {
		t.Fatal("upper-case hex refused")
	}
	for _, bad := range []string{"", "sha1=" + want[7:], "sha256=" + strings.Repeat("0", 64), want + "00"} {
		if meta.VerifySignature("c0ffee-app-secret", body, bad) {
			t.Errorf("verified %q", bad)
		}
	}
	if meta.VerifySignature("other-secret", body, want) || meta.VerifySignature("", body, want) {
		t.Fatal("another secret verified")
	}
	tampered := append([]byte(nil), body...)
	tampered[2] = 'O'
	if meta.VerifySignature("c0ffee-app-secret", tampered, want) {
		t.Fatal("a changed body verified")
	}
}

func TestVerifyChallenge(t *testing.T) {
	q := url.Values{"hub.mode": {"subscribe"}, "hub.verify_token": {"rc-wa-7Q2mX9vKp4"}, "hub.challenge": {"1158201444"}}
	if c, ok := meta.VerifyChallenge(q, "rc-wa-7Q2mX9vKp4"); !ok || c != "1158201444" {
		t.Fatalf("challenge %q %v", c, ok)
	}
	if _, ok := meta.VerifyChallenge(q, "another-token"); ok {
		t.Fatal("wrong token answered")
	}
	q.Set("hub.mode", "unsubscribe")
	if _, ok := meta.VerifyChallenge(q, "rc-wa-7Q2mX9vKp4"); ok {
		t.Fatal("wrong mode answered")
	}
}

func TestParseWebhook(t *testing.T) {
	at := time.Date(2026, 10, 14, 23, 2, 0, 0, time.UTC)
	b, sig := metatest.StatusWebhook("s", "104862390044410", "pn1", "wamid.1", "+13125550123", "failed", at, 131026)
	if !meta.VerifySignature("s", b, sig) {
		t.Fatal("fake signature")
	}
	evs, err := meta.ParseWebhook(b)
	if err != nil || len(evs) != 1 {
		t.Fatalf("%v %v", evs, err)
	}
	e := evs[0]
	if e.Kind != meta.EventStatus || e.Status != "failed" || e.MessageID != "wamid.1" || e.Recipient != "+13125550123" || !e.At.Equal(at) ||
		len(e.Errors) != 1 || e.Errors[0].Code != 131026 || e.PhoneNumberID != "pn1" || e.BusinessID != "104862390044410" {
		t.Fatalf("status %+v", e)
	}
	b, _ = metatest.MessageWebhook("s", "waba", "pn1", "+13125550123", "Sam Rivera", "Is there parking at the venue?", at)
	evs, _ = meta.ParseWebhook(b)
	if len(evs) != 1 || evs[0].Kind != meta.EventMessage || evs[0].From != "+13125550123" || evs[0].Name != "Sam Rivera" || evs[0].Text != "Is there parking at the venue?" {
		t.Fatalf("message %+v", evs)
	}
	b, _ = metatest.ButtonWebhook("s", "waba", "pn1", "+13125550123", "wamid.9", "Stop promotions", at)
	evs, _ = meta.ParseWebhook(b)
	if len(evs) != 1 || evs[0].ButtonText != "Stop promotions" || evs[0].ContextID != "wamid.9" || evs[0].Type != "button" {
		t.Fatalf("button %+v", evs)
	}
	b, _ = metatest.TemplateWebhook("s", "waba", "1234", "digest_highlights", "en_US", "REJECTED", "INVALID_FORMAT", "Add sample values for every variable.")
	evs, _ = meta.ParseWebhook(b)
	if len(evs) != 1 || evs[0].Kind != meta.EventTemplate || evs[0].TemplateEvent != "REJECTED" || evs[0].TemplateID != "1234" ||
		evs[0].ReasonText != "Add sample values for every variable." || evs[0].Reason != "INVALID_FORMAT" {
		t.Fatalf("template %+v", evs)
	}
	b, _ = metatest.QualityWebhook("s", "waba", "DOWNGRADE", "TIER_250", "TIER_1K")
	evs, _ = meta.ParseWebhook(b)
	if len(evs) != 1 || evs[0].Kind != meta.EventQuality || evs[0].CurrentLimit != "TIER_250" || evs[0].OldLimit != "TIER_1K" {
		t.Fatalf("quality %+v", evs)
	}
	if _, err := meta.ParseWebhook([]byte(`{"object":"page","entry":[]}`)); !errors.Is(err, meta.ErrWebhook) {
		t.Fatalf("other object: %v", err)
	}
}

func TestClientAgainstFake(t *testing.T) {
	f := metatest.New("104862390044410", "pn-0110", "EAAG-token")
	f.SetTemplates([]meta.Template{
		{ID: "1", Name: "summit_day_before", Status: "APPROVED", Category: "UTILITY", Language: "en_US",
			Components: []meta.TemplateComponent{{Type: "BODY", Text: "Hi {{1}}, see you tomorrow."}, {Type: "BUTTONS", Buttons: []meta.TemplateButton{{Type: "URL", Text: "View the agenda", URL: "https://news.acme.io/summit"}}}}},
		{ID: "2", Name: "summit_venue_change", Status: "PENDING", Category: "UTILITY", Language: "en_US"},
		{ID: "3", Name: "digest_highlights", Status: "REJECTED", Category: "MARKETING", Language: "en_US", RejectedReason: "INVALID_FORMAT"},
	})
	ctx := context.Background()
	c := &meta.Client{Creds: meta.Credentials{BusinessAccountID: "104862390044410", PhoneNumberID: "pn-0110", AccessToken: "EAAG-token", AppSecret: "s"}, HTTP: f.Client()}
	p, err := c.Phone(ctx)
	if err != nil || p.VerifiedName != "Acme" || meta.Quality(p.QualityRating) != "high" || meta.LimitOf(p.MessagingLimitTier) != 1000 ||
		meta.ParsePhoneDisplay(p.DisplayPhoneNumber) != "+13125550110" {
		t.Fatalf("phone %+v %v", p, err)
	}
	a, err := c.Account(ctx)
	if err != nil || a.Name != "Acme Inc" {
		t.Fatalf("account %+v %v", a, err)
	}
	if err := c.SubscribeApp(ctx); err != nil || !f.Subscribed {
		t.Fatalf("subscribe: %v", err)
	}
	// Templates come page by page.
	ts, err := c.Templates(ctx)
	if err != nil || len(ts) != 3 || ts[0].Body() != "Hi {{1}}, see you tomorrow." || len(ts[0].Buttons()) != 1 {
		t.Fatalf("templates %+v %v", ts, err)
	}
	s, err := c.Send(ctx, meta.Message{To: "+13125550123", Template: "summit_day_before", Language: "en_US", Params: []string{"Sam"}})
	if err != nil || !strings.HasPrefix(s.ID, "wamid.") || s.WAID != "13125550123" {
		t.Fatalf("send %+v %v", s, err)
	}
	sent := f.SentMessages()
	if len(sent) != 1 || sent[0].To != "+13125550123" || sent[0].Template != "summit_day_before" || len(sent[0].Params) != 1 || sent[0].Params[0] != "Sam" {
		t.Fatalf("recorded %+v", sent)
	}
	// A pending template can't be sent; Meta answers 132001.
	_, err = c.Send(ctx, meta.Message{To: "+13125550123", Template: "summit_venue_change"})
	var me *meta.Error
	if !errors.As(err, &me) || me.Code != 132001 || !meta.Permanent(me.Code) {
		t.Fatalf("pending template: %v", err)
	}
	// Free text outside the window: 131047.
	f.Fail("+13125550199", 131047)
	_, err = c.Send(ctx, meta.Message{To: "+13125550199", Text: "Yes, there's parking."})
	if !errors.As(err, &me) || me.Code != 131047 || meta.ReasonOf(me.Code) != "window_closed" {
		t.Fatalf("window: %v", err)
	}
	// An expired token.
	f.ExpireToken()
	_, err = c.Phone(ctx)
	if !errors.As(err, &me) || !me.TokenRejected() || !me.Expired() {
		t.Fatalf("expired: %v", err)
	}
	f.Renew("EAAG-new")
	c.Creds.AccessToken = "EAAG-new"
	if _, err := c.Phone(ctx); err != nil {
		t.Fatalf("renewed: %v", err)
	}
}
