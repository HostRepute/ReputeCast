package connect

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

func TestMessagesAreTranslated(t *testing.T) {
	if missing := msgs.Missing(i18n.Vietnamese); len(missing) > 0 {
		t.Fatalf("not in Vietnamese: %v", missing)
	}
}

var (
	mailgunLike = providers.Descriptor{Adapter: "mailgun", Name: "Mailgun", Kind: "api",
		Capabilities: providers.Capabilities{Bulk: providers.BulkAllowed, Webhooks: "Registered", DomainAPI: "full", BatchSize: 1, Sandbox: "detection", DedicatedIPs: "paid"},
		Fields:       []providers.Field{{Key: "api_key", Label: "Private API key", Secret: true, Required: true}}}
	smtpLike  = providers.Descriptor{Adapter: "smtp", Name: "Generic SMTP", Kind: "smtp", Capabilities: providers.Capabilities{Webhooks: "Bounce mailbox", BatchSize: 50}}
	postalish = providers.Descriptor{Adapter: "postal", Name: "Postal (self-hosted)", Kind: "self_hosted", Capabilities: providers.Capabilities{Webhooks: "Registered", BatchSize: 1}}
	unknown   = providers.Descriptor{Adapter: "acme", Name: "Acme Mail", Kind: "api",
		Capabilities: providers.Capabilities{Webhooks: "Event Grid", BatchSize: 1000, Sandbox: "test addresses", QuotaDiscovery: true, Pricing: "$1 per 1,000"}}
)

// The English the flow writes reads as it did before it had messages.
func TestEnglishAsItWas(t *testing.T) {
	en := i18n.Default
	for _, c := range []struct {
		q    providers.Quota
		want string
	}{
		{providers.Quota{}, "Read the account"},
		{providers.Quota{Kind: "rolling_24h", Max: 50000, MaxRate: 14}, "Read account: production access, 50,000 per 24 h, 14 per second"},
		{providers.Quota{Kind: "rolling_24h", Sandbox: true, Max: 200, MaxRate: 1}, "Read account: sandbox, 200 per 24 h, 1 per second"},
		{providers.Quota{Kind: "monthly_plan", Max: 50000}, "Read account: production access, 50,000 per month"},
		{providers.Quota{Account: "Active", MaxRate: 2.5}, "Read account: 2.5 per second"},
		{providers.Quota{Account: "Active · paid"}, "Read account: active · paid"},
	} {
		if got, _ := accountText(c.q); got != c.want {
			t.Errorf("account %+v: %q, want %q", c.q, got, c.want)
		}
	}
	for _, c := range []struct {
		r    RegionState
		want string
	}{
		{RegionState{Region: "us-east-1", Name: "N. Virginia", Production: true, Max24h: 50000}, "us-east-1 · N. Virginia · production access, quota 50,000 per day"},
		{RegionState{Region: "us-west-2", Name: "Oregon", Production: true}, "us-west-2 · Oregon · production access"},
		{RegionState{Region: "eu-west-1", Name: "Ireland", Max24h: 200}, "eu-west-1 · Ireland · sandbox"},
		{RegionState{Region: "ap-south-1", Err: "AccessDenied"}, "ap-south-1 · not available"},
	} {
		if got := regionLabel(en, c.r); got != c.want {
			t.Errorf("region: %q, want %q", got, c.want)
		}
	}
	chipsOf := func(l i18n.Lang, d providers.Descriptor) string {
		out := []string{}
		for _, ch := range chips(l, d, copyFor(l, d)) {
			out = append(out, ch.Kind+":"+ch.Label)
		}
		return strings.Join(out, ",")
	}
	if got := chipsOf(en, unknown); got != "yes:Webhooks (Event Grid),partial:Domain, manual DNS,partial:Suppression, local,yes:Batch 1,000,no:Dedicated IPs,yes:Sandbox test addresses,yes:Quota discovery" {
		t.Errorf("chips: %s", got)
	}
	if got := chipsOf(en, mailgunLike); got != "yes:Webhooks,yes:Domain via API,partial:Suppression, local,no:Batch,partial:Dedicated IPs, paid,yes:Sandbox domain,partial:Quota, set manually" {
		t.Errorf("mailgun's chips: %s", got)
	}
	if terms, _ := termsBadge(en, smtpLike, copyFor(en, smtpLike)); terms != "Depends on the server" {
		t.Errorf("terms: %q", terms)
	}
	c := copyFor(en, smtpLike)
	if c.Display != "Any SMTP server" || c.Step1 != "SMTP server" || c.Title != "Connect an SMTP server" || c.Heading != "SMTP server details" || c.Short != "SMTP" {
		t.Errorf("smtp: %+v", c)
	}
	c = copyFor(en, postalish)
	if c.Display != "Postal (self-hosted)" || c.Step1 != "Postal" || c.Title != "Connect Postal" || c.Section != "1. Server and API key" {
		t.Errorf("postal: %+v", c)
	}
	c = copyFor(en, mailgunLike)
	if c.Heading != "Paste your Mailgun API key" || c.Lead != "ReputeCast registers webhooks and fetches the DNS records for your domain through the Mailgun API." || c.Section != "1. API key" {
		t.Errorf("mailgun: %+v", c)
	}
	c = copyFor(en, unknown)
	if c.Title != "Connect Acme Mail" || c.Heading != "Connect Acme Mail" || c.ConnectionHint != "API key, region, webhooks" || c.Section != "1. Credentials" ||
		c.WebhookEvents != "delivery, bounce and complaint events" || c.Pricing != "$1 per 1,000" || c.Short != "Acme Mail" {
		t.Errorf("an adapter the table doesn't know: %+v", c)
	}
	e := keyError(mailgunLike, built{settings: map[string]any{}}, &providers.CallError{Op: "GET /v3/domains", Status: 401, Code: "Forbidden", Auth: true})
	if e.Message != "Mailgun rejected this key: `401 Unauthorized · Forbidden`. Check that it's a Private API key (not a Sending key or the HTTP webhook signing key) and that it belongs to the region selected below." {
		t.Errorf("key error: %q", e.Message)
	}
	if got := roundTrip(en, 2400*time.Millisecond, mailgunLike, false); got != "Round trip 2.4 s from queue to delivery event. Webhooks work." {
		t.Errorf("round trip: %q", got)
	}
}

// What the setup keeps reads in each reader's language.
func TestStoredTextReadsInTheReadersLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	e := keyError(mailgunLike, built{settings: map[string]any{}}, &providers.CallError{Op: "GET /v3/domains", Status: 401, Code: "Forbidden", Auth: true})
	if got := textsIn(e.Message, e.Texts, vi); got != "Mailgun đã từ chối khóa này: `401 Unauthorized · Forbidden`. Hãy kiểm tra đó là Private API key (không phải Sending key hay HTTP webhook signing key) và khóa thuộc khu vực đã chọn bên dưới." {
		t.Errorf("key error: %q", got)
	}
	if got := textsIn(e.Message, e.Texts, i18n.Default); got != e.Message {
		t.Errorf("English: %q", got)
	}
	account, text := accountText(providers.Quota{Kind: "rolling_24h", Max: 50000, MaxRate: 14})
	if got := textIn(account, text, vi); got != "Đã đọc tài khoản: có quyền gửi chính thức, 50.000 mỗi 24 giờ, 14 mỗi giây" {
		t.Errorf("account: %q", got)
	}
	// A row that failed on a private public URL: the code stays, the reason
	// and the hint read in Vietnamese.
	reach, _ := unreachable(t.Context(), mustURL(t, "http://localhost:8080/hooks/ses/x"), timeZero)
	row := ProvisionState{Key: "sns_topic", Status: "failed"}
	row.unreachable(reach)
	if row.Error != "ECONNREFUSED: localhost only works on this machine" || row.errorIn(vi) != "ECONNREFUSED: localhost chỉ hoạt động trên máy này" ||
		!strings.HasPrefix(row.in(vi, "hint", row.Hint), "AWS phải truy cập được URL công khai của ReputeCast") {
		t.Errorf("row: %q / %q / %q", row.Error, row.errorIn(vi), row.in(vi, "hint", row.Hint))
	}
	// A row written before the messages were kept reads in English.
	old := ProvisionState{Label: "Create configuration set `rc-system`", Error: "AccessDenied: ses:CreateConfigurationSet"}
	if old.in(vi, "label", old.Label) != old.Label || old.errorIn(vi) != old.Error {
		t.Errorf("old row: %q", old.errorIn(vi))
	}
	// The stepper's line is written again from what the setup kept.
	if got := connectionSummaryIn(vi, "ses", Connection{Summary: "us-east-1 · production"}); got != "us-east-1 · gửi chính thức" {
		t.Errorf("SES summary: %q", got)
	}
	cn := Connection{Summary: "eu · connected", Account: "Connected", AccountText: mSummaryConnected.Keep(nil), Settings: map[string]string{"region": "eu"}}
	if got := connectionSummaryIn(vi, "api", cn); got != "eu · đã kết nối" {
		t.Errorf("API summary: %q", got)
	}
	cn.AccountText = i18n.Text{}
	if got := connectionSummaryIn(vi, "api", cn); got != "eu · connected" {
		t.Errorf("API summary written before: %q", got)
	}
	if got := regionLabel(vi, RegionState{Region: "us-east-1", Name: "N. Virginia", Production: true, Max24h: 50000}); got != "us-east-1 · N. Virginia · có quyền gửi chính thức, hạn mức 50.000 mỗi ngày" {
		t.Errorf("region: %q", got)
	}
	c := copyFor(vi, smtpLike)
	if c.Display != "Máy chủ SMTP bất kỳ" || c.Title != "Kết nối máy chủ SMTP" {
		t.Errorf("smtp: %+v", c)
	}
	labels := []string{}
	for _, ch := range chips(vi, unknown, copyFor(vi, unknown)) {
		labels = append(labels, ch.Label)
	}
	if got := strings.Join(labels, ","); got != "Webhook (Event Grid),Tên miền, DNS thủ công,Chặn gửi, chỉ cục bộ,Gửi lô 1.000,IP riêng,Sandbox với địa chỉ thử,Dò hạn mức" {
		t.Errorf("chips: %s", got)
	}
	for in, want := range map[string]string{"Khóa API": "khóa API", "SMTP host": "SMTP host", "Đã duyệt": "đã duyệt", "": ""} {
		if got := lowerFirst(in); got != want {
			t.Errorf("lowerFirst(%q) = %q", in, got)
		}
	}
}

var timeZero = time.Time{}

func mustURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
