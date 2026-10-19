package ai

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/hostrepute/reputecast/internal/store"
)

// The details line of the assistant's audit entries (SettingsAudit), such as
// "model claude-sonnet-5-5 · key added (write-only) · cap $25.00 a month ·
// all 7 features on". Audit entries are records: like the other features'
// lines they are written once, in English, when the change is made.

// featureLabels are the features' names in the settings.
var featureLabels = map[string]string{
	FeatureSubjectIdeas:  "Subject ideas",
	FeatureRewrite:       "Rewrite, shorten and tone",
	FeatureTranslate:     "Translate a language version",
	FeatureAltText:       "Alt text",
	FeatureAccessibility: "Accessibility fixes",
	FeatureSegment:       "Describe the segment",
	FeatureSummary:       "Report summaries",
}

// Key lines.
const (
	keyAdded    = "key added (write-only)"
	keyReplaced = "key replaced (write-only)"
	keyKept     = "key unchanged"
	keyNone     = "no key"
)

// usdText writes 25 as $25.00 and 100000 as $100,000.00.
func usdText(v float64) string {
	cents := int64(math.Round(v * 100))
	return fmt.Sprintf("$%s.%02d", groupThousands(cents/100), cents%100)
}

// capText is the monthly cap: "$25.00 a month", "20,000,000 tokens a month"
// or "no cap".
func capText(kind string, usd float64, tokens int64) string {
	switch kind {
	case "usd":
		return usdText(usd) + " a month"
	case "tokens":
		return groupThousands(tokens) + " tokens a month"
	}
	return "no cap"
}

// pricesText is "$2.00 in, $10.00 out".
func pricesText(input, output float64) string {
	return usdText(input) + " in, " + usdText(output) + " out"
}

// featuresText counts the features that are on, as the settings show them:
// alt text needs a model that reads images.
func featuresText(row store.AiConnection) string {
	on := 0
	for _, f := range Features {
		if !slices.Contains(row.FeaturesOff, f) && (f != FeatureAltText || row.ImageInput) {
			on++
		}
	}
	switch on {
	case len(Features):
		return fmt.Sprintf("all %d features on", on)
	case 0:
		return "every feature off"
	}
	return fmt.Sprintf("%d of %d features on", on, len(Features))
}

// connectionLine describes a connection made or changed.
func connectionLine(row store.AiConnection, key string) string {
	parts := []string{"model " + row.Model, key}
	if row.CapKind == "none" {
		parts = append(parts, "no cap")
	} else {
		parts = append(parts, "cap "+capText(row.CapKind, row.CapUsd, row.CapTokens))
	}
	return strings.Join(append(parts, featuresText(row)), " · ")
}

// modelLine is "model claude-sonnet-5-5 → claude-opus-5", with the prices
// when they changed too.
func modelLine(from, to string, beforeIn, beforeOut, afterIn, afterOut float64) string {
	line := "model " + from + " → " + to
	if beforeIn != afterIn || beforeOut != afterOut {
		line += " · prices " + pricesText(beforeIn, beforeOut) + " → " + pricesText(afterIn, afterOut) + " per million tokens"
	}
	return line
}

// capLine is "cap $20.00 a month → $25.00 a month".
func capLine(row store.AiConnection, kind string, usd float64, tokens int64) string {
	return "cap " + capText(row.CapKind, row.CapUsd, row.CapTokens) + " → " + capText(kind, usd, tokens)
}

// pricesLine is "$2.00 in, $10.00 out → $3.00 in, $15.00 out per million tokens".
func pricesLine(beforeIn, beforeOut, afterIn, afterOut float64) string {
	return pricesText(beforeIn, beforeOut) + " → " + pricesText(afterIn, afterOut) + " per million tokens"
}

// featureLine is "Subject ideas turned off".
func featureLine(feature string, on bool) string {
	name := featureLabels[feature]
	if name == "" {
		name = feature
	}
	if on {
		return name + " turned on"
	}
	return name + " turned off"
}

// disconnectLine says what a disconnect deletes and keeps.
const disconnectLine = "key deleted · the model, the cap and the prices are kept for the next connection"
