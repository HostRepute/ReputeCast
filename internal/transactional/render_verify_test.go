package transactional

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/mjml"
)

// The editor's HTML is sent only when it shows and does what the template's
// MJML says (composer review, CO-3): a link swapped in the HTML alone is
// replaced by the server's own compile.
func TestCompiledHTMLChecksTheEditorsHTML(t *testing.T) {
	src := `<mjml><mj-body><mj-section><mj-column><mj-text>Reset your password: <a href="{{ reset_url }}">reset</a></mj-text></mj-column></mj-section></mj-body></mjml>`
	honest := mjml.Compile(src).HTML
	c := content{MJML: src, HTML: honest}
	if got, err := c.compiledHTML(); err != nil || got != honest {
		t.Fatalf("honest HTML must be kept: %v", err)
	}
	c.HTML = strings.Replace(honest, "{{ reset_url }}", "https://evil.example/reset", 1)
	got, err := c.compiledHTML()
	if err != nil || strings.Contains(got, "evil.example") || !strings.Contains(got, "{{ reset_url }}") {
		t.Fatalf("tampered HTML must be recompiled: %v", err)
	}
	// HTML-only templates have nothing to check against.
	c = content{HTML: "<p>Raw</p>"}
	if got, _ := c.compiledHTML(); got != "<p>Raw</p>" {
		t.Fatal("HTML-only template")
	}
}
