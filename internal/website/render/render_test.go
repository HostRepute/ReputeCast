package render

import (
	"strings"
	"testing"
)

const digest = `<mjml>
  <mj-head><mj-title>Weekly Digest</mj-title></mj-head>
  <mj-body>
    <mj-section><mj-column>
      <mj-text font-size="11px">Trouble reading? <a href="{{ view_in_browser_url }}">View in your browser</a></mj-text>
    </mj-column></mj-section>
    <!-- rc:header-logo -->
    <mj-section><mj-column><mj-image src="/media/abc/logo.png" alt="Acme"></mj-image></mj-column></mj-section>
    <mj-section><mj-column>
      <mj-image src="https://cdn.example/hero.png" alt="Rows of switches, most of them on"></mj-image>
      <mj-text font-size="26px" font-weight="700"><h1 style="margin:0;font-size:26px">Feature flags at scale</h1></mj-text>
      <mj-text>Hi {{ first_name | default: "there" }}, here's what shipped <b style="color:red">this week</b>.</mj-text>
      <mj-button href="https://docs.acme.io/flags">Read the docs</mj-button>
    </mj-column></mj-section>
    <mj-section rc-name="Beta corner" rc-visible='{"id":"beta01","show":"only","conditions":[{"kind":"list","op":"includes","value":"x","label":"Beta Program"}]}'>
      <mj-column><mj-text>For Beta Program readers only.</mj-text></mj-column>
    </mj-section>
    <mj-section rc-visible='{"id":"paid01","show":"only","conditions":[{"kind":"paid_tier","op":"is","value":""}]}'>
      <mj-column><mj-text>Insider extras: the runbook template.</mj-text></mj-column>
    </mj-section>
    <mj-section rc-show-on="web"><mj-column><mj-text>Only on the web.</mj-text></mj-column></mj-section>
    <mj-section rc-show-on="email"><mj-column><mj-text>Only in the inbox.</mj-text></mj-column></mj-section>
    <mj-section>
      <mj-column><mj-text>Left column</mj-text></mj-column>
      <mj-column><mj-text>Right column</mj-text></mj-column>
    </mj-section>
    <!-- rc:footer -->
    <mj-section><mj-column><mj-text>Acme Inc · <a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>
  </mj-body>
</mjml>`

func TestRenderDrawsTheEmailInTheSitesTheme(t *testing.T) {
	res, err := Render(digest, Options{Title: "Feature flags at scale", CoverSrc: "https://cdn.example/hero.png"})
	if err != nil {
		t.Fatal(err)
	}
	body := res.Body
	for _, want := range []string{
		"<p>Hi there, here&#39;s what shipped <strong>this week</strong>.</p>",
		`<a class="st-btn" href="https://docs.acme.io/flags">Read the docs</a>`,
		"Only on the web.",
		`<div class="st-cols"><div><p>Left column</p></div><div><p>Right column</p></div></div>`,
		PaidPlaceholder("paid01"),
		`<img src="/media/abc/logo.png" alt="Acme"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body lacks %q\n%s", want, body)
		}
	}
	for _, gone := range []string{"style=", "View in your browser", "Unsubscribe", "Beta Program readers", "Only in the inbox", "hero.png", "<h1", "Insider extras", "{{"} {
		if strings.Contains(body, gone) {
			t.Errorf("body keeps %q\n%s", gone, body)
		}
	}
	if len(res.Paid) != 1 || res.Paid[0].ID != "paid01" || !strings.Contains(res.Paid[0].HTML, "Insider extras") || len(res.Paid[0].Tiers) != 0 {
		t.Fatalf("paid blocks: %+v", res.Paid)
	}
	kinds := map[string]string{}
	for _, h := range res.Hidden {
		kinds[h.Kind+"/"+h.Reason] = h.Name + h.Condition
	}
	if _, ok := kinds["view_online/email_only"]; !ok {
		t.Errorf("view online not hidden: %+v", res.Hidden)
	}
	if _, ok := kinds["footer/email_only"]; !ok {
		t.Errorf("footer not hidden: %+v", res.Hidden)
	}
	if got := kinds["block/conditional"]; got != "Beta cornerBeta Program" {
		t.Errorf("conditional block: %q (%+v)", got, res.Hidden)
	}
	if _, ok := kinds["block/email_only"]; !ok {
		t.Errorf("email-only block not listed: %+v", res.Hidden)
	}
	if len(res.Fallbacks) != 1 || res.Fallbacks[0].Token != "first_name" || res.Fallbacks[0].Sample != "Hi there" ||
		res.Fallbacks[0].Source != `Hi {{ first_name | default: "there" }}` {
		t.Fatalf("fallbacks: %+v", res.Fallbacks)
	}
	if res.HasMarker || res.Preview != "" {
		t.Fatalf("no marker expected: %+v", res)
	}
	if res.Words == 0 || strings.Contains(res.Text, "Insider extras") {
		t.Fatalf("text: %d words, %q", res.Words, res.Text)
	}
}

func TestFreePreviewMarker(t *testing.T) {
	src := `<mjml><mj-body><mj-section><mj-column>
	  <mj-text><p>One.</p></mj-text>
	  <mj-text><p>Two.</p></mj-text>
	  <mj-text><p>Three.</p></mj-text>
	  <mj-raw rc-preview="end"><!-- rc:preview-end --></mj-raw>
	  <mj-text><h2>1. Size the target</h2><p>Secret runbook step.</p></mj-text>
	</mj-column></mj-section>
	<mj-section><mj-column><mj-text>After.</mj-text></mj-column><mj-column><mj-text>Also after.</mj-text></mj-column></mj-section>
	</mj-body></mjml>`
	res, err := Render(src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.HasMarker || res.MarkerAfter != 3 {
		t.Fatalf("marker: %v after %d", res.HasMarker, res.MarkerAfter)
	}
	if !strings.Contains(res.Preview, "Three.") || strings.Contains(res.Preview, "Secret") || strings.Contains(res.PreviewText, "Secret") {
		t.Fatalf("preview leaks the gated text: %q / %q", res.Preview, res.PreviewText)
	}
	if !strings.Contains(res.Body, "Secret runbook step.") || !strings.Contains(res.Text, "Secret") {
		t.Fatalf("body misses the rest: %q", res.Body)
	}
}

func TestMarkerInsideColumnsClosesThem(t *testing.T) {
	src := `<mjml><mj-body><mj-section>
	  <mj-column><mj-text>Left.</mj-text><mj-raw><!-- rc:preview-end --></mj-raw><mj-text>Hidden left.</mj-text></mj-column>
	  <mj-column><mj-text>Hidden right.</mj-text></mj-column>
	</mj-section></mj-body></mjml>`
	res, err := Render(src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Preview != `<div class="st-cols"><div><p>Left.</p></div></div>` {
		t.Fatalf("preview: %q", res.Preview)
	}
}

func TestSanitizeDropsUnsafeMarkup(t *testing.T) {
	src := `<mjml><mj-body><mj-section><mj-column>
	  <mj-text><p onclick="x()">Hello <script>alert(1)</script><a href="javascript:alert(1)">bad</a> <a href="https://ok.example/">ok</a><iframe src="https://evil"></iframe><img src="data:image/png;base64,AAA" alt="d"></p></mj-text>
	  <mj-raw><div>raw email html</div></mj-raw>
	</mj-column></mj-section></mj-body></mjml>`
	res, err := Render(src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"onclick", "<script", "alert", "javascript:", "<iframe", "data:image", "raw email html"} {
		if strings.Contains(res.Body, bad) {
			t.Errorf("body keeps %q: %s", bad, res.Body)
		}
	}
	if !strings.Contains(res.Body, `<a href="https://ok.example/">ok</a>`) || !strings.Contains(res.Body, "Hello bad") {
		t.Fatalf("body: %s", res.Body)
	}
}

func TestPaidRuleForChosenTiers(t *testing.T) {
	v := visRule{ID: "p", Show: "only", Match: "any"}
	v.Conditions = append(v.Conditions, struct {
		Kind  string `json:"kind"`
		Op    string `json:"op"`
		Value string `json:"value"`
		Label string `json:"label"`
	}{Kind: PaidTierKind, Op: "is", Value: "tier-a"})
	tiers, ok := paidRule(v)
	if !ok || len(tiers) != 1 || tiers[0] != "tier-a" {
		t.Fatalf("paid rule: %v %v", tiers, ok)
	}
	v.Conditions[0].Kind = "list"
	if _, ok := paidRule(v); ok {
		t.Fatal("a list rule isn't a paid rule")
	}
	if got := ReadingMinutes(2480); got != 11 {
		t.Fatalf("reading minutes: %d", got)
	}
}
