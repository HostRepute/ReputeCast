package push

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// A Send push step's issues read in the member's language on the canvas;
// their English stays as it was.
func TestStepIssuesInTheMembersLanguage(t *testing.T) {
	n := automations.AutomationNode{Data: automations.AutomationNodeData{Push: &automations.AutomationPushStep{Title: strings.Repeat("a", TitleMax+1), Link: "not a link"}}}
	issues := Step{}.Validate(n)
	if len(issues) != 2 || issues[0].Message != "Keep the title to 50 characters: longer titles are cut on every platform." ||
		issues[1].Message != "Enter the address the notification opens, like news.acme.io/summit/live." {
		t.Fatalf("English: %+v", issues)
	}
	vi := i18n.For(i18n.Vietnamese)
	if got := issues[0].Text.In(vi); got != "Giữ tiêu đề trong 50 ký tự: tiêu đề dài hơn sẽ bị cắt trên mọi nền tảng." {
		t.Errorf("title: %q", got)
	}
	if got := issues[1].Text.In(vi); got == "" || got == issues[1].Message {
		t.Errorf("link: %q", got)
	}
}
