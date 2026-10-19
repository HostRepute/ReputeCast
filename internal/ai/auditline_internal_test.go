package ai

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/store"
)

func TestAuditLines(t *testing.T) {
	row := store.AiConnection{Model: "claude-sonnet-5-5", ImageInput: true, CapKind: "usd", CapUsd: 25, FeaturesOff: []string{}}
	cases := []struct{ name, got, want string }{
		{"connected", connectionLine(row, keyAdded), "model claude-sonnet-5-5 · key added (write-only) · cap $25.00 a month · all 7 features on"},
		{"a big cap", usdText(100000), "$100,000.00"},
		{"cents", usdText(0.5), "$0.50"},
		{"a token cap", capText("tokens", 0, 20000000), "20,000,000 tokens a month"},
		{"no cap", capLine(row, "none", 0, 0), "cap $25.00 a month → no cap"},
		{"a model, same prices", modelLine("a", "b", 2, 10, 2, 10), "model a → b"},
		{"a model, new prices", modelLine("a", "b", 2, 10, 5, 25), "model a → b · prices $2.00 in, $10.00 out → $5.00 in, $25.00 out per million tokens"},
		{"prices", pricesLine(2, 10, 3, 15), "$2.00 in, $10.00 out → $3.00 in, $15.00 out per million tokens"},
		{"a feature", featureLine(FeatureSegment, true), "Describe the segment turned on"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: %q, want %q", c.name, c.got, c.want)
		}
	}
	// Alt text needs a model that reads images; every switch off says so.
	text := store.AiConnection{Model: "llama3.3:70b", CapKind: "none", FeaturesOff: []string{FeatureTranslate}}
	if got := connectionLine(text, keyNone); got != "model llama3.3:70b · no key · no cap · 5 of 7 features on" {
		t.Errorf("a text-only model: %q", got)
	}
	if got := featuresText(store.AiConnection{FeaturesOff: Features}); got != "every feature off" {
		t.Errorf("every switch off: %q", got)
	}
}
