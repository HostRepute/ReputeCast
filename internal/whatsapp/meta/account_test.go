package meta_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/whatsapp/meta"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta/metatest"
)

// A signature header that can't be a signature is refused before the body
// is read: anyone can post to the webhook's address.
func TestSignatureShape(t *testing.T) {
	good := meta.Sign("c0ffee-app-secret", []byte(`{}`))
	for _, h := range []string{good, " " + good + " ", good[:7] + strings.ToUpper(good[7:])} {
		if !meta.SignatureShape(h) {
			t.Errorf("%q can be a signature", h)
		}
	}
	for _, h := range []string{"", "sha256=", "sha256=00", "sha1=" + good[7:], good + "00", "sha256=" + strings.Repeat("z", 64)} {
		if meta.SignatureShape(h) {
			t.Errorf("%q can't be a signature", h)
		}
	}
}

// both is one delivery carrying the entries of two.
func both(t *testing.T, a, b []byte) []byte {
	t.Helper()
	var x, y struct {
		Object string            `json:"object"`
		Entry  []json.RawMessage `json:"entry"`
	}
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		t.Fatal("not webhooks")
	}
	x.Entry = append(x.Entry, y.Entry...)
	out, _ := json.Marshal(x)
	return out
}

// Workspaces whose accounts share a Meta app get each other's deliveries,
// signed with the same app secret: only the entries of the connected
// WhatsApp Business Account, and of its number, are kept.
func TestForKeepsTheConnectedAccount(t *testing.T) {
	at := time.Date(2026, 10, 14, 23, 2, 0, 0, time.UTC)
	mine, _ := metatest.StatusWebhook("s", "waba-1", "pn-1", "wamid.1", "+13125550123", "delivered", at, 0)
	got, n, err := meta.For(mine, "waba-1", "pn-1")
	if err != nil || n != 0 || !bytes.Equal(got, mine) {
		t.Fatalf("the account's own delivery: %s, %d left out, %v", got, n, err)
	}
	tmpl, _ := metatest.TemplateWebhook("s", "waba-1", "1234", "digest_highlights", "en_US", "APPROVED", "NONE", "")
	if got, n, err := meta.For(tmpl, "waba-1", "pn-1"); err != nil || n != 0 || !bytes.Equal(got, tmpl) {
		t.Fatalf("a template of the account (no number): %s, %d, %v", got, n, err)
	}

	otherNumber, _ := metatest.MessageWebhook("s", "waba-1", "pn-2", "+13125550124", "Ana", "STOP", at)
	otherAccount, _ := metatest.TemplateWebhook("s", "waba-2", "5678", "digest_highlights", "en_US", "REJECTED", "INVALID_FORMAT", "")
	for name, body := range map[string][]byte{"another number": otherNumber, "another account": otherAccount} {
		got, n, err := meta.For(body, "waba-1", "pn-1")
		evs, perr := meta.ParseWebhook(got)
		if err != nil || perr != nil || n != 1 || len(evs) != 0 {
			t.Fatalf("%s: %s, %d left out, %d kept, %v %v", name, got, n, len(evs), err, perr)
		}
	}

	mixed := both(t, both(t, mine, otherNumber), otherAccount)
	got, n, err = meta.For(mixed, "waba-1", "pn-1")
	if err != nil || n != 2 {
		t.Fatalf("mixed: %d left out, %v", n, err)
	}
	evs, err := meta.ParseWebhook(got)
	if err != nil || len(evs) != 1 || evs[0].MessageID != "wamid.1" || evs[0].PhoneNumberID != "pn-1" {
		t.Fatalf("kept %+v, %v", evs, err)
	}
	if strings.Contains(string(got), "STOP") || strings.Contains(string(got), "waba-2") {
		t.Fatalf("another account's data was kept: %s", got)
	}

	if _, _, err := meta.For([]byte(`{"object":"page","entry":[]}`), "waba-1", "pn-1"); err == nil {
		t.Fatal("not a WhatsApp webhook")
	}
}
