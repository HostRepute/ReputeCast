package campaigns_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The composer reads its content checks, its layout warnings and its token
// menu in the member's language; API clients without Accept-Language keep
// the English.
func TestComposerChecksInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	ws := "/workspaces/acme-newsletter"
	flash := campaignID(t, env, "Flash sale: 48 hours only")
	var d detail
	en.Do("GET", ws+"/campaigns/"+flash, nil).MustStatus(t, 200).JSON(t, &d)

	type issue struct{ ID, Title, Detail string }
	type checks struct{ Issues []issue }
	body := map[string]any{"subject": "", "preheader": d.Preheader, "mjml": d.MJML, "html": d.HTML}
	titleOf := func(c checks, id string) string {
		for _, is := range c.Issues {
			if is.ID == id {
				return is.Title
			}
		}
		return ""
	}
	var vc, ec checks
	vi.Do("POST", ws+"/campaigns/"+flash+"/checks", body).MustStatus(t, 200).JSON(t, &vc)
	en.Do("POST", ws+"/campaigns/"+flash+"/checks", body).MustStatus(t, 200).JSON(t, &ec)
	if got := titleOf(vc, "subject.missing"); got != "Tiêu đề đang trống" {
		t.Errorf("subject.missing in Vietnamese: %q", got)
	}
	if got := titleOf(ec, "subject.missing"); got != "The subject is empty" {
		t.Errorf("subject.missing in English: %q", got)
	}

	// The layout warnings of the compile endpoint.
	src := `<mjml><mj-body><mj-section><mj-column width="70%"><mj-text>a</mj-text></mj-column><mj-column width="50%"><mj-text>b</mj-text></mj-column></mj-section></mj-body></mjml>`
	var compiled struct{ Warnings []struct{ Message string } }
	vi.Do("POST", ws+"/mjml/compile", map[string]any{"mjml": src}).MustStatus(t, 200).JSON(t, &compiled)
	if len(compiled.Warnings) == 0 || !strings.HasPrefix(compiled.Warnings[0].Message, "Dòng ") {
		t.Errorf("lint in Vietnamese: %+v", compiled.Warnings)
	}
	en.Do("POST", ws+"/mjml/compile", map[string]any{"mjml": src}).MustStatus(t, 200).JSON(t, &compiled)
	if len(compiled.Warnings) == 0 || !strings.HasPrefix(compiled.Warnings[0].Message, "Line ") {
		t.Errorf("lint in English: %+v", compiled.Warnings)
	}

	// The token menu names the contact fields in the member's language and
	// suggests the first-name fallback the email's readers read.
	var ctx struct{ Tokens []struct{ Key, Label, Fallback string } }
	vi.Do("GET", ws+"/composer", nil).MustStatus(t, 200).JSON(t, &ctx)
	for _, tok := range ctx.Tokens {
		if tok.Key == "first_name" && (tok.Label != "Tên" || tok.Fallback != "bạn") {
			t.Errorf("first_name token in Vietnamese: %+v", tok)
		}
		if tok.Key == "view_in_browser_url" && tok.Label != "Liên kết xem trên trình duyệt" {
			t.Errorf("view_in_browser_url token in Vietnamese: %+v", tok)
		}
	}
	en.Do("GET", ws+"/composer", nil).MustStatus(t, 200).JSON(t, &ctx)
	for _, tok := range ctx.Tokens {
		if tok.Key == "first_name" && (tok.Label != "First name" || tok.Fallback != "there") {
			t.Errorf("first_name token in English: %+v", tok)
		}
	}
}
