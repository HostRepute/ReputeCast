package engine

import (
	"strings"
	"testing"
)

// Earn's personal links (design expansion X16) are filled in per reader
// like the unsubscribe link and never go through the click redirect: the
// upgrade page credits the email from its own token.
func TestPaidLinksArentRedirected(t *testing.T) {
	html := `<html><body>
<a href="https://acme.io/blog">Blog</a>
<a href="{{ upgrade_url }}">Start your free trial</a>
<a href="{{ manage_subscription_url }}">Manage your subscription</a>
</body></html>`
	tpl, err := Prepare(Content{Subject: "Hi", HTML: html}, Tracking{Opens: true, Clicks: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(tpl.Links) != 1 || tpl.Links[0].URL != "https://acme.io/blog" {
		t.Fatalf("only the blog link is tracked: %+v", tpl.Links)
	}
	vars := map[string]any{"upgrade_url": "https://news.acme.io/upgrade?r=TOKEN", "manage_subscription_url": "https://news.acme.io/billing/BILL",
		OpenVar: "https://x/t/o/OPEN", LinkVar(0): "https://x/t/c/L0"}
	out, err := tpl.Render(vars, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`href="https://x/t/c/L0"`, `href="https://news.acme.io/upgrade?r=TOKEN"`, `href="https://news.acme.io/billing/BILL"`} {
		if !strings.Contains(out.HTML, want) {
			t.Errorf("rendered HTML lacks %s:\n%s", want, out.HTML)
		}
	}
}
