package campaigns_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/mjml"
)

const conditionalDoc = `<mjml><mj-body>` +
	`<mj-section><mj-column><mj-text>Hi {{ first_name | default: "there" }}, the week in short.</mj-text></mj-column></mj-section>` +
	`<mj-section rc-visible='{"id":"beta1","show":"match","rules":[{"field":"plan","op":"is","value":"team"}]}'><mj-column><mj-text>Beta corner: <a href="https://acme.io/beta">try it</a>.</mj-text></mj-column></mj-section>` +
	`<mj-section><mj-column><mj-text>Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105 <a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>` +
	`</mj-body></mjml>`

// CO-3: the server checks the HTML it stores against its own compile of
// the MJML. Honest HTML is kept; HTML that shows or does something else
// (another link, missing conditional markers) is replaced by the server's
// compile, and MJML that doesn't parse is refused.
func TestComposerChecksStoredHTML(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var d detail
	c.Do("POST", ws+"/campaigns", map[string]any{"name": "Checked HTML"}).MustStatus(t, 201).JSON(t, &d)
	honest := mjml.Compile(conditionalDoc).HTML

	type saved struct {
		Version    int
		Recompiled string
		HTML       string
	}
	var s saved
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 1, "subject": "One", "mjml": conditionalDoc, "html": honest}).MustStatus(t, 200).JSON(t, &s)
	if s.Recompiled != "" || !strings.Contains(s.HTML, "<!--rc:if beta1-->") {
		t.Fatalf("honest HTML: %+v", s.Recompiled)
	}

	// A link the MJML doesn't have.
	phished := strings.Replace(honest, "https://acme.io/beta", "https://evil.example/login", 1)
	s = saved{}
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 2, "subject": "Two", "mjml": conditionalDoc, "html": phished}).MustStatus(t, 200).JSON(t, &s)
	if s.Recompiled != "links" || strings.Contains(s.HTML, "evil.example") || !strings.Contains(s.HTML, "https://acme.io/beta") {
		t.Fatalf("phished HTML: %q", s.Recompiled)
	}
	c.Do("GET", ws+"/campaigns/"+d.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if strings.Contains(d.HTML, "evil.example") {
		t.Fatal("the mismatched HTML was stored")
	}
	var meta string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT metadata->>'recompiled' FROM audit_events WHERE action = 'campaign.content.save' AND target_id = $1 ORDER BY occurred_at DESC LIMIT 1`, d.ID).Scan(&meta); err != nil || meta != "links" {
		t.Fatalf("audit metadata: %q %v", meta, err)
	}

	// Markers dropped: the beta corner would reach everyone.
	unmarked := strings.NewReplacer("<!--rc:if beta1-->", "", "<!--rc:fi beta1-->", "").Replace(honest)
	s = saved{}
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 3, "subject": "Three", "mjml": conditionalDoc, "html": unmarked}).MustStatus(t, 200).JSON(t, &s)
	if s.Recompiled != "conditional markers" || !strings.Contains(s.HTML, "<!--rc:if beta1-->") {
		t.Fatalf("unmarked HTML: %q", s.Recompiled)
	}

	// A content variant gets the same check.
	variantB := strings.Replace(conditionalDoc, "the week in short", "what shipped", 1)
	s = saved{}
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 4, "subject": "Four", "mjml": conditionalDoc, "html": honest,
		"abTest": map[string]any{"kind": "content", "testPercent": 20, "metric": "clicks", "durationHours": 2, "variants": []map[string]any{
			{"key": "A"}, {"key": "B", "mjml": variantB, "html": strings.Replace(mjml.Compile(variantB).HTML, "https://acme.io/beta", "https://evil.example/b", 1)},
		}}}).MustStatus(t, 200).JSON(t, &s)
	if s.Recompiled != "B: links" {
		t.Fatalf("variant B: %q", s.Recompiled)
	}
	var ab struct {
		ABTest struct{ Variants []struct{ Key, HTML string } }
	}
	c.Do("GET", ws+"/campaigns/"+d.ID, nil).MustStatus(t, 200).JSON(t, &ab)
	for _, v := range ab.ABTest.Variants {
		if strings.Contains(v.HTML, "evil.example") {
			t.Fatalf("variant %s kept the mismatched HTML", v.Key)
		}
	}

	// A language version gets the same check.
	c.Do("POST", ws+"/campaigns/"+d.ID+"/languages", map[string]any{"language": "es", "start": "copy"}).MustStatus(t, 201)
	var es struct {
		Version int
		HTML    string
	}
	c.Do("GET", ws+"/campaigns/"+d.ID+"?language=es", nil).MustStatus(t, 200).JSON(t, &es)
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content?language=es", map[string]any{"baseVersion": es.Version, "subject": "Uno", "mjml": conditionalDoc, "html": phished}).MustStatus(t, 200)
	c.Do("GET", ws+"/campaigns/"+d.ID+"?language=es", nil).MustStatus(t, 200).JSON(t, &es)
	if strings.Contains(es.HTML, "evil.example") || !strings.Contains(es.HTML, "https://acme.io/beta") {
		t.Fatal("the Spanish version kept the mismatched HTML")
	}

	// MJML that doesn't parse is refused, HTML or not.
	broken := `<mjml><mj-body><mj-section><mj-column><mj-text>x</mj-section></mj-body></mjml>`
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 5, "subject": "Five", "mjml": broken, "html": "<p>x</p>"}).MustStatus(t, 422)

	// A template's HTML is checked too, and drafts started from it store the checked HTML.
	var tpl struct{ ID, HTML string }
	c.Do("POST", ws+"/templates", map[string]any{"name": "Checked template", "mjml": conditionalDoc, "html": phished}).MustStatus(t, 201).JSON(t, &tpl)
	if strings.Contains(tpl.HTML, "evil.example") {
		t.Fatal("the template kept the mismatched HTML")
	}
	var fromTpl detail
	c.Do("POST", ws+"/campaigns", map[string]any{"templateId": tpl.ID}).MustStatus(t, 201).JSON(t, &fromTpl)
	if strings.Contains(fromTpl.HTML, "evil.example") || !strings.Contains(fromTpl.HTML, "<!--rc:if beta1-->") {
		t.Fatal("a draft from the template must store checked HTML")
	}
}
