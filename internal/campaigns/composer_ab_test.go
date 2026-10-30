package campaigns_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// CO-6: each A/B variant is checked; a subject variant without a subject
// saves (an autosave mid-edit must not fail) and the checks say so.
func TestComposerChecksEachVariant(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var d detail
	c.Do("POST", ws+"/campaigns", map[string]any{"name": "Variants"}).MustStatus(t, 201).JSON(t, &d)
	doc := `<mjml><mj-body><mj-section><mj-column><mj-text>Hello {{ first_name | default: "there" }}. <a href="https://acme.io/news">Read the news</a></mj-text>` +
		`<mj-text>Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105 <a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section></mj-body></mjml>`
	ab := func(kind string, b map[string]any) map[string]any {
		return map[string]any{"kind": kind, "testPercent": 20, "metric": "clicks", "durationHours": 2, "variants": []map[string]any{{"key": "A"}, b}}
	}
	// An empty variant subject saves.
	c.Do("PUT", ws+"/campaigns/"+d.ID+"/content", map[string]any{"baseVersion": 1, "subject": "What shipped this week", "mjml": doc,
		"abTest": ab("subject", map[string]any{"key": "B", "subject": ""})}).MustStatus(t, 200)

	type checks struct {
		Issues   []struct{ ID, Title, Severity string }
		Variants []struct {
			Key              string
			Blocks, Warnings int
			Issues           []struct{ ID, Title, Severity string }
		}
	}
	run := func(cl *apptest.Client, abTest map[string]any, mjml string) checks {
		t.Helper()
		var out checks
		cl.Do("POST", ws+"/campaigns/"+d.ID+"/checks", map[string]any{"subject": "What shipped this week", "preheader": "Flags, exports and more.", "mjml": mjml, "abTest": abTest}).MustStatus(t, 200).JSON(t, &out)
		return out
	}
	out := run(c, ab("subject", map[string]any{"key": "B", "subject": ""}), doc)
	if len(out.Variants) != 1 || out.Variants[0].Key != "B" || out.Variants[0].Blocks != 1 || out.Variants[0].Issues[0].Title != "Write a subject for variant B." {
		t.Fatalf("no subject for B: %+v", out.Variants)
	}
	// Only what B adds: A's own findings aren't repeated.
	out = run(c, ab("subject", map[string]any{"key": "B", "subject": "FREE!!! ACT NOW!!! 100% GUARANTEED"}), doc)
	if b := out.Variants[0]; b.Warnings == 0 || b.Issues[0].ID != "subject.spam" {
		t.Fatalf("spammy B: %+v", b)
	}
	for _, is := range out.Issues {
		if is.ID == "subject.spam" {
			t.Fatal("A's subject is fine")
		}
	}
	// A content variant: its own content is checked.
	shortened := strings.Replace(doc, "https://acme.io/news", "https://bit.ly/3xYz", 1)
	out = run(c, ab("content", map[string]any{"key": "B", "mjml": shortened}), doc)
	found := false
	for _, is := range out.Variants[0].Issues {
		found = found || is.ID == "links.shortener"
	}
	if !found {
		t.Fatalf("B's shortened link: %+v", out.Variants[0])
	}
	// A send-time test has nothing per variant to check.
	if out = run(c, map[string]any{"kind": "send_time", "testPercent": 20, "metric": "clicks", "durationHours": 2, "variants": []map[string]any{{"key": "A", "sendTime": "09:00"}, {"key": "B", "sendTime": "14:00"}}}, doc); len(out.Variants) != 0 {
		t.Fatalf("send time: %+v", out.Variants)
	}
	// In the member's language.
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	out = run(vi, ab("subject", map[string]any{"key": "B", "subject": " "}), doc)
	if title := out.Variants[0].Issues[0].Title; title == "Write a subject for variant B." || !strings.Contains(title, "B") {
		t.Fatalf("Vietnamese: %q", title)
	}
}
