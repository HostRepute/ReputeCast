package whatsapp

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// A Send WhatsApp step's issues read in the member's language on the
// canvas; their English stays as it was.
func TestStepIssuesInTheMembersLanguage(t *testing.T) {
	n := automations.AutomationNode{Data: automations.AutomationNodeData{WhatsApp: &automations.AutomationWhatsAppStep{Mode: "text", OutsideWindow: "template"}}}
	issues := Step{}.Validate(n)
	if len(issues) < 2 || issues[0].Message != "Write the message, or pick an approved template." || issues[1].Message != "Pick the template sent outside the 24-hour window." {
		t.Fatalf("English: %+v", issues)
	}
	vi := i18n.For(i18n.Vietnamese)
	if got := issues[0].Text.In(vi); got != "Hãy viết tin nhắn, hoặc chọn một mẫu đã được duyệt." {
		t.Errorf("text: %q", got)
	}
	if got := issues[1].Text.In(vi); got != "Hãy chọn mẫu được gửi ngoài khung 24 giờ." {
		t.Errorf("fallback: %q", got)
	}
}
