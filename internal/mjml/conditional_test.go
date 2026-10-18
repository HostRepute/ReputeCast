package mjml

import (
	"strings"
	"testing"
)

const conditionalDoc = `<mjml>
  <mj-body width="600px">
    <mj-section padding="10px"><mj-column><mj-text>Header</mj-text></mj-column></mj-section>
    <mj-section rc-name="Beta corner" rc-visible="{&quot;id&quot;:&quot;beta01&quot;,&quot;show&quot;:&quot;only&quot;,&quot;match&quot;:&quot;all&quot;,&quot;conditions&quot;:[]}" padding="10px">
      <mj-column><mj-text>Beta corner</mj-text></mj-column>
    </mj-section>
    <mj-section padding="10px">
      <mj-column><mj-text>Always</mj-text><mj-text rc-visible='{"id":"txt002","show":"except"}'>Except some</mj-text></mj-column>
      <mj-column rc-visible='{"id":"col003","show":"only"}'><mj-text>Second column</mj-text></mj-column>
    </mj-section>
    <mj-section rc-visible='{"id":"evry04","show":"everyone"}'><mj-column><mj-text>Everyone</mj-text></mj-column></mj-section>
    <mj-section><mj-column><mj-text>Footer</mj-text></mj-column></mj-section>
  </mj-body>
</mjml>`

func TestPrepareCompile(t *testing.T) {
	out := PrepareCompile(conditionalDoc)
	if strings.Contains(out, "rc-") && strings.Contains(out, "rc-visible") {
		t.Fatalf("rc attributes left:\n%s", out)
	}
	for _, id := range []string{"beta01", "txt002", "col003"} {
		if !strings.Contains(out, "<mj-raw>"+MarkerIf+id+"--></mj-raw>") || !strings.Contains(out, "<mj-raw>"+MarkerFi+id+"--></mj-raw>") {
			t.Fatalf("markers of %s missing:\n%s", id, out)
		}
	}
	if strings.Contains(out, "evry04") {
		t.Fatalf("a block shown to everyone is marked:\n%s", out)
	}
	if _, err := Parse(out); err != nil {
		t.Fatalf("prepared document doesn't parse: %v\n%s", err, out)
	}
	// Nothing to do: the document is returned as is.
	plain := `<mjml><mj-body><mj-section><mj-column><mj-text>Hi</mj-text></mj-column></mj-section></mj-body></mjml>`
	if PrepareCompile(plain) != plain {
		t.Fatal("a document without rc attributes changed")
	}
}

// gomjml keeps the markers, outside Outlook's conditional comments, around
// each conditional block's HTML.
func TestCompileKeepsMarkers(t *testing.T) {
	res := Compile(conditionalDoc)
	if len(res.Errors) > 0 {
		t.Fatalf("compile: %+v", res.Errors)
	}
	h := res.HTML
	for _, id := range []string{"beta01", "txt002", "col003"} {
		start := strings.Index(h, MarkerIf+id+"-->")
		end := strings.Index(h, MarkerFi+id+"-->")
		if start < 0 || end < start {
			t.Fatalf("markers of %s: %d, %d\n%s", id, start, end, h)
		}
		// The marker is never inside an Outlook conditional comment.
		before := h[:start]
		if strings.Count(before, "<!--[if") != strings.Count(before, "<![endif]-->") {
			t.Fatalf("marker of %s is inside a conditional comment:\n%s", id, h)
		}
	}
	inner := h[strings.Index(h, MarkerIf+"beta01-->"):strings.Index(h, MarkerFi+"beta01-->")]
	if !strings.Contains(inner, "Beta corner") || strings.Contains(inner, "Always") {
		t.Fatalf("beta01 wraps the wrong HTML:\n%s", inner)
	}
	if strings.Contains(h, "rc-visible") || strings.Contains(h, "rc-name") {
		t.Fatal("rc attributes reached the HTML")
	}
	if StripMarkers(h) == h || strings.Contains(StripMarkers(h), "<!--rc:") {
		t.Fatal("StripMarkers left markers")
	}
}
