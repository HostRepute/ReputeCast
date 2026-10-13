package mjml

import (
	"strings"
	"testing"
)

const doc = `<mjml>
  <mj-head>
    <mj-title>Old</mj-title>
  </mj-head>
  <mj-body width="600px">
    <!-- rc:header-logo -->
    <mj-section padding="20px 24px">
      <mj-column width="40%"><mj-image src="/media/1/logo.png" alt="Acme" width="96px"/></mj-column>
      <mj-column width="60%"><mj-text>Blog · Docs</mj-text></mj-column>
    </mj-section>
    <mj-section padding="20px 24px">
      <mj-column>
        <mj-image src="/media/1/hero.png" alt='The "Q4" roadmap' width="552px" padding="0px">
        <mj-text>Hi {{ first_name | default: "there" }}</mj-text>
      </mj-column>
    </mj-section>
    <mj-section padding="16px 24px">
      <mj-column width="50%"><mj-image src="/a.png" width="600px"></mj-image></mj-column>
      <mj-column width="50%"><mj-button href="https://x.io">Go</mj-button></mj-column>
    </mj-section>
    <mj-section><mj-column><mj-divider/></mj-column></mj-section>
    <!-- rc:footer -->
    <mj-section><mj-column><mj-text>Acme Inc · 500 Market St<br><a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>
  </mj-body>
</mjml>`

func TestParseAndOutline(t *testing.T) {
	root, err := Parse(doc)
	if err != nil {
		t.Fatal(err)
	}
	body := Body(root)
	if body == nil || len(body.Children) != 5 {
		t.Fatalf("body: %+v", body)
	}
	hero := body.Children[1].Children[0].Children[0]
	if hero.Tag != "mj-image" || hero.Attr("alt") != `The "Q4" roadmap` {
		t.Fatalf("void image closed by its sibling: %+v", hero)
	}
	if body.Children[1].Children[0].Children[1].Tag != "mj-text" {
		t.Fatal("text after an unclosed image must be its sibling")
	}
	secs, err := Outline(doc)
	if err != nil {
		t.Fatal(err)
	}
	labels := []string{}
	for _, s := range secs {
		labels = append(labels, s.Label)
	}
	if strings.Join(labels, ",") != "header,hero,2 columns,divider,footer" {
		t.Fatalf("labels: %v", labels)
	}
}

func TestParseErrors(t *testing.T) {
	_, err := Parse("<mjml>\n<mj-body>\n<mj-section>\n<mj-column>\n<mj-text>x</mj-text>\n</mj-section>\n</mj-body>\n</mjml>")
	if err == nil || err.Error() != "Line 4: unclosed <mj-column>" {
		t.Fatalf("unclosed column: %v", err)
	}
	if _, err := Parse("<mj-body></mj-body>"); err == nil {
		t.Fatal("documents start with <mjml>")
	}
}

func TestLint(t *testing.T) {
	p := Lint(doc)
	if len(p) != 1 || !strings.Contains(p[0].Message, "mj-image width 600px exceeds the 50% column (276px)") || p[0].Line != 18 {
		t.Fatalf("lint: %+v", p)
	}
}

func TestHeadMeta(t *testing.T) {
	out := SetHeadMeta(doc, `Three launches & "one" deprecation`, "Preview text")
	title, preview := HeadMeta(out)
	if title != `Three launches & "one" deprecation` || preview != "Preview text" {
		t.Fatalf("head: %q %q", title, preview)
	}
	if strings.Count(out, "<mj-title>") != 1 {
		t.Fatal("title duplicated")
	}
	noHead := SetHeadMeta(`<mjml><mj-body></mj-body></mjml>`, "S", "")
	if !strings.Contains(noHead, "<mj-head>") || !strings.Contains(noHead, "<mj-title>S</mj-title>") || strings.Contains(noHead, "mj-preview") {
		t.Fatalf("no head: %s", noHead)
	}
}

func TestCompileKeepsQuotesEscaped(t *testing.T) {
	res := Compile(`<mjml><mj-body><mj-section><mj-column><mj-image src="https://x.io/a.png" alt="A &quot;quoted&quot; &lt;b&gt;" /><mj-text>Hi<br>there</mj-text></mj-column></mj-section></mj-body></mjml>`)
	if len(res.Errors) > 0 {
		t.Fatalf("compile: %+v", res.Errors)
	}
	if !strings.Contains(res.HTML, `alt="A &quot;quoted&quot; &lt;b&gt;"`) {
		t.Fatalf("alt not escaped: %s", res.HTML[strings.Index(res.HTML, "<img"):strings.Index(res.HTML, "<img")+80])
	}
	bad := Compile("<mjml><mj-body><mj-section><mj-column><mj-text>x</mj-section></mj-body></mjml>")
	if len(bad.Errors) != 1 || bad.Errors[0].Line != 1 {
		t.Fatalf("errors: %+v", bad.Errors)
	}
}

func TestThumbAndChanges(t *testing.T) {
	th := Thumb(doc)
	if len(th) != 4 || th[0].H != 10 || th[0].W != 40 || len(th[2].Row) != 2 {
		t.Fatalf("thumb: %+v", th)
	}
	changed := strings.Replace(doc, "Hi {{", "Hello {{", 1)
	if got := ChangedSections(doc, changed); len(got) != 1 || got[0] != "hero" {
		t.Fatalf("changed: %v", got)
	}
}
