package mjml_test

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/mjml"
)

func TestGeneratedEmailBoundary(t *testing.T) {
	wrap := func(body string) string {
		return `<mjml><mj-body><mj-section><mj-column>` + body + `</mj-column></mj-section></mj-body></mjml>`
	}
	policy := mjml.GeneratedPolicy{Links: []string{"https://example.test/read?a=1&b=2", "{{ unsubscribe_url }}"}, Images: []string{"https://example.test/banner.png"}}
	valid := wrap(`<mj-text font-family="Georgia, 'Times New Roman', serif"><p style="color:#123456;font-weight:600">Hello {{ first_name | default: "there" }}.</p><a href="https://example.test/read?a=1&amp;b=2">Read</a></mj-text><mj-image src="https://example.test/banner.png" alt="New release"/><mj-button href="{{ unsubscribe_url }}">Unsubscribe</mj-button>`)
	if err := mjml.CheckGenerated(valid, policy); err != nil {
		t.Fatal(err)
	}
	if result := mjml.Compile(valid); len(result.Errors) != 0 || !strings.Contains(result.HTML, "New release") {
		t.Fatalf("allowed email must compile: %+v", result.Errors)
	}
	for _, document := range []string{
		valid + valid,
		`<!-- outside root -->` + valid,
		strings.Replace(valid, "<mj-body>", `<mj-body><!--[if mso]><script>alert(1)</script><![endif]-->`, 1),
		strings.Replace(valid, "<mj-body>", `<mj-body><?unsafe processing?>`, 1),
	} {
		if err := mjml.CheckGenerated(document, policy); err == nil {
			t.Error("uninspected document markup accepted")
		}
	}
	for _, body := range []string{
		`<mj-raw><script>alert(1)</script></mj-raw>`,
		`<mj-include path="https://example.test/include"/>`,
		`<mj-text><img src=x onerror=alert(1)></mj-text>`,
		`<mj-text><a href="https://other.test/">Other</a></mj-text>`,
		`<mj-text><a href="jav&#x61;script:alert(1)">Other</a></mj-text>`,
		`<mj-text><svg><a href="https://example.test/read?a=1&amp;b=2">Other</a></svg></mj-text>`,
		`<mj-text><a href="https://example.test/read?a=1&amp;b=2" onclick="alert(1)">Read</a></mj-text>`,
		`<mj-text><p style="background-image:url(https://private.test/)">Hidden fetch</p></mj-text>`,
		`<mj-text><p style="color:r\65 d">Escaped CSS</p></mj-text>`,
		`<mj-text><p style="width:expression(alert(1))">Expression</p></mj-text>`,
		`<mj-text><!--[if mso]><script>alert(1)</script><![endif]--></mj-text>`,
		`<mj-text rc-visible="arbitrary">Conditional</mj-text>`,
		`<mj-image src="https://example.test/banner.png" srcset="https://other.test/a 2x"/>`,
		`<mj-text color="red&#34; onclick=&#34;alert(1)">Injection</mj-text>`,
	} {
		t.Run(body, func(t *testing.T) {
			if err := mjml.CheckGenerated(wrap(body), policy); err == nil {
				t.Fatal("unsafe generated markup accepted")
			}
		})
	}
	// An allowlisted string does not make a dangerous URL safe.
	for _, address := range []string{"javascript:alert(1)", "data:text/html,hello", "https://user:password@example.test/", "//example.test/", "https://example.test/\n"} {
		if err := mjml.CheckGenerated(wrap(`<mj-button href="`+address+`">Read</mj-button>`), mjml.GeneratedPolicy{Links: []string{address}}); err == nil {
			t.Errorf("unsafe allowlisted URL accepted: %q", address)
		}
	}
}
