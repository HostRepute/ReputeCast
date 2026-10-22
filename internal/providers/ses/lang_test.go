package ses

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

func TestTextsReadInTheReadersLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	d := Descriptor()
	if got := d.In(vi).Terms[0].Note; got != "Cần quyền gửi chính thức; tài khoản phải luôn ở dưới ngưỡng trả thư và khiếu nại của SES." {
		t.Errorf("terms in Vietnamese: %q", got)
	}
	if got := d.In(i18n.Default).Fields[1].Help; got != "A CloudFormation role (recommended) or IAM access keys." {
		t.Errorf("a field's hint in English: %q", got)
	}
	// A provisioning row reads in the reader's language, its English as it was.
	steps := (&Provider{}).ProvisionSteps(context.Background(), providers.ProvisionInput{WebhookURL: "https://rc.acme.io/hooks/ses/t"})
	if steps[0].Label != "Create SNS topic `reputecast-events` and subscribe `https://rc.acme.io/hooks/ses/t`" {
		t.Errorf("English: %q", steps[0].Label)
	}
	if got := providers.TextIn(steps[0].Label, steps[0].LabelText, vi); got != "Tạo SNS topic `reputecast-events` và đăng ký `https://rc.acme.io/hooks/ses/t`" {
		t.Errorf("Vietnamese: %q", got)
	}
	account, text := say(mAccount, i18n.Args{"mode": "production", "enforcement": "none"})
	if account != "Production · no enforcement" || providers.TextIn(account, text, vi) != "Gửi chính thức · không bị hạn chế" {
		t.Errorf("account: %q / %q", account, providers.TextIn(account, text, vi))
	}
}
