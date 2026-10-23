package contentcheck

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// A personalization tag that doesn't parse names its line in the reader's
// language; the template engine's own explanation after it stays English.
func TestLiquidProblemLineInVietnamese(t *testing.T) {
	r, err := (&Checker{}).Check(context.Background(), Input{Subject: "{% if first_name %}Hi", MJML: `<mjml><mj-body><mj-section><mj-column><mj-text>Hi</mj-text></mj-column></mj-section></mj-body></mjml>`})
	if err != nil {
		t.Fatal(err)
	}
	var en, vi string
	for _, is := range r.Issues {
		if is.ID == "personalization.syntax" {
			en, vi = is.Title, is.In(i18n.For(i18n.Vietnamese)).Title
			break
		}
	}
	if !strings.HasPrefix(en, "Subject: Line 1: ") {
		t.Fatalf("English title: %q", en)
	}
	if want := "Tiêu đề: Dòng 1: " + strings.TrimPrefix(en, "Subject: Line 1: "); vi != want {
		t.Fatalf("Vietnamese title: %q, want %q", vi, want)
	}
}
