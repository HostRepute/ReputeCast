package whatsapp_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/whatsapp"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta/metatest"
)

// WhatsApp's own words read in the reader's language: a problem, a
// template's language, the rejected template's notification (kept as
// messages, its row in English) and a keyword's consent change on the
// contact's timeline (stored in English). Meta's words and API clients
// that don't ask stay in English.
func TestWhatsAppInVietnamese(t *testing.T) {
	f := newFixture(t)
	env, p := f.env, f.path
	vi := env.Client()
	vi.SignIn("owner@x22wa.test", "")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	badID := map[string]any{"businessAccountId": "acme", "phoneNumberId": demo.WhatsAppPhoneID, "accessToken": demo.WhatsAppToken, "appSecret": "x"}
	if pr := vi.Do(http.MethodPost, p+"/whatsapp/account", badID).MustStatus(t, 422).Problem(t); pr["detail"] != "ID WhatsApp Business Account là một dãy số, lấy trong WhatsApp Manager › Account tools." {
		t.Fatalf("Vietnamese problem: %v", pr)
	}
	if pr := f.owner.Do(http.MethodPost, p+"/whatsapp/account", badID).MustStatus(t, 422).Problem(t); pr["detail"] != "The WhatsApp Business Account ID is a number, from WhatsApp Manager › Account tools." {
		t.Fatalf("English problem: %v", pr)
	}

	f.connect(t)
	var ov whatsapp.WhatsAppOverview
	vi.Do(http.MethodGet, p+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &ov)
	for _, tp := range ov.Templates {
		if tp.Name == "summit_day_before" && tp.LanguageLabel != "Tiếng Anh (Mỹ)" {
			t.Fatalf("template language: %q", tp.LanguageLabel)
		}
	}

	// The connection's sync found a rejected template: the row keeps the
	// English, each member reads it in their language.
	if title := one[string](t, env, `SELECT title FROM notifications WHERE workspace_id = $1 AND kind = 'whatsapp_attention'`, f.ws.ID); title != "WhatsApp template digest_highlights rejected" {
		t.Fatalf("stored title: %q", title)
	}
	var list map[string]any
	vi.Do(http.MethodGet, p+"/notifications", nil).MustStatus(t, 200).JSON(t, &list)
	if n := list["items"].([]any)[0].(map[string]any); n["title"] != "Mẫu WhatsApp digest_highlights bị từ chối" || n["categoryLabel"] != "Nhà cung cấp" {
		t.Fatalf("Vietnamese notification: %v", n)
	}

	// A STOP keyword's consent change, written in English, reads in Vietnamese.
	listID, ids := f.people(t)
	f.owner.Do(http.MethodPut, p+"/contacts/"+ids["ana"].String()+"/whatsapp/consent", map[string]any{"listId": listID, "status": "subscribed", "evidence": evidence()}).
		MustStatus(t, 200)
	body, sig := metatest.MessageWebhook(demo.WhatsAppAppSecret, demo.WhatsAppBusinessID, demo.WhatsAppPhoneID, "+13125550124", "Ana", "stop", time.Now())
	f.hook(t, body, sig)
	var tl struct {
		Items []struct {
			Type   string `json:"type"`
			Detail string `json:"detail"`
			Note   string `json:"note"`
		} `json:"items"`
	}
	read := func(c string) (detail, note string) {
		cl := f.owner
		if c == "vi" {
			cl = vi
		}
		cl.Do(http.MethodGet, p+"/contacts/"+ids["ana"].String()+"/activity?limit=20", nil).MustStatus(t, 200).JSON(t, &tl)
		for _, it := range tl.Items {
			if it.Type == "whatsapp_consent_opted_out" {
				return it.Detail, it.Note
			}
		}
		t.Fatalf("no opt-out on the timeline: %+v", tl.Items)
		return "", ""
	}
	if _, note := read("en"); note != "Opted out on WhatsApp" {
		t.Fatalf("English note: %q", note)
	}
	if _, note := read("vi"); note != "Đã hủy nhận trên WhatsApp" {
		t.Fatalf("Vietnamese note: %q", note)
	}
}
