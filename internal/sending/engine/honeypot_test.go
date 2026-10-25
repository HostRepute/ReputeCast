package engine

import (
	"strings"
	"testing"
)

func TestHoneypotInTrackedEmails(t *testing.T) {
	tpl, err := Prepare(Content{Subject: "s", HTML: sampleHTML}, Tracking{Opens: true, Clicks: true})
	if err != nil {
		t.Fatal(err)
	}
	vars := map[string]any{OpenVar: "https://x/t/o/O", LinkVar(0): "https://x/t/c/0", LinkVar(1): "https://x/t/c/1", LinkVar(2): "https://x/t/c/2",
		HoneypotVar: "https://x/t/h/H"}
	out, err := tpl.Render(vars, true)
	if err != nil {
		t.Fatal(err)
	}
	at := strings.Index(out.HTML, `href="https://x/t/h/H"`)
	if at < 0 || at > strings.Index(out.HTML, "</body>") {
		t.Fatalf("the hidden link isn't in the body:\n%s", out.HTML)
	}
	tag := out.HTML[strings.LastIndex(out.HTML[:at], "<a"):]
	tag = tag[:strings.Index(tag, "</a>")]
	for _, want := range []string{`aria-hidden="true"`, `tabindex="-1"`, "display:none"} {
		if !strings.Contains(tag, want) {
			t.Errorf("the hidden link lacks %s: %s", want, tag)
		}
	}
	if strings.TrimSpace(tag[strings.Index(tag, ">")+1:]) != "" {
		t.Errorf("the hidden link has text: %s", tag)
	}
	for _, l := range tpl.Links {
		if strings.Contains(l.URL, HoneypotVar) {
			t.Fatal("the hidden link is one of the email's tracked links")
		}
	}
	plain, err := tpl.Render(vars, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain.HTML, "/t/h/") {
		t.Fatal("the hidden link is in the email with the provider's tracking")
	}
	noClicks, err := Prepare(Content{Subject: "s", HTML: sampleHTML}, Tracking{Opens: true})
	if err != nil {
		t.Fatal(err)
	}
	if out, _ := noClicks.Render(vars, true); strings.Contains(out.HTML, "/t/h/") {
		t.Fatal("the hidden link is there with click tracking off")
	}
}
