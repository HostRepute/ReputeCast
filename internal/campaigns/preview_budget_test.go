package campaigns_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// A preview renders Liquid on the server. A template that loops on a range
// read from a variable once kept a core busy for good (and one that doubled
// a string killed the process); now the render stops on its budget and the
// preview says why. People who can't edit the email see it as saved: what
// they send in the request isn't rendered.
func TestPreviewRendersOnABudget(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ws := "/workspaces/acme-newsletter"
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var d struct{ ID string }
	linh.Do("POST", ws+"/campaigns", map[string]any{"name": "Budget"}).MustStatus(t, 201).JSON(t, &d)

	type preview struct {
		Subject  string
		HTML     string
		Problems []struct{ Message string }
	}
	start := time.Now()
	var pv preview
	linh.Do("POST", ws+"/campaigns/"+d.ID+"/preview", map[string]any{
		"sample": "random",
		"html":   `{% assign n = 50000000 %}{% for i in (1..n) %}{% endfor %}<p>never</p>`,
	}).MustStatus(t, 200).JSON(t, &pv)
	if took := time.Since(start); took > 15*time.Second {
		t.Fatalf("the preview took %v", took)
	}
	if len(pv.Problems) == 0 || !strings.Contains(pv.Problems[0].Message, "too much work") {
		t.Fatalf("problems: %+v", pv.Problems)
	}

	// An editor previews what they typed.
	linh.Do("POST", ws+"/campaigns/"+d.ID+"/preview", map[string]any{"sample": "random", "subject": "Typed", "html": "<p>TYPED</p>"}).
		MustStatus(t, 200).JSON(t, &pv)
	if pv.Subject != "Typed" || !strings.Contains(pv.HTML, "TYPED") {
		t.Fatalf("editor's preview: %q / %q", pv.Subject, pv.HTML)
	}

	// A Viewer gets the saved email.
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	aisha.Do("POST", ws+"/campaigns/"+d.ID+"/preview", map[string]any{
		"sample":  "random",
		"subject": "UNSAVED",
		"html":    `<p>UNSAVED</p>{% assign n = 50000000 %}{% for i in (1..n) %}{% endfor %}`,
	}).MustStatus(t, 200).JSON(t, &pv)
	if strings.Contains(pv.Subject, "UNSAVED") || strings.Contains(pv.HTML, "UNSAVED") || len(pv.Problems) != 0 {
		t.Fatalf("a Viewer's preview rendered what they sent: %q / %q / %+v", pv.Subject, pv.HTML, pv.Problems)
	}
}
